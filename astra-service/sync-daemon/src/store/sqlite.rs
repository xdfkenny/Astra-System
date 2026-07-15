#![deny(unsafe_code)]

//! SQLCipher-encrypted SQLite store for local kiosk state.
//!
//! This module provides a second storage implementation optimized for the
//! CRDT-aware sync daemon.  It keeps four primary tables:
//!
//! * `local_inventory` — SKU-level inventory counts with HLC timestamps.
//! * `local_transactions` — committed local transaction records.
//! * `sync_metadata` — sync watermarks, peer lists, and other daemon metadata.
//! * `offline_queue` — operations queued while offline, pending cloud flush.
//!
//! Encryption is provided by SQLCipher via the `bundled-sqlcipher` feature of
//! `rusqlite`.  The 32-byte raw key is read from the path configured at open
//! time and applied with `PRAGMA key = "x'...'"`.

use std::path::{Path, PathBuf};
use std::sync::Arc;

use parking_lot::Mutex;
use rusqlite::{params, Connection, OptionalExtension};
use serde::de::DeserializeOwned;
use serde::Serialize;
use tracing::{debug, info, trace};

use crate::crdt::hlc::Hlc;
use crate::AstraSyncError;

/// Encrypted SQLite store for local CRDT state and daemon metadata.
#[derive(Debug, Clone)]
pub struct Store {
    conn: Arc<Mutex<Connection>>,
    path: PathBuf,
}

impl Store {
    /// Opens (or creates) an encrypted SQLite database at `db_path` using the raw
    /// 32-byte key stored in `key_path`.
    pub fn open(
        db_path: impl AsRef<Path>,
        key_path: impl AsRef<Path>,
    ) -> Result<Self, AstraSyncError> {
        let db_path = db_path.as_ref().to_path_buf();
        let key = read_raw_key(key_path.as_ref())?;

        let conn = Connection::open(&db_path)
            .map_err(|e| AstraSyncError::Storage(format!("failed to open store: {e}")))?;

        conn.query_row(
            &format!("PRAGMA key = \"x'{}'\"", hex::encode(key)),
            [],
            |_| Ok(()),
        )
        .map_err(|e| AstraSyncError::Storage(format!("failed to set SQLCipher key: {e}")))?;

        // Verify encryption/decryption works before returning.
        conn.query_row("SELECT count(*) FROM sqlite_master", [], |_| Ok(()))
            .map_err(|e| {
                AstraSyncError::Storage(format!(
                    "SQLCipher decryption check failed (wrong key or corrupt DB): {e}"
                ))
            })?;

        conn.query_row("PRAGMA journal_mode = WAL", [], |_| Ok(()))
            .map_err(|e| AstraSyncError::Storage(format!("failed to set WAL mode: {e}")))?;

        info!(db_path = %db_path.display(), "SQLCipher store opened");

        Ok(Self {
            conn: Arc::new(Mutex::new(conn)),
            path: db_path,
        })
    }

    /// Applies the schema migrations.  Safe to call on every startup.
    pub fn migrate(&self) -> Result<(), AstraSyncError> {
        let mut conn = self.conn.lock();
        let tx = conn
            .transaction()
            .map_err(|e| AstraSyncError::Storage(format!("migration tx failed: {e}")))?;

        tx.execute(
            "CREATE TABLE IF NOT EXISTS schema_version (
                version INTEGER PRIMARY KEY
            )",
            [],
        )?;

        let current_version: i64 = tx.query_row(
            "SELECT COALESCE(MAX(version), 0) FROM schema_version",
            [],
            |row| row.get(0),
        )?;

        if current_version < 1 {
            tx.execute(
                "CREATE TABLE IF NOT EXISTS local_inventory (
                    sku TEXT PRIMARY KEY NOT NULL,
                    count INTEGER NOT NULL DEFAULT 0,
                    hlc_json TEXT NOT NULL,
                    updated_at_ms INTEGER NOT NULL
                )",
                [],
            )?;
            tx.execute(
                "CREATE INDEX IF NOT EXISTS idx_inventory_updated ON local_inventory(updated_at_ms)",
                [],
            )?;
        }

        if current_version < 2 {
            tx.execute(
                "CREATE TABLE IF NOT EXISTS local_transactions (
                    id TEXT PRIMARY KEY NOT NULL,
                    payload_json TEXT NOT NULL,
                    hlc_json TEXT NOT NULL,
                    committed INTEGER NOT NULL DEFAULT 0,
                    created_at_ms INTEGER NOT NULL
                )",
                [],
            )?;
            tx.execute(
                "CREATE INDEX IF NOT EXISTS idx_tx_committed ON local_transactions(committed, created_at_ms)",
                [],
            )?;
        }

        if current_version < 3 {
            tx.execute(
                "CREATE TABLE IF NOT EXISTS sync_metadata (
                    key TEXT PRIMARY KEY NOT NULL,
                    value TEXT NOT NULL,
                    updated_at_ms INTEGER NOT NULL
                )",
                [],
            )?;
        }

        if current_version < 4 {
            tx.execute(
                "CREATE TABLE IF NOT EXISTS offline_queue (
                    id TEXT PRIMARY KEY NOT NULL,
                    kind TEXT NOT NULL,
                    payload_json TEXT NOT NULL,
                    hlc_json TEXT NOT NULL,
                    created_at_ms INTEGER NOT NULL,
                    synced INTEGER NOT NULL DEFAULT 0
                )",
                [],
            )?;
            tx.execute(
                "CREATE INDEX IF NOT EXISTS idx_offline_synced ON offline_queue(synced, created_at_ms)",
                [],
            )?;
        }

        let new_version = 4i64;
        tx.execute(
            "INSERT OR REPLACE INTO schema_version (version) VALUES (?)",
            [new_version],
        )?;

        tx.commit()
            .map_err(|e| AstraSyncError::Storage(format!("migration commit failed: {e}")))?;

        info!(
            from = current_version,
            to = new_version,
            "Store migrations applied"
        );
        Ok(())
    }

    /// Inserts or updates an inventory count.
    pub fn set_inventory(&self, sku: &str, count: i64, hlc: &Hlc) -> Result<(), AstraSyncError> {
        let hlc_json =
            serde_json::to_string(hlc).map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
        let now = hlc.wallclock_ms as i64;
        let conn = self.conn.lock();
        conn.execute(
            "INSERT INTO local_inventory (sku, count, hlc_json, updated_at_ms)
             VALUES (?1, ?2, ?3, ?4)
             ON CONFLICT(sku) DO UPDATE SET
                count = excluded.count,
                hlc_json = excluded.hlc_json,
                updated_at_ms = excluded.updated_at_ms
             WHERE excluded.updated_at_ms >= local_inventory.updated_at_ms",
            params![sku, count, hlc_json, now],
        )
        .map_err(|e| AstraSyncError::Storage(format!("set_inventory failed: {e}")))?;
        trace!(sku, count, "Inventory updated");
        Ok(())
    }

    /// Reads the current inventory count and HLC for a SKU.
    pub fn get_inventory(&self, sku: &str) -> Result<Option<(i64, Hlc)>, AstraSyncError> {
        let conn = self.conn.lock();
        let result: Option<(i64, String)> = conn
            .query_row(
                "SELECT count, hlc_json FROM local_inventory WHERE sku = ?1",
                [sku],
                |row| Ok((row.get(0)?, row.get(1)?)),
            )
            .optional()
            .map_err(|e| AstraSyncError::Storage(format!("get_inventory failed: {e}")))?;

        result
            .map(|(count, json)| {
                let hlc = serde_json::from_str(&json)
                    .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
                Ok((count, hlc))
            })
            .transpose()
    }

    /// Lists inventory rows updated since `since_ms`.
    pub fn list_inventory_since(
        &self,
        since_ms: i64,
        limit: usize,
    ) -> Result<Vec<(String, i64, Hlc)>, AstraSyncError> {
        let conn = self.conn.lock();
        let mut stmt = conn
            .prepare(
                "SELECT sku, count, hlc_json FROM local_inventory
                 WHERE updated_at_ms > ?1 ORDER BY updated_at_ms ASC LIMIT ?2",
            )
            .map_err(|e| AstraSyncError::Storage(format!("prepare failed: {e}")))?;

        let rows = stmt
            .query_map(params![since_ms, limit as i64], |row| {
                let sku: String = row.get(0)?;
                let count: i64 = row.get(1)?;
                let hlc_json: String = row.get(2)?;
                let hlc = serde_json::from_str(&hlc_json).map_err(|e| {
                    rusqlite::Error::FromSqlConversionFailure(
                        2,
                        rusqlite::types::Type::Text,
                        Box::new(e),
                    )
                })?;
                Ok((sku, count, hlc))
            })
            .map_err(|e| AstraSyncError::Storage(format!("query failed: {e}")))?;

        let mut out = Vec::new();
        for row in rows {
            out.push(row.map_err(|e| AstraSyncError::Storage(format!("row failed: {e}")))?);
        }
        Ok(out)
    }

    /// Stores a local transaction record.
    pub fn insert_transaction(
        &self,
        id: &str,
        payload: &impl Serialize,
        hlc: &Hlc,
        committed: bool,
    ) -> Result<(), AstraSyncError> {
        let payload_json = serde_json::to_string(payload)
            .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
        let hlc_json =
            serde_json::to_string(hlc).map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
        let conn = self.conn.lock();
        conn.execute(
            "INSERT OR REPLACE INTO local_transactions (id, payload_json, hlc_json, committed, created_at_ms)
             VALUES (?1, ?2, ?3, ?4, ?5)",
            params![id, payload_json, hlc_json, committed as i32, hlc.wallclock_ms as i64],
        )
        .map_err(|e| AstraSyncError::Storage(format!("insert_transaction failed: {e}")))?;
        debug!(transaction_id = id, "Transaction stored");
        Ok(())
    }

    /// Loads uncommitted transactions.
    pub fn load_uncommitted_transactions<T: DeserializeOwned>(
        &self,
        limit: usize,
    ) -> Result<Vec<(T, Hlc)>, AstraSyncError> {
        self.load_transactions_by_committed::<T>(false, limit)
    }

    /// Loads committed transactions.
    pub fn load_committed_transactions<T: DeserializeOwned>(
        &self,
        limit: usize,
    ) -> Result<Vec<(T, Hlc)>, AstraSyncError> {
        self.load_transactions_by_committed::<T>(true, limit)
    }

    fn load_transactions_by_committed<T: DeserializeOwned>(
        &self,
        committed: bool,
        limit: usize,
    ) -> Result<Vec<(T, Hlc)>, AstraSyncError> {
        let conn = self.conn.lock();
        let mut stmt = conn
            .prepare(
                "SELECT payload_json, hlc_json FROM local_transactions
                 WHERE committed = ?1 ORDER BY created_at_ms ASC LIMIT ?2",
            )
            .map_err(|e| AstraSyncError::Storage(format!("prepare failed: {e}")))?;

        let rows = stmt
            .query_map(params![committed as i32, limit as i64], |row| {
                let payload_json: String = row.get(0)?;
                let hlc_json: String = row.get(1)?;
                let payload: T = serde_json::from_str(&payload_json).map_err(|e| {
                    rusqlite::Error::FromSqlConversionFailure(
                        0,
                        rusqlite::types::Type::Text,
                        Box::new(e),
                    )
                })?;
                let hlc: Hlc = serde_json::from_str(&hlc_json).map_err(|e| {
                    rusqlite::Error::FromSqlConversionFailure(
                        1,
                        rusqlite::types::Type::Text,
                        Box::new(e),
                    )
                })?;
                Ok((payload, hlc))
            })
            .map_err(|e| AstraSyncError::Storage(format!("query failed: {e}")))?;

        let mut out = Vec::new();
        for row in rows {
            out.push(row.map_err(|e| AstraSyncError::Storage(format!("row failed: {e}")))?);
        }
        Ok(out)
    }

    /// Marks a transaction as committed.
    pub fn commit_transaction(&self, id: &str) -> Result<(), AstraSyncError> {
        let conn = self.conn.lock();
        conn.execute(
            "UPDATE local_transactions SET committed = 1 WHERE id = ?1",
            [id],
        )
        .map_err(|e| AstraSyncError::Storage(format!("commit_transaction failed: {e}")))?;
        Ok(())
    }

    /// Reads a generic metadata value.
    pub fn get_metadata(&self, key: &str) -> Result<Option<String>, AstraSyncError> {
        let conn = self.conn.lock();
        let value: Option<String> = conn
            .query_row(
                "SELECT value FROM sync_metadata WHERE key = ?1",
                [key],
                |row| row.get(0),
            )
            .optional()
            .map_err(|e| AstraSyncError::Storage(format!("get_metadata failed: {e}")))?;
        Ok(value)
    }

    /// Writes a generic metadata value.
    pub fn set_metadata(
        &self,
        key: &str,
        value: &str,
        updated_at_ms: i64,
    ) -> Result<(), AstraSyncError> {
        let conn = self.conn.lock();
        conn.execute(
            "INSERT INTO sync_metadata (key, value, updated_at_ms)
             VALUES (?1, ?2, ?3)
             ON CONFLICT(key) DO UPDATE SET
                value = excluded.value,
                updated_at_ms = excluded.updated_at_ms",
            params![key, value, updated_at_ms],
        )
        .map_err(|e| AstraSyncError::Storage(format!("set_metadata failed: {e}")))?;
        Ok(())
    }

    /// Enqueues an offline operation.
    pub fn enqueue_offline(
        &self,
        id: &str,
        kind: &str,
        payload: &impl Serialize,
        hlc: &Hlc,
    ) -> Result<(), AstraSyncError> {
        let payload_json = serde_json::to_string(payload)
            .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
        let hlc_json =
            serde_json::to_string(hlc).map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
        let conn = self.conn.lock();
        conn.execute(
            "INSERT OR REPLACE INTO offline_queue (id, kind, payload_json, hlc_json, created_at_ms, synced)
             VALUES (?1, ?2, ?3, ?4, ?5, 0)",
            params![id, kind, payload_json, hlc_json, hlc.wallclock_ms as i64],
        )
        .map_err(|e| AstraSyncError::Storage(format!("enqueue_offline failed: {e}")))?;
        Ok(())
    }

    /// Loads unsynced offline queue entries.
    pub fn load_offline_queue(
        &self,
        limit: usize,
    ) -> Result<Vec<(OfflineEntry, Hlc)>, AstraSyncError> {
        let conn = self.conn.lock();
        let mut stmt = conn
            .prepare(
                "SELECT id, kind, payload_json, hlc_json FROM offline_queue
                 WHERE synced = 0 ORDER BY created_at_ms ASC LIMIT ?1",
            )
            .map_err(|e| AstraSyncError::Storage(format!("prepare failed: {e}")))?;

        let rows = stmt
            .query_map([limit as i64], |row| {
                let id: String = row.get(0)?;
                let kind: String = row.get(1)?;
                let payload_json: String = row.get(2)?;
                let hlc_json: String = row.get(3)?;
                let hlc: Hlc = serde_json::from_str(&hlc_json).map_err(|e| {
                    rusqlite::Error::FromSqlConversionFailure(
                        3,
                        rusqlite::types::Type::Text,
                        Box::new(e),
                    )
                })?;
                Ok((
                    OfflineEntry {
                        id,
                        kind,
                        payload_json,
                    },
                    hlc,
                ))
            })
            .map_err(|e| AstraSyncError::Storage(format!("query failed: {e}")))?;

        let mut out = Vec::new();
        for row in rows {
            out.push(row.map_err(|e| AstraSyncError::Storage(format!("row failed: {e}")))?);
        }
        Ok(out)
    }

    /// Marks offline queue entries as synced.
    pub fn mark_offline_synced(&self, ids: &[&str]) -> Result<usize, AstraSyncError> {
        let mut conn = self.conn.lock();
        let tx = conn
            .transaction()
            .map_err(|e| AstraSyncError::Storage(format!("mark_synced tx failed: {e}")))?;
        let mut count = 0usize;
        for id in ids {
            count += tx
                .execute("UPDATE offline_queue SET synced = 1 WHERE id = ?1", [id])
                .map_err(|e| AstraSyncError::Storage(format!("mark_synced failed: {e}")))?;
        }
        tx.commit()
            .map_err(|e| AstraSyncError::Storage(format!("mark_synced commit failed: {e}")))?;
        Ok(count)
    }

    /// Closes the database, checkpointing the WAL.
    pub fn close(&self) -> Result<(), AstraSyncError> {
        let conn = self.conn.lock();
        conn.execute("PRAGMA wal_checkpoint(TRUNCATE)", [])
            .map_err(|e| AstraSyncError::Storage(format!("final checkpoint failed: {e}")))?;
        info!(path = %self.path.display(), "Store closed");
        Ok(())
    }
}

/// A raw entry from the offline queue.
#[derive(Debug, Clone)]
pub struct OfflineEntry {
    pub id: String,
    pub kind: String,
    pub payload_json: String,
}

/// Reads exactly 32 raw bytes from `path` and validates file permissions.
fn read_raw_key(path: &Path) -> Result<[u8; 32], AstraSyncError> {
    let metadata = std::fs::metadata(path)
        .map_err(|e| AstraSyncError::Crypto(format!("failed to read key file metadata: {e}")))?;
    #[cfg(unix)]
    {
        use std::os::unix::fs::MetadataExt;
        let mode = metadata.mode() & 0o777;
        if mode > 0o600 {
            return Err(AstraSyncError::Crypto(
                "store key file permissions too permissive (must be <= 0o600)".to_string(),
            ));
        }
    }
    let bytes = std::fs::read(path)
        .map_err(|e| AstraSyncError::Crypto(format!("failed to read key file: {e}")))?;
    if bytes.len() != 32 {
        return Err(AstraSyncError::Crypto(format!(
            "store key must be exactly 32 bytes, got {}",
            bytes.len()
        )));
    }
    let mut key = [0u8; 32];
    key.copy_from_slice(&bytes);
    Ok(key)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::crdt::hlc::Hlc;

    fn temp_store() -> (Store, tempfile::TempDir, PathBuf) {
        let dir = tempfile::tempdir().expect("tempdir");
        let db_path = dir.path().join("store.db");
        let key_path = dir.path().join("store.key");
        std::fs::write(&key_path, [0u8; 32]).expect("write key");
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let mut perms = std::fs::metadata(&key_path)
                .expect("metadata")
                .permissions();
            perms.set_mode(0o600);
            std::fs::set_permissions(&key_path, perms).expect("set perms");
        }
        let store = Store::open(&db_path, &key_path).expect("open store");
        store.migrate().expect("migrate");
        (store, dir, key_path)
    }

    #[test]
    fn store_inventory_roundtrip() {
        let (store, _dir, _key) = temp_store();
        let hlc = Hlc::new("k1").unwrap();
        store.set_inventory("sku-1", 42, &hlc).unwrap();
        let (count, stored_hlc) = store.get_inventory("sku-1").unwrap().unwrap();
        assert_eq!(count, 42);
        assert_eq!(stored_hlc.node_id, "k1");
    }

    #[test]
    fn store_inventory_list_since() {
        let (store, _dir, _key) = temp_store();
        let hlc = Hlc::new("k1").unwrap();
        store.set_inventory("sku-1", 10, &hlc).unwrap();
        let rows = store.list_inventory_since(0, 10).unwrap();
        assert_eq!(rows.len(), 1);
        assert_eq!(rows[0].0, "sku-1");
    }

    #[test]
    fn store_transaction_commit() {
        let (store, _dir, _key) = temp_store();
        let hlc = Hlc::new("k1").unwrap();
        let payload = serde_json::json!({"total": 500});
        store
            .insert_transaction("tx-1", &payload, &hlc, false)
            .unwrap();
        let uncommitted: Vec<(serde_json::Value, Hlc)> =
            store.load_uncommitted_transactions(10).unwrap();
        assert_eq!(uncommitted.len(), 1);
        store.commit_transaction("tx-1").unwrap();
        let uncommitted: Vec<(serde_json::Value, Hlc)> =
            store.load_uncommitted_transactions(10).unwrap();
        assert!(uncommitted.is_empty());
    }

    #[test]
    fn store_metadata_roundtrip() {
        let (store, _dir, _key) = temp_store();
        store.set_metadata("last_sync", "12345", 1).unwrap();
        assert_eq!(
            store.get_metadata("last_sync").unwrap(),
            Some("12345".to_string())
        );
    }

    #[test]
    fn store_offline_queue() {
        let (store, _dir, _key) = temp_store();
        let hlc = Hlc::new("k1").unwrap();
        let payload = serde_json::json!({"op": "refill"});
        store
            .enqueue_offline("q-1", "refill", &payload, &hlc)
            .unwrap();
        let queued = store.load_offline_queue(10).unwrap();
        assert_eq!(queued.len(), 1);
        store.mark_offline_synced(&["q-1"]).unwrap();
        let queued = store.load_offline_queue(10).unwrap();
        assert!(queued.is_empty());
    }

    #[test]
    fn store_rejects_bad_key_size() {
        let dir = tempfile::tempdir().expect("tempdir");
        let db_path = dir.path().join("store.db");
        let key_path = dir.path().join("bad.key");
        std::fs::write(&key_path, [0u8; 16]).expect("write key");
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let mut perms = std::fs::metadata(&key_path)
                .expect("metadata")
                .permissions();
            perms.set_mode(0o600);
            std::fs::set_permissions(&key_path, perms).expect("set perms");
        }
        assert!(Store::open(&db_path, &key_path).is_err());
    }
}

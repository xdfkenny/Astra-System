#![deny(unsafe_code)]

//! Encrypted SQLite storage with SQLCipher.
//!
//! The database stores all CRDT state, Raft logs, offline payment queues,
//! and sync metadata. SQLCipher provides AES-256 encryption at rest.
//!
//! Schema:
//! - `sync_records`: generic table for all SyncRecord<T> types (JSON payload).
//! - `raft_logs`: Raft log entries for consensus state.
//! - `offline_payments`: queued offline payment tokens with HMAC signatures.
//! - `sync_state`: last sync timestamps and watermarks per data type.

use std::collections::HashMap;
use std::sync::Arc;

use chrono::Utc;
use parking_lot::Mutex;
use rusqlite::{Connection, OptionalExtension, params, ToSql};
use serde::de::DeserializeOwned;
use tracing::{debug, error, info, trace};

use crate::config::{StorageConfig, CryptoConfig};
use crate::crypto::SyncKey;
use crate::{DataType, AstraSyncError, KioskId};
use crate::SyncRecord;

/// Internal wrapper around a rusqlite connection with SQLCipher.
#[derive(Debug)]
pub struct SyncDatabase {
    conn: Arc<Mutex<Connection>>,
    db_path: std::path::PathBuf,
}

impl SyncDatabase {
    /// Opens (or creates) the encrypted SQLite database.
    /// The encryption key is read from the path specified in `StorageConfig`.
    pub async fn open(storage: &StorageConfig, crypto: &CryptoConfig) -> Result<Self, AstraSyncError> {
        let key = std::fs::read_to_string(&storage.encryption_key_path)
            .map_err(|e| AstraSyncError::Storage(format!("failed to read encryption key: {e}")))?
            .trim()
            .to_string();

        let conn = Connection::open(&storage.db_path)
            .map_err(|e| AstraSyncError::Storage(format!("failed to open database: {e}")))?;

        // SQLCipher PRAGMA: hex-encoded key or raw bytes via `PRAGMA key`.
        // If the key file contains a hex string, we pass it directly.
        conn.execute(&format!("PRAGMA key = \"x'{key}'\""), [])
            .map_err(|e| AstraSyncError::Storage(format!("failed to set SQLCipher key: {e}")))?;

        // Verify encryption is active by trying to read.
        conn.execute("SELECT count(*) FROM sqlite_master", [])
            .map_err(|e| AstraSyncError::Storage(format!(
                "SQLCipher decryption failed (wrong key or corrupt database): {e}"
            )))?;

        // WAL mode for concurrent reads and writes.
        conn.execute("PRAGMA journal_mode = WAL", [])
            .map_err(|e| AstraSyncError::Storage(format!("failed to set WAL mode: {e}")))?;
        conn.execute(&format!("PRAGMA wal_autocheckpoint = {}", storage.wal_checkpoint_pages), [])
            .map_err(|e| AstraSyncError::Storage(format!("failed to set WAL checkpoint: {e}")))?;
        conn.execute(&format!("PRAGMA max_page_count = {}", storage.max_db_size_mib * 1024 * 1024 / 4096), [])
            .map_err(|e| AstraSyncError::Storage(format!("failed to set max page count: {e}")))?;

        info!(db_path = %storage.db_path.display(), "Encrypted SQLite database opened");

        Ok(Self {
            conn: Arc::new(Mutex::new(conn)),
            db_path: storage.db_path.clone(),
        })
    }

    /// Runs database migrations. Safe to call on every startup — uses a version table.
    pub async fn migrate(&self) -> Result<(), AstraSyncError> {
        let mut conn = self.conn.lock();
        let tx = conn.transaction()
            .map_err(|e| AstraSyncError::Storage(format!("failed to start migration transaction: {e}")))?;

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
                "CREATE TABLE IF NOT EXISTS sync_records (
                    id TEXT PRIMARY KEY NOT NULL,
                    data_type INTEGER NOT NULL,
                    origin TEXT NOT NULL,
                    lamport_ts INTEGER NOT NULL,
                    wallclock_ts INTEGER NOT NULL,
                    payload_json TEXT NOT NULL,
                    hmac BLOB NOT NULL,
                    dirty INTEGER NOT NULL DEFAULT 1
                )",
                [],
            )?;
            tx.execute(
                "CREATE INDEX IF NOT EXISTS idx_sync_records_type ON sync_records(data_type)",
                [],
            )?;
            tx.execute(
                "CREATE INDEX IF NOT EXISTS idx_sync_records_dirty ON sync_records(dirty, data_type)",
                [],
            )?;
        }

        if current_version < 2 {
            tx.execute(
                "CREATE TABLE IF NOT EXISTS raft_logs (
                    term INTEGER NOT NULL,
                    index INTEGER NOT NULL,
                    command_type INTEGER NOT NULL,
                    command_data BLOB NOT NULL,
                    PRIMARY KEY (term, index)
                )",
                [],
            )?;
        }

        if current_version < 3 {
            tx.execute(
                "CREATE TABLE IF NOT EXISTS offline_payments (
                    token_id TEXT PRIMARY KEY NOT NULL,
                    payload_json TEXT NOT NULL,
                    hmac BLOB NOT NULL,
                    created_at INTEGER NOT NULL,
                    flushed INTEGER NOT NULL DEFAULT 0
                )",
                [],
            )?;
            tx.execute(
                "CREATE INDEX IF NOT EXISTS idx_offline_flushed ON offline_payments(flushed, created_at)",
                [],
            )?;
        }

        if current_version < 4 {
            tx.execute(
                "CREATE TABLE IF NOT EXISTS sync_state (
                    data_type INTEGER PRIMARY KEY NOT NULL,
                    last_sync_ts INTEGER NOT NULL DEFAULT 0,
                    last_lamport_ts INTEGER NOT NULL DEFAULT 0
                )",
                [],
            )?;
        }

        let new_version = 4;
        tx.execute(
            "INSERT OR REPLACE INTO schema_version (version) VALUES (?)",
            [new_version],
        )?;

        tx.commit()
            .map_err(|e| AstraSyncError::Storage(format!("migration commit failed: {e}")))?;

        info!(from = current_version, to = new_version, "Database migrations applied");
        Ok(())
    }

    /// Inserts or updates a sync record. The record is marked as dirty.
    pub async fn upsert<T: serde::Serialize + Send + Sync>(
        &self,
        record: &SyncRecord<T>,
    ) -> Result<(), AstraSyncError> {
        let payload_json = serde_json::to_string(&record.payload)
            .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
        let conn = self.conn.lock();
        conn.execute(
            "INSERT INTO sync_records (id, data_type, origin, lamport_ts, wallclock_ts, payload_json, hmac, dirty)
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, 1)
             ON CONFLICT(id) DO UPDATE SET
                data_type = excluded.data_type,
                origin = excluded.origin,
                lamport_ts = excluded.lamport_ts,
                wallclock_ts = excluded.wallclock_ts,
                payload_json = excluded.payload_json,
                hmac = excluded.hmac,
                dirty = 1",
            params![
                record.id,
                record.data_type as i64,
                record.origin.to_string(),
                record.lamport_ts as i64,
                record.wallclock_ts as i64,
                payload_json,
                &record.hmac,
            ],
        ).map_err(|e| AstraSyncError::Storage(format!("upsert failed: {e}")))?;
        trace!(record_id = %record.id, "Upserted sync record");
        Ok(())
    }

    /// Loads all sync records of a given data type.
    pub async fn load_all<T: DeserializeOwned + Send + Sync>(
        &self,
        data_type: DataType,
    ) -> Result<Vec<SyncRecord<T>>, AstraSyncError> {
        let conn = self.conn.lock();
        let mut stmt = conn.prepare(
            "SELECT id, origin, lamport_ts, wallclock_ts, payload_json, hmac
             FROM sync_records WHERE data_type = ?1 ORDER BY lamport_ts ASC"
        ).map_err(|e| AstraSyncError::Storage(format!("prepare failed: {e}")))?;

        let rows = stmt.query_map([data_type as i64], |row| {
            let id: String = row.get(0)?;
            let origin: String = row.get(1)?;
            let lamport_ts: i64 = row.get(2)?;
            let wallclock_ts: i64 = row.get(3)?;
            let payload_json: String = row.get(4)?;
            let hmac: Vec<u8> = row.get(5)?;
            let payload: T = serde_json::from_str(&payload_json)
                .map_err(|e| rusqlite::Error::FromSqlConversionFailure(
                    4, rusqlite::types::Type::Text, Box::new(e),
                ))?;
            Ok(SyncRecord {
                data_type,
                id,
                origin: KioskId::from(origin),
                lamport_ts: lamport_ts as u64,
                wallclock_ts: wallclock_ts as u64,
                payload,
                hmac,
            })
        }).map_err(|e| AstraSyncError::Storage(format!("query failed: {e}")))?;

        let mut records = Vec::new();
        for row in rows {
            records.push(row.map_err(|e| AstraSyncError::Storage(format!("row parse failed: {e}")))?);
        }

        debug!(data_type = ?data_type, count = records.len(), "Loaded sync records");
        Ok(records)
    }

    /// Loads dirty records that need to be synced to the mesh or cloud.
    pub async fn load_dirty<T: DeserializeOwned + Send + Sync>(
        &self,
        data_type: DataType,
        limit: usize,
    ) -> Result<Vec<SyncRecord<T>>, AstraSyncError> {
        let conn = self.conn.lock();
        let mut stmt = conn.prepare(
            "SELECT id, origin, lamport_ts, wallclock_ts, payload_json, hmac
             FROM sync_records WHERE data_type = ?1 AND dirty = 1
             ORDER BY lamport_ts ASC LIMIT ?2"
        ).map_err(|e| AstraSyncError::Storage(format!("prepare failed: {e}")))?;

        let rows = stmt.query_map(params![data_type as i64, limit as i64], |row| {
            let id: String = row.get(0)?;
            let origin: String = row.get(1)?;
            let lamport_ts: i64 = row.get(2)?;
            let wallclock_ts: i64 = row.get(3)?;
            let payload_json: String = row.get(4)?;
            let hmac: Vec<u8> = row.get(5)?;
            let payload: T = serde_json::from_str(&payload_json)
                .map_err(|e| rusqlite::Error::FromSqlConversionFailure(
                    4, rusqlite::types::Type::Text, Box::new(e),
                ))?;
            Ok(SyncRecord {
                data_type,
                id,
                origin: KioskId::from(origin),
                lamport_ts: lamport_ts as u64,
                wallclock_ts: wallclock_ts as u64,
                payload,
                hmac,
            })
        }).map_err(|e| AstraSyncError::Storage(format!("query failed: {e}")))?;

        let mut records = Vec::new();
        for row in rows {
            records.push(row.map_err(|e| AstraSyncError::Storage(format!("row parse failed: {e}")))?);
        }
        Ok(records)
    }

    /// Marks all records of a data type as clean (dirty = 0).
    pub async fn flush(&self, data_type: DataType) -> Result<usize, AstraSyncError> {
        let conn = self.conn.lock();
        let count = conn.execute(
            "UPDATE sync_records SET dirty = 0 WHERE data_type = ?1 AND dirty = 1",
            [data_type as i64],
        ).map_err(|e| AstraSyncError::Storage(format!("flush failed: {e}")))?;
        if count > 0 {
            debug!(data_type = ?data_type, count, "Flushed dirty records");
        }
        Ok(count)
    }

    /// Stores an offline payment token with its HMAC signature.
    pub async fn queue_offline_payment(
        &self,
        token_id: &str,
        payload_json: &str,
        hmac: &[u8],
    ) -> Result<(), AstraSyncError> {
        let conn = self.conn.lock();
        conn.execute(
            "INSERT INTO offline_payments (token_id, payload_json, hmac, created_at, flushed)
             VALUES (?1, ?2, ?3, ?4, 0)
             ON CONFLICT(token_id) DO UPDATE SET
                payload_json = excluded.payload_json,
                hmac = excluded.hmac,
                created_at = excluded.created_at",
            params![token_id, payload_json, hmac, Utc::now().timestamp_millis()],
        ).map_err(|e| AstraSyncError::Storage(format!("offline payment queue failed: {e}")))?;
        debug!(token_id, "Queued offline payment token");
        Ok(())
    }

    /// Loads unflushed offline payment tokens.
    pub async fn load_offline_queue(
        &self,
        limit: usize,
    ) -> Result<Vec<(String, String, Vec<u8>)>, AstraSyncError> {
        let conn = self.conn.lock();
        let mut stmt = conn.prepare(
            "SELECT token_id, payload_json, hmac FROM offline_payments
             WHERE flushed = 0 ORDER BY created_at ASC LIMIT ?1"
        ).map_err(|e| AstraSyncError::Storage(format!("prepare failed: {e}")))?;

        let rows = stmt.query_map([limit as i64], |row| {
            let token_id: String = row.get(0)?;
            let payload_json: String = row.get(1)?;
            let hmac: Vec<u8> = row.get(2)?;
            Ok((token_id, payload_json, hmac))
        }).map_err(|e| AstraSyncError::Storage(format!("query failed: {e}")))?;

        let mut out = Vec::new();
        for row in rows {
            out.push(row.map_err(|e| AstraSyncError::Storage(format!("row parse failed: {e}")))?);
        }
        Ok(out)
    }

    /// Marks offline payment tokens as flushed.
    pub async fn mark_offline_flushed(&self, token_ids: &[String]) -> Result<usize, AstraSyncError> {
        let mut conn = self.conn.lock();
        let tx = conn.transaction()
            .map_err(|e| AstraSyncError::Storage(format!("transaction failed: {e}")))?;
        let mut count = 0;
        for id in token_ids {
            count += tx.execute(
                "UPDATE offline_payments SET flushed = 1 WHERE token_id = ?1",
                [id],
            ).map_err(|e| AstraSyncError::Storage(format!("mark flushed failed: {e}")))?;
        }
        tx.commit()
            .map_err(|e| AstraSyncError::Storage(format!("commit failed: {e}")))?;
        debug!(count, "Marked offline payments as flushed");
        Ok(count)
    }

    /// Appends a Raft log entry.
    pub async fn append_raft_log(
        &self,
        term: u64,
        index: u64,
        command_type: u8,
        command_data: &[u8],
    ) -> Result<(), AstraSyncError> {
        let conn = self.conn.lock();
        conn.execute(
            "INSERT INTO raft_logs (term, index, command_type, command_data)
             VALUES (?1, ?2, ?3, ?4)
             ON CONFLICT(term, index) DO UPDATE SET
                command_type = excluded.command_type,
                command_data = excluded.command_data",
            params![term as i64, index as i64, command_type, command_data],
        ).map_err(|e| AstraSyncError::Storage(format!("append raft log failed: {e}")))?;
        Ok(())
    }

    /// Reads Raft log entries from a given index onward.
    pub async fn read_raft_logs(
        &self,
        from_index: u64,
        limit: usize,
    ) -> Result<Vec<(u64, u64, u8, Vec<u8>)>, AstraSyncError> {
        let conn = self.conn.lock();
        let mut stmt = conn.prepare(
            "SELECT term, index, command_type, command_data FROM raft_logs
             WHERE index >= ?1 ORDER BY index ASC LIMIT ?2"
        ).map_err(|e| AstraSyncError::Storage(format!("prepare failed: {e}")))?;

        let rows = stmt.query_map([from_index as i64, limit as i64], |row| {
            let term: i64 = row.get(0)?;
            let index: i64 = row.get(1)?;
            let command_type: u8 = row.get(2)?;
            let command_data: Vec<u8> = row.get(3)?;
            Ok((term as u64, index as u64, command_type, command_data))
        }).map_err(|e| AstraSyncError::Storage(format!("query failed: {e}")))?;

        let mut out = Vec::new();
        for row in rows {
            out.push(row.map_err(|e| AstraSyncError::Storage(format!("row parse failed: {e}")))?);
        }
        Ok(out)
    }

    /// Updates the last sync timestamp for a data type.
    pub async fn update_sync_state(&self, data_type: DataType, sync_ts: u64, lamport_ts: u64) -> Result<(), AstraSyncError> {
        let conn = self.conn.lock();
        conn.execute(
            "INSERT INTO sync_state (data_type, last_sync_ts, last_lamport_ts)
             VALUES (?1, ?2, ?3)
             ON CONFLICT(data_type) DO UPDATE SET
                last_sync_ts = excluded.last_sync_ts,
                last_lamport_ts = excluded.last_lamport_ts",
            params![data_type as i64, sync_ts as i64, lamport_ts as i64],
        ).map_err(|e| AstraSyncError::Storage(format!("update sync state failed: {e}")))?;
        Ok(())
    }

    /// Reads the last sync timestamp for a data type.
    pub async fn get_sync_state(&self, data_type: DataType) -> Result<(u64, u64), AstraSyncError> {
        let conn = self.conn.lock();
        let result: Option<(i64, i64)> = conn.query_row(
            "SELECT last_sync_ts, last_lamport_ts FROM sync_state WHERE data_type = ?1",
            [data_type as i64],
            |row| Ok((row.get(0)?, row.get(1)?)),
        ).optional().map_err(|e| AstraSyncError::Storage(format!("query sync state failed: {e}")))?;
        Ok(result.map_or((0, 0), |(ts, lp)| (ts as u64, lp as u64)))
    }

    /// Gracefully closes the database connection.
    pub async fn close(&self) -> Result<(), AstraSyncError> {
        let conn = self.conn.lock();
        // Force checkpoint to merge WAL into main database before shutdown.
        conn.execute("PRAGMA wal_checkpoint(TRUNCATE)", [])
            .map_err(|e| AstraSyncError::Storage(format!("final checkpoint failed: {e}")))?;
        info!("Database connection closed cleanly");
        Ok(())
    }
}

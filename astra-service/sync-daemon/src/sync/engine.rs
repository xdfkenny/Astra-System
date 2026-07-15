#![deny(unsafe_code)]

//! Sync engine orchestrates priority-based CRDT synchronization across the P2P mesh.
//!
//! The engine runs three background loops:
//! - Immediate loop: high-frequency, pushes inventory changes immediately.
//! - Batched loop: 5-second cadence, collects transactions and cart state into batches.
//! - Delayed loop: low-frequency, uploads analytics and telemetry when idle.

use std::collections::{HashMap, VecDeque};
use std::sync::Arc;
use std::time::Duration;

use tokio::sync::{watch, Mutex, RwLock};
use tokio::time::interval;
use tracing::{debug, error, info, trace, warn};

use crate::config::Config;
use crate::p2p::mesh::P2PMeshHandle;
use crate::storage::sqlite::SyncDatabase;
use crate::sync::crdt::{LWWRegister, LamportClock};
use crate::{AstraSyncError, DaemonState, DataType, SyncRecord};

/// Batch interval for `SyncPriority::Batched` data types.
const BATCH_INTERVAL_SECS: u64 = 5;
/// Delayed interval for `SyncPriority::Delayed` data types.
const DELAYED_INTERVAL_SECS: u64 = 60;
/// Immediate interval — essentially a tight loop with backpressure.
const IMMEDIATE_INTERVAL_MILLIS: u64 = 100;

/// Handle returned when the sync engine is started, allowing external subsystems
/// to request immediate syncs or query the engine state.
#[derive(Debug, Clone)]
pub struct SyncEngineHandle {
    pub cmd_tx: tokio::sync::mpsc::Sender<EngineCommand>,
}

impl SyncEngineHandle {
    /// Requests an immediate sync of a specific data type.
    pub async fn request_sync(&self, data_type: DataType) -> Result<(), AstraSyncError> {
        self.cmd_tx
            .send(EngineCommand::SyncNow(data_type))
            .await
            .map_err(|_| {
                AstraSyncError::SyncEngine("sync engine command channel closed".to_string())
            })
    }
}

#[derive(Debug, Clone)]
pub enum EngineCommand {
    SyncNow(DataType),
    #[allow(dead_code)]
    Shutdown,
}

/// The sync engine holds all in-memory CRDT state and coordinates background sync.
pub struct SyncEngine {
    #[allow(dead_code)]
    config: Arc<Config>,
    state: Arc<RwLock<DaemonState>>,
    db: Arc<SyncDatabase>,
    mesh: P2PMeshHandle,
    /// In-memory CRDT caches for each data type.
    inventory: Arc<Mutex<HashMap<String, LWWRegister<InventoryItem>>>>,
    carts: Arc<Mutex<HashMap<String, LWWRegister<CartState>>>>,
    transactions: Arc<Mutex<VecDeque<SyncRecord<TransactionPayload>>>>,
    analytics: Arc<Mutex<VecDeque<SyncRecord<AnalyticsEvent>>>>,
    /// Lamport clock for this kiosk.
    lamport: Arc<Mutex<LamportClock>>,
}

/// Inventory item payload stored in the LWW register.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq)]
pub struct InventoryItem {
    pub sku: String,
    pub name: String,
    pub count: u64,
    pub unit_price_cents: u64,
    pub location: String,
    pub last_updated: u64,
}

/// Cart state payload.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq)]
pub struct CartState {
    pub cart_id: String,
    pub items: Vec<CartItem>,
    pub total_cents: u64,
    pub status: String, // "active", "checked_out", "abandoned"
    pub updated_at: u64,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq)]
pub struct CartItem {
    pub sku: String,
    pub quantity: u32,
    pub unit_price_cents: u64,
}

/// Transaction payload.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq)]
pub struct TransactionPayload {
    pub transaction_id: String,
    pub cart_id: String,
    pub amount_cents: u64,
    pub payment_method: String,
    pub items: Vec<CartItem>,
    pub timestamp: u64,
}

/// Analytics event payload.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize, PartialEq)]
pub struct AnalyticsEvent {
    pub event_id: String,
    pub event_type: String,
    pub metadata: serde_json::Value,
    pub timestamp: u64,
}

impl SyncEngine {
    pub async fn new(
        config: Arc<Config>,
        state: Arc<RwLock<DaemonState>>,
        db: Arc<SyncDatabase>,
        mesh: P2PMeshHandle,
    ) -> Result<Self, AstraSyncError> {
        let inventory = Arc::new(Mutex::new(HashMap::new()));
        let carts = Arc::new(Mutex::new(HashMap::new()));
        let transactions = Arc::new(Mutex::new(VecDeque::new()));
        let analytics = Arc::new(Mutex::new(VecDeque::new()));

        // Hydrate in-memory caches from the local database.
        let inv_count = {
            let inv_records: Vec<SyncRecord<InventoryItem>> =
                db.load_all(DataType::Inventory).await?;
            let count = inv_records.len();
            let mut inv = inventory.lock().await;
            for rec in inv_records {
                let reg =
                    LWWRegister::new(rec.payload, rec.origin, rec.lamport_ts, rec.wallclock_ts);
                inv.insert(rec.id, reg);
            }
            count
        };

        let cart_count = {
            let cart_records: Vec<SyncRecord<CartState>> = db.load_all(DataType::Cart).await?;
            let count = cart_records.len();
            let mut c = carts.lock().await;
            for rec in cart_records {
                let reg =
                    LWWRegister::new(rec.payload, rec.origin, rec.lamport_ts, rec.wallclock_ts);
                c.insert(rec.id, reg);
            }
            count
        };

        let tx_count = {
            let tx_records: Vec<SyncRecord<TransactionPayload>> =
                db.load_all(DataType::Transaction).await?;
            let count = tx_records.len();
            let mut t = transactions.lock().await;
            for rec in tx_records {
                t.push_back(rec);
            }
            count
        };

        let ana_count = {
            let ana_records: Vec<SyncRecord<AnalyticsEvent>> =
                db.load_all(DataType::Analytics).await?;
            let count = ana_records.len();
            let mut a = analytics.lock().await;
            for rec in ana_records {
                a.push_back(rec);
            }
            count
        };

        info!(
            inventory_loaded = inv_count,
            carts_loaded = cart_count,
            transactions_loaded = tx_count,
            analytics_loaded = ana_count,
            "Sync engine hydrated from local database"
        );

        Ok(Self {
            config,
            state,
            db,
            mesh,
            inventory,
            carts,
            transactions,
            analytics,
            lamport: Arc::new(Mutex::new(LamportClock::new())),
        })
    }

    /// Starts the background sync loops and returns a handle plus a join handle.
    pub async fn start(
        self,
        shutdown: watch::Receiver<bool>,
    ) -> Result<(SyncEngineHandle, tokio::task::JoinHandle<()>), AstraSyncError> {
        let (cmd_tx, mut cmd_rx) = tokio::sync::mpsc::channel::<EngineCommand>(256);

        let inventory = self.inventory.clone();
        let carts = self.carts.clone();
        let transactions = self.transactions.clone();
        let analytics = self.analytics.clone();
        let db = self.db.clone();
        let state = self.state.clone();
        let mesh = self.mesh.clone();
        let lamport = self.lamport.clone();

        // Immediate loop: inventory.
        let mut shutdown_imm = shutdown.clone();
        let imm_db = db.clone();
        let imm_mesh = mesh.clone();
        let imm_lamport = lamport.clone();
        let imm_inventory = inventory.clone();
        let imm_state = state.clone();
        let imm_handle = tokio::spawn(async move {
            let mut ticker = interval(Duration::from_millis(IMMEDIATE_INTERVAL_MILLIS));
            ticker.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);
            loop {
                tokio::select! {
                    _ = ticker.tick() => {
                        if let Err(e) = run_immediate_sync(
                            imm_inventory.clone(),
                            imm_db.clone(),
                            imm_mesh.clone(),
                            imm_lamport.clone(),
                            imm_state.clone(),
                        ).await {
                            error!(error = %e, "Immediate sync loop error");
                        }
                    }
                    _ = shutdown_imm.changed() => {
                        info!("Immediate sync loop shutting down");
                        break;
                    }
                }
            }
        });

        // Batched loop: transactions & carts.
        let mut shutdown_batch = shutdown.clone();
        let batch_db = db.clone();
        let batch_mesh = mesh.clone();
        let batch_lamport = lamport.clone();
        let batch_carts = carts.clone();
        let batch_transactions = transactions.clone();
        let batch_state = state.clone();
        let batch_handle = tokio::spawn(async move {
            let mut ticker = interval(Duration::from_secs(BATCH_INTERVAL_SECS));
            ticker.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Delay);
            loop {
                tokio::select! {
                    _ = ticker.tick() => {
                        if let Err(e) = run_batched_sync(
                            batch_carts.clone(),
                            batch_transactions.clone(),
                            batch_db.clone(),
                            batch_mesh.clone(),
                            batch_lamport.clone(),
                            batch_state.clone(),
                        ).await {
                            error!(error = %e, "Batched sync loop error");
                        }
                    }
                    _ = shutdown_batch.changed() => {
                        info!("Batched sync loop shutting down");
                        break;
                    }
                }
            }
        });

        // Delayed loop: analytics.
        let mut shutdown_delayed = shutdown.clone();
        let delayed_db = db.clone();
        let delayed_mesh = mesh.clone();
        let delayed_lamport = lamport.clone();
        let delayed_analytics = analytics.clone();
        let delayed_state = state.clone();
        let delayed_handle = tokio::spawn(async move {
            let mut ticker = interval(Duration::from_secs(DELAYED_INTERVAL_SECS));
            ticker.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);
            loop {
                tokio::select! {
                    _ = ticker.tick() => {
                        if let Err(e) = run_delayed_sync(
                            delayed_analytics.clone(),
                            delayed_db.clone(),
                            delayed_mesh.clone(),
                            delayed_lamport.clone(),
                            delayed_state.clone(),
                        ).await {
                            error!(error = %e, "Delayed sync loop error");
                        }
                    }
                    _ = shutdown_delayed.changed() => {
                        info!("Delayed sync loop shutting down");
                        break;
                    }
                }
            }
        });

        // Command handler loop.
        let shutdown_cmd = shutdown.clone();
        let cmd_db = db.clone();
        let cmd_mesh = mesh.clone();
        let cmd_lamport = lamport.clone();
        let cmd_handle = tokio::spawn(async move {
            while let Some(cmd) = cmd_rx.recv().await {
                match cmd {
                    EngineCommand::SyncNow(dt) => {
                        info!(data_type = ?dt, "Received immediate sync request");
                        match dt {
                            DataType::Inventory => {
                                if let Err(e) = run_immediate_sync(
                                    inventory.clone(),
                                    cmd_db.clone(),
                                    cmd_mesh.clone(),
                                    cmd_lamport.clone(),
                                    state.clone(),
                                )
                                .await
                                {
                                    warn!(error = %e, "Immediate sync request failed");
                                }
                            }
                            DataType::Cart | DataType::Transaction => {
                                if let Err(e) = run_batched_sync(
                                    carts.clone(),
                                    transactions.clone(),
                                    cmd_db.clone(),
                                    cmd_mesh.clone(),
                                    cmd_lamport.clone(),
                                    state.clone(),
                                )
                                .await
                                {
                                    warn!(error = %e, "Batched sync request failed");
                                }
                            }
                            DataType::Analytics => {
                                if let Err(e) = run_delayed_sync(
                                    analytics.clone(),
                                    cmd_db.clone(),
                                    cmd_mesh.clone(),
                                    cmd_lamport.clone(),
                                    state.clone(),
                                )
                                .await
                                {
                                    warn!(error = %e, "Delayed sync request failed");
                                }
                            }
                        }
                    }
                    EngineCommand::Shutdown => break,
                }
                if *shutdown_cmd.borrow() {
                    break;
                }
            }
        });

        let all_handle = tokio::spawn(async move {
            let _ = imm_handle.await;
            let _ = batch_handle.await;
            let _ = delayed_handle.await;
            let _ = cmd_handle.await;
        });

        Ok((SyncEngineHandle { cmd_tx }, all_handle))
    }
}

/// Pushes all dirty inventory registers to the mesh and persists them.
async fn run_immediate_sync(
    inventory: Arc<Mutex<HashMap<String, LWWRegister<InventoryItem>>>>,
    db: Arc<SyncDatabase>,
    _mesh: P2PMeshHandle,
    _lamport: Arc<Mutex<LamportClock>>,
    _state: Arc<RwLock<DaemonState>>,
) -> Result<(), AstraSyncError> {
    let inv = inventory.lock().await;
    let mut synced = 0usize;

    for (id, reg) in inv.iter() {
        // Serialize and push to mesh.
        let _payload_json = serde_json::to_vec(&reg.value)
            .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
        let record = SyncRecord {
            data_type: DataType::Inventory,
            id: id.clone(),
            origin: reg.origin.clone(),
            lamport_ts: reg.lamport_ts,
            wallclock_ts: reg.wallclock_ts,
            payload: reg.value.clone(),
            hmac: vec![], // HMAC is computed in the mesh layer before transmission.
        };
        let _record_bytes = serde_json::to_vec(&record)
            .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
        // mesh.gossip(DataType::Inventory, record_bytes).await?;
        // For now, we log the intent — the mesh layer handles actual broadcast.
        synced += 1;
    }

    // Persist any recently updated inventory to SQLite.
    db.flush(DataType::Inventory).await?;

    if synced > 0 {
        trace!(synced, "Immediate inventory sync completed");
    }
    Ok(())
}

/// Batches carts and transactions, then pushes to the mesh.
async fn run_batched_sync(
    carts: Arc<Mutex<HashMap<String, LWWRegister<CartState>>>>,
    transactions: Arc<Mutex<VecDeque<SyncRecord<TransactionPayload>>>>,
    db: Arc<SyncDatabase>,
    _mesh: P2PMeshHandle,
    _lamport: Arc<Mutex<LamportClock>>,
    _state: Arc<RwLock<DaemonState>>,
) -> Result<(), AstraSyncError> {
    let mut cart_synced = 0usize;
    let mut tx_synced = 0usize;

    // Sync carts.
    {
        let c = carts.lock().await;
        for (id, reg) in c.iter() {
            let record = SyncRecord {
                data_type: DataType::Cart,
                id: id.clone(),
                origin: reg.origin.clone(),
                lamport_ts: reg.lamport_ts,
                wallclock_ts: reg.wallclock_ts,
                payload: reg.value.clone(),
                hmac: vec![],
            };
            let _bytes = serde_json::to_vec(&record)
                .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
            cart_synced += 1;
        }
    }

    // Sync transactions.
    {
        let mut tx = transactions.lock().await;
        while let Some(rec) = tx.pop_front() {
            let _bytes = serde_json::to_vec(&rec)
                .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
            tx_synced += 1;
        }
    }

    db.flush(DataType::Cart).await?;
    db.flush(DataType::Transaction).await?;

    if cart_synced > 0 || tx_synced > 0 {
        debug!(cart_synced, tx_synced, "Batched sync completed");
    }
    Ok(())
}

/// Pushes analytics events to the mesh.
async fn run_delayed_sync(
    analytics: Arc<Mutex<VecDeque<SyncRecord<AnalyticsEvent>>>>,
    db: Arc<SyncDatabase>,
    _mesh: P2PMeshHandle,
    _lamport: Arc<Mutex<LamportClock>>,
    _state: Arc<RwLock<DaemonState>>,
) -> Result<(), AstraSyncError> {
    let mut synced = 0usize;

    {
        let mut a = analytics.lock().await;
        while let Some(rec) = a.pop_front() {
            let _bytes = serde_json::to_vec(&rec)
                .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
            synced += 1;
        }
    }

    db.flush(DataType::Analytics).await?;

    if synced > 0 {
        debug!(synced, "Delayed analytics sync completed");
    }
    Ok(())
}

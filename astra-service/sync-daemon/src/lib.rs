pub mod config;
pub mod crypto;
pub mod cloud;
pub mod crdt;
pub mod differential_privacy;
pub mod grpc;
pub mod network;
pub mod offline;
pub mod p2p;
pub mod protocol;
pub mod raft;
pub mod storage;
pub mod store;
pub mod sync;
pub mod telemetry;
pub mod verifone;

use std::fmt;

/// AstraSync daemon version, exposed for health checks and logging.
pub const VERSION: &str = env!("CARGO_PKG_VERSION");

/// Unique identifier for a kiosk peer in the mesh network.
#[derive(Debug, Clone, PartialEq, Eq, Hash, serde::Serialize, serde::Deserialize)]
pub struct KioskId(pub String);

impl fmt::Display for KioskId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl From<String> for KioskId {
    fn from(s: String) -> Self {
        Self(s)
    }
}

impl From<&str> for KioskId {
    fn from(s: &str) -> Self {
        Self(s.to_string())
    }
}

/// Unified error type for the Astra sync daemon.
#[derive(thiserror::Error, Debug)]
pub enum AstraSyncError {
    #[error("Configuration error: {0}")]
    Config(String),

    #[error("P2P network error: {0}")]
    P2P(String),

    #[error("Storage error: {0}")]
    Storage(String),

    #[error("Cryptographic error: {0}")]
    Crypto(String),

    #[error("Sync engine error: {0}")]
    SyncEngine(String),

    #[error("Raft consensus error: {0}")]
    Raft(String),

    #[error("Cloud sync error: {0}")]
    Cloud(String),

    #[error("gRPC error: {0}")]
    Grpc(String),

    #[error("Verifone terminal error: {0}")]
    Verifone(String),

    #[error("Differential privacy error: {0}")]
    DifferentialPrivacy(String),

    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),

    #[error("Database error: {0}")]
    Database(String),

    #[error("Serialization error: {0}")]
    Serialization(String),

    #[error("Shutdown signal received")]
    Shutdown,
}

impl From<rusqlite::Error> for AstraSyncError {
    fn from(err: rusqlite::Error) -> Self {
        AstraSyncError::Storage(err.to_string())
    }
}

/// Result type alias for AstraSync operations.
pub type AstraResult<T> = Result<T, AstraSyncError>;

/// Data types supported by the sync engine.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, serde::Serialize, serde::Deserialize)]
#[repr(u8)]
pub enum DataType {
    Inventory = 0,
    Cart = 1,
    Transaction = 2,
    Analytics = 3,
}

impl DataType {
    /// Returns the sync priority associated with this data type.
    pub fn priority(&self) -> SyncPriority {
        match self {
            DataType::Inventory => SyncPriority::Immediate,
            DataType::Transaction => SyncPriority::Batched,
            DataType::Cart => SyncPriority::Batched,
            DataType::Analytics => SyncPriority::Delayed,
        }
    }

    /// Returns a human-readable name for this data type.
    pub fn as_str(&self) -> &'static str {
        match self {
            DataType::Inventory => "inventory",
            DataType::Cart => "cart",
            DataType::Transaction => "transaction",
            DataType::Analytics => "analytics",
        }
    }
}

impl fmt::Display for DataType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// Sync priority levels determine how aggressively data is propagated
/// through the mesh and to the cloud backend.
#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum SyncPriority {
    /// Immediate: inventory counts, critical stock updates.
    Immediate = 0,
    /// Batched: transactions, cart state — synced every 5 seconds.
    Batched = 1,
    /// Delayed: analytics, telemetry — synced when bandwidth permits.
    Delayed = 2,
}

/// A generic record wrapper that carries CRDT metadata and a typed payload.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct SyncRecord<T> {
    /// The data type this record represents.
    pub data_type: DataType,
    /// Unique record identifier (UUID v4).
    pub id: String,
    /// Kiosk that produced this record.
    pub origin: KioskId,
    /// Lamport timestamp for causal ordering.
    pub lamport_ts: u64,
    /// Wall-clock timestamp (millis since epoch).
    pub wallclock_ts: u64,
    /// The typed payload — serialized to JSON at rest.
    pub payload: T,
    /// HMAC-SHA256 signature over the canonical bytes of the record.
    pub hmac: Vec<u8>,
}

/// Canonical bytes used for HMAC computation and deterministic hashing.
/// The payload is serialized to JSON before signing so that the signature
/// is stable across serialization formats.
impl<T: serde::Serialize> SyncRecord<T> {
    /// Computes the canonical byte representation of this record for signing.
    pub fn canonical_bytes(&self) -> Result<Vec<u8>, AstraSyncError> {
        let mut buf = Vec::new();
        buf.extend_from_slice(self.id.as_bytes());
        buf.extend_from_slice(self.origin.0.as_bytes());
        buf.extend_from_slice(&self.lamport_ts.to_be_bytes());
        buf.extend_from_slice(&self.wallclock_ts.to_be_bytes());
        let payload_json = serde_json::to_vec(&self.payload)
            .map_err(|e| AstraSyncError::Serialization(e.to_string()))?;
        buf.extend_from_slice(&payload_json);
        Ok(buf)
    }
}

/// Shared state that every sub-system holds a clone of via `Arc<...>`.
#[derive(Debug, Clone)]
pub struct DaemonState {
    /// The unique identity of this kiosk.
    pub kiosk_id: KioskId,
    /// Whether this kiosk is currently connected to the internet.
    pub online: bool,
    /// Whether this kiosk is the Raft-elected leader.
    pub is_leader: bool,
    /// Current Raft term (monotonic).
    pub raft_term: u64,
    /// Time the daemon started (millis since epoch).
    pub started_at: u64,
}

impl DaemonState {
    pub fn new(kiosk_id: KioskId) -> Self {
        Self {
            kiosk_id,
            online: false,
            is_leader: false,
            raft_term: 0,
            started_at: chrono::Utc::now().timestamp_millis() as u64,
        }
    }
}

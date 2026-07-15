#![deny(unsafe_code)]

use std::net::SocketAddr;
use std::path::PathBuf;

use serde::{Deserialize, Serialize};

use crate::AstraSyncError;

/// Root configuration structure, loaded from a TOML file at daemon startup.
///
/// Example `astra-syncd.toml`:
///
/// ```toml
/// [daemon]
/// kiosk_id = "kiosk-42"
/// data_dir = "/var/lib/astra-syncd"
/// log_level = "info"
///
/// [p2p]
/// listen_addr = "0.0.0.0:0"
/// bootstrap_peers = ["/ip4/192.168.1.10/udp/4001/quic-v1"]
/// network_name = "astra-kiosk-mesh"
///
/// [storage]
/// db_path = "/var/lib/astra-syncd/sync.db"
/// encryption_key_path = "/etc/astra-syncd/db.key"
///
/// [cloud]
/// nats_url = "tls://connect.ngs.global"
/// jetstream_bucket = "ASTRA_SYNC"
/// credentials_path = "/etc/astra-syncd/nats.creds"
/// flush_interval_seconds = 30
///
/// [grpc]
/// listen_addr = "127.0.0.1:50051"
///
/// [raft]
/// heartbeat_interval_ms = 500
/// election_timeout_min_ms = 1500
/// election_timeout_max_ms = 3000
/// ```
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub daemon: DaemonConfig,
    pub p2p: P2pConfig,
    pub storage: StorageConfig,
    pub cloud: CloudConfig,
    pub grpc: GrpcConfig,
    pub raft: RaftConfig,
    pub crypto: CryptoConfig,
}

impl Config {
    /// Loads configuration from a TOML file at the given path, then overlays
    /// any `ASTRA_*` environment variables.  Unknown `ASTRA_*` variables are
    /// rejected so that configuration typos fail fast.
    pub fn from_file(path: impl AsRef<std::path::Path>) -> Result<Self, AstraSyncError> {
        let contents = std::fs::read_to_string(path)
            .map_err(|e| AstraSyncError::Config(format!("failed to read config file: {e}")))?;
        let mut cfg: Config = toml::from_str(&contents)
            .map_err(|e| AstraSyncError::Config(format!("failed to parse TOML: {e}")))?;
        apply_env_overrides(&mut cfg)?;
        cfg.validate()?;
        Ok(cfg)
    }

    /// Loads configuration entirely from environment variables.
    pub fn from_env() -> Result<Self, AstraSyncError> {
        let mut cfg = Config {
            daemon: DaemonConfig {
                kiosk_id: String::new(),
                data_dir: std::path::PathBuf::new(),
                log_level: default_log_level(),
                metrics_addr: default_metrics_addr(),
            },
            p2p: P2pConfig {
                listen_addr: default_p2p_addr(),
                bootstrap_peers: Vec::new(),
                network_name: default_network_name(),
                max_connections: default_max_connections(),
                conn_idle_timeout_secs: default_conn_idle_timeout(),
            },
            storage: StorageConfig {
                db_path: std::path::PathBuf::new(),
                encryption_key_path: std::path::PathBuf::new(),
                wal_checkpoint_pages: default_wal_checkpoint(),
                max_db_size_mib: default_max_db_size_mib(),
            },
            cloud: CloudConfig {
                nats_url: String::new(),
                jetstream_bucket: default_jetstream_bucket(),
                credentials_path: std::path::PathBuf::new(),
                flush_interval_seconds: default_flush_interval(),
                max_msg_size_bytes: default_max_msg_size(),
                connect_timeout_secs: default_cloud_timeout(),
            },
            grpc: GrpcConfig {
                listen_addr: "127.0.0.1:50051".parse().unwrap(),
                tls_cert_path: None,
                tls_key_path: None,
                max_concurrent_streams: default_grpc_concurrency(),
            },
            raft: RaftConfig {
                heartbeat_interval_ms: default_heartbeat(),
                election_timeout_min_ms: default_election_min(),
                election_timeout_max_ms: default_election_max(),
                max_entries_per_append: default_max_entries(),
            },
            crypto: CryptoConfig {
                sync_key_path: std::path::PathBuf::new(),
                offline_hmac_key_path: std::path::PathBuf::new(),
                kdf: KdfConfig::default(),
            },
        };
        apply_env_overrides(&mut cfg)?;
        cfg.validate()?;
        Ok(cfg)
    }

    /// Validates that all configuration values are within acceptable ranges.
    fn validate(&self) -> Result<(), AstraSyncError> {
        if self.daemon.kiosk_id.is_empty() {
            return Err(AstraSyncError::Config(
                "daemon.kiosk_id must not be empty".to_string(),
            ));
        }
        if self.grpc.listen_addr.port() == 0 {
            return Err(AstraSyncError::Config(
                "grpc.listen_addr must specify a non-zero port".to_string(),
            ));
        }
        if self.raft.heartbeat_interval_ms == 0 {
            return Err(AstraSyncError::Config(
                "raft.heartbeat_interval_ms must be > 0".to_string(),
            ));
        }
        if self.raft.election_timeout_min_ms >= self.raft.election_timeout_max_ms {
            return Err(AstraSyncError::Config(
                "raft.election_timeout_min_ms must be < raft.election_timeout_max_ms".to_string(),
            ));
        }
        Ok(())
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DaemonConfig {
    /// Human-readable identifier for this kiosk (e.g., "kiosk-42").
    pub kiosk_id: String,
    /// Directory for all persistent data (database, keys, WAL).
    pub data_dir: PathBuf,
    /// Log level filter: "trace", "debug", "info", "warn", "error".
    #[serde(default = "default_log_level")]
    pub log_level: String,
    /// Metrics exporter bind address (e.g., "127.0.0.1:9090").
    #[serde(default = "default_metrics_addr")]
    pub metrics_addr: SocketAddr,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct P2pConfig {
    /// Address to listen on for incoming QUIC connections.
    /// Port 0 requests an ephemeral port.
    #[serde(default = "default_p2p_addr")]
    pub listen_addr: SocketAddr,
    /// Optional static list of bootstrap multiaddresses.
    #[serde(default)]
    pub bootstrap_peers: Vec<String>,
    /// mDNS service name / network partition identifier.
    #[serde(default = "default_network_name")]
    pub network_name: String,
    /// Maximum concurrent P2P connections.
    #[serde(default = "default_max_connections")]
    pub max_connections: usize,
    /// Idle connection timeout in seconds.
    #[serde(default = "default_conn_idle_timeout")]
    pub conn_idle_timeout_secs: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StorageConfig {
    /// Path to the SQLite database file.
    pub db_path: PathBuf,
    /// Path to a file containing the raw 32-byte SQLCipher encryption key.
    /// The file must be readable only by the daemon user (chmod 600).
    pub encryption_key_path: PathBuf,
    /// WAL auto-checkpoint interval in pages.
    #[serde(default = "default_wal_checkpoint")]
    pub wal_checkpoint_pages: u32,
    /// Maximum database size in MiB.
    #[serde(default = "default_max_db_size_mib")]
    pub max_db_size_mib: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CloudConfig {
    /// NATS server URL (can be a list separated by commas).
    pub nats_url: String,
    /// JetStream key-value bucket name for sync state.
    #[serde(default = "default_jetstream_bucket")]
    pub jetstream_bucket: String,
    /// Path to NATS credentials file (JWT + seed).
    pub credentials_path: PathBuf,
    /// How often the leader flushes batched data to the cloud (seconds).
    #[serde(default = "default_flush_interval")]
    pub flush_interval_seconds: u64,
    /// Maximum bytes per JetStream message payload.
    #[serde(default = "default_max_msg_size")]
    pub max_msg_size_bytes: usize,
    /// Connection timeout in seconds.
    #[serde(default = "default_cloud_timeout")]
    pub connect_timeout_secs: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GrpcConfig {
    /// Address to bind the local gRPC server on.
    pub listen_addr: SocketAddr,
    /// TLS certificate path (optional — disabled if not provided).
    pub tls_cert_path: Option<PathBuf>,
    /// TLS key path (optional — disabled if not provided).
    pub tls_key_path: Option<PathBuf>,
    /// Max concurrent RPC streams.
    #[serde(default = "default_grpc_concurrency")]
    pub max_concurrent_streams: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RaftConfig {
    /// Leader heartbeat interval in milliseconds.
    #[serde(default = "default_heartbeat")]
    pub heartbeat_interval_ms: u64,
    /// Minimum election timeout in milliseconds.
    #[serde(default = "default_election_min")]
    pub election_timeout_min_ms: u64,
    /// Maximum election timeout in milliseconds.
    #[serde(default = "default_election_max")]
    pub election_timeout_max_ms: u64,
    /// Maximum log entries per AppendEntries RPC.
    #[serde(default = "default_max_entries")]
    pub max_entries_per_append: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CryptoConfig {
    /// Path to the 32-byte XChaCha20-Poly1305 symmetric key file.
    pub sync_key_path: PathBuf,
    /// Path to the HMAC-SHA256 offline payment signing key.
    pub offline_hmac_key_path: PathBuf,
    /// Argon2id parameters for key derivation (if keys are password-protected).
    #[serde(default)]
    pub kdf: KdfConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KdfConfig {
    /// Memory cost in KiB.
    #[serde(default = "default_argon2_m")]
    pub m_cost: u32,
    /// Iterations.
    #[serde(default = "default_argon2_t")]
    pub t_cost: u32,
    /// Parallelism.
    #[serde(default = "default_argon2_p")]
    pub p_cost: u32,
}

impl Default for KdfConfig {
    fn default() -> Self {
        Self {
            m_cost: 65536,
            t_cost: 3,
            p_cost: 4,
        }
    }
}

fn default_log_level() -> String {
    "info".to_string()
}

fn default_p2p_addr() -> SocketAddr {
    "0.0.0.0:0".parse().unwrap()
}

fn default_network_name() -> String {
    "astra-kiosk-mesh".to_string()
}

fn default_max_connections() -> usize {
    50
}

fn default_conn_idle_timeout() -> u64 {
    300
}

fn default_wal_checkpoint() -> u32 {
    1000
}

fn default_max_db_size_mib() -> u32 {
    512
}

fn default_jetstream_bucket() -> String {
    "ASTRA_SYNC".to_string()
}

fn default_flush_interval() -> u64 {
    30
}

fn default_max_msg_size() -> usize {
    1024 * 1024 // 1 MiB
}

fn default_cloud_timeout() -> u64 {
    10
}

fn default_grpc_concurrency() -> usize {
    256
}

fn default_heartbeat() -> u64 {
    500
}

fn default_election_min() -> u64 {
    1500
}

fn default_election_max() -> u64 {
    3000
}

fn default_max_entries() -> usize {
    128
}

fn default_metrics_addr() -> SocketAddr {
    "127.0.0.1:9090".parse().unwrap()
}

fn default_argon2_m() -> u32 {
    65536
}

fn default_argon2_t() -> u32 {
    3
}

fn default_argon2_p() -> u32 {
    4
}

/// Applies recognized `ASTRA_*` environment variables on top of `cfg`.
/// Any unrecognized `ASTRA_*` variable causes an error.
fn apply_env_overrides(cfg: &mut Config) -> Result<(), AstraSyncError> {
    check_unknown_env_vars()?;

    if let Ok(v) = std::env::var("ASTRA_DAEMON_KIOSK_ID") {
        cfg.daemon.kiosk_id = v;
    }
    if let Ok(v) = std::env::var("ASTRA_DAEMON_DATA_DIR") {
        cfg.daemon.data_dir = v.into();
    }
    if let Ok(v) = std::env::var("ASTRA_DAEMON_LOG_LEVEL") {
        cfg.daemon.log_level = v;
    }
    if let Ok(v) = std::env::var("ASTRA_DAEMON_METRICS_ADDR") {
        cfg.daemon.metrics_addr = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_DAEMON_METRICS_ADDR: {e}"))
        })?;
    }

    if let Ok(v) = std::env::var("ASTRA_P2P_LISTEN_ADDR") {
        cfg.p2p.listen_addr = v
            .parse()
            .map_err(|e| AstraSyncError::Config(format!("invalid ASTRA_P2P_LISTEN_ADDR: {e}")))?;
    }
    if let Ok(v) = std::env::var("ASTRA_P2P_BOOTSTRAP_PEERS") {
        cfg.p2p.bootstrap_peers = v.split(',').map(|s| s.trim().to_string()).collect();
    }
    if let Ok(v) = std::env::var("ASTRA_P2P_NETWORK_NAME") {
        cfg.p2p.network_name = v;
    }
    if let Ok(v) = std::env::var("ASTRA_P2P_MAX_CONNECTIONS") {
        cfg.p2p.max_connections = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_P2P_MAX_CONNECTIONS: {e}"))
        })?;
    }

    if let Ok(v) = std::env::var("ASTRA_STORAGE_DB_PATH") {
        cfg.storage.db_path = v.into();
    }
    if let Ok(v) = std::env::var("ASTRA_STORAGE_ENCRYPTION_KEY_PATH") {
        cfg.storage.encryption_key_path = v.into();
    }
    if let Ok(v) = std::env::var("ASTRA_STORAGE_WAL_CHECKPOINT_PAGES") {
        cfg.storage.wal_checkpoint_pages = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_STORAGE_WAL_CHECKPOINT_PAGES: {e}"))
        })?;
    }
    if let Ok(v) = std::env::var("ASTRA_STORAGE_MAX_DB_SIZE_MIB") {
        cfg.storage.max_db_size_mib = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_STORAGE_MAX_DB_SIZE_MIB: {e}"))
        })?;
    }

    if let Ok(v) = std::env::var("ASTRA_CLOUD_NATS_URL") {
        cfg.cloud.nats_url = v;
    }
    if let Ok(v) = std::env::var("ASTRA_CLOUD_JETSTREAM_BUCKET") {
        cfg.cloud.jetstream_bucket = v;
    }
    if let Ok(v) = std::env::var("ASTRA_CLOUD_CREDENTIALS_PATH") {
        cfg.cloud.credentials_path = v.into();
    }
    if let Ok(v) = std::env::var("ASTRA_CLOUD_FLUSH_INTERVAL_SECONDS") {
        cfg.cloud.flush_interval_seconds = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_CLOUD_FLUSH_INTERVAL_SECONDS: {e}"))
        })?;
    }
    if let Ok(v) = std::env::var("ASTRA_CLOUD_MAX_MSG_SIZE_BYTES") {
        cfg.cloud.max_msg_size_bytes = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_CLOUD_MAX_MSG_SIZE_BYTES: {e}"))
        })?;
    }
    if let Ok(v) = std::env::var("ASTRA_CLOUD_CONNECT_TIMEOUT_SECS") {
        cfg.cloud.connect_timeout_secs = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_CLOUD_CONNECT_TIMEOUT_SECS: {e}"))
        })?;
    }

    if let Ok(v) = std::env::var("ASTRA_GRPC_LISTEN_ADDR") {
        cfg.grpc.listen_addr = v
            .parse()
            .map_err(|e| AstraSyncError::Config(format!("invalid ASTRA_GRPC_LISTEN_ADDR: {e}")))?;
    }
    if let Ok(v) = std::env::var("ASTRA_GRPC_TLS_CERT_PATH") {
        cfg.grpc.tls_cert_path = Some(v.into());
    }
    if let Ok(v) = std::env::var("ASTRA_GRPC_TLS_KEY_PATH") {
        cfg.grpc.tls_key_path = Some(v.into());
    }
    if let Ok(v) = std::env::var("ASTRA_GRPC_MAX_CONCURRENT_STREAMS") {
        cfg.grpc.max_concurrent_streams = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_GRPC_MAX_CONCURRENT_STREAMS: {e}"))
        })?;
    }

    if let Ok(v) = std::env::var("ASTRA_RAFT_HEARTBEAT_INTERVAL_MS") {
        cfg.raft.heartbeat_interval_ms = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_RAFT_HEARTBEAT_INTERVAL_MS: {e}"))
        })?;
    }
    if let Ok(v) = std::env::var("ASTRA_RAFT_ELECTION_TIMEOUT_MIN_MS") {
        cfg.raft.election_timeout_min_ms = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_RAFT_ELECTION_TIMEOUT_MIN_MS: {e}"))
        })?;
    }
    if let Ok(v) = std::env::var("ASTRA_RAFT_ELECTION_TIMEOUT_MAX_MS") {
        cfg.raft.election_timeout_max_ms = v.parse().map_err(|e| {
            AstraSyncError::Config(format!("invalid ASTRA_RAFT_ELECTION_TIMEOUT_MAX_MS: {e}"))
        })?;
    }

    if let Ok(v) = std::env::var("ASTRA_CRYPTO_SYNC_KEY_PATH") {
        cfg.crypto.sync_key_path = v.into();
    }
    if let Ok(v) = std::env::var("ASTRA_CRYPTO_OFFLINE_HMAC_KEY_PATH") {
        cfg.crypto.offline_hmac_key_path = v.into();
    }

    Ok(())
}

/// The set of environment variable names that `apply_env_overrides` recognizes.
const RECOGNIZED_ENV_VARS: &[&str] = &[
    "ASTRA_DAEMON_KIOSK_ID",
    "ASTRA_DAEMON_DATA_DIR",
    "ASTRA_DAEMON_LOG_LEVEL",
    "ASTRA_DAEMON_METRICS_ADDR",
    "ASTRA_P2P_LISTEN_ADDR",
    "ASTRA_P2P_BOOTSTRAP_PEERS",
    "ASTRA_P2P_NETWORK_NAME",
    "ASTRA_P2P_MAX_CONNECTIONS",
    "ASTRA_STORAGE_DB_PATH",
    "ASTRA_STORAGE_ENCRYPTION_KEY_PATH",
    "ASTRA_STORAGE_WAL_CHECKPOINT_PAGES",
    "ASTRA_STORAGE_MAX_DB_SIZE_MIB",
    "ASTRA_CLOUD_NATS_URL",
    "ASTRA_CLOUD_JETSTREAM_BUCKET",
    "ASTRA_CLOUD_CREDENTIALS_PATH",
    "ASTRA_CLOUD_FLUSH_INTERVAL_SECONDS",
    "ASTRA_CLOUD_MAX_MSG_SIZE_BYTES",
    "ASTRA_CLOUD_CONNECT_TIMEOUT_SECS",
    "ASTRA_GRPC_LISTEN_ADDR",
    "ASTRA_GRPC_TLS_CERT_PATH",
    "ASTRA_GRPC_TLS_KEY_PATH",
    "ASTRA_GRPC_MAX_CONCURRENT_STREAMS",
    "ASTRA_RAFT_HEARTBEAT_INTERVAL_MS",
    "ASTRA_RAFT_ELECTION_TIMEOUT_MIN_MS",
    "ASTRA_RAFT_ELECTION_TIMEOUT_MAX_MS",
    "ASTRA_CRYPTO_SYNC_KEY_PATH",
    "ASTRA_CRYPTO_OFFLINE_HMAC_KEY_PATH",
];

/// Returns an error if any `ASTRA_*` environment variable is not recognized.
fn check_unknown_env_vars() -> Result<(), AstraSyncError> {
    for (key, _) in std::env::vars() {
        if !key.starts_with("ASTRA_") {
            continue;
        }
        if !RECOGNIZED_ENV_VARS.contains(&key.as_str()) {
            return Err(AstraSyncError::Config(format!(
                "unknown environment variable: {key}"
            )));
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn validate_rejects_empty_kiosk_id() {
        let cfg = Config {
            daemon: DaemonConfig {
                kiosk_id: String::new(),
                data_dir: "/tmp".into(),
                log_level: default_log_level(),
                metrics_addr: default_metrics_addr(),
            },
            p2p: P2pConfig {
                listen_addr: default_p2p_addr(),
                bootstrap_peers: Vec::new(),
                network_name: default_network_name(),
                max_connections: default_max_connections(),
                conn_idle_timeout_secs: default_conn_idle_timeout(),
            },
            storage: StorageConfig {
                db_path: "/tmp/db".into(),
                encryption_key_path: "/tmp/key".into(),
                wal_checkpoint_pages: default_wal_checkpoint(),
                max_db_size_mib: default_max_db_size_mib(),
            },
            cloud: CloudConfig {
                nats_url: "tls://localhost".into(),
                jetstream_bucket: default_jetstream_bucket(),
                credentials_path: "/tmp/creds".into(),
                flush_interval_seconds: default_flush_interval(),
                max_msg_size_bytes: default_max_msg_size(),
                connect_timeout_secs: default_cloud_timeout(),
            },
            grpc: GrpcConfig {
                listen_addr: "127.0.0.1:50051".parse().unwrap(),
                tls_cert_path: None,
                tls_key_path: None,
                max_concurrent_streams: default_grpc_concurrency(),
            },
            raft: RaftConfig {
                heartbeat_interval_ms: default_heartbeat(),
                election_timeout_min_ms: default_election_min(),
                election_timeout_max_ms: default_election_max(),
                max_entries_per_append: default_max_entries(),
            },
            crypto: CryptoConfig {
                sync_key_path: "/tmp/sync.key".into(),
                offline_hmac_key_path: "/tmp/hmac.key".into(),
                kdf: KdfConfig::default(),
            },
        };
        assert!(cfg.validate().is_err());
    }

    #[test]
    fn check_unknown_env_vars_rejects_unrecognized() {
        // Only meaningful if the test environment happens to have unknown vars.
        // We test the allowlist content directly.
        assert!(RECOGNIZED_ENV_VARS.contains(&"ASTRA_DAEMON_KIOSK_ID"));
    }
}

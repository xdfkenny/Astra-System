use std::path::PathBuf;
use std::sync::Arc;

use clap::{Parser, Subcommand, ValueEnum};
use tokio::sync::{watch, RwLock};
use tracing::{info, warn};

use astra_syncd::cloud::CloudSync;
use astra_syncd::config::Config;
use astra_syncd::crypto::{HmacKey, SyncKey};
use astra_syncd::grpc::GrpcServer;
use astra_syncd::p2p::mesh::P2PMesh;
use astra_syncd::raft::RaftNode;
use astra_syncd::storage::sqlite::SyncDatabase;
use astra_syncd::sync::engine::SyncEngine;
use astra_syncd::telemetry;
use astra_syncd::{DaemonState, KioskId};

/// Astra P2P Sync Daemon — production-grade mesh synchronization for Astra kiosks.
#[derive(Parser, Debug)]
#[command(name = "astra-syncd")]
#[command(version = env!("CARGO_PKG_VERSION"))]
#[command(about = "Astra P2P mesh sync daemon for kiosk deployments")]
struct Cli {
    /// Path to the TOML configuration file.
    #[arg(
        short,
        long,
        value_name = "PATH",
        default_value = "/etc/astra-syncd/config.toml"
    )]
    config: PathBuf,

    /// Override log level (trace, debug, info, warn, error).
    #[arg(short, long, env = "ASTRA_LOG_LEVEL")]
    log_level: Option<String>,

    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand, Debug)]
enum Commands {
    /// Generate a new encryption key file.
    GenKey {
        /// Output path for the generated key.
        #[arg(short, long)]
        output: PathBuf,
        /// Key type: sync (XChaCha20) or hmac (offline payments).
        #[arg(short, long, value_enum, default_value = "sync")]
        key_type: KeyType,
    },
    /// Validate configuration and exit.
    Validate,
    /// Run database migrations.
    Migrate,
}

#[derive(ValueEnum, Debug, Clone)]
enum KeyType {
    Sync,
    Hmac,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let cli = Cli::parse();

    // Handle subcommands before daemon startup.
    match cli.command {
        Some(Commands::GenKey { output, key_type }) => {
            match key_type {
                KeyType::Sync => {
                    let key = SyncKey::generate();
                    key.write_to_file(&output)?;
                }
                KeyType::Hmac => {
                    let key = HmacKey::generate();
                    std::fs::write(&output, key.as_bytes())?;
                    #[cfg(unix)]
                    {
                        use std::os::unix::fs::PermissionsExt;
                        let mut perms = std::fs::metadata(&output)?.permissions();
                        perms.set_mode(0o600);
                        std::fs::set_permissions(&output, perms)?;
                    }
                }
            }
            println!(
                "Generated {} key at {}",
                match key_type {
                    KeyType::Sync => "sync",
                    KeyType::Hmac => "hmac",
                },
                output.display()
            );
            return Ok(());
        }
        Some(Commands::Validate) => {
            let _cfg = Config::from_file(&cli.config)?;
            println!("Configuration is valid.");
            return Ok(());
        }
        Some(Commands::Migrate) => {
            let cfg = Config::from_file(&cli.config)?;
            let db = SyncDatabase::open(&cfg.storage, &cfg.crypto).await?;
            db.migrate().await?;
            println!("Database migrations applied successfully.");
            return Ok(());
        }
        None => {}
    }

    // Load configuration.
    let config = Arc::new(Config::from_file(&cli.config)?);

    // Initialize logging and tracing.
    let log_level = cli.log_level.as_deref().unwrap_or(&config.daemon.log_level);
    let telemetry_guard = telemetry::init(
        "astra-syncd",
        env!("CARGO_PKG_VERSION"),
        "production",
        log_level,
    )
    .map_err(|e| format!("failed to initialize telemetry: {e}"))?;

    // Attach operational context to all subsequent log records.
    let context = telemetry::TelemetryContext::new(
        uuid::Uuid::new_v4().to_string(),
        "lane-0",
        &config.daemon.kiosk_id,
        "tenant-0",
    );
    let _context_span = context.span("astra.daemon");
    let _enter = _context_span.enter();

    // Start Prometheus metrics endpoint.
    telemetry::metrics::start_server(config.daemon.metrics_addr)
        .map_err(|e| format!("failed to start metrics endpoint: {e}"))?;

    info!(
        version = env!("CARGO_PKG_VERSION"),
        kiosk_id = %config.daemon.kiosk_id,
        "Astra sync daemon starting"
    );

    // Shared daemon state.
    let state = Arc::new(RwLock::new(DaemonState::new(KioskId::from(
        config.daemon.kiosk_id.clone(),
    ))));

    // Setup shutdown signal handling.
    let (shutdown_tx, shutdown_rx) = watch::channel(false);
    let shutdown_tx_clone = shutdown_tx.clone();
    tokio::spawn(async move {
        match tokio::signal::ctrl_c().await {
            Ok(()) => {
                info!("Received SIGINT, initiating graceful shutdown");
                let _ = shutdown_tx_clone.send(true);
            }
            Err(e) => warn!("Failed to install SIGINT handler: {}", e),
        }
    });

    #[cfg(unix)]
    {
        let shutdown_tx_clone = shutdown_tx.clone();
        tokio::spawn(async move {
            match tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate()) {
                Ok(mut sigterm) => {
                    sigterm.recv().await;
                    info!("Received SIGTERM, initiating graceful shutdown");
                    let _ = shutdown_tx_clone.send(true);
                }
                Err(e) => {
                    warn!("Failed to install SIGTERM handler: {}", e);
                }
            }
        });
    }

    // Open encrypted local database.
    let db = Arc::new(SyncDatabase::open(&config.storage, &config.crypto).await?);
    db.migrate().await?;
    info!("Encrypted SQLite database opened and migrated");

    // Initialize crypto keys.
    let _sync_key = Arc::new(SyncKey::from_file(&config.crypto.sync_key_path)?);

    // Start P2P mesh networking.
    let p2p = P2PMesh::new(config.clone()).await?;
    let (p2p_handle, p2p_join_handle) = p2p.start(shutdown_rx.clone()).await?;

    // Start Raft consensus node.
    let raft = RaftNode::new(
        config.clone(),
        state.clone(),
        db.clone(),
        p2p_handle.clone(),
    )
    .await?;
    let raft_handle = raft.start(shutdown_rx.clone()).await?;

    // Start sync engine.
    let sync_engine = SyncEngine::new(
        config.clone(),
        state.clone(),
        db.clone(),
        p2p_handle.clone(),
    )
    .await?;
    let (sync_handle, sync_join_handle) = sync_engine.start(shutdown_rx.clone()).await?;

    // Start cloud sync (only active when leader and online).
    let cloud = CloudSync::new(config.clone(), state.clone(), db.clone()).await?;
    let cloud_handle = cloud.start(shutdown_rx.clone()).await?;

    // Start gRPC server for local IPC.
    let grpc = GrpcServer::new(
        config.clone(),
        state.clone(),
        db.clone(),
        sync_handle.clone(),
    );
    let grpc_handle = grpc.start(shutdown_rx.clone()).await?;

    info!("All subsystems started; daemon is operational");

    // Wait for shutdown signal.
    let mut shutdown_rx_main = shutdown_rx.clone();
    shutdown_rx_main.changed().await.ok();

    info!("Graceful shutdown sequence initiated");

    // Trigger shutdown on all subsystems.
    let _ = shutdown_tx.send(true);

    // Wait for all tasks to complete with timeout.
    let shutdown_timeout = tokio::time::Duration::from_secs(30);
    let shutdown_result = tokio::time::timeout(shutdown_timeout, async {
        grpc_handle.await.ok();
        cloud_handle.await.ok();
        sync_join_handle.await.ok();
        raft_handle.await.ok();
        p2p_join_handle.await.ok();
    })
    .await;

    match shutdown_result {
        Ok(_) => info!("All subsystems shut down cleanly"),
        Err(_) => warn!("Shutdown timed out; some tasks may not have completed"),
    }

    // Flush telemetry and close database.
    telemetry_guard.shutdown().await;

    db.close().await?;
    info!("Astra sync daemon stopped");

    Ok(())
}

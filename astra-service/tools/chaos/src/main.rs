//! Astra Chaos Engineering CLI.
//!
//! Randomly partitions the kiosk mesh network during integration tests to
//! validate resilience and convergence of the P2P sync layer. Also supports
//! disk pressure and memory pressure injection for edge-device resource
//! exhaustion testing.

use std::process::ExitCode;

use clap::{Parser, Subcommand};
use tracing::{error, info};

mod backend;
mod cli;
mod disk_pressure;
mod memory_pressure;
mod partition;

use cli::{DiskPressureArgs, MemoryPressureArgs, PartitionArgs, SwapPressureArgs};

/// Astra Chaos — fault injection for kiosk mesh integration tests.
#[derive(Parser, Debug)]
#[command(name = "astra-chaos")]
#[command(version = env!("CARGO_PKG_VERSION"))]
#[command(about = "Chaos engineering CLI for Astra kiosk mesh networks")]
struct Cli {
    /// Increase logging verbosity.
    #[arg(short, long, action = clap::ArgAction::Count)]
    verbose: u8,

    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand, Debug)]
enum Commands {
    /// Randomly partition selected peers from the mesh.
    Partition(PartitionArgs),
    /// Inject disk I/O pressure to test SQLite locking resilience.
    DiskPressure(DiskPressureArgs),
    /// Inject memory pressure to test WASM CRDT worker saturation.
    MemoryPressure(MemoryPressureArgs),
    /// Inject swap pressure to test kernel OOM behavior.
    SwapPressure(SwapPressureArgs),
}

fn main() -> ExitCode {
    let cli = Cli::parse();

    let log_level = match cli.verbose {
        0 => "warn",
        1 => "info",
        2 => "debug",
        _ => "trace",
    };
    tracing_subscriber::fmt()
        .with_env_filter(log_level)
        .with_writer(std::io::stderr)
        .init();

    match cli.command {
        Commands::Partition(args) => {
            if let Err(e) = partition::run(args) {
                error!(error = %e, "partition failed");
                return ExitCode::FAILURE;
            }
            info!("partition completed");
        }
        Commands::DiskPressure(args) => {
            let config = disk_pressure::DiskPressureConfig {
                target_dir: args.dir.into(),
                target_bytes: args.target_bytes,
                io_pattern: args.pattern,
                block_size: args.block_size,
                concurrency: args.concurrency,
                duration: args.duration,
                dry_run: args.dry_run,
            };
            match disk_pressure::run(&config) {
                Ok(stats) => {
                    info!(
                        bytes_written = stats.bytes_written,
                        io_errors = stats.io_errors,
                        files_created = stats.files_created,
                        "disk pressure completed"
                    );
                }
                Err(e) => {
                    error!(error = %e, "disk pressure failed");
                    return ExitCode::FAILURE;
                }
            }
        }
        Commands::MemoryPressure(args) => {
            let config = memory_pressure::MemoryPressureConfig {
                target_rss_ratio: args.ratio,
                block_size: args.block_size,
                concurrency: args.concurrency,
                duration: args.duration,
                dry_run: args.dry_run,
            };
            match memory_pressure::run(&config) {
                Ok(stats) => {
                    info!(
                        total_allocated_mib = stats.total_allocated_bytes / (1024 * 1024),
                        peak_rss_mib = stats.peak_rss_bytes / (1024 * 1024),
                        allocations = stats.allocation_count,
                        errors = stats.allocation_errors,
                        swap_triggered = stats.swap_triggered,
                        "memory pressure completed"
                    );
                }
                Err(e) => {
                    error!(error = %e, "memory pressure failed");
                    return ExitCode::FAILURE;
                }
            }
        }
        Commands::SwapPressure(args) => {
            let config = memory_pressure::SwapPressureConfig {
                pages_to_lock: args.pages,
                page_size: args.page_size,
                duration: args.duration,
                concurrency: args.concurrency,
                dry_run: args.dry_run,
            };
            match memory_pressure::run_swap_pressure(&config) {
                Ok(stats) => {
                    info!(
                        pages_locked = stats.pages_locked,
                        errors = stats.swap_errors,
                        duration_secs = stats.duration_secs,
                        "swap pressure completed"
                    );
                }
                Err(e) => {
                    error!(error = %e, "swap pressure failed");
                    return ExitCode::FAILURE;
                }
            }
        }
    }

    ExitCode::SUCCESS
}

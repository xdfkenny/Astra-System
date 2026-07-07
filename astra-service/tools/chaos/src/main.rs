//! Astra Chaos Engineering CLI.
//!
//! Randomly partitions the kiosk mesh network during integration tests to
//! validate resilience and convergence of the P2P sync layer.

use std::process::ExitCode;

use clap::{Parser, Subcommand};
use tracing::{error, info};

mod backend;
mod cli;
mod partition;

use cli::PartitionArgs;

/// Astra Chaos — network fault injection for kiosk mesh integration tests.
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
    }

    ExitCode::SUCCESS
}

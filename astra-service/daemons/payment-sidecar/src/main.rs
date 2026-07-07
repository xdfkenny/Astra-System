//! Astra Payment Sidecar entrypoint.

use clap::Parser;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let cli = astra_payment_sidecar::Cli::parse();

    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::new(&cli.log_level))
        .with_target(true)
        .with_thread_ids(true)
        .with_line_number(true)
        .json()
        .flatten_event(true)
        .init();

    astra_payment_sidecar::run(cli).await.map_err(|e| e.into())
}

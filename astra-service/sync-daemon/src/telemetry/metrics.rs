#![deny(unsafe_code)]

use std::net::SocketAddr;

use metrics_exporter_prometheus::PrometheusBuilder;

/// Starts a Prometheus metrics scrape endpoint on the supplied bind address.
///
/// The HTTP server runs on the current Tokio runtime until the process exits.
pub fn start_server(bind_addr: SocketAddr) -> Result<(), MetricsError> {
    PrometheusBuilder::new()
        .with_http_listener(bind_addr)
        .install_recorder()
        .map_err(|e| MetricsError::InstallRecorder(e.to_string()))?;
    Ok(())
}

/// Errors that can occur while starting the Prometheus metrics endpoint.
#[derive(Debug, thiserror::Error)]
pub enum MetricsError {
    #[error("failed to install Prometheus recorder: {0}")]
    InstallRecorder(String),
}

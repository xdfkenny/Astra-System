//! Astra Payment Sidecar — local bridge between the kiosk browser and the
//! Verifone payment terminal.
//!
//! This crate intentionally keeps all PCI-scoped card data inside native code.
//! The browser only exchanges JSON messages containing opaque tokens and
//! authorization status. The sidecar binds to 127.0.0.1 by default and refuses
//! to start if configured to bind to a non-loopback address in production.

pub mod api;
pub mod verifone;

use std::fmt;
use std::net::{IpAddr, SocketAddr};
use std::sync::Arc;

use clap::Parser;
use tracing::{info, warn};

use crate::verifone::SimulatedTerminal;

/// Sidecar runtime errors.
#[derive(Debug)]
pub enum PaymentError {
    InvalidRequest(String),
    Terminal(String),
    Internal(String),
}

impl fmt::Display for PaymentError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            PaymentError::InvalidRequest(msg) => write!(f, "invalid request: {msg}"),
            PaymentError::Terminal(msg) => write!(f, "terminal error: {msg}"),
            PaymentError::Internal(msg) => write!(f, "internal error: {msg}"),
        }
    }
}

impl std::error::Error for PaymentError {}

impl From<serde_json::Error> for PaymentError {
    fn from(err: serde_json::Error) -> Self {
        PaymentError::InvalidRequest(err.to_string())
    }
}

impl From<std::io::Error> for PaymentError {
    fn from(err: std::io::Error) -> Self {
        PaymentError::Terminal(err.to_string())
    }
}

/// Command-line options.
#[derive(Parser, Debug)]
#[command(name = "astra-payment-sidecar")]
#[command(version = env!("CARGO_PKG_VERSION"))]
#[command(about = "Local Verifone payment sidecar for Astra kiosks")]
pub struct Cli {
    /// IP address to bind to. Production builds require 127.0.0.1.
    #[arg(short, long, env = "ASTRA_SIDECAR_HOST", default_value = "127.0.0.1")]
    pub host: String,

    /// TCP port to listen on.
    #[arg(short, long, env = "ASTRA_SIDECAR_PORT", default_value = "8963")]
    pub port: u16,

    /// Verifone terminal URI. In simulator mode this is ignored except for logs.
    #[arg(short, long, env = "ASTRA_VERIFONE_URI", default_value = "sim://localhost")]
    pub terminal_uri: String,

    /// Allow binding to non-loopback addresses. DANGEROUS in production.
    #[arg(long, env = "ASTRA_SIDECAR_ALLOW_NON_LOOPBACK")]
    pub allow_non_loopback: bool,

    /// Log level.
    #[arg(short, long, env = "ASTRA_LOG_LEVEL", default_value = "info")]
    pub log_level: String,
}

impl Cli {
    /// Resolves and validates the listen address.
    pub fn socket_addr(&self, require_loopback: bool) -> Result<SocketAddr, PaymentError> {
        let ip: IpAddr = self
            .host
            .parse()
            .map_err(|_| PaymentError::InvalidRequest(format!("invalid host: {}", self.host)))?;

        if require_loopback && !is_loopback(ip) && !self.allow_non_loopback {
            return Err(PaymentError::InvalidRequest(
                "sidecar must bind to a loopback address in production".to_string(),
            ));
        }

        Ok(SocketAddr::new(ip, self.port))
    }
}

fn is_loopback(ip: IpAddr) -> bool {
    match ip {
        IpAddr::V4(v4) => v4.is_loopback(),
        IpAddr::V6(v6) => v6.is_loopback(),
    }
}

/// Run the sidecar with the given CLI options. This is the main entrypoint
/// used by `main.rs` and by integration tests.
pub async fn run(cli: Cli) -> Result<(), PaymentError> {
    let is_production = std::env::var("ASTRA_ENV").unwrap_or_default() == "production";
    let addr = cli.socket_addr(is_production)?;

    if !is_loopback(addr.ip()) {
        warn!(%addr, "sidecar bound to non-loopback address; this is unsafe in production");
    }

    info!(terminal_uri = %cli.terminal_uri, %addr, "starting payment sidecar");

    let terminal: Arc<dyn verifone::Terminal> =
        Arc::new(SimulatedTerminal::new(cli.terminal_uri));

    api::serve(addr, terminal)
        .await
        .map_err(|e| PaymentError::Internal(e.to_string()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_loopback_required_in_production() {
        std::env::set_var("ASTRA_ENV", "production");
        let cli = Cli {
            host: "0.0.0.0".to_string(),
            port: 8963,
            terminal_uri: "sim://localhost".to_string(),
            allow_non_loopback: false,
            log_level: "info".to_string(),
        };
        assert!(cli.socket_addr(true).is_err());
    }

    #[test]
    fn test_loopback_allowed_by_default() {
        let cli = Cli {
            host: "127.0.0.1".to_string(),
            port: 8963,
            terminal_uri: "sim://localhost".to_string(),
            allow_non_loopback: false,
            log_level: "info".to_string(),
        };
        assert!(cli.socket_addr(true).is_ok());
    }
}

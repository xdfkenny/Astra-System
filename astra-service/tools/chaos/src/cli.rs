//! CLI argument definitions for astra-chaos.

use std::time::Duration;

use clap::Args;

/// Arguments for the `partition` subcommand.
#[derive(Args, Debug, Clone)]
pub struct PartitionArgs {
    /// Network interface to apply rules to.
    #[arg(short, long, default_value = "eth0")]
    pub interface: String,

    /// Peer IP addresses that form the mesh.
    #[arg(short, long, value_delimiter = ',', required = true)]
    pub peers: Vec<String>,

    /// Duration to hold the partition before automatically restoring connectivity.
    #[arg(short, long, default_value = "30s", value_parser = humantime::parse_duration)]
    pub duration: Duration,

    /// Probability [0.0, 1.0] of partitioning each peer.
    #[arg(short = 'r', long, default_value = "0.3")]
    pub probability: f64,

    /// Random seed for reproducible partitions.
    #[arg(long)]
    pub seed: Option<u64>,

    /// Print the planned commands without executing them.
    #[arg(long)]
    pub dry_run: bool,

    /// Force a specific backend (tc, pf, iptables). Auto-detected when omitted.
    #[arg(short, long)]
    pub backend: Option<String>,

    /// Direction to partition: ingress, egress, or both.
    #[arg(short = 'D', long, default_value = "both")]
    pub direction: String,
}

impl PartitionArgs {
    /// Validates the argument combination.
    pub fn validate(&self) -> Result<(), String> {
        if self.peers.is_empty() {
            return Err("at least one peer is required".to_string());
        }
        if !(0.0..=1.0).contains(&self.probability) {
            return Err("probability must be between 0.0 and 1.0".to_string());
        }
        if self.duration.is_zero() {
            return Err("duration must be greater than zero".to_string());
        }
        match self.direction.as_str() {
            "ingress" | "egress" | "both" => {}
            _ => return Err("direction must be ingress, egress, or both".to_string()),
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn partition_args_validation() {
        let args = PartitionArgs {
            interface: "eth0".to_string(),
            peers: vec!["10.0.0.1".to_string()],
            duration: Duration::from_secs(10),
            probability: 0.5,
            seed: None,
            dry_run: true,
            backend: None,
            direction: "both".to_string(),
        };
        assert!(args.validate().is_ok());
    }

    #[test]
    fn partition_args_rejects_invalid_probability() {
        let args = PartitionArgs {
            interface: "eth0".to_string(),
            peers: vec!["10.0.0.1".to_string()],
            duration: Duration::from_secs(10),
            probability: 1.5,
            seed: None,
            dry_run: false,
            backend: None,
            direction: "both".to_string(),
        };
        assert!(args.validate().is_err());
    }
}

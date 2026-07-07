//! Platform-specific network backends for injecting partitions.

use std::fmt;

pub mod iptables;
pub mod pf;
pub mod tc;

/// A network command to apply or restore connectivity.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NetworkCommand {
    pub program: String,
    pub args: Vec<String>,
    pub description: String,
}

impl NetworkCommand {
    pub fn new(program: &str, args: Vec<&str>, description: &str) -> Self {
        Self {
            program: program.to_string(),
            args: args.iter().map(|s| s.to_string()).collect(),
            description: description.to_string(),
        }
    }
}

impl fmt::Display for NetworkCommand {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{} {}", self.program, self.args.join(" "))
    }
}

/// Backend trait for applying and restoring network partitions.
pub trait Backend: fmt::Debug + Send + Sync {
    /// Returns the human-readable backend name.
    fn name(&self) -> &str;

    /// Generates commands to partition `peers` on `interface`.
    fn partition_commands(
        &self,
        interface: &str,
        peers: &[String],
        direction: &str,
    ) -> Vec<NetworkCommand>;

    /// Generates commands to restore connectivity.
    fn restore_commands(&self,
        interface: &str,
        peers: &[String],
        direction: &str,
    ) -> Vec<NetworkCommand>;
}

/// Picks the best backend for the current platform.
pub fn auto_detect() -> Result<Box<dyn Backend>, String> {
    #[cfg(target_os = "linux")]
    {
        if command_exists("tc") {
            return Ok(Box::new(tc::TcBackend));
        }
        if command_exists("iptables") {
            return Ok(Box::new(iptables::IptablesBackend));
        }
    }
    #[cfg(target_os = "macos")]
    {
        if command_exists("pfctl") {
            return Ok(Box::new(pf::PfBackend));
        }
    }
    Err("no supported network backend found (tc/iptables on Linux, pfctl on macOS)".to_string())
}

/// Creates a backend by name or auto-detects the platform default.
pub fn resolve(name: Option<&str>) -> Result<Box<dyn Backend>, String> {
    match name {
        Some("tc") => Ok(Box::new(tc::TcBackend)),
        Some("iptables") => Ok(Box::new(iptables::IptablesBackend)),
        Some("pf") => Ok(Box::new(pf::PfBackend)),
        Some(other) => Err(format!("unknown backend: {}", other)),
        None => auto_detect(),
    }
}

fn command_exists(cmd: &str) -> bool {
    std::process::Command::new("which")
        .arg(cmd)
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn resolve_known_backends() {
        assert_eq!(resolve(Some("tc")).unwrap().name(), "tc");
        assert_eq!(resolve(Some("iptables")).unwrap().name(), "iptables");
        assert_eq!(resolve(Some("pf")).unwrap().name(), "pf");
    }

    #[test]
    fn resolve_unknown_backend_fails() {
        assert!(resolve(Some("foobar")).is_err());
    }
}

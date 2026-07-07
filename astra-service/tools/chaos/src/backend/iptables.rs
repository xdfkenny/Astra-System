//! Linux `iptables` backend.
//!
//! Inserts DROP rules for selected peers and removes them on restore.

use super::{Backend, NetworkCommand};

#[derive(Debug)]
pub struct IptablesBackend;

impl Backend for IptablesBackend {
    fn name(&self) -> &str {
        "iptables"
    }

    fn partition_commands(
        &self,
        _interface: &str,
        peers: &[String],
        direction: &str,
    ) -> Vec<NetworkCommand> {
        let mut cmds = Vec::new();
        for peer in peers {
            if direction == "egress" || direction == "both" {
                cmds.push(NetworkCommand::new(
                    "iptables",
                    vec!["-A", "OUTPUT", "-d", peer, "-j", "DROP"],
                    &format!("drop egress to {}", peer),
                ));
            }
            if direction == "ingress" || direction == "both" {
                cmds.push(NetworkCommand::new(
                    "iptables",
                    vec!["-A", "INPUT", "-s", peer, "-j", "DROP"],
                    &format!("drop ingress from {}", peer),
                ));
            }
        }
        cmds
    }

    fn restore_commands(
        &self,
        _interface: &str,
        peers: &[String],
        direction: &str,
    ) -> Vec<NetworkCommand> {
        let mut cmds = Vec::new();
        for peer in peers {
            if direction == "egress" || direction == "both" {
                cmds.push(NetworkCommand::new(
                    "iptables",
                    vec!["-D", "OUTPUT", "-d", peer, "-j", "DROP"],
                    &format!("restore egress to {}", peer),
                ));
            }
            if direction == "ingress" || direction == "both" {
                cmds.push(NetworkCommand::new(
                    "iptables",
                    vec!["-D", "INPUT", "-s", peer, "-j", "DROP"],
                    &format!("restore ingress from {}", peer),
                ));
            }
        }
        cmds
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn iptables_partition_both_directions() {
        let backend = IptablesBackend;
        let peers = vec!["10.0.0.1".to_string()];
        let cmds = backend.partition_commands("eth0", &peers, "both");
        assert_eq!(cmds.len(), 2);
        assert!(cmds.iter().any(|c| c.args.contains(&"OUTPUT".to_string())));
        assert!(cmds.iter().any(|c| c.args.contains(&"INPUT".to_string())));
    }

    #[test]
    fn iptables_restore_matches_partition() {
        let backend = IptablesBackend;
        let peers = vec!["10.0.0.1".to_string()];
        let partition = backend.partition_commands("eth0", &peers, "both");
        let restore = backend.restore_commands("eth0", &peers, "both");
        assert_eq!(partition.len(), restore.len());
        assert!(restore.iter().all(|c| c.args.contains(&"-D".to_string())));
    }
}

//! Linux `tc` (traffic control) backend.
//!
//! Uses `tc qdisc` with `netem loss 100%` to drop egress traffic toward
//! selected peers. Restoration removes the egress qdisc.

use super::{Backend, NetworkCommand};

#[derive(Debug)]
pub struct TcBackend;

impl Backend for TcBackend {
    fn name(&self) -> &str {
        "tc"
    }

    fn partition_commands(
        &self,
        interface: &str,
        peers: &[String],
        direction: &str,
    ) -> Vec<NetworkCommand> {
        let mut cmds = Vec::new();
        if direction == "egress" || direction == "both" {
            cmds.push(NetworkCommand::new(
                "tc",
                vec!["qdisc", "add", "dev", interface, "root", "handle", "1:", "prio"],
                &format!("add prio qdisc on {}", interface),
            ));
            for (idx, peer) in peers.iter().enumerate() {
                let class = format!("1:{}", idx + 1);
                cmds.push(NetworkCommand::new(
                    "tc",
                    vec![
                        "filter", "add", "dev", interface, "protocol", "ip", "parent", "1:",
                        "prio", "1", "u32", "match", "ip", "dst", peer, "flowid", &class,
                    ],
                    &format!("filter egress traffic to {}", peer),
                ));
                cmds.push(NetworkCommand::new(
                    "tc",
                    vec![
                        "qdisc", "add", "dev", interface, "parent", &class, "handle",
                        &format!("10{}:", idx), "netem", "loss", "100%",
                    ],
                    &format!("drop 100% of packets to {}", peer),
                ));
            }
        }
        cmds
    }

    fn restore_commands(
        &self,
        interface: &str,
        _peers: &[String],
        _direction: &str,
    ) -> Vec<NetworkCommand> {
        vec![NetworkCommand::new(
            "tc",
            vec!["qdisc", "del", "dev", interface, "root"],
            &format!("remove tc qdisc from {}", interface),
        )]
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tc_partition_commands() {
        let backend = TcBackend;
        let peers = vec!["10.0.0.1".to_string(), "10.0.0.2".to_string()];
        let cmds = backend.partition_commands("eth0", &peers, "egress");
        assert!(!cmds.is_empty());
        assert!(cmds.iter().any(|c| c.program == "tc" && c.args.contains(&"netem".to_string())));
    }

    #[test]
    fn tc_restore_command() {
        let backend = TcBackend;
        let cmds = backend.restore_commands("eth0", &[], "egress");
        assert_eq!(cmds.len(), 1);
        assert!(cmds[0].args.contains(&"del".to_string()));
    }
}

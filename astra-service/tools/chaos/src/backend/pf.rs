//! macOS `pfctl` backend.
//!
//! Creates a temporary anchor under `astra-chaos` that blocks traffic to/from
//! selected peers. Restoration flushes and removes the anchor.

use super::{Backend, NetworkCommand};

const ANCHOR: &str = "astra-chaos";

#[derive(Debug)]
pub struct PfBackend;

impl Backend for PfBackend {
    fn name(&self) -> &str {
        "pf"
    }

    fn partition_commands(
        &self,
        _interface: &str,
        peers: &[String],
        direction: &str,
    ) -> Vec<NetworkCommand> {
        let mut cmds = Vec::new();
        cmds.push(NetworkCommand::new(
            "pfctl",
            vec!["-a", ANCHOR, "-F", "rules"],
            "flush existing chaos anchor rules",
        ));

        let mut rules = String::new();
        for peer in peers {
            if direction == "egress" || direction == "both" {
                rules.push_str(&format!("block drop out to {}\n", peer));
            }
            if direction == "ingress" || direction == "both" {
                rules.push_str(&format!("block drop in from {}\n", peer));
            }
        }

        if !rules.is_empty() {
            let shell_rule = rules.trim_end().replace('\n', "\\n");
            cmds.push(NetworkCommand::new(
                "sh",
                vec![
                    "-c",
                    &format!(
                        "printf '{}' | pfctl -a {} -f -",
                        shell_rule, ANCHOR
                    ),
                ],
                &format!("load block rules into anchor {}", ANCHOR),
            ));
        }
        cmds
    }

    fn restore_commands(
        &self,
        _interface: &str,
        _peers: &[String],
        _direction: &str,
    ) -> Vec<NetworkCommand> {
        vec![NetworkCommand::new(
            "pfctl",
            vec!["-a", ANCHOR, "-F", "rules"],
            "flush chaos anchor rules",
        )]
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pf_partition_uses_anchor() {
        let backend = PfBackend;
        let peers = vec!["10.0.0.1".to_string()];
        let cmds = backend.partition_commands("en0", &peers, "both");
        assert!(cmds.iter().all(|c| c.program == "pfctl" || c.program == "sh"));
        assert!(cmds.iter().any(|c| c.args.contains(&ANCHOR.to_string())));
    }

    #[test]
    fn pf_restore_flushes_anchor() {
        let backend = PfBackend;
        let cmds = backend.restore_commands("en0", &[], "both");
        assert_eq!(cmds.len(), 1);
        assert!(cmds[0].args.contains(&"-F".to_string()));
    }
}

//! Partition plan generation and execution.

use std::process::{Command, Stdio};
use std::thread;

use rand::rngs::StdRng;
use rand::{Rng, SeedableRng};
use tracing::{info, warn};

use crate::backend::{resolve, NetworkCommand};
use crate::cli::PartitionArgs;

/// Executes a partition according to the supplied arguments.
pub fn run(args: PartitionArgs) -> Result<(), Box<dyn std::error::Error>> {
    args.validate().map_err(|e| format!("validation: {}", e))?;

    let backend = resolve(args.backend.as_deref())?;
    let selected = select_peers(&args.peers, args.probability, args.seed);

    if selected.is_empty() {
        info!("no peers selected for partition");
        return Ok(());
    }

    info!(
        backend = backend.name(),
        interface = %args.interface,
        duration = ?args.duration,
        selected = ?selected,
        "planning partition"
    );

    let partition_cmds = backend.partition_commands(&args.interface, &selected, &args.direction);
    let restore_cmds = backend.restore_commands(&args.interface, &selected, &args.direction);

    if args.dry_run {
        info!("dry-run mode: printing planned commands");
        println!("# partition commands");
        for cmd in &partition_cmds {
            println!("# {}", cmd.description);
            println!("{}", cmd);
        }
        println!("# sleep {:?}", args.duration);
        println!("# restore commands");
        for cmd in &restore_cmds {
            println!("# {}", cmd.description);
            println!("{}", cmd);
        }
        return Ok(());
    }

    execute_commands(&partition_cmds, "partition")?;
    info!(duration = ?args.duration, "holding partition");
    thread::sleep(args.duration);
    execute_commands(&restore_cmds, "restore")?;

    Ok(())
}

/// Randomly selects peers to partition.
fn select_peers(peers: &[String], probability: f64, seed: Option<u64>) -> Vec<String> {
    let mut rng: StdRng = match seed {
        Some(s) => SeedableRng::seed_from_u64(s),
        None => SeedableRng::from_entropy(),
    };
    peers
        .iter()
        .filter(|_| rng.gen::<f64>() < probability)
        .cloned()
        .collect()
}

/// Executes a list of network commands, failing fast on the first error.
fn execute_commands(cmds: &[NetworkCommand], phase: &str) -> Result<(), Box<dyn std::error::Error>> {
    for cmd in cmds {
        info!(phase = phase, command = %cmd, "executing");
        let child = Command::new(&cmd.program)
            .args(&cmd.args)
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .map_err(|e| format!("failed to spawn {}: {}", cmd.program, e))?;

        let output = child.wait_with_output()?;
        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            warn!(
                phase = phase,
                command = %cmd,
                stderr = %stderr,
                "command failed"
            );
            return Err(format!(
                "{} command failed: {} ({})",
                phase, cmd, stderr
            )
            .into());
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn select_peers_respects_seed() {
        let peers: Vec<String> = (0..100).map(|i| format!("10.0.0.{}", i)).collect();
        let selected1 = select_peers(&peers, 0.3, Some(42));
        let selected2 = select_peers(&peers, 0.3, Some(42));
        assert_eq!(selected1, selected2);
        assert!(!selected1.is_empty());
    }

    #[test]
    fn select_peers_zero_probability() {
        let peers = vec!["10.0.0.1".to_string()];
        let selected = select_peers(&peers, 0.0, Some(1));
        assert!(selected.is_empty());
    }

    #[test]
    fn select_peers_full_probability() {
        let peers = vec!["10.0.0.1".to_string(), "10.0.0.2".to_string()];
        let selected = select_peers(&peers, 1.0, Some(1));
        assert_eq!(selected.len(), 2);
    }
}

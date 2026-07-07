# astra-chaos

Chaos engineering CLI for Astra kiosk mesh networks. Injects network partitions
during integration tests to validate P2P sync resilience.

## Usage

```bash
# Randomly partition 30% of peers for 30 seconds using the auto-detected backend.
astra-chaos partition --peers 10.0.0.1,10.0.0.2,10.0.0.3

# Dry-run to see the commands that would be executed.
astra-chaos partition --peers 10.0.0.1,10.0.0.2 --duration 10s --probability 0.5 --dry-run --backend tc

# Reproducible partition with a fixed seed.
astra-chaos partition --peers 10.0.0.1,10.0.0.2 --seed 42 --probability 1.0
```

## Backends

- `tc` (Linux): uses `tc qdisc` with `netem loss 100%`.
- `iptables` (Linux): adds DROP rules in `INPUT`/`OUTPUT` chains.
- `pf` (macOS): creates an `astra-chaos` anchor with `pfctl`.

The backend is auto-detected when `--backend` is omitted.

## Options

| Option | Description |
|--------|-------------|
| `-i, --interface` | Network interface (default: `eth0`). |
| `-p, --peers` | Comma-separated list of peer IPs. |
| `-d, --duration` | Partition duration (default: `30s`). |
| `-r, --probability` | Probability of partitioning each peer (default: `0.3`). |
| `-D, --direction` | `ingress`, `egress`, or `both` (default: `both`). |
| `--seed` | Random seed for reproducibility. |
| `--dry-run` | Print commands instead of executing. |
| `-b, --backend` | Force `tc`, `iptables`, or `pf`. |

## Build

```bash
cargo build --release
```

## Test

```bash
cargo test
```

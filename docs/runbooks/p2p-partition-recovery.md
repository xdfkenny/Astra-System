# P2P Partition Recovery Runbook

> **See also:**
> - [P2P Mesh Sync](../networking/p2p-mesh.md) — libp2p, QUIC, Raft, CRDT details
> - [Offline-First Strategy](../architecture/offline-first.md) — CRDT merge and reconciliation
> - [Observability Stack](../infrastructure/monitoring.md) — mesh health metrics

## Purpose

Restore convergence and consensus when the kiosk P2P mesh splits into partitions or loses Raft leadership.

## Symptoms

- Alert: `astra_mesh_peer_count` dropped for one or more kiosks.
- Alert: `astra_sync_lag_seconds` increasing.
- Multiple kiosks report different Raft leaders.
- Store staff report inventory counts differ across kiosks.

## Concepts

- **Mesh split:** Kiosks can no longer reach some peers, often due to LAN/VLAN failure.
- **Partition:** Two or more groups of kiosks continue operating independently.
- **CRDT convergence:** When partitions heal, CRDT merge rules deterministically reconcile state.
- **Raft leader:** Only one leader should exist per 3+ kiosk cluster.

## Diagnostic Steps

### 1. View Mesh Topology

```bash
# On any kiosk
astra-syncd mesh status

# Prometheus query for peer count
curl -s 'http://prometheus:9090/api/v1/query?query=astra_mesh_peer_count'
```

### 2. Check Network Layer

```bash
# Verify mDNS discovery
avahi-browse -a

# Verify QUIC reachability between two kiosks
astra-syncd ping --peer <peer-id>

# Check switch/VLAN status with store IT
```

### 3. Inspect Raft State

```bash
# On each kiosk
astra-syncd raft status
```

Expected output for a healthy 3-kiosk cluster:

```text
leader: kiosk-1
term: 42
peers: [kiosk-1, kiosk-2, kiosk-3]
commit_index: 12345
```

### 4. Review Sync Logs

```bash
journalctl -u astra-syncd -n 500 --no-pager | grep -iE "partition|merge|raft|peer"
```

## Resolution Procedures

### Single Kiosk Disconnected

1. Restart the sync daemon:
   ```bash
   sudo systemctl restart astra-syncd
   ```
2. Check logs for mDNS or TLS errors.
3. Verify network cable / Wi-Fi on the kiosk.

### Network Partition (Multiple Groups)

1. Identify the root cause (switch failure, VLAN misconfig, IP conflict).
2. Coordinate with store IT to restore Layer 2 connectivity.
3. Do **not** reboot kiosks unless necessary; CRDTs will converge automatically once the network heals.
4. Monitor `astra_sync_lag_seconds` until it returns to baseline.

### Stuck Raft Election

If kiosks repeatedly fail to elect a leader:

1. Ensure an odd number of voting peers (3, 5, ...).
2. If a kiosk is permanently removed, force Raft reconfiguration:
   ```bash
   astra-syncd raft remove-peer --id <dead-kiosk-id>
   ```
3. Trigger a new election:
   ```bash
   astra-syncd raft step-down  # on current leader if any
   ```

### Inventory Divergence After Merge

1. Open the manager dashboard `Circuit Breaker` view.
2. Review flagged conflicts (e.g., concurrent price overrides).
3. For auto-resolved PN-Counter differences, no action needed.
4. For conflicts requiring human judgment, follow the store manager approval workflow.

## Verification

After recovery:

1. All kiosks report the same leader:
   ```bash
   for k in kiosk-1 kiosk-2 kiosk-3; do ssh $k astra-syncd raft status; done
   ```
2. `astra_mesh_peer_count` equals expected peer count for every kiosk.
3. `astra_sync_lag_seconds` is < 1 second.
4. Inventory counts match across kiosks for sampled SKUs.
5. No unresolved conflicts in the manager dashboard.

## Prevention

- Use redundant switches and links in store LAN.
- Deploy 3+ kiosks per store to enable Raft consensus.
- Run chaos engineering partition tests weekly in CI.
- Monitor switch/port metrics where available.

## Related Documentation

- [Incident Response Runbook](./incident-response.md)
- [Offline Mode Operations](./offline-mode-operations.md)

# Offline Mode Operations Runbook

> **See also:**
> - [Offline-First Strategy](../architecture/offline-first.md) — 3-tier resilience model, CRDT details
> - [P2P Mesh Sync](../networking/p2p-mesh.md) — mesh sync during offline periods
> - [Payment Orchestrator](../backend/payment-orchestrator.md) — offline token system

## Purpose

Operate Astra-System kiosks during extended cloud connectivity loss. Offline mode is a designed, first-class state — not a failure condition — but requires monitoring to ensure the 48-hour resilience window is not exceeded.

## When Offline Mode Activates

A kiosk enters offline mode when it cannot reach the cloud API gateway for 30 seconds. Detection uses a health check to `https://api.astra.internal/health` every 10 seconds.

## Expected Behavior in Offline Mode

| Capability        | Behavior in Offline Mode                               |
| ----------------- | ------------------------------------------------------ |
| Customer checkout | Fully functional; Verifone terminal processes payments |
| Inventory sync    | P2P mesh only; cloud inventory not updated             |
| New orders        | Stored locally and replicated to peers                 |
| Employee auth     | Local biometric hash verification                      |
| Software updates  | Deferred                                               |
| Cloud reports     | Not available locally; queued for upload               |

## Monitoring

### Key Metrics

| Metric                           | Warning | Critical | Meaning                            |
| -------------------------------- | ------- | -------- | ---------------------------------- |
| `astra_offline_queue_depth`      | > 50    | > 100    | Unsettled offline payment tokens   |
| `astra_offline_duration_minutes` | > 12 h  | > 36 h   | Time since cloud connectivity lost |
| `astra_mesh_peer_count`          | < 2     | 0        | Number of reachable peer kiosks    |
| `astra_sync_lag_seconds`         | > 5 s   | > 30 s   | Delay replicating state to peers   |

### Check Offline Status on a Kiosk

```bash
astra-syncd status
```

Expected output:

```text
cloud_reachable: false
offline_since: 2026-07-05T02:00:00Z
mode: autonomous
raft_leader: kiosk-1
queue_depth: 12
oldest_token_age: 45m
```

## Operational Procedures

### Extending the Offline Window

If cloud connectivity will not be restored before 48 hours:

1. Notify store management and Astra operations.
2. Optionally enable **extended offline mode**:
   - Increases payment token TTL to 72 hours.
   - Requires manager PIN code for activation.
3. Document all offline transactions for manual reconciliation.

### Manual Reconciliation Mode

If offline duration exceeds 48 hours, the kiosk enters reconciliation mode on reconnection:

1. Cloud settlement pauses for the store.
2. Store manager logs into the admin panel.
3. Review the list of offline transactions.
4. Approve or reject each transaction:
   - Approved: settlement proceeds.
   - Rejected: transaction is voided; customer may need refund.
5. Once all transactions are resolved, normal settlement resumes.

### Draining the Offline Queue

When connectivity returns:

1. Verify cloud health:
   ```bash
   curl https://api.astra.internal/health
   ```
2. The Raft leader automatically batches queued tokens.
3. Monitor queue depth:
   ```bash
   watch -n 5 'curl -s http://localhost:9090/metrics | grep astra_offline_queue_depth'
   ```
4. If the leader fails to upload, manually promote a new leader:
   ```bash
   astra-syncd raft step-down
   ```

### Inventory Adjustments While Offline

1. Use the employee handheld device or kiosk admin mode.
2. Adjustments are broadcast via P2P mesh immediately.
3. When online, the leader uploads inventory deltas to `inventory-svc`.

### Preventing Duplicate Tokens

Each offline token has a UUID v7 ID and is signed with the kiosk's daily key. If a token is accidentally uploaded twice, the payment orchestrator rejects it by idempotency key.

## Contingency: Complete Mesh Failure

If all kiosks lose P2P connectivity AND cloud connectivity:

1. Each kiosk continues to accept transactions independently.
2. Customers may see inconsistent inventory across kiosks.
3. Store staff should:
   - Limit high-value items.
   - Post manual "cash only" signage if terminal connectivity is also lost.
4. Once any connectivity returns, reconcile all independent queues.

## Returning to Online Mode

1. Confirm `cloud_reachable: true` on the leader.
2. Confirm queue depth reaches zero.
3. Verify cloud dashboards show the store as online.
4. Send an all-clear notification to store staff.

## Prevention

- Deploy redundant WAN links (primary + 4G/5G failover).
- Maintain 3+ kiosks per store for mesh resilience.
- Keep terminal connectivity independent from kiosk LAN where possible.
- Test offline scenarios in chaos engineering runs.

## Related Documentation

- [Incident Response Runbook](./incident-response.md)
- [Payment Failure Runbook](./payment-failure-runbook.md)
- [P2P Partition Recovery](./p2p-partition-recovery.md)

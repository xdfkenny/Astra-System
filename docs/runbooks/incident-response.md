# Incident Response Runbook

> **See also:**
> - [Observability Stack](../infrastructure/monitoring.md) — metrics, alerts, dashboards
> - [Deployment Guide](../operations/deployment.md) — production topology
> - [Security Model](../architecture/security-model.md) — security incident context

## Purpose

This runbook defines the step-by-step response process for production incidents affecting Astra-System kiosks, cloud services, or the P2P mesh.

## Severity Definitions

| Severity | Criteria                                                 | Response Time     | Lead Role          |
| -------- | -------------------------------------------------------- | ----------------- | ------------------ |
| SEV-1    | Payment system down, all kiosks offline, data breach     | 5 minutes         | Incident Commander |
| SEV-2    | Partial kiosk outage, mesh partition, major feature down | 15 minutes        | On-call engineer   |
| SEV-3    | Degraded performance, isolated kiosk faults              | 1 hour            | Support engineer   |
| SEV-4    | Minor bugs, non-impacting alerts                         | Next business day | Engineering team   |

## Initial Response (First 5 Minutes)

1. **Acknowledge the alert** in PagerDuty / Slack `#incidents`.
2. **Declare an incident** with `/incident start <summary>`.
3. **Identify the incident commander (IC)** — typically the on-call engineer; escalate to SEV-1 lead if needed.
4. **Open an incident bridge** (Zoom/Meet) for SEV-1/SEV-2.
5. **Gather context:**
   - Grafana dashboard: `Astra / Service Overview`
   - Loki logs filtered by `trace_id` or `kiosk_id`
   - Recent deployments from GitHub Actions
   - Cloud provider status pages

## Diagnostic Checklist

### Cloud Services

```bash
# Check pod health
kubectl get pods -n astra

# Check service logs
kubectl logs -n astra -l app=api-gateway --tail=200

# Check NATS stream health
nats stream info astra-events

# Check database connectivity
psql $DATABASE_URL -c "SELECT count(*) FROM outbox WHERE processed_at IS NULL;"
```

### Kiosk Mesh

```bash
# Check sync daemon logs on a kiosk
journalctl -u astra-syncd -n 200 --no-pager

# Check peer count metric
curl -s http://localhost:9090/metrics | grep astra_mesh_peer_count

# Check Raft status
astra-syncd raft status
```

### Payment Pipeline

```bash
# Check offline queue depth
curl -s http://localhost:9090/metrics | grep astra_offline_queue_depth

# Check payment success rate
curl -s http://localhost:9090/metrics | grep astra_payment_success_rate
```

## Common Response Patterns

### Rollback a Bad Deployment

1. Identify the last known-good image tag.
2. Roll back the affected deployment:
   ```bash
   kubectl rollout undo deployment/api-gateway -n astra
   ```
3. Verify health checks pass.
4. Notify the incident channel.

### Scale Up to Handle Load

1. Check HPA status:
   ```bash
   kubectl get hpa -n astra
   ```
2. Manually scale if HPA is capped:
   ```bash
   kubectl scale deployment/order-svc --replicas=10 -n astra
   ```

### Isolate a Compromised Kiosk

1. Revoke the kiosk certificate in Vault:
   ```bash
   vault write astra-pki/revoke serial_number=<serial>
   ```
2. Block the kiosk at the API gateway by `kiosk_id`.
3. Coordinate with store staff to physically power off the device.
4. Preserve logs and forensic images before remediation.

## Communication

| Time            | Action                                                          |
| --------------- | --------------------------------------------------------------- |
| 0–15 min        | Internal Slack `#incidents`, status page marked "Investigating" |
| Every 30 min    | Status update in incident thread and on status page             |
| Resolution      | Mark status page resolved; post summary in `#incidents`         |
| Within 24 hours | Schedule post-mortem; create follow-up tickets                  |

## Post-Mortem Template

1. **Summary:** What happened, impact, and duration.
2. **Timeline:** Detection, response, mitigation, resolution.
3. **Root Cause:** 5 Whys or fault-tree analysis.
4. **Action Items:** Owner + due date for each remediation.
5. **Lessons Learned:** Process or tooling gaps.

## Escalation Path

1. On-call engineer
2. Engineering manager (if SEV-1 or unresolved after 30 min)
3. VP Engineering / CTO (if customer-facing downtime > 1 hour)
4. Security team (if breach or PCI data suspected)
5. Legal / compliance (if regulatory notification may be required)

## Related Runbooks

- [Payment Failure Runbook](./payment-failure-runbook.md)
- [P2P Partition Recovery](./p2p-partition-recovery.md)
- [Offline Mode Operations](./offline-mode-operations.md)

# Payment Failure Runbook

> **See also:**
> - [Payment Orchestrator](../backend/payment-orchestrator.md) — payment flow, offline tokens
> - [Security Model](../architecture/security-model.md) — PCI-DSS, Verifone isolation
> - [Observability Stack](../infrastructure/monitoring.md) — payment metrics and alerts

## Purpose

Resolve payment failures across Verifone terminals, the offline token queue, and cloud settlement. This runbook covers both online and offline payment paths.

## Symptoms

- Customer reports card declined or terminal frozen.
- Alert: `astra_payment_success_rate` < 95%.
- Offline queue depth rising.
- Settlement batch rejected by acquirer.

## Severity

| Symptom                                                | Severity |
| ------------------------------------------------------ | -------- |
| Single terminal error                                  | SEV-3    |
| All terminals in a store failing                       | SEV-2    |
| Payment success rate < 90% or settlement batch failing | SEV-1    |

## Diagnostic Steps

### 1. Check Terminal Health

```bash
# On the affected kiosk
astra-syncd verifone status

# Inspect sync daemon logs
journalctl -u astra-syncd -n 200 --no-pager | grep -i verifone
```

Look for:

- `VxInitTerminal` failures (USB/Ethernet disconnect)
- `VxProcessPayment` timeout
- Error codes from the Verifone SDK

### 2. Verify Network Path

```bash
# Ping terminal
ping <terminal-ip>

# Check TLS to cloud payment orchestrator
curl -v https://payment-svc.astra.internal/health
```

### 3. Inspect Offline Queue (Offline Mode)

```bash
# Count pending tokens
sqlite3 /var/lib/astra/kiosk.db "SELECT count(*) FROM offline_payment_tokens WHERE settled_at IS NULL;"

# View oldest token
sqlite3 /var/lib/astra/kiosk.db "SELECT token_id, created_at, amount, currency FROM offline_payment_tokens WHERE settled_at IS NULL ORDER BY created_at LIMIT 5;"
```

### 4. Check Cloud Settlement Logs

```bash
kubectl logs -n astra -l app=payment-svc --tail=500 | grep -i "settlement\|rejected\|invalid signature"
```

## Resolution Procedures

### Terminal Frozen / Unresponsive

1. Cancel the transaction on the terminal if UI allows.
2. Call `VxCloseTerminal()` from the kiosk admin panel.
3. Power-cycle the terminal via store staff.
4. Re-run `VxInitTerminal()` and verify status.

### Single Terminal Repeated Failures

1. Swap the terminal with a spare unit.
2. Update the kiosk config with the new `terminal_id`.
3. Open a vendor ticket for the failed terminal.

### All Terminals in Store Failing

1. Check store internet/VPN connectivity.
2. Verify DNS resolution for `payment-svc.astra.internal`.
3. Check if a recent Verifone SDK or certificate change was deployed.
4. Roll back the `astra-syncd` or `payment-svc` deployment if correlated.

### Offline Token Settlement Failures

1. Identify rejected tokens from payment-svc logs.
2. Validate signatures:
   ```bash
   astra-admin verify-token --token-id <id>
   ```
3. If signature invalid and kiosk is online, rotate the kiosk signing key.
4. If token expired (> 48 h), flag for manual manager review.
5. For acquirer rejects (insufficient funds, card blocked), follow standard refund/void process.

### Refund Processing

```bash
# Initiate refund via Verifone FFI
astra-admin refund --transaction-id <txid> --amount <cents> --currency USD
```

Verify refund appears in:

- Local SQLite transaction log
- P2P mesh broadcast
- Cloud settlement batch (when online)

## Verification

After resolving:

1. Run a test transaction on the affected terminal.
2. Confirm `astra_payment_success_rate` returns to > 99%.
3. Confirm offline queue depth is decreasing or zero.
4. Update the incident thread.

## Prevention

- Monitor terminal firmware age; schedule updates quarterly.
- Keep acquirer TLS certificates rotated before expiry.
- Run nightly settlement reconciliation job.
- Include payment failure scenarios in chaos engineering tests.

## Related Documentation

- [Incident Response Runbook](./incident-response.md)
- [Offline Mode Operations](./offline-mode-operations.md)

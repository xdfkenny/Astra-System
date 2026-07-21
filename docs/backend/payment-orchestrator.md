# Payment Orchestrator

## Overview

The Payment Orchestrator (`services/payment-orchestrator/`) manages the complete payment lifecycle: authorization, capture, settlement, refund, and offline token management. It integrates with Verifone payment terminals via the Rust sidecar bridge.

## Architecture

```
Kiosk App → Payment MFE → Payment Sidecar (Rust, localhost:8963)
  → Verifone Terminal → Token → Payment Orchestrator (Go, :8086)
  → PostgreSQL (payments, offline_tokens, refunds)
  → NATS JetStream (payment events)
```

## Payment Flow

### Online Payment

```
1. InitiatePayment (gRPC): amount, method, metadata
2. Forward to Verifone terminal via payment-sidecar
3. Verifone processes, returns token
4. Store payment record in PostgreSQL
5. Publish PaymentInitiated event to NATS
6. CapturePayment: finalize the transaction
7. SettlePayment: submit for settlement
8. Publish PaymentCompleted event
```

### Offline Payment

```
1. System detects offline (no cloud connectivity)
2. buildOfflineToken(): creates signed, time-limited token
3. Store token in offline_tokens table
4. Customer receives offline confirmation
5. Token queued for settlement
6. On reconnection: batch settle via /v1/offline-tokens/settle
```

### Refund Flow

```
1. RefundPayment (gRPC): payment_id, amount, reason
2. Validate refund eligibility (within window, not fully refunded)
3. Process refund through Verifone
4. Store refund record
5. Publish refund event
```

## gRPC Endpoints

| RPC | Description | Idempotent |
|-----|-------------|------------|
| `InitiatePayment` | Create payment intent | Yes (Idempotency-Key) |
| `CapturePayment` | Capture authorized payment | Yes |
| `RefundPayment` | Process refund | Yes |
| `SettleOfflineToken` | Settle single offline token | Yes |
| `GetPaymentStatus` | Check payment status | Read-only |

## Offline Token System

### Token Structure

```json
{
  "token_id": "uuid",
  "amount_cents": 1500,
  "currency": "USD",
  "store_id": "uuid",
  "kiosk_id": "uuid",
  "created_at": 1712345678000,
  "expires_at": 1712518478000,
  "signature": "base64_ed25519_sig"
}
```

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PAYMENT_OFFLINE_TOKEN_SEED` | - | Seed for token signing key |
| `PAYMENT_MAX_OFFLINE_AMOUNT_CENTS` | 5000 | Max offline value ($50) |
| `PAYMENT_OFFLINE_TTL_SECONDS` | 172800 | Token validity (48h) |

## Database Tables

- `payments` - Payment records with Verifone tokens
- `refunds` - Refund records
- `offline_tokens` - Tokens queued for settlement

## Events Published

- `PaymentInitiated` - Payment started
- `PaymentAuthorized` - Authorization successful
- `PaymentCaptured` - Payment captured
- `PaymentSettled` - Settlement confirmed
- `PaymentFailed` - Payment failed
- `RefundProcessed` - Refund completed

# Sync Service

## Overview

The Sync Service (`services/sync-service/`) serves as the cloud-side gateway for kiosk mesh synchronization. It handles batch upload/download of CRDT deltas and kiosk heartbeat monitoring.

## Responsibilities

- CRDT batch upload from mesh leaders
- Batch download for kiosk state convergence
- Kiosk heartbeat collection and health tracking
- Mesh leader authentication (Bearer token)
- NATS event relay for sync notifications

## gRPC Endpoints

| RPC | Type | Description |
|-----|------|-------------|
| `UploadBatch` | Client stream | Receive CRDT deltas from mesh leader |
| `DownloadBatch` | Server stream | Send cloud changes to kiosk |
| `Heartbeat` | Unary | Receive kiosk health/liveness |
| `StreamHeartbeats` | Bidirectional stream | Real-time heartbeat streaming |

## Auth

Mesh leaders authenticate via Bearer token validated by the gRPC auth interceptor.

**File:** `internal/auth/interceptor.go`

## Sync Flow

```
Kiosk Mesh (leader) → UploadBatch(gRPC) → Sync Service
  → Store in PostgreSQL → Publish NATS event
  → Other subscribers notified

Kiosk → DownloadBatch(gRPC) → Sync Service
  → Read pending changes → Stream to kiosk
  → Kiosk applies CRDT merge
```

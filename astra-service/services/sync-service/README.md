# astra-sync-service

Cloud-side sync gateway for the Astra kiosk mesh. This service exposes the
`astra.sync.v1.SyncService` gRPC contract plus a JSON REST fallback.

## Responsibilities

- **Authenticate kiosk leaders**: every request is authenticated against the
  `kiosks` table using a bearer token that must match the kiosk's
  `signing_key_hash`; only kiosks where `is_leader = true` are accepted.
- **UploadBatch**: accept a batch of CRDT deltas from the leader kiosk,
  persist raw sync events in PostgreSQL, and publish a `astra.sync.batch_ingested`
  notification over NATS JetStream.
- **DownloadBatch**: compute the set of deltas originating from *other* kiosks
  in the same store since the caller's last sync checkpoint.
- **Heartbeat**: record kiosk status and vector clock, deduplicating rapid
  successive calls, and return the current `is_leader` state.

## Project layout

```
cmd/sync-service/main.go          # entrypoint
internal/config/config.go         # env-based configuration
internal/server/server.go         # gRPC + REST servers
internal/service/service.go       # SyncService implementation
internal/auth/                    # gRPC and REST auth interceptors
internal/repository/repository.go # PostgreSQL persistence
internal/eventbus/publisher.go    # NATS JetStream publisher
internal/model/model.go           # shared domain types
```

## API

### gRPC

Listen port: `SYNC_SERVICE_GRPC_PORT` (default `50051`).

- `astra.sync.v1.SyncService/UploadBatch`
- `astra.sync.v1.SyncService/DownloadBatch`
- `astra.sync.v1.SyncService/Heartbeat`

### REST

Listen port: `SYNC_SERVICE_HTTP_PORT` (default `8087`).

All endpoints expect `Authorization: Bearer <signing_key_hash>` and a JSON body
that matches the protobuf message shapes.

- `POST /v1/sync/upload`
- `POST /v1/sync/download`
- `POST /v1/sync/heartbeat`
- `GET /health`
- `GET /live`
- `GET /ready`

## Database schema expectations

The service assumes the following tables exist (see `database/schemas/go_structs.go`):

- `kiosks` — kiosk identity, leader flag, and `signing_key_hash`.
- `sync_events` — raw sync events, idempotent on `sync_event_id`.
- `sync_heartbeats` — per-kiosk heartbeat log.

```sql
CREATE TABLE IF NOT EXISTS sync_heartbeats (
    kiosk_id       UUID NOT NULL,
    store_id       UUID NOT NULL,
    status         TEXT NOT NULL,
    vector_clock   JSONB NOT NULL DEFAULT '{}',
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (kiosk_id, date_trunc('second', acknowledged_at))
);
```

## Running locally

```bash
cp .env.example .env
# edit .env with your Postgres and NATS URLs
go run ./cmd/sync-service
```

## Testing

```bash
go test -race ./...
```

Tests use in-memory fake repositories so they do not require Postgres or NATS.

## Docker

```bash
docker build -f Dockerfile -t astra-sync-service ..
```

The Dockerfile is designed to be built from the repository root so it can copy
`go.work`, `proto/`, `packages/go-common/`, and the service source together.

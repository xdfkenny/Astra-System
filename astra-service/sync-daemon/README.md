# Astra Sync Daemon (`astra-syncd`)

Production-grade peer-to-peer mesh synchronization daemon for the Astra kiosk ecosystem. Each kiosk runs `astra-syncd` to maintain strongly consistent inventory, cart, and transaction state across the local network using CRDTs, with Raft-based leader election for cloud gateway responsibilities.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        astra-syncd                               │
├─────────────────────────────────────────────────────────────────┤
│  gRPC Server (tonic)  │  Sync Engine  │  Cloud Sync (NATS)     │
│  Local IPC interface   │  CRDT ops     │  Leader-only upload    │
├─────────────────────────────────────────────────────────────────┤
│  P2P Mesh (libp2p)                                              │
│  QUIC + Noise │ mDNS discovery │ GossipSub │ Request/Response  │
├─────────────────────────────────────────────────────────────────┤
│  Raft Consensus (simplified)                                    │
│  Leader election │ Heartbeat │ Failover detection             │
├─────────────────────────────────────────────────────────────────┤
│  Storage Layer                                                  │
│  SQLCipher (AES-256) │ WAL mode │ Offline payment queue       │
└─────────────────────────────────────────────────────────────────┘
```

## Features

- **P2P Mesh Networking**: libp2p with QUIC transport, Noise XX handshake, and mDNS LAN discovery. No central server required for local sync.
- **CRDT Synchronization**: LWW-Registers, PN-Counters, and OR-Sets with Lamport timestamps for causal ordering and deterministic merge.
- **Priority-Based Sync**:
  - **Immediate**: Inventory counts (real-time, ~100ms loop)
  - **Batched**: Transactions and cart state (5-second batching)
  - **Delayed**: Analytics and telemetry (60-second interval)
- **Raft Consensus**: Simplified Raft for leader election. The leader is responsible for NATS JetStream cloud uploads.
- **Offline Resilience**: When internet is unavailable, all data syncs locally. Payments are queued with HMAC-SHA256 signed offline tokens.
- **Encryption at Rest & in Transit**: XChaCha20-Poly1305 for P2P messages, SQLCipher AES-256 for local database, HMAC-SHA256 for payment integrity.
- **Structured Logging**: JSON-formatted `tracing` logs with configurable verbosity.
- **Graceful Shutdown**: SIGTERM/SIGINT handling with 30-second timeout for subsystem cleanup.

## Building

### Prerequisites

- Rust 1.75+ (stable)
- SQLite with SQLCipher support (or use `bundled` feature via `rusqlite`)
- Protocol Buffers compiler (`protoc`) for gRPC code generation
- Linux/macOS for Unix signal handling

### Compile

```bash
cd astra-service/sync-daemon/
cargo build --release
```

The release binary is optimized with LTO and `codegen-units = 1` for maximum performance.

### Generate Keys

```bash
# Generate the P2P sync encryption key (32 bytes, XChaCha20-Poly1305)
cargo run -- gen-key --output /etc/astra-syncd/sync.key --key-type sync

# Generate the offline payment HMAC key (32 bytes, HMAC-SHA256)
cargo run -- gen-key --output /etc/astra-syncd/hmac.key --key-type hmac
```

Ensure key files are `chmod 600` — the daemon will refuse to load them if permissions are too permissive.

## Configuration

Create `/etc/astra-syncd/config.toml`:

```toml
[daemon]
kiosk_id = "kiosk-42"
data_dir = "/var/lib/astra-syncd"
log_level = "info"
metrics_addr = "127.0.0.1:9090"

[p2p]
listen_addr = "0.0.0.0:0"            # 0 = ephemeral port
bootstrap_peers = []
network_name = "astra-kiosk-mesh"
max_connections = 50
conn_idle_timeout_secs = 300

[storage]
db_path = "/var/lib/astra-syncd/sync.db"
encryption_key_path = "/etc/astra-syncd/db.key"
wal_checkpoint_pages = 1000
max_db_size_mib = 512

[cloud]
nats_url = "tls://connect.ngs.global"
jetstream_bucket = "ASTRA_SYNC"
credentials_path = "/etc/astra-syncd/nats.creds"
flush_interval_seconds = 30
max_msg_size_bytes = 1048576
connect_timeout_secs = 10

[grpc]
listen_addr = "127.0.0.1:50051"
max_concurrent_streams = 256

[raft]
heartbeat_interval_ms = 500
election_timeout_min_ms = 1500
election_timeout_max_ms = 3000
max_entries_per_append = 128

[crypto]
sync_key_path = "/etc/astra-syncd/sync.key"
offline_hmac_key_path = "/etc/astra-syncd/hmac.key"
```

### Database Encryption Key

The SQLCipher database key is a hex string (64 characters = 32 bytes). Generate it securely:

```bash
openssl rand -hex 32 > /etc/astra-syncd/db.key
chmod 600 /etc/astra-syncd/db.key
```

## Running

```bash
# Validate configuration
astra-syncd --config /etc/astra-syncd/config.toml validate

# Run database migrations
astra-syncd --config /etc/astra-syncd/config.toml migrate

# Start the daemon
astra-syncd --config /etc/astra-syncd/config.toml
```

### systemd Service

```ini
[Unit]
Description=Astra P2P Sync Daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
ExecStart=/usr/local/bin/astra-syncd --config /etc/astra-syncd/config.toml
Restart=on-failure
RestartSec=5
User=astra
Group=astra
# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/astra-syncd
ReadOnlyPaths=/etc/astra-syncd

[Install]
WantedBy=multi-user.target
```

## gRPC API

The daemon exposes a local gRPC service on the configured `grpc.listen_addr` for integration with other Astra services.

```protobuf
service AstraSync {
  rpc HealthCheck(Empty) returns (HealthResponse);
  rpc SyncNow(SyncRequest) returns (SyncResponse);
  rpc GetSyncStatus(Empty) returns (SyncStatus);
  rpc SubmitTransaction(TransactionPayload) returns (TransactionResponse);
  rpc GetMeshInfo(Empty) returns (MeshInfo);
  rpc GetLeaderStatus(Empty) returns (LeaderStatus);
  rpc GetOfflineQueue(Empty) returns (OfflineQueueStatus);
  rpc ForceCloudFlush(Empty) returns (FlushResponse);
}
```

Example with `grpcurl`:

```bash
grpcurl -plaintext localhost:50051 astra.sync.AstraSync/HealthCheck
grpcurl -plaintext -d '{"dataType": "INVENTORY"}' localhost:50051 astra.sync.AstraSync/SyncNow
```

## CRDT Design

### LWW-Register
Used for inventory items and cart state. Conflicts are resolved by:
1. Higher Lamport timestamp wins.
2. If timestamps are equal, lexicographically higher `KioskId` wins.

### PN-Counter
Used for inventory counts. Each kiosk tracks its own increments and decrements. Merge takes the per-origin maximum, ensuring no over-counting during partitions.

### OR-Set
Used for tags and collections. Add-wins semantics: if an element is added on one side and removed on the other during a partition, the add wins.

## Security Model

| Layer | Mechanism | Purpose |
|-------|-----------|---------|
| P2P Transport | Noise XX + QUIC | Authenticated, encrypted transport |
| P2P Messages | XChaCha20-Poly1305 | End-to-end encryption of sync payloads |
| Local Storage | SQLCipher AES-256 | Encryption at rest |
| Offline Payments | HMAC-SHA256 | Integrity and authenticity of queued tokens |
| Key Storage | Filesystem permissions (0o600) | Prevent unauthorized key access |

## Monitoring

Prometheus metrics are exposed on `daemon.metrics_addr` (default `127.0.0.1:9090`). Key metrics include:

- `astra_sync_records_total{data_type}` — Total records synced by type.
- `astra_p2p_peers_connected` — Number of connected P2P peers.
- `astra_raft_state` — Current Raft state (0=Follower, 1=Candidate, 2=Leader).
- `astra_cloud_flush_duration_seconds` — Histogram of cloud flush latency.
- `astra_offline_queue_size` — Number of pending offline payments.

## License

Apache-2.0 © Astra Systems

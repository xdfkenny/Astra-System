# Environment Variables Reference

> Complete reference from `.env.example` (123 variables)

## General

| Variable | Default | Description |
|----------|---------|-------------|
| `ASTRA_ENV` | development | Runtime environment |
| `ASTRA_LOG_LEVEL` | info | Log level (debug, info, warn, error) |
| `ASTRA_TRACE_SAMPLE_RATE` | 0.1 | OpenTelemetry trace sample rate |

## PostgreSQL

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_HOST` | localhost | Database host |
| `POSTGRES_PORT` | 5432 | Database port |
| `POSTGRES_DB` | astra_service | Database name |
| `POSTGRES_USER` | astra | Database user |
| `POSTGRES_PASSWORD` | - | Database password |
| `DATABASE_URL` | - | Full connection string |

## Redis

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_URL` | localhost:6379 | Redis connection |
| `REDIS_PASSWORD` | - | Redis password |

## NATS

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | nats://localhost:4222 | NATS connection |
| `NATS_CLUSTER_ID` | astra | NATS cluster ID |
| `NATS_JETSTREAM_DOMAIN` | astra | JetStream domain |

## Gateway

| Variable | Default | Description |
|----------|---------|-------------|
| `GATEWAY_PORT` | 8080 | HTTP listen port |
| `GATEWAY_JWT_ISSUER` | astra-system | JWT issuer |
| `GATEWAY_HMAC_SIGNING_KEY` | - | HMAC signing key |
| `GATEWAY_RATE_LIMIT_RPS` | 100 | Rate limit (req/s) |
| `GATEWAY_RATE_LIMIT_BURST` | 200 | Rate limit burst |
| `GATEWAY_CORS_ORIGINS` | * | CORS allowed origins |

## mTLS

| Variable | Default | Description |
|----------|---------|-------------|
| `ASTRA_TLS_CA_PATH` | - | CA certificate path |
| `ASTRA_TLS_CERT_PATH` | - | Service certificate path |
| `ASTRA_TLS_KEY_PATH` | - | Service key path |
| `ASTRA_MTLS_ENABLED` | true | Enable mTLS |

## Vault

| Variable | Default | Description |
|----------|---------|-------------|
| `VAULT_ADDR` | - | Vault server address |
| `VAULT_TOKEN` | - | Vault authentication token |
| `VAULT_MOUNT` | secret | Secrets mount path |
| `VAULT_BASE_PATH` | astra/ | Base path for secrets |

## Kiosk

| Variable | Default | Description |
|----------|---------|-------------|
| `KIOSK_ID` | - | Unique kiosk identifier |
| `KIOSK_STORE_ID` | - | Store identifier |
| `KIOSK_MESH_PSK` | - | P2P mesh pre-shared key |

## P2P / Sync Daemon

| Variable | Default | Description |
|----------|---------|-------------|
| `ASTRA_SYNCD_LISTEN_QUIC` | 0.0.0.0:4499 | QUIC listen address |
| `ASTRA_SYNCD_MDNS_SERVICE` | _astra-sync._udp.local | mDNS service type |
| `ASTRA_SYNCD_RAFT_ELECTION_TIMEOUT_MS` | 3000 | Raft election timeout |
| `ASTRA_SYNCD_DATA_DIR` | /var/lib/astra/syncd | Data directory |
| `ASTRA_SYNCD_BOOTSTRAP_PEERS` | - | Static peer addresses |

## Verifone Payments

| Variable | Default | Description |
|----------|---------|-------------|
| `VERIFONE_TERMINAL_IP` | 127.0.0.1 | Terminal IP |
| `VERIFONE_TERMINAL_PORT` | 10001 | Terminal port |
| `VERIFONE_MERCHANT_ID` | - | Merchant ID |
| `VERIFONE_TLS_CERT` | - | TLS certificate |
| `VERIFONE_TLS_KEY` | - | TLS key |

## Payment Orchestrator

| Variable | Default | Description |
|----------|---------|-------------|
| `PAYMENT_OFFLINE_TOKEN_SEED` | - | Token signing seed |
| `PAYMENT_MAX_OFFLINE_AMOUNT_CENTS` | 5000 | Max offline amount ($50) |
| `PAYMENT_OFFLINE_TTL_SECONDS` | 172800 | Token TTL (48h) |

## Observability

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | http://localhost:4317 | OTLP endpoint |
| `OTEL_SERVICE_NAME` | astra-service | Service name |
| `PROMETHEUS_PORT` | 9090 | Prometheus port |
| `LOKI_URL` | http://localhost:3100 | Loki URL |

## Updates

| Variable | Default | Description |
|----------|---------|-------------|
| `UPDATE_SERVER_URL` | - | Update server URL |
| `UPDATE_CHANNEL` | stable | Update channel |
| `UPDATE_PUBLIC_KEY` | - | Ed25519 public key |
| `UPDATE_CHECK_INTERVAL_SECONDS` | 21600 | Check interval (6h) |

## Secrets Management

| Variable | Default | Description |
|----------|---------|-------------|
| `ASTRA_SECRETS_BACKEND` | env | Secrets backend (vault, keyring, env) |
| `ASTRA_SECRETS_STORE_DIR` | - | Secrets store directory |

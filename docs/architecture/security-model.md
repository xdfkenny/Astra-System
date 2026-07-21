# Security Model

## Zero-Trust Architecture

Astra-System implements a **zero-trust security model**: no entity is trusted by default, whether inside or outside the network perimeter.

### Core Principles

1. **mTLS Everywhere** - All inter-service communication requires mutual TLS authentication
2. **SPIFFE Identities** - Every service has a cryptographically-verified identity
3. **Defense in Depth** - Multiple security layers (network, transport, application, data)
4. **Least Privilege** - Minimal permissions per service, role, and user
5. **PCI-DSS Isolation** - Card data path is completely isolated from main processing

## Security Layers

```
┌──────────────────────────────────────────────────┐
│                  Application Layer                │
│  JWT │ WebAuthn/FIDO2 │ RBAC │ HMAC Signing      │
├──────────────────────────────────────────────────┤
│                  Transport Layer                  │
│  mTLS (all services) │ Noise (P2P) │ TLS (HTTPS) │
├──────────────────────────────────────────────────┤
│                  Network Layer                    │
│  Network Policies │ Service Mesh │ Firewall       │
├──────────────────────────────────────────────────┤
│                  Data Layer                       │
│  SQLCipher (AES-256) │ HSM │ Vault │ Env Secrets │
├──────────────────────────────────────────────────┤
│                  Supply Chain                     │
│  Cosign (image signing) │ SBOM │ Trivy scanning  │
└──────────────────────────────────────────────────┘
```

## Authentication

### Employee Authentication: WebAuthn/FIDO2

File: `services/webauthn-service/`

```
Employee → Kiosk → WebAuthn begin → Device challenge
  → Biometric/PIN verification → Signed assertion
  → WebAuthn service verifies → JWT issued (15min TTL)
  → Employee authorized for store operations
```

**Endpoints:**
- `POST /v1/webauthn/register/begin` - Start credential registration
- `POST /v1/webauthn/register/finish` - Complete registration
- `POST /v1/webauthn/authenticate/begin` - Start authentication
- `POST /v1/webauthn/authenticate/finish` - Complete authentication

### Admin Authentication: JWT + RBAC

```
Admin → Admin Dashboard → Login → JWT (with role claims)
  → RouteGuard validates permissions → UI renders authorized views
```

**JWT Claims:**
- `sub`: User ID
- `tenant_id`: Tenant scope
- `role`: Assigned role (admin, manager, operator, support, viewer)
- `is_admin`: Boolean flag for GraphQL access
- `exp`: 15-minute TTL

### Inter-Service: mTLS + SPIFFE

```
Service A → mTLS handshake → Service B
  → Certificate validation (CA-signed SPIFFE SVID)
  → Identity verified → Request processed
```

**Certificate Infrastructure:**
- CA: Internal certificate authority
- SVIDs: SPIFFE Verifiable Identity Documents
- Rotation: Automatic via `infra/tls/generate-certs.sh`

### API Access: HMAC Request Signing

File: `services/api-gateway/internal/middleware/signing.go`

```
Client: HMAC-SHA256(request_body + timestamp + nonce, signing_key)
Server: Recomputes HMAC, verifies against Authorization header
  → Validates timestamp within tolerance (30s skew)
  → Checks nonce uniqueness (replay prevention)
```

## Data Encryption

### At Rest

| Storage | Encryption | Key Management |
|---------|------------|----------------|
| PostgreSQL | TDE (Transparent Data Encryption) | Vault |
| Kiosk SQLCipher | AES-256-GCM | Derived from device key |
| IndexedDB | Not encrypted (browser sandbox) | N/A |
| Backups | GPG/AES-256 | Vault |

### In Transit

| Path | Protocol | Cipher |
|------|----------|--------|
| Browser → Gateway | HTTPS (TLS 1.3) | ECDHE + AES-256-GCM |
| Service → Service | gRPC + mTLS (TLS 1.3) | ECDHE + AES-256-GCM |
| P2P Mesh | QUIC + Noise Protocol | Noise XX + ChaCha20-Poly1305 |
| Kiosk → Cloud | HTTPS (TLS 1.3) | ECDHE + AES-256-GCM |

## PCI-DSS Compliance

### Payment Data Isolation

```
Kiosk App (React)
    │ Payment intent (amount, items)
    ▼
Payment Sidecar (Rust) — isolated process, no card data in main memory
    │ Verifone protocol
    ▼
Verifone Terminal — PCI-PTS certified, card data never leaves terminal
    │ Tokenized response
    ▼
Payment Orchestrator (Go) — stores tokens, not PAN
```

**Key Safeguards:**
- Card data never enters kiosk main memory
- Verifone FFI bridge (`packages/verifone-ffi/`) runs in separate process
- Payment tokens stored, never raw PAN
- Offline tokens are time-limited and amount-capped
- Full audit trail for all payment operations

## Supply Chain Security

| Measure | Tool | Phase |
|---------|------|-------|
| Image Signing | Cosign | Post-build |
| SBOM Generation | Syft (SPDX) | Post-build |
| Vulnerability Scan | Trivy | Post-build |
| Dependency Audit | cargo audit, govulncheck | CI |
| Secret Scanning | Gitleaks | CI |
| Code Linting | Clippy, golangci-lint, Biome | Pre-commit |
| Commit Signing | GPG | Pre-push |

## Secrets Management

**Backend:** Vault (Hashicorp) for production, environment variables for development.

**Supported backends** (via `ASTRA_SECRETS_BACKEND`):
- `vault` - Hashicorp Vault (production)
- `keyring` - OS keychain (development)
- `env` - Environment variables (local dev)

**Stored Secrets:**
- Database credentials
- JWT signing keys
- HMAC signing keys
- TLS private keys
- Mesh PSK (pre-shared key)
- Verifone merchant credentials

## RBAC Model

### Role Hierarchy

| Role | Scope | Permissions |
|------|-------|-------------|
| `admin` | Tenant | Full CRUD on all resources |
| `manager` | Store | Ops + employee management |
| `operator` | Store | Daily operations (overrides, voids) |
| `support` | Tenant | Read-only + support actions |
| `viewer` | Store | Read-only dashboards |

### Resource Permissions

Every resource (`kiosks`, `menu`, `inventory`, `orders`, `payments`, `employees`, etc.) has CRUD permissions checked via `RouteGuard` components and GraphQL resolvers.

## Network Security

| Layer | Control |
|-------|---------|
| Kubernetes | Network policies (namespace isolation, ingress rules) |
| Service Mesh | mTLS enforcement, authorization policies |
| Kiosk | AppArmor profiles, seccomp filters, read-only filesystem |
| P2P | Noise Protocol encrypted, PSK-authenticated peers |
| API | Rate limiting (Redis token bucket), circuit breakers |

## Security Monitoring

- **Audit Logs**: Immutable, append-only, hash-chained (`audit_logs` table, partitioned monthly)
- **Event Store**: Complete domain event history
- **SIEM Integration**: Structured JSON logs shipped to Loki
- **Alerting**: Grafana alerts for security events
- **Metrics**: Rate limit violations, auth failures, unusual patterns

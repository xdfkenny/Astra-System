# Authentication

## Overview

Multi-layered authentication system supporting employee (WebAuthn/FIDO2), admin (JWT), inter-service (mTLS/SPIFFE), and API (HMAC) authentication.

## Authentication Methods

| Method | Users | Protocol | TTL | Details |
|--------|-------|----------|-----|---------|
| WebAuthn/FIDO2 | Store employees | FIDO2 CTAP | Per session | [WebAuthn Service](../backend/webauthn-service.md) |
| JWT Bearer | Admin users | JWT (EdDSA) | 15 min | Admin dashboard, API |
| mTLS + SPIFFE | Services | TLS 1.3 | 24h | All inter-service communication |
| HMAC-SHA256 | Kiosk → Gateway | HMAC signed requests | Per request | API Gateway middleware |

## WebAuthn / FIDO2

File: `services/webauthn-service/`

**Registration Flow:**
```
1. Employee → POST /v1/webauthn/register/begin
2. Server returns challenge + relying party info
3. Browser/platform creates credential via WebAuthn API
4. Employee biometric/PIN verification
5. POST /v1/webauthn/register/finish (attestation)
6. Credential stored in PostgreSQL (employees table)
```

**Authentication Flow:**
```
1. Employee → POST /v1/webauthn/authenticate/begin
2. Server returns challenge for stored credentials
3. Browser/platform signs challenge with private key
4. POST /v1/webauthn/authenticate/finish (assertion)
5. Server verifies signature, issues session
```

## JWT Authentication

**Token Structure:**
```json
{
  "sub": "user-uuid",
  "tenant_id": "tenant-uuid",
  "role": "admin",
  "is_admin": true,
  "exp": 1712345678,
  "iat": 1712344778
}
```

**Validation:**
- EdDSA signature verification (Ed25519)
- `exp` claim within acceptable clock skew (30s)
- RBAC enforcement via RouteGuard components and GraphQL resolvers

## mTLS / SPIFFE

- Every service has a SPIFFE identity: `spiffe://astra-system/{service-name}`
- Certificates issued by internal CA (24h TTL, auto-rotated)
- gRPC connections require valid client certificates
- SPIFFE IDs verified on every request

## HMAC Request Signing

File: `services/api-gateway/internal/middleware/signing.go`

```
Authorization: HMAC-SHA256 timestamp={ts},nonce={n},signature={sig}
```

- Header includes timestamp (max 30s skew) and unique nonce
- Secret key per-kiosk, provisioned during manufacturing
- Replay attack prevention via nonce tracking in Redis

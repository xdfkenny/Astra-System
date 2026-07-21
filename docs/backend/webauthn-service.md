# WebAuthn Service

## Overview

The WebAuthn Service (`services/webauthn-service/`) implements FIDO2/WebAuthn passwordless authentication for store employees and admin users.

## Responsibilities

- WebAuthn credential registration
- WebAuthn assertion verification
- Override token validation (manager override)
- Credential management (list, remove)
- Session token issuance

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/webauthn/register/begin` | Start credential registration |
| POST | `/v1/webauthn/register/finish` | Complete registration |
| POST | `/v1/webauthn/authenticate/begin` | Start authentication |
| POST | `/v1/webauthn/authenticate/finish` | Complete authentication |

## Auth Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/auth/webauthn/begin` | Begin WebAuthn assertion |
| POST | `/v1/auth/webauthn/verify` | Verify assertion |
| POST | `/v1/auth/override/validate` | Validate override token |

## Integration

- Credentials stored in `employees.webauthn_credential_id` (PostgreSQL)
- Used by kiosk UI for employee authentication (overrides, voids)
- Used by admin dashboard for admin authentication
- Supports multiple credentials per user (phone, laptop, security key)

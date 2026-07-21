# Encryption

## At Rest

| Storage Layer | Algorithm | Key Management |
|---------------|-----------|----------------|
| PostgreSQL 16 | TDE (AES-256) | Vault (auto-rotation) |
| Kiosk SQLCipher | AES-256-GCM | Derived from device TPM + PIN |
| IndexedDB (browser) | Browser sandbox isolation | N/A |
| Backups | GPG (AES-256) | Vault |
| Secrets (Vault) | AES-256-GCM | Auto-unseal (KMS) |

### SQLCipher Configuration

Used by the Rust sync daemon for offline storage:

```
PRAGMA key = 'derived_from_tpm_and_pin';
PRAGMA cipher_page_size = 4096;
PRAGMA kdf_iter = 64000;
PRAGMA cipher_hmac_algorithm = HMAC_SHA512;
PRAGMA cipher_kdf_algorithm = PBKDF2_HMAC_SHA512;
```

## In Transit

| Path | Protocol | Cipher Suite |
|------|----------|--------------|
| Browser → Gateway | HTTPS (TLS 1.3) | TLS_AES_256_GCM_SHA384 |
| Service → Service | gRPC + mTLS (TLS 1.3) | TLS_AES_256_GCM_SHA384 |
| P2P Mesh | QUIC + Noise XX | ChaCha20-Poly1305 |
| Kiosk → Cloud | HTTPS (TLS 1.3) | TLS_AES_256_GCM_SHA384 |
| Verifone Terminal | PCI-PTS encrypted | Vendor-specific |

## Key Management

| Key Type | Storage | Rotation | Length |
|----------|---------|----------|--------|
| Service mTLS keys | Vault + local FS | 24h | 2048-bit RSA / ECDSA P-256 |
| Kiosk identity keys | TPM + SQLCipher | 90 days | Ed25519 |
| JWT signing keys | Vault | 24h | Ed25519 |
| HMAC signing keys | Vault + kiosk config | Daily | 256-bit |
| Mesh PSK | Kiosk config (encrypted) | Per provisioning | 256-bit |
| Database passwords | Vault | 30 days | 64-char alphanumeric |

## Certificate Infrastructure

File: `infra/tls/generate-certs.sh`

- Internal CA issues all service certificates
- Service certificates: 24h TTL, auto-rotated via Vault
- Kiosk certificates: 90-day TTL, renewed via update daemon
- All certificates include SANs for DNS names and `localhost`
- Revocation via CRL (Certificate Revocation List)

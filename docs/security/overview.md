# Security Overview

## Philosophy

Astra-System implements a **defense-in-depth, zero-trust security model**. No entity is trusted by default, all communication is authenticated and encrypted, and sensitive data paths are isolated.

## Security Layers

| Layer | Controls | Documentation |
|-------|----------|---------------|
| Application | JWT, WebAuthn/FIDO2, RBAC, HMAC signing | [Authentication](./authentication.md) |
| Transport | mTLS (TLS 1.3), Noise Protocol, HTTPS | [Encryption](./encryption.md) |
| Network | K8s network policies, AppArmor, seccomp | [K8s Deployment](../infrastructure/kubernetes.md) |
| Data | SQLCipher AES-256, Vault, env encryption | [Encryption](./encryption.md) |
| Supply Chain | Cosign signing, SBOM, Trivy scanning | [CI/CD](../infrastructure/ci-cd.md) |

## Key Principles

1. **mTLS Everywhere** - All inter-service communication requires mutual TLS
2. **Least Privilege** - Minimal permissions per service, role, user
3. **PCI-DSS Isolation** - Card data never enters kiosk application memory
4. **Immutable Audit** - Hash-chained, append-only audit logs
5. **Short-Lived Credentials** - Automatic rotation of all secrets and certificates

## Threat Model

| Threat | Mitigation |
|--------|------------|
| Kiosk physical compromise | Encrypted storage (SQLCipher), TPM, tamper detection |
| Network eavesdropping | All traffic encrypted (TLS 1.3 / Noise Protocol) |
| Service impersonation | mTLS with SPIFFE identities |
| Replay attacks | HMAC nonces, timestamps, idempotency keys |
| Data breach at rest | Column-level encryption, Vault for keys |
| Supply chain attack | Cosign image signing, SBOM verification, Trivy scanning |

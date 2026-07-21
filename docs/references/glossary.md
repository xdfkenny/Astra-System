# Glossary

## Terms

| Term | Definition |
|------|------------|
| **A/B Partition** | Dual-partition update strategy (active + standby) for kiosk OS |
| **API Gateway** | Single entry point for all external API requests (Fiber:8080) |
| **AppArmor** | Linux kernel security module for mandatory access control |
| **CRDT** | Conflict-free Replicated Data Type — data structure that converges across peers |
| **Drizzle ORM** | TypeScript ORM used for database schema definitions |
| **Event Sourcing** | Storing state changes as an append-only event log |
| **Fiber** | Go HTTP web framework used for the API Gateway |
| **Ghost Cart** | Cart created on mobile device, transferred to kiosk via WebRTC/NFC |
| **gRPC** | High-performance RPC framework using Protocol Buffers |
| **HLC** | Hybrid Logical Clock — causal ordering without synchronized clocks |
| **HMAC** | Hash-based Message Authentication Code (HMAC-SHA256) |
| **JetStream** | NATS persistence layer for at-least-once delivery |
| **libp2p** | Modular network stack for P2P applications |
| **LWW-Register** | Last-Writer-Wins Register CRDT |
| **mTLS** | Mutual TLS — both sides authenticate via certificates |
| **mDNS** | Multicast DNS — zero-configuration service discovery |
| **MFE** | Micro-Frontend — independently deployable frontend module |
| **Module Federation** | Webpack/Vite plugin for sharing code across builds |
| **NATS** | Cloud-native messaging system (event bus) |
| **Noise Protocol** | Framework for building cryptographic protocols (P2P encryption) |
| **Offline Token** | Signed payment token generated when cloud is unreachable |
| **ONNX** | Open Neural Network Exchange — ML model format |
| **OR-Set** | Observed-Removed Set CRDT |
| **OTA** | Over-the-Air — wireless software updates |
| **Outbox Pattern** | Transactional outbox for reliable event publication |
| **P2P** | Peer-to-peer — direct communication between kiosks |
| **PCI-DSS** | Payment Card Industry Data Security Standard |
| **PLU** | Price Look-Up code (produce identification) |
| **PN-Counter** | Positive-Negative Counter CRDT |
| **QUIC** | Multiplexed transport protocol over UDP |
| **Raft** | Distributed consensus algorithm for leader election |
| **RBAC** | Role-Based Access Control |
| **RTC** | Real-Time Clock (system clock) |
| **seccomp** | Linux kernel security facility for syscall filtering |
| **SPIFFE** | Secure Production Identity Framework for Everyone |
| **SPDX** | Software Package Data Exchange — SBOM format |
| **SQLCipher** | Encrypted SQLite database (AES-256-GCM) |
| **SSE** | Server-Sent Events — unidirectional real-time updates |
| **Strangler Fig** | Migration pattern for gradual system replacement |
| **SVID** | SPIFFE Verifiable Identity Document |
| **TanStack Query** | React data-fetching and caching library |
| **Turborepo** | Monorepo task orchestration tool |
| **Valtio** | Reactive state management (proxy-based) |
| **WebAuthn** | Web Authentication API (FIDO2 standard) |
| **XState** | JavaScript state machine library (v5) |
| **Zustand** | Small, fast state management library |
| **zstd** | Zstandard compression algorithm (for sync batches) |

## Acronyms

| Acronym | Full Form |
|---------|-----------|
| CI/CD | Continuous Integration / Continuous Deployment |
| CQRS | Command Query Responsibility Segregation |
| CRDT | Conflict-free Replicated Data Type |
| CSP | Content Security Policy |
| CV | Computer Vision |
| DR | Disaster Recovery |
| E2E | End-to-End (testing) |
| EKS | Amazon Elastic Kubernetes Service |
| FFI | Foreign Function Interface |
| FIDO2 | Fast IDentity Online 2 |
| GHA | GitHub Actions |
| GKE | Google Kubernetes Engine |
| HLC | Hybrid Logical Clock |
| HPA | Horizontal Pod Autoscaler |
| HSM | Hardware Security Module |
| IaC | Infrastructure as Code |
| JWT | JSON Web Token |
| K8s | Kubernetes |
| MFE | Micro-Frontend |
| ML | Machine Learning |
| mTLS | Mutual Transport Layer Security |
| NFC | Near-Field Communication |
| OTA | Over-the-Air |
| OTLP | OpenTelemetry Protocol |
| P2P | Peer-to-Peer |
| PAN | Primary Account Number (card number) |
| PCI | Payment Card Industry |
| PIN | Personal Identification Number |
| PKI | Public Key Infrastructure |
| PLU | Price Look-Up |
| POS | Point of Sale |
| PWA | Progressive Web Application |
| RBAC | Role-Based Access Control |
| RPO | Recovery Point Objective |
| RTO | Recovery Time Objective |
| SPA | Single Page Application |
| SPDX | Software Package Data Exchange |
| SSE | Server-Sent Events |
| SVID | SPIFFE Verifiable Identity Document |
| TDE | Transparent Data Encryption |
| TFLite | TensorFlow Lite |
| TTL | Time To Live |
| UFSM | Universal Finite State Machine |
| WAF | Web Application Firewall |
| WAL | Write-Ahead Log |
| WASM | WebAssembly |

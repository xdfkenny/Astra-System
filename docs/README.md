# Astra-System Documentation

> Production-grade, offline-first, automated self-checkout platform for 24/7 retail environments.

## Documentation Structure

### Architecture
| Document | Description |
|----------|-------------|
| [System Overview](architecture/overview.md) | High-level architecture, design philosophy, tech stack |
| [System Design Patterns](architecture/system-design.md) | Microservices, micro-frontends, event sourcing, CQRS |
| [Offline-First Strategy](architecture/offline-first.md) | 48h resilience, CRDTs, P2P mesh, three-tier sync |
| [Security Model](architecture/security-model.md) | Zero-trust, mTLS, WebAuthn, PCI-DSS compliance |

### Backend
| Document | Description |
|----------|-------------|
| [Microservices Overview](backend/microservices.md) | All 13 Go services, responsibilities, communication |
| [API Gateway](backend/api-gateway.md) | Fiber HTTP gateway, rate limiting, auth, routing |
| [Menu Service](backend/menu-service.md) | Catalog, categories, items, modifiers |
| [Cart Service](backend/cart-service.md) | CRDT-based cart, merge, ghost carts |
| [Order Service](backend/order-service.md) | Order lifecycle, fulfillment, refunds |
| [Inventory Service](backend/inventory-service.md) | Stock levels, reservations, soft holds |
| [Payment Orchestrator](backend/payment-orchestrator.md) | Payment flows, offline tokens, Verifone integration |
| [Sync Service](backend/sync-service.md) | Cloud sync gateway, batch upload/download, heartbeats |
| [WebAuthn Service](backend/webauthn-service.md) | FIDO2 passwordless authentication |
| [Admin GraphQL](backend/admin-graphql.md) | Admin dashboard API |
| [ML Lane Intelligence](backend/ml-lane-intel.md) | YOLOv8n lane queue estimation |
| [REST API Reference](backend/rest-api.md) | Complete REST API endpoint inventory |
| [gRPC API Reference](backend/grpc-api.md) | Protocol Buffer service definitions |

### Frontend
| Document | Description |
|----------|-------------|
| [Micro-Frontends Overview](frontend/micro-frontends.md) | Module Federation architecture, MFEs |
| [Kiosk Applications](frontend/kiosk-apps.md) | Shell, menu, cart, payment, admin apps |
| [State Management](frontend/state-management.md) | XState v5, Zustand, Valtio, TanStack Query |
| [Module Federation](frontend/module-federation.md) | Remote loading, versioning, atomic rollback |

### Database
| Document | Description |
|----------|-------------|
| [Schema Overview](database/schema.md) | All 22 tables, enums, relationships |
| [Migrations Guide](database/migrations.md) | Migration workflow, partitioning |
| [Entity Relationships](database/entities.md) | Domain entity relationships and data flow |

### Infrastructure
| Document | Description |
|----------|-------------|
| [Docker Setup](infrastructure/docker.md) | Containerization, Dockerfiles, Compose stacks |
| [Kubernetes Deployment](infrastructure/kubernetes.md) | Helm chart, K8s manifests, HPA, network policies |
| [Observability](infrastructure/monitoring.md) | Prometheus, Grafana, Loki, Jaeger, OpenTelemetry |
| [CI/CD Pipeline](infrastructure/ci-cd.md) | GitHub Actions, build, test, security scan, release |

### Networking
| Document | Description |
|----------|-------------|
| [P2P Mesh Sync](networking/p2p-mesh.md) | libp2p, QUIC, Raft consensus, CRDT replication |
| [Communication Protocols](networking/protocols.md) | gRPC, NATS, REST, WebRTC, mDNS |

### Development
| Document | Description |
|----------|-------------|
| [Development Setup](development/setup.md) | Prerequisites, installation, dev environment |
| [Development Workflow](development/workflow.md) | Git workflow, building, testing, conventions |
| [Testing Guide](development/testing.md) | Unit, integration, E2E, chaos testing |
| [Coding Standards](development/coding-standards.md) | Linting, formatting, conventions per language |

### Operations
| Document | Description |
|----------|-------------|
| [Deployment Guide](operations/deployment.md) | Production deployment, scaling, updates |
| [Incident Response](operations/runbooks/incident-response.md) | Incident response procedures |
| [Offline Mode Operations](operations/runbooks/offline-mode-operations.md) | Offline operations guide |
| [P2P Partition Recovery](operations/runbooks/p2p-partition-recovery.md) | Mesh recovery procedures |
| [Payment Failure Runbook](operations/runbooks/payment-failure-runbook.md) | Payment troubleshooting |

### Security
| Document | Description |
|----------|-------------|
| [Security Overview](security/overview.md) | Security architecture, threat model |
| [Authentication](security/authentication.md) | JWT, WebAuthn/FIDO2, HMAC signing, SPIFFE |
| [Encryption](security/encryption.md) | mTLS, Noise Protocol, SQLCipher, Ed25519 |

### Protobuf
| Document | Description |
|----------|-------------|
| [Protobuf Overview](protobuf/overview.md) | Proto definitions, code generation |
| [Service Definitions](protobuf/service-definitions.md) | All service contracts |

### Extensions
| Document | Description |
|----------|-------------|
| [Plugin Architecture](extensions/plugins.md) | Extension mechanisms, remote modules |
| [Legacy POS Adapter](extensions/legacy-pos-adapter.md) | Strangler Fig migration pattern |

### References
| Document | Description |
|----------|-------------|
| [Environment Variables](references/env-vars.md) | All configuration variables |
| [Glossary](references/glossary.md) | Domain terminology |
| [Architecture Decision Records](references/adrs.md) | Key technical decisions |

## Quick Links

- [Main README](../README.md) - Project overview
- [ARCHITECTURE.md](../ARCHITECTURE.md) - Original architecture document
- [API Backend Reference](API-BACKEND-ASTRA.md) - Full API inventory
- [GitHub Repository](https://github.com/MOTHER/Astra-System)

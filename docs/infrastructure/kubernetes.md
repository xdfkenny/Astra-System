# Kubernetes Deployment

## Overview

Production deployment uses **Kubernetes** with **Helm** chart packaging and **Terraform** for cloud infrastructure provisioning.

## Helm Chart

**Location:** `infra/helm/astra-service/`

```
Chart.yaml          → Chart metadata (apiVersion, version, dependencies)
values.yaml         → 192 lines of default values (all services, replicas, resources)
templates/
├── configmap.yaml  → Shared configuration
├── deployment-*.yaml  → Service deployments
├── statefulset-*.yaml → PostgreSQL, Redis, NATS
├── service.yaml    → Service definitions
├── ingress.yaml    → Ingress rules
├── hpa.yaml        → Horizontal Pod Autoscaler
└── network-policy.yaml → Network isolation
```

## Key Configuration (values.yaml)

### Service Replicas

| Service | Default Replicas | Notes |
|---------|-----------------|-------|
| gateway | 3 | Horizontally scalable |
| menu-service | 2 | Cache-heavy, read mostly |
| cart-service | 3 | Stateful, versioned |
| order-service | 2 | Transactional |
| inventory-service | 2 | Stock operations |
| payment-orchestrator | 2 | Payment flows |
| sync-service | 2 | Sync gateway |
| webauthn-service | 2 | Auth |
| admin-graphql | 1 | Admin only |
| ml-lane-intel | 1 | ML inference |

### Resource Limits

Default memory limits: 256Mi-512Mi per service
Default CPU limits: 500m-1000m per service

## Kubernetes Manifests

**Location:** `infra/k8s/`

16 standalone manifest files for manual deployment (alternative to Helm):

| File | Purpose |
|------|---------|
| `namespace.yaml` | `astra-service` namespace |
| `configmap.yaml` | Environment configuration |
| `secret-template.yaml` | Secret template |
| `deployment-gateway.yaml` | Gateway deployment |
| `deployment-cart-service.yaml` | Cart service |
| `deployment-inventory-service.yaml` | Inventory service |
| `deployment-menu-service.yaml` | Menu service |
| `deployment-order-service.yaml` | Order service |
| `deployment-payment-orchestrator.yaml` | Payment orchestrator |
| `deployment-sync-service.yaml` | Sync service |
| `statefulset-postgresql.yaml` | PostgreSQL StatefulSet |
| `statefulset-redis.yaml` | Redis StatefulSet |
| `statefulset-nats.yaml` | NATS StatefulSet |
| `ingress.yaml` | Ingress rules |
| `hpa.yaml` | Horizontal Pod Autoscaler |
| `network-policy.yaml` | Network policies |

## Network Policies

Micro-segmentation with Kubernetes Network Policies:

- **Default deny** all ingress/egress
- **Allow** from gateway to all services (port-specific)
- **Allow** services to PostgreSQL (5432), Redis (6379), NATS (4222)
- **Allow** from sync-service to external (for cloud sync)
- **Deny** all other traffic

## HPA Configuration

Horizontal Pod Autoscaling based on:
- CPU utilization (target: 70%)
- Memory utilization (target: 80%)
- Custom metrics (request rate, queue depth)

## Ingress

TLS-terminated ingress with:
- Let's Encrypt certificates (cert-manager)
- Path-based routing to gateway
- WebSocket support for SSE endpoints

## Terraform

**Location:** `infra/terraform/`

```hcl
main.tf       → K8s cluster, Helm release, secrets
variables.tf  → Input variables (region, cluster size, etc.)
```

**Provisioned Resources:**
- EKS (Amazon Elastic Kubernetes Service) cluster
- Node groups (compute-optimized)
- VPC with public/private subnets
- RDS PostgreSQL (Multi-AZ)
- ElastiCache Redis (Cluster mode)
- NATS (self-managed on K8s)

## Storage

- **PostgreSQL:** Managed RDS (production) or StatefulSet with PVC (self-managed)
- **Redis:** ElastiCache (production) or StatefulSet (self-managed)
- **NATS:** StatefulSet with PVC for JetStream file storage
- **Kiosk data:** Ephemeral (no persistent storage needed)

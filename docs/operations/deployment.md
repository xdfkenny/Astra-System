# Deployment Guide

## Deployment Options

| Method | Environment | Complexity | Use Case |
|--------|-------------|------------|----------|
| Docker Compose | Development | Low | Local testing, single-machine |
| Docker Compose (prod) | Production | Medium | Single-store deployment |
| Kubernetes + Helm | Production | High | Multi-store, cloud-managed |
| Terraform + Helm | Production | High | IaC, multi-region |

## Docker Compose Deployment

### Development

```bash
# Start full dev stack
docker compose up -d

# View logs
docker compose logs -f gateway menu-service
```

### Production

```bash
# Start hardened production stack
docker compose -f docker-compose.prod.yml up -d

# Verify all services
curl http://localhost:8080/health
```

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster (EKS/GKE recommended)
- `kubectl` configured
- Helm 3 installed

### Using Helm

```bash
# Install the chart
helm install astra-service infra/helm/astra-service \
  --namespace astra-service \
  --create-namespace \
  -f infra/helm/astra-service/values.yaml

# Upgrade
helm upgrade astra-service infra/helm/astra-service \
  -f infra/helm/astra-service/values.yaml

# Uninstall
helm uninstall astra-service -n astra-service
```

### Using Manifests

```bash
# Apply all manifests
kubectl apply -f infra/k8s/

# Check status
kubectl get pods -n astra-service
kubectl get svc -n astra-service
```

## Terraform Deployment

```bash
cd infra/terraform
terraform init
terraform plan -var-file=prod.tfvars
terraform apply -var-file=prod.tfvars
```

## Windows Kiosk Deployment

See [Windows Installer Guide](../WINDOWS-INSTALLER.md).

## Update Strategy

### Cloud Services

- **Rolling updates** via Kubernetes (zero-downtime)
- **Health checks** must pass before traffic is routed
- **Rollback** via `kubectl rollout undo` or Helm rollback

### Kiosk Software

- **OTA updates** via `update-server` service
- **A/B partition** strategy (active + standby)
- **Ed25519 signature verification** for all updates
- **Automatic rollback** on health check failure

## Post-Deployment Verification

```bash
# Check all services
curl http://localhost:8080/health
curl http://localhost:8080/live
curl http://localhost:8080/ready

# Verify menu service
curl http://localhost:8085/v1/menu/{store_id}

# Verify WebAuthn
curl -X POST http://localhost:8090/v1/webauthn/authenticate/begin

# Check P2P mesh
curl http://localhost:4499/healthz
```

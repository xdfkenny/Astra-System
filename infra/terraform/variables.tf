variable "kubeconfig_path" {
  description = "Path to the kubeconfig file for the target Kubernetes cluster"
  type        = string
  default     = "~/.kube/config"
}

variable "namespace" {
  description = "Kubernetes namespace for the Astra deployment"
  type        = string
  default     = "astra-service"
}

variable "chart_path" {
  description = "Path to the Helm chart directory. Empty uses default path."
  type        = string
  default     = ""
}

variable "container_registry" {
  description = "Container registry for Astra service images"
  type        = string
  default     = "ghcr.io/astra-systems"
}

variable "image_tag" {
  description = "Container image tag to deploy"
  type        = string
  default     = "latest"
}

variable "environment" {
  description = "Deployment environment (staging, production)"
  type        = string
  default     = "staging"
}

variable "dns_domain" {
  description = "Internal DNS domain for the cluster"
  type        = string
  default     = "astra.svc.cluster.local"
}

variable "nats_url" {
  description = "NATS JetStream URL"
  type        = string
  default     = "nats://nats:4222"
}

variable "redis_address" {
  description = "Redis address"
  type        = string
  default     = "redis:6379"
}

variable "postgres_host" {
  description = "PostgreSQL host"
  type        = string
  default     = "postgres"
}

variable "postgres_password" {
  description = "PostgreSQL password (auto-generated if empty)"
  type        = string
  default     = ""
}

variable "otel_endpoint" {
  description = "OpenTelemetry Collector endpoint"
  type        = string
  default     = "http://otel-collector:4317"
}

variable "gateway_replicas" {
  description = "Number of gateway replicas"
  type        = number
  default     = 2
}

variable "gateway_host" {
  description = "Ingress hostname for the API gateway"
  type        = string
  default     = "api.astra.local"
}

variable "gateway_cpu" {
  description = "CPU limit for gateway"
  type        = string
  default     = "1"
}

variable "gateway_memory" {
  description = "Memory limit for gateway"
  type        = string
  default     = "512Mi"
}

variable "enable_network_policy" {
  description = "Enable network policies"
  type        = bool
  default     = true
}

# ── Outputs ───────────────────────────────────────────────────

output "namespace" {
  description = "The Kubernetes namespace where Astra is deployed"
  value       = kubernetes_namespace.astra.metadata[0].name
}

output "gateway_address" {
  description = "The ingress hostname for the API gateway"
  value       = var.gateway_host
}

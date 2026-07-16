# Astra-System Infrastructure
# Provision the underlying cloud resources for the Astra kiosk mesh.
#
# Usage:
#   terraform init
#   terraform workspace new staging
#   terraform plan -var-file=environments/staging.tfvars
#   terraform apply -var-file=environments/staging.tfvars

terraform {
  required_version = "~> 1.9"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.32"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.15"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  backend "s3" {
    bucket = "astra-terraform-state"
    key    = "astra-service/terraform.tfstate"
    region = "us-east-1"
    encrypt = true
    dynamodb_table = "astra-terraform-locks"
  }
}

provider "kubernetes" {
  config_path = var.kubeconfig_path
}

provider "helm" {
  kubernetes {
    config_path = var.kubeconfig_path
  }
}

# ── Random suffixes for resource naming ───────────────────────

resource "random_id" "suffix" {
  byte_length = 4
}

# ── Kubernetes Namespace ──────────────────────────────────────

resource "kubernetes_namespace" "astra" {
  metadata {
    name = var.namespace
    labels = {
      "app.kubernetes.io/name"       = "astra-service"
      "app.kubernetes.io/managed-by" = "terraform"
    }
  }
}

# ── PostgreSQL Credentials Secret ─────────────────────────────

resource "random_password" "postgres" {
  length  = 24
  special = false
}

resource "kubernetes_secret" "postgres_credentials" {
  metadata {
    name      = "postgres-credentials"
    namespace = kubernetes_namespace.astra.metadata[0].name
    labels = {
      "app.kubernetes.io/managed-by" = "terraform"
    }
  }
  data = {
    password = random_password.postgres.result
  }
}

# ── Astra Helm Release ────────────────────────────────────────

resource "helm_release" "astra" {
  name       = "astra"
  namespace  = kubernetes_namespace.astra.metadata[0].name
  chart      = var.chart_path != "" ? var.chart_path : "${path.module}/../helm/astra-service"
  timeout    = 600

  values = [
    templatefile("${path.module}/values.tftpl", {
      global_registry         = var.container_registry
      image_tag               = var.image_tag
      environment             = var.environment
      dns_domain              = var.dns_domain
      nats_url                = var.nats_url
      redis_address           = var.redis_address
      postgres_host           = var.postgres_host
      postgres_password       = random_password.postgres.result
      otel_endpoint           = var.otel_endpoint
      gateway_replicas        = var.gateway_replicas
      gateway_host            = var.gateway_host
      gateway_cpu             = var.gateway_cpu
      gateway_memory          = var.gateway_memory
      enable_network_policy   = var.enable_network_policy
    })
  ]

  set {
    name  = "global.postgres.secretName"
    value = kubernetes_secret.postgres_credentials.metadata[0].name
  }

  depends_on = [kubernetes_secret.postgres_credentials]
}

# ─────────────────────────────────────────────────────────────
# NeuroForge SDX — Terraform Deployment Configuration
# ─────────────────────────────────────────────────────────────

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.27"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
  }

  # Backend configuration — override in CI via -backend-config.
  # Example: terraform init -backend-config="bucket=sdx-tf-state"
  # backend "s3" {}
}

# ─── Variables ───

variable "namespace" {
  description = "Kubernetes namespace for SDX deployment"
  type        = string
  default     = "sdx-system"
}

variable "chart_path" {
  description = "Path to the SDX Helm chart. Use a relative path for local dev, or a chart repository URL for CI."
  type        = string
  default     = "../helm/neuroforge-sdx"
}

variable "chart_values_file" {
  description = "Path to a values override file"
  type        = string
  default     = ""
}

variable "release_name" {
  description = "Helm release name"
  type        = string
  default     = "neuroforge-sdx"
}

variable "kubeconfig_path" {
  description = "Path to kubeconfig. Leave empty to use in-cluster config."
  type        = string
  default     = ""
}

# ─── Providers ───

provider "kubernetes" {
  config_path = var.kubeconfig_path != "" ? var.kubeconfig_path : null
}

provider "helm" {
  kubernetes {
    config_path = var.kubeconfig_path != "" ? var.kubeconfig_path : null
  }
}

# ─── Namespace ───

resource "kubernetes_namespace" "sdx" {
  metadata {
    name = var.namespace
    labels = {
      "app.kubernetes.io/part-of" = "neuroforge-sdx"
      "app.kubernetes.io/managed-by" = "terraform"
    }
  }
}

# ─── Helm Release ───

resource "helm_release" "sdx" {
  name       = var.release_name
  chart      = var.chart_path
  namespace  = kubernetes_namespace.sdx.metadata[0].name
  
  create_namespace = false
  wait             = true
  timeout          = 600

  dynamic "set" {
    for_each = var.chart_values_file != "" ? [1] : []
    content {
      name  = "unused"
      value = "unused"
    }
  }

  values = var.chart_values_file != "" ? [file(var.chart_values_file)] : []
}

# General

variable "name_prefix" {
  description = "Prefix used when naming IAM resources."
  type        = string
  default     = "bravo"
}

variable "tags" {
  description = "Tags applied to IAM resources."
  type        = map(string)
  default     = {}
}

# EKS OIDC / IRSA

variable "eks_oidc_provider_arn" {
  description = "ARN of the IAM OIDC provider associated with the EKS cluster."
  type        = string

  validation {
    condition = can(regex(
      "^arn:aws[a-zA-Z-]*:iam::[0-9]{12}:oidc-provider/",
      var.eks_oidc_provider_arn
    ))

    error_message = "eks_oidc_provider_arn must be a valid IAM OIDC provider ARN."
  }
}

variable "eks_oidc_provider_url" {
  description = "EKS OIDC issuer URL without the https:// prefix."
  type        = string

  validation {
    condition     = !startswith(var.eks_oidc_provider_url, "https://")
    error_message = "eks_oidc_provider_url must not include the https:// prefix."
  }
}

# Kubernetes

variable "kubernetes_namespace" {
  description = "Kubernetes namespace containing the application workloads."
  type        = string
  default     = "default"
}

variable "order_service_account_name" {
  description = "Kubernetes ServiceAccount used by order-service."
  type        = string
  default     = "order-service"
}

variable "payment_service_account_name" {
  description = "Kubernetes ServiceAccount used by payment-service."
  type        = string
  default     = "payment-service"
}

variable "shipping_service_account_name" {
  description = "Kubernetes ServiceAccount used by shipping-service."
  type        = string
  default     = "shipping-service"
}

variable "worker_service_account_name" {
  description = "Kubernetes ServiceAccount used by worker."
  type        = string
  default     = "worker"
}

# SQS

variable "sqs_queue_arns" {
  description = "SQS queue ARNs used by application event producers and the worker consumer."
  type        = list(string)

  validation {
    condition     = length(var.sqs_queue_arns) > 0
    error_message = "At least one SQS queue ARN must be supplied."
  }
}
<<<<<<< Updated upstream

variable "external_secrets_namespace" {
  description = "Kubernetes namespace used by External Secrets Operator."
=======
# Karpenter

variable "cluster_name" {
  description = "Name of the EKS cluster."
  type        = string
}
variable "karpenter_namespace" {
  description = "Kubernetes namespace containing the Karpenter controller."
  type        = string
  default     = "karpenter"
}

variable "external_secrets_secret_arns" {
  description = "Secrets Manager secret ARNs that External Secrets Operator is allowed to read."
  type        = list(string)
}

variable "external_dns_namespace" {
  description = "Kubernetes namespace used by ExternalDNS."
  type        = string
  default     = "external-dns"
}

variable "external_dns_service_account_name" {
  description = "Kubernetes ServiceAccount used by ExternalDNS."
  type        = string
  default     = "external-dns"
}

# Karpenter

variable "cluster_name" {
  description = "Name of the EKS cluster."
>>>>>>> Stashed changes
  type        = string
  default     = "external-secrets"
}

variable "external_secrets_service_account_name" {
  description = "Kubernetes ServiceAccount used by External Secrets Operator."
  type        = string
  default     = "external-secrets"
}

variable "external_secrets_secret_arns" {
  description = "Secrets Manager secret ARNs that External Secrets Operator is allowed to read."
  type        = list(string)
}

variable "external_dns_namespace" {
  description = "Kubernetes namespace used by ExternalDNS."
  type        = string
  default     = "external-dns"
}

variable "external_dns_service_account_name" {
  description = "Kubernetes ServiceAccount used by ExternalDNS."
  type        = string
  default     = "external-dns"
}
# VPC
variable "vpc_name" {
  description = "Name of the VPC and related networking resources"
  type        = string
}

variable "enable_nat_gateway" {
  description = "Whether to create a NAT Gateway"
  type        = bool
  default     = true
}

# RDS
variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"
}

variable "database_name" {
  description = "Initial PostgreSQL database name"
  type        = string
  default     = "bravodatabase"
}

variable "master_username" {
  description = "PostgreSQL master username"
  type        = string
  default     = "bravodbadmin"
}

variable "rds_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t4g.micro"
}

variable "rds_multi_az" {
  description = "Whether RDS should use Multi-AZ"
  type        = bool
  default     = false
}

variable "rds_deletion_protection" {
  description = "Protect RDS from accidental deletion"
  type        = bool
  default     = true
}

# SQS
variable "sqs_queue_name" {
  description = "Name of the main SQS queue"
  type        = string
}


# EKS
variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
}

variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.33"
}

variable "system_node_instance_types" {
  description = "EC2 instance types for the EKS system node group"
  type        = list(string)
  default     = ["t3.medium"]
}

variable "system_node_desired_size" {
  description = "Desired number of EKS system nodes"
  type        = number
  default     = 2
}

variable "system_node_min_size" {
  description = "Minimum number of EKS system nodes"
  type        = number
  default     = 1
}

variable "system_node_max_size" {
  description = "Maximum number of EKS system nodes"
  type        = number
  default     = 3
}

# IAM
variable "kubernetes_namespace" {
  description = "Kubernetes namespace containing application workloads"
  type        = string
  default     = "default"
}

variable "order_service_account_name" {
  description = "ServiceAccount name for order-service"
  type        = string
  default     = "order-service"
}

variable "payment_service_account_name" {
  description = "ServiceAccount name for payment-service"
  type        = string
  default     = "payment-service"
}

variable "shipping_service_account_name" {
  description = "ServiceAccount name for shipping-service"
  type        = string
  default     = "shipping-service"
}

variable "worker_service_account_name" {
  description = "ServiceAccount name for worker"
  type        = string
  default     = "worker"
}

variable "tags" {
  description = "Tags applied to infrastructure resources"
  type        = map(string)

  default = {
    Project     = "bravo-eks"
    Environment = "prod"
    ManagedBy   = "Terraform"
    Team        = "Bravo"
  }
}

# Karpenter
variable "karpenter_namespace" {
  description = "Namespace containing the Karpenter controller"
  type        = string
  default     = "karpenter"
}

variable "karpenter_service_account_name" {
  description = "ServiceAccount used by the Karpenter controller"
  type        = string
  default     = "karpenter"
}
variable "cluster_admin_principal_arns" {
  description = "IAM principals granted Kubernetes administrator access to the Production EKS cluster."
  type        = set(string)
  default     = ["arn:aws:iam::083268669572:role/aws-reserved/sso.amazonaws.com/eu-west-2/AWSReservedSSO_AdministratorAccess_e62c70aeb5c436b0"]
}

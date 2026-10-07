variable "project_name" {
  description = "Project name used for dev resource naming and tags."
  type        = string
  default     = "bravo"
}

variable "environment" {
  description = "Environment name used for resource naming and tags."
  type        = string
  default     = "dev"
}

variable "aws_region" {
  description = "AWS region for the dev environment."
  type        = string
  default     = "eu-west-2"
}

variable "enable_nat_gateway" {
  description = "Create the single NAT Gateway used by private subnets."
  type        = bool
  default     = true
}

variable "system_node_instance_types" {
  description = "EC2 instance types for the dev EKS system node group."
  type        = list(string)
  default     = ["t3.medium"]
}

variable "system_node_desired_size" {
  description = "Desired number of dev EKS system nodes."
  type        = number
  default     = 2
}

variable "system_node_min_size" {
  description = "Minimum number of dev EKS system nodes."
  type        = number
  default     = 2
}

variable "system_node_max_size" {
  description = "Maximum number of dev EKS system nodes."
  type        = number
  default     = 2
}

variable "database_name" {
  description = "Initial PostgreSQL database name."
  type        = string
  default     = "bravodev"
}

variable "database_master_username" {
  description = "PostgreSQL master username."
  type        = string
  default     = "bravoadmin"
}

variable "database_instance_class" {
  description = "RDS instance class for the dev environment."
  type        = string
  default     = "db.t4g.micro"
}

# k8s IAM
variable "kubernetes_namespace" {
  description = "Kubernetes namespace containing the dev application workloads."
  type        = string
  default     = "microservices"
}

variable "order_service_account_name" {
  description = "ServiceAccount name for order-service."
  type        = string
  default     = "order-service"
}

variable "payment_service_account_name" {
  description = "ServiceAccount name for payment-service."
  type        = string
  default     = "payment-service"
}

variable "shipping_service_account_name" {
  description = "ServiceAccount name for shipping-service."
  type        = string
  default     = "shipping-service"
}

variable "worker_service_account_name" {
  description = "ServiceAccount name for worker."
  type        = string
  default     = "worker"
}

variable "cluster_version" {
  description = "Kubernetes version for the dev EKS cluster."
  type        = string
  default     = "1.33"
}

variable "alert_email" {
  description = "Optional email address for CloudWatch alarm notifications."
  type        = string
  default     = null
  nullable    = true
}
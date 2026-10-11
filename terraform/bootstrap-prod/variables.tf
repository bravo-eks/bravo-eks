variable "terraform_state_bucket_name" {
  description = "Name of the S3 bucket that stores the PROD bootstrap Terraform state"
  type        = string
  default     = "bravo-eks-prod-tfstate"
}

variable "ecr_repositories" {
  description = "Names of the ECR repositories created for PROD application images"
  type        = list(string)
  default = [
    "api-gateway",
    "dashboard-api",
    "inventory-service",
    "notification-service",
    "order-service",
    "payment-service",
    "scheduler",
    "shipping-service",
    "support-service",
    "worker"
  ]
}

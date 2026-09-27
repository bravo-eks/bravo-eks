output "ecr_repository_arns" {
  description = "ARNs of the ECR repositories"
  value       = module.ecr.repository_arns
}

# VPC
output "vpc_id" {
  description = "ID of the VPC"
  value       = module.vpc.vpc_id
}

output "public_subnet_ids" {
  description = "IDs of the public subnets"
  value       = module.vpc.public_subnet_ids
}

output "private_subnet_ids" {
  description = "IDs of the private EKS/application subnets"
  value       = module.vpc.private_subnet_ids
}

output "database_subnet_ids" {
  description = "IDs of the private database subnets"
  value       = module.vpc.database_subnet_ids
}

output "nat_gateway_id" {
  description = "ID of the NAT Gateway"
  value       = module.vpc.nat_gateway_id
}

# SQS
output "sqs_queue_url" {
  description = "URL of the main SQS queue"
  value       = module.sqs.queue_url
}

output "sqs_queue_arn" {
  description = "ARN of the main SQS queue"
  value       = module.sqs.queue_arn
}

output "sqs_dlq_url" {
  description = "URL of the SQS dead-letter queue"
  value       = module.sqs.dlq_url
}

output "sqs_dlq_arn" {
  description = "ARN of the SQS dead-letter queue"
  value       = module.sqs.dlq_arn
}

# EKS
output "eks_cluster_name" {
  description = "Name of the EKS cluster"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "Endpoint of the EKS cluster"
  value       = module.eks.cluster_endpoint
}

output "eks_oidc_provider_arn" {
  description = "ARN of the EKS OIDC provider"
  value       = module.eks.eks_oidc_provider_arn
}

# RDS
output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint"
  value       = module.rds.db_endpoint
}

output "rds_port" {
  description = "RDS PostgreSQL port"
  value       = module.rds.db_port
}

output "rds_master_user_secret_arn" {
  description = "ARN of the RDS master credentials secret"
  value       = module.rds.master_user_secret_arn
}

# IAM
output "order_service_role_arn" {
  description = "IAM role ARN for order-service"
  value       = module.iam.order_service_role_arn
}

output "payment_service_role_arn" {
  description = "IAM role ARN for payment-service"
  value       = module.iam.payment_service_role_arn
}

output "shipping_service_role_arn" {
  description = "IAM role ARN for shipping-service"
  value       = module.iam.shipping_service_role_arn
}

output "worker_role_arn" {
  description = "IAM role ARN for worker"
  value       = module.iam.worker_role_arn
}
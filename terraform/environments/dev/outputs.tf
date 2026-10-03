output "sqs_queue_url" {
  description = "URL of the dev application event queue."
  value       = module.sqs.queue_url
}

output "sqs_queue_arn" {
  description = "ARN of the dev application event queue."
  value       = module.sqs.queue_arn
}

output "sqs_dlq_url" {
  description = "URL of the dev application dead-letter queue."
  value       = module.sqs.dlq_url
}

output "vpc_id" {
  description = "ID of the dev VPC."
  value       = module.vpc.vpc_id
}

output "eks_cluster_name" {
  description = "Name of the dev EKS cluster."
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "Endpoint of the dev EKS cluster."
  value       = module.eks.cluster_endpoint
}

output "rds_endpoint" {
  description = "Endpoint of the dev PostgreSQL instance."
  value       = module.rds.db_endpoint
}

output "ecr_repository_urls" {
  description = "URLs of the dev ECR repositories."
  value       = module.ecr.repository_urls
}

output "ecr_repository_arns" {
  description = "ARNs of the dev ECR repositories."
  value       = module.ecr.repository_arns
}

# EKS / Kubernetes

output "eks_cluster_ca_certificate" {
  description = "CA certificate for the dev EKS cluster."
  value       = module.eks.cluster_ca_certificate
  sensitive   = true
}

output "eks_oidc_issuer_url" {
  description = "OIDC issuer URL for the dev EKS cluster."
  value       = module.eks.cluster_oidc_issuer_url
}

output "eks_node_role_arn" {
  description = "IAM role ARN used by dev EKS worker nodes."
  value       = module.eks.node_role_arn
}

output "eks_node_role_name" {
  description = "IAM role name used by dev EKS worker nodes."
  value       = module.eks.node_role_name
}

# IAM

output "order_service_role_arn" {
  description = "IAM role ARN for the dev order-service."
  value       = module.iam.order_service_role_arn
}

output "payment_service_role_arn" {
  description = "IAM role ARN for the dev payment-service."
  value       = module.iam.payment_service_role_arn
}

output "shipping_service_role_arn" {
  description = "IAM role ARN for the dev shipping-service."
  value       = module.iam.shipping_service_role_arn
}

output "worker_role_arn" {
  description = "IAM role ARN for the dev worker."
  value       = module.iam.worker_role_arn
}

output "karpenter_controller_role_arn" {
  description = "IAM role ARN for the dev Karpenter controller."
  value       = module.iam.karpenter_controller_role_arn
}

# External Secrets Operator

output "external_secrets_role_arn" {
  description = "ARN of the IAM role used by External Secrets Operator in development."
  value       = module.iam.external_secrets_role_arn
}

# ExternalDNS

output "external_dns_role_arn" {
  description = "ARN of the IAM role used by ExternalDNS in development."
  value       = module.iam.external_dns_role_arn
}
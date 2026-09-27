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
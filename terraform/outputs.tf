# ECR
output "ecr_repository_urls" {
  description = "URLs of the ECR repositories"
  value       = module.ecr.repository_urls
}

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

# RDS
# output "rds_endpoint" {
#   description = "RDS PostgreSQL connection endpoint"
#   value       = module.rds.db_endpoint
# }

# output "rds_address" {
#   description = "RDS PostgreSQL hostname"
#   value       = module.rds.db_address
# }

# output "rds_port" {
#   description = "RDS PostgreSQL port"
#   value       = module.rds.db_port
# }

# output "rds_security_group_id" {
#   description = "Security group ID attached to RDS"
#   value       = module.rds.rds_security_group_id
# }

# output "rds_master_user_secret_arn" {
#   description = "ARN of the Secrets Manager secret containing RDS credentials"
#   value       = module.rds.master_user_secret_arn
# }

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
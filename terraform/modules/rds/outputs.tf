output "db_instance_id" {
  description = "RDS PostgreSQL instance identifier"
  value       = aws_db_instance.postgres.id
}

output "db_endpoint" {
  description = "RDS PostgreSQL connection endpoint"
  value       = aws_db_instance.postgres.endpoint
}

output "db_address" {
  description = "RDS PostgreSQL hostname"
  value       = aws_db_instance.postgres.address
}

output "db_port" {
  description = "RDS PostgreSQL port"
  value       = aws_db_instance.postgres.port
}

output "db_name" {
  description = "Initial PostgreSQL database name"
  value       = aws_db_instance.postgres.db_name
}

output "rds_security_group_id" {
  description = "Security group ID attached to RDS"
  value       = aws_security_group.rds.id
}

output "master_user_secret_arn" {
  description = "ARN of the Secrets Manager secret containing the RDS master credentials"
  value       = aws_db_instance.postgres.master_user_secret[0].secret_arn
}
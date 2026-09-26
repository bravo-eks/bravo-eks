# ECR
variable "aws_region" {
  description = "AWS region to deploy resources into"
  type        = string
  default     = "eu-west-2"
}

variable "ecr_repositories" {
  description = "List of ECR repositories to create"
  type        = list(string)
}

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
  default     = "bravo-database"
}

variable "master_username" {
  description = "PostgreSQL master username"
  type        = string
  default     = "bravo-dbadmin"
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


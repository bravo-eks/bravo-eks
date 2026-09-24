variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"
}

variable "vpc_id" {
  description = "VPC ID where RDS will be deployed"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs used by the RDS subnet group"
  type        = list(string)
}

variable "eks_security_group_id" {
  description = "EKS security group allowed to connect to PostgreSQL"
  type        = string
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

variable "engine_version" {
  description = "PostgreSQL engine version"
  type        = string
  default     = "17"
}

variable "instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t4g.micro"
}

variable "allocated_storage" {
  description = "Initial allocated storage in GB"
  type        = number
  default     = 20
}

variable "max_allocated_storage" {
  description = "Maximum storage RDS can automatically scale to in GB"
  type        = number
  default     = 100
}

variable "backup_retention_period" {
  description = "Number of days automated backups are retained"
  type        = number
  default     = 7
}

variable "backup_window" {
  description = "Daily UTC backup window"
  type        = string
  default     = "02:00-03:00"
}

variable "maintenance_window" {
  description = "Weekly UTC maintenance window"
  type        = string
  default     = "sun:03:00-sun:04:00"
}

variable "multi_az" {
  description = "Whether RDS should use a Multi-AZ deployment"
  type        = bool
  default     = false
}

variable "deletion_protection" {
  description = "Protect the RDS instance from accidental deletion"
  type        = bool
  default     = true
}
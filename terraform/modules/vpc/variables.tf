variable "name" {
  description = "Name of the VPC and related networking resources"
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "Availability zones used by the VPC"
  type        = list(string)

  default = [
    "eu-west-2a",
    "eu-west-2b"
  ]

  validation {
    condition     = length(var.availability_zones) == 2
    error_message = "Exactly 2 availability zones must be provided."
  }
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for public subnets"
  type        = list(string)

  default = [
    "10.0.1.0/24",
    "10.0.2.0/24"
  ]

  validation {
    condition     = length(var.public_subnet_cidrs) == 2
    error_message = "Exactly 2 public subnet CIDRs must be provided."
  }
}

variable "private_subnet_cidrs" {
  description = "CIDR blocks for private EKS/application subnets"
  type        = list(string)

  default = [
    "10.0.11.0/24",
    "10.0.12.0/24"
  ]

  validation {
    condition     = length(var.private_subnet_cidrs) == 2
    error_message = "Exactly 2 private subnet CIDRs must be provided."
  }
}

variable "database_subnet_cidrs" {
  description = "CIDR blocks for private database subnets"
  type        = list(string)

  default = [
    "10.0.21.0/24",
    "10.0.22.0/24"
  ]

  validation {
    condition     = length(var.database_subnet_cidrs) == 2
    error_message = "Exactly 2 database subnet CIDRs must be provided."
  }
}

variable "enable_nat_gateway" {
  description = "Whether to create one NAT Gateway"
  type        = bool
  default     = true
}
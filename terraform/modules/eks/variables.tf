variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
}

variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.33"
}

variable "vpc_id" {
  description = "VPC ID for the EKS cluster"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for the EKS cluster"
  type        = list(string)
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "system_node_instance_types" {
  description = "EC2 instance types for the system node group"
  type        = list(string)
  default     = ["t3.medium"]
}

variable "system_node_desired_size" {
  description = "Desired number of system nodes"
  type        = number
  default     = 2
}

variable "system_node_min_size" {
  description = "Minimum number of system nodes"
  type        = number
  default     = 1
}

variable "system_node_max_size" {
  description = "Maximum number of system nodes"
  type        = number
  default     = 3
}

variable "tags" {
  description = "Tags to apply to EKS resources"
  type        = map(string)
  default     = {}
}

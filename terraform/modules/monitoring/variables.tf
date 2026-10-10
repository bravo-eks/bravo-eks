variable "project_name" {
  description = "Project name used for monitoring resource names."
  type        = string
}

variable "environment" {
  description = "Environment being monitored."
  type        = string
}

variable "rds_instance_identifier" {
  description = "RDS instance identifier to monitor."
  type        = string
}

variable "sqs_queue_name" {
  description = "Main SQS queue name to monitor."
  type        = string
}

variable "sqs_dlq_name" {
  description = "SQS dead-letter queue name to monitor."
  type        = string
}

variable "rds_cpu_threshold" {
  description = "RDS CPU utilisation percentage that triggers an alarm."
  type        = number
  default     = 80
}

variable "rds_free_storage_threshold_bytes" {
  description = "Minimum RDS free storage in bytes before triggering an alarm."
  type        = number
  default     = 2147483648
}

variable "sqs_queue_depth_threshold" {
  description = "Maximum expected number of visible SQS messages."
  type        = number
  default     = 100
}

variable "tags" {
  description = "Tags applied to monitoring resources."
  type        = map(string)
  default     = {}
}
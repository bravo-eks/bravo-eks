variable "queue_name" {
  description = "Name of the SQS queue"
  type        = string
}

variable "visibility_timeout_seconds" {
  description = "How long a received message stays hidden from other consumers"
  type        = number
  default     = 30
}

# 4 days
variable "message_retention_seconds" {
  description = "How long messages are kept in the main queue"
  type        = number
  default     = 345600
}

variable "max_receive_count" {
  description = "Number of failed receives before a message is moved to the DLQ"
  type        = number
  default     = 5
}

# 14 days
variable "dlq_message_retention_seconds" {
  description = "How long failed messages are kept in the dead-letter queue"
  type        = number
  default     = 1209600
}
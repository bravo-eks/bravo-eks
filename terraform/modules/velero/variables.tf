variable "bucket_name" {
  description = "Name of the S3 bucket used for Velero backups."
  type        = string
}

variable "tags" {
  description = "Tags applied to the Velero backup bucket."
  type        = map(string)
  default     = {}
}
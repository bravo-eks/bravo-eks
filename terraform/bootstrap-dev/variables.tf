variable "aws_region" {
  description = "The AWS region to deploy resources"
  type        = string
  default     = "eu-west-2"
}

variable "s3_name" {
  description = "The name of the S3 bucket for Terraform state"
  type        = string
  default     = "bravo-eksv2-project-s3"
}

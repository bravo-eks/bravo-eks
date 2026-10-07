variable "ses_email_identity" {
  description = "Email address to verify with Amazon SES. Set to null when SES identity management is handled outside Terraform."
  type        = string
  default     = null
  nullable    = true
}

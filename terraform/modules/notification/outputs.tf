output "ses_email_identity_arn" {
  description = "ARN of the SES email identity used by the notification service."
  value       = var.ses_email_identity == null ? null : aws_sesv2_email_identity.notification[0].arn
}

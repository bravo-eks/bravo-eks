resource "aws_sesv2_email_identity" "notification" {
  count = var.ses_email_identity == null ? 0 : 1

  email_identity = var.ses_email_identity
}

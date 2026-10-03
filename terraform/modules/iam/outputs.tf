# Policies

output "sqs_producer_policy_arn" {
  description = "ARN of the SQS producer policy."
  value       = aws_iam_policy.sqs_producer.arn
}

output "worker_sqs_policy_arn" {
  description = "ARN of the worker SQS consumer policy."
  value       = aws_iam_policy.worker_sqs.arn
}

# Order Service

output "order_service_role_arn" {
  description = "ARN of the IAM role assumed by the order-service Kubernetes ServiceAccount."
  value       = aws_iam_role.order_service.arn
}

output "order_service_role_name" {
  description = "Name of the order-service IAM role."
  value       = aws_iam_role.order_service.name
}

# Payment Service

output "payment_service_role_arn" {
  description = "ARN of the IAM role assumed by the payment-service Kubernetes ServiceAccount."
  value       = aws_iam_role.payment_service.arn
}

output "payment_service_role_name" {
  description = "Name of the payment-service IAM role."
  value       = aws_iam_role.payment_service.name
}

# Shipping Service

output "shipping_service_role_arn" {
  description = "ARN of the IAM role assumed by the shipping-service Kubernetes ServiceAccount."
  value       = aws_iam_role.shipping_service.arn
}

output "shipping_service_role_name" {
  description = "Name of the shipping-service IAM role."
  value       = aws_iam_role.shipping_service.name
}

# Worker

output "worker_role_arn" {
  description = "ARN of the IAM role assumed by the worker Kubernetes ServiceAccount."
  value       = aws_iam_role.worker.arn
}

output "worker_role_name" {
  description = "Name of the worker IAM role."
  value       = aws_iam_role.worker.name
}

# Policies

output "external_secrets_role_arn" {
  description = "ARN of the IAM role used by External Secrets Operator."
  value       = aws_iam_role.external_secrets.arn
}

output "external_dns_role_arn" {
  description = "ARN of the IAM role used by ExternalDNS."
  value       = aws_iam_role.external_dns.arn
}

# Karpenter

output "karpenter_controller_role_arn" {
  description = "ARN of the IAM role assumed by the Karpenter controller."
  value       = aws_iam_role.karpenter_controller.arn
}

output "karpenter_controller_role_name" {
  description = "Name of the Karpenter controller IAM role."
  value       = aws_iam_role.karpenter_controller.name
}
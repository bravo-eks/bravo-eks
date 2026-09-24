# Worker IAM Role

output "worker_role_arn" {
  description = "ARN of the IAM role used by the worker Kubernetes service account."
  value       = aws_iam_role.worker.arn
}


output "worker_role_name" {
  description = "Name of the IAM role used by the worker Kubernetes service account."
  value       = aws_iam_role.worker.name
}

# Worker SQS Policy

output "worker_sqs_policy_arn" {
  description = "ARN of the IAM policy granting the worker access to required SQS queues."
  value       = aws_iam_policy.worker_sqs.arn
}

# Secrets Reader IAM Role

output "secrets_reader_role_arn" {
  description = "ARN of the IAM role used by the secrets-reader Kubernetes service account."
  value       = aws_iam_role.secrets_reader.arn
}


output "secrets_reader_role_name" {
  description = "Name of the IAM role used by the secrets-reader Kubernetes service account."
  value       = aws_iam_role.secrets_reader.name
}

# Secrets Reader Policy

output "secrets_reader_policy_arn" {
  description = "ARN of the IAM policy granting access to required Secrets Manager secrets."
  value       = aws_iam_policy.secrets_reader.arn
}
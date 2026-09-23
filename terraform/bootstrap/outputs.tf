output "terraform_state_bucket_name" {
  description = "Name of the Terraform state bucket"
  value       = aws_s3_bucket.terraform_state.id
}

output "terraform_state_bucket_arn" {
  description = "ARN of the Terraform state bucket"
  value       = aws_s3_bucket.terraform_state.arn
}

output "ecr_repository_urls" {
  description = "URLs of the ECR repositories"
  value       = module.ecr.repository_urls
}

output "ecr_repository_arns" {
  description = "ARNs of the ECR repositories"
  value       = module.ecr.repository_arns
}

output "github_actions_role_arn" {
  description = "ARN of the IAM role used by GitHub Actions"
  value       = aws_iam_role.github_actions.arn
}
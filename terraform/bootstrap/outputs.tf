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
  description = "ARN of the production GitHub Actions role"
  value       = aws_iam_role.github_actions.arn
}

output "github_terraform_plan_role_arn" {
  description = "ARN of the GitHub Actions Terraform PR plan role"
  value       = aws_iam_role.github_actions_terraform_plan.arn
}
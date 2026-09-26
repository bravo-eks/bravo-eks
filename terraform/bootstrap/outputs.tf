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

output "github_actions_ecr_arn" {
  description = "ARN of the production GitHub Actions ECR role"
  value       = aws_iam_role.github_actions_ecr.arn
}

output "github_terraform_plan_pr_arn" {
  description = "ARN of the GitHub Actions Terraform PR plan role"
  value       = aws_iam_role.github_actions_terraform_plan.arn
}

output "github_terraform_production_arn" {
  description = "ARN of the GitHub Actions Terraform production role"
  value       = aws_iam_role.github_actions_terraform_production.arn
}
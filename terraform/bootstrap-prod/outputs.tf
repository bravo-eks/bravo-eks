output "terraform_state_bucket_name" {
  value = aws_s3_bucket.terraform_state.id
}

output "terraform_state_bucket_arn" {
  value = aws_s3_bucket.terraform_state.arn
}

output "ecr_repository_urls" {
  value = module.ecr.repository_urls
}

output "ecr_repository_arns" {
  value = module.ecr.repository_arns
}

output "github_ecr_role_arn" {
  value = aws_iam_role.github_ecr.arn
}

output "github_terraform_pr_role_arn" {
  value = aws_iam_role.github_terraform_pr.arn
}

output "github_terraform_deploy_role_arn" {
  value = aws_iam_role.github_terraform_deploy.arn
}

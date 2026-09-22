output "repository_urls" {
  description = "URLs of the ECR repositories"
  value = {
    for name, repository in aws_ecr_repository.this :
    name => repository.repository_url
  }
}

output "repository_arns" {
  description = "ARNs of the ECR repositories"
  value = {
    for name, repository in aws_ecr_repository.this :
    name => repository.arn
  }
}
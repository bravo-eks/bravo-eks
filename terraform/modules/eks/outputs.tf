output "cluster_name" {
  description = "Name of the EKS cluster"
  value       = aws_eks_cluster.this.name
}

output "cluster_endpoint" {
  description = "Endpoint of the EKS cluster"
  value       = aws_eks_cluster.this.endpoint
}

output "cluster_ca_certificate" {
  description = "Base64 encoded cluster CA certificate"
  value       = aws_eks_cluster.this.certificate_authority[0].data
}

output "cluster_oidc_issuer_url" {
  description = "OIDC issuer URL for the EKS cluster"
  value       = aws_eks_cluster.this.identity[0].oidc[0].issuer
}

output "eks_oidc_provider_arn" {
  description = "ARN of the IAM OIDC provider associated with the EKS cluster."
  value       = aws_iam_openid_connect_provider.this.arn
}

output "eks_oidc_provider_url" {
  description = "EKS OIDC issuer URL without the https:// prefix."
  value       = replace(aws_iam_openid_connect_provider.this.url, "https://", "")
}
output "bucket_name" {
  description = "Velero backup bucket name."
  value       = aws_s3_bucket.velero.bucket
}

output "bucket_arn" {
  description = "Velero backup bucket ARN."
  value       = aws_s3_bucket.velero.arn
}
resource "aws_s3_bucket" "velero" {
  bucket = var.bucket_name

  tags = merge(
    var.tags,
    {
      Name      = var.bucket_name
      Component = "velero"
    }
  )
}

resource "aws_s3_bucket_public_access_block" "velero" {
  bucket = aws_s3_bucket.velero.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "velero" {
  bucket = aws_s3_bucket.velero.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "velero" {
  bucket = aws_s3_bucket.velero.id

  rule {
    blocked_encryption_types = ["SSE-C"]

    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}
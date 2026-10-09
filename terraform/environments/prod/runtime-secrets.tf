resource "random_password" "jwt_secret" {
  length  = 64
  special = false
}

resource "random_password" "redis_password" {
  length  = 32
  special = false
}

resource "aws_secretsmanager_secret" "runtime" {
  name        = "${var.project_name}-${var.environment}-runtime"
  description = "Runtime credentials for Bravo production workloads."
  tags        = var.tags
}

resource "aws_secretsmanager_secret_version" "runtime" {
  secret_id = aws_secretsmanager_secret.runtime.id
  secret_string = jsonencode({
    jwt_secret     = random_password.jwt_secret.result
    redis_password = random_password.redis_password.result
  })
}

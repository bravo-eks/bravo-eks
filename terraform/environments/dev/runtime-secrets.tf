resource "random_password" "jwt_secret" {
  length  = 64
  special = false
}

resource "random_password" "redis_password" {
  length  = 32
  special = false
}

resource "aws_secretsmanager_secret" "runtime" {
  name        = "${local.name_prefix}-runtime"
  description = "Runtime credentials for Bravo dev workloads."

  tags = local.tags
}

resource "aws_secretsmanager_secret_version" "runtime" {
  secret_id = aws_secretsmanager_secret.runtime.id

  secret_string = jsonencode({
    jwt_secret     = random_password.jwt_secret.result
    redis_password = random_password.redis_password.result
  })
}

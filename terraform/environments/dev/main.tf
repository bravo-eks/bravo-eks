
data "aws_s3_bucket" "support_email" {
  bucket = "bravo-dev-support-email"
}

locals {
  name_prefix = "${var.project_name}-${var.environment}"

  tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}

## SQS
module "sqs" {
  source = "../../modules/sqs"

  queue_name = "${local.name_prefix}-events"
}


module "support_sqs" {
  source = "../../modules/sqs"

  queue_name                    = "${local.name_prefix}-support-inbound"
  visibility_timeout_seconds    = 60
  message_retention_seconds     = 345600
  max_receive_count             = 5
  dlq_message_retention_seconds = 1209600
}

## VPC
module "vpc" {
  source = "../../modules/vpc"

  name               = local.name_prefix
  enable_nat_gateway = var.enable_nat_gateway
}

## EKS
module "eks" {
  source = "../../modules/eks"

  cluster_name               = local.name_prefix
  cluster_version            = var.cluster_version
  vpc_id                     = module.vpc.vpc_id
  private_subnet_ids         = module.vpc.private_subnet_ids
  environment                = var.environment
  system_node_instance_types = var.system_node_instance_types
  system_node_desired_size   = var.system_node_desired_size
  system_node_min_size       = var.system_node_min_size
  system_node_max_size       = var.system_node_max_size
  tags                       = local.tags
}

## RDS PostgreSQL
module "rds" {
  source = "../../modules/rds"

  project_name          = var.project_name
  environment           = var.environment
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.database_subnet_ids
  eks_security_group_id = module.eks.cluster_security_group_id

  database_name   = var.database_name
  master_username = var.database_master_username
  instance_class  = var.database_instance_class

  # Cheaper/disposable dev settings
  multi_az                = false
  deletion_protection     = false
  backup_retention_period = 1
}

## IAM
module "iam" {
  source = "../../modules/iam"

  name_prefix = local.name_prefix
  tags        = local.tags

  eks_oidc_provider_arn = module.eks.eks_oidc_provider_arn
  eks_oidc_provider_url = module.eks.eks_oidc_provider_url

  cluster_name            = module.eks.cluster_name
  karpenter_node_role_arn = module.eks.node_role_arn

  kubernetes_namespace                = var.kubernetes_namespace
  order_service_account_name          = var.order_service_account_name
  payment_service_account_name        = var.payment_service_account_name
  shipping_service_account_name       = var.shipping_service_account_name
  worker_service_account_name         = var.worker_service_account_name
  notification_service_account_name   = var.notification_service_account_name
  notification_ses_email_identity_arn = "arn:aws:ses:eu-west-2:304960797322:identity/bravoorders.com"

  support_service_account_name = var.support_service_account_name
  support_sqs_queue_arn        = module.support_sqs.queue_arn
  support_s3_bucket_arn        = data.aws_s3_bucket.support_email.arn

  sqs_queue_arns = [module.sqs.queue_arn]

  external_secrets_secret_arns = [
    module.rds.master_user_secret_arn,
    aws_secretsmanager_secret.runtime.arn,
    "arn:aws:secretsmanager:eu-west-2:304960797322:secret:bravo/dev/*/database-*"
  ]
  velero_backup_bucket_arn = module.velero.bucket_arn
}

module "velero" {
  source = "../../modules/velero"

  bucket_name = "bravo-eksv2-velero-backups"

  tags = {
    Environment = "dev"
    Project     = "bravo-eksv2"
  }
}

# Monitoring
module "monitoring" {
  source = "../../modules/monitoring"

  project_name = var.project_name
  environment  = var.environment

  rds_instance_identifier = module.rds.db_instance_id

  sqs_queue_name = "${local.name_prefix}-events"
  sqs_dlq_name   = "${local.name_prefix}-events-dlq"

  alert_email = var.alert_email

  tags = local.tags
}


data "aws_iam_policy_document" "support_s3_to_sqs" {
  statement {
    sid    = "AllowSupportS3ToSendMessage"
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["s3.amazonaws.com"]
    }

    actions   = ["sqs:SendMessage"]
    resources = [module.support_sqs.queue_arn]

    condition {
      test     = "ArnEquals"
      variable = "aws:SourceArn"
      values   = [data.aws_s3_bucket.support_email.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "aws:SourceAccount"
      values   = ["304960797322"]
    }
  }
}

resource "aws_sqs_queue_policy" "support_s3" {
  queue_url = module.support_sqs.queue_url
  policy    = data.aws_iam_policy_document.support_s3_to_sqs.json
}


resource "aws_s3_bucket_notification" "support_email" {
  bucket = data.aws_s3_bucket.support_email.id

  queue {
    queue_arn     = module.support_sqs.queue_arn
    events        = ["s3:ObjectCreated:*"]
    filter_prefix = "incoming/"
  }

  depends_on = [
    aws_sqs_queue_policy.support_s3
  ]
}

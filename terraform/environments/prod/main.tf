# VPC
module "vpc" {
  source = "../../modules/vpc"

  name               = var.vpc_name
  enable_nat_gateway = var.enable_nat_gateway
}

# SQS
module "sqs" {
  source = "../../modules/sqs"

  queue_name = var.sqs_queue_name
}

module "support_sqs" {
  source = "../../modules/sqs"

  queue_name                    = "${var.project_name}-${var.environment}-support-inbound"
  visibility_timeout_seconds    = 60
  message_retention_seconds     = 345600
  max_receive_count             = 5
  dlq_message_retention_seconds = 1209600
}

data "aws_s3_bucket" "support_email" {
  bucket = var.support_email_bucket_name
}

data "aws_caller_identity" "current" {}

# EKS
module "eks" {
  source = "../../modules/eks"

  cluster_name    = var.cluster_name
  cluster_version = var.cluster_version
  environment     = var.environment

  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids

  system_node_instance_types = var.system_node_instance_types
  system_node_desired_size   = var.system_node_desired_size
  system_node_min_size       = var.system_node_min_size
  system_node_max_size       = var.system_node_max_size

  tags = var.tags
}

# IAM
module "iam" {
  source = "../../modules/iam"

  name_prefix = "${var.project_name}-${var.environment}"

  eks_oidc_provider_arn = module.eks.eks_oidc_provider_arn
  eks_oidc_provider_url = module.eks.eks_oidc_provider_url

  cluster_name = module.eks.cluster_name

  kubernetes_namespace              = var.kubernetes_namespace
  order_service_account_name        = var.order_service_account_name
  payment_service_account_name      = var.payment_service_account_name
  shipping_service_account_name     = var.shipping_service_account_name
  worker_service_account_name       = var.worker_service_account_name
  notification_service_account_name = var.notification_service_account_name
  support_service_account_name      = var.support_service_account_name
  karpenter_namespace               = var.karpenter_namespace
  karpenter_service_account_name    = var.karpenter_service_account_name
  karpenter_node_role_arn           = module.eks.node_role_arn

  notification_ses_email_identity_arn = var.notification_ses_email_identity_arn
  support_sqs_queue_arn               = module.support_sqs.queue_arn
  support_s3_bucket_arn               = data.aws_s3_bucket.support_email.arn

  sqs_queue_arns = [
    module.sqs.queue_arn
  ]

  external_secrets_secret_arns = [module.rds.master_user_secret_arn]

  velero_backup_bucket_arn = module.velero.bucket_arn
  tags                     = var.tags
}

# RDS
module "rds" {
  source = "../../modules/rds"

  project_name = var.project_name
  environment  = var.environment

  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.database_subnet_ids
  eks_security_group_id = module.eks.cluster_security_group_id
  database_name         = var.database_name
  master_username       = var.master_username
  instance_class        = var.rds_instance_class
  multi_az              = var.rds_multi_az
  deletion_protection   = var.rds_deletion_protection
}

module "velero" {
  source = "../../modules/velero"

  bucket_name = "bravo-eks-prod-velero-backups"

  tags = var.tags
}

module "monitoring" {
  source = "../../modules/monitoring"

  project_name = var.project_name
  environment  = var.environment

  rds_instance_identifier = module.rds.db_instance_id

  sqs_queue_name = var.sqs_queue_name
  sqs_dlq_name   = "${var.sqs_queue_name}-dlq"

  alert_email = var.alert_email

  tags = var.tags
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
      values   = [data.aws_caller_identity.current.account_id]
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
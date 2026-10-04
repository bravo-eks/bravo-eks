locals {
  name_prefix = "${var.project_name}-${var.environment}"

  tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
  }

  services = [
    "api-gateway",
    "order-service",
    "inventory-service",
    "payment-service",
    "notification-service",
    "shipping-service",
    "worker",
    "scheduler",
    "dashboard-api"
  ]
}

# ECR repositories
module "ecr" {
  source = "../../modules/ecr"

  ecr_repositories = [
    for service in local.services :
    "${local.name_prefix}-${service}"
  ]
}

# SQS
module "sqs" {
  source = "../../modules/sqs"

  queue_name = "${local.name_prefix}-events"
}

# VPC
module "vpc" {
  source = "../../modules/vpc"

  name               = local.name_prefix
  enable_nat_gateway = var.enable_nat_gateway
}

# EKS
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

# RDS PostgreSQL
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

# IAM
module "iam" {
  source = "../../modules/iam"

  name_prefix = local.name_prefix
  tags        = local.tags

  eks_oidc_provider_arn = module.eks.eks_oidc_provider_arn
  eks_oidc_provider_url = module.eks.eks_oidc_provider_url

  cluster_name            = module.eks.cluster_name
  karpenter_node_role_arn = module.eks.node_role_arn

  kubernetes_namespace          = var.kubernetes_namespace
  order_service_account_name    = var.order_service_account_name
  payment_service_account_name  = var.payment_service_account_name
  shipping_service_account_name = var.shipping_service_account_name
  worker_service_account_name   = var.worker_service_account_name

  sqs_queue_arns = [module.sqs.queue_arn]

  external_secrets_secret_arns = [
    module.rds.master_user_secret_arn,
    aws_secretsmanager_secret.runtime.arn
  ]
}
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

  kubernetes_namespace          = var.kubernetes_namespace
  order_service_account_name    = var.order_service_account_name
  payment_service_account_name  = var.payment_service_account_name
  shipping_service_account_name = var.shipping_service_account_name
  worker_service_account_name   = var.worker_service_account_name

  sqs_queue_arns = [
    module.sqs.queue_arn
  ]

  tags = var.tags
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
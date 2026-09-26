#ECR
module "ecr" {
  source           = "./modules/ecr"
  ecr_repositories = var.ecr_repositories
}

# VPC
module "vpc" {
  source = "./modules/vpc"

  name               = var.vpc_name
  enable_nat_gateway = var.enable_nat_gateway
}

# RDS
# module "rds" {
# source = "./modules/rds"

# project_name = var.project_name
# environment  = var.environment


# Networking from VPC
# vpc_id             = module.vpc.vpc_id
# private_subnet_ids = module.vpc.database_subnet_ids

# Security from EKS (CHECK AFTER MERGE)
#eks_security_group_id = module.eks.eks_security_group_id

# database_name       = var.database_name
# master_username     = var.master_username
# instance_class      = var.rds_instance_class
# multi_az            = var.rds_multi_az
# deletion_protection = var.rds_deletion_protection
#}

# SQS
module "sqs" {
  source = "./modules/sqs"

  queue_name = var.sqs_queue_name
}

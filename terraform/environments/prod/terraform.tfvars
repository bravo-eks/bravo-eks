# VPC
vpc_name           = "bravo-eks-prod"
enable_nat_gateway = true

# RDS
project_name = "bravo-eks"
environment  = "prod"

database_name   = "bravoprod"
master_username = "bravo-dbadmin"

rds_instance_class      = "db.t4g.micro"
rds_multi_az            = false
rds_deletion_protection = true

# SQS
sqs_queue_name = "bravo-eks-prod-events"

# EKS
cluster_name    = "bravo-eks-prod"
cluster_version = "1.33"

system_node_instance_types = ["t3.medium"]
system_node_desired_size   = 2
system_node_min_size       = 1
system_node_max_size       = 3

# IAM
kubernetes_namespace          = "default"
order_service_account_name    = "order-service"
payment_service_account_name  = "payment-service"
shipping_service_account_name = "shipping-service"
worker_service_account_name   = "worker"
# Replace with the hosted zone ID created in the NEW AWS account.
external_dns_hosted_zone_id = "REPLACE_WITH_NEW_PROD_HOSTED_ZONE_ID"

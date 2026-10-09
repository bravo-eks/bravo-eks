terraform {
  required_version = ">= 1.10.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.27.0"
    }
  }
}

provider "aws" {
  region = "eu-west-2"

  allowed_account_ids = ["083268669572"]

  default_tags {
    tags = {
      Project     = "bravo-eks"
      Environment = "prod"
      ManagedBy   = "Terraform"
      Team        = "Bravo"
    }
  }
}

terraform {
  backend "s3" {
    bucket       = "bravo-eks-prod-tfstate"
    key          = "environments/prod/terraform.tfstate"
    region       = "eu-west-2"
    encrypt      = true
    use_lockfile = true
  }
}
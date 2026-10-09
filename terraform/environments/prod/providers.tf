terraform {
  required_version = ">= 1.10.0"

  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "6.27.0"
    }
  }
}

provider "aws" {
  region = "eu-west-2"

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
    bucket       = "REPLACE_WITH_NEW_ACCOUNT_PROD_STATE_BUCKET"
    key          = "environments/prod/terraform.tfstate"
    region       = "eu-west-2"
    encrypt      = true
    use_lockfile = true
  }
}
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
    bucket       = "bravo-eksv2-project-s3"
    key          = "environments/prod/terraform.tfstate"
    region       = "eu-west-2"
    encrypt      = true
    use_lockfile = true
  }
}
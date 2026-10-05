terraform {
  required_version = ">= 1.10.0"

  backend "s3" {
    bucket       = "bravo-eksv2-project-s3"
    key          = "bootstrap/terraform.tfstate"
    region       = "eu-west-2"
    encrypt      = true
    use_lockfile = true
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.27.0"
    }
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project   = "bravo-eks"
      ManagedBy = "Terraform"
      Team      = "Bravo"
    }
  }
}
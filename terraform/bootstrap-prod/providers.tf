terraform {
  required_version = ">= 1.10.0"

  # First bootstrap uses local state. Never point this root at Account A.
  backend "local" {}

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
    }
  }
}

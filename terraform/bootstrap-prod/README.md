# Account B production Terraform backend bootstrap

This is a **new, independent Terraform root** for AWS Account B (`083268669572`).
It creates only the production state bucket. It does not modify the existing
`terraform/bootstrap` root, Account A, or Dev infrastructure.

## First run (PowerShell, clean checkout)

Ensure `aws sts get-caller-identity` reports account `083268669572`.
From `terraform/bootstrap-prod` run:

```powershell
aws sts get-caller-identity
terraform init
terraform plan -out=bootstrap.tfplan
terraform show bootstrap.tfplan
terraform apply bootstrap.tfplan
```

Review the plan before applying. It should create only the bucket and its
versioning, encryption, public-access-block, and ownership-control resources.
Do not proceed if the plan includes deletions or unexpected changes.
The bucket name is globally unique; if it is unavailable, choose another
unique name and update both the resource and subsequent backend settings.

## Migrate only this newly created Account B state

After the bucket exists, change `providers.tf` from `backend "local" {}` to:

```hcl
backend "s3" {
  bucket       = "bravo-eks-prod-tfstate"
  key          = "bootstrap-prod/terraform.tfstate"
  region       = "eu-west-2"
  encrypt      = true
  use_lockfile = true
}
```

Then run `terraform init -migrate-state` and `terraform state list`.
Keep a secure local state backup until verified. Commit the remote backend
configuration to the production branch after migration, and configure the
production infrastructure root to use the same bucket but its own state key
(`environments/prod/terraform.tfstate`).

Never migrate or import Account A state. Do not commit Terraform state,
plan files, or AWS credentials.

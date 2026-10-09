# BRAVO cross-account production deployment

This branch is **preparation only**. Do not merge or deploy until the remaining items are completed and reviewed. The existing Dev account must remain unchanged.

## Account prerequisites
- Create a dedicated S3 Terraform state bucket in the new account, with versioning, encryption, restricted access, and state locking.
- Replace `REPLACE_WITH_NEW_ACCOUNT_PROD_STATE_BUCKET` in prod/providers.tf.
- Create or identify the new account Route53 hosted zone for bravoorders.com; replace `REPLACE_WITH_NEW_PROD_HOSTED_ZONE_ID` in prod/terraform.tfvars. Do not change registrar nameservers until cutover.
- Bootstrap GitHub OIDC and production-specific PR/deploy/ECR IAM roles in the new account. Verify the GitHub custom subject format and role permissions.
- Verify AWS CLI identity with `aws sts get-caller-identity` before any production Terraform operation.
- Confirm EKS quotas, EC2 capacity, RDS sizing, SES identity verification, and production backup requirements.

## Still required before deployment
- Split Argo CD applications and Helm values into Dev and production overlays. The shared values contain old account ECR URIs, Dev role ARNs, Dev SQS URLs, Dev Secrets Manager names, and Dev Karpenter selectors.
- Replace production image URIs with new-account ECR repositories and publish all nine application images. The build workflow currently targets an environment secret and must be explicitly configured for the new account without disrupting Dev.
- Configure production ExternalDNS with the new zone ID and a distinct TXT owner. Keep it isolated from the active Dev hosted zone during migration.
- Ensure production RDS database ExternalSecrets, service-specific credentials, KEDA, notification/SES, cert-manager, monitoring, and Velero values are correct.
- Separate the Terraform PR and deploy workflows by account. They currently assume old-account roles for every environment.
- Verify Terraform formatting, validate, Checkov, Helm lint/template, production plans, and all rendered resource identifiers.

## Safety gates
1. Do not run prod terraform init until the state bucket placeholder is replaced.
2. Do not run prod terraform plan until the hosted zone placeholder is replaced and the AWS identity is verified.
3. Do not apply production resources without reviewing the full plan and the IAM permissions.
4. Do not bootstrap production Argo CD against the shared Dev Kubernetes values.
5. Do not delegate the live DNS zone or switch bravoorders.com until production is tested.
6. Never run terraform destroy in the Dev account as part of production setup.

## Changes started in this branch
- Normalize prod Terraform naming to `bravo-eks-prod` and a valid PostgreSQL database name.
- Add an independent production runtime secret for JWT and Redis.
- Extend ESO secret ARN permissions to cover the new runtime secret.
- Parameterize ExternalDNS IAM hosted zone and explicitly require a new account hosted zone ID.
- Replace the old production backend bucket with a deliberate, nonfunctional placeholder.

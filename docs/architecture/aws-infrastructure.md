# AWS infrastructure and Terraform

Bravo EKS uses Terraform to provision and manage its AWS infrastructure across separate DEV and PROD accounts. Terraform is the source of truth for the VPCs, EKS clusters, IAM, queues, RDS, monitoring, and backup resources.

## Environment separation

| Environment | AWS account | Terraform root | GitOps application tree | Policy difference |
|---|---:|---|---|---|
| DEV | `304960797322` | `terraform/environments/dev` | `kubernetes/argocd/applications-dev` | Automated Argo CD self-heal and automated DEV image publishing |
| PROD | `083268669572` | `terraform/environments/prod` | `kubernetes/argocd/applications-prod` | Manual sync and explicit manual image publication |

The environment split is deliberate: the GitHub Actions workflows authenticate to the expected AWS account using environment-specific roles, then validate the account and registry before running image publication or Terraform deployment. The bootstrap stacks (`terraform/bootstrap-dev` and `terraform/bootstrap-prod`) create the Terraform state buckets, ECR repositories, and GitHub OIDC trust roles used by the pipelines.

## Terraform structure

```text
terraform/
├── bootstrap-dev/          # DEV state bucket, ECR repos, GitHub OIDC roles
├── bootstrap-prod/         # PROD state bucket, ECR repos, GitHub OIDC roles
├── environments/
│   ├── dev/               # VPC, EKS, RDS, IAM, SQS, monitoring, Velero
│   └── prod/              # Production environment root with prod-specific variables
└── modules/
    ├── ecr/
    ├── eks/
    ├── iam/
    ├── monitoring/
    ├── rds/
    ├── sqs/
    ├── velero/
    └── vpc/
```

The environment roots call reusable modules instead of embedding cluster or networking logic in each root. This keeps DEV and PROD consistent while allowing environment-specific values such as cluster names, instance sizing, and region/account-scoped role ARNs.

## Remote state and bootstrap

Both environments use an S3 backend created by their matching bootstrap stack. Each bootstrap root also creates an OpenID Connect provider for GitHub Actions and the roles used for ECR publication and Terraform deployment/PR checks. The state bucket configuration includes versioning, AES256 encryption, and public-access blocking, which keeps the infrastructure state durable and locked down.

The bootstrap roles are scoped to GitHub environments and repository identity. Workflows validate the selected role and AWS account before proceeding, so an invalid environment mapping stops the operation.

## VPC and networking

The VPC module creates:

- one VPC per environment
- public subnets for internet-facing ingress and NAT
- private application subnets for EKS workloads
- private database subnets for the PostgreSQL RDS subnet group
- one Internet Gateway for the public path
- one NAT Gateway in the first public subnet when enabled

The current network layout is a two-AZ model across the AWS region `eu-west-2` (`eu-west-2a` and `eu-west-2b`). Public subnets are mapped with `map_public_ip_on_launch = true`; the private application and database subnets are not internet-routable.

### Route model

- Public route tables route `0.0.0.0/0` to the Internet Gateway.
- Private EKS subnets route `0.0.0.0/0` to a NAT Gateway when enabled.
- Database subnets rely on the VPC default route semantics and are isolated from direct internet access.
- The private application subnets are where the EKS worker nodes and in-cluster workloads are scheduled.

### Security groups

Terraform manages the security groups:

- the EKS cluster uses the VPC security group path for cluster traffic
- the RDS instance accepts PostgreSQL only from the EKS cluster security group via a dedicated ingress rule on port 5432
- public access is disabled on PostgreSQL, which preserves a private-only database boundary

## EKS

The EKS module creates:

- the EKS control plane with public and private endpoint access enabled
- a KMS key for cluster secret encryption with automatic key rotation
- EKS cluster log types for API, audit, authenticator, controller manager, and scheduler
- the cluster OIDC provider for IRSA
- the core add-ons `vpc-cni`, `coredns`, and `kube-proxy`
- the EBS CSI driver and role with IRSA
- an EKS managed node group for system workloads

The EKS cluster is configured to use private subnets for the cluster endpoint and worker nodes, while still allowing the public endpoint for API access. The managed node role is assigned the standard EKS and ECR permissions; the Karpenter controller gets its own dedicated IAM role.

## RDS PostgreSQL

The RDS module provisions a private PostgreSQL instance using Amazon RDS with:

- `publicly_accessible = false`
- `storage_encrypted = true`
- `multi_az = false` for DEV; PROD is configured to support a production-safe multi-AZ model when selected by that environment variable set
- automatic backups enabled via `backup_retention_period`
- `skip_final_snapshot = false` with a final snapshot identifier
- PostgreSQL log export to CloudWatch
- database subnet group spanning the private database subnets

The instance uses AWS-managed master credentials generated in Secrets Manager and is accessed by workloads only through the EKS security-group path. The repository stores only the secret ARN/metadata in Terraform; actual credential values remain in AWS Secrets Manager and are distributed to Kubernetes through External Secrets Operator.

## SQS and DLQ

Each environment creates a main queue and dead-letter queue via the SQS module:

- main queue name is `${project}-${environment}-events`
- DLQ name is `${main}-dlq`
- the main queue has a visibility timeout configured by the module
- the DLQ redrive policy uses `maxReceiveCount` to move messages that fail repeatedly
- both queue types use SQS-managed server-side encryption

The queue exists as the asynchronous event backbone for order, payment, and shipping activity. Worker services consume the queue; publishers have least-privilege `sqs:SendMessage` permissions. The KEDA autoscaling stack reads queue depth information through a dedicated IAM role and scales the Worker from queue backlog.

## ECR and container delivery

The bootstrap stacks create ECR repositories for each service:

- api-gateway
- dashboard-api
- inventory-service
- notification-service
- order-service
- payment-service
- scheduler
- shipping-service
- worker

GitHub Actions builds and pushes images to these registries using short-lived OIDC credentials. The workflows validate the target account and ECR registry before pushing, so image publication is scoped to the selected environment.

## Route53, S3, and related AWS resources

The repository also includes the AWS resources required for platform operations:

- Route53 records are managed by ExternalDNS using a dedicated IRSA role
- Velero backup storage is provisioned in an encrypted S3 bucket with versioning and public-access blocking
- EBS snapshots are enabled through the CSI snapshot controller and IRSA-backed Velero role
- CloudWatch alarms monitor RDS CPU/storage and SQS queue/DLQ depth

Route 53 resolves the public DNS names to the load balancer, which forwards traffic to the Kubernetes ingress stack.

## IAM and least-privilege design

The Terraform IAM module creates workload-specific IRSA roles for:

- order-service
- payment-service
- shipping-service
- worker
- external-secrets
- external-dns
- velero
- KEDA
- notification service and other service components as required

Each role is bound to a specific Kubernetes service account subject, and associated permissions are narrowly scoped to the relevant task. For example:

- producer services receive only SQS `SendMessage` access to the event queue
- worker receives queue consumer permissions (`ReceiveMessage`, `DeleteMessage`, `ChangeMessageVisibility`, `GetQueueAttributes`)
- External Secrets reads only the approved Secrets Manager ARNs
- Velero can access its dedicated backup bucket and EBS snapshot APIs
- ExternalDNS can manage the Route53 zone for the bravoorders.com domain

This workload-identity pattern keeps long-lived AWS credentials out of the cluster and scopes access to the Kubernetes service account identity.

## GitHub OIDC and deployment guardrails

The bootstrap stacks create the GitHub OIDC provider and the AWS IAM roles for:

- GitHub image publishing (`bravo-github-ecr`)
- Terraform PR validation (`bravo-github-terraform-pr`)
- Terraform deployment (`bravo-github-terraform-deploy`)

Those roles are environment-specific and scoped to the correct GitHub environment or repository identity. The deployment workflows validate the selected target before applying Terraform or publishing images. This is the primary safeguard against accidental cross-environment drift.

## Network and database topology

The deployed topology combines a two-AZ VPC and private database networking with:

- DEV and PROD are separated at the account and role boundary
- one internet-facing Classic Load Balancer spanning the configured Availability Zones
- a private PostgreSQL instance whose subnet group spans two Availability Zones; the database instance itself is single-AZ

See [GitHub Actions pipelines](github-actions-pipelines.md) for the delivery workflow logic, environment mapping, and deployment guardrails.

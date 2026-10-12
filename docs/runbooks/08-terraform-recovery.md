# Terraform state, apply and destroy recovery

> **Safety:** Select the correct AWS account, cluster, namespace and environment before acting. Production changes require incident/change approval. Commands are diagnostic unless explicitly marked otherwise. Do not paste secret values into tickets or logs.

**Baseline:** AWS region `eu-west-2`; DEV/PROD are separate. The repository's canonical operational directory is `docs/runbooks/`. Cluster names should be verified with `aws eks list-clusters --region eu-west-2` rather than assumed.

```bash
export AWS_PROFILE=bravo
export AWS_REGION=eu-west-2
aws sts get-caller-identity
kubectl config current-context
kubectl get nodes -o wide
```

If the cluster has been destroyed, AWS and Kubernetes checks will fail by design. Recreate only through the approved Terraform workflow.

## Repository layout

Environment roots are under `terraform/environments/dev` and `terraform/environments/prod`; bootstrap/state infrastructure is separate. Inspect each environment's backend and variables before executing. Never assume DEV and PROD share the same state.

## Preflight

```bash
cd terraform/environments/dev   # choose intentionally
aws sts get-caller-identity
terraform init
terraform workspace show
terraform state list
terraform plan
```

## Error: state lock / S3 PreconditionFailed (412)

A 412 while acquiring an S3 state lock can mean another operation owns the lock or concurrent writes raced. **Do not force-unlock while another terminal, CI job, or engineer is applying/destroying.**

1. Check other terminals, GitHub Actions runs, and team ownership.
2. Wait for the active Terraform process to finish; re-run `terraform plan`.
3. If abandoned, collect lock ID and state backend details; obtain approval for recovery.
4. Use `terraform force-unlock <LOCK_ID>` **only after proving no active writer exists**. Never delete state or lock objects blindly.
5. Re-run `terraform plan`, compare remote state and AWS resources, then resume approved operation.

## Failed apply / drift

Check provider errors, IAM, quotas, dependencies and partially created resources. Use `terraform plan -refresh-only` for a read-only review of external changes (do not apply without approval). Import resources only after verifying address/identity. Never manually edit state as first-line remediation.

## Destroy checklist

Confirm environment/account, RDS backup requirements, Velero bucket retention, S3 state/bootstrap dependencies, and GitOps controllers that might recreate resources. Run `terraform plan -destroy` and review before `terraform destroy`. Afterwards verify EKS, RDS, load balancers, EC2/EBS, NAT gateways, Elastic IPs and S3 costs. AWS resources created by Kubernetes controllers may require separate cleanup.

## Verification

Run a fresh plan or state inspection, cross-check AWS console/CLI and document residual resources. Never interpret a partially completed destroy as full cleanup.

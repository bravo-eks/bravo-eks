# 01 — Cluster access and authentication

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

## Symptoms

`Unable to locate credentials`, `ExpiredToken`, `Unauthorized`, `Forbidden`, timeout to EKS API, or wrong context.

## Diagnose

```bash
aws configure list-profiles
aws sts get-caller-identity
aws eks list-clusters --region eu-west-2
aws eks describe-cluster --name <cluster-name> --region eu-west-2 --query 'cluster.{status:status,endpoint:endpoint,access:accessConfig}'
aws eks update-kubeconfig --name <cluster-name> --region eu-west-2 --profile bravo --alias <environment-alias>
kubectl config get-contexts
kubectl auth can-i get pods --all-namespaces
kubectl get namespaces
```

## Decision tree

1. **STS fails:** inspect AWS profile/SSO session, assumed role, MFA and expiration; reauthenticate through approved IAM workflow.
2. **Cluster not found:** check account/region, Terraform state, and whether DEV was intentionally destroyed.
3. **API timeout:** inspect EKS endpoint public/private access, client VPN/VPC route, DNS and security groups. Do not enable public API access casually.
4. **Unauthorized:** AWS identity may lack an EKS access entry or authorized IAM mapping; ask an administrator to inspect cluster access configuration.
5. **Forbidden:** authenticated successfully, but Kubernetes RBAC denies the verb/resource. Inspect role bindings; request least-privilege access rather than cluster-admin.

## Verification

Confirm correct account, region, context and namespace; perform a permitted read-only query. Record which authorization layer was responsible. Never commit kubeconfig credentials or tokens.

## Related

[Security incidents](07-security-incidents.md), [Terraform recovery](08-terraform-recovery.md).

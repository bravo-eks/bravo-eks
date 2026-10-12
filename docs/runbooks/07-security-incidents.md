# 07 — Security incidents: IAM, RBAC and secrets

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

## Trigger examples

Unexpected privileged API calls, suspicious pod/image, exposed credential, unexpected secret access, broad security-group rule, or unauthorized Kubernetes access.

## Immediate response

1. Declare security incident and involve authorized security/incident owner.
2. Preserve audit evidence (CloudTrail, Kubernetes audit if enabled, pod events, IAM principal, timestamps). Do not delete evidence prematurely.
3. Scope blast radius: affected account, EKS namespace, service account, AWS role, Secrets Manager secrets, RDS/SQS permissions.
4. Contain with least disruptive approved action: revoke/rotate compromised credential, restrict principal or isolate affected workload. Coordinate before terminating pods or rotating shared secrets.
5. Eradicate root cause, recover trusted workloads through GitOps, monitor for recurrence.

## Read-only checks

```bash
aws sts get-caller-identity
kubectl auth can-i --list -n <namespace>
kubectl get rolebindings,clusterrolebindings -A
kubectl get serviceaccounts -A
kubectl get externalsecrets,secretstores,clustersecretstores -A
kubectl describe externalsecret <name> -n <namespace>
kubectl get networkpolicies -A
aws cloudtrail describe-trails --region eu-west-2
```

## External Secrets failure path

Check ExternalSecret status, SecretStore readiness, AWS role/service-account trust, Secrets Manager ARN and region, `secretsmanager:GetSecretValue`, and `kms:Decrypt` where required. Inspect only metadata; do not print decoded Kubernetes Secrets or AWS secret strings.

## Hardening checklist

Least-privilege IAM/IRSA or Pod Identity as configured, namespace-scoped RBAC, restricted network access, ECR scanning and image provenance, encrypted secrets, audit logging, credential rotation, and reviewed Terraform changes. **Validate each control exists before claiming it is enforced.**

## Recovery verification

Compromised access no longer works; legitimate workloads remain healthy; new credentials are used; CloudTrail and Kubernetes evidence shows no ongoing suspicious actions. Document timeline, impact, approvals and preventive controls.

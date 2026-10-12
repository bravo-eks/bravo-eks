# 11 — Routine maintenance and upgrades

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

## Recurring checklist

**Daily / after changes:** check Argo CD health/sync, failed pods, node readiness, active alerts, SQS age/DLQ, RDS availability and recent deployment failures.

**Weekly:** review capacity trends, security findings, certificate expiry, backups and backup-job results, GitHub Actions failures, and AWS cost anomalies.

**Monthly / release window:** review EKS/Kubernetes support versions, node AMIs, Karpenter/KEDA/Traefik/Argo CD/Helm chart compatibility, IAM access, Terraform provider lockfile changes, retention and recovery exercises.

## Upgrade sequence

1. Open a reviewed change with scope, compatibility matrix, rollback constraints, and maintenance window.
2. Take/verify backups and ensure recovery steps are current.
3. Test in DEV; validate controllers, CRDs, webhook compatibility, PodDisruptionBudgets, node provisioning and application smoke tests.
4. Upgrade one component at a time; EKS control plane, add-ons and node versions have ordering constraints. Follow AWS-supported version-skew policies.
5. Monitor errors, latency, pending pods, Argo CD sync, SQS backlog and database health.
6. Document rollback feasibility: Kubernetes/EKS control-plane downgrades may not be supported; recovery may require replacement rather than downgrade.

## Example health snapshot

```bash
aws eks describe-cluster --name <cluster-name> --region eu-west-2 --query 'cluster.version'
kubectl get nodes -o wide
kubectl get pods -A
kubectl get applications -n argocd
kubectl get hpa,scaledobjects -A
kubectl get certificates -A
kubectl get backups -n velero
```

## Done criteria

Git desired state matches cluster, customer workflow succeeds, no new critical alerts, backups remain available, and the change record contains version changes and observed outcomes.

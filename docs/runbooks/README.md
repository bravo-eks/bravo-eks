# Bravo EKS operational runbooks

Use these procedures during incidents, planned maintenance, and study exercises. Start with **scope → evidence → root cause → safest mitigation → verification → follow-up**. Do not treat every symptom as a reason to restart a pod.

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

## Incident priority

- **SEV1:** widespread customer-facing outage, confirmed compromise, or data-loss risk. Declare incident, stop risky changes, coordinate recovery.
- **SEV2:** degraded critical service, growing SQS backlog, or partial customer impact. Assign owner and communicate status.
- **SEV3:** isolated failure, noncritical alert, or planned maintenance. Track and resolve in normal workflow.

## Runbook index

| File | Focus |
|---|---|
| [01-cluster-access.md](01-cluster-access.md) | AWS identity, EKS authentication, RBAC |
| [02-deployment-rollback.md](02-deployment-rollback.md) | Helm/Argo CD rollouts and recovery |
| [03-pod-troubleshooting.md](03-pod-troubleshooting.md) | CrashLoopBackOff, OOMKilled, Pending |
| [04-networking-ingress.md](04-networking-ingress.md) | Route 53, Classic LB, Traefik, Gateway API |
| [05-scaling-capacity.md](05-scaling-capacity.md) | HPA, KEDA, Karpenter, scheduling |
| [06-monitoring-alerts.md](06-monitoring-alerts.md) | Prometheus, Grafana, Alertmanager, CloudWatch |
| [07-security-incidents.md](07-security-incidents.md) | IAM, RBAC, secrets, containment |
| [08-terraform-recovery.md](08-terraform-recovery.md) | State locks, drift, failed apply/destroy |
| [09-disaster-recovery.md](09-disaster-recovery.md) | Velero, RDS PITR, outage recovery |
| [10-cost-operations.md](10-cost-operations.md) | EC2, NAT, LB, RDS, storage, logs |
| [11-maintenance.md](11-maintenance.md) | Version upgrades and recurring checks |
| [12-rds-database-connectivity.md](12-rds-database-connectivity.md) | Historical Dashboard API PostgreSQL grants incident and connectivity checks |
| [13-sqs-monitoring.md](13-sqs-monitoring.md) | SQS backlog, DLQ, KEDA and monitoring evidence |
| [14-cicd-gitops-troubleshooting.md](14-cicd-gitops-troubleshooting.md) | GitHub Actions, ECR, Argo CD, migrations |
| [15-aws-identity-secrets.md](15-aws-identity-secrets.md) | IRSA, IAM, ESO, Secrets Manager, SES |
| [16-cni-karpenter-network-recovery.md](16-cni-karpenter-network-recovery.md) | VPC CNI, pod IP allocation, Karpenter nodes |
| [17-application-order-flow.md](17-application-order-flow.md) | Customer catalog, orders, payments, shipping, notifications |
| [18-incident-history-and-triage.md](18-incident-history-and-triage.md) | Historical PR map and symptom-first routing |

## Historical coverage and limitations

The two original specialist runbooks are now numbered **12** and **13**, preserving the RDS worked example and the SQS DEV monitoring evidence. Use [18 — Incident history](18-incident-history-and-triage.md) to route historical fixes to procedures. Historical PR titles are not proof of a production incident or tested remediation. This collection is a living reference and cannot guarantee every possible failure.

## Standard runbook convention

Files are named `NN-kebab-case-topic.md` (except this index); each begins `# NN — Descriptive title`. Procedures should explain **purpose, symptoms, diagnosis, resolution, verification, escalation/prevention** and include links to verified history when applicable. Always verify environment, account and Git desired state before applying a change.

## Evidence and closure template

Record: UTC start/end, environment, customer impact, alerts, relevant deployments/commits, commands and redacted output, root cause (or unknown), mitigation, validation, owner, and prevention task. Avoid claiming an untested restore, failover, or SLO.

## Study approach

For each file, explain the dependency chain aloud, run read-only commands in DEV, and answer: **What layer failed? What evidence distinguishes causes? What would make my remediation unsafe? How will I know recovery succeeded?**

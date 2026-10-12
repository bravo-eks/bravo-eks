# Disaster recovery and data restoration

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

## Scope and dependencies

Bravo EKS uses Velero with S3/EBS snapshots for Kubernetes recovery and **separate RDS automated backups/PITR** for PostgreSQL. Velero is not a substitute for RDS backups. Redis and SQS have different durability/replay semantics. The repository README describes PROD RDS as private **single-AZ**; do not claim Multi-AZ failover.

## Triage

Identify whether the outage affects application pods, worker nodes, EKS control plane, VPC/ingress, RDS data, or an AWS regional service. Set recovery priority by customer impact and data integrity.

```bash
kubectl get nodes,pods -A
kubectl get backups -n velero
kubectl get backupstoragelocations -n velero
aws rds describe-db-instances --region eu-west-2
aws rds describe-db-snapshots --region eu-west-2
```

## Restore planning

1. Identify last verified backup, timestamp, encryption/KMS access, target environment and RPO/RTO requirements.
2. For Kubernetes objects/volumes, follow the approved Velero backup/restore procedure; inspect backup phase and CSI/EBS snapshot availability. Restore to a **separate nonproduction target first** where feasible.
3. For PostgreSQL corruption/loss, select an appropriate RDS snapshot or PITR timestamp. RDS PITR normally creates a **new DB instance**; plan DNS/connection-string/Secrets Manager cutover and schema compatibility.
4. Freeze or coordinate writers before cutover to prevent divergent data. Evaluate SQS message duplication/idempotency and DLQ replay.
5. Do not delete original data until restore and integrity checks succeed.

## Verification

Check restored object counts, PVC binding, app readiness, RDS connection and representative queries, order lifecycle, SQS worker idempotency, DNS/ingress and monitoring. Measure actual recovery duration and data gap; don't assert tested RTO/RPO without evidence.

## Related

[RDS connectivity](rds-database-connectivity.md), [SQS monitoring](sqs-monitoring.md), [Runbook index and incident priority](README.md).

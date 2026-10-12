# Backup and Recovery Architecture

Bravo EKS uses complementary recovery mechanisms for Kubernetes workloads and transactional data:

- **Velero** — Kubernetes resource and persistent-volume recovery
- **Amazon RDS backups and PITR** — PostgreSQL database recovery
- **Amazon S3** — durable storage for Velero backup artifacts
- **EBS CSI snapshots** — persistent-volume snapshot recovery

This separates Kubernetes recovery from database recovery while providing dedicated protection for each layer.

## Kubernetes Backup with Velero

Velero is deployed through the Kubernetes platform layer and configured with the AWS plugin.

The platform configuration provides:

- AWS-backed backup storage
- S3 backup persistence
- EBS CSI snapshot support
- IRSA-based AWS access
- scheduled backups for application workloads

Daily backup schedules protect the:

- `microservices` namespace
- `redis` namespace

The Kubernetes recovery path is:

```text
Kubernetes Resources
        │
        ▼
      Velero
       │   │
       │   └──► EBS CSI Snapshots
       │
       └──────► Amazon S3
```

This allows Kubernetes resources and supported persistent volumes to be restored through the same backup platform.

## S3 Backup Storage

Terraform provisions dedicated S3 storage for Velero backup artifacts.

The backup bucket includes:

- versioning
- server-side encryption
- public-access blocking
- environment-specific IAM access

Velero accesses the bucket through its Kubernetes service account and IRSA role, avoiding static AWS credentials inside the cluster.

Backup storage remains separate from the Terraform state backend, keeping infrastructure state and workload backups isolated.

## EBS CSI Snapshots

Bravo EKS integrates Velero with Kubernetes CSI snapshot support.

The platform includes the snapshot controller and `VolumeSnapshotClass` configuration required for EBS-backed persistent volumes.

With Velero's CSI integration enabled, persistent-volume recovery follows:

```text
Kubernetes PVC
      ↓
EBS CSI
      ↓
Volume Snapshot
      ↓
Velero Recovery
```

This complements Velero's Kubernetes resource backups with storage-level snapshots for workloads using persistent EBS volumes.

## RDS Backup and Point-in-Time Recovery

PostgreSQL recovery is handled independently through Amazon RDS.

The RDS configuration includes:

- automated backup retention
- configured backup windows
- encrypted storage
- snapshot tagging
- final snapshot protection
- PostgreSQL log exports

RDS automated backups support point-in-time recovery (PITR), providing a database-specific recovery path for the transactional state owned by the application services.

```text
Application Data
       ↓
Amazon RDS PostgreSQL
       ↓
Automated Backups
       ↓
Point-in-Time Recovery
```

This keeps PostgreSQL recovery
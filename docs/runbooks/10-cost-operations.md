# Cost operations and anomaly investigation

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

## Main cost drivers

EKS cluster hours, EC2 worker nodes (including Karpenter), RDS instance/storage/backups, NAT gateway hourly and processing fees, Classic Load Balancer, EBS snapshots, CloudWatch log ingestion/retention, S3/Velero, and data transfer.

## Investigate

```bash
aws sts get-caller-identity
aws ec2 describe-instances --region eu-west-2 --filters Name=instance-state-name,Values=running
aws eks list-clusters --region eu-west-2
aws rds describe-db-instances --region eu-west-2
aws elb describe-load-balancers --region eu-west-2
aws ec2 describe-nat-gateways --region eu-west-2
kubectl get nodes -o wide
kubectl top nodes
kubectl get nodepools,nodeclaims -A
```

Use AWS Cost Explorer for daily service/usage-type deltas and tags; the CLI may require explicit Cost Explorer permissions. Compare billable resources with Terraform state and Kubernetes ownership.

## Decision guide

- **EC2 spike:** check Karpenter NodeClaims, requests, consolidation/disruption policy, Spot/on-demand mix, and runaway HPA/KEDA replicas.
- **NAT spike:** inspect cross-AZ routing, outbound image pulls, logs and possible VPC endpoints; never remove egress needed for ECR/STS/S3/Secrets Manager.
- **RDS spike:** check instance class, storage, backups, retention and unnecessary always-on DEV resources.
- **CloudWatch spike:** inspect high-volume log groups, retention and debug-level logs.
- **Orphaned cost after destroy:** look for load balancers, EBS, snapshots, NAT/EIP, S3 and manually provisioned resources.

## Safety and verification

Estimate savings against reliability/security impacts. Do not downsize below peak requirements or remove backups merely to cut costs. Record before/after daily costs, capacity, latency and error rates. Costs may lag actual resource changes.

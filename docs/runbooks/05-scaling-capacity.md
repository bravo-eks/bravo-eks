# Scaling and capacity

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

## Model

**HPA** scales replicas from resource/custom metrics; **KEDA** scales event-driven consumers from triggers such as SQS queue depth; **Karpenter** provisions EC2 capacity for unschedulable pods. Scaling pods does not automatically mean nodes exist; scaling nodes does not guarantee healthy applications.

## Diagnose

```bash
kubectl get hpa -A
kubectl describe hpa <hpa> -n <namespace>
kubectl get scaledobjects -A
kubectl describe scaledobject <name> -n <namespace>
kubectl get nodepools,nodeclaims,ec2nodeclasses -A
kubectl describe nodepool <name>
kubectl get pods -A --field-selector=status.phase=Pending
kubectl get nodes -o wide
kubectl top nodes
kubectl get events -A --sort-by=.lastTimestamp
```

## Common failures

| Layer | Signal | Investigate |
|---|---|---|
| HPA | Unknown metrics | metrics-server, requests, HPA targets |
| KEDA | No scale on queue backlog | ScaledObject, authentication, SQS metric, polling, IAM |
| Scheduler | Insufficient CPU/memory | Pod requests vs allocatable, quotas |
| Karpenter | No new NodeClaim | NodePool requirements/limits, EC2NodeClass selectors, subnet IPs, IAM, EC2 quota |
| Nodes | NotReady or pressure | kubelet, CNI, disk/memory pressure, EC2 health |
| App | Latency worsens despite replicas | DB pool, RDS limits, Redis, downstream saturation |

## SQS-specific caution

Queue backlog and oldest-message age matter more than raw replica count. Increasing workers may overload PostgreSQL or downstream services. Inspect DLQ and visibility timeout; never blindly replay messages that may have side effects.

## Mitigation

Tune requests/limits based on measured usage; repair metrics/auth/selectors; adjust HPA/KEDA bounds or Karpenter capacity **through reviewed Git**. Assess quotas and downstream limits first. For emergencies, use an approved temporary override with documented expiry.

## Verify

Pending count falls, replicas/nodes stabilize, queue age declines, error rate stays acceptable, and costs do not spike unexpectedly.

## Related

[SQS monitoring](sqs-monitoring.md), [Cost operations](10-cost-operations.md).

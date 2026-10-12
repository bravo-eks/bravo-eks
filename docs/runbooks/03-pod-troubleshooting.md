# 03 — Pod troubleshooting

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

## Triage

```bash
kubectl get pods -A -o wide
kubectl describe pod <pod> -n <namespace>
kubectl logs <pod> -n <namespace> --all-containers --tail=100
kubectl logs <pod> -n <namespace> --previous --tail=100
kubectl get events -n <namespace> --sort-by=.lastTimestamp
kubectl top pod -n <namespace>
```

| Symptom | Evidence | Likely cause / next step |
|---|---|---|
| CrashLoopBackOff | Previous logs, termination reason | App crash, config, dependency or probe failure |
| OOMKilled | Exit code 137, memory trend | Memory pressure, leak, insufficient limit |
| Pending | Scheduler events | CPU/memory shortage, taints, affinity, PVC, Karpenter |
| ImagePullBackOff | Pod events | Bad image/tag, ECR permissions, network |
| CreateContainerConfigError | Events | Missing Secret/ConfigMap or invalid references |
| Running but NotReady | Readiness events | Endpoint/dependency failure or wrong probe |
| Evicted | Events/node conditions | Node memory/disk pressure |

## Resolve by layer

- **Application:** fix code/config or revert Git revision; don't mask repeated crashes by restarting.
- **Dependencies:** test RDS/Redis/SQS and ExternalSecret readiness without exposing credentials.
- **Capacity:** compare requests/limits to node allocatable and inspect Karpenter NodePools/NodeClaims.
- **Storage:** inspect PVC/PV, CSI events and EBS attachment/zone alignment.
- **Probes:** distinguish slow startup from genuinely unhealthy process; only adjust after evidence.

## Verification

```bash
kubectl get pods -n <namespace> -w
kubectl rollout status deployment/<deployment> -n <namespace>
kubectl get endpointslices -n <namespace>
```

Confirm stable restart count, Ready pods, working service endpoints, acceptable latency/errors, and no recurrence during an observation window.

## Related

[Scaling](05-scaling-capacity.md), [RDS connectivity](12-rds-database-connectivity.md), [Deployment rollback](02-deployment-rollback.md).

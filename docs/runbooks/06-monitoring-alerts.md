# Monitoring, observability and alert response

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

## Stack

Prometheus, Grafana, kube-state-metrics, node exporter, Alertmanager and CloudWatch are represented in the repository. Verify deployed targets, dashboards and alert rules for the selected environment; definitions alone do not prove an alert is live.

## First five minutes

1. Confirm alert name, firing duration, labels, environment and severity.
2. Identify customer impact: HTTP 5xx, latency, order completion, queue backlog, RDS health.
3. Compare onset with GitHub Actions builds, Argo CD syncs and Terraform changes.
4. Inspect the failing layer and preserve relevant graphs/logs.
5. Assign owner, communicate impact and choose safe mitigation.

## Diagnostic commands

```bash
kubectl get pods -A
kubectl top nodes
kubectl top pods -A
kubectl get svc -A
kubectl get events -A --sort-by=.lastTimestamp
kubectl get prometheusrules,servicemonitors -A
kubectl get alertmanager,prometheus -A
aws cloudwatch describe-alarms --region eu-west-2
```

## High-value signals

| Signal | Why it matters | First correlation |
|---|---|---|
| Request rate / 5xx / p95 latency | Customer experience | Ingress, app, recent deploy |
| Pod restarts / readiness | Workload health | Logs, probes, OOM |
| CPU/memory saturation | Capacity | HPA, requests, Karpenter |
| SQS age / DLQ count | Asynchronous work at risk | Worker, IAM, dependencies |
| RDS connections / CPU / storage | Database bottleneck | Query load, connection pool |
| Argo CD sync/health | Desired vs actual state | Git revision and drift |

## SLO study example (not a claim of implementation)

Availability SLI = successful eligible requests / total eligible requests. A 99.9% 30-day objective permits approximately 43.2 minutes of unavailability in that period. Define exclusions, measurement source, burn-rate alerts, and owner **before** presenting an SLO as operational.

## Close alert

Verify the symptom clears **and** the user journey recovers; annotate Grafana/incident record with timeline and preventive change. If metrics disappear, treat observability loss as an incident rather than proof of health.

## Related

[Runbook index and incident priority](README.md), [SQS monitoring](sqs-monitoring.md).

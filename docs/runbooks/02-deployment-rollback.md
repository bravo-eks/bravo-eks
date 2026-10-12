# Deployment failure and rollback (GitOps)

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

## Architecture

GitHub Actions builds/publishes images to ECR; Helm defines Kubernetes resources; Argo CD reconciles environment-specific Git manifests. **Git is the durable desired state**; an ad-hoc `kubectl rollout undo` may be reverted by Argo CD self-heal.

## Detect and diagnose

```bash
kubectl get deploy,pods -A
kubectl rollout status deployment/<deployment> -n <namespace> --timeout=120s
kubectl describe deployment/<deployment> -n <namespace>
kubectl get events -n <namespace> --sort-by=.lastTimestamp
kubectl logs deployment/<deployment> -n <namespace> --tail=100
kubectl get applications -n argocd
kubectl describe application <application> -n argocd
```

Check image tag/digest, ECR image availability, ImagePullBackOff, readiness/liveness, resource limits, ConfigMap/ExternalSecret changes, failed Helm templates, and Argo CD sync/health. Use `kubectl get rs -n <namespace>` to inspect revisions.

## Safest recovery

1. Declare impact, identify last healthy Git revision and image digest.
2. Revert the faulty **Git values/manifest change** via reviewed PR, or use an approved Argo CD rollback workflow that preserves desired-state consistency.
3. Observe Argo CD synchronization and rollout. Avoid manually editing resources that Argo CD owns.
4. If immediate mitigation is essential, coordinate temporary sync policy changes with the incident lead; document and restore Git reconciliation afterward.
5. If schema/database migrations are involved, check backward compatibility **before** rollback.

## Verify

```bash
kubectl rollout status deployment/<deployment> -n <namespace>
kubectl get pods,svc,endpointslices -n <namespace>
```

Confirm Argo CD is **Synced/Healthy**, traffic succeeds through the customer path, 5xx/error rate returns to baseline, and SQS consumers (if affected) resume. Record commit and image digest.

## Related

[Pod troubleshooting](03-pod-troubleshooting.md), [Monitoring and alerts](06-monitoring-alerts.md).

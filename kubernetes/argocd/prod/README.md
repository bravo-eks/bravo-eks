# Production Argo CD bootstrap (staged, not activated)

Argo CD is already installed manually in the Production EKS cluster
`bravo-eks-prod` using the `argo/argo-cd` Helm chart version `10.10.1`
and `kubernetes/platform/argocd/values.yaml`.

This directory provides a separate Production root and AppProject. It does
not modify the existing Dev GitOps tree or install workloads.

## Safety gates

- The Production repository credential is configured separately in the
  Production cluster. Never commit credentials or Kubernetes Secrets.
- The root points at `kubernetes/argocd/prod/applications`, not the
  existing Dev-oriented `kubernetes/argocd/applications` tree.
- The root does not use automated sync, prune or self-heal.
- The applications directory contains no manifests yet.
- Do not apply the root or project until the PR has been reviewed and
  merged and the target path exists on `main`.
- Do not enable ExternalDNS, change Route53 records or deploy services as
  part of this bootstrap.

## Future activation (requires separate approval)

After reviewing and merging this PR, validate the Production context and
review any added Production application manifests before creating the
AppProject and root Application:

```sh
aws sts get-caller-identity
kubectl --context bravo-eks-prod get nodes
kubectl --context bravo-eks-prod apply --dry-run=server -f kubernetes/argocd/prod/bootstrap-project.yaml
kubectl --context bravo-eks-prod apply --dry-run=server -f kubernetes/argocd/prod/root-app.yaml
```

These commands are **dry runs only**. Actual application and sync operations
require explicit authorization after configuration review.

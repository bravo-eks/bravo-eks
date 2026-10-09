# Production Argo CD applications

This directory is intentionally empty of Kubernetes manifests.

Do not copy the shared Dev-oriented `kubernetes/argocd/applications` tree into
Production. Review and add Production-specific platform Applications in separate
PRs, verifying AWS account IDs, IAM roles, cluster names, subnet/security-group
selectors, secrets, storage and DNS settings before any sync.

The Production root Application is deliberately manual: it has no
`syncPolicy.automated` block. Do not enable auto-prune or self-heal during
bootstrap. No applications should be synced until their manifests and impact
have been reviewed.

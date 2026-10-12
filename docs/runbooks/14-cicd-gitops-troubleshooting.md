# 14 — CI/CD and GitOps troubleshooting

## Purpose and scope
Use for GitHub Actions build, Terraform PR/deploy, ECR publish, Helm render, Argo CD sync, and migration Job failures. Repository workflows and environment branches are the source of truth; validate the current workflow before editing permissions.

## Symptoms and initial triage
- Workflow rejected by GitHub Actions parser; job never starts.
- AWS OIDC assumption fails with `AccessDenied` or missing IAM permission.
- Image builds but cannot push/pull from ECR.
- Argo CD application is `OutOfSync`, `Degraded`, or sync fails.
- Helm chart renders invalid resources; DB migration Job fails.

```bash
aws sts get-caller-identity
kubectl get applications -n argocd
kubectl describe application <application> -n argocd
kubectl get jobs -A
kubectl describe job <job> -n <namespace>
kubectl logs job/<job> -n <namespace> --all-containers --tail=100
kubectl get events -A --sort-by=.lastTimestamp
```

## Troubleshooting decision tree
1. **Workflow parse error:** inspect YAML indentation and duplicate keys/steps before looking at AWS. The Trivy action workflow was corrected in [PR #189](https://github.com/bravo-eks/bravo-eks/pull/189).
2. **OIDC/permissions:** inspect role trust conditions for repository/ref/environment and required actions; prefer narrowly scoped permissions. Review the sequence [#145](https://github.com/bravo-eks/bravo-eks/pull/145), [#151](https://github.com/bravo-eks/bravo-eks/pull/151), [#185](https://github.com/bravo-eks/bravo-eks/pull/185), [#189](https://github.com/bravo-eks/bravo-eks/pull/189). Do not grant broad admin access to fix one job.
3. **ECR push/pull:** check repository name/account/region, image digest, role permissions, node image-pull connectivity and image tag references. Compare build output to Helm values.
4. **Argo CD sync:** check repository auth, AppProject source/destination allow-lists, chart paths, destination namespace, CRDs and diff. [PR #109](https://github.com/bravo-eks/bravo-eks/pull/109) addressed bootstrap AppProjects; [#122](https://github.com/bravo-eks/bravo-eks/pull/122) addressed Velero namespace policy.
5. **Migrations:** inspect Job image, DB Secret key/property, role grants, database state, failed SQL and backward compatibility. [#113](https://github.com/bravo-eks/bravo-eks/pull/113), [#115](https://github.com/bravo-eks/bravo-eks/pull/115), [#118](https://github.com/bravo-eks/bravo-eks/pull/118) are historical references.
6. **Environment isolation:** ensure DEV/PROD build and deployment workflows point to the correct account, state and Argo CD app. See [#168](https://github.com/bravo-eks/bravo-eks/pull/168) and [#170](https://github.com/bravo-eks/bravo-eks/pull/170).

## Safe remediation
Revert the faulty workflow/values through a reviewed PR, rerun only the failed workflow/job after verifying idempotency, and reconcile Argo CD. Never rerun destructive Terraform or schema migration steps without checking state and backups.

## Verification
Successful GitHub Actions run, correct ECR image digest, Argo CD Synced/Healthy, migration Job Complete, ready application pods, and customer-facing smoke test.

## Escalation / prevention
Record workflow run URL, commit, IAM action denied, Argo CD application, and rollout result. Add a targeted CI validation check for the failure class.

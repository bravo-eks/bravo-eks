# 15 — AWS identity, IRSA and External Secrets

## Purpose and scope
Diagnose AWS permissions from pods/controllers and secret injection without revealing secret material. Applies to External Secrets Operator (ESO), KEDA, SQS workers, SES and Velero.

## Symptoms
`AccessDenied`, `AssumeRoleWithWebIdentity` failure, `SecretSynced=False`, missing `DATABASE_URL`, KEDA SQS scaler authentication failure, or SES publish failure.

```bash
aws sts get-caller-identity
kubectl get sa -A
kubectl describe sa <service-account> -n <namespace>
kubectl get externalsecrets,secretstores,clustersecretstores -A
kubectl describe externalsecret <name> -n <namespace>
kubectl get pods -n <namespace> -o custom-columns=NAME:.metadata.name,SA:.spec.serviceAccountName
kubectl logs deployment/<controller> -n <namespace> --tail=100
```

## Diagnose in order
1. Confirm actual pod namespace and service account. A historical PROD correction changed workload IAM namespace from `default` to `microservices` ([PR #184](https://github.com/bravo-eks/bravo-eks/pull/184)).
2. Check role ARN annotation and the IAM role trust policy's OIDC issuer, `sub` and `aud`. Verify controller service-account names before changing trust.
3. Check permissions to exact SQS queue, Secrets Manager secret, SES identity, KMS key or S3 bucket as appropriate.
4. For ESO, inspect SecretStore and ExternalSecret status, remote key and `property` mapping. [PR #113](https://github.com/bravo-eks/bravo-eks/pull/113) added missing `DATABASE_URL` property mappings; [#130](https://github.com/bravo-eks/bravo-eks/pull/130) expanded database secret access; [#149](https://github.com/bravo-eks/bravo-eks/pull/149) addressed Alertmanager Slack secret access.
5. For KEDA, check operator service account and `TriggerAuthentication` provider value. [PR #144](https://github.com/bravo-eks/bravo-eks/pull/144) introduced operator IRSA; [#148](https://github.com/bravo-eks/bravo-eks/pull/148) corrected provider `aws-eks` to `aws`.
6. For SES, check verified identity, sandbox restrictions, IAM and service configuration; see [#147](https://github.com/bravo-eks/bravo-eks/pull/147) and [#160](https://github.com/bravo-eks/bravo-eks/pull/160).

## Resolution
Apply the minimum IAM/trust/secret-property correction via Terraform or Helm/GitOps. Do not expose credentials or bypass workload identity with static keys. If an access key is compromised, follow [security incidents](07-security-incidents.md).

## Verification
Confirm ESO `SecretSynced=True`, expected Kubernetes Secret metadata exists, affected workload is Ready, and one safe service-specific AWS action succeeds. Never display decoded Secret values.

## Escalation / prevention
Document denied API action and resource ARN (redact account-sensitive context as appropriate), affected role and PR; add IAM trust/secret-property validation to review checklist.

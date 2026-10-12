# 13 — SQS monitoring and queue recovery

## Purpose
This document preserves the original DEV monitoring validation from [PR #154](https://github.com/bravo-eks/bravo-eks/pull/154). It is **historical evidence**, not a guarantee of current queue health.

## Historical monitoring validation

Reviewed and verified the existing SQS monitoring and alerting configuration.

### What was verified

* CloudWatch monitors the SQS message backlog using `ApproximateNumberOfMessagesVisible`.
* A dedicated CloudWatch alarm monitors the main queue backlog.
* A dedicated CloudWatch alarm monitors DLQ activity.
* Both alarms are connected to the existing SNS alerting path.
* The backlog alarm is configured with a threshold of 100 messages across two 5-minute evaluation periods.
* Verified the current queue health and alarm state in the dev environment.

### Testing

The current backlog alarm is in an `OK` state and the queue currently has no visible messages.

A live failure test was not performed because triggering the alarm would require stopping active consumers and intentionally creating a sustained backlog in the shared environment.

### Changes

Added documentation of the existing SQS monitoring and alerting configuration. No Terraform or application changes were required.

## Current incident procedure

1. Confirm environment, queue URL and DLQ ARN in Terraform; do not assume DEV queue names apply to PROD.
2. Inspect SQS CloudWatch metrics `ApproximateNumberOfMessagesVisible`, `ApproximateAgeOfOldestMessage`, `NumberOfMessagesDeleted`, and DLQ visible messages. Queue counts are approximate.
3. Check worker pods, errors, KEDA ScaledObject and TriggerAuthentication, IRSA and SQS permissions, visibility timeout, redrive policy and consumer idempotency.
4. Distinguish producer spike, stalled consumers, poison messages, permission errors and downstream RDS saturation before scaling workers.
5. Do **not** purge or redrive DLQ messages without approval, message inspection and duplicate-side-effect controls.

```bash
aws sts get-caller-identity
kubectl get scaledobjects -A
kubectl get pods -n microservices
kubectl logs deployment/<worker-deployment> -n microservices --since=20m --tail=100
aws cloudwatch describe-alarms --region eu-west-2
aws sqs list-queues --region eu-west-2
aws sqs get-queue-attributes --queue-url <queue-url> --attribute-names ApproximateNumberOfMessages ApproximateNumberOfMessagesDelayed ApproximateNumberOfMessagesNotVisible RedrivePolicy --region eu-west-2
```

## Recovery and verification
Repair IAM/consumer dependency or scale through reviewed GitOps. Confirm queue age/backlog decreases, DLQ stops growing, worker logs are clean, and representative order workflows complete once. Preserve original alarm settings until reviewed; the documented threshold of 100 messages for two five-minute periods was verified in DEV at the time of PR #154, not revalidated here.

## Related runbooks
[05 — Scaling](05-scaling-capacity.md) · [15 — AWS identity](15-aws-identity-secrets.md) · [17 — Application flow](17-application-order-flow.md) · [18 — Incident history](18-incident-history-and-triage.md)

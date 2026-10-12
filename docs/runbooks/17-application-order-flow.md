# 17 — Application, orders, payments, shipping and notifications

## Purpose and scope
Diagnose customer-facing failures even when EKS and pods are healthy. Bravo has Dashboard API, API Gateway, order, inventory, payment, shipping, notification, worker and scheduler services; PostgreSQL, Redis and SQS are shared dependencies.

## Request and event paths
HTTP: DNS → Classic LB → Traefik → Gateway API → Dashboard API → API Gateway/domain service. Async: order/payment/shipping events → SQS → worker → domain actions. Payment service simulates payments; do not describe it as a real card processor.

## Symptoms and checks
- **Catalog empty/UI blank:** check Dashboard API image and static asset routes, product API response, inventory DB permissions and query results. [PRs #176, #179, #181](https://github.com/bravo-eks/bravo-eks/pull/181) addressed production catalog rendering.
- **Orders fail:** trace API Gateway auth/rate limit, order-service logs, RDS transaction, inventory reservation, SQS publish and worker handling.
- **Shipping stuck:** inspect shipping-service logs and schema, event payload contract, worker retries and DLQ; see historical [#124](https://github.com/bravo-eks/bravo-eks/pull/124) and [#126–128](https://github.com/bravo-eks/bravo-eks/pull/128).
- **Notifications absent:** inspect notification-service database, worker events, SES identity/permissions and external delivery constraints.
- **Unexpected 429:** check API Gateway Redis connectivity and rate-limit configuration, not just ingress.
- **Refund inconsistency:** correlate payment ledger, order status, SQS redelivery and idempotency keys; never replay a potentially non-idempotent payment/refund blindly.

```bash
kubectl get deployments,pods,svc -n microservices
kubectl logs deployment/<deployment-name> -n microservices --since=20m --tail=200
kubectl get events -n microservices --sort-by=.lastTimestamp
kubectl get externalsecrets -n microservices
aws sqs list-queues --region eu-west-2
```

## Triage workflow
1. Reproduce using a non-sensitive test account/order and capture timestamp/request or order ID (redact PII).
2. Identify whether HTTP, persistence, or asynchronous event processing failed.
3. Follow the same correlation ID/order ID across service logs; if absent, use timestamps and safe identifiers.
4. Compare recent image/Helm changes and DB migrations; verify downstream health and permissions.
5. Mitigate via GitOps rollback only if data/schema compatibility is understood. Avoid direct DB edits without approved corrective procedure.

## Verification
Run a safe end-to-end catalog → cart/order → simulated payment → shipping/notification smoke test in an approved environment. Confirm database state and SQS backlog/DLQ, not just HTTP 200.

## Escalation / prevention
Record failing step, service, Git revision, event ID, DB table/schema and idempotency result; add regression tests for the exact failure.

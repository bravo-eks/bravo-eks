<p align="center">
  <img src="images/coderco.jpg" alt="CoderCo" width="200"/>
</p>

# EKS v2 - Order Fulfillment Platform

A multi-service order platform on Amazon EKS. Nine services, one cluster, stateful workloads running on Kubernetes. The application code is provided. You build everything else.

---

## Services

| Service | Description |
|---------|-------------|
| **api-gateway** | Auth, rate limiting, routes requests to internal services |
| **order-service** | Order lifecycle and state machine |
| **inventory-service** | Stock management and reservations |
| **payment-service** | Payment processing, refunds, ledger |
| **notification-service** | Email and SMS dispatch |
| **shipping-service** | Shipments, tracking, carrier webhooks |
| **worker** | SQS consumer, orchestrates cross-service events |
| **scheduler** | Cron jobs (expired reservations, abandoned orders, retries) |
| **dashboard-api** | Admin dashboard UI, analytics and reporting |

Read the source code. Environment variables, endpoints and data models are in the code.

---

## Your Job

Write the Dockerfiles. Write the Terraform. Write the Kubernetes manifests. Write the CI/CD pipeline. Deploy all nine services to EKS, with PostgreSQL and Redis running in-cluster on persistent volumes.

### Requirements

- EKS (1.33 or above) with managed node groups across 3 AZs
- Karpenter for node autoscaling. Cluster Autoscaler is fine to compare against, but Karpenter is the default you should be reaching for.
- Nine Deployments behind a single Ingress with TLS, routed to the right backend
- PostgreSQL on a StatefulSet with a 20Gi PVC (gp3, encrypted)
- Redis on a StatefulSet with AOF persistence on a 10Gi PVC (gp3, encrypted)
- AWS EBS CSI Driver with IRSA, gp3 as the default storage class, a VolumeSnapshotClass configured
- SQS queue with a dead letter queue as the event bus. If you would rather run in-cluster Kafka via the Strimzi operator, that is also accepted, but you will need to modify the four event-producing services to use Kafka clients. Pick one path, defend it.
- ECR repositories, one per service
- VPC with private subnets. Avoid NAT gateways if you can.
- Secrets sourced from AWS Secrets Manager via External Secrets or the Secrets Store CSI Driver. Not hardcoded, not in env files.
- Traefik as the Ingress controller, fronted by an AWS NLB. ingress-nginx is retired (no releases, no security fixes after March 2026) so do not pick it. cert-manager with Let's Encrypt for TLS. ExternalDNS managing Route 53 records.
- GitHub Actions with OIDC. No long-lived AWS credentials.
- ArgoCD in the cluster, App-of-Apps pattern, auto-sync on the dev overlay
- Zero-downtime rollouts with rollback on failure
- Least-privilege IAM with IRSA for every service that touches AWS
- Terraform with remote state
- Multi-stage Docker builds
- Container image scanning before deploy

### Deliverables

- [ ] Dockerfiles, one per service
- [ ] Terraform for all infrastructure (VPC, EKS, IAM, ECR, SQS, addons)
- [ ] Kubernetes manifests (Kustomize or Helm)
- [ ] ArgoCD Applications wiring the cluster to your manifests repo
- [ ] GitHub Actions pipelines for infra and app, separated
- [ ] Working deployment with all services healthy and the end-to-end flow functional
- [ ] Dashboard UI reachable over HTTPS at a real DNS name, connected to all services
- [ ] README covering the sections below

---

## What Your README Must Cover

This is not optional. Your README is part of the submission.

**Architecture decisions** - what you built, why you built it that way, what you traded off. Why StatefulSets for Postgres instead of RDS. Why Kustomize over Helm or the other way round.

**Deployment pipeline** - a developer pushes a change to the payment service. Walk through exactly what happens from commit to live traffic. How do app deploys and infra changes stay out of each other's way? What triggers what? Where does ArgoCD fit in that flow?

**Secrets management** - nine services need database credentials, API keys, JWT secrets. How do they get from Secrets Manager into a pod? What happens when you rotate a secret?

**Storage** - Postgres holds the only durable state in the system. How is the PVC backed? Encrypted? Snapshotted? What is your restore procedure and have you actually tested it?

**Scaling strategy** - which services scale, on what metric, with HPA or KEDA? What stays fixed? What breaks first under load?

**Database migrations** - seven services share one database. How do schema changes get applied? Job, init container, manual? What about rollback?

---

## Things to Consider

These are not requirements. They are the kind of problems you will hit in production. How you handle them is up to you.

- Your Postgres pod gets rescheduled to a different AZ. What happens to its PVC?
- The worker processes events from SQS. What happens to events that fail three times?
- The payment service goes down for two minutes. What happens to in-flight orders?
- You need to add a column to the orders table. The dashboard service reads from that table. How do you deploy both without downtime?
- A junior dev pushes a bad image for the notification service. How quickly can you roll back without affecting the other eight? Does ArgoCD help or hurt here?
- Spot instances save money. Which workloads tolerate eviction? Which absolutely cannot?
- Your logging pipeline ingests from nine services plus the cluster control plane. What does that cost per month? Is there a cheaper way?
- You rotate the database password. Do all nine deployments restart? Is there a way to avoid that?
- A single-AZ EBS volume becomes a problem when the AZ goes down. What is your answer?

---

## Local Development

```bash
docker compose up --build
```

---

## Advanced

Not required for submission. These will set your project apart.

**Observability**. It is 2am. Orders are failing. You are on call. You need to answer four questions fast: which service is the problem, when did it start, what changed, who is affected. If your setup cannot answer those in under 10 minutes without `kubectl exec`, it is not production-ready. kube-prometheus-stack gives you the building blocks. RED metrics for the API layer. Saturation metrics for the data layer. Dashboards grouped by service, not by pod. Alerts that mean something. A way to follow one order across all nine services.

**Service mesh**. Istio or Linkerd. mTLS between every service. Authorization policies that block lateral movement. Traffic shifting for canary releases.

**Distributed tracing with OpenTelemetry**. Run an OTel collector. Instrument the Go services (the SDK is small). Send spans to Tempo or Jaeger. Follow a single order across api-gateway, order-service, payment-service, shipping-service and worker. This is the bit that pays you back at 2am.

**Gateway API instead of Ingress**. Traefik supports Gateway API CRDs (GatewayClass, Gateway, HTTPRoute). The Kubernetes community is moving in this direction now that ingress-nginx is gone. Build the platform with Gateway resources instead of `Ingress`.

**Backup and disaster recovery**. Velero for cluster-level backup. EBS snapshots on a schedule. Restore in a fresh cluster and prove the application comes back up with its data intact.

**Chaos drill**. Kill a Postgres pod live during your demo. Watch the StatefulSet bring it back. Watch the application recover. One concrete drill, not a chaos platform.

---

## Grading

- All nine services running and healthy on EKS
- End-to-end flow works through the dashboard UI (create order -> reserve inventory -> process payment -> ship -> deliver)
- Postgres and Redis on StatefulSets with persistent volumes that survive pod restarts
- Volume snapshot taken and restored successfully
- Application reachable over HTTPS at a real DNS name
- Pipeline deploys only what changed
- Secrets not hardcoded anywhere
- No long-lived AWS credentials
- README covers all required sections with real decisions, not filler
- You can explain every resource you created

**Tear down when done.** EKS, EBS, NLB and data transfer add up fast.

Everything else is on you. Good luck.

---

## Found a bug?

The services have rough edges (see the audit notes in the team review channel). If you spot a real bug, open a PR against this repo. Include screenshots of the bug reproducing, your fix, and the same scenario working after the fix. The CoderCo team will review it. Good fixes stand out at grading time.

# # Data Architecture and Asynchronous Messaging

Bravo EKS uses three data technologies with distinct responsibilities:

- **Amazon RDS PostgreSQL** — transactional application state
- **Amazon SQS** — asynchronous domain events
- **Redis** — API Gateway rate-limiting state

The architecture follows clear service ownership boundaries: domain services own their PostgreSQL schemas, while the Worker coordinates asynchronous workflows through SQS.

## PostgreSQL Architecture

Each environment uses a private Amazon RDS for PostgreSQL instance deployed within the database subnet layer.

RDS:

- is not publicly accessible
- accepts PostgreSQL traffic from the EKS environment
- uses encrypted storage
- provides the shared database platform for the domain services

Application data is separated using a **schema-per-domain** model.

| Service | PostgreSQL User | Schema / Access |
|---|---|---|
| order-service | `order_user` | `orders` |
| inventory-service | `inventory_user` | `inventory` |
| payment-service | `payment_user` | `payments` |
| notification-service | `notification_user` | `notifications` |
| shipping-service | `shipping_user` | `shipping` |
| dashboard-api | `dashboard_user` | Read-only reporting access |
| scheduler | `scheduler_user` | Controlled cross-schema access |

Each state-owning service normally writes only to its own schema.

Dashboard API receives read access required for reporting and customer/admin views, while Scheduler receives explicit cross-schema permissions for scheduled maintenance and operational jobs.

The complete authorization model is documented in [RDS Permission Matrix](rds-permission-matrix.md).

## Service Data Ownership

The database structure follows the application's domain boundaries:

- **Order Service** — order lifecycle and order events
- **Inventory Service** — products, stock, and reservations
- **Payment Service** — payments and ledger entries
- **Shipping Service** — shipments and tracking
- **Notification Service** — notifications and templates
- **Dashboard API** — read-oriented access across required reporting data
- **Scheduler** — scheduled maintenance and lifecycle operations

API Gateway and Worker do not require direct PostgreSQL access.

This keeps transactional writes with the services that own the corresponding business data.

## Database Access

Database credentials are service-specific rather than using a shared application administrator account.

Runtime `DATABASE_URL` values are delivered through the platform's secret-management flow:

```text id="yx61dn"
AWS Secrets Manager
        ↓
External Secrets Operator
        ↓
Kubernetes Secret
        ↓
Service DATABASE_URL
        ↓
Amazon RDS PostgreSQL
```

Combined with PostgreSQL grants, this provides both network-level and database-level access boundaries.

## Amazon SQS

Amazon SQS provides the asynchronous messaging layer for Bravo EKS.

Each environment has:

- a main event queue
- a dead-letter queue (DLQ)
- a redrive policy
- server-side encryption
- CloudWatch queue monitoring

Domain services publish events to SQS, while the Worker consumes those events and coordinates downstream processing.

```text id="g8f4mm"
Domain Service
     │
     ▼
 Amazon SQS
     │
     ▼
   Worker
     │
     ├──► Order Service
     ├──► Inventory Service
     ├──► Payment Service
     ├──► Shipping Service
     └──► Notification Service
```

Events include workflow transitions such as:

- `order.created`
- `payment.completed`
- `payment.failed`
- shipment lifecycle events

SQS provides at-least-once delivery, while the DLQ captures messages that repeatedly fail processing so they remain available for investigation and recovery.

## Worker Orchestration

The Worker is the central consumer for asynchronous domain events.

It long-polls the SQS queue, processes incoming events, and coordinates the appropriate downstream service calls.

Typical workflows include:

- inventory reservation
- payment initiation
- payment result processing
- order status transitions
- shipment creation
- notification processing

The Worker coordinates these actions without directly owning the underlying PostgreSQL data.

This preserves the service ownership model while allowing multi-service workflows to operate asynchronously.

## Redis

Redis provides rate-limiting state for API Gateway.

API Gateway receives its `REDIS_URL` through the secret-management flow and uses Redis to maintain short-lived request counters.

```text id="u2w5mi"
Client Request
      ↓
 API Gateway
      ↓
    Redis
 rate:<client-ip>
      ↓
Rate-limit decision
```

Redis runs as a stateful Kubernetes workload in the `redis` namespace and is deployed through the dedicated Redis Helm chart.

Its responsibility is deliberately focused on fast, temporary rate-limiting state, while PostgreSQL remains the transactional system of record.

## Service Dependencies

| Service | PostgreSQL | SQS | Redis |
|---|---|---|---|
| api-gateway | No | No | Yes |
| dashboard-api | Yes | No | No |
| inventory-service | Yes | No | No |
| notification-service | Yes | No | No |
| order-service | Yes | Yes | No |
| payment-service | Yes | Yes | No |
| scheduler | Yes | No | No |
| shipping-service | Yes | Yes | No |
| worker | No | Yes | No |

## Data Architecture Summary

The three data layers have clearly separated responsibilities:

```text id="yq4f7c"
                   Bravo EKS Data Layer

                       Application
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
     PostgreSQL           SQS             Redis
          │                │                │
   Transactional       Domain Events    Rate Limiting
      State                │
          │                ▼
   Service-owned         Worker
      Schemas              │
                           ▼
                    Service Workflows
```

This gives Bravo EKS a structured data architecture where PostgreSQL provides durable transactional state, SQS decouples asynchronous workflows, Redis handles high-speed rate-limiting state, and service-specific permissions preserve domain ownership.

## Related Documentation

- [RDS Permission Matrix](rds-permission-matrix.md)
- [Secrets Management](secrets-management.md)
- [AWS Infrastructure & Terraform](aws-infrastructure.md)
- [Application Flow](application-flow.md)
- [SQS Monitoring Runbook](../runbooks/13-sqs-monitoring.md)
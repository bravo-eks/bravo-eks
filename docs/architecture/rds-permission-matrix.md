# BRAVO RDS Permission Matrix

## Database architecture

One PostgreSQL RDS instance contains one application database:

- Database: `bravodev`
- PostgreSQL schemas:
  - `orders`
  - `inventory`
  - `payments`
  - `notifications`
  - `shipping`

Each state-owning service has its own PostgreSQL user and schema.

---

## Service database ownership

| Service | PostgreSQL User | Schema | Tables |
|---|---|---|---|
| order-service | `order_user` | `orders` | `orders`, `order_events` |
| inventory-service | `inventory_user` | `inventory` | `products`, `reservations` |
| payment-service | `payment_user` | `payments` | `payments`, `ledger_entries` |
| notification-service | `notification_user` | `notifications` | `notifications`, `notification_templates` |
| shipping-service | `shipping_user` | `shipping` | `shipments`, `tracking_events` |

---

## Runtime permissions

Normal service users can operate only within their own schema.

| PostgreSQL User | Schema | Table | SELECT | INSERT | UPDATE | DELETE |
|---|---|---|---:|---:|---:|---:|
| `order_user` | orders | orders | ✓ | ✓ | ✓ | ✓ |
| `order_user` | orders | order_events | ✓ | ✓ | ✓ | ✓ |
| `inventory_user` | inventory | products | ✓ | ✓ | ✓ | ✓ |
| `inventory_user` | inventory | reservations | ✓ | ✓ | ✓ | ✓ |
| `payment_user` | payments | payments | ✓ | ✓ | ✓ | ✓ |
| `payment_user` | payments | ledger_entries | ✓ | ✓ | ✓ | ✓ |
| `notification_user` | notifications | notifications | ✓ | ✓ | ✓ | ✓ |
| `notification_user` | notifications | notification_templates | ✓ | ✓ | ✓ | ✓ |
| `shipping_user` | shipping | shipments | ✓ | ✓ | ✓ | ✓ |
| `shipping_user` | shipping | tracking_events | ✓ | ✓ | ✓ | ✓ |

> DELETE permissions should be validated against the service SQL before final grants are applied.

---

## Cross-service access

### scheduler_user

Permissions confirmed from `services/scheduler/main.go` and live scheduler execution:

| Schema | Table | SELECT | INSERT | UPDATE | DELETE |
|---|---|---:|---:|---:|---:|
| inventory | reservations | ✓ | ✗ | ✓ | ✗ |
| inventory | products | ✗ | ✗ | ✓ | ✗ |
| orders | orders | ✓ | ✗ | ✓ | ✗ |
| orders | order_events | ✓ | ✗ | ✗ | ✓ |
| payments | payments | ✓ | ✗ | ✗ | ✗ |

### dashboard_user

Permissions confirmed from `services/dashboard-api/main.go`:

| Schema | Table | SELECT | INSERT | UPDATE | DELETE |
|---|---|---:|---:|---:|---:|
| orders | orders | ✓ | ✗ | ✗ | ✗ |
| payments | payments | ✓ | ✗ | ✗ | ✗ |
| inventory | products | ✓ | ✗ | ✗ | ✗ |
| shipping | shipments | ✓ | ✗ | ✗ | ✗ |

---

## Services with no PostgreSQL access

| Service | PostgreSQL |
|---|---|
| api-gateway | ✗ |
| worker | ✗ |

---

## Schema isolation

Each normal service user must have:

```sql
ALTER ROLE <service_user> SET search_path = <service_schema>;
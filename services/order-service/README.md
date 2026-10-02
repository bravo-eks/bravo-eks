# Order Service

The Order Service manages order creation and order status transitions. It persists order data and order events in PostgreSQL and publishes order events to SQS for downstream processing.

## Production Hardening Changes

1. **SQS Integration**
   - Added AWS SDK for Go v2.
   - Replaced the previous SQS logging placeholder with actual SQS `SendMessage`.
   - Existing `order.created` and `order.status_changed` event structures are preserved.

2. **SQS Configuration Validation**
   - `SQS_QUEUE_URL` is required, and the application fails fast when it is missing.
   - AWS configuration is loaded using `config.LoadDefaultConfig`.
   - No AWS credentials are hardcoded; the AWS default credential chain supports workload credentials such as IRSA in EKS.

3. **SQS Error Handling**
   - SQS publishing errors are returned and handled rather than silently ignored.
   - Failed event publishing is logged and returned as an application error.

4. **Order Creation Transaction**
   - Order creation and the corresponding `order_events` insert use a database transaction.
   - Database errors are handled, and transaction commit errors are checked.

5. **Status Update Transaction**
   - Status updates and the corresponding `order_events` insert use a database transaction.
   - The current order status is locked while the transition is processed to protect against concurrent updates.
   - Database and transaction errors are handled.

6. **Order Input Validation**
   - Orders must contain at least one item, and product IDs are required.
   - Quantities must be greater than zero; negative prices are rejected.
   - Existing default currency behaviour remains.

7. **Status Update Validation**
   - Required status-update fields are validated.
   - Existing allowed status-transition rules remain unchanged.

8. **Migration Control**
   - Database migrations can be explicitly enabled using `RUN_MIGRATIONS=true`.
   - This separates migration execution from normal application runtime; migrations are disabled by default.
   - Existing migration definitions remain unchanged.

## Runtime Configuration

### Required

- `DATABASE_URL`: PostgreSQL connection string.
- `SQS_QUEUE_URL`: identifies the queue used for order events.

### Optional

- `PORT`: HTTP listen port; defaults to `8081`.
- `RUN_MIGRATIONS`: set to `true` when migrations should run; otherwise migrations are disabled.

AWS credentials are not stored in the application. In EKS, AWS authentication is expected to use workload credentials/IRSA through the AWS SDK default credential chain.

## Validation and Testing

### Go Build

The service was compiled successfully using Go 1.22 in Docker:

```sh
docker run --rm \
  -v "$PWD/services/order-service:/app" \
  -w /app \
  golang:1.22 \
  go build ./...
```

Result: build completed successfully.

## Result

The Order Service persists orders and status changes transactionally, validates order and status inputs, and publishes events to SQS with explicit configuration and error handling. The Go 1.22 Docker build completed successfully.

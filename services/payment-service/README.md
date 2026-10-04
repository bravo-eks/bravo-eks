# Payment Service

The Payment Service handles payment charges, refunds, payment state, ledger entries, and payment event publishing. It persists payment and ledger data in PostgreSQL and publishes payment events to SQS for downstream processing.

## Production Hardening Changes

1. **SQS Integration**
   - Added real AWS SQS `SendMessage` using AWS SDK for Go v2.
   - Uses the AWS default credential chain and supports IRSA/workload credentials in EKS.
   - Existing event names and payload formats are preserved.

2. **SQS Configuration Validation**
   - `SQS_QUEUE_URL` is required.
   - The service fails fast when it is missing.

3. **SQS Error Handling**
   - Publishing errors are returned and handled.
   - Failures are logged rather than silently ignored.

4. **Charge Transaction Safety**
   - Payment and ledger writes remain in the same database transaction.
   - Charge and revenue ledger insert errors are checked.
   - `tx.Commit()` errors are checked.
   - Failed ledger operations prevent the transaction from committing.

5. **Refund Transaction Safety**
   - The refund payment, refund ledger entry, and original payment status update occur in the same transaction.
   - The original payment is selected with `FOR UPDATE` during refund processing.
   - Ledger and status update errors are handled, and commit errors are checked.

6. **Charge Validation**
   - `order_id` must be greater than zero and `customer_id` is required.
   - The amount must be positive.
   - The existing GBP default behaviour is preserved when currency is omitted.

7. **Refund Validation**
   - `payment_id` is required and the refund amount must be positive.
   - Refunds are allowed for completed and partially refunded payments.
   - Existing refunds are calculated before another refund is accepted, and a refund cannot exceed the remaining refundable balance.
   - Multiple partial refunds are supported; further refunds are rejected after the payment is fully refunded.

8. **Ledger Error Handling**
   - Ledger query and row scan errors are handled.
   - `rows.Err()` is checked after iteration.

9. **Balance Error Handling**
   - Errors calculating charged and refunded balances are handled.
   - Database failures return an error instead of silently reporting misleading zero balances.

10. **Migration Control**
    - Migrations do not run automatically during normal startup.
    - Set `RUN_MIGRATIONS=true` to explicitly enable migrations.

11. **Simulated Payment Provider Preserved**
    - Existing simulated payment provider behaviour remains unchanged, with a 90% success and 10% failure rate.

## Runtime Configuration

### Required

- `DATABASE_URL`: PostgreSQL connection string.
- `SQS_QUEUE_URL`: queue URL used for payment event publishing.

### Optional

- `PORT`: HTTP listen port; defaults to `8083`.
- `RUN_MIGRATIONS`: set to `true` to run database migrations; defaults to `false`.

AWS credentials are not stored in the application. The AWS SDK default credential chain allows workload credentials such as IRSA to be used in EKS.

## Validation and Testing

The following were local integration tests using PostgreSQL and LocalStack. No tests against real AWS SQS or AWS RDS are claimed.

### Go Build

Using Go 1.22, `gofmt` and `go mod tidy` completed, and `go build ./...` completed successfully with no errors.

### Liveness Test

`/livez` returned HTTP 200.

### Database Health Test

`/healthz` returned HTTP 200 with `payment-service` status `ok`.

### Charge Test

Submitted a charge with `order_id: 1`, `customer_id: test@example.com`, `amount: 100`, `currency: GBP`, and `method: card`. The simulated provider returned `completed`, and the payment was stored successfully in local PostgreSQL.

Payment ID: `pay_1790872324450828047_8801`

### Charge Ledger Test

Database verification showed the charge ledger entry with a debit of `100` and the revenue ledger entry with a credit of `100`.

### Payment SQS Integration Test

Using LocalStack SQS, the successful charge produced a `payment.completed` message through AWS SDK `SendMessage`. The message was retrieved from the LocalStack queue and contained the expected event type and payload.

### Partial Refund Test

A GBP 30 refund was made against the GBP 100 payment. Database verification showed the original payment as `partially_refunded`, the refund payment as `completed` and referencing the original payment, and a GBP 30 refund ledger entry.

### Multiple Refund Test

A second GBP 70 refund was successfully processed against the same payment. This verified that a partially refunded payment can receive further refunds up to its remaining refundable balance. The original payment then became `refunded`.

### Over-Refund Protection Test

After the GBP 100 had been fully refunded, a further GBP 1 refund was rejected with `can only refund completed or partially refunded payments`. This verified that a fully refunded payment cannot be refunded again.

### Refund SQS Integration Test

The successful GBP 30 and GBP 70 refunds each produced a `payment.refunded` event, and both events were retrieved from LocalStack SQS.

### Current Consistency Limitation

The PostgreSQL transaction and SQS `SendMessage` are not atomic. The database transaction commits before SQS publishing, so payment or refund state may exist without its event being delivered if publishing fails. A transactional outbox/retry pattern could address this in a future improvement; it is outside the scope of this work.

## Result

The Payment Service publishes payment events to SQS with AWS/IRSA credential resolution, fails fast when required SQS configuration is missing, and handles payment, refund, ledger, and commit errors. It validates charges and refunds, supports controlled partial refunds, prevents over-refunding, separates migrations from normal runtime, and preserves the simulated provider and existing event formats.

# Scheduler Service

The Scheduler Service runs recurring background jobs for reservation expiry, abandoned orders, failed-payment reporting, daily reporting, and old-event cleanup. It uses PostgreSQL as its shared data source and runs each scheduled job independently.

## Production Hardening Changes

### 1. Scheduled Job Error Handling

Scheduled jobs now return errors to the scheduler runner instead of silently swallowing database failures.

- Failed jobs are logged as failed.
- Failed jobs are not logged as successfully completed.
- A failure in one scheduled job does not stop the other scheduled jobs.
- Existing job intervals and the current goroutine-based scheduler architecture are preserved.

### 2. Reservation Expiry Error Handling

The reservation-expiry job now checks database errors throughout processing.

- Reservation query and `rows.Scan()` errors are checked.
- Transaction creation errors are handled.
- Reservation and product update errors are checked.
- Affected row counts are verified.
- Transaction commit errors are handled.
- `rows.Err()` is checked after iteration.
- Reservations are only counted as expired after a successful transaction commit.

### 3. Reservation Transaction Safety

Reservation expiry is performed transactionally.

The reservation is marked as expired and the corresponding product's reserved quantity is reduced within the same PostgreSQL transaction. If the product update fails, the transaction is rolled back so the reservation does not remain incorrectly marked as expired.

This prevents partial reservation-expiry updates.

### 4. Abandoned Order Error Handling

The abandoned-order job now validates each database operation.

- `rows.Scan()` errors are checked.
- Order update errors are handled.
- `RowsAffected()` is checked to verify the order was actually updated.
- An order is only logged as cancelled after a successful database update.
- `rows.Err()` is checked after iteration.
- A failure processing one order does not prevent other scheduled jobs from continuing.

### 5. Reporting Job Error Handling

The failed-payment and daily-digest jobs now detect PostgreSQL failures rather than silently using zero values.

The failed-payment job checks the result of its count query.

The daily-digest job independently checks queries for:

- orders created today;
- pending orders;
- completed orders;
- completed-payment revenue.

The digest is not logged as successful if one of its required database queries fails.

### 6. Old Event Cleanup Error Handling

The old-event cleanup job now checks:

- the PostgreSQL delete operation;
- the result of `RowsAffected()`.

Cleanup is only logged after the database operation succeeds.

### 7. Migration Separation

The Scheduler Service does not own database migrations.

It consumes tables created and owned by other application services, including:

- `products`
- `reservations`
- `orders`
- `order_events`
- `payments`

No scheduler migration logic or `RUN_MIGRATIONS` behaviour was introduced as part of this hardening work.

### 8. Runtime Health

- `/livez` reports scheduler process liveness.
- `/healthz` checks PostgreSQL connectivity before reporting the scheduler as healthy.

## Runtime Configuration

### Required

`DATABASE_URL`

PostgreSQL connection string used by the Scheduler Service.

For example:

`postgres://user:password@host:5432/database?sslmode=disable`

### Optional

`HEALTH_PORT`

Port used for the scheduler health server. Defaults to `8091`.

## Scheduled Jobs

The existing scheduled-job architecture and intervals remain unchanged.

- `expire_reservations` — every 1 minute
- `abandoned_carts` — every 5 minutes
- `retry_failed_payments` — every 15 minutes
- `cleanup_old_events` — every 30 minutes
- `daily_digest` — every 1 hour

Each job also runs immediately when the scheduler starts.

## Validation and Testing

The scheduler hardening changes were validated locally using Go 1.22 and PostgreSQL 16. These were local integration tests and were not run against AWS or RDS.

### Go Build Test

Go 1.22 was verified using the `golang:1.22` Docker image, which reported Go `1.22.12`.

The scheduler was formatted successfully using `gofmt`.

`go build ./...` completed successfully with exit code `0`.

### Scheduler Startup Test

The Scheduler Service started successfully against the PostgreSQL test database.

All five scheduled jobs performed their immediate startup run.

### Health Probe Test

`/livez` responded successfully.

`/healthz` responded successfully while PostgreSQL was available.

### Reservation Expiry Test

An expired active reservation for order `9001` was created with quantity `2`.

Before processing, the associated product had a reserved quantity of `5`.

After the scheduler ran:

- the reservation status changed to `expired`;
- the product reserved quantity changed from `5` to `3`.

This confirmed the scheduler made the expected PostgreSQL side effects.

### Abandoned Order Test

Order `1001` was created in `pending` status with a creation time more than 30 minutes old.

After the scheduler ran, PostgreSQL showed the order status as `cancelled`.

### Old Event Cleanup Test

An `order_events` row more than 90 days old was created.

After the cleanup job ran, querying PostgreSQL returned zero matching old test events, confirming the row had been deleted.

### Failed Payment Reporting Test

A failed payment created within the previous hour was added to PostgreSQL.

The scheduler detected one failed payment eligible for retry and logged the result successfully.

### Daily Digest Test

Seeded order and payment data was used to exercise the daily digest.

The scheduler reported:

- 2 orders for the current day during the initial test;
- 1 pending order;
- 1 completed order;
- completed-payment revenue of `50.00`.

The digest completed successfully with the available PostgreSQL data.

### Reporting Database Failure Test

The `payments` table was temporarily made unavailable to simulate a PostgreSQL failure.

The scheduler correctly logged:

- `retry_failed_payments` as failed because the `payments` relation was unavailable;
- `daily_digest` as failed when its revenue query could not access the `payments` table.

At the same time:

- `cleanup_old_events` completed;
- `expire_reservations` completed;
- `abandoned_carts` completed.

This confirmed that a database failure in one scheduled job does not terminate or disrupt the other scheduled jobs.

### Reservation Transaction Rollback Test

A fresh expired reservation for order `9003` was created.

The `products` table was then temporarily made unavailable so the product update inside the reservation-expiry transaction would fail.

The scheduler logged the product update error and marked the `expire_reservations` job as failed.

A direct PostgreSQL query confirmed that reservation `9003` remained in `active` status.

This verified that the earlier reservation status update was rolled back when the corresponding product update failed.

Other scheduled jobs continued running successfully during this failure.

### Reservation Recovery Test

After restoring the `products` table, the scheduler was restarted.

Reservation `9003` was then processed successfully.

PostgreSQL confirmed:

- the reservation status changed to `expired`;
- the product reserved quantity was reduced to `2`.

This confirmed that a failed reservation remained available for processing after the database dependency recovered.

### Abandoned Order Update Failure Test

Order `1003` was created as an abandoned `pending` order.

A temporary PostgreSQL trigger was used to force updates to that order to fail.

The scheduler logged the simulated update error and marked the `abandoned_carts` job as failed.

A direct PostgreSQL query confirmed that order `1003` remained `pending`.

The scheduler therefore did not falsely persist or report the failed cancellation as successful.

The other scheduled jobs continued to complete normally.

### Post-Failure Health Test

After the scheduled-job failure tests, `/healthz` still responded successfully.

This confirmed that individual job failures did not terminate the scheduler or break its PostgreSQL health check.

## Future Improvements

The following improvements were deliberately left outside the scope of this ticket:

- Add automated unit and integration tests for scheduled-job success and failure paths.
- Add structured metrics for scheduled-job success, failure, and execution duration.
- Add per-job contexts and execution timeouts for database operations.
- Add explicit distributed coordination or locking if multiple scheduler replicas are expected to process the same jobs concurrently.

## Result

The Scheduler Service now detects and reports database, scan, update, transaction, and commit failures without incorrectly logging failed operations as successful.

Reservation expiry is transaction-safe, abandoned-order updates are verified before success is reported, reporting jobs surface database failures correctly, and scheduled-job failures remain isolated from one another.

The existing scheduler architecture and job intervals were preserved, and no infrastructure or migration responsibilities were added to the service.
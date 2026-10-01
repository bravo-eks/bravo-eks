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

The scheduler was built with Go 1.22 and tested locally against PostgreSQL 16.

### Go Build Test

The Go 1.22 build completed successfully.

### Scheduler Startup and Job Isolation Test

The Scheduler Service started successfully against PostgreSQL 16. All five scheduled jobs started and ran independently:

- `expire_reservations`
- `abandoned_carts`
- `retry_failed_payments`
- `daily_digest`
- `cleanup_old_events`

`/livez` returned HTTP 200. With the database available, `/healthz` returned HTTP 200.

### Database Outage and Recovery Test

PostgreSQL was stopped to simulate a database outage. `expire_reservations` correctly reported failure rather than success, while the Scheduler Service remained running. After PostgreSQL was restarted, `expire_reservations` succeeded automatically on its next run.

### Reservation Expiry Test

An expired active reservation for 3 units was created. The scheduler changed its status from `active` to `expired`; the product's reserved quantity changed from 3 to 0 while stock remained 10. Database queries confirmed both changes.

### Abandoned Order Test

A `pending` order older than 30 minutes was created. `abandoned_carts` detected it and changed its status to `cancelled`. Database queries confirmed the order change.

These tests verify safe database failure handling, job isolation and recovery, reservation transaction handling, and abandoned-order processing.

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
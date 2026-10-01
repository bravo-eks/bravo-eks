# Notification Service

The Notification Service manages application notifications using predefined templates and stores notification history in PostgreSQL. Email and SMS delivery are currently simulated; this service does not integrate AWS SES or SNS.

## Production Hardening Changes

### 1. Migration Separation

- Database migrations do not run automatically during normal application startup.
- Migrations and default notification template seeding run only when `RUN_MIGRATIONS=true`.
- Normal production runtime can start without attempting schema changes.

### 2. Migration and Template Seed Error Handling

- Migration database errors are checked.
- Errors while seeding default notification templates are handled.
- Startup fails clearly if an explicitly requested migration or template seed operation fails.

### 3. Notification Request Validation

Notification requests trim input strings and validate that `recipient` and `template` are present. The `channel` defaults to `email` when omitted; supported channels are `email` and `sms`. Invalid channels are rejected before database operations.

### 4. Missing Template Handling

- Template lookup distinguishes a missing template from a PostgreSQL/database failure.
- A missing template returns HTTP 404; database failures return a server error.
- The previous fallback that used a missing template name as the subject and body and reported it as sent has been removed.

### 5. Template Channel Validation

The selected template's channel must match the requested notification channel. For example, email templates must be used with `email`, and SMS templates with `sms`. A mismatch is rejected.

### 6. Notification Persistence Safety

- The notification insert is checked, and a successful response is returned only after PostgreSQL inserts the notification and returns its ID.
- Notification metadata encoding errors are handled.
- Database failures cannot be silently ignored while the API reports a notification as sent.

### 7. Database Read Error Handling

Notification history and template queries check query errors, individual `rows.Scan()` errors, and `rows.Err()` after iteration. Database operations use request contexts where appropriate, preventing partial or failed reads from being returned as successful API responses.

### 8. Health Checking

- `/livez` reports process liveness.
- `/healthz` checks PostgreSQL connectivity using the HTTP request context before reporting database-backed health.

### 9. Simulated Email and SMS Delivery

Email and SMS delivery remain intentionally simulated. Successful notifications are validated, use an existing template, are persisted to PostgreSQL, logged by the service, and returned with `status: sent`. No AWS SES or SNS integration was introduced.

## Runtime Configuration

### Required

`DATABASE_URL`

PostgreSQL connection string used by the Notification Service. For example: `postgres://user:password@host:5432/database?sslmode=disable`.

### Optional

`PORT`

HTTP service port. Defaults to `8084`.

`RUN_MIGRATIONS`

Controls whether database migrations and default template seeding run during startup. Defaults to `false`; set to `true` to run them explicitly.

## Validation and Testing

The production-hardening changes were validated locally using Go 1.22 and PostgreSQL 16. These tests were not run against AWS, RDS, SES, or SNS.

### Go Build Test

The service was formatted with `gofmt` and successfully compiled using Go 1.22 in Docker. Go formatting and `go build ./...` completed successfully.

### Health Probe Test

With PostgreSQL available, `GET /livez` and `GET /healthz` both returned HTTP 200, confirming process liveness and database-backed health checking.

### Template Query Test

After migrations and template seeding, `GET /templates` returned the expected seeded email and SMS templates:

- `order_confirmed`
- `order_shipped`
- `order_delivered`
- `payment_failed`
- `low_stock_alert`
- `order_confirmed_sms`
- `order_shipped_sms`

### Simulated Email Test

Submitted a notification for `customer@example.com` using the `email` channel and `order_confirmed` template. The request returned HTTP 201, PostgreSQL returned a notification ID, and the status was `sent`. Delivery was simulated; no real email was sent.

### Notification History Test

Queried notification history for `customer@example.com`. The previously created notification was returned with recipient `customer@example.com`, channel `email`, template `order_confirmed`, and status `sent`, confirming it was persisted in PostgreSQL.

### Simulated SMS Test

Submitted a notification using the `sms` channel and `order_confirmed_sms` template. The request returned HTTP 201 with status `sent`. Delivery was simulated; no real SMS was sent.

### Invalid Channel Test

A notification request using `channel: whatsapp` returned HTTP 400. The unsupported channel was rejected.

### Missing Template Test

A notification request using template `does-not-exist` returned HTTP 404 with `notification template not found`. No fallback notification was created using the missing template name.

### Missing Recipient Test

A notification request without a recipient returned HTTP 400 and was rejected before notification processing.

### Migration Separation Test

The service was restarted without `RUN_MIGRATIONS=true`. It started successfully, migrations and template seeding did not run, and existing PostgreSQL state remained available.

## Future Improvement

Real provider integrations could be considered in a future improvement: Amazon SES for email and Amazon SNS for SMS. They were deliberately left out of this work to preserve simulated delivery and focus on runtime and data-safety hardening.

## Result

The Notification Service strengthens runtime and PostgreSQL behavior while preserving simulated email and SMS delivery. It provides explicit migration control, stronger request validation, correct missing-template and channel handling, checked notification persistence, consistent query error handling, and database-backed health checking. Real notification-provider integration remains a future enhancement.

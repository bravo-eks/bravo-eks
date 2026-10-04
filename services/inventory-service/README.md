# Inventory Service

The Inventory Service manages product inventory, stock reservations, releases, and low-stock reporting. It uses PostgreSQL for persistent inventory and reservation state.

## Production Hardening Changes

### 1. Migration Separation

- Database migrations do not run automatically during normal application startup.
- Migrations run only when `RUN_MIGRATIONS=true`, allowing production runtime to start without attempting schema changes.

### 2. Product Validation

Product creation trims string inputs and validates that the product ID, name, and SKU are present, the price is greater than zero, and initial stock is not negative. Invalid product data is rejected before it is written to PostgreSQL.

### 3. Database Error Handling

- Product list and low-stock queries check `rows.Scan()` and `rows.Err()` errors.
- Product lookup distinguishes `sql.ErrNoRows` from database failures; database failures return server errors rather than appearing as missing products.
- Database operations use request contexts where appropriate, and internal PostgreSQL errors are logged rather than exposed directly to clients.

### 4. Reservation Validation

Reservation requests require an `order_id` greater than zero, at least one item, a `product_id` for each item, and a quantity greater than zero. This prevents zero or negative quantities from corrupting reserved inventory.

### 5. Inventory State Protection

- Before reserving stock, the service rejects negative stock, negative reserved stock, and reserved stock greater than total stock.
- Available inventory is calculated as `stock - reserved`.
- A reservation is rejected when its requested quantity exceeds available stock.

### 6. Concurrency-Safe Reservations

- Product inventory rows are locked with PostgreSQL `SELECT ... FOR UPDATE` while reservations are processed.
- The stock update includes a database condition preventing reserved stock from exceeding total stock.
- The affected row count is checked to confirm the update succeeded.

### 7. Transaction Safety

Reservation and release operations run in PostgreSQL transactions. The service checks transaction creation, queries and scans, inventory updates, reservation inserts, affected row counts, and transaction commits. Failed operations return without committing, leaving the transaction to roll back.

### 8. Safe Stock Release

- Release requests require a valid `order_id`; active reservation quantities are validated before inventory changes.
- Product rows are locked with `FOR UPDATE` before reserved stock is decreased.
- The service verifies that each product exists, reserved stock is sufficient, and the update affects the expected row. The SQL update also requires `reserved >= quantity`.
- Successfully released reservations are marked `released`; repeated release requests do not decrement inventory again.

### 9. Runtime Health

- `/livez` reports process liveness.
- `/healthz` checks PostgreSQL connectivity before reporting the service as healthy.

## Runtime Configuration

### Required

`DATABASE_URL`

PostgreSQL connection string used by the Inventory Service. For example: `postgres://user:password@host:5432/database?sslmode=disable`.

### Optional

`PORT`

HTTP service port. Defaults to `8082`.

`RUN_MIGRATIONS`

Controls whether database migrations run during startup. Defaults to `false`; set to `true` to explicitly run migrations.

## Validation and Testing

The production-hardening changes were validated locally using Go 1.22 and PostgreSQL 16. These were local integration tests, not tests against AWS or RDS.

### Go Build Test

The service was formatted with `gofmt` and successfully compiled using Go 1.22 in Docker. Go formatting completed successfully, and `go build ./...` completed successfully.

### Health Probe Test

With PostgreSQL available, `GET /livez` and `GET /healthz` both returned HTTP 200. The health response reported `inventory-service` status `ok`.

### Product Creation Test

Created product `prod-001` with SKU `TEST-001`, price `25.99`, and stock `10`. Product creation returned HTTP 201.

### Invalid Stock Test

A product creation request with stock `-5` returned HTTP 400. Negative initial inventory was rejected.

### Reservation Test

Order `1001` reserved 4 units of `prod-001`. Before reservation, stock was 10, reserved was 0, and available was 10. Reservation returned HTTP 201; afterward, stock was 10, reserved was 4, and available was 6.

### Over-Reservation Test

With 6 units available, order `1002` attempted to reserve 7 units. The request returned HTTP 409, and inventory remained at stock 10, reserved 4, and available 6. The failed transaction did not partially modify inventory.

### Invalid Quantity Test

A reservation with quantity `0` returned HTTP 400 and was rejected before inventory changed.

### Stock Release Test

The reservation for order `1001` was released with HTTP 200. Reserved stock returned from 4 to 0, and available stock returned from 6 to 10. Final inventory was stock 10, reserved 0, and available 10.

### Repeated Release Test

Releasing order `1001` again returned HTTP 200 with `released: 0`. Reserved stock remained 0 and inventory was not decremented again.

### Migration Separation Test

The service was restarted without `RUN_MIGRATIONS=true`. It started successfully, migrations did not run during normal startup, and existing PostgreSQL state remained available.

## Result

The Inventory Service strengthens product and reservation validation, PostgreSQL error handling, transaction safety, and concurrency protection while preserving its existing API and PostgreSQL inventory model. It also supports explicit migration control, safe repeated releases, and database-backed health checks, preparing the service for production EKS deployment without coupling migrations to normal runtime.

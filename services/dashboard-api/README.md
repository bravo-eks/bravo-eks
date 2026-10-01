# Dashboard API

The Dashboard API provides read-only dashboard data for the order fulfilment platform. It exposes dashboard summaries, order statistics, revenue information, inventory alerts, and shipping metrics.

## Database Error Handling Changes

1. **Database Query Error Handling**

   * Database query errors are now checked throughout all dashboard handlers.
   * Failed `Query()` operations return HTTP 500 instead of silently continuing.
   * Failed `QueryRow()` operations return HTTP 500 instead of returning potentially misleading zero values.
   * Database failures are no longer hidden from API consumers.

2. **Scan Error Handling**

   * `Scan()` errors are no longer ignored.
   * Errors while scanning individual database results return HTTP 500.
   * This prevents malformed or failed database results from being returned as valid dashboard data.

3. **Rows Error Handling**

   * `rows.Err()` is checked after iterating through database results.
   * Errors encountered while reading result sets are returned as HTTP 500 responses.
   * This applies to order statistics, inventory alerts, revenue data, and shipping data.

4. **Dashboard Summary Error Handling**

   * All database operations used by `/dashboard/summary` now check for errors.
   * Order counts, revenue calculations, product counts, low-stock counts, and shipment counts all handle database failures.
   * Status aggregation errors are also handled.
   * The endpoint no longer returns a partially populated dashboard when a database operation fails.

5. **Revenue Error Handling**

   * Total revenue and refund queries now check `Scan()` errors.
   * Daily revenue query failures return HTTP 500.
   * Daily revenue scan and row iteration errors are handled.
   * Failed revenue queries no longer result in zero-value financial data being returned.

6. **Shipping Error Handling**

   * Shipment status query errors are handled.
   * Carrier breakdown query errors are handled.
   * Average delivery time query errors are handled.
   * Scan and result iteration errors are also checked.

7. **Read-Only Behaviour Preserved**

   * The Dashboard API remains read-only.
   * No INSERT, UPDATE, DELETE, or other database write operations were introduced.
   * Existing dashboard SQL queries and business logic are preserved.

## Runtime Configuration

### Required

* `DATABASE_URL`

The application fails to start if `DATABASE_URL` is not configured.

### Optional

* `PORT` (defaults to `8086`)

`DATABASE_URL` is expected to be injected at runtime rather than stored in source code.

## Endpoints

### Health and Liveness

* `GET /livez`
* `GET /healthz`
* `GET /dashboard/healthz`

### Dashboard Data

* `GET /dashboard/summary`
* `GET /dashboard/orders/stats`
* `GET /dashboard/revenue`
* `GET /dashboard/inventory/alerts`
* `GET /dashboard/shipping/overview`

All dashboard endpoints remain read-only.

## Error Handling

Database failures return an HTTP 500 response using the existing JSON error format:

```json
{
  "error": "query failed"
}
```

This ensures database failures are reported to the caller instead of returning misleading empty or zero-valued dashboard data.

## Validation and Testing

### Formatting

The updated Go source should be formatted using:

```sh
gofmt -w main.go
```

### Tests

Run the Dashboard API tests using:

```sh
go test ./...
```

### Build

Build the service using:

```sh
go build .
```

### Database Configuration Test

Run the service without `DATABASE_URL`:

```sh
go run .
```

Expected behaviour:

```text
DATABASE_URL is required
```

The application should fail during startup because the database configuration is required.

### Database Error Handling Test

With the application running, make a request to a dashboard endpoint while the database is unavailable.

For example:

```sh
GET /dashboard/summary
```

Expected behaviour:

```text
HTTP 500
{
  "error": "query failed"
}
```

The API should return an HTTP 500 response rather than returning zero values or an incomplete dashboard response.

### Read-Only Validation

The Dashboard API should continue to perform only database reads.

No write operations were introduced as part of BRAVO-108.

## Result

The Dashboard API now handles database failures consistently across its dashboard endpoints. `Query()`, `QueryRow().Scan()`, `rows.Scan()`, and `rows.Err()` failures are handled instead of being ignored, preventing failed database operations from being presented as valid zero-valued dashboard data.

The existing endpoints, `DATABASE_URL` requirement, SQL/business logic, and read-only behaviour remain unchanged.

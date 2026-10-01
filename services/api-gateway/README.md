# API Gateway

The API Gateway is the entry point for application traffic. It handles routing, JWT authentication, Redis-backed rate limiting, and proxying requests to downstream services.

## Production Hardening Changes

1. **JWT Secret Validation**
   - Removed the insecure `change-me-in-production` fallback.
   - `JWT_SECRET` is required.
   - The application fails fast when `JWT_SECRET` is missing.
   - This prevents the gateway from accidentally running with a known signing secret.

2. **Downstream Service URL Validation**
   - Downstream service URLs are validated during application startup.
   - Invalid upstream URLs cause startup to fail with a clear error.
   - This catches configuration problems before the gateway receives traffic.

3. **Redis Configuration**
   - `REDIS_URL` is required because Redis is a required dependency for rate limiting.
   - Invalid `REDIS_URL` values cause startup failure.
   - Redis connectivity is checked during startup.
   - The gateway fails startup if Redis cannot be reached.

4. **Redis Rate Limiting Error Handling**
   - Redis `INCR` and expiry errors are no longer ignored.
   - Runtime Redis failures return HTTP 503 instead of silently bypassing rate limiting.

5. **Health and Liveness**
   - `/livez` checks whether the API Gateway process is alive.
   - `/healthz` checks application readiness, including Redis connectivity.
   - The Redis health check uses a timeout.
   - `/healthz` returns HTTP 503 when Redis is unavailable.
   - These endpoints support separate Kubernetes liveness and readiness probes.

6. **JWT Algorithm Validation**
   - JWT generation uses HS256.
   - JWT validation explicitly accepts HS256 rather than any HMAC signing algorithm.

7. **Login Input Validation**
   - Login requires non-empty email and password values.
   - Invalid login input returns HTTP 400.
   - Authentication remains intentionally simulated for this project.

8. **Reverse Proxy Handling**
   - Manual `X-Forwarded-For` overwriting was removed.
   - Go's reverse proxy handles forwarding information appropriately.
   - Existing `X-Request-ID` behavior remains.

9. **Graceful Shutdown**
   - HTTP server shutdown errors are handled.
   - Redis connections are explicitly closed during shutdown.
   - This supports clean termination during SIGTERM and Kubernetes pod termination.

## Runtime Configuration

### Required

- `JWT_SECRET`
- `REDIS_URL`

### Configurable Downstream Service URLs

- `ORDER_SERVICE_URL`
- `INVENTORY_SERVICE_URL`
- `PAYMENT_SERVICE_URL`
- `NOTIFICATION_SERVICE_URL`
- `SHIPPING_SERVICE_URL`
- `DASHBOARD_SERVICE_URL`

### Optional

- `PORT` (defaults to `8080`)

Secrets such as `JWT_SECRET` are expected to be injected at runtime rather than stored in source code.

## Validation and Testing

### Docker Build

The service was built successfully using:

```sh
docker build -t bravo-api-gateway-test ./services/api-gateway
```

This also confirmed that the updated Go application compiles successfully in the project's container build environment.

### Missing JWT Secret

Test:

```sh
docker run --rm bravo-api-gateway-test
```

Observed expected behavior:

```text
JWT_SECRET is required
```

### Missing Redis Configuration

Test:

```sh
docker run --rm \
  -e JWT_SECRET=test-secret \
  bravo-api-gateway-test
```

Expected/observed behavior:

```text
REDIS_URL is required
```

### Redis Integration Test

A temporary Redis container was started and the API Gateway was run with `JWT_SECRET` and `REDIS_URL` configured. The gateway successfully connected to Redis and started on port 8080.

### Liveness Test

`GET /livez`

Expected:

```text
HTTP 200
```

### Readiness Test

`GET /healthz` while Redis is available.

Expected:

```text
HTTP 200
{"service":"api-gateway","status":"ok"}
```

### Login Validation Test

`POST /auth/login` with empty email/password.

Expected:

```text
HTTP 400
{"error":"email and password required"}
```

`POST /auth/login` with test credentials.

Expected: HTTP 200 and a JWT.

### Redis Failure Test

Redis was stopped while the API Gateway remained running.

`GET /healthz`

Expected: HTTP 503

`GET /livez`

Expected: HTTP 200

This confirms that the distinction between liveness and readiness works as intended.

## Result

The API Gateway now fails fast on invalid required configuration, handles Redis failures safely, has improved JWT validation, reports meaningful readiness state, and shuts down cleanly.

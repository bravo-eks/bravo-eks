# 12 — RDS database connectivity

## 1. Purpose

Use this runbook when a Bravo service that uses RDS cannot connect to
PostgreSQL or repeatedly reports a database readiness error. The Dashboard API
is the worked example throughout:
commands and results below use the resources that were actually investigated.
For another service, substitute its Deployment, ExternalSecret, Kubernetes
Secret, database user, schema, and tables; do not assume the Dashboard API's
resources or permissions apply.

## 2. Symptoms

The Dashboard API worked example showed repeated log messages:

```text
Waiting for database...
```

## 3. Expected architecture

- One AWS RDS PostgreSQL instance hosts the `bravodev` application database.
- Services use separate PostgreSQL users and schemas.
- The Dashboard API connects as the read-only cross-service user
  `dashboard_user` and reads data from `orders`, `inventory`, `payments`, and
  `shipping`.
- AWS Secrets Manager stores the connection information. External Secrets
  Operator syncs it to the `dashboard-api-database` Kubernetes Secret in the
  `microservices` namespace.
- The `dashboard-api` Helm release injects `DATABASE_URL` into its
  `dashboard-api-bravo-app` Deployment from that Secret.

Never print or expose `DATABASE_URL`, passwords, the RDS master password, or
other secret values. Apply only permissions appropriate for the service being
troubleshot.

## 4. Diagnostic procedure

Run these checks in order. Stop when a check fails and resolve that layer
before proceeding.

### 4.1 ExternalSecret sync

Check whether External Secrets Operator has synced the Dashboard API
ExternalSecret. For another service, inspect that service's ExternalSecret.

```bash
kubectl describe externalsecret dashboard-api-bravo-app -n microservices
```

Healthy result: `SecretSynced=True` with no sync error. If sync has failed,
resolve the ExternalSecret or secret-store issue before checking the
Kubernetes Secret. In the Dashboard API worked example, External Secrets
Operator was working.

### 4.2 Kubernetes Secret exists

Check that the synced Dashboard API Secret exists. For another service,
substitute its Kubernetes Secret name and namespace.

```bash
kubectl get secret dashboard-api-database -n microservices
```

Healthy result: the command finds `dashboard-api-database`. If it is missing,
return to the ExternalSecret sync check. Do not use `-o yaml`, decode `.data`,
or otherwise display its values.
In the Dashboard API worked example, `dashboard-api-database` existed.

### 4.3 Environment injection

Check that the Dashboard API container received `DATABASE_URL` without
printing its value. For another service, target its Deployment and verify the
database URL variable it uses.

```bash
kubectl exec deployment/dashboard-api-bravo-app -n microservices -- \
  sh -c 'test -n "$DATABASE_URL" && echo "DATABASE_URL is set"'
```

Healthy result: `DATABASE_URL is set`. If the check fails, inspect the
Deployment's Secret reference and pod events; do not print the variable value.
In the Dashboard API worked example, `DATABASE_URL` was injected.

### 4.4 RDS network connectivity

Check TCP connectivity from a Dashboard API pod to the RDS endpoint on port
5432. For another service, run the equivalent check from one of its pods to
its RDS endpoint.

```bash
kubectl exec deployment/dashboard-api-bravo-app -n microservices -- \
  nc -vz <rds-endpoint> 5432
```

Healthy result: the connection succeeds. This rules out basic pod-to-RDS
connectivity, including VPC routing, RDS security groups, and Kubernetes
NetworkPolicy. If it fails, investigate those network controls before
proceeding. In the Dashboard API worked example, TCP connectivity succeeded.

### 4.5 PostgreSQL authentication

Check PostgreSQL authentication. The Dashboard API image does not include
`psql`, so create a temporary `postgres:17-alpine` pod that imports the
`dashboard-api-database` Secret as environment variables without displaying
them. For another service, use its Secret and expected database identity; do
not assume the Dashboard API's identity or create generic commands for an
untested service:

```bash
kubectl run postgres-diagnostic -n microservices \
  --image=postgres:17-alpine \
  --restart=Never \
  --overrides='{"spec":{"containers":[{"name":"postgres-diagnostic","image":"postgres:17-alpine","command":["sleep","3600"],"envFrom":[{"secretRef":{"name":"dashboard-api-database"}}]}]}}'
```

Open a shell in the pod and connect using the injected URL:

```bash
kubectl exec -it postgres-diagnostic -n microservices -- sh
psql "$DATABASE_URL"
```

At the `psql` prompt, verify the authenticated user, database, and server
version:

```sql
SELECT current_user, current_database(), version();
```

Healthy result: user `dashboard_user`, database `bravodev`, and PostgreSQL 17.
Successful connection confirms authentication. If connection fails, diagnose
the PostgreSQL authentication or connection configuration. If the connected
user, database, or server version differs, verify that the intended connection
target is being used. Never paste or display the connection URL or password.
In the Dashboard API worked example, `dashboard_user` authenticated to
`bravodev` on PostgreSQL 17.

### 4.6 Schemas and tables

As `dashboard_user`, check whether the Dashboard API's required tables are
visible. For another service, query its required schemas and tables instead:

```sql
SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_schema IN ('orders', 'inventory', 'payments', 'shipping')
  AND table_type = 'BASE TABLE'
ORDER BY table_schema, table_name;
```

Healthy result: the query returns the required tables in `orders`,
`inventory`, `payments`, and `shipping`. No rows means those tables are not
visible to this user; continue to the privilege check to distinguish missing
schema access from missing table access.
In the Dashboard API worked example, the required schemas existed.

### 4.7 PostgreSQL privileges

As `dashboard_user`, check `USAGE` on each required schema and `SELECT` on a
representative table from each. For another service, check the schemas and
tables that service is expected to access:

```sql
SELECT
  has_schema_privilege(current_user, 'orders', 'USAGE') AS orders_usage,
  has_schema_privilege(current_user, 'inventory', 'USAGE') AS inventory_usage,
  has_schema_privilege(current_user, 'payments', 'USAGE') AS payments_usage,
  has_schema_privilege(current_user, 'shipping', 'USAGE') AS shipping_usage,
  has_table_privilege(current_user, 'orders.orders', 'SELECT') AS orders_select,
  has_table_privilege(current_user, 'inventory.products', 'SELECT') AS products_select,
  has_table_privilege(current_user, 'payments.payments', 'SELECT') AS payments_select,
  has_table_privilege(current_user, 'shipping.shipments', 'SELECT') AS shipments_select;
```

Healthy result: all values are `true`. A false schema `USAGE` indicates a
schema privilege issue. If schema `USAGE` is true but table `SELECT` is false,
the likely issue is missing table-level read privileges; see the known issue
below.

In the Dashboard API worked example, schema `USAGE` was present and
table-level `SELECT` was missing.

## 5. Known issue: missing table SELECT privileges

The known failure condition was that `dashboard_user` could reach RDS and
authenticate to `bravodev`, and had schema `USAGE`, but lacked table `SELECT`.
If the checks show this same combination, PostgreSQL table authorization is
the likely cause—not networking, secret delivery, or authentication.

## 6. Resolution for the Dashboard API missing SELECT privileges

The following grants are the verified resolution for the Dashboard API
authorization issue described above. They are not a generic fix for another
service. For any other service, first diagnose the failure and consult the
[RDS permission matrix](../architecture/rds-permission-matrix.md) to determine
its intended permissions; do not apply these grants automatically.

For the Dashboard API condition, use an authorized database administrator or
table owner connection to grant `SELECT` on existing tables in its four
schemas:

```sql
GRANT SELECT ON ALL TABLES IN SCHEMA orders TO dashboard_user;
GRANT SELECT ON ALL TABLES IN SCHEMA inventory TO dashboard_user;
GRANT SELECT ON ALL TABLES IN SCHEMA payments TO dashboard_user;
GRANT SELECT ON ALL TABLES IN SCHEMA shipping TO dashboard_user;
```

These grants provide read access only. Preserve the read-only permission
model: do not grant `INSERT`, `UPDATE`, `DELETE`, `CREATE`, or ownership
privileges. Do not run these grants as the Dashboard API user.

## 7. Verification

For the Dashboard API, reconnect as `dashboard_user` and rerun the privilege
query in section 4.7.
Confirm `SELECT` returns `true` for the representative tables in all four
schemas and that the required tables appear in the query from section 4.6.
For another service, verify only the permissions it is meant to have according
to the [RDS permission matrix](../architecture/rds-permission-matrix.md).

Restart the Dashboard API Deployment so the application reconnects, then wait
for rollout completion. For another service, use its Deployment and expected
replica count:

```bash
kubectl rollout restart deployment/dashboard-api-bravo-app -n microservices
kubectl rollout status deployment/dashboard-api-bravo-app -n microservices
```

Check that both replicas are ready and have not restarted:

```bash
kubectl get pods -n microservices \
  -l app.kubernetes.io/instance=dashboard-api \
  -o custom-columns='NAME:.metadata.name,READY:.status.containerStatuses[*].ready,RESTARTS:.status.containerStatuses[*].restartCount,PHASE:.status.phase'
```

Healthy result: both Dashboard API pods are `Running` and ready, with `0`
restarts. Check recent logs to confirm the waiting behaviour has stopped:

```bash
kubectl logs -n microservices \
  -l app.kubernetes.io/instance=dashboard-api \
  --all-containers=true --since=10m
```

If the message continues, review the logs and repeat the diagnostic checks;
do not assume a grant resolved a different failure layer.

## 8. Known limitation / prevention

`GRANT SELECT ON ALL TABLES` applies only to tables that exist when the grant
is run; it does not automatically grant access to tables created later. When a
migration adds tables, verify that the affected service has its intended
access. Have the relevant table-owning role configure appropriate default
privileges if ongoing access is required. Keep grants scoped to the service's
intended schemas and preserve its required permission model.

See the [RDS permission matrix](../architecture/rds-permission-matrix.md) for
the service ownership and intended runtime permissions.

## 9. Cleanup

Remove the temporary Dashboard API diagnostic pod after testing:

```bash
kubectl delete pod postgres-diagnostic -n microservices --ignore-not-found
```

## 10. Related runbooks

[15 — AWS identity and External Secrets](15-aws-identity-secrets.md) · [14 — CI/CD and migrations](14-cicd-gitops-troubleshooting.md) · [18 — Incident history](18-incident-history-and-triage.md)

# Secrets Management and Security Flow

Bravo EKS centralizes runtime credentials, database credentials, and shared secrets in AWS Secrets Manager, with External Secrets Operator securely delivering the required values to Kubernetes workloads.

```text
AWS Secrets Manager → External Secrets Operator → Kubernetes Secret → Workload
```

This approach keeps sensitive values out of application code, Helm values, and source control. Terraform provisions the AWS secret resources and IAM permissions, while Kubernetes workloads reference only the secrets they require.

## AWS Secrets Manager

AWS Secrets Manager acts as the central store for application and database credentials across Bravo EKS.

Managed secrets include:

- RDS master credentials
- environment-specific application runtime secrets
- JWT configuration
- Redis connection details
- service-specific database credentials and connection values

Access to these secrets is controlled through IAM. The External Secrets Operator role is granted access to the configured secret ARNs, providing a least-privilege boundary rather than broad access across Secrets Manager.

This establishes a clear separation between **secret storage**, **secret access**, and **application consumption**.

## External Secrets Operator

External Secrets Operator (ESO) provides the integration between AWS Secrets Manager and Kubernetes.

The platform defines a `ClusterSecretStore` named `aws-secrets-manager`, configured for AWS Secrets Manager in `eu-west-2`. Authentication is handled through IRSA, allowing ESO to access AWS without static AWS credentials stored inside the cluster.

The shared microservice Helm chart includes a reusable `ExternalSecret` template. Individual services define the remote secret properties they require, and ESO synchronizes those values into service-specific Kubernetes `Secret` resources.

Examples include:

- `JWT_SECRET`
- `REDIS_URL`
- database credentials and connection values
- service-specific runtime configuration

Helm controls **which secret properties a workload receives**, while the sensitive values themselves remain centrally managed in AWS Secrets Manager.

## IRSA and Workload Identity

Bravo EKS uses IAM Roles for Service Accounts (IRSA) to provide AWS permissions to Kubernetes workloads and platform controllers.

IRSA is used by components including:

- **External Secrets Operator** — reads approved values from AWS Secrets Manager
- **ExternalDNS** — manages Route53 DNS records
- **Velero** — accesses S3 backup storage and AWS snapshot APIs
- **KEDA** — reads SQS queue attributes for event-driven autoscaling
- **application workloads** — receive AWS permissions appropriate to their service responsibilities

IAM roles are associated with specific Kubernetes service accounts through OIDC trust relationships and restrictive subject conditions.

This allows AWS permissions to follow Kubernetes workload identity without distributing long-lived AWS access keys inside the cluster.

## Kubernetes Secret Injection

The shared Helm chart integrates secret delivery directly into application deployment.

When enabled for a service, the chart renders an `ExternalSecret`. ESO retrieves the configured properties from AWS Secrets Manager and creates or refreshes the corresponding Kubernetes `Secret`.

The workload then consumes the resulting secret through `envFrom` or `secretKeyRef`.

For example, API Gateway receives runtime values including:

- `JWT_SECRET`
- `REDIS_URL`

The end-to-end flow is:

1. Terraform provisions or references the secret in AWS Secrets Manager.
2. IAM and IRSA authorize External Secrets Operator to retrieve the approved secret.
3. ESO synchronizes the required properties into a Kubernetes `Secret`.
4. Helm-configured workloads consume the secret through environment-variable references.
5. ESO refreshes the Kubernetes secret from the authoritative AWS source.

This creates a repeatable secret-delivery model shared across the microservices.

## Database and Service Credentials

Database credentials follow the same centralized secret-management pattern.

Amazon RDS uses AWS-managed master credentials through Secrets Manager, while application workloads receive service-specific database connection details.

Bravo EKS uses separate PostgreSQL identities for its data-owning services, allowing database access to follow the same service boundaries used throughout the application architecture.

This supports:

- service-specific database credentials
- schema-level access boundaries
- least-privilege database permissions
- controlled cross-schema access for services such as Scheduler and Dashboard API
- independent credential delivery to each workload

The detailed database authorization model is documented in [RDS Permission Matrix](rds-permission-matrix.md).

Redis connection details are also delivered through the secrets-management path, allowing API Gateway to establish its rate-limiting dependency without embedding Redis credentials in source control.

## JWT and Redis Configuration

API Gateway demonstrates the complete runtime secret flow.

It requires:

- `JWT_SECRET` for token handling
- `REDIS_URL` for the Redis-backed rate limiter

Both values are sourced from the environment-specific runtime secret and injected into the API Gateway workload.

This keeps authentication and infrastructure credentials separate from application code while ensuring required runtime configuration is available when the service starts.

## DEV and PROD Isolation

Secrets are separated between DEV and PROD.

- **DEV** uses the `bravo-dev-runtime` secret path and DEV-specific configuration.
- **PROD** uses `bravo/prod/runtime` and production-scoped secret ARNs.

The same secret-management architecture is reusable across both environments while credentials remain isolated within their respective AWS environments.

This ensures development workloads cannot share or consume production credentials through the normal platform secret path.

## Encryption and Secure Handling

Secret management is supported by encryption and access controls across the platform:

- EKS secrets are protected with KMS-backed encryption.
- RDS storage is encrypted at rest.
- SQS uses server-side encryption.
- Velero backup storage is encrypted.
- Terraform state and relevant S3 resources use storage encryption and public-access blocking.
- AWS permissions are delivered through IAM roles and OIDC/IRSA rather than long-lived access keys.
- Runtime secret values originate from AWS Secrets Manager rather than source-controlled configuration.
- Kubernetes Secrets are generated and refreshed through External Secrets Operator instead of being manually maintained in application manifests.

Together, these controls protect sensitive configuration from storage through to workload delivery.

## Secret Refresh and Rotation

The shared service chart configures `ExternalSecret` resources with a refresh interval, allowing Kubernetes secrets to be synchronized with their authoritative AWS Secrets Manager values.

When an upstream secret changes, External Secrets Operator refreshes the corresponding Kubernetes `Secret` during its synchronization cycle.

This provides a consistent operational model:

```text
Secret updated in AWS
        ↓
External Secrets Operator refresh
        ↓
Kubernetes Secret updated
        ↓
Workload receives managed runtime configuration
```

Centralizing secrets in AWS Secrets Manager means credential lifecycle management remains separated from application manifests and deployment configuration.

## Security Architecture Summary

Bravo EKS combines AWS-native identity and secret management with Kubernetes-native workload delivery:

```text
                    AWS IAM / OIDC
                         │
                         ▼
                Kubernetes ServiceAccount
                         │
                        IRSA
                         │
                         ▼
AWS Secrets Manager ──► External Secrets Operator
                         │
                         ▼
                  Kubernetes Secret
                         │
                         ▼
                 Application Workload
```

The result is a consistent security model across the platform: secrets remain centralized in AWS, AWS access is tied to workload identity, environments remain isolated, and applications receive only the runtime configuration required for their responsibilities.

## Related Documentation

- [AWS Infrastructure & Terraform](aws-infrastructure.md)
- [Data & Messaging](data-and-messaging.md)
- [RDS Permission Matrix](rds-permission-matrix.md)
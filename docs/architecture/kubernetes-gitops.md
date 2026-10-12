## Kubernetes and GitOps Architecture

Bravo EKS uses Kubernetes as the application runtime and Argo CD as the GitOps delivery layer. Application workloads, platform components, Helm configuration, and environment-specific values are maintained in Git and reconciled into the EKS clusters through Argo CD.

The delivery model is:

```text
Git
 │
 ├── Application source
 ├── Helm charts
 ├── Environment values
 └── Platform configuration
          │
          ▼
   GitHub Actions
   Build / Validate
          │
          ▼
       Argo CD
          │
          ▼
     Amazon EKS
          │
   ┌──────┴───────┐
   ▼              ▼
Applications    Platform
```

This provides a consistent deployment model across DEV and PROD while allowing each environment to use its own operational policies and configuration.

## Workloads and Namespaces

The Kubernetes architecture separates application workloads from platform controllers through dedicated namespaces.

| Namespace | Role |
|---|---|
| `microservices` | Runtime namespace for the nine Go services |
| `redis` | Redis stateful service used for API Gateway rate limiting |
| `traefik` | Traefik Gateway and ingress controller |
| `cert-manager` | TLS certificate automation |
| `external-dns` | Route53 DNS record management |
| `external-secrets` | AWS Secrets Manager synchronization |
| `karpenter` | Karpenter controller and node provisioning |
| `keda` | KEDA controller and event-driven autoscaling |
| `monitoring` | Prometheus, Alertmanager, and Grafana |
| `velero` | Backup and restore controller |
| `argocd` | Argo CD control plane |
| `kube-system` | Kubernetes and EKS system components |

The nine application services run in the `microservices` namespace:

- `api-gateway`
- `dashboard-api`
- `inventory-service`
- `notification-service`
- `order-service`
- `payment-service`
- `scheduler`
- `shipping-service`
- `worker`

Redis is deployed separately as a stateful workload in the `redis` namespace.

## GitOps Repository Structure

Application and platform configuration are maintained under the Kubernetes GitOps tree.

The application workloads use a shared Helm chart:

```text
kubernetes/helm/microservices
```

Environment-specific service configuration is maintained separately:

```text
kubernetes/helm/service-values-dev
kubernetes/helm/service-values-prod
```

Argo CD configuration is also separated by environment:

```text
kubernetes/argocd/bootstrap-dev
kubernetes/argocd/bootstrap-prod

kubernetes/argocd/applications-dev
kubernetes/argocd/applications-prod
```

This allows the same deployment architecture to be reused across environments while preserving independent configuration and release controls.

## Argo CD Bootstrap

Argo CD is bootstrapped independently for DEV and PROD.

The environment root applications are:

```text
kubernetes/argocd/bootstrap-dev/root-app.yaml
kubernetes/argocd/bootstrap-prod/root-app.yaml
```

The DEV root application targets:

```text
kubernetes/argocd/applications-dev/app-groups
```

The PROD root application targets:

```text
kubernetes/argocd/applications-prod/app-groups
```

The bootstrap application uses the `bravo-bootstrap` Argo CD project and establishes the root of the environment's application hierarchy.

From that point, the App-of-Apps pattern allows Argo CD to manage the remaining application and platform resources.

```text
                    Root Application
                           │
                           ▼
                    Argo CD App Groups
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
           Projects       Apps       Platform
                            │            │
                     Microservices   Controllers
                            │            │
                            ▼            ▼
                       Kubernetes      EKS Platform
```

This gives Bravo EKS a structured GitOps hierarchy rather than requiring each Kubernetes workload to be deployed independently.

## Argo CD Projects

Argo CD projects provide separation between the major deployment areas.

The project structure includes:

- `bravo-bootstrap` — root application and application-group bootstrap
- `bravo-apps` — application workloads in the `microservices` namespace
- `bravo-platform` — cluster platform controllers and services
- `bravo-redis` — Redis stateful deployment

The project definitions control the permitted Git repositories, cluster destinations, and namespaces available to each application group.

This provides clear boundaries between bootstrap configuration, application workloads, platform services, and stateful supporting components.

## Application Groups

Each environment contains an `app-groups` layer used by the root application.

The structure includes:

- `apps.yaml` — application workloads
- `platform.yaml` — platform services
- `projects.yaml` — Argo CD project configuration

These groups form the bridge between the root application and the individual Argo CD Applications.

```text
root-app
   │
   └── app-groups
         │
         ├── projects
         │
         ├── applications
         │     ├── api-gateway
         │     ├── dashboard-api
         │     ├── inventory-service
         │     ├── notification-service
         │     ├── order-service
         │     ├── payment-service
         │     ├── scheduler
         │     ├── shipping-service
         │     └── worker
         │
         ├── redis
         │
         └── platform
               ├── Traefik
               ├── cert-manager
               ├── ExternalDNS
               ├── External Secrets
               ├── Karpenter
               ├── KEDA
               ├── monitoring
               └── Velero
```

Each application service references the shared Helm chart together with its environment-specific values file.

## Shared Microservice Helm Chart

Bravo EKS uses a reusable Helm chart rather than maintaining a separate Kubernetes manifest set for every microservice.

The shared chart at:

```text
kubernetes/helm/microservices
```

provides common Kubernetes resources including:

- `Deployment`
- `Service`
- `ServiceAccount`
- `HorizontalPodAutoscaler`
- `NetworkPolicy`
- `PodDisruptionBudget`
- `HTTPRoute`
- `ExternalSecret`
- KEDA `ScaledObject`
- migration Job support

The chart also standardizes common workload behaviour including:

- container ports
- startup probes
- readiness probes
- liveness probes
- CPU and memory requests
- CPU and memory limits
- rolling deployment strategy
- topology spread
- affinity
- node selection
- service accounts
- secret injection
- network policy
- autoscaling

This keeps Kubernetes behaviour consistent across the nine Go services while allowing each service to define only the configuration that differs.

## Environment-Specific Service Values

Service configuration is separated between:

```text
kubernetes/helm/service-values-dev
kubernetes/helm/service-values-prod
```

Each service values file can define:

- ECR image repository and image tag
- application port
- Kubernetes Service configuration
- environment variables
- secret references
- `ExternalSecret` mappings
- CPU and memory resources
- startup, readiness, and liveness probes
- Gateway API routing
- HPA configuration
- PodDisruptionBudget configuration
- KEDA configuration
- workload placement

This separates reusable Kubernetes logic from service and environment configuration.

The resulting deployment pattern is:

```text
Shared Helm Chart
        +
Service Configuration
        +
Environment Configuration
        │
        ▼
Rendered Kubernetes Resources
```

## DEV and PROD GitOps

DEV and PROD share the same GitOps architecture while using environment-specific operational policies.

### DEV

DEV uses automated Argo CD reconciliation with:

- `prune: true`
- `selfHeal: true`

This allows the development cluster to converge automatically toward the desired state stored in Git.

DEV image publication is also integrated with the main development workflow, supporting rapid iteration and validation.

### PROD

PROD uses the same Helm and App-of-Apps architecture with a controlled manual synchronization model.

Production changes therefore remain Git-defined while deployment execution is explicitly controlled.

This creates a clear environment progression:

```text
Code / Configuration Change
           │
           ▼
          DEV
   Automated reconciliation
           │
           ▼
       Validation
           │
           ▼
         PROD
   Controlled reconciliation
```

Both environments retain Git as the authoritative definition of the intended cluster state.

## GitOps Reconciliation

Argo CD continuously compares the desired configuration in Git with the resources deployed to Kubernetes.

For DEV, automated synchronization allows Argo CD to reconcile configuration drift and apply approved Git changes automatically.

The model is:

```text
Git desired state
       │
       ▼
    Argo CD
       │
       ├── Compare
       ├── Render Helm
       ├── Apply
       └── Reconcile
              │
              ▼
          EKS Cluster
```

This provides a declarative deployment model where Kubernetes configuration is version-controlled, reviewable, and reproducible from Git.

## Deployment and Rollback

Because application configuration and image references are stored in Git, deployment history is tied directly to repository history.

A standard deployment follows:

```text
Git change
    ↓
Validation
    ↓
Argo CD
    ↓
Helm render
    ↓
Kubernetes rollout
    ↓
Health checks
```

Rollback follows the same declarative model.

A previous Git configuration or known-good image reference can be restored, after which Argo CD reconciles the cluster back to the desired version.

This keeps deployment and rollback aligned with the same Git-controlled workflow.

## Gateway API and Traffic Routing

Bravo EKS uses Traefik with the Kubernetes Gateway API for application ingress.

Traefik is configured through:

```text
kubernetes/platform/traefik/values.yaml
```

The platform enables the Kubernetes Gateway provider and creates a Gateway with HTTP and HTTPS listeners.

Application routes are generated through the shared Helm chart using `HTTPRoute` resources.

The public request flow is:

```text
Internet
   │
   ▼
Route53
   │
   ▼
AWS Load Balancer
   │
   ▼
Traefik Gateway
   │
   ▼
HTTPRoute
   │
   ▼
Kubernetes Service
   │
   ▼
Application Pods
```

Gateway API provides a clear separation between the shared ingress infrastructure and the routing rules owned by individual workloads.

## TLS and Certificate Automation

TLS is integrated into the Kubernetes platform through cert-manager.

The platform includes:

- cert-manager deployment
- `ClusterIssuer` configuration
- certificate resources
- the `bravo-tls` Kubernetes TLS secret
- HTTPS Gateway listener configuration

The certificate flow is:

```text
Certificate Authority
        │
        ▼
    cert-manager
        │
        ▼
 Kubernetes TLS Secret
        │
        ▼
 Traefik HTTPS Listener
        │
        ▼
  Encrypted Client Traffic
```

cert-manager manages certificate issuance and renewal, while Traefik uses the resulting TLS secret to terminate HTTPS traffic at the platform edge.

## DNS Automation with ExternalDNS

ExternalDNS integrates Kubernetes with Amazon Route53.

The controller uses IRSA to receive the AWS permissions required to manage DNS records without embedding AWS credentials in Kubernetes manifests.

This connects application routing with the AWS DNS layer:

```text
Kubernetes routing configuration
             │
             ▼
        ExternalDNS
             │
             ▼
        Amazon Route53
             │
             ▼
      bravoorders.com
```

Together, ExternalDNS, cert-manager, Traefik, and Gateway API provide an automated path from public DNS through TLS termination to Kubernetes services.

## NetworkPolicy

The shared Helm chart includes reusable Kubernetes `NetworkPolicy` support.

Per-service configuration defines the network paths required by each workload, including:

- permitted ingress sources
- downstream service communication
- Redis access
- PostgreSQL access
- external HTTPS egress

This provides network-level segmentation between workloads while allowing the communication paths required by the microservice architecture.

Instead of giving every service unrestricted east-west connectivity, network behaviour can be defined alongside each service's deployment configuration.

## RBAC and Service Accounts

Platform controllers and workloads use Kubernetes service accounts as part of their identity model.

Dedicated service accounts are used by components including:

- Argo CD
- External Secrets Operator
- ExternalDNS
- Karpenter
- KEDA
- Velero
- application workloads

For AWS-integrated controllers, Kubernetes service accounts are combined with IAM Roles for Service Accounts (IRSA).

This creates an identity chain from Kubernetes to AWS:

```text
Kubernetes Pod
      │
      ▼
ServiceAccount
      │
      ▼
     IRSA
      │
      ▼
   IAM Role
      │
      ▼
Required AWS Service
```

This allows AWS permissions to be associated with workload identity rather than static AWS credentials.

## Platform Controllers

The GitOps platform manages the supporting controllers required by the application architecture.

| Component | Responsibility |
|---|---|
| Traefik | Gateway API and application ingress |
| cert-manager | TLS certificate lifecycle |
| ExternalDNS | Route53 DNS management |
| External Secrets Operator | AWS Secrets Manager synchronization |
| Karpenter | Dynamic EKS worker-node provisioning |
| metrics-server | Kubernetes resource metrics |
| Prometheus | Metrics collection |
| Grafana | Monitoring dashboards |
| Alertmanager | Alert management |
| KEDA | SQS-driven workload autoscaling |
| Velero | Kubernetes backup and restore |

Managing these components through the same GitOps architecture keeps platform configuration version-controlled alongside the workloads that depend on it.

## How a Git Change Becomes a Work
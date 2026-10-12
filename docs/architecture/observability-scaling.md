# # Observability and Scaling

Bravo EKS combines Kubernetes-native observability with AWS-native monitoring to provide visibility across application workloads, cluster resources, event queues, and database infrastructure.

Prometheus and Grafana provide cluster and workload observability, while Amazon CloudWatch monitors key AWS services including RDS and SQS.

The platform also uses a layered autoscaling model:

```text
Application demand
       │
       ├── CPU / workload metrics ──► HPA ──► Pod scaling
       │
       ├── SQS queue backlog ───────► KEDA ─► Worker scaling
       │
       └── Pod scheduling demand ───► Karpenter ─► EKS node provisioning
```

This allows Bravo EKS to scale both application workloads and the underlying compute capacity according to demand.

## Monitoring Stack

The Kubernetes monitoring stack is deployed through GitOps and configured under `kubernetes/platform/monitoring/values.yaml`.

| Component | Role |
|---|---|
| Prometheus | Metrics collection and alert evaluation |
| Grafana | Dashboards and visualisation |
| kube-state-metrics | Kubernetes object and workload state |
| node-exporter | Node and operating-system metrics |
| Alertmanager | Alert routing and grouping |
| CloudWatch | AWS-level RDS and SQS monitoring |

Together, these components provide visibility from individual Kubernetes workloads through to supporting AWS infrastructure.

## Prometheus and Grafana

Prometheus provides the primary metrics collection layer for Kubernetes.

It is configured with a 30-second scrape interval and seven-day retention, providing a continuous view of cluster and workload behaviour.

Grafana provides the visualisation layer and is configured with the platform domain:

`grafana.bravoorders.com`

The monitoring stack includes resource requests and limits for Prometheus and Grafana so that the observability components participate in the same Kubernetes resource-management model as the application workloads.

The resulting monitoring flow is:

```text
Kubernetes workloads
        │
        ├── kube-state-metrics
        ├── node-exporter
        └── application / platform metrics
                    │
                    ▼
               Prometheus
                    │
          ┌─────────┴─────────┐
          ▼                   ▼
       Grafana           Alertmanager
```

## Kubernetes Metrics

### kube-state-metrics

`kube-state-metrics` exposes metrics describing the state of Kubernetes resources.

This provides Prometheus with visibility into resources such as:

- Pods
- Deployments
- ReplicaSets
- Nodes
- resource states
- workload availability

These metrics complement application health probes by providing a cluster-level view of how Kubernetes is managing the workloads.

### node-exporter

`prometheus-node-exporter` provides host and operating-system metrics from the EKS worker nodes.

This extends monitoring beyond individual containers and allows infrastructure health to be observed alongside Kubernetes workload state.

Together, `kube-state-metrics` and `node-exporter` provide visibility across both the Kubernetes control surface and the underlying worker-node resources.

## AWS Monitoring with CloudWatch

Amazon CloudWatch provides the AWS infrastructure monitoring layer.

The Terraform monitoring module creates alarms for key RDS and SQS signals, including:

- RDS CPU utilization
- RDS free storage space
- SQS main queue depth
- SQS dead-letter queue depth

This provides monitoring outside the Kubernetes cluster for services that form part of the wider Bravo EKS architecture.

The overall observability model therefore spans both platform layers:

```text
                 Bravo EKS Observability

        Kubernetes                     AWS
             │                          │
     ┌───────┴────────┐          ┌──────┴───────┐
     │                │          │              │
Workloads           Nodes       RDS            SQS
     │                │          │              │
     └──── Prometheus ┘          └── CloudWatch ┘
              │
           Grafana
```

This separation ensures that AWS service health remains observable independently of the Kubernetes monitoring stack.

## Application Health

The shared microservice Helm chart provides a consistent health-check model across the application workloads.

Services can use:

- `startupProbe`
- `livenessProbe`
- `readinessProbe`

Each probe serves a different purpose:

| Probe | Purpose |
|---|---|
| Startup | Allows Kubernetes to determine when an application has successfully started |
| Liveness | Detects unhealthy containers that should be restarted |
| Readiness | Determines whether a workload is ready to receive traffic |

Readiness probes integrate directly with Kubernetes service routing, preventing traffic from being sent to workloads until they are ready.

Combined with Prometheus metrics, this provides both active workload health management and operational visibility.

## Resource Management

Application workloads define Kubernetes resource requests and limits through the shared Helm chart and environment-specific service values.

This allows Kubernetes to make informed scheduling decisions and provides a predictable resource model for autoscaling.

The deployment model combines:

- CPU and memory requests
- CPU and memory limits
- health probes
- rolling deployment strategies
- horizontal pod autoscaling
- event-driven scaling
- dynamic node provisioning

These controls work together rather than operating as independent features.

## Horizontal Pod Autoscaling

The shared microservice Helm chart includes reusable HorizontalPodAutoscaler resources.

HPA can be enabled through individual service values, allowing workloads such as API Gateway and other HTTP-facing services to scale their replica count according to resource demand.

The scaling path is:

```text
Workload demand
      ↓
CPU utilisation
      ↓
HPA evaluates target
      ↓
Replica count increases/decreases
      ↓
Kubernetes schedules required Pods
```

Because HPA configuration is part of the shared Helm architecture, autoscaling behaviour can be applied consistently while remaining configurable for each service.

## Event-Driven Scaling with KEDA

Bravo EKS uses KEDA to scale event-driven workloads based on SQS activity.

The Worker is the primary queue consumer and uses an AWS SQS trigger so that its replica count can respond directly to event backlog.

Worker scaling configuration includes:

- environment-specific SQS queue URL
- AWS region
- queue-depth thresholds
- minimum replica count
- maximum replica count
- scheduled scaling profiles

This allows Worker capacity to follow the amount of asynchronous work waiting to be processed rather than relying only on CPU utilization.

The scaling flow is:

```text
Domain events
      ↓
Amazon SQS
      ↓
Queue backlog
      ↓
KEDA
      ↓
Worker replica scaling
      ↓
Events processed
```

KEDA accesses the required SQS information through AWS workload identity, integrating event-driven scaling with the platform's IAM and IRSA model.

## Node Autoscaling with Karpenter

Karpenter provides the infrastructure scaling layer for the EKS cluster.

While HPA and KEDA determine how many Pods are required, Karpenter ensures that sufficient worker-node capacity exists to schedule those Pods.

Karpenter uses the platform's `NodePool` and `EC2NodeClass` configuration to provision application compute capacity in the private subnets.

The configured node model includes:

- application nodes labelled with `workload: application`
- private subnet placement
- workload-aware node provisioning
- Spot and On-Demand capacity types
- node consolidation
- node expiration and lifecycle management

This creates an end-to-end scaling chain:

```text
Traffic / Events
       ↓
HPA / KEDA
       ↓
More Pods required
       ↓
Kubernetes Scheduler
       ↓
Additional capacity required
       ↓
Karpenter
       ↓
EC2 worker capacity
       ↓
Pods scheduled
```

This enables application demand to drive both workload scaling and infrastructure capacity.

## Scheduling and Resilience

The shared Helm architecture includes additional controls that support workload availability during deployments, scaling events, and node changes.

These include:

- topology spread constraints
- PodDisruptionBudgets (PDBs)
- rolling deployment strategies
- controlled `maxUnavailable`
- controlled `maxSurge`
- affinity rules
- node selectors

### Topology Spread

Topology spread constraints allow replicas to be distributed across available infrastructure rather than concentrating workloads unnecessarily on the same node or failure domain.

### PodDisruptionBudgets

PodDisruptionBudgets protect application availability during voluntary disruptions by controlling how many replicas may be unavailable at once.

### Rolling Deployments

Deployments use bounded `maxUnavailable` and `maxSurge` settings so Kubernetes can introduce updated replicas progressively while maintaining service capacity.

### Workload Placement

Affinity rules, labels, and node selectors help separate platform and application workloads and allow Kubernetes to place services onto the intended compute capacity.

## Observability and Scaling Architecture

The complete model combines monitoring, workload health, pod scaling, queue-driven scaling, and infrastructure elasticity:

```text
                         User Traffic
                              │
                              ▼
                     Application Workloads
                       │             │
                       │             └──── CPU demand ────► HPA
                       │
                       └── Health / Metrics
                               │
                               ▼
                           Prometheus
                               │
                      ┌────────┴────────┐
                      ▼                 ▼
                   Grafana         Alertmanager


Domain Events ──► Amazon SQS ──► KEDA ──► Worker replicas
                      │
                      └────────────► CloudWatch


RDS ──────────────────────────────► CloudWatch


HPA / KEDA
     │
     ▼
Pod scheduling demand
     │
     ▼
Karpenter
     │
     ▼
EKS worker capacity
```

This provides Bravo EKS with observability and scaling across multiple layers:

- **application health** through Kubernetes probes
- **workload metrics** through Prometheus
- **visualisation** through Grafana
- **Kubernetes state** through kube-state-metrics
- **node health** through node-exporter
- **AWS infrastructure monitoring** through CloudWatch
- **resource-based pod scaling** through HPA
- **event-driven Worker scaling** through KEDA
- **infrastructure scaling** through Karpenter
- **deployment resilience** through PDBs, topology spreading, and rolling updates

The result is an integrated platform where monitoring provides visibility into system behaviour while Kubernetes and AWS-native scaling mechanisms dynamically adjust workload and compute capacity.

## Related Documentation

- [AWS Infrastructure & Terraform](aws-infrastructure.md)
- [Kubernetes & GitOps](kubernetes-gitops.md)
- [Data & Messaging](data-and-messaging.md)
- [Runbook Index](../runbooks/README.md)
- [Scaling & Capacity Runbook](../runbooks/05-scaling-capacity.md)
- [Monitoring & Alerts Runbook](../runbooks/06-monitoring-alerts.md)
- [SQS Monitoring Runbook](../runbooks/13-sqs-monitoring.md)
# BRAVO EKS

> Production Engineering Simulation — AWS, Kubernetes & Microservices

BRAVO is a production-oriented e-commerce platform built as a distributed microservices application and deployed on **Amazon EKS**.

The project simulates the work of a production engineering team responsible for taking a containerised application from local development through to a highly available, observable and secure Kubernetes environment.

The platform supports customer and admin workflows including product browsing, order creation, inventory management, payments, shipping, notifications and order tracking.

The project is developed by a **five-engineer production engineering team**, with work spanning application services, AWS infrastructure, Kubernetes, CI/CD, security, observability and reliability.

---

## Project at a Glance

|                             |                                                 |
| --------------------------- | ----------------------------------------------- |
| **Cloud**                   | AWS                                             |
| **Region**                  | `eu-west-2`                                     |
| **Container orchestration** | Amazon EKS                                      |
| **Infrastructure as Code**  | Terraform                                       |
| **Application**             | Go microservices                                |
| **Containers**              | Docker                                          |
| **Kubernetes packaging**    | Helm                                            |
| **Deployment**              | Argo CD / GitOps                                |
| **Autoscaling**             | Karpenter                                       |
| **Ingress**                 | Traefik                                         |
| **TLS**                     | cert-manager                                    |
| **DNS**                     | External DNS                                    |
| **Secrets**                 | AWS Secrets Manager + External Secrets Operator |
| **Database**                | PostgreSQL / Amazon RDS                         |
| **Caching**                 | Redis                                           |
| **Messaging**               | Amazon SQS                                      |
| **Monitoring**              | Prometheus                                      |
| **Dashboards**              | Grafana                                         |
| **Alerting**                | Alertmanager / CloudWatch                       |
| **Backups**                 | Velero                                          |
| **CI/CD**                   | GitHub Actions                                  |

---

# Architecture

The platform is composed of containerised microservices running on Amazon EKS, supported by AWS infrastructure and Kubernetes platform services.

The architecture diagram below provides the high-level view of how the application, infrastructure and platform components fit together.

### Architecture Diagram

<!-- Add architecture diagram here -->

`docs/architecture/`

For detailed application behaviour and service interactions, see:

[`docs/architecture/application-flow.md`](docs/architecture/application-flow.md)

---

# Application

BRAVO is a distributed order-management platform made up of independent services.

### Core services

| Service                | Purpose                                              |
| ---------------------- | ---------------------------------------------------- |
| `dashboard-api`        | Customer and admin web interface                     |
| `api-gateway`          | API entry point, authentication and request handling |
| `order-service`        | Order creation and order lifecycle                   |
| `inventory-service`    | Product stock and inventory                          |
| `payment-service`      | Payments, refunds and payment records                |
| `shipping-service`     | Shipments and tracking                               |
| `notification-service` | Customer notifications                               |
| `worker`               | Background event processing                          |
| `scheduler`            | Scheduled database and maintenance tasks             |

The application uses a combination of synchronous HTTP communication and asynchronous messaging through Amazon SQS.

---

# Key Platform Components

The project uses AWS and Kubernetes technologies to provide the underlying platform.

### AWS

* Amazon EKS
* Amazon VPC
* Amazon RDS PostgreSQL
* Amazon SQS / DLQs
* Amazon ECR
* AWS IAM
* AWS KMS
* AWS Secrets Manager
* Amazon EBS
* Amazon CloudWatch
* Amazon SNS

### Kubernetes

* Kubernetes
* Helm
* Argo CD
* Karpenter
* Traefik
* cert-manager
* External DNS
* External Secrets Operator
* Velero
* EBS CSI Driver

### Application & Delivery

* Go
* Docker
* GitHub Actions
* Terraform
* GitHub
* Amazon ECR

---

# Local Development

The application can be run locally using Docker Compose.

## Prerequisites

Install:

* Docker
* Docker Compose
* Git

Clone the repository:

```bash
git clone https://github.com/bravo-eks/bravo-eks.git
cd bravo-eks
```

### Configure the environment

Create a local environment file from the example:

```bash
cp .env.example .env
```

Review the values in `.env` before starting the application.

> Do not commit `.env` or any credentials to Git.

### Start the application

```bash
docker compose up --build
```

This builds and starts the application services and their local dependencies defined in the Docker Compose configuration.

### Stop the application

```bash
docker compose down
```

### Remove local volumes

If you need to reset local persistent data:

```bash
docker compose down -v
```

Then start the application again:

```bash
docker compose up --build
```

### Check running containers

```bash
docker compose ps
```

### View logs

View all logs:

```bash
docker compose logs
```

View a specific service:

```bash
docker compose logs <service-name>
```

Follow logs:

```bash
docker compose logs -f <service-name>
```

---

# Kubernetes & Deployment

The application is deployed to Amazon EKS.

Terraform provisions the AWS infrastructure, while Kubernetes and Helm define the workloads running inside the cluster.

Argo CD provides the GitOps deployment model.

```text
GitHub
   |
   v
GitHub Actions
   |
   v
Amazon ECR
   |
   v
Argo CD
   |
   v
Amazon EKS
```

Application deployments are managed through the Kubernetes configuration in:

```text
kubernetes/
```

Infrastructure is managed through:

```text
terraform/
```

---

# Environments

The project contains separate Terraform environments for development and production.

```text
terraform/
├── bootstrap/
├── environments/
│   ├── dev/
│   └── prod/
└── modules/
```

### Development

Used for development, testing and validating platform changes before production.

### Production

Used for the production-simulation environment and demonstrates how the platform would be operated in a more production-oriented setup.

---

# CI/CD

GitHub Actions automates application and infrastructure workflows.

Application changes follow a workflow similar to:

```text
Pull Request
     |
     v
Detect changed services
     |
     v
Build Docker image
     |
     v
Run checks
     |
     v
Merge
     |
     v
Build image with Git SHA
     |
     v
Push to Amazon ECR
     |
     v
Argo CD deploys to EKS
```

Terraform changes are also validated and planned through GitHub Actions before infrastructure changes are applied.

Workflow definitions:

```text
.github/workflows/
```

---

# Security

Security has been considered across the application, infrastructure and deployment pipeline.

Key controls include:

* AWS IAM for access control
* GitHub Actions OIDC for AWS authentication
* Kubernetes service accounts
* IRSA / AWS identity integrations
* Private AWS networking
* KMS encryption
* Encrypted persistent storage
* AWS Secrets Manager
* External Secrets Operator
* TLS certificates through cert-manager
* Kubernetes RBAC
* Container image security checks
* Terraform security scanning with Checkov
* No credentials committed to the repository

Secrets follow the general flow:

```text
AWS Secrets Manager
        |
        v
External Secrets Operator
        |
        v
Kubernetes Secret
        |
        v
Application
```

---

# Monitoring & Observability

The platform includes both Kubernetes and AWS monitoring.

## Prometheus

Prometheus collects metrics from the Kubernetes environment and application platform.

<!-- Prometheus screenshot -->

### Prometheus Dashboard

**Screenshot:**

<!-- Add Prometheus screenshot here -->

---

## Grafana

Grafana provides dashboards for visualising platform and application metrics.

<!-- Grafana screenshot -->

### Grafana Dashboard

**Screenshot:**

<!-- Add Grafana screenshot here -->

---

## Alertmanager

Alertmanager handles alerts generated by Prometheus and routes notifications based on the configured alerting rules.

### Alerting

**Screenshot:**

<!-- Add Alertmanager screenshot here -->

---

## AWS CloudWatch

CloudWatch monitoring is also configured for important AWS resources, including infrastructure and application dependencies.

Examples include:

* RDS CPU utilisation
* RDS free storage
* RDS database connections
* SQS queue depth
* SQS dead-letter queue messages
* Load balancer health
* Other environment-specific infrastructure signals

---

# Backups & Reliability

The platform includes reliability and recovery components designed to simulate production operational practices.

### Velero

Velero is used for Kubernetes backup and recovery.

### Persistent Storage

Persistent workloads use AWS-backed storage, including:

* Amazon RDS for PostgreSQL
* Amazon EBS
* EBS CSI Driver
* Persistent Volume Claims

### Autoscaling

Karpenter provides dynamic Kubernetes node provisioning based on workload requirements.

---

# Repository Structure

```text
.
├── .github/
│   └── workflows/              # CI/CD workflows
│
├── docs/                       # Project documentation
│   ├── architecture/
│   ├── runbooks/
│   └── tests/
│
├── kubernetes/                 # Kubernetes & GitOps configuration
│   ├── argocd/
│   ├── helm/
│   └── platform/
│
├── services/                   # Application microservices
│   ├── api-gateway/
│   ├── dashboard-api/
│   ├── inventory-service/
│   ├── notification-service/
│   ├── order-service/
│   ├── payment-service/
│   ├── scheduler/
│   ├── shipping-service/
│   └── worker/
│
├── terraform/                  # AWS infrastructure
│   ├── bootstrap/
│   ├── environments/
│   └── modules/
│
├── docker-compose.yml          # Local development
└── .env.example                # Example local configuration
```

---

# Useful Documentation

| Documentation                                                                    | Description                                     |
| -------------------------------------------------------------------------------- | ----------------------------------------------- |
| [`docs/architecture/application-flow.md`](docs/architecture/application-flow.md) | Detailed application architecture and workflows |
| [`docs/runbooks/`](docs/runbooks/)                                               | Operational procedures                          |
| [`docs/tests/`](docs/tests/)                                                     | Infrastructure and testing documentation        |
| [`kubernetes/`](kubernetes/)                                                     | Kubernetes and GitOps configuration             |
| [`terraform/`](terraform/)                                                       | AWS infrastructure                              |

Individual services also contain service-specific documentation where applicable.

---

# Getting Access to EKS

Engineers with the appropriate AWS permissions can configure `kubectl` for an environment using:

```bash
aws eks update-kubeconfig \
  --name <cluster-name> \
  --region eu-west-2
```

Verify access:

```bash
kubectl get nodes
```

View workloads:

```bash
kubectl get pods -A
```

View Argo CD applications:

```bash
kubectl get applications -n argocd
```

Cluster names and AWS permissions are environment-specific and should not be hard-coded into this README.

---

# Live Application

The BRAVO application is available here:

**[Live BRAVO Application](https://bravoorders.com)**

<!-- Replace the URL above with the final production/demo URL if different. -->

---

# Project Team

BRAVO is developed as a **five-engineer production engineering team**.

The team works across:

* AWS infrastructure
* Terraform
* Kubernetes
* Application development
* CI/CD
* Security
* Observability
* Reliability
* Disaster recovery
* Production operations

The project is designed to reflect the collaboration, ownership and engineering practices expected when operating a real production platform.

---

# Project Goal

The goal of BRAVO is to demonstrate how a distributed application can be designed, deployed and operated using modern cloud and platform engineering practices.

The project brings together:

* Infrastructure as Code
* AWS cloud infrastructure
* Kubernetes
* Microservices
* GitOps
* CI/CD
* Secure identity and secrets management
* Event-driven processing
* Autoscaling
* Monitoring and observability
* Backup and recovery
* Production-focused operational practices

The result is a complete platform that can be developed locally, deployed through automated pipelines and operated on Amazon EKS.

---

## Technology Stack

```text
Cloud              AWS
Infrastructure     Terraform
Containers         Docker
Orchestration     Kubernetes / EKS
Packaging         Helm
GitOps            Argo CD
Autoscaling       Karpenter
Ingress           Traefik
TLS               cert-manager
DNS               External DNS
Secrets           AWS Secrets Manager / External Secrets
Database          PostgreSQL / RDS
Cache             Redis
Messaging         Amazon SQS
Monitoring        Prometheus
Dashboards        Grafana
Alerting          Alertmanager / CloudWatch
Backups           Velero
CI/CD             GitHub Actions
Application       Go
```

# DEV infrastructure health check

## Scope and context

Environment: dev
AWS region: eu-west-2
AWS account: 304960797322
EKS cluster: bravo-dev
Route53 domain: bravoorders.com
VPC CIDR: 10.0.0.0/16

This runbook records the infrastructure validation that was completed after deployment for the Bravo DEV environment. It is intentionally scoped to foundational AWS and Kubernetes health checks only. The EKS cluster had not yet been bootstrapped with Argo CD, so platform/application workloads were not expected to exist yet and application readiness is separate from infrastructure health.

The deployed architecture includes:

- VPC
- 2 public subnets
- 2 private application subnets
- 2 private database subnets
- Internet Gateway
- 1 NAT Gateway
- EKS
- Managed EKS system node group
- VPC CNI
- CoreDNS
- kube-proxy
- EBS CSI
- RDS PostgreSQL
- SQS + DLQ
- ECR
- IAM / IRSA
- AWS Secrets Manager
- Route53

## Health check summary

| Area | Status | Notes |
|---|---|---|
| AWS / Terraform state | PASS | Expected AWS resources were present in the deployed state. |
| Existing infrastructure health | PASS | The AWS and EKS foundation was functioning at the time of the validation. |
| Terraform convergence | PENDING | A non-empty plan remained due to expected runtime secret and IAM policy changes. |
| Destructive changes | NONE | No resources were scheduled for destruction. |
| Kubernetes API access | PASS | CLI access succeeded after a required EKS access entry was added manually. |
| EKS worker nodes | PASS | Two Ready worker nodes were present in the cluster. |
| Application deployment readiness | NOT READY | Argo CD had not yet been bootstrapped, so application workloads were not expected. |

## 1. AWS / Terraform

### What is being tested

Whether the deployed DEV infrastructure is present in Terraform state and whether the current AWS foundation is healthy before any application rollout work begins.

### Why it is being tested

Terraform state is the source of truth for the already-deployed infrastructure. A healthy foundation should include the expected EKS, RDS, ECR, VPC, and IAM resources, and any remaining plan changes should be understood as pending prerequisites rather than as evidence of broken infrastructure.

### Command used

```bash
cd terraform/environments/dev
terraform state list
terraform plan
```

### Expected healthy result

- `terraform state list` includes the expected DEV resources.
- `terraform plan` shows the deployment is either converged or only contains expected, understood follow-up changes.
- No destructive changes are reported for existing infrastructure.

### Actual verified result

The state contained the expected DEV infrastructure, including:

- ECR repositories and lifecycle policies
- EKS cluster
- EKS managed node group
- EKS add-ons
- EKS IAM/OIDC resources
- workload IAM roles/policies
- RDS
- SQS and DLQ
- VPC
- NAT Gateway
- route tables
- public/private/database subnets

The plan output at the time of the health check reported:

- Plan: 4 to add, 1 to change, 0 to destroy

The pending changes were related to runtime secrets:

- `bravo-dev-runtime` Secrets Manager secret
- runtime secret version
- generated JWT secret
- generated Redis password
- update to the External Secrets IAM policy so it can read the required runtime secret

These were known pending deployment prerequisites and did not indicate that the existing infrastructure was unhealthy.

### Status

- Terraform state: PASS
- Existing infrastructure health: PASS
- Terraform convergence: PENDING / expected changes remain
- Destructive changes: NONE

Important note: A non-empty Terraform plan does not by itself mean the already-deployed infrastructure is unhealthy. It means additional planned changes remain to be reviewed and applied deliberately when required.

## 2. EKS API / kubectl access

### What is being tested

Whether the AWS identity used locally can authenticate to the EKS control plane and use `kubectl` against the `bravo-dev` cluster.

### Why it is being tested

A cluster may exist and be healthy at the AWS layer while a local administrator identity still lacks Kubernetes access. This check confirms that the expected identity and EKS access configuration are in place before any Kubernetes validation is attempted.

### Command used

```bash
aws eks update-kubeconfig \
  --region eu-west-2 \
  --name bravo-dev
```

```bash
aws eks describe-cluster \
  --name bravo-dev \
  --region eu-west-2 \
  --query 'cluster.accessConfig'
```

```bash
aws eks list-access-entries \
  --cluster-name bravo-dev \
  --region eu-west-2
```

### Expected healthy result

- The kubeconfig context updates successfully.
- The relevant IAM principal has the required Kubernetes access entry and RBAC permissions.
- `kubectl` commands run without authentication failures.

### Actual verified result

The kubeconfig was successfully updated to the cluster context:

```text
arn:aws:eks:eu-west-2:304960797322:cluster/bravo-dev
```

Initial `kubectl` authentication failed because the local IAM user did not have an EKS access entry. The cluster authentication mode was inspected and reported:

```text
authenticationMode = API_AND_CONFIG_MAP
```

The required administrator access entry was then configured manually, and `kubectl` access succeeded.

### Status

PASS

### Operational note

AWS IAM Administrator permissions do not automatically grant Kubernetes/EKS cluster access. Future engineers should validate the appropriate EKS access entry and IAM policy mapping for their team and identity requirements before relying on `kubectl` access. Do not document or suggest a command that automatically creates access entries without explicit review and approval.

## 3. EKS worker nodes

### What is being tested

Whether the EKS cluster is running a healthy managed worker node fleet.

### Why it is being tested

The worker nodes underpin all Kubernetes workloads, including the control plane add-ons and any future application workloads. Validating node readiness confirms the cluster is operational at the Kubernetes layer before Argo CD or service deployment begins.

### Command used

```bash
kubectl get nodes
```

### Expected healthy result

- Worker nodes are present.
- All nodes report `Ready` status.
- Nodes are running a compatible Kubernetes version.

### Actual verified result

Two nodes were present:

- `ip-10-0-11-115.eu-west-2.compute.internal`
- `ip-10-0-12-157.eu-west-2.compute.internal`

Both reported:

- `STATUS = Ready`

Kubernetes version reported by the cluster was:

```text
v1.33.13-eks-3b4a6ca
```

The node IPs confirmed that the worker nodes were operating across both private application subnet ranges:

- `10.0.11.0/24`
- `10.0.12.0/24`

### Status

PASS

### Operational note

The exact node IP addresses are ephemeral and should not be treated as the future expected state. They are included here only as evidence of the verified runtime state at the time of the health check.

## Final assessment

The DEV AWS and EKS foundation was healthy at the time of validation. Terraform state showed the expected infrastructure resources, kubectl access was restored with the required cluster access configuration, and the worker nodes were Ready. The only open item was the expected runtime-secret and IAM policy convergence work reflected in a non-empty Terraform plan, which should be reviewed before any future apply. Application readiness remains separate from infrastructure health and is not yet established because Argo CD had not yet been bootstrapped.

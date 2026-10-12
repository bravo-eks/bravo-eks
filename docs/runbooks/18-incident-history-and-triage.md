# 18 — Incident history, known fixes and triage map

## Purpose
This index maps notable **historical PRs/commits** to the right operational procedure. A merged PR title is evidence of a code/config change, **not** proof that a live incident occurred or that a fix was tested in production. Use linked diffs and runtime evidence to establish the exact root cause. This is a living index, not an exhaustive list of every commit.

## Where to start
| Symptom | First runbook |
|---|---|
| Access denied / unauthorized | [01 Cluster access](01-cluster-access.md), [15 AWS identity](15-aws-identity-secrets.md) |
| Bad release / Argo CD degraded | [02 Deployment rollback](02-deployment-rollback.md), [14 CI/CD](14-cicd-gitops-troubleshooting.md) |
| CrashLoopBackOff / failed readiness | [03 Pods](03-pod-troubleshooting.md) |
| DNS/TLS/404/503 | [04 Networking](04-networking-ingress.md) |
| Pending / pod IP / nodes | [05 Scaling](05-scaling-capacity.md), [16 CNI/Karpenter](16-cni-karpenter-network-recovery.md) |
| Alerts / queue backlog | [06 Monitoring](06-monitoring-alerts.md), [13 SQS](13-sqs-monitoring.md) |
| RDS waiting / permission denied | [12 RDS](12-rds-database-connectivity.md) |
| Broken customer order | [17 Application flow](17-application-order-flow.md) |
| Terraform lock or destroy | [08 Terraform](08-terraform-recovery.md) |
| Lost data / backup | [09 Disaster recovery](09-disaster-recovery.md) |

## Selected historical changes (GitHub PR links)
| PR | Change shown by history | Troubleshooting destination |
|---|---|---|
| [#189](https://github.com/bravo-eks/bravo-eks/pull/189), [#185](https://github.com/bravo-eks/bravo-eks/pull/185), [#145](https://github.com/bravo-eks/bravo-eks/pull/145), [#151](https://github.com/bravo-eks/bravo-eks/pull/151) | Terraform pipeline permissions, workflow validation | 14, 08 |
| [#184](https://github.com/bravo-eks/bravo-eks/pull/184) | PROD IRSA workload namespace corrected | 15, 07 |
| [#113](https://github.com/bravo-eks/bravo-eks/pull/113), [#130](https://github.com/bravo-eks/bravo-eks/pull/130), [#149](https://github.com/bravo-eks/bravo-eks/pull/149) | Database/Alertmanager secret properties and access | 15, 12 |
| [#144](https://github.com/bravo-eks/bravo-eks/pull/144), [#148](https://github.com/bravo-eks/bravo-eks/pull/148) | KEDA IRSA and scaler provider fix | 05, 15, 13 |
| [#132](https://github.com/bravo-eks/bravo-eks/pull/132), [#133](https://github.com/bravo-eks/bravo-eks/pull/133), [#131](https://github.com/bravo-eks/bravo-eks/pull/131) | CNI IP allocation and Karpenter pod density | 16, 05 |
| [#120](https://github.com/bravo-eks/bravo-eks/pull/120), [#137](https://github.com/bravo-eks/bravo-eks/pull/137), [#138](https://github.com/bravo-eks/bravo-eks/pull/138), [#139](https://github.com/bravo-eks/bravo-eks/pull/139) | Spot permissions/service-linked role | 16, 05 |
| [#105](https://github.com/bravo-eks/bravo-eks/pull/105), [#108](https://github.com/bravo-eks/bravo-eks/pull/108), [#172](https://github.com/bravo-eks/bravo-eks/pull/172), [#175](https://github.com/bravo-eks/bravo-eks/pull/175) | Gateway/cert-manager, Traefik, HTTPS and Route 53 | 04 |
| [#109](https://github.com/bravo-eks/bravo-eks/pull/109), [#122](https://github.com/bravo-eks/bravo-eks/pull/122) | Argo CD AppProject bootstrap/namespace policy | 14, 02 |
| [#115](https://github.com/bravo-eks/bravo-eks/pull/115), [#118](https://github.com/bravo-eks/bravo-eks/pull/118) | GitOps migration Jobs and image tags | 14, 12 |
| [#119](https://github.com/bravo-eks/bravo-eks/pull/119) | Dashboard RDS missing table SELECT grants, documented diagnostic | 12 |
| [#124](https://github.com/bravo-eks/bravo-eks/pull/124), [#126](https://github.com/bravo-eks/bravo-eks/pull/126), [#128](https://github.com/bravo-eks/bravo-eks/pull/128) | Shipping service changes | 17 |
| [#141](https://github.com/bravo-eks/bravo-eks/pull/141), [#143](https://github.com/bravo-eks/bravo-eks/pull/143) | Order events and KEDA scaling | 17, 13, 05 |
| [#134](https://github.com/bravo-eks/bravo-eks/pull/134), [#135](https://github.com/bravo-eks/bravo-eks/pull/135), [#150](https://github.com/bravo-eks/bravo-eks/pull/150) | Velero backup, PVC/DR, PROD infrastructure | 09 |
| [#168](https://github.com/bravo-eks/bravo-eks/pull/168), [#170](https://github.com/bravo-eks/bravo-eks/pull/170), [#171](https://github.com/bravo-eks/bravo-eks/pull/171) | DEV/PROD pipelines and secret path separation | 14, 15 |
| [#176](https://github.com/bravo-eks/bravo-eks/pull/176), [#181](https://github.com/bravo-eks/bravo-eks/pull/181) | PROD catalog rendering | 17 |

## During an incident
1. Verify AWS account, environment and cluster.
2. Classify impact, capture timestamp, alert, customer symptom and recent changes.
3. Pick the **first failing layer** using the table above.
4. Run read-only diagnostics, compare expected and actual state, then choose an approved reversible mitigation.
5. Verify the user journey and system metrics, document outcome and prevention.

## Coverage limitations
No static runbook can anticipate **everything**. This index was assembled from recent GitHub history and selected diffs; it is not a verified exhaustive audit of all older commits, issues, AWS console incidents, on-call records or live environments. For each new incident, add a redacted worked example, evidence, root cause, exact fix and validation results. Do not infer production testing from a merged PR.

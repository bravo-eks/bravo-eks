# 16 — VPC CNI, pod IPs and Karpenter node recovery

## Purpose and scope
Use when new pods cannot get IPs, nodes fail to register, CoreDNS is unreachable, or Karpenter cannot provision enough capacity. Distinguish EC2 instance quota from pod density and subnet IP exhaustion.

## Symptoms
`FailedCreatePodSandBox`, `failed to assign an IP address`, Pending pods, nodes NotReady, Karpenter NodeClaims stalled, or intermittent service DNS.

```bash
kubectl get nodes -o wide
kubectl describe node <node>
kubectl get pods -n kube-system -o wide
kubectl logs -n kube-system daemonset/aws-node --tail=100
kubectl get nodepools,nodeclaims,ec2nodeclasses -A
kubectl describe nodeclaim <nodeclaim>
kubectl get events -A --sort-by=.lastTimestamp
aws ec2 describe-subnets --region eu-west-2 --query 'Subnets[].{id:SubnetId,available:AvailableIpAddressCount,az:AvailabilityZone}'
```

## Decision tree
1. **No pod IP:** inspect VPC CNI add-on status, `aws-node` logs, subnet available IPs, security groups, ENI limits and instance-type pod density.
2. **Prefix delegation mismatch:** compare Terraform `vpc-cni` add-on configuration with Karpenter `EC2NodeClass` kubelet maxPods. [PR #132](https://github.com/bravo-eks/bravo-eks/pull/132) removed prefix-delegation-specific settings and excluded very small nodes; [#133](https://github.com/bravo-eks/bravo-eks/pull/133) explicitly disabled prefix delegation. Do not re-enable without a tested IP allocation and node replacement plan.
3. **No new node:** inspect NodePool constraints/limits, EC2NodeClass subnet/SG discovery tags, instance family availability, IAM, quotas, Spot service-linked role and controller logs. See [#120](https://github.com/bravo-eks/bravo-eks/pull/120), [#137–139](https://github.com/bravo-eks/bravo-eks/pull/139).
4. **Node exists but pods Pending:** inspect taints/tolerations, requests, affinity, topology spread, EBS zone constraints and namespace quotas.
5. **DNS:** inspect CoreDNS readiness and kube-dns endpoints, then test DNS from an authorized diagnostic pod; distinguish DNS from routing and upstream timeout.

## Recovery
Change one variable at a time through reviewed Terraform/GitOps. Avoid indiscriminate node drains: protect availability, PDBs, RDS connections and worker idempotency. Use a controlled node rotation only after spare capacity is confirmed.

## Verification
Pods receive IPs, nodes Ready, CoreDNS queries succeed, pending pods schedule, service traffic and worker queue processing recover. Record available IPs and pod density before/after.

# 04 — Networking and ingress failures

> **Safety:** Select the correct AWS account, cluster, namespace and environment before acting. Production changes require incident/change approval. Commands are diagnostic unless explicitly marked otherwise. Do not paste secret values into tickets or logs.

**Baseline:** AWS region `eu-west-2`; DEV/PROD are separate. The repository's canonical operational directory is `docs/runbooks/`. Cluster names should be verified with `aws eks list-clusters --region eu-west-2` rather than assumed.

```bash
export AWS_PROFILE=bravo
export AWS_REGION=eu-west-2
aws sts get-caller-identity
kubectl config current-context
kubectl get nodes -o wide
```

If the cluster has been destroyed, AWS and Kubernetes checks will fail by design. Recreate only through the approved Terraform workflow.

## Actual request path

`Route 53 DNS → internet-facing Classic Load Balancer → Traefik TLS termination → Gateway API / HTTPRoute → Dashboard API → API Gateway / domain services`. Do **not** troubleshoot this deployment as if an ALB Ingress Controller is the entry point.

## Symptoms

DNS resolution failure, TLS/certificate error, 404/502/503/504, only one route broken, or internal service timeout.

## Layer-by-layer diagnosis

```bash
kubectl get svc -A
kubectl get gateways,httproutes -A
kubectl describe gateway <gateway> -n <namespace>
kubectl describe httproute <route> -n <namespace>
kubectl get endpointslices -A
kubectl get pods -A -o wide
kubectl get networkpolicy -A
aws elb describe-load-balancers --region eu-west-2
aws route53 list-hosted-zones
```

1. **DNS:** check Route 53 record target and TTL; distinguish NXDOMAIN from an unreachable LB.
2. **Classic LB:** check listener, registered instances/health, security groups and target reachability via EC2/ELB console or CLI.
3. **TLS:** inspect Traefik certificate Secret, cert-manager Certificate/Challenge resources and renewal events.
4. **Gateway/HTTPRoute:** verify Accepted/ResolvedRefs status, hostnames, parent refs, match rules and backend port.
5. **Service:** confirm selector, port/targetPort, EndpointSlices and Ready pods.
6. **East-west traffic:** check NetworkPolicies, CoreDNS, service discovery, SG/NACL routes and dependencies.

## Interpret errors

- **404:** often route/hostname mismatch or application-level missing path.
- **502/503:** backend missing/unhealthy, bad port, or proxy-to-pod failure.
- **504:** timeout; examine application latency and dependencies before raising timeouts.

## Recovery and verification

Correct desired-state manifests through GitOps, then verify DNS resolution, TLS validity, HTTP health and customer order flow. Never disable TLS or open security groups broadly as a quick fix.

## Related

[Pod troubleshooting](03-pod-troubleshooting.md), [Security](07-security-incidents.md).

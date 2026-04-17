---
schemaVersion: 1
surface: repo-authored-semantics
service: loadbalancer
slug: loadbalancer
gaps: []
---

# Logic Gaps

## Current runtime path

- `LoadBalancer` reconciles through the generated `LoadBalancerServiceManager` under `pkg/servicemanager/loadbalancer/loadbalancer`; shared CRUD orchestration still comes from `generatedruntime`, but a package-local client now owns the reviewed semantics, explicit request field mapping, and follow-up behavior for bind, update, and delete confirmation.
- The package-local client keeps standalone `GetLoadBalancer` and `DeleteLoadBalancer` operations scoped to the tracked load balancer OCID, and it keeps bind-versus-create scoped to exact `compartmentId` plus `displayName` list lookup.
- When no tracked OCI ID is present, the generated runtime probes `ListLoadBalancers` using the desired `compartmentId` and `displayName`. A single reusable match binds into the generated observe or update path; multiple matches fail rather than binding an ambiguous load balancer, and no match falls through to `CreateLoadBalancer`.

## Repo-authored semantics

- Create or bind is explicit. `LoadBalancer` reuses an existing load balancer addressed by the same compartment and display name instead of issuing a duplicate create, and the bind path records the resolved OCI ID into status for later observe, update, and delete retries.
- Supported in-place updates are limited to `displayName`, `freeformTags`, and `definedTags`, matching `UpdateLoadBalancerDetails`. Drift on `compartmentId`, `shapeName`, `subnetIds`, `shapeDetails`, `isPrivate`, `ipMode`, `reservedIps`, `listeners`, `hostnames`, `backendSets`, `networkSecurityGroupIds`, `certificates`, `sslCipherSuites`, `pathRouteSets`, and `ruleSets` remains create-only and is rejected before OCI mutation.
- The generated runtime follows create and update with `GetLoadBalancer` until OCI reports `ACTIVE`. Delete keeps the finalizer until `GetLoadBalancer` or the list fallback confirms the resource is gone or terminally `DELETED`. No Kubernetes secret reads or writes are part of this path.

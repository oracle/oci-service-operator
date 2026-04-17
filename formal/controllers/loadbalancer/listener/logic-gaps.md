---
schemaVersion: 1
surface: repo-authored-semantics
service: loadbalancer
slug: listener
gaps: []
---

# Logic Gaps

## Current runtime path

- `Listener` reconciles through the generated `ListenerServiceManager` under `pkg/servicemanager/loadbalancer/listener`; shared CRUD orchestration still comes from `generatedruntime`, but a package-local read bridge owns the parent load balancer lookup, the `loadBalancerId` status mirror, and the synthetic tracked ID in `status.status.ocid`.
- The package-local bridge resolves the OCI path identity from repo-authored fields: `loadBalancerId` from `status.loadBalancerId` or `spec.loadBalancerId`, and listener name from `status.name`, `spec.name`, or the Kubernetes object name.
- Because the Load Balancing API does not expose standalone `GetListener` or `ListListeners` calls, the local bridge reads `GetLoadBalancer`, projects the named listener from the parent `listeners` map, and uses that projection for bind-versus-create, update drift comparison, read-after-write follow-up, and delete confirmation.

## Repo-authored semantics

- Create or bind is explicit. `Listener` reuses an existing listener addressed by the same load balancer and listener name instead of issuing a duplicate create, and the bind path records a synthetic tracked ID so later observe, update, and delete retries stay on the same bound listener even though OCI does not expose a distinct Listener OCID.
- Supported in-place updates are limited to `defaultBackendSetName`, `port`, `protocol`, `hostnameNames`, `pathRouteSetName`, `sslConfiguration`, `connectionConfiguration`, `routingPolicyName`, and `ruleSetNames`, matching `UpdateListenerDetails`. Drift on `loadBalancerId` or `name` remains create-only and is rejected before OCI mutation.
- The generated runtime follows create and update with a `GetLoadBalancer`-backed read. Because the projected Listener payload has no independent lifecycle field, the write path may requeue once before the next observe settles `Active`.
- Delete keeps the finalizer until `GetLoadBalancer` no longer returns the named listener. No Kubernetes secret reads or writes are part of this path.

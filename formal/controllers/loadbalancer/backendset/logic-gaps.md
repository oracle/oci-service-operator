---
schemaVersion: 1
surface: repo-authored-semantics
service: loadbalancer
slug: backendset
gaps: []
---

# Logic Gaps

## Current runtime path

- `BackendSet` reconciles through the generated `BackendSetServiceManager` under `pkg/servicemanager/loadbalancer/backendset`; shared CRUD orchestration still comes from `generatedruntime`, but a package-local client owns request-path lookup, the `loadBalancerId` and backend set name status mirrors, and a synthetic tracked ID in `status.status.ocid`.
- The package-local client resolves the OCI path identity from repo-authored fields: `loadBalancerId` from `status.loadBalancerId` or `spec.loadBalancerId`, and backend set name from `status.name`, `spec.name`, or the Kubernetes object name.
- Before create, the generated runtime probes `GetBackendSet` on that deterministic path. A matching backend set binds into the generated observe or update path; a 404 falls back through `ListBackendSets`, and an unmatched name proceeds to `CreateBackendSet`.

## Repo-authored semantics

- Create or bind is explicit. `BackendSet` reuses an existing backend set addressed by the same load balancer and backend set name instead of issuing a duplicate create, and create or bind persists a synthetic tracked ID so later update and delete retries stay on the same bound path even though OCI does not expose a distinct BackendSet OCID.
- Supported in-place updates are limited to `policy`, `backends`, `healthChecker`, `sslConfiguration`, `sessionPersistenceConfiguration`, and `lbCookieSessionPersistenceConfiguration`, matching `UpdateBackendSetDetails`. Drift on `loadBalancerId` or `name` remains create-only and is rejected before OCI mutation.
- `sessionPersistenceConfiguration` and `lbCookieSessionPersistenceConfiguration` remain mutually exclusive and are rejected before OCI mutation when both are set.
- The generated runtime follows create and update with read-based observation. Because the live `BackendSet` payload has no lifecycle field, the write path may requeue once before the next observe settles `Active`.
- Delete keeps the finalizer until `GetBackendSet` or the list fallback confirms the backend set is gone. No Kubernetes secret reads or writes are part of this path.

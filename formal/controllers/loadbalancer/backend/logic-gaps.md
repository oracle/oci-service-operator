---
schemaVersion: 1
surface: repo-authored-semantics
service: loadbalancer
slug: backend
gaps: []
---

# Logic Gaps

## Current runtime path

- `Backend` reconciles through the generated `BackendServiceManager` under `pkg/servicemanager/loadbalancer/backend`; shared CRUD orchestration still comes from `generatedruntime`, but a package-local wrapper owns Backend-specific path identity and bind logic.
- The package-local wrapper resolves the OCI path identity from repo-authored fields: `loadBalancerId`, `backendSetName`, and the backend name derived from the bound `status.name` or the desired `spec.ipAddress:spec.port`.
- Before create, the local wrapper calls `GetBackend` with that deterministic path identity. A matching backend binds into the generated observe/update path; a 404 falls through to `CreateBackend`.

## Repo-authored semantics

- Create or bind is explicit. `Backend` binds an existing backend addressed by the same load balancer, backend set, and derived backend name instead of issuing a duplicate create.
- Supported in-place updates are limited to `weight`, `backup`, `drain`, and `offline`, matching `UpdateBackendDetails`. Drift on `loadBalancerId`, `backendSetName`, `ipAddress`, or `port` remains create-only and is rejected before OCI mutation.
- The generated runtime follows create and update with read-based observation. Because the live `Backend` payload has no lifecycle field, the write path may requeue once before the next observe settles `Active`.
- Delete keeps the finalizer until `GetBackend` or the list fallback confirms the backend is gone. No Kubernetes secret reads or writes are part of this path.

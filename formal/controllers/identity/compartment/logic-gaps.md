---
schemaVersion: 1
surface: repo-authored-semantics
service: identity
slug: compartment
gaps: []
---

# Logic Gaps

## Current runtime path

- `Compartment` is the narrowed `identity` primary resource for `bk6.2` and now routes through the checked-in `pkg/servicemanager/identity/compartment` package via `CompartmentServiceManager`.
- The checked-in controller and service-manager use the shared `pkg/servicemanager/generatedruntime` path directly; there is no legacy adapter, webhook seam, or Kubernetes secret side effect in this rollout.
- Generated runtime resolution first honors an observed OCI ID and otherwise reuses a unique `ListCompartments` match on `compartmentId` plus `name` before create, then reads the live resource with `GetCompartment`.

## Repo-authored semantics

- Status projection is required. The current `Compartment` response is merged into the CR status after create, observe, or update, and `status.status.ocid` tracks the bound OCI identity.
- Lifecycle closure is explicit for this path: `CREATING` requeues as provisioning, `ACTIVE` and `INACTIVE` are treated as steady states, and delete retains the finalizer until `GetCompartment` or list fallback confirms `DELETED` or NotFound.
- Update follows OCI-supported metadata mutation only: `description`, `name`, and tags remain mutable, `compartmentId` stays create-only, and no additional endpoint or secret materialization happens in this path.

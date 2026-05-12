---
schemaVersion: 1
surface: repo-authored-semantics
service: wlms
slug: managedinstance
gaps: []
---

# Logic Gaps

## Current runtime path

- `ManagedInstance` keeps the generated controller, service-manager shell, and
  registration wiring, but the reviewed runtime contract is finalized in
  `pkg/servicemanager/wlms/managedinstance/managedinstance_runtime_client.go`.
- The pinned SDK exposes `GetManagedInstance`, `ListManagedInstances`, and
  `UpdateManagedInstance`, but it does not expose `CreateManagedInstance` or
  `DeleteManagedInstance`. The published runtime is therefore bind-existing
  plus update-only: reconcile binds by explicit `spec.id` or by a unique
  `ListManagedInstances` match on `spec.compartmentId` plus
  `spec.displayName`, optionally narrowed by `spec.pluginStatus`, and it fails
  instead of inventing a create path.
- Only `configuration` reconciles in place. Bind inputs stay replacement-only
  drift, and the runtime rejects direct-ID drift instead of falling back to a
  list-based adoption path.
- Visible WLMS mutation responses do not expose work-request identifiers or
  lifecycle states. The reviewed runtime therefore uses a bounded read-after-
  write convergence loop: `UpdateManagedInstance` applies the sparse
  configuration patch, status projects the returned `ManagedInstance` body, and
  the next reconcile rereads `GetManagedInstance` to confirm drift closure
  instead of inventing waiter or work-request polling.
- Delete is CR-local unbind only. Deleting the Kubernetes resource clears the
  finalizer immediately and never calls OCI delete helpers.

## Repo-authored semantics

- Status projection is required and publishes the bound managed-instance
  identity, display name, compartment, hostname, server count, plugin status,
  operating-system fields, configuration, and timestamps with no secret side
  effects.
- `pluginStatus` is a bind filter and observed status field, not a lifecycle
  progression signal. `ACTIVE` and `INACTIVE` settle as readable steady state
  observations, while service errors remain terminal failures.

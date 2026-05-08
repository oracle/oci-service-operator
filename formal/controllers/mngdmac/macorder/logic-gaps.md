---
schemaVersion: 1
surface: repo-authored-semantics
service: mngdmac
slug: macorder
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `mngdmac/MacOrder` row after the
published runtime replaced the scaffold baseline with an explicit
workrequest-aware contract.

## Current runtime path

- `MacOrder` keeps the generated controller, service-manager shell, and
  registration wiring, but the live published behavior is owned by
  `pkg/servicemanager/mngdmac/macorder/macorder_runtime_client.go` rather than
  the unmodified generated baseline.
- The vendored SDK exposes direct `CreateMacOrder`, `GetMacOrder`,
  `ListMacOrders`, and `UpdateMacOrder` operations plus `CancelMacOrder`,
  `ChangeMacOrderCompartment`, and service-local `GetWorkRequest` helpers.
  The reviewed runtime follows those service-local work requests for create,
  update, and delete. `CREATING` requeues as provisioning, `UPDATING`
  requeues as updating, `ACTIVE` settles success, `NEEDS_ATTENTION` and
  `FAILED` are terminal without requeue, and delete confirmation waits through
  `DELETING` until OCI reports `DELETED`, the order read-model reports
  `orderStatus=CANCELED`, or the resource becomes unreadable.
- Pre-create reuse is bounded and explicit. Existing-before-create lookup is
  skipped unless both `spec.compartmentId` and `spec.displayName` are set.
  When lookup is enabled, the runtime lists by exact `compartmentId` plus
  `displayName`, reuses only a unique candidate in reusable lifecycles
  (`ACTIVE`, `CREATING`, or `UPDATING`), and fails duplicate exact matches
  instead of guessing.
- Mutable drift is explicit: `displayName`, `ipRange`, `orderDescription`,
  `orderSize`, and `shape` reconcile in place. `compartmentId` and
  `commitmentTerm` stay replacement-only drift, and the runtime intentionally
  omits `ChangeMacOrderCompartment` from the published contract. Optional empty
  strings do not force blind clears for `displayName` or `ipRange`.
- Kubernetes delete is an explicit cancel contract, not a pretend
  `DeleteMacOrder` surface. The runtime sends `CancelMacOrder`, tracks the
  resulting work request, then rereads `GetMacOrder` or `ListMacOrders` until
  the order is unambiguously gone or canceled before releasing the finalizer.
- Required status projection remains part of the repo-authored contract. The
  runtime mirrors the observed `MacOrder` body, including `id`,
  `compartmentId`, `displayName`, `orderDescription`, `orderSize`,
  `isDocusigned`, `shape`, `commitmentTerm`, `orderStatus`, `lifecycleState`,
  billing timestamps, `cancelReason`, `timeCanceled`, and lifecycle details,
  alongside the shared OSOK status and async breadcrumbs.

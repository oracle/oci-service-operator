---
schemaVersion: 1
surface: repo-authored-semantics
service: core
slug: instance
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- `Instance` is the narrowed compute primary for the `core` rollout and stays under `pkg/servicemanager/core/instance`.
- Reconcile first honors a tracked OCI identity and otherwise reuses a unique `ListInstances` match on the configured filter set before launch. Successful create, observe, and update all project the live OCI `Instance` payload back into the CR status, including `status.id`, `status.lifecycleState`, and `status.status.ocid`.
- Success is OCI `RUNNING` or `STOPPED`. Requeue continues while OCI reports `PROVISIONING`, `STARTING`, `MOVING`, `STOPPING`, or `TERMINATING`.
- Delete is finalizer-backed: the controller issues `TerminateInstance`, then keeps the finalizer until OCI readback reports `TERMINATED` or stops finding the instance. Observed `TERMINATING` remains the intermediate delete state.
- Supported in-place updates are limited to `displayName`, `definedTags`, and `freeformTags`, matching the first landed parity slice from the donor compute manager while keeping launch-only inputs reviewable.
- Create-only drift remains explicit for `availabilityDomain`, `compartmentId`, `shape`, `shapeConfig`, `sourceDetails`, and `subnetId`. Secret side effects are out of scope because the compute instance runtime does not publish connection material.

## Why this row is seeded

- `core/Instance` is no longer intended to stay on the previous observe-only generated stub. The checked-in rollout now owns an explicit create/read/list/update/delete contract for compute instances in this branch.
- The older `computeinstance` manager from the donoftime fork is a behavior reference, but the landed path remains the current `core/instance` package plus `api/core/v1beta1` schema in this checkout.

---
schemaVersion: 1
surface: repo-authored-semantics
service: appmgmtcontrol
slug: monitoredinstance
gaps: []
---

# Logic Gaps

## Current runtime path

- `MonitoredInstance` keeps the generated controller, service-manager shell,
  and registration wiring, but the reviewed runtime contract is finalized in
  `pkg/servicemanager/appmgmtcontrol/monitoredinstance/monitoredinstance_runtime_client.go`.
- The pinned SDK exposes `GetMonitoredInstance` and
  `ListMonitoredInstances` only. The published runtime is therefore
  bind-existing and observe-only: reconcile binds by explicit
  `spec.instanceId` or by a unique `ListMonitoredInstances` match on
  `spec.compartmentId` plus `spec.displayName`, and it fails instead of
  inventing create, update, or delete paths.
- Direct-ID binding is explicit. The runtime preflights
  `GetMonitoredInstance` when `spec.instanceId` is set so a missing direct bind
  cannot silently fall back to list adoption.
- `ListMonitoredInstances` returns `MonitoredInstanceSummary` rather than the
  full `MonitoredInstance` body. The reviewed runtime therefore reruns
  `GetMonitoredInstance` immediately after a list-based bind so status projects
  timestamps and `lifecycleDetails` from the full body instead of settling on
  summary-only state.
- Delete is CR-local unbind only. Deleting the Kubernetes resource clears the
  finalizer immediately and never calls auxiliary monitoring-plugin or
  work-request helpers.

## Repo-authored semantics

- Status projection is required and publishes the bound monitored-instance
  identity, compartment, display name, management-agent identity, timestamps,
  monitoring state, lifecycle state, and lifecycle details with no secret side
  effects.
- Lifecycle handling is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` and `INACTIVE` settle success,
  `DELETING` and `DELETED` remain terminating observations, and `FAILED` is
  terminal failure.
- Bind inputs are replacement-only drift. `instanceId`, `compartmentId`, and
  `displayName` define the observe-only binding surface, and direct-ID drift
  never downgrades into list adoption.

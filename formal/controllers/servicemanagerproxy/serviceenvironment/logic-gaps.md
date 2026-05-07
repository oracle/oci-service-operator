---
schemaVersion: 1
surface: repo-authored-semantics
service: servicemanagerproxy
slug: serviceenvironment
gaps: []
---

# Logic Gaps

## Current runtime path

- `ServiceEnvironment` keeps the generated controller, service-manager shell,
  and registration wiring, but publishes the reviewed observe-only contract in
  `pkg/servicemanager/servicemanagerproxy/serviceenvironment/serviceenvironment_runtime_client.go`.
- The pinned SDK exposes `GetServiceEnvironment` and
  `ListServiceEnvironments` only. The published kind is therefore bind-existing
  and observe-only: reconcile binds by explicit `spec.serviceEnvironmentId`
  plus `spec.compartmentId`, records the returned Service Manager identifier in
  shared tracked status, and uses `ListServiceEnvironments` only as a
  confirmation fallback when `GetServiceEnvironment` no longer returns the
  tracked environment.
- No OCI create, update, or delete path is published. Delete is Kubernetes
  local finalizer cleanup only.

## Repo-authored semantics

- Status projection is required. The runtime mirrors the observed
  `ServiceEnvironment` body into the published status read model and tracks the
  bound Service Manager identifier through `status.status.ocid` even though the
  identifier is not an OCID.
- Lifecycle classification is explicit for the entitlement-state surface.
  `INITIALIZED` and `BEGIN_ACTIVATION` requeue as provisioning.
  `BEGIN_SOFT_TERMINATION`, `BEGIN_TERMINATION`, `BEGIN_DISABLING`,
  `BEGIN_ENABLING`, `BEGIN_MIGRATION`, `BEGIN_SUSPENSION`,
  `BEGIN_RESUMPTION`, `BEGIN_LOCK_RELOCATION`, `BEGIN_RELOCATION`,
  `BEGIN_UNLOCK_RELOCATION`, `BEGIN_DISABLING_ACCESS`, and
  `BEGIN_ENABLING_ACCESS` requeue as updating.
  Stable entitlement states `ACTIVE`, `SOFT_TERMINATED`, `CANCELED`,
  `TERMINATED`, `DISABLED`, `SUSPENDED`, `LOCKED_RELOCATION`, `RELOCATED`,
  `UNLOCKED_RELOCATION`, and `ACCESS_DISABLED` settle success while the raw SDK
  status stays visible on the CR.
  `FAILED_ACTIVATION`, `FAILED_MIGRATION`, `FAILED_LOCK_RELOCATION`, and
  `TRA_UNKNOWN` are terminal failure states.
- Mutation policy is bind-only. `serviceEnvironmentId` and `compartmentId` are
  replacement-only identity fields, and no in-place mutable OCI surface is
  published in this story.
- Delete confirmation is best-effort because there is no OCI delete call. Once
  the Kubernetes object is being removed, the runtime clears controller state
  locally and releases the finalizer without waiting on a cloud-side terminal
  state.

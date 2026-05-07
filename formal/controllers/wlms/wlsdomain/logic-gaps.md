---
schemaVersion: 1
surface: repo-authored-semantics
service: wlms
slug: wlsdomain
gaps: []
---

# Logic Gaps

## Current runtime path

- `WlsDomain` keeps the generated controller, service-manager shell, and
  registration wiring, but the reviewed runtime contract is finalized in
  `pkg/servicemanager/wlms/wlsdomain/wlsdomain_runtime_client.go`.
- WLMS v65.110.0 does not expose `CreateWlsDomain`. The published runtime is
  therefore manage-existing only: when no tracked OCI identifier is present,
  reconcile binds an existing domain by explicit `spec.id` or by a unique
  exact `ListWlsDomains` match on `compartmentId` plus `displayName`,
  optionally narrowed by `middlewareType` and `weblogicVersion`, and it fails
  instead of inventing a create path.
- Only `configuration`, `freeformTags`, and `definedTags` reconcile in place.
  `ChangeWlsDomainCompartment`, restart/stop/start/patch/scan helpers, and
  other WLMS auxiliaries remain out of scope for the first rollout, so
  identity fields and operational controls stay replacement-only drift.
- Lifecycle handling stays read-after-write and read-after-delete:
  `CREATING` requeues as provisioning, `UPDATING` requeues as updating,
  `ACTIVE` settles success, `DELETING` blocks finalizer release until
  confirm-delete completes, and `FAILED` plus `NEEDS_ATTENTION` are terminal
  failure states.
- `DeleteWlsDomain` returns only request headers and the visible mutation
  responses do not expose work-request IDs. The runtime therefore relies on
  `GetWlsDomain` and `ListWlsDomains` rereads, not service-local work-request
  polling, as the authoritative async signal.
- Required status projection keeps WLMS configuration, lifecycle details,
  patch-readiness, accepted-terms, and system tags on the published kind, and
  the runtime has no secret side effects.

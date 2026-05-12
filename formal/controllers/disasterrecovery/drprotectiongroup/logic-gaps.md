---
schemaVersion: 1
surface: repo-authored-semantics
service: disasterrecovery
slug: drprotectiongroup
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded
`disasterrecovery/DrProtectionGroup` row after the reviewed runtime contract
replaces the scaffold placeholder with the published work-request-backed
generated-runtime path.

## Current runtime contract

- `DrProtectionGroup` keeps the generated controller, service-manager shell,
  and registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/disasterrecovery/drprotectiongroup/drprotectiongroup_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/disasterrecovery/drprotectiongroup/drprotectiongroup_serviceclient.go`.
- The vendored SDK exposes
  `Create/Get/List/Update/DeleteDrProtectionGroup` plus
  `GetWorkRequest` and `ListWorkRequests`. The reviewed runtime keeps the
  published surface scoped to the base resource and treats
  `CREATING` as provisioning, `UPDATING` as updating, `DELETING` as
  terminating, `ACTIVE` and `INACTIVE` as settled success, and
  `NEEDS_ATTENTION` plus `FAILED` as terminal failure without requeue.
- Create, update, and delete are work-request-backed. `CreateDrProtectionGroup`
  returns a `DrProtectionGroup` body plus `opc-work-request-id`, while
  `UpdateDrProtectionGroup` and `DeleteDrProtectionGroup` return only
  work-request headers. The reviewed runtime persists the in-flight OCI work
  request in `status.async.current`, normalizes Disaster Recovery
  `OperationStatus*` values into shared async classes, maps
  `CREATE_DR_PROTECTION_GROUP`, `UPDATE_DR_PROTECTION_GROUP`, and
  `DELETE_DR_PROTECTION_GROUP` into create/update/delete phases, recovers the
  affected DR protection group OCID from work-request resources, and rereads
  `GetDrProtectionGroup` before settling create, update, or delete.
- Pre-create lookup is explicit. `ListDrProtectionGroups` always scopes by
  exact `compartmentId` plus `displayName`, and optionally narrows candidates
  with repo-authored exact-match filters for `lifecycleState`, `role`, and
  `lifecycleSubState` when those fields are set in the spec. Unique reusable
  matches in `ACTIVE`, `CREATING`, `UPDATING`, or `INACTIVE` bind; duplicate
  exact matches fail instead of guessing.
- Mutation policy stays aligned with `UpdateDrProtectionGroupDetails` for
  `displayName`, `logLocation`, `members`, `freeformTags`, and `definedTags`.
  `compartmentId` and create-only `association` stay replacement-only drift,
  and auxiliary families
  `AssociateDrProtectionGroup`, `DisassociateDrProtectionGroup`,
  `UpdateRoleDrProtectionGroup`, `ChangeDrProtectionGroupCompartment`, DR
  plans, and plan executions remain unpublished helper drift outside this
  rollout.
- The members list remains truthful and polymorphic. The published API surface
  keeps the member `memberType` discriminator and subtype payload instead of
  flattening the list into untyped maps, and the runtime rebuilds concrete SDK
  create and update member detail bodies from that typed CRD shape before OCI
  writes.
- Required status projection is part of the checked-in contract. The runtime
  projects `status.id`, `status.compartmentId`, `status.displayName`,
  `status.role`, `status.lifecycleState`, `status.lifecycleSubState`,
  `status.lifeCycleDetails`, `status.peerId`, `status.peerRegion`,
  `status.logLocation`, `status.members`, `status.timeCreated`,
  `status.timeUpdated`, `status.freeformTags`, `status.definedTags`, and
  `status.systemTags` from concrete `GetDrProtectionGroup` rereads because list
  summaries do not expose members or log location truthfully.
- Delete confirmation is required, not best-effort. The runtime does not report
  success until `GetDrProtectionGroup` confirms the tracked resource is gone or
  exposes lifecycle state `DELETED`, even when the work request itself reaches a
  terminal status first.

## Authority and scoped cleanup

- `formal/controllers/disasterrecovery/drprotectiongroup/*` is the
  authoritative formal path for the promoted `DrProtectionGroup` runtime
  contract.
- The published rollout keeps `secret_side_effects = none`.

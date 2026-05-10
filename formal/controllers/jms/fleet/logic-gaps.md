---
schemaVersion: 1
surface: repo-authored-semantics
service: jms
slug: fleet
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `jms/Fleet` row after the runtime
review replaced the scaffold semantics with the published work-request-backed
contract.

## Current runtime path

- `Fleet` keeps the generated controller, service-manager shell, and
  registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/jms/fleet/fleet_runtime_client.go` rather than the
  generated baseline in `pkg/servicemanager/jms/fleet/fleet_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes JMS
  `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `CANCELING`,
  `SUCCEEDED`, `FAILED`, and `CANCELED`) into shared async classes, maps
  `CREATE_FLEET`, `UPDATE_FLEET`, and `DELETE_FLEET` into
  create/update/delete phases, and resumes reconciliation from that shared
  async tracker across requeues.
- Create-time identity recovery is work-request-backed. Because
  `CreateFleetResponse`, `UpdateFleetResponse`, and `DeleteFleetResponse`
  expose work-request headers but do not return a `Fleet` body, the runtime
  recovers the affected Fleet OCID from work-request resources before
  rereading `GetFleet`.
- Bind resolution is bounded. When no OCI identifier is tracked, the runtime
  only attempts pre-create reuse when `spec.compartmentId` and
  `spec.displayName` are present and then adopts only a unique `ListFleets`
  match on exact `compartmentId` plus `displayName`. `ACTIVE`, `CREATING`,
  `UPDATING`, and `NEEDS_ATTENTION` summaries are reusable; `FAILED`,
  `DELETING`, and `DELETED` summaries are not; duplicate exact-name matches
  fail instead of binding arbitrarily.
- Mutable drift is limited to `displayName`, `description`, `inventoryLog`,
  `operationLog`, `isAdvancedFeaturesEnabled`, `freeformTags`, and
  `definedTags`. The handwritten update-body builder preserves clear-to-empty
  intent for `description` and both tag maps, keeps explicit `false` updates
  for `isAdvancedFeaturesEnabled`, and updates `inventoryLog` in place when
  the desired required log IDs differ from OCI. `operationLog` reconciles in
  place only when both log identifiers are supplied; an empty `operationLog`
  value is treated as omission because the published spec cannot distinguish
  omit from clear. `compartmentId` remains replacement-only drift, and
  `ChangeFleetCompartment` stays out of scope for the published runtime
  contract.
- `inventoryLog` remains a required published spec field because
  `CreateFleet` requires it from the first public contract even though
  `UpdateFleet` keeps it optional.
- Reviewed lifecycle mapping is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` and `NEEDS_ATTENTION` settle
  success, `FAILED` is terminal without requeue, and delete confirmation
  waits through `DELETING` until `DELETED` or NotFound.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and
  the published `status.id`, `status.displayName`, `status.description`,
  `status.compartmentId`, approximate inventory counters,
  `status.timeCreated`, `status.lifecycleState`, `status.inventoryLog`,
  `status.operationLog`, `status.isAdvancedFeaturesEnabled`,
  `status.isExportSettingEnabled`, `status.freeformTags`,
  `status.definedTags`, and `status.systemTags` read-model fields when OCI
  returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetFleet` or fallback `ListFleets`
  confirms the Fleet is gone or reports lifecycle state `DELETED`.

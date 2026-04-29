---
schemaVersion: 1
surface: repo-authored-semantics
service: clusterplacementgroups
slug: clusterplacementgroup
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded
`clusterplacementgroups/ClusterPlacementGroup` row after the runtime review
replaced the scaffold semantics with the published work-request-backed
contract.

## Current runtime path

- `ClusterPlacementGroup` keeps the generated controller, service-manager shell,
  and registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/clusterplacementgroups/clusterplacementgroup/clusterplacementgroup_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/clusterplacementgroups/clusterplacementgroup/clusterplacementgroup_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes
  `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `WAITING`,
  `NEEDS_ATTENTION`, `CANCELING`, `SUCCEEDED`, `FAILED`, and `CANCELED`) into
  shared async classes, maps
  `CREATE_CLUSTER_PLACEMENT_GROUP`,
  `UPDATE_CLUSTER_PLACEMENT_GROUP`, and
  `DELETE_CLUSTER_PLACEMENT_GROUP` into create/update/delete phases, and
  resumes reconciliation from that shared async tracker across requeues.
- Create-time identity recovery is work-request-backed. The runtime records the
  tracked ClusterPlacementGroup OCID from the create response body when OCI
  returns it and otherwise recovers the affected OCID from work-request
  resources before rereading `GetClusterPlacementGroup`.
- Bind resolution is bounded. When no OCI identifier is tracked, the runtime
  only attempts pre-create reuse when `spec.name` and
  `spec.availabilityDomain` are non-empty and then adopts only a unique
  `ListClusterPlacementGroups` match on exact `compartmentId`, `name`,
  `availabilityDomain`, and `clusterPlacementGroupType`. Summaries in
  `FAILED`, `DELETING`, or `DELETED` are not reused, and duplicate exact
  matches fail instead of binding arbitrarily.
- Mutable drift is limited to `description`, `freeformTags`, and
  `definedTags`. The handwritten update-body builder preserves clear-to-empty
  intent for description and both tag maps, while `name`,
  `availabilityDomain`, `clusterPlacementGroupType`, `compartmentId`,
  `placementInstruction`, and `capabilities` remain replacement-only drift.
  Auxiliary mutators `ActivateClusterPlacementGroup`,
  `DeactivateClusterPlacementGroup`, and
  `ChangeClusterPlacementGroupCompartment` stay out of scope for the published
  runtime.
- Reviewed lifecycle mapping is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` and `INACTIVE` settle success,
  `FAILED` is terminal without requeue, and delete confirmation waits through
  `DELETING` until `DELETED` or NotFound.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and
  the published `status.id`, `status.name`, `status.description`,
  `status.compartmentId`, `status.availabilityDomain`,
  `status.clusterPlacementGroupType`, `status.timeCreated`,
  `status.timeUpdated`, `status.lifecycleState`,
  `status.lifecycleDetails`, `status.placementInstruction`,
  `status.capabilities`, `status.freeformTags`, `status.definedTags`, and
  `status.systemTags` read-model fields when OCI returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetClusterPlacementGroup` confirms
  the ClusterPlacementGroup is gone or reports lifecycle state `DELETED`.

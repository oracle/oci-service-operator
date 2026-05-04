---
schemaVersion: 1
surface: repo-authored-semantics
service: delegateaccesscontrol
slug: delegationcontrol
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded
`delegateaccesscontrol/DelegationControl` row after the runtime review replaced
the scaffold semantics with the published work-request-backed contract.

## Current runtime path

- `DelegationControl` keeps the generated controller, service-manager shell,
  and registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/delegateaccesscontrol/delegationcontrol/delegationcontrol_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/delegateaccesscontrol/delegationcontrol/delegationcontrol_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes Delegate
  Access Control `OperationStatus*` values
  (`ACCEPTED`, `IN_PROGRESS`, `WAITING`, `CANCELING`, `SUCCEEDED`, `FAILED`,
  `CANCELED`, and `NEEDS_ATTENTION`) into shared async classes, maps
  `CREATE_DELEGATION_CONTROL`, `UPDATE_DELEGATION_CONTROL`, and
  `DELETE_DELEGATION_CONTROL` into create/update/delete phases, and resumes
  reconciliation from that shared async tracker across requeues.
- Create-time identity recovery is work-request-backed. The runtime prefers the
  create response body when OCI returns a `DelegationControl` identifier and
  otherwise recovers the OCID from work-request resources before reading the
  resource by ID and projecting status.
- Bind resolution is explicit and conservative. When no OCI identifier is
  tracked, the runtime only attempts pre-create reuse when
  `spec.compartmentId`, `spec.displayName`, `spec.resourceType`, and at least
  one `spec.resourceIds` entry are present, then lists by exact
  `compartmentId`, `displayName`, `resourceType`, and the first `resourceId`
  filter. It rereads each reusable candidate through `GetDelegationControl` and
  reuses only a unique candidate whose full `resourceIds` set and create-only
  vault inputs still match. Duplicate matches fail instead of guessing.
- Mutable drift is limited to `displayName`, `description`,
  `numApprovalsRequired`, `delegationSubscriptionIds`,
  `isAutoApproveDuringMaintenance`, `resourceIds`,
  `preApprovedServiceProviderActionNames`, `notificationTopicId`,
  `notificationMessageFormat`, `freeformTags`, and `definedTags`. The
  handwritten update builder preserves clear-to-empty intent for description,
  list fields, and tag maps. `compartmentId`, `resourceType`, `vaultId`, and
  `vaultKeyId` remain replacement-only drift, and the auxiliary
  `ChangeDelegationControlCompartment` operation stays out of scope for the
  published runtime.
- `vaultId` and `vaultKeyId` remain create-time only. OCI models them only for
  `resourceType=CLOUDVMCLUSTER`, so the reviewed runtime keeps that constraint
  explicit and never attempts in-place vault reconfiguration.
- Reviewed lifecycle mapping treats `CREATING` as provisioning, `UPDATING` as
  updating, `ACTIVE` as success, `DELETING` as terminating, `DELETED` as the
  delete-confirmation target, and `FAILED` plus `NEEDS_ATTENTION` as terminal
  failure without requeue.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and the
  published `status.id`, `status.displayName`, `status.compartmentId`,
  `status.resourceType`, `status.description`,
  `status.numApprovalsRequired`,
  `status.preApprovedServiceProviderActionNames`,
  `status.delegationSubscriptionIds`, `status.isAutoApproveDuringMaintenance`,
  `status.resourceIds`, `status.notificationTopicId`,
  `status.notificationMessageFormat`, `status.vaultId`, `status.vaultKeyId`,
  `status.timeCreated`, `status.timeUpdated`, `status.timeDeleted`,
  `status.lifecycleState`, `status.lifecycleStateDetails`,
  `status.freeformTags`, `status.definedTags`, and `status.systemTags`
  read-model fields when OCI returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetDelegationControl` or fallback
  `ListDelegationControls` confirms the resource is gone or exposes lifecycle
  state `DELETED`.

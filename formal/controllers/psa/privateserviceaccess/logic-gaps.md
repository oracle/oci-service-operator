---
schemaVersion: 1
surface: repo-authored-semantics
service: psa
slug: privateserviceaccess
gaps: []
---

# Logic Gaps

## Current runtime contract

- `PrivateServiceAccess` keeps the generated controller, service-manager shell,
  and registration wiring, but the published runtime contract is owned by
  `pkg/servicemanager/psa/privateserviceaccess/privateserviceaccess_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/psa/privateserviceaccess/privateserviceaccess_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes PSA
  `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `WAITING`,
  `NEEDS_ATTENTION`, `FAILED`, `SUCCEEDED`, `CANCELLING`, and `CANCELLED`)
  into shared async classes, maps
  `CREATE_PRIVATE_SERVICE_ACCESS`,
  `UPDATE_PRIVATE_SERVICE_ACCESS`, and
  `DELETE_PRIVATE_SERVICE_ACCESS`
  into create/update/delete phases, and resumes reconciliation from that shared
  async tracker across requeues.
- Create-time identity recovery is work-request-backed. The runtime prefers the
  create response body when OCI returns a `PrivateServiceAccess` identifier and
  otherwise recovers the OCID from work-request resources before rereading
  `GetPrivateServiceAccess`.
- Bind resolution is explicit and conservative. When no OCI identifier is
  tracked, the runtime only attempts pre-create reuse when
  `spec.compartmentId`, `spec.subnetId`, `spec.serviceId`, and
  `spec.displayName` are all non-empty, then adopts only a unique exact
  `ListPrivateServiceAccesses` match on `compartmentId`, `displayName`, and
  `serviceId` whose returned `subnetId` also matches and whose lifecycle is
  reusable (`ACTIVE`, `CREATING`, or `UPDATING`). Duplicate exact matches fail
  instead of binding arbitrarily.
- Mutable drift is limited to `definedTags`, `freeformTags`,
  `securityAttributes`, `displayName`, `description`, and `nsgIds`. The
  handwritten update-body builder preserves clear-to-empty intent for
  description, tag maps, and `nsgIds`, while `compartmentId`, `subnetId`,
  `serviceId`, and `ipv4Ip` remain replacement-only drift.
  `ChangePrivateServiceAccessCompartment` stays out of scope for the published
  runtime.
- Reviewed lifecycle mapping treats `CREATING` as provisioning, `UPDATING` as
  updating, `ACTIVE` as success, `DELETING` as terminating, `DELETED` as the
  delete-confirmation target, and `FAILED` plus `NEEDS_ATTENTION` as terminal
  failure without requeue.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and the
  published `status.id`, `status.displayName`, `status.compartmentId`,
  `status.vcnId`, `status.subnetId`, `status.vnicId`,
  `status.lifecycleState`, `status.serviceId`, `status.fqdns`,
  `status.definedTags`, `status.freeformTags`, `status.systemTags`,
  `status.securityAttributes`, `status.description`, `status.timeCreated`,
  `status.timeUpdated`, `status.nsgIds`, and `status.ipv4Ip` read-model
  fields when OCI returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetPrivateServiceAccess` or fallback
  `ListPrivateServiceAccesses` confirms the resource is gone or exposes
  lifecycle state `DELETED`.

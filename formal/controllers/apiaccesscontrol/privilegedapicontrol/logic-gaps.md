---
schemaVersion: 1
surface: repo-authored-semantics
service: apiaccesscontrol
slug: privilegedapicontrol
gaps: []
---

# Logic Gaps

## Current runtime contract

- `PrivilegedApiControl` keeps the generated controller, service-manager shell,
  and registration wiring, but the published runtime contract is owned by
  `pkg/servicemanager/apiaccesscontrol/privilegedapicontrol/privilegedapicontrol_runtime_client.go`
  rather than the generated helper baseline in
  `pkg/servicemanager/apiaccesscontrol/privilegedapicontrol/privilegedapicontrol_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes
  API Access Control `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`,
  `WAITING`, `NEEDS_ATTENTION`, `FAILED`, `SUCCEEDED`, `CANCELING`, and
  `CANCELED`) into shared async classes, maps
  `CREATE_PRIVILEGED_API_CONTROL`, `UPDATE_PRIVILEGED_API_CONTROL`, and
  `DELETE_PRIVILEGED_API_CONTROL` into create/update/delete phases, and resumes
  reconciliation from that shared async tracker across requeues.
- Create-time identity recovery is work-request-backed. The runtime prefers the
  create response body when OCI returns a `PrivilegedApiControl` identifier and
  otherwise recovers the OCID from work-request resources before reading the
  resource by ID and projecting status.
- Bind resolution is explicit and conservative. When no OCI identifier is
  tracked, the runtime only attempts pre-create reuse when
  `spec.compartmentId`, `spec.displayName`, and `spec.resourceType` are all
  non-empty, then adopts only a unique exact `ListPrivilegedApiControls` match
  on `compartmentId`, `displayName`, and `resourceType` in reusable
  lifecycles (`ACTIVE`, `CREATING`, or `UPDATING`). Duplicate exact-name
  matches fail instead of binding arbitrarily.
- Mutable drift is limited to `displayName`, `description`,
  `notificationTopicId`, `approverGroupIdList`, `resourceType`, `resources`,
  `privilegedOperationList`, `numberOfApprovers`, `freeformTags`, and
  `definedTags`. The handwritten update-body builder preserves clear-to-empty
  intent for `description`, list fields, and tag maps. `compartmentId` remains
  replacement-only drift, and the auxiliary
  `ChangePrivilegedApiControlCompartment` operation stays out-of-scope for the
  published runtime.
- Reviewed lifecycle mapping treats `CREATING` as provisioning, `UPDATING` as
  updating, `ACTIVE` as success, `DELETING` as terminating, `DELETED` as the
  delete-confirmation target, and `FAILED` plus `NEEDS_ATTENTION` as terminal
  failure without requeue.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and the
  published `status.id`, `status.displayName`, `status.compartmentId`,
  `status.timeCreated`, `status.timeUpdated`, `status.timeDeleted`,
  `status.state`, `status.lifecycleState`, `status.description`,
  `status.notificationTopicId`, `status.approverGroupIdList`,
  `status.resourceType`, `status.resources`, `status.privilegedOperationList`,
  `status.numberOfApprovers`, `status.freeformTags`, `status.definedTags`,
  `status.systemTags`, `status.stateDetails`, and `status.lifecycleDetails`
  read-model fields when OCI returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetPrivilegedApiControl` or fallback
  `ListPrivilegedApiControls` confirms the resource is gone or exposes
  lifecycle state `DELETED`.

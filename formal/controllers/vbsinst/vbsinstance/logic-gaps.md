---
schemaVersion: 1
surface: repo-authored-semantics
service: vbsinst
slug: vbsinstance
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `vbsinst/VbsInstance` row after the
runtime review replaced the scaffold semantics with the published
work-request-backed contract.

## Current runtime path

- `VbsInstance` keeps the generated controller, service-manager shell, and
  registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/vbsinst/vbsinstance/vbsinstance_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/vbsinst/vbsinstance/vbsinstance_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes VBS
  `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `CANCELING`,
  `SUCCEEDED`, `FAILED`, and `CANCELED`) into shared async classes, maps
  `CREATE_VBS_INSTANCE`, `UPDATE_VBS_INSTANCE`, and `DELETE_VBS_INSTANCE`
  into create/update/delete phases, and resumes reconciliation from that
  shared async tracker across requeues.
- Create-time identity recovery is work-request-backed. Because
  `CreateVbsInstanceResponse`, `UpdateVbsInstanceResponse`, and
  `DeleteVbsInstanceResponse` expose work-request headers but do not return a
  `VbsInstance` body, the runtime recovers the affected resource OCID from the
  work-request resources before rereading `GetVbsInstance`.
- Bind resolution is bounded. When no OCI identifier is tracked, the runtime
  only attempts pre-create reuse when `spec.compartmentId` and `spec.name` are
  present and then adopts only a unique `ListVbsInstances` match on exact
  `compartmentId` plus `name`. `ACTIVE`, `CREATING`, and `UPDATING` summaries
  are reusable; duplicate exact-name matches fail instead of binding
  arbitrarily; `displayName` is not part of the identity contract.
- Mutable drift is limited to `displayName`,
  `isResourceUsageAgreementGranted`, `resourceCompartmentId`, `freeformTags`,
  and `definedTags`. The handwritten update-body builder preserves
  clear-to-empty intent for the tag maps, while `name` remains create-time
  identity and `compartmentId` stays out of scope for the published
  create/get/list/update/delete contract even though the provider also exposes
  `ChangeVbsInstanceCompartment`.
- The provider-only `idcs_access_token` header input is intentionally not part
  of the published CRD surface for `VbsInstance`. The reviewed runtime keeps
  the controller-backed contract limited to the generated spec fields plus the
  shared async tracker.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and
  the published `status.id`, `status.name`, `status.displayName`,
  `status.compartmentId`, `status.isResourceUsageAgreementGranted`,
  `status.resourceCompartmentId`, `status.vbsAccessUrl`, `status.timeCreated`,
  `status.timeUpdated`, `status.lifecycleState`, `status.lifecycleDetails`,
  `status.freeformTags`, `status.definedTags`, and `status.systemTags`
  read-model fields when VBS returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetVbsInstance` or fallback
  `ListVbsInstances` confirms the `VbsInstance` is gone or reports lifecycle
  state `DELETED`.

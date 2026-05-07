---
schemaVersion: 1
surface: repo-authored-semantics
service: opa
slug: opainstance
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `opa/OpaInstance` row after the
runtime review replaced the scaffold placeholder with the published
work-request-backed contract.

## Current runtime path

- `OpaInstance` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/opa/opainstance/opainstance_runtime_client.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes OPA
  `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `WAITING`,
  `CANCELING`, `SUCCEEDED`, `FAILED`, and `CANCELED`) into shared async
  classes, maps `CREATE_OPA_INSTANCE`, `UPDATE_OPA_INSTANCE`, and
  `DELETE_OPA_INSTANCE` into create/update/delete phases, and resumes
  reconciliation from that shared async tracker across requeues.
- Create-time identity recovery is work-request-backed. The runtime records the
  tracked OpaInstance OCID when OCI exposes it and otherwise resolves the
  created resource OCID from work-request resources before reading the
  OpaInstance by ID and projecting status.
- Bind resolution is bounded. When no OCI identifier is tracked, the runtime
  only attempts pre-create reuse when `spec.compartmentId` and
  `spec.displayName` are both present and then adopts only a unique
  `ListOpaInstances` match on exact `compartmentId` plus `displayName`.
  Summaries in `FAILED`, `DELETING`, or `DELETED` are not reused, and duplicate
  exact-name matches fail instead of binding arbitrarily.
- Mutable drift is limited to `displayName`, `description`, `freeformTags`,
  and `definedTags`. The handwritten update-body builder preserves
  clear-to-empty intent for description and both tag maps, while
  `compartmentId`, `shapeName`, `consumptionModel`, `meteringType`, `idcsAt`,
  and `isBreakglassEnabled` remain replacement-only drift.
- `IdcsAt` remains a create-time input only. OCI does not echo it back on
  `OpaInstance`, so the reviewed runtime intentionally normalizes it out of
  post-create parity checks instead of treating the missing value as drift.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and the
  published `status.id`, `status.displayName`, `status.compartmentId`,
  `status.shapeName`, `status.lifecycleState`, `status.description`,
  `status.instanceUrl`, `status.consumptionModel`, `status.meteringType`,
  `status.timeCreated`, `status.timeUpdated`, `status.identityAppGuid`,
  `status.identityAppDisplayName`, `status.identityDomainUrl`,
  `status.identityAppOpcServiceInstanceGuid`, `status.isBreakglassEnabled`,
  `status.freeformTags`, `status.definedTags`, `status.systemTags`, and
  `status.attachments` read-model fields when OPA returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetOpaInstance` confirms the
  resource is gone or reports lifecycle state `DELETED`; NotFound is also
  treated as delete confirmation.
- Provider auxiliary mutators `ChangeOpaInstanceCompartment`,
  `StartOpaInstance`, and `StopOpaInstance` remain out-of-scope drift for the
  published runtime surface. The service-local `WorkRequest*` helper kinds and
  the attachment-related auxiliary operations remain unpublished helper
  surfaces.

---
schemaVersion: 1
surface: repo-authored-semantics
service: ailanguage
slug: project
gaps: []
---

# Logic Gaps

## Current runtime contract

- `Project` keeps the generated controller, service-manager shell, and
  registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/ailanguage/project/project_runtime_client.go` rather than
  the generated helper/read-after-write baseline in
  `pkg/servicemanager/ailanguage/project/project_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes AI Language
  `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `WAITING`,
  `CANCELING`, `SUCCEEDED`, `CANCELED`, and `NEEDS_ATTENTION`) into shared
  async classes, normalizes work-request `OperationType*` values into
  create/update/delete phases, and resumes reconciliation from that shared
  async tracker across requeues.
- Create-time identity recovery is work-request-backed. The runtime records the
  create response Project OCID when OCI returns it, otherwise resolves the
  created Project OCID from work-request resources before reading the Project
  by ID and projecting status.
- Bind resolution uses an explicit OCI identity first. When no OCI identifier
  is tracked, the runtime may adopt only a unique paginated non-`DELETED`
  `ListProjects` match on `compartmentId` plus `displayName`. Delete fallback
  reuses `status.compartmentId`, `status.displayName`, and tracked `projectId`
  when `GetProject` returns NotFound so finalizer release does not depend on
  stale spec values.
- The audited v65.110.0 request surface stays explicit. `ListProjects` keeps
  optional tracked-identity query field `projectId`, while `GetProject`,
  `UpdateProject`, and `DeleteProject` use only path `projectId`; the service
  SDK does not expose lock-override query fields for this resource, so the
  handwritten runtime scope remains unchanged.
- Mutable drift is limited to `displayName`, `description`, `freeformTags`, and
  `definedTags`. `compartmentId` remains replacement-only drift, so the live
  runtime never exercises provider-only `ChangeProjectCompartment` even though
  that auxiliary operation remains part of the imported provider surface.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and the
  published `status.id`, `status.compartmentId`, `status.timeCreated`,
  `status.lifecycleState`, `status.displayName`, `status.description`,
  `status.timeUpdated`, `status.lifecycleDetails`, `status.freeformTags`,
  `status.definedTags`, and `status.systemTags` read-model fields when OCI
  returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetProject` or fallback
  `ListProjects` confirms the Project is gone or exposes lifecycle state
  `DELETED`.
- Retryable update conflicts stay explicit and resumable. When OCI returns a
  409 "currently being modified" response without a work-request ID, the
  runtime projects a lifecycle-sourced pending update async state, rereads the
  Project before retrying, clears that async state if mutable drift has already
  converged, and otherwise keeps the update pending until the standard requeue
  backoff has elapsed instead of hot-looping `UpdateProject` on every `ACTIVE`
  reread.

## Authority and scoped cleanup

- `formal/controllers/ailanguage/project/*` is the authoritative formal path
  for the promoted `ailanguage/Project` work-request-backed runtime contract.
- `pkg/servicemanager/ailanguage/project/project_runtime_client.go` and
  `pkg/servicemanager/ailanguage/project/project_runtime_client_test.go` own
  the live runtime behavior.
- `pkg/servicemanager/ailanguage/project/project_serviceclient.go` still
  records the generated helper baseline and should not be treated as the active
  execution contract.

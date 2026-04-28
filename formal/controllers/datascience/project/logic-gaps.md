---
schemaVersion: 1
surface: repo-authored-semantics
service: datascience
slug: project
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `datascience/Project` row after the
runtime audit replaced the scaffold placeholder with the reviewed
generated-runtime contract.

## Current runtime path

- `Project` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/datascience/project/project_runtime_client.go`.
- The handwritten runtime config binds `CreateProject`, `GetProject`,
  `ListProjects`, `UpdateProject`, and `DeleteProject` through
  `generatedruntime.ServiceClient` rather than a service-local legacy adapter.
  The reviewed hooks narrow reusable `ListProjects` requests to
  `compartmentId`, `displayName`, and optional tracked `id`, and they clear
  the generated auxiliary `ChangeProjectCompartment` helper so the runtime
  stays on plain create/get/list/update/delete semantics.
- Lifecycle handling is explicit: create and update settle directly on
  `ACTIVE`, and delete confirmation waits through `DELETING` until `DELETED`
  or NotFound via `GetProject` plus `ListProjects` fallback when identity must
  be re-resolved.
- Required status projection remains part of the repo-authored contract. The
  generated runtime stamps OSOK lifecycle conditions plus the published
  `status.id`, `status.timeCreated`, `status.displayName`,
  `status.compartmentId`, `status.createdBy`, `status.lifecycleState`,
  `status.description`, `status.freeformTags`, `status.definedTags`, and
  `status.systemTags` read-model fields when OCI returns them.

## Repo-authored semantics

- Pre-create lookup is explicit. When no OCI identifier is already tracked, the
  generated runtime queries `ListProjects` with the identifying request shape
  `compartmentId`, `displayName`, and optional tracked `id`; the OCI list
  request also exposes `lifecycleState`, `createdBy`, `sortBy`, `sortOrder`,
  `limit`, and `page`, but the runtime does not send those provider-only
  filters for identity reuse and instead evaluates reusability from the
  returned summary payload.
- Mutation policy is explicit: only `displayName`, `description`,
  `freeformTags`, and `definedTags` reconcile in place. `compartmentId` stays
  replacement-only drift even though the provider exposes
  `ChangeProjectCompartment`; the runtime validates that drift against live
  `GetProject` responses, never calls the auxiliary compartment-move helper,
  and skips `UpdateProject` when the mutable surface already matches the live
  OCI response.
- Create, update, and delete use plain provider helper semantics
  (`tfresource.CreateResource`, `tfresource.UpdateResource`,
  `tfresource.DeleteResource`) with read-after-write follow-up for create and
  update plus confirm-delete follow-up for delete. The runtime does not add
  service-local work-request polling or Kubernetes secret side effects.
- Delete keeps the finalizer until `GetProject` or fallback `ListProjects`
  confirms the project is gone; a `DELETING` list summary keeps reconcile in
  the terminating bucket instead of removing the finalizer early.

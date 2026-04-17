---
schemaVersion: 1
surface: repo-authored-semantics
service: aidocument
slug: project
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `aidocument/Project` row after the
runtime audit replaced the scaffold placeholder with the reviewed
generated-runtime contract.

## Current runtime path

- `Project` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/aidocument/project/project_runtime_client.go`.
- The handwritten runtime config binds `CreateProject`, `GetProject`,
  `ListProjects`, `UpdateProject`, and `DeleteProject` through
  `generatedruntime.ServiceClient` rather than a service-local legacy adapter.
- Lifecycle handling is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` settles success, and delete
  confirmation waits through `DELETING` until `DELETED` or NotFound via
  `GetProject` plus `ListProjects` fallback when identity must be re-resolved.
- Required status projection remains part of the repo-authored contract. The
  generated runtime stamps OSOK lifecycle conditions plus the published
  `status.id`, `status.compartmentId`, `status.timeCreated`,
  `status.lifecycleState`, `status.displayName`, `status.description`,
  `status.timeUpdated`, `status.lifecycleDetails`, `status.freeformTags`,
  `status.definedTags`, and `status.systemTags` read-model fields when OCI
  returns them.

## Repo-authored semantics

- Pre-create lookup is explicit. When no OCI identifier is already tracked, the
  generated runtime queries `ListProjects` with the identifying request shape
  `compartmentId`, `displayName`, and optional tracked `id`; the OCI list
  request also exposes `lifecycleState`, `sortBy`, `sortOrder`, `limit`, and
  `page`, but reusable matching remains a repo-authored decision layered on top
  of those provider facts.
- Mutation policy is explicit: only `displayName`, `description`,
  `freeformTags`, and `definedTags` reconcile in place. `compartmentId` stays
  replacement-only drift and the runtime skips `UpdateProject` when the mutable
  surface already matches the live OCI response.
- Create, update, and delete use plain provider helper semantics
  (`tfresource.CreateResource`, `tfresource.UpdateResource`,
  `tfresource.DeleteResource`) with read-after-write follow-up for create and
  update plus confirm-delete follow-up for delete. The runtime does not add
  service-local work-request polling or Kubernetes secret side effects.
- Delete keeps the finalizer until `GetProject` or fallback `ListProjects`
  confirms the project is gone; a `DELETING` list summary keeps reconcile in
  the terminating bucket instead of removing the finalizer early.

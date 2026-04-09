---
schemaVersion: 1
surface: repo-authored-semantics
service: dataflow
slug: application
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `dataflow/Application` row after the
active generatedruntime-backed override semantics were encoded into the
repo-authored formal metadata and regenerated diagrams.

## Current runtime path

- `Application` keeps the generated `ApplicationServiceManager` shell but
  overrides the generated client seam in
  `pkg/servicemanager/dataflow/application/application_runtime.go`. The active
  path is an `applicationGeneratedRuntimeClient` wrapper around
  `generatedruntime.ServiceClient`, not a standalone handwritten fallback.
- Reconcile is tracked-identity-first and never binds by list lookup. When
  `status.status.ocid` or `status.id` is present, the wrapper preflights that
  OCID through `GetApplication`, projects the live payload into status,
  validates create-only drift for `compartmentId` and `type`, and only enters
  create when no tracked identity remains.
- If a tracked application disappears or comes back with lifecycle `DELETED`,
  the wrapper clears the recorded identity and reruns `CreateApplication`
  instead of rebinding by display name or leaving the resource in a retry-only
  steady state.
- Create, observe, and update treat OCI `ACTIVE` and `INACTIVE` as successful
  steady states. Delete stays finalizer-backed and only completes once
  `GetApplication` returns not found; a `DELETED` lifecycle readback remains
  terminating during delete confirmation.
- Imported provider facts now cover `CreateApplication`, `GetApplication`,
  `UpdateApplication`, and `DeleteApplication`, matching the active
  generatedruntime-backed override surface.

## Repo-authored semantics

- The generatedruntime delegate uses custom create and update body builders so
  Data Flow enum validation, optional-field omission, and the active OCI
  request shape stay explicit in checked-in code instead of falling back to
  generic JSON projection.
- `spec.execute` has higher precedence than `className`, `fileUri`,
  `arguments`, `configuration`, and `parameters`. When `execute` is set,
  create and update intentionally suppress those subordinate fields rather than
  emitting conflicting drift.
- Status projection fully replaces `ApplicationStatus` from the latest OCI
  payload while preserving OSOK bookkeeping, so optional fields omitted by
  create, get, or update responses clear out of status instead of lingering
  from older reads.
- No Kubernetes Secret reads or writes are part of this resource.

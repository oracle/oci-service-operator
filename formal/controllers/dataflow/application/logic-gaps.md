---
schemaVersion: 1
surface: repo-authored-semantics
service: dataflow
slug: application
gaps: []
---

# Logic Gaps

## Current runtime path

- `Application` now uses a package-local generatedruntime wrapper for create,
  update, and steady-state observation, while delete confirmation remains
  package-local explicit logic in `application_runtime.go`.
- Fresh reconciles do not use list-based bind-or-reuse semantics. When no
  tracked `status.osokStatus.ocid` exists, the runtime goes straight to
  `CreateApplication`.
- When a tracked `status.osokStatus.ocid` exists, the wrapper first
  `GetApplication`s that OCI identity. Confirmed OCI not-found or observed
  `DELETED` clears the tracked identity and immediately recreates instead of
  attempting shared list fallback.
- Success is OCI `ACTIVE` or `INACTIVE`. Unknown lifecycle values fail rather
  than requeue.
- Delete confirmation is absence-based: `DeleteApplication` is followed by
  `GetApplication` until OCI no longer finds the resource. Observed `ACTIVE`,
  `INACTIVE`, and `DELETING` after the delete request remain terminating
  intermediate states that keep the finalizer in place, while `DELETED` or
  NotFound confirms removal.

## Repo-authored semantics

- Create request shaping is package-local and preserves the handwritten contract
  for zero-value omission on optional nested bodies.
- `spec.execute` keeps precedence over `className`, `fileUri`, `arguments`,
  `configuration`, and `parameters` for both create and update request shaping.
- Supported in-place updates are limited to the package-local
  `UpdateApplicationDetails` builder surface: `archiveUri`,
  `applicationLogConfig`, `arguments`, `className`, `configuration`,
  `definedTags`, `description`, `displayName`, `driverShape`,
  `driverShapeConfig`, `execute`, `executorShape`, `executorShapeConfig`,
  `fileUri`, `freeformTags`, `idleTimeoutInMinutes`, `language`,
  `logsBucketUri`, `maxDurationInMinutes`, `metastoreId`, `numExecutors`,
  `parameters`, `poolId`, `privateEndpointId`, `sparkVersion`, and
  `warehouseBucketUri`.
- Create-only drift is rejected for `compartmentId` and `type`, matching the
  previous handwritten runtime and tests.
- Status projection remains required. Create and update project directly from
  the immediate OCI response body, while delete-phase status updates still use
  the package-local explicit status projector so `ACTIVE`, `INACTIVE`, and
  `DELETING` delete-confirmation reads stay visible in the CR status.

## Why this row is seeded

- Runtime ownership is now explicit: package-local generatedruntime wrapper for
  create/update plus package-local explicit delete confirmation.
- The row has no open logic gaps for lifecycle classification, mutation policy,
  tracked-identity recreation, or delete confirmation semantics.

---
schemaVersion: 1
surface: repo-authored-semantics
service: dataflow
slug: application
gaps: []
---

# Logic Gaps

## Current runtime intent

- `Application` continues to use the handwritten runtime override under `pkg/servicemanager/dataflow/application` through the generated `newApplicationServiceClient` seam. This seeded formal row exists to capture the current repo-owned contract and regenerate the generatedruntime baseline without replacing the active handwritten path yet.
- Reconcile is tracked-identity-first and does not bind by list lookup. When `status.status.ocid` is empty the runtime issues `CreateApplication`; when `GetApplication` later returns not found for a tracked OCID, the runtime clears the recorded identity and recreates instead of adopting by display name.
- Status projection is required. The runtime mirrors the live OCI `Application` payload into the published `ApplicationStatus`, including lifecycle, tags, parameters, shape config, timestamps, owner fields, and `status.status.ocid`.
- Mutable drift is explicit and repo-authored. The runtime updates display name, shapes, executor count, spark version, file and execute inputs, parameters, tags, logging and metastore fields, warehouse and archive URIs, and related optional nested config while rejecting create-only drift for `compartmentId` and `type`.
- `execute` precedence is an active handwritten behavior. When `spec.execute` is set, subordinate `className`, `fileUri`, `arguments`, `configuration`, and `parameters` differences are intentionally ignored so reconcile follows the Data Flow service contract instead of emitting conflicting updates.
- Delete confirmation is required and OCI-read-backed. The runtime issues `DeleteApplication` for the tracked OCID, keeps the finalizer while readback still returns an `Application` payload, and only finishes once `GetApplication` returns not found. No secret reads or writes are part of this path.

## Out of scope follow-on work

- This metadata-seeding pass does not promote the generatedruntime `Application` baseline into the active reconcile path. Replacing the handwritten runtime remains follow-on work.

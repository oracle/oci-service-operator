---
schemaVersion: 1
surface: repo-authored-semantics
service: logging
slug: log
gaps: []
---

# Logic Gaps

## Current runtime path

- `Log` uses the generated `LogServiceManager` and generated controller shell,
  but keeps one package-local runtime companion in
  `pkg/servicemanager/logging/log/log_runtime_client.go` so the
  path-scoped `logGroupId` behavior stays under `logging/log` rather than
  leaking into `pkg/servicemanager/generatedruntime`.
- The Logging Management API scopes `CreateLog`, `GetLog`, `ListLogs`,
  `UpdateLog`, and `DeleteLog` under a log-group path. The package-local
  companion keeps create and bind anchored to `spec.logGroupId`, then switches
  tracked reads, updates, and deletes to `status.logGroupId` once OCI identity
  has been recorded so immutable `logGroupId` drift cannot orphan delete
  cleanup.
- Pre-create lookup is explicit: `ListLogs` stays scoped to the desired log
  group and reuses only a single candidate that matches `displayName`,
  `logType`, `sourceService`, and `sourceResource` in reusable lifecycle
  states. Duplicate matches fail instead of guessing.
- Success is OCI `ACTIVE` or `INACTIVE`. Requeue continues while OCI reports
  `CREATING`, `UPDATING`, or `DELETING`, and delete keeps the finalizer until
  `GetLog` or list fallback stops finding the tracked log.
- Supported in-place updates are limited to `displayName`, `isEnabled`,
  `definedTags`, `freeformTags`, and `retentionDuration`. `logGroupId`,
  `logType`, and the `configuration` block remain create-time only drift and
  are rejected before OCI update calls.
- `CreateLog`, `UpdateLog`, and `DeleteLog` return request/work-request
  headers, so the generated runtime follows create and update with read-based
  status projection and uses confirm-delete rereads before finalizer release.
- No Kubernetes secret reads or writes are part of `Log`.

## Why this row is seeded

- `logging/Log` is the first priority logging resource on the baseline
  generatedruntime path for this rollout, so the checked-in branch now owns an
  explicit repo-authored create/update/delete contract for it.

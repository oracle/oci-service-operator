---
schemaVersion: 1
surface: repo-authored-semantics
service: usageapi
slug: schedule
gaps: []
---

# Logic Gaps

## Current runtime path

- `Schedule` keeps the generated `ScheduleServiceManager` scaffold and generated runtime client configuration on the checked-in `usageapi` path.
- The generated runtime reuses an existing schedule before create by calling `ListSchedules` with the required `compartmentId` request scope plus the desired `name` filter, then binding on the user-unique schedule name returned in the list summary before `GetSchedule` loads the full payload for drift evaluation.
- Delete confirmation resolves tracked OCI identities through `GetSchedule`; when no `status.status.ocid` is recorded, the runtime falls back to the same list-on-name lookup before issuing `DeleteSchedule`.

## Repo-authored semantics

- The Usage API exposes `lifecycleState` directly on `Schedule` bodies, so create and update settle from the direct response plus normal `GetSchedule` rereads without a service-local synchronous wrapper. Both `ACTIVE` and `INACTIVE` are treated as settled resource-exists states for the OSOK contract.
- Supported in-place updates are limited to `description`, `outputFileFormat`, `resultLocation`, `freeformTags`, and `definedTags`, matching `UpdateScheduleDetails`. Drift for `name`, `compartmentId`, `scheduleRecurrences`, `timeScheduled`, `savedReportId`, and `queryProperties` is replacement-only and is rejected before OCI mutation.
- Status projection remains required. The generated runtime merges the live OCI `Schedule` body into the published status read-model fields while still stamping `status.status.ocid` and shared OSOK request breadcrumbs.
- Delete is explicit and required. The controller retains the finalizer until `DeleteSchedule` succeeds and a follow-up `GetSchedule` confirms OCI no longer returns the schedule.
- No repo-authored secret reads or writes are part of this path.

## Why this row is seeded

- The checked-in generated controller and service-manager path now has an explicit bind-or-create answer, an in-place update boundary, and required delete confirmation for `usageapi/Schedule`.
- No open formal gaps remain for the current generatedruntime contract.

---
schemaVersion: 1
surface: repo-authored-semantics
service: databasemigration
slug: assessment
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `databasemigration/Assessment` row
after the runtime review replaced the scaffold placeholder with the published
work-request-backed generated-runtime contract.

## Current runtime path

- `Assessment` keeps the generated controller, service-manager shell, and
  registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/databasemigration/assessment/assessment_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/databasemigration/assessment/assessment_serviceclient.go`.
- The reviewed runtime requires explicit `spec.databaseCombination` and builds
  concrete `CreateMySqlAssessmentDetails` or
  `CreateOracleAssessmentDetails`, plus the matching concrete update body
  type, before OCI calls. The runtime then rereads the concrete subtype
  truthfully through `GetAssessment` so status projection comes from the live
  polymorphic resource body rather than the scaffolded union alone.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes Database
  Migration `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `WAITING`,
  `CANCELING`, `SUCCEEDED`, `FAILED`, and `CANCELED`) into shared async
  classes, maps `CREATE_ASSESSMENT`, `UPDATE_ASSESSMENT`, and
  `DELETE_ASSESSMENT` into create/update/delete phases, and resumes
  reconciliation from that shared async tracker across requeues.
- Create and update completion are both reread-backed. `CreateAssessment`
  returns an `Assessment` body plus `opc-work-request-id`, while
  `UpdateAssessment` and `DeleteAssessment` return work-request headers only,
  so the published runtime follows all three paths with `GetWorkRequest` and a
  concrete `GetAssessment` reread before declaring success or confirmed
  deletion.
- Lifecycle handling is explicit: `CREATING` and `IN_PROGRESS` requeue as
  provisioning, `UPDATING` requeues as updating, `ACTIVE` and `SUCCEEDED`
  settle success, `DELETING` blocks finalizer release until `DELETED`, and any
  remaining unmodeled terminal lifecycle such as `NEEDS_ATTENTION` surfaces as
  failure without requeue instead of being treated as ready.
- Required status projection remains part of the repo-authored contract. The
  runtime projects `status.id`, `status.displayName`,
  `status.compartmentId`, `status.networkSpeedMegabitPerSecond`,
  `status.acceptableDowntime`, `status.databaseDataSize`,
  `status.ddlExpectation`, `status.creationType`,
  `status.sourceDatabaseConnection`, `status.targetDatabaseConnection`,
  `status.lifecycleState`, `status.timeCreated`, `status.timeUpdated`,
  `status.description`, `status.freeformTags`, `status.definedTags`,
  `status.systemTags`, `status.migrationId`,
  `status.assessmentMigrationType`, `status.databaseCombination`, and
  `status.isCdbSupported` when OCI returns them.
- The broader assessor and object-helper families stay out of scope for this
  rollout. The published runtime rejects `spec.includeObjects`,
  `spec.excludeObjects`, and `spec.bulkIncludeExcludeData` instead of claiming
  support for the larger helper surface that the story explicitly excluded.

## Repo-authored semantics

- Pre-create reuse is exact-match and opt-in. The runtime only attempts
  bind-before-create when `compartmentId`, `displayName`, and
  `databaseCombination` are present. It lists by compartment and display name,
  narrows summaries by `databaseCombination`, then rereads each candidate via
  `GetAssessment` and binds only when source/target connection identity plus
  network speed, downtime, database size, DDL expectation, and creation type
  match the desired spec exactly.
- Mutation policy is explicit: only the `UpdateAssessmentDetails` surface
  reconciles in place. `compartmentId` and `databaseCombination` remain
  replacement-only drift, and the out-of-scope object-helper fields are
  rejected instead of being passed through to OCI as partial or misleading
  support.

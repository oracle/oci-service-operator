---
schemaVersion: 1
surface: repo-authored-semantics
service: apmtraces
slug: scheduledquery
gaps:
  - category: drift-guard
    status: open
    stopCondition: "Close when the generated ScheduledQuery spec preserves explicit zero-value update intent for scheduledQueryDescription, scheduledQueryMaximumRuntimeInSeconds, scheduledQueryRetentionPeriodInMs, freeformTags, and definedTags so the runtime can distinguish omission from clear-to-empty."
---

# Logic Gaps

## Current runtime contract

- `ScheduledQuery` keeps the generated controller, service-manager shell, and
  registration wiring, but the published runtime contract is reviewed in the
  manual seam `pkg/servicemanager/apmtraces/scheduledquery/scheduledquery_runtime_client.go`.
- The vendored SDK exposes `Create/Get/List/Update/DeleteScheduledQuery` plus
  lifecycle states `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`, `DELETED`,
  and `FAILED`. The reviewed runtime maps `CREATING` to provisioning,
  `UPDATING` to updating, `ACTIVE` to success, `FAILED` to terminal failure,
  and delete confirmation waits through `DELETING` until `DELETED` or NotFound.
- Pre-create lookup is explicit. `ListScheduledQueries` always scopes by
  `apmDomainId`, sends the SDK `displayName` filter from
  `spec.scheduledQueryName`, and adopts only a unique exact-name match in
  reusable lifecycle states (`ACTIVE`, `CREATING`, or `UPDATING`).
  Duplicate exact-name matches fail instead of guessing.
- Mutation policy stays aligned with `UpdateScheduledQueryDetails` for
  meaningful desired values. The published runtime reconciles
  `scheduledQueryName`, `scheduledQueryProcessingType`,
  `scheduledQueryProcessingSubType`, `scheduledQueryText`,
  `scheduledQuerySchedule`, `scheduledQueryDescription`,
  `scheduledQueryMaximumRuntimeInSeconds`,
  `scheduledQueryRetentionPeriodInMs`,
  `scheduledQueryProcessingConfiguration`,
  `scheduledQueryRetentionCriteria`, `freeformTags`, and `definedTags`
  in place. `apmDomainId` remains replacement-only drift.
- `status.apmDomainId` mirrors the bound request domain because OCI response
  bodies and summaries do not echo it, but the published kind keeps required
  status projection for the scheduled query body, lifecycle state, and
  `systemTags`.
- Create and update return the `ScheduledQuery` body directly and delete
  returns headers only. The reviewed runtime stays lifecycle-based and uses
  read-after-write plus confirm-delete rereads rather than a work-request
  polling surface. The row keeps `secret_side_effects = none`.

## Open Gap

- Explicit clear-to-empty intent for zero-value top-level fields is not fully
  representable in the current generated `ScheduledQuery` spec shape. Empty
  description/runtime/retention values and empty tag maps collapse to omission
  before parity checks, so the runtime cannot yet distinguish "clear this
  field" from "leave the current OCI value alone" for those cases.

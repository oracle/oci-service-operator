---
schemaVersion: 1
surface: repo-authored-semantics
service: generativeai
slug: model
gaps: []
---

# Logic Gaps

## Current runtime path

- `Model` uses the generated `ModelServiceManager` and generated runtime client,
  with one service-local wrapper in `pkg/servicemanager/generativeai/model`
  that only narrows pre-create reuse.
- When `spec.displayName` is empty, the service-local wrapper bypasses
  list-based reuse so create does not guess across earlier fine-tune jobs; if a
  tracked OCI identifier is stale, the wrapper clears it before re-entering the
  generated create path.
- When `spec.displayName` is set and no OCI identifier is already tracked, the
  generated runtime may bind an existing custom model by exact
  `compartmentId`, `baseModelId`, `fineTuneDetails.dedicatedAiClusterId`, and
  `displayName` match before create.
- Delete confirmation resolves tracked OCI identifiers through `GetModel`; when
  no `status.status.ocid` is recorded, the runtime falls back to `ListModels`
  with the same formal identity criteria before issuing `DeleteModel`.

## Repo-authored semantics

- Ready state maps to OCI `ACTIVE`, while `CREATING` remains provisioning and
  `DELETING` keeps the controller in terminating finalizer-retention state.
  The vendored SDK does not expose a separate `UPDATING` lifecycle enum for
  `Model`, so update uses read-after-write follow-up and then returns to the
  active steady state.
- The checked-in runtime supports in-place updates only for `displayName`,
  `description`, `vendor`, `version`, `freeformTags`, and `definedTags`,
  matching the `UpdateModelDetails` SDK surface.
- Drift for `compartmentId`, `baseModelId`, and `fineTuneDetails` is
  replacement-only and is rejected against the current tracked resource instead
  of silently rebinding or mutating a different model.
- Status projection remains required. The generated runtime merges the live OCI
  `Model` response into the published status fields, stamps
  `status.status.ocid`, and keeps OSOK lifecycle conditions aligned to the
  observed OCI lifecycle.
- Delete is explicit and required. The controller retains the finalizer until
  `DeleteModel` succeeds and follow-up observation confirms `DELETED` or OCI no
  longer returns the model.

## Why this row is seeded

- The checked-in generated controller and service-manager path now has an
  explicit bind-or-create answer, mutable-vs-replacement update policy, and
  required delete confirmation for `generativeai/Model`.
- No open formal gaps remain for the current generatedruntime contract.

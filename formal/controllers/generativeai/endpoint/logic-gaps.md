---
schemaVersion: 1
surface: repo-authored-semantics
service: generativeai
slug: endpoint
gaps: []
---

# Logic Gaps

## Current runtime path

- `Endpoint` uses the generated `EndpointServiceManager` and generated runtime
  client directly; there is no checked-in service-local wrapper for this
  resource.
- The generated runtime may bind an existing endpoint before create by listing
  on the required compartment scope, then matching the returned
  `EndpointSummary` items against the tracked `modelId` and
  `dedicatedAiClusterId`, plus any populated `displayName`.
- Delete confirmation resolves tracked OCI identities through `GetEndpoint`;
  when no `status.status.ocid` is recorded, the runtime falls back to
  `ListEndpoints` with the same formal identity criteria before issuing
  `DeleteEndpoint`.

## Repo-authored semantics

- `Endpoint` uses read-after-write follow-up for create and update, requeues
  while OCI reports `CREATING`, `UPDATING`, or `DELETING`, and treats `ACTIVE`
  as the steady-state success target.
- The checked-in runtime supports in-place updates only for
  `contentModerationConfig`, `displayName`, `description`, `freeformTags`, and
  `definedTags`.
- Drift for `compartmentId`, `modelId`, and `dedicatedAiClusterId` is
  replacement-only and is rejected against the current tracked resource instead
  of silently binding or mutating a different endpoint.
- Status projection remains required. The generated runtime merges the live OCI
  `Endpoint` response into the published status fields, stamps
  `status.status.ocid`, and keeps OSOK lifecycle conditions aligned to the
  observed OCI lifecycle.
- Delete is explicit and required. The controller retains the finalizer until
  `DeleteEndpoint` succeeds and follow-up observation confirms `DELETED` or OCI
  no longer returns the endpoint.

## Why this row is seeded

- The checked-in generated controller and service-manager path now has an
  explicit bind-or-create answer, mutable-vs-replacement update policy, and
  required delete confirmation for `generativeai/Endpoint`.
- No open formal gaps remain for the current generatedruntime contract.

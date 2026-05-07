---
schemaVersion: 1
surface: repo-authored-semantics
service: vnmonitoring
slug: pathanalyzertest
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `vnmonitoring/PathAnalyzerTest` row
after the scaffold baseline is replaced by the published synchronous
generated-runtime contract.

## Current runtime contract

- `PathAnalyzerTest` keeps the generated controller, service-manager shell, and
  registration wiring, but the published runtime contract is reviewed in the
  manual seam
  `pkg/servicemanager/vnmonitoring/pathanalyzertest/pathanalyzertest_runtime_client.go`.
- The vendored SDK exposes `Create/Get/List/Update/DeletePathAnalyzerTest`,
  returns the `PathAnalyzerTest` body directly from create/get/update, and
  returns only headers from delete. The resource exposes only `ACTIVE` and
  `DELETED`, so the reviewed runtime treats create and update as synchronous
  read-after-write flows instead of relying on provisional lifecycle states.
- Pre-create lookup is explicit. The runtime requires `spec.compartmentId`,
  skips reuse when `spec.displayName` is empty, scopes `ListPathAnalyzerTests`
  by compartment and exact display name when available, and adopts only a
  unique exact match on the reviewed identity surface
  `compartmentId + displayName + protocol + sourceEndpoint +
  destinationEndpoint + queryOptions + protocolParameters`.
- Mutation policy stays aligned with `UpdatePathAnalyzerTestDetails`. The
  published runtime reconciles `displayName`, `protocol`, `sourceEndpoint`,
  `destinationEndpoint`, `protocolParameters`, `queryOptions`,
  `freeformTags`, and `definedTags` in place. `compartmentId` remains
  replacement-only because the separate change-compartment action stays
  unpublished.
- Required status projection remains part of the repo-authored contract. The
  published status read model keeps the OCI body surface, including identifiers,
  timestamps, lifecycle state, query options, endpoint details,
  protocol-parameter details, and tag maps. The row has no secret side effects.
- Delete is explicit and required. The controller retains the finalizer until
  `DeletePathAnalyzerTest` succeeds and a follow-up `GetPathAnalyzerTest` or
  fallback `ListPathAnalyzerTests` reread confirms OCI either reports
  `DELETED` or no longer returns the resource.

---
schemaVersion: 1
surface: repo-authored-semantics
service: apmsynthetics
slug: script
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `apmsynthetics/Script` row after the
runtime review replaced the scaffold baseline with the published synchronous
generated-runtime contract.

## Current runtime contract

- `Script` keeps the generated controller, service-manager shell, and
  registration wiring, but the published runtime contract is reviewed in the
  manual seam `pkg/servicemanager/apmsynthetics/script/script_runtime_client.go`.
- The vendored SDK exposes `Create/Get/List/Update/DeleteScript`, scopes every
  request by query `apmDomainId`, returns the `Script` body directly from
  create/get/update, and returns only headers from delete. The service does not
  expose lifecycle-state fields for scripts, so create and update settle
  through read-after-write confirmation and the runtime converts that confirmed
  reread into settled `Active` status instead of requeueing on a provisional
  lifecycle fallback.
- Pre-create lookup is explicit. The runtime requires `spec.apmDomainId` and
  only attempts reuse when `spec.displayName` and `spec.contentType` are both
  non-empty. It scopes `ListScripts` by `apmDomainId`, sends exact
  `displayName` and `contentType` filters, and adopts only a unique exact match
  on that reviewed identity surface. Duplicate matches fail instead of
  guessing.
- Mutation policy stays aligned with `UpdateScriptDetails`. The published
  runtime reconciles `displayName`, `contentType`, `content`, `contentFileName`,
  `parameters`, `freeformTags`, and `definedTags` in place. `apmDomainId`
  remains replacement-only drift.
- Required status projection remains part of the repo-authored contract.
  `status.apmDomainId` mirrors the bound request domain because OCI response
  bodies and summaries do not echo it. The published status read model keeps
  script metadata, timestamps, tag maps, `contentFileName`,
  `contentSizeInBytes`, `contentType`, and monitor-count surfaces, but it
  intentionally excludes `content` and `parameters` to avoid projecting
  secret-capable script bodies or parameter values into status.
- Delete is explicit and required. The controller retains the finalizer until
  `DeleteScript` succeeds and a follow-up `GetScript` or fallback `ListScripts`
  reread confirms OCI no longer returns the script.

---
schemaVersion: 1
surface: repo-authored-semantics
service: apmconfig
slug: config
gaps:
  - category: drift-guard
    status: open
    stopCondition: "Close when the generated Config spec can preserve explicit false APDEX rule booleans inside spec.rules[] so the runtime can distinguish omission from `isEnabled=false` and `isApplyToErrorSpans=false`."
---

# Logic Gaps

## Current runtime contract

- `Config` keeps the generated controller, service-manager shell, and
  registration wiring, but the published polymorphic runtime contract is owned
  by `pkg/servicemanager/apmconfig/config/config_runtime_client.go`.
- `spec.configType` is explicit and required. The runtime decodes
  `CreateConfigDetails` and `UpdateConfigDetails` into the matching OCI subtype
  across `AGENT`, `OPTIONS`, `MACS_APM_EXTENSION`, `METRIC_GROUP`, `APDEX`,
  and `SPAN_FILTER`, and rejects top-level fields that do not belong to the
  selected subtype before OCI calls.
- The runtime is synchronous. Create and update use direct service calls plus a
  read-after-write follow-up, publish no provisioning or updating lifecycle
  buckets, and settle success from the follow-up read rather than a service
  work-request or lifecycle waiter.
- Delete confirmation is required. The finalizer stays until `DeleteConfig`
  succeeds and follow-up `GetConfig` or fallback `ListConfigs` rereads stop
  finding the resource.
- Pre-create reuse is bounded and subtype-aware. The runtime only attempts
  existing-before-create lookup when the selected subtype exposes enough
  identity to query safely, sends `apmDomainId` plus `configType` and any
  available `displayName` or `optionsGroup` list filters, then adopts only a
  unique exact match on the reviewed identity surface `configType`,
  `displayName`, `group`, `filterId`, `filterText`, `namespace`,
  `managementAgentId`, `matchAgentsWithAttributeValue`, and `serviceName`.
  Duplicate matches fail instead of guessing.
- Mutation policy is explicit and type-agnostic at the published top level.
  `agentVersion`, `attachInstallDir`, `config`, `definedTags`, `description`,
  `dimensions`, `displayName`, `filterId`, `filterText`, `freeformTags`,
  `group`, `metrics`, `namespace`, `options`, `overrides`, `processFilter`,
  `rules`, `runAsUser`, and `serviceName` reconcile in place.
  `apmDomainId`, `configType`, `managementAgentId`, and
  `matchAgentsWithAttributeValue` stay replacement-only drift.
- Required status projection remains part of the repo-authored contract. OCI
  bodies provide the subtype-specific read model, while `status.apmDomainId`
  mirrors the bound request domain because the service does not echo it in
  response payloads.

## Open Gap

- APDEX rule booleans remain an explicit `drift-guard` gap. The current
  generated `Config` spec helper types do not preserve explicit false values
  inside `spec.rules[]`, so `rules[].isEnabled=false` and
  `rules[].isApplyToErrorSpans=false` collapse to omission before the runtime
  decodes the OCI request body.

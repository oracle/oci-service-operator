---
schemaVersion: 1
surface: repo-authored-semantics
service: accessgovernancecp
slug: governanceinstance
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded
`accessgovernancecp/GovernanceInstance` row after the runtime review replaced
the scaffold placeholder with the published lifecycle-backed contract.

## Current runtime path

- `GovernanceInstance` keeps the generated controller, service-manager shell,
  and registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/accessgovernancecp/governanceinstance/governanceinstance_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/accessgovernancecp/governanceinstance/governanceinstance_serviceclient.go`.
- The vendored SDK exposes direct
  `Create/Get/List/Update/DeleteGovernanceInstance` operations plus the
  auxiliary mutator `ChangeGovernanceInstanceCompartment`. The lifecycle enum is
  `CREATING`, `ACTIVE`, `DELETING`, `DELETED`, and `NEEDS_ATTENTION`. The
  reviewed runtime treats `CREATING` as provisioning, `ACTIVE` as success,
  `DELETING` as terminating, `DELETED` as a delete-confirmation target, and
  `NEEDS_ATTENTION` as a terminal failure without requeue. There is no
  published in-place update lifecycle bucket because the SDK does not expose an
  `UPDATING` state for this kind.
- Pre-create reuse is bounded and tenant-safe. The runtime skips
  existing-before-create lookup unless `spec.compartmentId`,
  `spec.displayName`, `spec.licenseType`, and `spec.tenancyNamespace` are all
  non-empty. When lookup is enabled, it lists by exact `compartmentId` plus
  `displayName`, narrows those summaries to matching `licenseType` values in
  reusable lifecycles (`ACTIVE` or `CREATING`), then rereads each candidate via
  `GetGovernanceInstance` and reuses only a unique candidate whose
  `tenancyNamespace` also matches. Duplicate exact matches fail instead of
  guessing.
- Mutable drift is explicit: `displayName`, `description`, `licenseType`,
  `freeformTags`, and `definedTags` reconcile in place. The handwritten update
  builder preserves clear-to-empty intent for `description` and both tag maps.
  `compartmentId`, `tenancyNamespace`, `idcsAccessToken`, and `systemTags`
  remain replacement-only drift, and `ChangeGovernanceInstanceCompartment`
  stays out of scope for the published runtime.
- `IdcsAccessToken` remains a create-time input only. OCI does not project it
  back on `GovernanceInstance`, so post-create reconciles normalize it out of
  parity checks instead of treating it as perpetual unsupported drift.
- Create, update, and delete responses all expose `opc-work-request-id`, but
  the service package does not expose service-local work-request readers. The
  reviewed runtime records request and work-request breadcrumbs when OCI
  returns them, but lifecycle projection plus confirm-delete rereads remain the
  authoritative async signal.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, shared async breadcrumbs, and the
  published `status.displayName`, `status.compartmentId`, `status.timeCreated`,
  `status.lifecycleState`, `status.id`, `status.timeUpdated`,
  `status.description`, `status.licenseType`, `status.tenancyNamespace`,
  `status.instanceUrl`, `status.definedTags`, `status.freeformTags`, and
  `status.systemTags` fields when OCI returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  `DeleteGovernanceInstance` succeeds and `GetGovernanceInstance` confirms the
  resource is gone or reports lifecycle state `DELETED`.

---
schemaVersion: 1
surface: repo-authored-semantics
service: apmcontrolplane
slug: apmdomain
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `apmcontrolplane/ApmDomain` row after
the runtime review replaced the scaffold semantics with the published
work-request-backed contract.

## Current runtime path

- `ApmDomain` keeps the generated controller, service-manager shell, and
  registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/apmcontrolplane/apmdomain/apmdomain_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/apmcontrolplane/apmdomain/apmdomain_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes APM Control
  Plane `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `CANCELING`,
  `SUCCEEDED`, `FAILED`, and `CANCELED`) into shared async classes, maps
  `CREATE_APM_DOMAIN`, `UPDATE_APM_DOMAIN`, and `DELETE_APM_DOMAIN` into
  create/update/delete phases, and resumes reconciliation from that shared
  async tracker across requeues.
- Create-time identity recovery is work-request-backed. The runtime records the
  tracked ApmDomain OCID when OCI exposes it and otherwise resolves the created
  resource OCID from work-request resources before reading the ApmDomain by ID
  and projecting status.
- Bind resolution is bounded. When no OCI identifier is tracked, the runtime
  only attempts pre-create reuse when `spec.displayName` is non-empty and then
  adopts only a unique `ListApmDomains` match on exact `compartmentId` plus
  `displayName`. Summaries in `FAILED`, `DELETING`, or `DELETED` are not
  reused, and duplicate exact-name matches fail instead of binding
  arbitrarily.
- Mutable drift is limited to `displayName`, `description`, `freeformTags`,
  and `definedTags`. The handwritten update-body builder preserves clear-to-
  empty intent for description and both tag maps, while `compartmentId` and
  `isFreeTier` remain replacement-only drift. The provider-only
  `ChangeApmDomainCompartment` auxiliary operation stays out-of-scope for the
  published runtime.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and the
  published `status.id`, `status.displayName`, `status.compartmentId`,
  `status.description`, `status.lifecycleState`, `status.isFreeTier`,
  `status.timeCreated`, `status.timeUpdated`, `status.freeformTags`,
  `status.definedTags`, and `status.dataUploadEndpoint` read-model fields when
  APM Control Plane returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetApmDomain` or fallback
  `ListApmDomains` confirms the ApmDomain is gone or reports lifecycle state
  `DELETED`.

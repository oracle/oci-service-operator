---
schemaVersion: 1
surface: repo-authored-semantics
service: distributeddatabase
slug: distributeddatabaseprivateendpoint
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded
`distributeddatabase/DistributedDatabasePrivateEndpoint` row after the first
controller-backed publication.

## Current runtime contract

- `DistributedDatabasePrivateEndpoint` keeps the generated controller,
  service-manager shell, and registration wiring, but the published runtime
  uses generatedruntime plus a small package-local overlay in
  `pkg/servicemanager/distributeddatabase/distributeddatabaseprivateendpoint/`
  for update-body fidelity and tracked-identity cleanup.
- CRUD is lifecycle-driven even though create and delete responses can carry
  `opc-work-request-id`. The runtime records those IDs as shared request
  breadcrumbs when they are present, rereads with
  `GetDistributedDatabasePrivateEndpoint`, treats `CREATING` as provisioning,
  `UPDATING` as updating, `ACTIVE` and `INACTIVE` as success, `FAILED` as
  terminal, and confirms delete through `DELETING` until `DELETED` or NotFound.
- Pre-create lookup is explicit. The runtime lists by exact `compartmentId`
  plus `displayName`, preserves the service-supported `lifecycleState` filter
  in the published list contract, and only reuses a unique exact summary
  match.
- Update semantics are explicit. The runtime sends only `displayName`,
  `description`, `nsgIds`, `freeformTags`, and `definedTags` through
  `UpdateDistributedDatabasePrivateEndpointDetails`; `compartmentId` and
  `subnetId` remain replacement-only drift because the published CRUD surface
  does not expose compartment or subnet change helpers.
- `ListDistributedDatabasePrivateEndpoints` returns
  `DistributedDatabasePrivateEndpointSummary`, which omits fields such as
  `privateIp`. The runtime therefore forces a follow-up
  `GetDistributedDatabasePrivateEndpoint` after bind, create, and update so
  status projection stays truthful.
- Required status projection keeps the relevant OCI body surface, including
  `id`, `compartmentId`, `subnetId`, `vcnId`, `displayName`,
  `lifecycleState`, `timeCreated`, `timeUpdated`, `description`, `privateIp`,
  `nsgIds`, `lifecycleDetails`, `proxyComputeInstanceId`, `freeformTags`,
  `definedTags`, and `systemTags`. The row has no Kubernetes Secret side
  effects.

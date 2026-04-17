---
schemaVersion: 1
surface: repo-authored-semantics
service: databasetools
slug: databasetoolsconnection
gaps:
  - category: drift-guard
    status: open
    stopCondition: "Close when the DatabaseToolsConnection runtime explicitly rejects post-create locks drift as replacement-only, or threads locks through update-body comparison and OCI mutation."
---

# Logic Gaps

The scaffold placeholder is replaced by the reviewed
`databasetools/DatabaseToolsConnection` generated-runtime contract, but one
drift guard remains explicit while the handwritten runtime stays scoped to the
current supported surface.

## Current runtime path

- `DatabaseToolsConnection` keeps the generated controller, service-manager
  shell, and registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/databasetools/databasetoolsconnection/databasetoolsconnection_runtime_client.go`.
- The handwritten runtime config binds `CreateDatabaseToolsConnection`,
  `GetDatabaseToolsConnection`, `ListDatabaseToolsConnections`,
  `UpdateDatabaseToolsConnection`, and `DeleteDatabaseToolsConnection` through
  `generatedruntime.ServiceClient` while resolving secret-backed spec values
  before OCI calls.
- Lifecycle handling is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` and `INACTIVE` settle success,
  `FAILED` is terminal without requeue, and delete confirmation waits through
  `DELETING` until `DELETED` or NotFound.
- Required status projection remains part of the repo-authored contract. The
  generated runtime stamps OSOK lifecycle conditions plus the published
  `status.id`, `status.displayName`, `status.compartmentId`,
  `status.lifecycleState`, `status.timeCreated`, `status.timeUpdated`,
  `status.lifecycleDetails`, `status.runtimeSupport`, `status.type`,
  `status.connectionString`, `status.relatedResource`, `status.userName`,
  `status.userPassword`, `status.advancedProperties`, `status.keyStores`,
  `status.privateEndpointId`, `status.proxyClient`, `status.url`,
  `status.freeformTags`, `status.definedTags`, `status.systemTags`, and
  `status.locks` read-model fields when OCI returns them.

## Repo-authored semantics

- Polymorphic request bodies are explicit. `spec.type` must resolve to one of
  `GENERIC_JDBC`, `MYSQL`, `ORACLE_DATABASE`, or `POSTGRESQL`; the runtime
  decodes the resolved Kubernetes spec into the matching OCI create and update
  body shape and rejects fields unsupported by that connection type before OCI
  calls.
- Pre-create lookup is explicit. When no OCI identifier is already tracked, the
  runtime queries `ListDatabaseToolsConnections` with `compartmentId`,
  `displayName`, and optional `relatedResourceIdentifier`, then narrows
  candidates in memory with the type-aware identity surface `compartmentId`,
  `displayName`, `type`, `connectionString`, `url`,
  `relatedResource.identifier`, `userName`, `privateEndpointId`, and
  `runtimeSupport`.
- Mutation policy is explicit: only `advancedProperties`, `connectionString`,
  `definedTags`, `displayName`, `freeformTags`, `keyStores`,
  `privateEndpointId`, `proxyClient`, `relatedResource`, `url`, `userName`,
  and `userPassword` reconcile in place. `compartmentId`, `runtimeSupport`,
  and `type` stay replacement-only drift even though the provider exposes
  broader update paths.
- Create, update, and delete use plain provider helper semantics
  (`tfresource.CreateResource`, `tfresource.UpdateResource`,
  `tfresource.DeleteResource`) with read-after-write follow-up for create and
  update plus confirm-delete follow-up for delete. The runtime does not adopt
  provider work-request waiters or Kubernetes secret side effects.

## Open Gap

- `locks` remain an explicit `drift-guard` gap. The create body accepts lock
  inputs, but the handwritten update path does not classify post-create `locks`
  drift as replacement-only and cannot send `locks` through OCI update shapes
  because those shapes omit the field.

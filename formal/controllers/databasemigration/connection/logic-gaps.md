---
schemaVersion: 1
surface: repo-authored-semantics
service: databasemigration
slug: connection
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `databasemigration/Connection` row
after the runtime review replaced the scaffold placeholder with the published
work-request-backed generated-runtime contract.

## Current runtime path

- `Connection` keeps the generated controller, service-manager shell, and
  registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/databasemigration/connection/connection_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/databasemigration/connection/connection_serviceclient.go`.
- The reviewed runtime builds `CreateConnectionDetails` and
  `UpdateConnectionDetails` explicitly so OCI receives concrete `MYSQL` or
  `ORACLE` request bodies with the required `connectionType` discriminator.
  The runtime rejects cross-type field mixes before OCI calls instead of
  silently dropping unsupported fields from the union-shaped spec.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes Database
  Migration `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `WAITING`,
  `CANCELING`, `SUCCEEDED`, `FAILED`, and `CANCELED`) into shared async
  classes, maps `CREATE_CONNECTION`, `UPDATE_CONNECTION`, and
  `DELETE_CONNECTION` into create/update/delete phases, and resumes
  reconciliation from that shared async tracker across requeues.
- Create-time identity recovery is work-request-backed. When the create
  response does not already carry a tracked Connection OCID, the runtime
  resolves that OCID from work-request resources before reading the Connection
  by ID and projecting status.
- Lifecycle handling is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` and `INACTIVE` settle success,
  `FAILED` is terminal without requeue, and delete confirmation waits through
  `DELETING` until `DELETED` or NotFound.
- Required status projection remains part of the repo-authored contract. The
  runtime projects `status.id`, `status.displayName`, `status.compartmentId`,
  `status.lifecycleState`, `status.timeCreated`, `status.timeUpdated`,
  `status.lifecycleDetails`, `status.description`, `status.freeformTags`,
  `status.definedTags`, `status.systemTags`, `status.vaultId`,
  `status.keyId`, `status.subnetId`, `status.ingressIps`, `status.nsgIds`,
  `status.replicationUsername`, `status.secretId`,
  `status.privateEndpointId`, `status.username`, `status.connectionType`,
  `status.host`, `status.port`, `status.databaseName`,
  `status.additionalAttributes`, `status.dbSystemId`,
  `status.technologyType`, `status.securityProtocol`, `status.sslMode`,
  `status.connectionString`, `status.databaseId`, `status.sshHost`,
  `status.sshUser`, and `status.sshSudoLocation` when OCI returns them. The
  published status intentionally excludes password, replicationPassword, and
  sshKey material even though the OCI get surface can return them.

## Repo-authored semantics

- Pre-create lookup is explicit. The runtime only attempts list reuse when
  `spec.displayName`, `spec.connectionType`, and `spec.technologyType` are all
  non-empty. It queries `ListConnections` with exact `compartmentId` plus
  `displayName`, then narrows candidates in memory with `connectionType`,
  `technologyType`, and observable type-specific identity fields including
  `databaseId`, `connectionString`, `databaseName`, `dbSystemId`, `host`, and
  `port`.
- Mutation policy is explicit: only the update-body surface from
  `UpdateMysqlConnectionDetails` or `UpdateOracleConnectionDetails` reconciles
  in place, including tag changes, credential changes, vault or key changes,
  network fields, and the type-specific connection details. `compartmentId`,
  `connectionType`, and `technologyType` remain replacement-only drift even
  though the provider surface also exposes `ChangeConnectionCompartment`.
- Auxiliary Database Migration mutators stay out of scope for the published
  runtime. `ChangeConnectionCompartment` and `ConnectionDiagnostics` are
  visible in provider or SDK facts but are not invoked by the reviewed
  controller-backed surface.

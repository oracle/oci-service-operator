# Distributed Database Onboarding Audit

This audit is the `US-149` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/distributeddatabase` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/distributeddatabase`.
- `vendor/modules.txt` and
  `vendor/github.com/oracle/oci-go-sdk/v65/distributeddatabase/` now carry the
  branch-local SDK surface for this service.

## SDK Audit

### `DistributedDatabasePrivateEndpoint`

- Full CRUD family is present:
  `CreateDistributedDatabasePrivateEndpoint`,
  `GetDistributedDatabasePrivateEndpoint`,
  `ListDistributedDatabasePrivateEndpoints`,
  `UpdateDistributedDatabasePrivateEndpoint`, and
  `DeleteDistributedDatabasePrivateEndpoint`.
- `CreateDistributedDatabasePrivateEndpointResponse` returns
  `DistributedDatabasePrivateEndpoint` in the body and exposes
  `OpcWorkRequestId` and `Etag`.
- `GetDistributedDatabasePrivateEndpointResponse` returns
  `DistributedDatabasePrivateEndpoint`.
- `ListDistributedDatabasePrivateEndpointsRequest` requires `compartmentId` and
  supports `lifecycleState`, `displayName`, paging, and sort controls.
- `ListDistributedDatabasePrivateEndpointsResponse` returns
  `DistributedDatabasePrivateEndpointCollection`, whose items are
  `DistributedDatabasePrivateEndpointSummary` rather than full resource bodies.
- `UpdateDistributedDatabasePrivateEndpointResponse` returns the updated
  `DistributedDatabasePrivateEndpoint` body plus `Etag`; it does not expose a
  work-request ID.
- `DeleteDistributedDatabasePrivateEndpointResponse` returns headers only and
  exposes `OpcWorkRequestId`.
- `DistributedDatabasePrivateEndpoint` carries `compartmentId`, `subnetId`,
  `vcnId`, `displayName`, `lifecycleState`, `description`, `privateIp`,
  `nsgIds`, and `lifecycleDetails`. The list summary keeps most identity and
  networking fields, but the full get body is still required for
  `privateIp`.
- Lifecycle states are `ACTIVE`, `FAILED`, `INACTIVE`, `DELETING`, `DELETED`,
  `UPDATING`, and `CREATING`.
- The package exposes `GetWorkRequest`, `ListWorkRequests`,
  `ListWorkRequestErrors`, and `ListWorkRequestLogs`.

### Auxiliary Families

- The package also exposes broader out-of-scope surfaces for
  `DistributedDatabase` parents, catalogs, shards, GDS control nodes,
  wallet and certificate flows, start or stop actions, password rotation,
  raft metrics, replication-unit moves, network validation helpers, and
  multiple compartment-change operations.

## Generator Implications For `US-151`

- `DistributedDatabasePrivateEndpoint` is the planned first published kind for
  `US-151`.
- Recommended `formalSpec` is `distributeddatabaseprivateendpoint`.
- The async contract is intentionally asymmetric: create and delete are
  workrequest-based, while update returns the updated body directly. `US-151`
  should treat this as a lifecycle resource with explicit workrequest handling
  on create and delete instead of assuming one uniform mutation contract.
- The main rollout risks are explicit here: list returns summaries rather than
  full bodies, update omits `OpcWorkRequestId`, and the full get path is needed
  for fields such as `privateIp`. The later story should keep those read-after-
  write and delete-confirmation seams truthful without broadening into the
  larger distributed-database families.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching checked-in provider-fact imports or local repo
  evidence for `DistributedDatabasePrivateEndpoint` in this checkout, so
  `US-151` should treat provider-backed imports as absent or unconfirmed until a
  pinned provider surface is proven directly.

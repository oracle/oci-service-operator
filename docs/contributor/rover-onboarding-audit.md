# Rover Onboarding Audit

This audit is the `US-118` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/rover` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `rover` package in the module cache; the
  repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/rover` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/rover` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `RoverCluster`

- Full CRUD family is present: `CreateRoverCluster`, `GetRoverCluster`,
  `ListRoverClusters`, `UpdateRoverCluster`, and `DeleteRoverCluster`.
- Additional mutator is present: `ChangeRoverClusterCompartment`.
- `CreateRoverClusterResponse`, `GetRoverClusterResponse`, and
  `UpdateRoverClusterResponse` return `RoverCluster`.
- `DeleteRoverClusterResponse` returns only headers; it does not return a
  `RoverCluster` body or an `opc-work-request-id`.
- `ListRoverClustersResponse` returns `RoverClusterCollection`.
- `ListRoverClustersRequest` requires `compartmentId` and exposes
  `displayName`, `clusterType`, `lifecycleState`, page, and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`.
- The package exposes generic `GetWorkRequest`, `ListWorkRequests`,
  `ListWorkRequestErrors`, and `ListWorkRequestLogs` helpers, but the
  `RoverCluster` CRUD responses do not expose `opc-work-request-id`.
- The `RoverCluster` model includes secret-like or operationally sensitive
  fields such as `superUserPassword`, `unlockPassphrase`,
  `exteriorDoorCode`, and `interiorAlarmDisarmCode`.

### Auxiliary Families

- Additional full CRUD families are `RoverNode` and `RoverEntitlement`.
- Bundle request/status operations, certificate retrieval, shape discovery,
  and additional-node workflows should stay unpublished initially.

## Generator Implications For `US-121`

- `RoverCluster` is the planned first published kind for `US-121`.
- Recommended `formalSpec` is `rovercluster`.
- Recommended async classification is `lifecycle`.
- `RoverCluster` looks viable as a controller-backed rollout because
  get/create/update project the resource body and list exposes the required
  compartment-based lookup filters.
- The main rollout risk is explicit here: although the package exposes generic
  workrequest helpers, the `RoverCluster` create/update/delete responses do not
  return workrequest IDs. `US-121` should therefore default to
  lifecycle/read-after-write handling unless it can prove a reliable
  workrequest correlation path. The returned secret-like fields also mean
  status and Secret publication policy must be explicit before rollout.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for `RoverCluster` in the
  accessible local provider/docs layout.
- `US-121` should treat provider-backed imports as absent or unconfirmed until
  a pinned provider surface is proven directly.

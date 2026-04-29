# Database Migration Onboarding Audit

This audit is the `US-84` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/databasemigration` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `databasemigration` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/databasemigration` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/databasemigration` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `Connection`

- Full CRUD family is present:
  `CreateConnection`, `GetConnection`, `ListConnections`, `UpdateConnection`,
  and `DeleteConnection`.
- Additional mutators are present: `ChangeConnectionCompartment` and
  `ConnectionDiagnostics`.
- `GetConnectionResponse` returns `Connection`.
- `ListConnectionsResponse` returns `ConnectionCollection`.
- `ListConnectionsRequest` exposes required `compartmentId`, plus
  `technologyType`, `technologySubType`, `connectionType`,
  `sourceConnectionId`, `displayName`, and `lifecycleState`, plus page and
  sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `INACTIVE`,
  `DELETING`, `DELETED`, and `FAILED`.
- `CreateConnectionResponse` returns `Connection` and exposes
  `OpcWorkRequestId`.
- `UpdateConnectionResponse` and `DeleteConnectionResponse` both expose
  `OpcWorkRequestId`.

### Auxiliary Families

- Additional SDK-discovered families include `Assessment`, `Migration`,
  `ParameterFileVersion`, `Job`, `WorkRequest`, `WorkRequestError`, and
  `WorkRequestLog`, plus a larger set of advisor, assessor, object, and
  transfer helper families.
- `Assessment` and `Migration` each carry their own CRUD surface.
- `ParameterFileVersion` exposes create/get/list/delete, and `Job` exposes
  get/list/update/delete.
- Those broader migration workflows should stay unpublished initially while the
  first `Connection` rollout lands.

## Generator Implications For `US-91`

- `Connection` is the narrowest foundational kind in the package and already
  matches the approved follow-on story.
- Recommended `formalSpec` is `connection`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `Connection` looks viable as a direct controller-backed generated rollout
  without handwritten runtime work because the service ships the full
  work-request helper surface and base CRUD already exposes the work-request
  identifiers needed by the shared generated seam.
- `Assessment`, `Migration`, diagnostics, compartment changes, and the broader
  advisor or object helper families should stay unpublished initially.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_database_migration_connection` as both
  the resource and singular data source, plus
  `oci_database_migration_connections` as the list data source.
- The provider resource uses `GetWorkRequest` and `ListWorkRequestErrors` for
  create, update, delete, and compartment-change flows, which matches the
  recommended `async.strategy=workrequest` baseline.

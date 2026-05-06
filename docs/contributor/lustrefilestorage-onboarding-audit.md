# Lustre File Storage Onboarding Audit

This audit is the `US-105` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/lustrefilestorage` before
`services.yaml` publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `lustrefilestorage` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/lustrefilestorage` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/lustrefilestorage` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `LustreFileSystem`

- Full CRUD family is present:
  `CreateLustreFileSystem`, `GetLustreFileSystem`,
  `ListLustreFileSystems`, `UpdateLustreFileSystem`, and
  `DeleteLustreFileSystem`.
- Additional mutator is present: `ChangeLustreFileSystemCompartment`.
- `GetLustreFileSystemResponse` returns `LustreFileSystem`.
- `ListLustreFileSystemsResponse` returns `LustreFileSystemCollection`.
- `ListLustreFileSystemsRequest` exposes `compartmentId`,
  `availabilityDomain`, `lifecycleState`, `displayName`, and `id`, plus page
  and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `INACTIVE`,
  `DELETING`, `DELETED`, and `FAILED`.
- `CreateLustreFileSystemResponse` returns the resource body and
  `OpcWorkRequestId`.
- `UpdateLustreFileSystemResponse` and `DeleteLustreFileSystemResponse`
  expose `OpcWorkRequestId`; update and delete do not return a
  `LustreFileSystem` body.
- The package also exposes service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, and `ListWorkRequestLogs`
  helpers.

### Auxiliary Families

- `ObjectStorageLink` is a second full CRUD family with its own
  `ChangeObjectStorageLinkCompartment` mutator and should stay unpublished
  initially.
- `SyncJob` is get/list only and should stay unpublished initially.
- Maintenance-window helper families are list-only support surfaces and should
  stay unpublished initially.

## Generator Implications For `US-106`

- `LustreFileSystem` is the requested initial kind and the cleanest first
  published resource in the package.
- Recommended `formalSpec` is `lustrefilesystem`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `LustreFileSystem` looks viable as a direct controller-backed generated
  rollout because GET/list expose lifecycle state and the service ships the
  work-request helpers needed to follow mutating operations.
- `US-106` should keep `ObjectStorageLink`, `SyncJob`, and maintenance helper
  surfaces unpublished initially, and it should decide explicitly whether the
  SDK's `INACTIVE` lifecycle state is treated as a reusable steady state or as
  a terminal non-ready condition in repo-authored runtime semantics.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are
  `oci_lustre_file_storage_lustre_file_system` as the resource,
  `oci_lustre_file_storage_lustre_file_system` as the singular data source,
  and `oci_lustre_file_storage_lustre_file_systems` as the list data source.
- Provider docs publish list filters for `availability_domain`,
  `compartment_id`, `display_name`, `id`, and `state`, which aligns with the
  SDK-side list family and supports a direct `LustreFileSystem` rollout.

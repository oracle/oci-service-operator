# API Platform Onboarding Audit

This audit is the `US-84` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/apiplatform` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `apiplatform` package in the module cache;
  the repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/apiplatform` only
  because nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/apiplatform` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `ApiPlatformInstance`

- Full CRUD family is present:
  `CreateApiPlatformInstance`, `GetApiPlatformInstance`,
  `ListApiPlatformInstances`, `UpdateApiPlatformInstance`, and
  `DeleteApiPlatformInstance`.
- Additional mutator is present: `ChangeApiPlatformInstanceCompartment`.
- `GetApiPlatformInstanceResponse` returns `ApiPlatformInstance`.
- `ListApiPlatformInstancesResponse` returns `ApiPlatformInstanceCollection`.
- `ListApiPlatformInstancesRequest` exposes required `compartmentId`, plus
  `name`, `id`, and `lifecycleState`, plus page and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`.
- `CreateApiPlatformInstanceResponse` returns `ApiPlatformInstance` and
  exposes `OpcWorkRequestId`.
- `UpdateApiPlatformInstanceResponse` returns `ApiPlatformInstance` but does
  not expose `OpcWorkRequestId`.
- `DeleteApiPlatformInstanceResponse` exposes `OpcWorkRequestId`.

### Auxiliary Families

- Additional SDK-discovered families are `WorkRequest`, `WorkRequestError`, and
  `WorkRequestLog`.
- No other top-level CRUD family competes with `ApiPlatformInstance` for the
  first rollout.

## Generator Implications For `US-89`

- `ApiPlatformInstance` is the only publishable top-level kind and already
  matches the approved follow-on story.
- Recommended `formalSpec` is `apiplatforminstance`.
- Recommended async classification is `lifecycle`.
- `ApiPlatformInstance` looks viable as a direct controller-backed generated
  rollout without handwritten runtime work because the GET surface projects
  lifecycle state directly, create and update both return the resource body,
  and only create and delete carry work-request headers.
- `ChangeApiPlatformInstanceCompartment` and the work-request auxiliaries
  should stay unpublished initially while the first `ApiPlatformInstance`
  rollout lands.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_api_platform_api_platform_instance` as
  both the resource and singular data source, plus
  `oci_api_platform_api_platform_instances` as the list data source.
- The provider resource waits on `GetWorkRequest` for create and delete, but
  its update and compartment-change paths continue through `WaitForUpdatedState`
  lifecycle rereads. That mixed behavior matches the recommended
  `async.strategy=lifecycle` baseline for the first generated rollout.

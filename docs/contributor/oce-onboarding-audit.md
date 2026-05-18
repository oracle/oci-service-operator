# OCE Onboarding Audit

This audit is the `US-105` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/oce` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `oce` package in the module cache; the repo
  lacked `vendor/github.com/oracle/oci-go-sdk/v65/oce` only because nothing
  imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/oce` so `go mod vendor` keeps the package
  in the branch-local inputs.

## SDK Audit

### `OceInstance`

- Full CRUD family is present: `CreateOceInstance`, `GetOceInstance`,
  `ListOceInstances`, `UpdateOceInstance`, and `DeleteOceInstance`.
- Additional mutator is present: `ChangeOceInstanceCompartment`.
- `GetOceInstanceResponse` returns `OceInstance`.
- `ListOceInstancesResponse` returns `OceInstanceCollection`.
- `ListOceInstancesRequest` exposes `compartmentId`, `tenancyId`,
  `displayName`, and `lifecycleState`, plus page and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`.
- `CreateOceInstanceResponse`, `UpdateOceInstanceResponse`, and
  `DeleteOceInstanceResponse` all expose `OpcWorkRequestId`.
- The mutating responses do not project an `OceInstance` body, so the selected
  kind depends on service-local work-request follow-up before it can reread
  the live resource through GET.
- The package also exposes service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, and `ListWorkRequestLogs`
  helpers.

### Auxiliary Families

- No additional top-level CRUD families are apparent beyond the service-local
  work-request auxiliaries.

## Generator Implications For `US-107`

- `OceInstance` is the requested initial kind and the only top-level full CRUD
  family in the package.
- Recommended `formalSpec` is `oceinstance`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `OceInstance` looks viable as a direct controller-backed generated rollout
  because GET/list expose lifecycle state and the service ships the work
  request helpers needed to follow mutations.
- The main rollout risk is identity recovery: the mutation responses only
  return work-request headers, so `US-107` must recover the affected OCID
  through `GetWorkRequest` and then reread `GetOceInstance` instead of assuming
  create or update returns a full resource body.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_oce_oce_instance` as the resource,
  `oci_oce_oce_instance` as the singular data source, and
  `oci_oce_oce_instances` as the list data source.
- Provider docs publish the same single-instance family as both a resource and
  singular/list data sources, which matches the SDK's one-family rollout shape.

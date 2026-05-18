# OPA Onboarding Audit

This audit is the `US-105` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/opa` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `opa` package in the module cache; the repo
  lacked `vendor/github.com/oracle/oci-go-sdk/v65/opa` only because nothing
  imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/opa` so `go mod vendor` keeps the package
  in the branch-local inputs.

## SDK Audit

### `OpaInstance`

- Full CRUD family is present: `CreateOpaInstance`, `GetOpaInstance`,
  `ListOpaInstances`, `UpdateOpaInstance`, and `DeleteOpaInstance`.
- Additional mutator is present: `ChangeOpaInstanceCompartment`.
- `GetOpaInstanceResponse` returns `OpaInstance`.
- `ListOpaInstancesResponse` returns `OpaInstanceCollection`.
- `ListOpaInstancesRequest` exposes `compartmentId`, `lifecycleState`,
  `displayName`, and `id`, plus page and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `INACTIVE`,
  `DELETING`, `DELETED`, and `FAILED`.
- `CreateOpaInstanceResponse`, `UpdateOpaInstanceResponse`, and
  `DeleteOpaInstanceResponse` all expose `OpcWorkRequestId`.
- The mutating responses do not return an `OpaInstance` body, so the runtime
  must recover or confirm the live resource through work-request follow-up and
  GET.
- The package also exposes service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, and `ListWorkRequestLogs`
  helpers.

### Auxiliary Families

- No additional top-level CRUD families are apparent beyond the service-local
  work-request auxiliaries.

## Generator Implications For `US-108`

- `OpaInstance` is the requested initial kind and the only full CRUD family in
  the package.
- Recommended `formalSpec` is `opainstance`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `OpaInstance` looks viable as a direct controller-backed generated rollout
  because GET/list expose lifecycle state and the service includes the
  work-request helpers needed for mutation follow-up.
- `US-108` should keep one explicit lifecycle policy for the SDK's `INACTIVE`
  state instead of silently treating it as either success or failure, because
  the surface exposes that state but no separate start/stop API family.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_opa_opa_instance` as the resource,
  `oci_opa_opa_instance` as the singular data source, and
  `oci_opa_opa_instances` as the list data source.
- Provider docs publish the same single-instance family as both a resource and
  singular/list data sources, which matches the SDK's one-family rollout
  contract.

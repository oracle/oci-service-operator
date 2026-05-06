# VBS Inst Onboarding Audit

This audit is the `US-105` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/vbsinst` before `services.yaml` publishes
the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `vbsinst` package in the module cache; the
  repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/vbsinst` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/vbsinst` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `VbsInstance`

- Full CRUD family is present: `CreateVbsInstance`, `GetVbsInstance`,
  `ListVbsInstances`, `UpdateVbsInstance`, and `DeleteVbsInstance`.
- Additional mutator is present: `ChangeVbsInstanceCompartment`.
- `GetVbsInstanceResponse` returns `VbsInstance`.
- `ListVbsInstancesResponse` returns `VbsInstanceSummaryCollection`.
- `ListVbsInstancesRequest` exposes `compartmentId`, `id`, `name`, and
  `lifecycleState`, plus page and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`.
- `CreateVbsInstanceResponse`, `UpdateVbsInstanceResponse`, and
  `DeleteVbsInstanceResponse` all expose `OpcWorkRequestId`.
- The mutating responses do not return a `VbsInstance` body, so the runtime
  must recover or confirm the live resource through work-request follow-up and
  GET.
- The package also exposes service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, and `ListWorkRequestLogs`
  helpers.

### Auxiliary Families

- No additional top-level CRUD families are apparent beyond the service-local
  work-request auxiliaries.

## Generator Implications For `US-109`

- `VbsInstance` is the requested initial kind and the only full CRUD family in
  the package.
- Recommended `formalSpec` is `vbsinstance`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `VbsInstance` looks viable as a direct controller-backed generated rollout
  because GET/list expose lifecycle state and the service ships the work
  request helpers needed for mutation follow-up.
- The main rollout risk is identity shape: list filters key on immutable
  `name` plus optional `id`, while `displayName` is only user-facing status, so
  `US-109` should keep bind-versus-create rules anchored on `name` instead of
  assuming `displayName` is unique.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_vbs_inst_vbs_instance` as the resource,
  `oci_vbs_inst_vbs_instance` as the singular data source, and
  `oci_vbs_inst_vbs_instances` as the list data source.
- Provider docs publish list filters for `compartment_id`, `id`, `name`, and
  `state`, which matches the SDK-side identity shape and reinforces the
  `name`-versus-`displayName` rollout risk.

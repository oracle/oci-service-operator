# Visual Builder Onboarding Audit

This audit is the `US-105` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/visualbuilder` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `visualbuilder` package in the module cache;
  the repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/visualbuilder`
  only because nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/visualbuilder` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `VbInstance`

- Full CRUD family is present: `CreateVbInstance`, `GetVbInstance`,
  `ListVbInstances`, `UpdateVbInstance`, and `DeleteVbInstance`.
- Additional mutator is present: `ChangeVbInstanceCompartment`.
- `GetVbInstanceResponse` returns `VbInstance`.
- `ListVbInstancesResponse` returns `VbInstanceSummaryCollection`.
- `ListVbInstancesRequest` exposes `compartmentId`, `displayName`, and
  `lifecycleState`, plus page and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `INACTIVE`,
  `DELETING`, `DELETED`, and `FAILED`.
- `CreateVbInstanceResponse`, `UpdateVbInstanceResponse`, and
  `DeleteVbInstanceResponse` all expose `OpcWorkRequestId`.
- The mutating responses do not return a `VbInstance` body, so the runtime
  must recover or confirm the live resource through work-request follow-up and
  GET.
- The package also exposes service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, and `ListWorkRequestLogs`
  helpers.

### Auxiliary Families

- No additional top-level CRUD families are apparent beyond the service-local
  work-request auxiliaries.

## Generator Implications For `US-110`

- `VbInstance` is the requested initial kind and the only full CRUD family in
  the package.
- Recommended `formalSpec` is `vbinstance`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `VbInstance` looks viable as a direct controller-backed generated rollout
  because GET/list expose lifecycle state and the service ships the work
  request helpers needed for mutation follow-up.
- The primary rollout risk is shape complexity: the resource carries nested
  custom endpoint and `networkEndpointDetails` state, so `US-110` must handle
  observed-state projection and clear-to-empty update semantics explicitly
  instead of assuming a flat status surface.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_visual_builder_vb_instance` as the
  resource, `oci_visual_builder_vb_instance` as the singular data source, and
  `oci_visual_builder_vb_instances` as the list data source.
- Provider docs publish the same nested endpoint and custom-hostname shape on
  both the resource and data sources, which reinforces the SDK-side
  status-projection risk rather than removing it.

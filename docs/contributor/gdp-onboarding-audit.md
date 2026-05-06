# Guarded Data Pipeline Onboarding Audit

This audit is the `US-105` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/gdp` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `gdp` package in the module cache; the repo
  lacked `vendor/github.com/oracle/oci-go-sdk/v65/gdp` only because nothing
  imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/gdp` so `go mod vendor` keeps the package
  in the branch-local inputs.

## SDK Audit

### `GdpPipeline`

- Full CRUD family is present: `CreateGdpPipeline`, `GetGdpPipeline`,
  `ListGdpPipelines`, `UpdateGdpPipeline`, and `DeleteGdpPipeline`.
- Additional mutator is present: `ChangeGdpPipelineCompartment`.
- `GetGdpPipelineResponse` returns `GdpPipeline`.
- `ListGdpPipelinesResponse` returns `GdpPipelineCollection`.
- `ListGdpPipelinesRequest` exposes `compartmentId`, `displayName`,
  `gdpPipelineId`, and `lifecycleState`, plus page and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `INACTIVE`,
  `DELETING`, `DELETED`, `FAILED`, and `NEEDS_ATTENTION`.
- `CreateGdpPipelineResponse`, `UpdateGdpPipelineResponse`, and
  `DeleteGdpPipelineResponse` all expose `OpcWorkRequestId`.
- The mutating responses do not return a `GdpPipeline` body, so the runtime
  must recover or confirm the live resource through work-request follow-up and
  GET.
- The package also exposes service-local `GetGdpWorkRequest`,
  `ListGdpWorkRequests`, `ListGdpWorkRequestErrors`, and
  `ListGdpWorkRequestLogs` helpers.

### Auxiliary Families

- No additional top-level CRUD families are apparent beyond the service-local
  work-request auxiliaries.

## Generator Implications For `US-112`

- `GdpPipeline` is the requested initial kind and the only full CRUD family in
  the package.
- Recommended `formalSpec` is `gdppipeline`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `GdpPipeline` looks viable as a direct controller-backed generated rollout
  because GET/list expose lifecycle state and the service ships the
  work-request helpers needed for mutation follow-up.
- `US-112` should classify `INACTIVE` and `NEEDS_ATTENTION` explicitly instead
  of treating them as default success, and it should keep the list-lookup path
  keyed to `gdpPipelineId` rather than assuming a generic `id` filter.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_gdp_gdp_pipeline` as the resource,
  `oci_gdp_gdp_pipeline` as the singular data source, and
  `oci_gdp_gdp_pipelines` as the list data source.
- Provider docs publish the same single pipeline family as both a resource and
  singular/list data sources, which matches the SDK's one-family rollout
  contract.

# VN Monitoring Onboarding Audit

This audit is the `US-105` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/vnmonitoring` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `vnmonitoring` package in the module cache;
  the repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/vnmonitoring` only
  because nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/vnmonitoring` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `PathAnalyzerTest`

- Full CRUD family is present:
  `CreatePathAnalyzerTest`, `GetPathAnalyzerTest`,
  `ListPathAnalyzerTests`, `UpdatePathAnalyzerTest`, and
  `DeletePathAnalyzerTest`.
- Additional mutator is present: `ChangePathAnalyzerTestCompartment`.
- `GetPathAnalyzerTestResponse` returns `PathAnalyzerTest`.
- `ListPathAnalyzerTestsResponse` returns `PathAnalyzerTestCollection`.
- `ListPathAnalyzerTestsRequest` exposes `compartmentId`, `lifecycleState`,
  and `displayName`, plus page and sort controls.
- Lifecycle states are only `ACTIVE` and `DELETED`.
- `CreatePathAnalyzerTestResponse` and `UpdatePathAnalyzerTestResponse`
  return the resource body directly and do not expose `OpcWorkRequestId`.
- `DeletePathAnalyzerTestResponse` returns only headers and does not expose
  `OpcWorkRequestId`.
- The package exposes `GetWorkRequest`, `ListWorkRequests`,
  `ListWorkRequestErrors`, `ListWorkRequestLogs`, and
  `ListWorkRequestResults`, but the `PathAnalyzerTest` mutation responses do
  not return work-request IDs that would let the runtime follow those helpers
  directly.

### Auxiliary Families

- `GetPathAnalysis` is a read-only support surface and should stay unpublished
  initially.
- The work-request families should also stay unpublished initially because the
  selected kind does not surface work-request IDs on mutation.

## Generator Implications For `US-114`

- `PathAnalyzerTest` is the requested initial kind and the only full CRUD
  family aligned with the follow-on story.
- Recommended `formalSpec` is `pathanalyzertest`.
- Recommended async classification is `none`.
- `PathAnalyzerTest` looks viable as a direct controller-backed generated
  rollout because create and update return the resource body immediately and
  the live object exposes only steady-state lifecycle values.
- The main rollout risk is shape complexity rather than async behavior:
  `sourceEndpoint`, `destinationEndpoint`, and `protocolParameters` are
  polymorphic structures, and the lifecycle model has no transitional states
  to lean on if observed-state projection or replace-versus-update rules are
  ambiguous.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are
  `oci_vn_monitoring_path_analyzer_test` as the resource,
  `oci_vn_monitoring_path_analyzer_test` as the singular data source, and
  `oci_vn_monitoring_path_analyzer_tests` as the list data source.
- Provider docs publish the same path-analyzer-test family as both a resource
  and singular/list data sources, which matches the SDK's one-family rollout
  contract even though the SDK shape is more synchronous than the usual
  workrequest-backed service pattern.

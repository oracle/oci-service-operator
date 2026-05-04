# APM Traces Onboarding Audit

This audit is the `US-95` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/apmtraces` before `services.yaml` publishes
the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `apmtraces` package in the module cache;
  the repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/apmtraces` only
  because nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/apmtraces` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `ScheduledQuery`

- Full CRUD family is present:
  `CreateScheduledQuery`, `GetScheduledQuery`, `ListScheduledQueries`,
  `UpdateScheduledQuery`, and `DeleteScheduledQuery`.
- `GetScheduledQueryResponse`, `CreateScheduledQueryResponse`, and
  `UpdateScheduledQueryResponse` return `ScheduledQuery`.
- `ListScheduledQueriesResponse` returns `ScheduledQueryCollection` with
  `[]ScheduledQuerySummary`.
- `ListScheduledQueriesRequest` exposes required `apmDomainId`, plus
  `displayName`, page, and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`.
- The CRUD responses do not expose work-request IDs or service-local
  work-request helper APIs.
- `DeleteScheduledQueryResponse` does not return a resource body.

### Auxiliary Families

- Additional SDK-discovered families are `AggregatedSnapshot`, `Log`,
  `QuickPick`, `Span`, `StatusAutoActivate`, `Trace`, and `TraceSnapshot`.
- Those auxiliaries are read-only discovery or query surfaces and should stay
  unpublished while the first `ScheduledQuery` rollout lands.

## Generator Implications For `US-98`

- `ScheduledQuery` is the requested initial kind and the only full CRUD family
  in the package.
- Recommended `formalSpec` is `scheduledquery`.
- Recommended async classification is `lifecycle`.
- `ScheduledQuery` looks viable as a direct controller-backed generated
  rollout because create/get/list/update all project the resource body and the
  kind carries a standard lifecycle enum.
- `US-98` should still verify nested schedule and processing fields carefully:
  every operation is scoped by `apmDomainId`, and the nested processing
  configuration can fan out into other service integrations that may need
  explicit parity or exclusion decisions.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_apm_traces_scheduled_query` as both the
  resource and singular data source, plus `oci_apm_traces_scheduled_queries`
  as the list data source.
- Those provider pages align cleanly with the requested kind name, so
  `US-98` can map the generated resource directly onto an existing provider
  surface instead of inventing a repo-only alias.

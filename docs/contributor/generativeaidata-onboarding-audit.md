# Generative AI Data Onboarding Audit

This audit is the `US-118` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/generativeaidata` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `generativeaidata` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/generativeaidata` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/generativeaidata` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `EnrichmentJob`

- The selected top-level family is job-shaped rather than CRUD:
  `GenerateEnrichmentJob`, `GetEnrichmentJob`, `ListEnrichmentJobs`, and
  `CancelEnrichmentJob`.
- `GenerateEnrichmentJobRequest` and `GetEnrichmentJobRequest` are both scoped
  under `semanticStoreId`; `EnrichmentJob` identity is not a standalone
  top-level path.
- `GenerateEnrichmentJobResponse` and `GetEnrichmentJobResponse` return
  `EnrichmentJob`.
- `CancelEnrichmentJobResponse` returns headers only plus `OpcWorkRequestId`.
- `ListEnrichmentJobsResponse` returns `EnrichmentJobCollection`.
- `ListEnrichmentJobsRequest` requires both `semanticStoreId` and
  `compartmentId`, plus optional `displayName`, `lifecycleState`, page, and
  sort controls.
- Lifecycle states are `ACCEPTED`, `IN_PROGRESS`, `FAILED`, `SUCCEEDED`,
  `CANCELING`, and `CANCELED`.
- The package does not expose service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, or `ListWorkRequestLogs`
  helpers for `EnrichmentJob`.
- The job request and model both use a polymorphic
  `EnrichmentJobConfiguration` interface with `FULL_BUILD`, `PARTIAL_BUILD`,
  and `DELTA_REFRESH` variants.

### Auxiliary Families

- The package also exposes a separate `GenerateSqlFromNl` job surface.
- `GenerateSqlFromNl` and any future semantic-store-adjacent job families
  should stay unpublished while the first `EnrichmentJob` contract lands.

## Generator Implications For `US-123`

- `EnrichmentJob` is the planned first published kind for `US-123`.
- Recommended `formalSpec` is `enrichmentjob`.
- Recommended async classification is `lifecycle`.
- The required non-standard risk callout is explicit here: the pinned SDK is a
  generate/get/list/cancel job surface with semantic-store-scoped identity and
  polymorphic job configuration, not plain CRUD. `US-123` must keep the
  published contract truthful to that shape and must not rename
  `GenerateEnrichmentJob` into fake create/update/delete semantics in code or
  docs.
- Because the job object itself carries lifecycle state and the package lacks a
  matching workrequest helper seam, the first rollout should rely on direct job
  lifecycle observation rather than a generated workrequest contract.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for `EnrichmentJob` in the
  accessible local provider/docs layout.
- `US-123` should treat provider-backed imports as absent or unconfirmed until
  a pinned provider surface is proven directly.

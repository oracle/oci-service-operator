# APM Control Plane Onboarding Audit

This audit is the `US-84` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/apmcontrolplane` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `apmcontrolplane` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/apmcontrolplane` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/apmcontrolplane` so `go mod vendor` keeps
  the package in the branch-local inputs.

## SDK Audit

### `ApmDomain`

- Full CRUD family is present:
  `CreateApmDomain`, `GetApmDomain`, `ListApmDomains`, `UpdateApmDomain`, and
  `DeleteApmDomain`.
- Additional mutator is present: `ChangeApmDomainCompartment`.
- `GetApmDomainResponse` returns `ApmDomain`.
- `ListApmDomainsResponse` returns `[]ApmDomainSummary`.
- `ListApmDomainsRequest` exposes required `compartmentId`, plus
  `displayName` and `lifecycleState`, plus page and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`.
- `CreateApmDomainResponse`, `UpdateApmDomainResponse`, and
  `DeleteApmDomainResponse` all expose `OpcWorkRequestId`.
- The CRUD responses do not project an `ApmDomain` body, so the selected kind
  depends on the service-local work-request APIs to recover or confirm the
  resource after mutations.

### Auxiliary Families

- Additional SDK-discovered families are `ApmDomainWorkRequest`, `DataKey`,
  `WorkRequest`, `WorkRequestError`, and `WorkRequestLog`.
- `DataKey` and `ApmDomainWorkRequest` are list-only auxiliaries.
- The work-request families should stay unpublished initially.

## Generator Implications For `US-92`

- `ApmDomain` is the only top-level CRUD family in the package and the clear
  first published kind.
- Recommended `formalSpec` is `apmdomain`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `ApmDomain` still looks viable as a direct controller-backed generated
  rollout without handwritten runtime work because the package ships the full
  work-request helper surface needed by the shared generated seam.
- `ChangeApmDomainCompartment`, `DataKey`, `ApmDomainWorkRequest`, and the
  generic work-request auxiliaries should stay unpublished initially while the
  first `ApmDomain` rollout lands.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_apm_apm_domain` as both the resource and
  singular data source, plus `oci_apm_apm_domains` as the list data source.
- The provider resource uses `GetWorkRequest` and `ListWorkRequestErrors` for
  create, update, delete, and compartment-change flows, which matches the
  recommended `async.strategy=workrequest` baseline.

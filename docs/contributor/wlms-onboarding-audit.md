# WebLogic Management Service Onboarding Audit

This audit is the `US-105` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/wlms` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `wlms` package in the module cache; the
  repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/wlms` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/wlms` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `WlsDomain`

- The requested initial kind is **not** a full CRUD family at this pin.
- The package exposes `GetWlsDomain`, `ListWlsDomains`, `UpdateWlsDomain`,
  `DeleteWlsDomain`, and `ChangeWlsDomainCompartment`, but it does not expose
  `CreateWlsDomain`.
- `GetWlsDomainResponse` and `UpdateWlsDomainResponse` return `WlsDomain`.
- `ListWlsDomainsResponse` returns `WlsDomainCollection`.
- `ListWlsDomainsRequest` exposes `compartmentId`, `lifecycleState`,
  `displayName`, `id`, `weblogicVersion`, `middlewareType`, and
  `patchReadinessStatus`, plus page and sort controls.
- Lifecycle states are `ACTIVE`, `CREATING`, `DELETED`, `DELETING`, `FAILED`,
  `NEEDS_ATTENTION`, and `UPDATING`.
- `DeleteWlsDomainResponse` returns only headers and does not expose
  `OpcWorkRequestId`.
- The package also exposes service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, and `ListWorkRequestLogs`, but
  the visible `WlsDomain` mutation responses do not return work-request IDs
  that would let the runtime correlate those helpers safely.

### Auxiliary Families

- `AgreementRecord` is create/list only, but `CreateAgreementRecord` requires
  an existing `wlsDomainId`; it does not create a `WlsDomain`.
- `Configuration`, `ManagedInstance`, and `WlsDomainCredential` are
  get/update-only families.
- Server, patch, scan-result, backup, required-policy, and agreement families
  are read-only or workflow-support surfaces and should stay unpublished
  initially.

## Generator Implications For `US-113`

- `WlsDomain` is the requested initial kind, but it is not currently viable as
  a direct controller-backed create/get/list/update/delete rollout because the
  SDK lacks `CreateWlsDomain`.
- Recommended `formalSpec` is `wlsdomain` if the later story keeps the same
  selected kind.
- Recommended async classification is `lifecycle` for any adopt/manage-existing
  design, because GET/list expose lifecycle state directly while the visible
  mutation responses do not return work-request IDs.
- `US-113` should either re-scope to an adopt/manage-existing contract,
  identify the actual domain-creation API outside this package, or change the
  initial kind before publishing generated surfaces.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces I could verify are `oci_wlms_wls_domain` as the
  singular data source and `oci_wlms_wls_domains` as the list data source.
- I could not verify a matching `oci_wlms_wls_domain` resource page or
  resource-discovery entry, which mirrors the SDK-side absence of
  `CreateWlsDomain` and reinforces the rollout risk.

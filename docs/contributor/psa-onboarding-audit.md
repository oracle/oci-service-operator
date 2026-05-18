# Private Service Access Onboarding Audit

This audit is the `US-105` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/psa` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `psa` package in the module cache; the repo
  lacked `vendor/github.com/oracle/oci-go-sdk/v65/psa` only because nothing
  imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/psa` so `go mod vendor` keeps the package
  in the branch-local inputs.

## SDK Audit

### `PrivateServiceAccess`

- Full CRUD family is present:
  `CreatePrivateServiceAccess`, `GetPrivateServiceAccess`,
  `ListPrivateServiceAccesses`, `UpdatePrivateServiceAccess`, and
  `DeletePrivateServiceAccess`.
- Additional mutator is present: `ChangePrivateServiceAccessCompartment`.
- `GetPrivateServiceAccessResponse` returns `PrivateServiceAccess`.
- `ListPrivateServiceAccessesResponse` returns
  `PrivateServiceAccessCollection`.
- `ListPrivateServiceAccessesRequest` exposes `compartmentId`,
  `lifecycleState`, `displayName`, `id`, `vcnId`, and `serviceId`, plus page
  and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`.
- `CreatePrivateServiceAccessResponse` returns the resource body and
  `OpcWorkRequestId`.
- `UpdatePrivateServiceAccessResponse` and
  `DeletePrivateServiceAccessResponse` expose `OpcWorkRequestId`; update and
  delete do not return a `PrivateServiceAccess` body.
- The package also exposes service-local `GetPsaWorkRequest`,
  `ListPsaWorkRequests`, `ListPsaWorkRequestErrors`, and
  `ListPsaWorkRequestLogs` helpers.

### Auxiliary Families

- `PsaService` is list-only and should stay unpublished initially.
- `CancelPsaWorkRequest` is a service-local control surface and should stay
  unpublished initially alongside the work-request families.

## Generator Implications For `US-111`

- `PrivateServiceAccess` is the requested initial kind and the only full CRUD
  family aligned with the follow-on story.
- Recommended `formalSpec` is `privateserviceaccess`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `PrivateServiceAccess` looks viable as a direct controller-backed generated
  rollout because GET/list expose lifecycle state and the service ships the
  work-request helpers needed for mutation follow-up.
- The main rollout risk is identity shape: both the SDK and provider list
  surfaces include `vcnId` and `serviceId`, so `US-111` should not assume
  compartment plus display name is sufficient for safe pre-create reuse.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are
  `oci_psa_private_service_access` as the resource,
  `oci_psa_private_service_access` as the singular data source, and
  `oci_psa_private_service_accesses` as the list data source.
- Provider docs publish the same `vcn_id`, `service_id`, `display_name`, and
  `state` filters on the list data source, which matches the SDK-side identity
  contract and reinforces the bind-versus-create risk.

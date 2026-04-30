# Delegate Access Control Onboarding Audit

This audit is the `US-95` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/delegateaccesscontrol` before
`services.yaml` publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `delegateaccesscontrol` package in the
  module cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/delegateaccesscontrol` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/delegateaccesscontrol` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `DelegationControl`

- Full CRUD family is present:
  `CreateDelegationControl`, `GetDelegationControl`,
  `ListDelegationControls`, `UpdateDelegationControl`, and
  `DeleteDelegationControl`.
- Additional mutator is present: `ChangeDelegationControlCompartment`.
- `GetDelegationControlResponse`, `CreateDelegationControlResponse`, and
  `UpdateDelegationControlResponse` return `DelegationControl`.
- `ListDelegationControlsResponse` returns
  `DelegationControlSummaryCollection` with `[]DelegationControlSummary`.
- `ListDelegationControlsRequest` exposes required `compartmentId`, plus
  `lifecycleState`, `displayName`, `resourceType`, `resourceId`, page, and
  sort controls.
- Lifecycle states are `CREATING`, `ACTIVE`, `UPDATING`, `DELETING`,
  `DELETED`, `FAILED`, and `NEEDS_ATTENTION`.
- Create, update, delete, and change-compartment responses expose
  `OpcWorkRequestId`; create and update also return the resource body.
- The package also exposes service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, and `ListWorkRequestLogs`
  helpers.

### Auxiliary Families

- Additional SDK-discovered families are `DelegatedResourceAccessRequest`,
  `DelegatedResourceAccessRequestAuditLogReport`,
  `DelegatedResourceAccessRequestHistory`, `DelegationControlResource`,
  `DelegationSubscription`, `ServiceProvider`, `ServiceProviderAction`,
  `ServiceProviderInteraction`, `WorkRequest`, `WorkRequestError`, and
  `WorkRequestLog`.
- `DelegationSubscription` is also full CRUD, but `DelegationControl` is the
  requested first kind and is the top-level policy object that governs the
  request flow.

## Generator Implications For `US-102`

- `DelegationControl` is the requested initial kind and a viable first
  controller-backed surface.
- Recommended `formalSpec` is `delegationcontrol`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `DelegationControl` looks viable as a direct controller-backed generated
  rollout because GET/list expose lifecycle state and the SDK ships the
  service-local workrequest helpers needed to follow long-running mutations.
- `US-102` should still keep conditional fields explicit: `resourceType`
  currently narrows the object to `VMCLUSTER` and `CLOUDVMCLUSTER`, and the
  vault-related fields are only valid for `CLOUDVMCLUSTER`.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Accessible provider docs confirm the singular data source
  `oci_delegate_access_control_delegation_control`.
- The provider resource-discovery guides also list
  `oci_delegate_access_control_delegation_control` as a discoverable resource
  type, which is enough to anchor the later published kind name and import
  path.
- I did not locate a separate plural list data-source page in the accessible
  provider docs, so `US-102` should keep any list-import assumptions explicit
  if they matter to formal coverage.

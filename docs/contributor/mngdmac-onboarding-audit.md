# Managed Services for Mac Onboarding Audit

This audit is the `US-118` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/mngdmac` before `services.yaml` publishes
the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `mngdmac` package in the module cache; the
  repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/mngdmac` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/mngdmac` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `MacOrder`

- The pinned mutation family is `CreateMacOrder`, `GetMacOrder`,
  `ListMacOrders`, `UpdateMacOrder`, and `CancelMacOrder`; the SDK does not
  expose `DeleteMacOrder`.
- Additional mutator is present: `ChangeMacOrderCompartment`.
- `CreateMacOrderResponse` and `GetMacOrderResponse` return `MacOrder`.
- `UpdateMacOrderResponse`, `CancelMacOrderResponse`, and
  `ChangeMacOrderCompartmentResponse` return workrequest headers only, not a
  `MacOrder` body.
- `ListMacOrdersResponse` returns `MacOrderCollection`.
- `ListMacOrdersRequest` exposes `compartmentId`, `lifecycleState`,
  `displayName`, `id`, page, and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `NEEDS_ATTENTION`,
  `DELETING`, `DELETED`, and `FAILED`.
- The package exposes service-local `GetWorkRequest`, `ListWorkRequests`,
  `ListWorkRequestErrors`, and `ListWorkRequestLogs` helpers.
- `MacOrder` also carries a second `orderStatus` enum with customer-review and
  provisioning phases beyond the coarse lifecycle state.

### Auxiliary Families

- `MacDevice` is get/list-only and should stay unpublished initially.
- `TerminateMacDevice` is an auxiliary device-management workflow rather than
  part of the first `MacOrder` publication contract.

## Generator Implications For `US-120`

- `MacOrder` is the planned first published kind for `US-120`.
- Recommended `formalSpec` is `macorder`.
- Recommended async classification is `workrequest`.
- `MacOrder` is viable as the first published kind for create/update/observe
  flows because the service ships explicit workrequest helpers and get/list
  expose stable identity plus lifecycle state.
- The main rollout risk is explicit here: the pinned SDK does not expose a
  true `DeleteMacOrder`; it exposes `CancelMacOrder` instead. `US-120` must
  decide whether Kubernetes delete maps to cancel semantics or whether the
  published contract needs narrower delete behavior instead of pretending a
  delete API exists.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for `MacOrder` in the
  accessible local provider/docs layout.
- `US-120` should treat provider-backed imports as absent or unconfirmed until
  a pinned provider surface is proven directly.

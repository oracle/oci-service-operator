# Capacity Management Onboarding Audit

This audit is the `US-95` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/capacitymanagement` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `capacitymanagement` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/capacitymanagement` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/capacitymanagement` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `OccCapacityRequest`

- Full CRUD family is present:
  `CreateOccCapacityRequest`, `GetOccCapacityRequest`,
  `ListOccCapacityRequests`, `UpdateOccCapacityRequest`, and
  `DeleteOccCapacityRequest`.
- `GetOccCapacityRequestResponse`, `CreateOccCapacityRequestResponse`, and
  `UpdateOccCapacityRequestResponse` return `OccCapacityRequest`.
- `ListOccCapacityRequestsResponse` returns `OccCapacityRequestCollection`
  with `[]OccCapacityRequestSummary`.
- `ListOccCapacityRequestsRequest` exposes required `compartmentId`, plus
  `occAvailabilityCatalogId`, `namespace`, `requestType`, `displayName`, `id`,
  page, and sort controls.
- `GetOccCapacityRequestRequest` is path-addressed by `occCapacityRequestId`
  and does not carry compartment identity.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`.
- Create, update, and delete responses do not expose work-request IDs, but
  create, update, and delete all expose `RetryAfter` headers and create/update
  return the resource body.

### Auxiliary Families

- The package contains many other families, including `OccAvailabilityCatalog`,
  `OccCustomerGroup`, `OccmDemandSignal`, `OccmDemandSignalItem`, and
  multiple `Internal*` management surfaces.
- The required risk callout is explicit here: in addition to the public
  `ListOccCapacityRequests` call, the package also exposes
  `ListOccCapacityRequestsInternal`, which requires both `compartmentId` and
  `occCustomerGroupId`.
- The same internal split also appears in `UpdateInternalOccCapacityRequest`,
  so later runtime code must keep public and internal identity boundaries
  explicit.

## Generator Implications For `US-101`

- `OccCapacityRequest` is the requested initial kind and already matches the
  later story's selected package path and formal slug.
- Recommended `formalSpec` is `occcapacityrequest`.
- Recommended async classification is `lifecycle`.
- `OccCapacityRequest` looks viable as a direct controller-backed generated
  rollout because GET/create/update project the resource body and the kind
  carries a standard lifecycle enum, but the runtime must honor `RetryAfter`
  hints when requeueing long-running mutations.
- The required risk callout remains the main blocker: `US-101` must explicitly
  design resource identity and pre-create lookup around the public
  `ListOccCapacityRequests` surface versus the stricter
  `ListOccCapacityRequestsInternal` path instead of assuming the usual
  compartment-plus-display-name reuse contract.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are
  `oci_capacity_management_occ_capacity_request` as both the resource and
  singular data source, plus
  `oci_capacity_management_occ_capacity_requests` as the list data source.
- Provider docs and discovery pages confirm the same public `OccCapacityRequest`
  object name, so the later story can align the published kind to an existing
  provider surface while still treating the SDK's internal-list split as a
  repo-authored runtime decision.

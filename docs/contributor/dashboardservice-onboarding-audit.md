# Dashboard Service Onboarding Audit

This audit is the `US-95` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/dashboardservice` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `dashboardservice` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/dashboardservice` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/dashboardservice` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `DashboardGroup`

- Full CRUD family is present:
  `CreateDashboardGroup`, `GetDashboardGroup`, `ListDashboardGroups`,
  `UpdateDashboardGroup`, and `DeleteDashboardGroup`.
- Additional mutators are present: `ChangeDashboardGroup` and
  `ChangeDashboardGroupCompartment`.
- `GetDashboardGroupResponse`, `CreateDashboardGroupResponse`, and
  `UpdateDashboardGroupResponse` return `DashboardGroup`.
- `ListDashboardGroupsResponse` returns `DashboardGroupCollection` with
  `[]DashboardGroupSummary`.
- `ListDashboardGroupsRequest` exposes required `compartmentId`, plus
  `lifecycleState`, `displayName`, `id`, page, and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`.
- The CRUD responses do not expose work-request IDs or service-local
  work-request helper APIs.
- `DeleteDashboardGroupResponse` does not return a resource body.

### Auxiliary Families

- `Dashboard` is a second full CRUD family in the same package.
- `DashboardGroup` is still the requested first kind and keeps the initial
  rollout narrower than the dashboard content model itself.
- The package-level service docs warn that dashboard resources created outside
  the tenancy home region are not viewable in the Console, so that regional
  caveat must stay explicit in later docs and formal metadata.

## Generator Implications For `US-99`

- `DashboardGroup` is the requested initial kind and already has a plain
  body-returning lifecycle CRUD shape that fits a direct controller-backed
  rollout.
- Recommended `formalSpec` is `dashboardgroup`.
- Recommended async classification is `lifecycle`.
- `DashboardGroup` looks viable as a direct controller-backed generated
  rollout because GET/list return normal summary and detail bodies, create and
  update return the resource body, and no polymorphic or workrequest seam is
  required for the first pass.
- `Dashboard` and the group-change auxiliaries should stay unpublished
  initially while `US-99` proves the narrower `DashboardGroup` runtime path.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  pages, or a resource-discovery entry, for `DashboardGroup` in the accessible
  provider docs.
- The closest accessible Oracle automation surface is the official
  `oci_dashboard_service_dashboard_group` Ansible module, which suggests
  adjacent automation coverage but does not replace pinned Terraform-provider
  facts.
- `US-99` should therefore treat Terraform provider facts as absent or
  unconfirmed and keep any provider-backed import assumptions explicit until a
  pinned provider path is proven directly.

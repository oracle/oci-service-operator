# Cluster Placement Groups Onboarding Audit

This audit is the `US-84` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/clusterplacementgroups` before
`services.yaml` publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `clusterplacementgroups` package in the
  module cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/clusterplacementgroups` only
  because nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/clusterplacementgroups` so
  `go mod vendor` keeps the package in the branch-local inputs.

## SDK Audit

### `ClusterPlacementGroup`

- Full CRUD family is present:
  `CreateClusterPlacementGroup`, `GetClusterPlacementGroup`,
  `ListClusterPlacementGroups`, `UpdateClusterPlacementGroup`, and
  `DeleteClusterPlacementGroup`.
- Additional mutators are present: `ActivateClusterPlacementGroup`,
  `DeactivateClusterPlacementGroup`, and
  `ChangeClusterPlacementGroupCompartment`.
- `GetClusterPlacementGroupResponse` returns `ClusterPlacementGroup`.
- `ListClusterPlacementGroupsResponse` returns
  `ClusterPlacementGroupCollection`.
- `ListClusterPlacementGroupsRequest` exposes required `compartmentId`, plus
  `name`, `ad`, `id`, `lifecycleState`, and `compartmentIdInSubtree`, plus
  page and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `INACTIVE`,
  `DELETING`, `DELETED`, and `FAILED`.
- `CreateClusterPlacementGroupResponse` returns `ClusterPlacementGroup` and
  exposes `OpcWorkRequestId`. The update and delete responses also expose
  `OpcWorkRequestId`.

### Auxiliary Families

- Additional SDK-discovered families are `WorkRequest`, `WorkRequestError`,
  and `WorkRequestLog`.
- No other top-level CRUD family competes with `ClusterPlacementGroup` for the
  first rollout.

## Generator Implications For `US-87`

- `ClusterPlacementGroup` is the only direct controller-backed generated
  candidate in the current SDK surface and already matches the approved
  follow-on story.
- Recommended `formalSpec` is `clusterplacementgroup`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `ClusterPlacementGroup` still looks viable as a direct
  controller-backed generated rollout without handwritten runtime work because
  the create response returns the resource body and the service also ships the
  full work-request helper surface for CRUD follow-up.
- `ActivateClusterPlacementGroup`, `DeactivateClusterPlacementGroup`,
  compartment-change flows, and the work-request auxiliaries should stay
  unpublished initially while the first `ClusterPlacementGroup` rollout lands.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are
  `oci_cluster_placement_groups_cluster_placement_group` as both the resource
  and singular data source, plus
  `oci_cluster_placement_groups_cluster_placement_groups` as the list data
  source.
- The provider resource uses `GetWorkRequest` and `ListWorkRequestErrors` for
  base CRUD, while the activate, deactivate, and compartment-change helpers
  continue through lifecycle rereads. That makes `workrequest` the right
  classification for the initial CRUD rollout while keeping those extra
  mutators out of scope.

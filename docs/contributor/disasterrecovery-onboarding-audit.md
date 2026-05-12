# Disaster Recovery Onboarding Audit

This audit is the `US-149` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/disasterrecovery` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/disasterrecovery`.
- `vendor/modules.txt` and
  `vendor/github.com/oracle/oci-go-sdk/v65/disasterrecovery/` now carry the
  branch-local SDK surface for this service.

## SDK Audit

### `DrProtectionGroup`

- Full CRUD family is present: `CreateDrProtectionGroup`, `GetDrProtectionGroup`,
  `ListDrProtectionGroups`, `UpdateDrProtectionGroup`, and
  `DeleteDrProtectionGroup`.
- `CreateDrProtectionGroupResponse` returns `DrProtectionGroup` in the body and
  exposes `Location`, `Etag`, and `OpcWorkRequestId`.
- `GetDrProtectionGroupResponse` returns `DrProtectionGroup`.
- `ListDrProtectionGroupsRequest` requires `compartmentId` and supports
  `drProtectionGroupId`, `displayName`, `lifecycleState`, `role`,
  `lifecycleSubState`, paging, and sort controls.
- `ListDrProtectionGroupsResponse` returns `DrProtectionGroupCollection`, whose
  items are `DrProtectionGroupSummary` rather than full `DrProtectionGroup`
  bodies.
- `UpdateDrProtectionGroupResponse` and `DeleteDrProtectionGroupResponse`
  return headers only; both expose `OpcWorkRequestId` and do not return a
  `DrProtectionGroup` body.
- `DrProtectionGroup` carries `role`, `lifecycleState`,
  `lifecycleSubState`, `peerId`, `peerRegion`, `lifeCycleDetails`, and a
  polymorphic `members []DrProtectionGroupMember` slice. Later rollout work
  must not flatten that member shape.
- Lifecycle states are `CREATING`, `ACTIVE`, `UPDATING`, `INACTIVE`,
  `NEEDS_ATTENTION`, `DELETING`, `DELETED`, and `FAILED`.
- Roles are `PRIMARY`, `STANDBY`, and `UNCONFIGURED`.
- Lifecycle sub-states are `DR_DRILL_IN_PROGRESS`,
  `DR_PLAN_EXECUTION_IN_PROGRESS`, and
  `AUTOMATIC_DR_PLAN_EXECUTION_IN_PROGRESS`.
- The package exposes `GetWorkRequest`, `ListWorkRequests`,
  `ListWorkRequestErrors`, and `ListWorkRequestLogs`.

### Auxiliary Families

- The package also exposes large out-of-scope surfaces for DR plans and plan
  executions, automatic DR configurations, associate or disassociate flows, DR
  role changes, compartment changes, failover or switchover helpers, and many
  polymorphic member detail families across database, compute, storage,
  networking, and Kubernetes resources.

## Generator Implications For `US-150`

- `DrProtectionGroup` is the planned first published kind for `US-150`.
- Recommended `formalSpec` is `drprotectiongroup`.
- Recommended async classification is `workrequest` for `create`, `update`, and
  `delete`, because all three lifecycle mutations expose `OpcWorkRequestId` and
  the package ships the supporting work-request read surfaces.
- The main rollout risks are explicit here: list returns summaries instead of
  full bodies, update and delete are header-only, and the resource's `members`
  field is polymorphic. `US-150` must keep member modeling, role handling, peer
  fields, and lifecycle sub-state truthfully surfaced without broadening into
  DR plan or member-action families.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching checked-in provider-fact imports or local repo
  evidence for `DrProtectionGroup` in this checkout, so `US-150` should treat
  provider-backed imports as absent or unconfirmed until a pinned provider
  surface is proven directly.

---
schemaVersion: 1
surface: repo-authored-semantics
service: dashboardservice
slug: dashboardgroup
gaps:
  - category: drift-guard
    status: open
    stopCondition: "Close when the generated DashboardGroup spec preserves explicit empty-string update intent for displayName and description so the runtime can distinguish omission from clear-to-empty."
---

# Logic Gaps

`DashboardGroup` no longer relies on the scaffold-only bootstrap runtime, but
one string-update gap remains open until the generated spec can represent
explicit clear intent separately from omission.

## Current runtime path

- `DashboardGroup` keeps the generated controller, service-manager shell, and
  registration wiring, but the published runtime contract is reviewed in
  `pkg/servicemanager/dashboardservice/dashboardgroup/dashboardgroup_runtime_client.go`.
- The vendored SDK exposes direct
  `Create/Get/List/Update/DeleteDashboardGroup` operations plus the auxiliary
  mutator `ChangeDashboardGroupCompartment`. The reviewed runtime treats
  `CREATING` as provisioning, `UPDATING` as updating, `ACTIVE` as success,
  `FAILED` as terminal failure, and delete confirmation waits through
  `DELETING` until `DELETED` or NotFound.
- Pre-create reuse is bounded. Existing-before-create lookup runs only when
  both `spec.compartmentId` and `spec.displayName` are non-empty, then
  `ListDashboardGroups` scopes by exact `compartmentId` and `displayName` and
  reuses only a unique candidate in reusable lifecycle states (`ACTIVE`,
  `CREATING`, or `UPDATING`). Duplicate exact-name matches fail instead of
  guessing.
- Mutable drift is explicit: `displayName`, `description`, `freeformTags`, and
  `definedTags` reconcile in place when the desired value is representable.
  The handwritten update builder keeps non-empty string updates for
  `displayName` and `description` and preserves explicit empty-map clears for
  both tag maps. Because the current generated spec models `displayName` and
  `description` as plain optional strings, empty-string values collapse with
  omission and therefore do not clear the live OCI value.
  `compartmentId` and `systemTags` remain replacement-only drift, and
  `ChangeDashboardGroupCompartment` stays out of scope for the published
  runtime.
- Create, get, and update return the `DashboardGroup` body directly, while
  delete returns headers only. The reviewed runtime stays lifecycle-based with
  read-after-write plus confirm-delete rereads; there is no work-request
  polling surface for this kind.
- Required status projection remains part of the repo-authored contract. The
  runtime projects the shared OSOK lifecycle/async/request breadcrumbs plus the
  published `status.id`, `status.displayName`, `status.description`,
  `status.compartmentId`, `status.timeCreated`, `status.timeUpdated`,
  `status.lifecycleState`, `status.freeformTags`, `status.definedTags`, and
  `status.systemTags` fields when OCI returns them.
- The Dashboards service caveat remains explicit: resources created outside the
  tenancy home region are not viewable in the Console even though the SDK still
  permits those calls, so the generated docs and formal metadata keep that
  limitation visible.

## Open Gap

- Explicit clear-to-empty intent for `displayName` and `description` is not
  representable in the current generated `DashboardGroup` spec shape. Empty
  strings are treated as omission during reconcile, so the runtime leaves the
  current OCI string value in place instead of sending an explicit clear until
  the spec can model that distinction.

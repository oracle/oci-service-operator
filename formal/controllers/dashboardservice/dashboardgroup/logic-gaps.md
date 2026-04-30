---
schemaVersion: 1
surface: repo-authored-semantics
service: dashboardservice
slug: dashboardgroup
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `dashboardservice/DashboardGroup` row
after the runtime review replaced the scaffold bootstrap with a published
generated-runtime contract plus a narrow manual update/reuse seam.

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
  `definedTags` reconcile in place. The handwritten update builder preserves
  clear-to-empty intent for `displayName`, `description`, and both tag maps.
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

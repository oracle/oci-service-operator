---
schemaVersion: 1
surface: repo-authored-semantics
service: dashboardservice
slug: dashboard
gaps:
  - category: drift-guard
    status: open
    stopCondition: "Close when the published Dashboard spec can distinguish omission from explicit clear for displayName, description, and null config so the runtime can safely propagate empty-string and null updates without treating absent fields as deletes."
---

# Logic Gaps

`Dashboard` no longer uses the scaffold-only placeholder, but one explicit
clear-intent gap remains open until the generated spec can separate omission
from clear for the affected fields.

## Current runtime path

- `Dashboard` keeps the generated controller, service-manager shell, and
  registration wiring, but the reviewed runtime contract is owned by
  `pkg/servicemanager/dashboardservice/dashboard/dashboard_runtime_client.go`.
- The vendored SDK exposes direct `Create/Get/List/Update/DeleteDashboard`
  operations plus the auxiliary mutator `ChangeDashboardGroup`. The published
  runtime stays narrower: it uses the CRUD family only, keeps
  `ChangeDashboardGroup` out of scope, and treats `dashboardGroupId` moves as
  replacement-only drift.
- Create, get, and update return the polymorphic `Dashboard` interface. The
  reviewed runtime dispatches request and response bodies by `schemaVersion`,
  currently supports only the `V1` subtype, defaults an omitted
  `spec.schemaVersion` to `V1`, and fails fast on unsupported values before
  sending an OCI request.
- Lifecycle handling is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` settles success, `FAILED` is a
  terminal failure state without requeue, and delete confirmation waits
  through `DELETING` until `DELETED` or NotFound.
- Pre-create reuse is bounded. Existing-before-create lookup runs only when
  both `spec.dashboardGroupId` and `spec.displayName` are non-empty, then
  `ListDashboards` scopes by exact `dashboardGroupId` and `displayName` and
  reuses only a unique candidate in reusable lifecycle states (`ACTIVE`,
  `CREATING`, or `UPDATING`). Duplicate exact-name matches fail instead of
  guessing.
- Mutable drift is explicit: `displayName`, `description`, `freeformTags`,
  `definedTags`, `config`, and `widgets` reconcile in place when the desired
  value is representable. `schemaVersion`, `dashboardGroupId`, and
  `systemTags` remain replacement-only drift for the published runtime.
- Required status projection remains part of the repo-authored contract. The
  runtime projects the shared OSOK lifecycle/async/request breadcrumbs plus the
  published `status.id`, `status.dashboardGroupId`, `status.displayName`,
  `status.description`, `status.compartmentId`, `status.timeCreated`,
  `status.timeUpdated`, `status.lifecycleState`, `status.freeformTags`,
  `status.definedTags`, `status.systemTags`, `status.schemaVersion`,
  `status.widgets`, and `status.config` fields when OCI returns them.
- The Dashboards service caveat remains explicit: resources created outside the
  tenancy home region are not viewable in the Console even though the SDK still
  permits those calls, so the generated docs and formal metadata keep that
  limitation visible.

## Open Gap

- Explicit clear-to-empty intent for `displayName` and `description`, plus
  explicit clear-to-null intent for `config`, are not representable in the
  current generated `Dashboard` spec shape. Empty strings and `null` collapse
  with omission during reconcile, so the runtime leaves the current OCI values
  in place instead of sending destructive clears until the spec can model those
  intents separately.

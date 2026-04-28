---
schemaVersion: 1
surface: repo-authored-semantics
service: core
slug: servicegateway
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- Success is OCI `AVAILABLE`.
- Requeue covers OCI `PROVISIONING`, `TERMINATING`, and `TERMINATED`.
- Delete confirmation requires `GetServiceGateway` to stop finding the
  resource. Observed OCI `TERMINATING` and `TERMINATED` remain intermediate
  finalizer-holding states instead of confirming deletion.
- Supported in-place updates are limited to `blockTraffic`, `displayName`,
  `definedTags`, `freeformTags`, `routeTableId`, and `services`, matching the
  pinned `UpdateServiceGatewayDetails` SDK surface and the handwritten runtime.
- Service-list reconciliation is normalized by `serviceId`, so OCI ordering and
  response-only fields such as `serviceName` do not trigger spurious updates.
- The runtime reconciles removals for mutable `displayName`, `definedTags`,
  `freeformTags`, `routeTableId`, and `services` when OCI still retains those
  values.
- Create-only drift is rejected for `compartmentId` and `vcnId`.
- Status projection is authoritative for `id`, `compartmentId`, `vcnId`,
  `blockTraffic`, `services`, `timeCreated`, `displayName`, tags,
  `routeTableId`, `lifecycleState`, and `status.ocid`, and clears stale
  optional fields when OCI later omits them.
- The pinned `CreateServiceGatewayDetails` SDK surface does not expose
  `blockTraffic`, so repo-authored create behavior treats it as an update-only
  field on create: once the created or recreated gateway reaches a
  non-retryable lifecycle state, the runtime reapplies `blockTraffic` through
  `UpdateServiceGateway` whenever OCI still reports a different value.
- Tracked OCI identity prefers `status.ocid` and falls back to projected `id`,
  so delete confirmation and post-create parity keep working for older status
  records until `status.ocid` is backfilled.

## Authority and scoped cleanup

- `formal/controllers/core/servicegateway/*` is the authoritative formal path
  for the handwritten ServiceGateway runtime.
- `formal/controller_manifest.tsv` still contains a separate `coreservicegateway`
  row. Any deduplication between those rows is separate cleanup and is not
  folded into this runtime-contract task.
- List/bind-style provider datasource semantics are not part of the handwritten
  runtime contract here. The runtime observes by tracked OCID and recreates only
  on OCI not-found.

## Why this row is seeded

- The handwritten ServiceGateway runtime now defines explicit success, requeue,
  mutation, status-projection, normalized service-list update, and
  delete-confirmation semantics.
- Secret side effects and bind-by-name semantics remain out of scope because the
  runtime reconciles directly against OCI identity and does not publish
  connection material.

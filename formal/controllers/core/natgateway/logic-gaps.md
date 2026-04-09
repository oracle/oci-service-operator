---
schemaVersion: 1
surface: repo-authored-semantics
service: core
slug: natgateway
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- Success is OCI `AVAILABLE`.
- Requeue covers OCI `PROVISIONING`, `TERMINATING`, and `TERMINATED`.
- Delete confirmation requires `GetNatGateway` to stop finding the resource.
  Observed OCI `TERMINATING` and `TERMINATED` remain intermediate
  finalizer-holding states instead of confirming deletion.
- Supported in-place updates are limited to `blockTraffic`, `displayName`,
  `definedTags`, `freeformTags`, and `routeTableId`, matching the pinned
  `UpdateNatGatewayDetails` SDK surface and the handwritten runtime.
- The runtime reconciles removals for mutable `displayName`, `definedTags`,
  `freeformTags`, and `routeTableId` when OCI still retains those values.
- The runtime persists `status.publicIpIdCreateIntent` for the current tracked
  NatGateway. Create-only drift is rejected for `compartmentId` and `vcnId`,
  and for `publicIpId` when that persisted create intent is `Explicit` or the
  current spec explicitly sets `publicIpId`. When the tracked NatGateway was
  first managed with `publicIpId` omitted and the current spec still omits it,
  OCI-assigned `publicIpId` values are accepted and projected into status.
- Status projection is authoritative for `id`, `compartmentId`, `vcnId`,
  `blockTraffic`, `natIp`, `timeCreated`, `displayName`, tags, `publicIpId`,
  `routeTableId`, `lifecycleState`, and `status.ocid`, and clears stale optional
  fields when OCI later omits them.

## Authority and scoped cleanup

- `formal/controllers/core/natgateway/*` is the authoritative formal path for
  the handwritten NatGateway runtime.
- `formal/controller_manifest.tsv` still contains a separate `corenatgateway`
  row. Any deduplication between those rows is separate cleanup and is not
  folded into this runtime-contract task.
- List/bind-style provider datasource semantics are not part of the handwritten
  runtime contract here. The runtime observes by tracked OCID and recreates only
  on OCI not-found.

## Why this row is seeded

- The handwritten NatGateway runtime now defines explicit success, requeue,
  mutation, status-projection, and delete-confirmation semantics.
- Secret side effects and bind-by-name semantics remain out of scope because the
  runtime reconciles directly against OCI identity and does not publish
  connection material.

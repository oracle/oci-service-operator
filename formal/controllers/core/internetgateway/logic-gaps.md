---
schemaVersion: 1
surface: repo-authored-semantics
service: core
slug: internetgateway
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- Success is OCI `AVAILABLE`.
- Requeue covers OCI `PROVISIONING`, `TERMINATING`, and `TERMINATED`.
- Delete confirmation requires `GetInternetGateway` to stop finding the
  resource. Observed OCI `TERMINATING` and `TERMINATED` remain intermediate
  finalizer-holding states instead of confirming deletion.
- Supported in-place updates are limited to `displayName`, `definedTags`,
  `freeformTags`, `isEnabled`, and `routeTableId`, matching the pinned
  `UpdateInternetGatewayDetails` SDK surface and the handwritten runtime.
- Create-only drift is rejected for `compartmentId` and `vcnId`.
- Status projection is authoritative for `id`, `compartmentId`, `vcnId`,
  `displayName`, tags, `routeTableId`, `isEnabled`, `lifecycleState`, and
  `timeCreated`, and clears stale optional fields when OCI later omits them.

## Authority and scoped cleanup

- `formal/controllers/core/internetgateway/*` is the authoritative formal path
  for the handwritten InternetGateway runtime.
- `formal/controller_manifest.tsv` still contains a separate `coreinternetgateway`
  row. Any deduplication between those rows is separate cleanup and is not folded
  into this runtime-contract task.
- List/bind-style provider datasource semantics are not part of the handwritten
  runtime contract here. The runtime observes by tracked OCID and recreates only
  on OCI not-found.

## Why this row is seeded

- The handwritten InternetGateway runtime now defines explicit success,
  requeue, mutation, status-projection, and delete-confirmation semantics.
- Secret side effects and bind-by-name semantics remain out of scope because the
  runtime reconciles directly against OCI identity and does not publish
  connection material.

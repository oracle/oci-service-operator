---
schemaVersion: 1
surface: repo-authored-semantics
service: core
slug: drg
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- Success is OCI `AVAILABLE`.
- Requeue covers OCI `PROVISIONING`, the defensive literal `UPDATING`,
  `TERMINATING`, and `TERMINATED`.
- The runtime first trusts tracked `status.osokStatus.ocid`. When no tracked
  OCI identity exists and `spec.displayName` is non-empty, it searches
  `ListDrgs` in the requested compartment, rereads a single exact
  `displayName` match through `GetDrg`, and fails on ambiguous duplicates
  instead of guessing.
- Delete confirmation requires `GetDrg` to stop finding the resource. Observed
  OCI `TERMINATING` and `TERMINATED` remain intermediate finalizer-holding
  states instead of confirming deletion.
- Supported in-place updates are limited to `displayName`, `definedTags`,
  `freeformTags`, and `defaultDrgRouteTables`, matching the pinned
  `UpdateDrgDetails` SDK surface and the handwritten runtime.
- `defaultDrgRouteTables` reconcile only for explicitly requested non-empty
  attachment-type defaults; omitted entries are left unchanged.
- Create-only drift is rejected for `compartmentId`.
- Status projection is authoritative for `id`, `compartmentId`, `displayName`,
  tags, `lifecycleState`, `timeCreated`, `defaultDrgRouteTables`,
  `defaultExportDrgRouteDistributionId`, and `status.ocid`, and clears stale
  optional fields when OCI later omits them.
- The create-fallback wrapper only recovers first-create post-read failures:
  when status already carries a newly created DRG OCID plus lifecycle
  `AVAILABLE` or a provisioning-like state, the wrapper reclassifies the create
  as successful; once a tracked OCI identity already exists, errors are
  returned unchanged.

## Authority and scope

- `formal/controllers/core/drg/*` is the authoritative formal path for the
  handwritten DRG runtime.
- Promotion in this story is intentionally limited to the single `core/drg`
  manifest row; no other `core/*` formal rows move with it.

## Why this row is seeded

- The handwritten DRG runtime now defines explicit bind-versus-create reuse,
  create-fallback, mutation, status-projection, and delete-confirmation
  semantics.
- Secret side effects remain out of scope because DRG reconciliation does not
  publish connection material.

---
schemaVersion: 1
surface: repo-authored-semantics
service: mediaservices
slug: mediaasset
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `mediaservices/MediaAsset` row after
the scaffold placeholder was replaced with repo-authored lifecycle, mutation,
and delete semantics.

## Current seeded contract

- `MediaAsset` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/mediaservices/mediaasset/mediaasset_runtime_client.go`.
- OCI lifecycle classification is explicit: `CREATING` requeues as
  provisioning, `UPDATING` requeues as updating, `ACTIVE` settles success,
  `FAILED` is terminal without requeue, and delete confirmation observes
  `DELETING` until `DELETED` or OCI NotFound.
- Pre-create reuse is guarded and explicit. The runtime skips list-before-create
  when the spec has no identity stronger than required `compartmentId` plus
  `type`, or when `mediaWorkflowJobId` or `sourceMediaWorkflowVersion` is set
  without `sourceMediaWorkflowId`. When reuse is allowed,
  `ListMediaAssets` scopes by exact `compartmentId`, `type`, and any provided
  `displayName`, `bucketName` and `objectName`, workflow lineage, and
  parent/master asset identifiers, then reuses only a unique candidate in
  reusable lifecycle states (`ACTIVE`, `CREATING`, or `UPDATING`). Duplicate
  matches fail instead of guessing.
- Mutation policy is explicit: only `UpdateMediaAssetDetails` fields
  `displayName`, `type`, `parentMediaAssetId`, `masterMediaAssetId`, `metadata`,
  `mediaAssetTags`, `freeformTags`, and `definedTags` reconcile in place. The
  handwritten update builder preserves explicit empty clears for the mutable
  string identifiers, metadata/tag slices, and tag maps. The auxiliary mutator
  `ChangeMediaAssetCompartment` stays out of scope, and `compartmentId`, source
  workflow lineage, object-location fields, segment ranges, and locks remain
  create-only drift for the published runtime.
- Create, get, and update return the `MediaAsset` body directly, while delete
  returns only headers and no `opc-work-request-id`. The reviewed runtime stays
  lifecycle-based and records shared request breadcrumbs when present; there is
  no service-local work-request polling surface for this kind.
- Delete policy is explicit: the runtime calls `DeleteMediaAsset` without
  `deleteMode` or `isLockOverride`, so hierarchical delete modes
  `DELETE_CHILDREN` and `DELETE_DERIVATIONS` stay outside the published
  contract. Delete confirmation relies on follow-up `GetMediaAsset` rereads
  until OCI reports `DELETED` or NotFound.
- Matching create-time locks are normalized out of post-create parity checks by
  ignoring server-populated `timeCreated` metadata. Changing the requested
  locks after create is explicit unsupported drift, not an implicit lock
  mutation flow.

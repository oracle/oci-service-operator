---
schemaVersion: 1
surface: repo-authored-semantics
service: gdp
slug: gdppipeline
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `gdp/GdpPipeline` row after the
runtime audit replaced the scaffold-only generated semantics with a reviewed
work-request-backed contract.

## Current runtime path

- `GdpPipeline` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/gdp/gdppipeline/gdppipeline_runtime_client.go`.
- Create, update, and delete are work-request-backed. The reviewed runtime
  reads `opc-work-request-id`, polls `GetGdpWorkRequest`, recovers the pipeline
  OCID from work-request resources, and then rereads `GetGdpPipeline` before
  projecting status or confirming delete.
- Lifecycle handling is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` and `INACTIVE` settle success,
  `DELETING` keeps finalizer-backed delete confirmation pending, and
  `FAILED` plus `NEEDS_ATTENTION` are terminal non-success states.
- Pre-create lookup is explicit. The runtime reuses only a single candidate
  that matches exact `compartmentId`, `displayName`, `pipelineType`, and
  `peeringRegion`, reuses tracked identity through the `gdpPipelineId` list
  filter when an OCID is already known, and confirms `bucketDetails` from a
  follow-up GET before binding to an existing pipeline.
- Mutation policy is explicit: only `UpdateGdpPipelineDetails` fields
  `displayName`, `description`, `serviceLogGroupId`, `fileTypes`,
  `authorizationDetails`, `isFileOverrideInDestinationEnabled`,
  `isScanningEnabled`, `isChunkingEnabled`, `isApprovalNeeded`,
  `approvalKeyVaultId`, `freeformTags`, and `definedTags` reconcile in place.
  `bucketDetails`, `compartmentId`, `peeringRegion`, and `pipelineType` remain
  replacement-only drift for the published runtime. The handwritten update-body
  builder preserves clear-to-empty intent for strings, maps, and false
  booleans instead of dropping zero values. Non-empty `fileTypes` replacements
  reconcile in place, but zero-length `fileTypes` clears remain out-of-scope
  because the pinned SDK update request omits empty slices on the wire.
- Auxiliary SDK mutators `ChangeGdpPipelineCompartment`, `PeerGdpPipeline`,
  `RotateGdpPipelineKeys`, `StartGdpPipeline`, and `StopGdpPipeline` remain
  out-of-scope drift for the published runtime surface.
- The row keeps `secret_side_effects = none`; the reviewed runtime does not
  create or mutate Kubernetes Secrets.

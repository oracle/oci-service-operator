---
schemaVersion: 1
surface: repo-authored-semantics
service: containerengine
slug: nodepool
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `containerengine/NodePool` row after
the scaffold placeholder was replaced with repo-authored lifecycle, mutation,
and list-lookup semantics.

## Current seeded contract

- `NodePool` keeps the generated `NodePoolServiceManager` shell but overrides
  the generated client seam with
  `pkg/servicemanager/containerengine/nodepool/nodepool_runtime_client.go`.
- OCI lifecycle classification is explicit: `CREATING` requeues as
  provisioning, `UPDATING` requeues as updating, `ACTIVE`, `INACTIVE`, and
  `NEEDS_ATTENTION` settle success, `FAILED` is terminal without requeue, and
  delete confirmation observes `DELETING` until `DELETED` or NotFound.
- Mutation policy is explicit: only `UpdateNodePoolDetails` fields are mutable
  in place. That mutable surface is `name`, `kubernetesVersion`,
  `initialNodeLabels`, `quantityPerSubnet`, `subnetIds`, `nodeConfigDetails`,
  `nodeMetadata`, `nodeSourceDetails`, `sshPublicKey`, `nodeShape`,
  `nodeShapeConfig`, `definedTags`, `freeformTags`,
  `nodeEvictionNodePoolSettings`, and `nodePoolCyclingDetails`. Fields omitted
  from `UpdateNodePoolDetails` remain create-only drift and never open implicit
  replacement.
- Deprecated placement inputs stay explicit: `nodeConfigDetails` conflicts with
  `subnetIds` and `quantityPerSubnet`, while `nodeImageName` remains create-only
  drift beside the preferred `nodeSourceDetails` path. When
  `nodeConfigDetails` is the active create or update path, the runtime strips
  deprecated placement fields from the serialized SDK body instead of sending
  empty placeholder values such as `subnetIds: []`.
- Pre-create lookup semantics are explicit: `ListNodePools` searches by
  `compartmentId`, `clusterId`, `name`, and reusable lifecycle state, and only
  a single exact-name match in `ACTIVE`, `CREATING`, `UPDATING`, `INACTIVE`, or
  `NEEDS_ATTENTION` is safe to reuse. `FAILED`, `DELETING`, and `DELETED`
  candidates are never reusable.
- Create, update, and delete are work-request-backed at the API boundary
  because their SDK responses return `opc-work-request-id`, but the seeded
  lifecycle contract is still expressed in observed NodePool states rather than
  separate work-request status objects.
- Kubernetes secret reads and writes are out of scope for `NodePool`; the row
  keeps `secret_side_effects = none`.

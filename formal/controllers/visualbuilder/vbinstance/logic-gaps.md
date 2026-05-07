---
schemaVersion: 1
surface: repo-authored-semantics
service: visualbuilder
slug: vbinstance
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `visualbuilder/VbInstance` row after
the reviewed runtime contract replaced the provisional scaffold semantics.

## Current runtime path

- `VbInstance` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/visualbuilder/vbinstance/vbinstance_runtime_client.go`.
- The vendored SDK exposes `Create/Get/List/Update/DeleteVbInstance` plus
  lifecycle states `CREATING`, `UPDATING`, `ACTIVE`, `INACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`. The published runtime keeps shared
  `generatedruntime` lifecycle handling: `CREATING`, `UPDATING`, and
  `DELETING` requeue; `ACTIVE` and `INACTIVE` settle success; `FAILED` is
  terminal without requeue; delete confirmation waits for `DELETED` or NotFound.
- Pre-create lookup is explicit: `ListVbInstances` searches by exact
  `compartmentId` and `displayName`, then reuses only a single candidate in
  reusable lifecycle states (`ACTIVE`, `CREATING`, `UPDATING`, or `INACTIVE`).
  Duplicate exact-name matches fail instead of guessing.
- Mutation policy is explicit: in-place reconcile covers `displayName`,
  `nodeCount`, `freeformTags`, `definedTags`, one-way enablement of
  `isVisualBuilderEnabled`, `customEndpoint`, `alternateCustomEndpoints`, and
  the mutable `networkEndpointDetails` fields that OCI echoes back. The runtime
  preserves explicit empty-map and empty-slice clears for tag maps, alternate
  custom endpoints, public allowlists, and private-endpoint NSG lists instead
  of silently dropping those updates.
- `IdcsOpenId` remains create-only in the published runtime contract even
  though the SDK accepts it on update. OCI does not project that field back on
  `VbInstance`, so post-create reconciles do not attempt drift detection or
  blind reapplication.
- `ConsumptionModel` and private-endpoint `privateEndpointIp` remain
  create-time-only inputs for the published runtime contract. `ChangeVbInstanceCompartment`,
  `StartVbInstance`, `StopVbInstance`, and
  `ReconfigurePrivateEndpointVbInstance` remain out-of-scope auxiliary drift for
  the reviewed create/get/list/update/delete surface.
- Create, update, and delete responses expose only `opc-work-request-id`
  headers. The reviewed runtime follows those service-local work requests
  through `GetWorkRequest`, recovers the `VbInstance` OCID from the work-request
  resources when OCI returns it, and then rereads `GetVbInstance` as the
  authoritative observed-state source.
- `CustomEndpoint`, `AlternateCustomEndpoints`, `ConsumptionModel`, and the
  public/private `NetworkEndpointDetails` variants stay inside required status
  projection, alongside the service and management NAT/VCN identifiers. The row
  keeps `secret_side_effects = none`.

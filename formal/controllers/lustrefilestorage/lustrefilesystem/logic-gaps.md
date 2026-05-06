---
schemaVersion: 1
surface: repo-authored-semantics
service: lustrefilestorage
slug: lustrefilesystem
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `lustrefilestorage/LustreFileSystem`
row after the onboarding audit was converted into an explicit generatedruntime
contract.

## Current runtime intent

- `LustreFileSystem` is published as a controller-backed generated service with
  generated controller, service-manager shell, and registration wiring.
- OCI lifecycle classification is explicit: `CREATING` requeues as
  provisioning, `UPDATING` requeues as updating, `ACTIVE` and `INACTIVE`
  settle success, `FAILED` is terminal without requeue, and delete
  confirmation observes `DELETING` until `DELETED` or NotFound.
- Pre-create lookup semantics are explicit: `ListLustreFileSystems` searches by
  exact `compartmentId`, `availabilityDomain`, `displayName`, and reusable
  lifecycle state, and only a single exact-name match in `ACTIVE`, `CREATING`,
  `UPDATING`, or `INACTIVE` is safe to reuse.
- Mutation policy is explicit: only `UpdateLustreFileSystemDetails` fields
  reconcile in place. Mutable drift is limited to `displayName`,
  `fileSystemDescription`, `freeformTags`, `definedTags`, `nsgIds`, `kmsKeyId`,
  `capacityInGBs`, `rootSquashConfiguration`, and `maintenanceWindow`.
- Create, update, and delete are work-request-backed at the API boundary
  because their SDK responses return `opc-work-request-id`, but steady-state
  readiness is still classified from observed `LustreFileSystem` lifecycle
  states rather than separate published work-request kinds.
- Kubernetes secret reads and writes are out of scope for `LustreFileSystem`;
  the row keeps `secret_side_effects = none`.

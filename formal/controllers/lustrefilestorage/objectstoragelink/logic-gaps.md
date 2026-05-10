---
schemaVersion: 1
surface: repo-authored-semantics
service: lustrefilestorage
slug: objectstoragelink
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded
`lustrefilestorage/ObjectStorageLink` row after the publish contract was made
explicit for the generated runtime path.

## Current runtime intent

- `ObjectStorageLink` is published as a controller-backed generated service with
  generated controller, service-manager shell, registration wiring, and a
  small handwritten hook layer in
  `pkg/servicemanager/lustrefilestorage/objectstoragelink/objectstoragelink_runtime_client.go`
  that wires Lustre delete work-request polling into the generatedruntime
  baseline.
- OCI lifecycle classification is explicit: `CREATING` requeues as
  provisioning, `ACTIVE` settles success, `DELETING` requeues as terminating
  during delete confirmation, `FAILED` is terminal without requeue, and delete
  completes after `DELETED` or NotFound.
- Pre-create lookup is explicit: `ListObjectStorageLinks` searches exact
  `compartmentId`, `availabilityDomain`, `lustreFileSystemId`, and
  `displayName`, then only a single exact-name match in `ACTIVE` or `CREATING`
  is safe to reuse. Duplicate exact-name matches fail instead of guessing.
- Mutation policy is explicit: only `displayName`, `isOverwrite`,
  `freeformTags`, and `definedTags` reconcile in place because that is the
  full `UpdateObjectStorageLinkDetails` surface. `compartmentId`,
  `availabilityDomain`, `lustreFileSystemId`, `fileSystemPath`, and
  `objectStoragePrefix` remain replacement-only drift, and
  `ChangeObjectStorageLinkCompartment` stays out of scope for the published
  runtime.
- Create and update are synchronous body-returning calls followed by standard
  read-after-write observation. Delete is the only work-request-backed
  mutation; the runtime records the delete work request in
  `status.async.current` and holds the finalizer until `GetObjectStorageLink`
  confirms `DELETED` or the resource disappears.
- Required status projection remains part of the contract. The published kind
  keeps the shared OSOK status plus the read-model fields OCI returns,
  including lifecycle detail, sync job identifiers, tags, and the bound
  Lustre/Object Storage identity fields.
- Kubernetes secret reads and writes are out of scope for
  `ObjectStorageLink`; the row keeps `secret_side_effects = none`.

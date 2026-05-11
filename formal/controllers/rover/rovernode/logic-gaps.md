---
schemaVersion: 1
surface: repo-authored-semantics
service: rover
slug: rovernode
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `rover/RoverNode` row after the
scaffold baseline is replaced by the reviewed lifecycle-driven
generated-runtime contract.

## Current runtime contract

- `RoverNode` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/rover/rovernode/rovernode_runtime_client.go`.
- The vendored SDK exposes
  `Create/Get/List/Update/DeleteRoverNode`,
  `ChangeRoverNodeCompartment`, and generic Rover work-request helpers. The
  reviewed runtime treats CRUD as lifecycle-driven: `CREATING` requeues as
  provisioning, `UPDATING` requeues as updating, `ACTIVE` settles success,
  `FAILED` is terminal without requeue, and delete confirmation waits through
  `DELETING` until `DELETED` or NotFound.
- Pre-create lookup is explicit. `ListRoverNodes` scopes by exact
  `compartmentId`, `displayName`, and `shape`, then reuses only a unique exact
  `serialNumber` match in reusable `ACTIVE`, `CREATING`, or `UPDATING`
  summaries. The reviewed runtime intentionally removes provider-managed
  `lifecycleState` from the list request so desired-spec drift cannot hide a
  reusable candidate.
- Mutation policy stays aligned with `UpdateRoverNodeDetails` for
  `displayName`, `shape`, `serialNumber`, `customerShippingAddress`,
  `nodeWorkloads`, `superUserPassword`, `unlockPassphrase`,
  `pointOfContact`, `pointOfContactPhoneNumber`,
  `oracleShippingTrackingUrl`, `shippingPreference`, `shippingVendor`,
  `timePickupExpected`, `timeReturnWindowStarts`, `timeReturnWindowEnds`,
  `isImportRequested`, `importCompartmentId`, `importFileBucket`,
  `dataValidationCode`, `publicKey`, `certificateAuthorityId`,
  `timeCertValidityEnd`, `commonName`, `certCompartmentId`,
  `certKeyAlgorithm`, `certSignatureAlgorithm`, `enclosureType`,
  `freeformTags`, and `definedTags`. The separate
  `ChangeRoverNodeCompartment` action stays unpublished; `compartmentId` and
  `masterKeyId` remain replacement-only drift.
- `LifecycleState`, `LifecycleStateDetails`, `SystemTags`, and
  `nodeWorkloads.workRequestId` are provider-managed request-shape fields on
  the generated SDK types. The reviewed runtime normalizes them out of desired
  parity and never sends them on create or update writes.
- The rover package exposes `GetWorkRequest` and `ListWorkRequests`, but
  RoverNode CRUD does not return `opc-work-request-id`, and
  `ListWorkRequests` only documents the `ADD_NODES` operation type. The
  published runtime is therefore workrequest-aware but not workrequest-driven:
  it records shared request breadcrumbs when available and relies on lifecycle
  projection plus confirm-delete rereads instead of attempting CRUD
  work-request correlation.
- Required status projection keeps the OCI body surface except excluded
  credential-like fields and one-time URLs/codes `dataValidationCode`,
  `imageExportPar`, `returnShippingLabelUri`, `superUserPassword`, and
  `unlockPassphrase`. The row keeps `secret_side_effects = none`.

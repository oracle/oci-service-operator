---
schemaVersion: 1
surface: repo-authored-semantics
service: mediaservices
slug: mediaworkflow
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `mediaservices/MediaWorkflow` row
after the runtime review replaced the provisional generated scaffold semantics
with the published synchronous controller-backed contract.

## Current runtime contract

- `MediaWorkflow` keeps the generated controller, service-manager shell, and
  registration wiring, but the published runtime contract is owned by
  `pkg/servicemanager/mediaservices/mediaworkflow/mediaworkflow_runtime_client.go`
  instead of the generated baseline in
  `pkg/servicemanager/mediaservices/mediaworkflow/mediaworkflow_serviceclient.go`.
- The vendored SDK exposes direct
  `Create/Get/List/Update/DeleteMediaWorkflow` operations plus the auxiliary
  mutator `ChangeMediaWorkflowCompartment`. Create, get, and update return the
  `MediaWorkflow` body directly, while delete returns only headers. The
  resource exposes lifecycle states `ACTIVE`, `NEEDS_ATTENTION`, and
  `DELETED`, so the reviewed runtime treats create and update as synchronous
  read-after-write flows, settles `ACTIVE` as success, treats
  `NEEDS_ATTENTION` as terminal failure without requeue, and uses `DELETED` or
  OCI NotFound as delete-confirmation targets instead of inventing
  work-request or provisional lifecycle polling.
- Pre-create lookup is explicit. The runtime requires `spec.compartmentId`,
  skips reuse when `spec.displayName` is empty, scopes `ListMediaWorkflows` by
  compartment and exact display name when available, and adopts only a unique
  exact match on the reviewed identity surface `compartmentId + displayName`.
  Duplicate matches fail instead of binding arbitrarily.
- Mutation policy stays aligned with `UpdateMediaWorkflowDetails`. The
  published runtime reconciles `displayName`, `tasks`,
  `mediaWorkflowConfigurationIds`, `parameters`, `freeformTags`, and
  `definedTags` in place. `compartmentId` remains replacement-only because
  `ChangeMediaWorkflowCompartment` stays unpublished.
- Create-time locks remain explicit but non-mutable. Matching create-time locks
  are normalized out of post-create parity checks by ignoring OCI-populated
  `timeCreated` metadata, and any later lock drift remains create-only
  unsupported drift instead of implicitly calling add/remove lock APIs.
- Required status projection remains part of the repo-authored contract. The
  published status read model keeps identifiers, timestamps, lifecycle state,
  version, tasks, configuration IDs, parameters, locks, and tag maps mirrored
  from the observed OCI body. The row has no secret side effects.
- Delete is explicit and required. The controller retains the finalizer until
  `DeleteMediaWorkflow` succeeds and a follow-up `GetMediaWorkflow` or fallback
  `ListMediaWorkflows` reread confirms OCI either reports `DELETED` or no
  longer returns the resource.

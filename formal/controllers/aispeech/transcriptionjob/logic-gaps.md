---
schemaVersion: 1
surface: repo-authored-semantics
service: aispeech
slug: transcriptionjob
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `aispeech/TranscriptionJob` row after
the runtime audit replaced the scaffold placeholder with the reviewed
generated-runtime contract.

## Current runtime path

- `TranscriptionJob` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/aispeech/transcriptionjob/transcriptionjob_runtime_client.go`.
- The handwritten runtime config still delegates CRUD projection to
  `generatedruntime.ServiceClient`, but it adds a narrow lifecycle override for
  `FAILED`, `CANCELING`, and `CANCELED` plus a delete-confirmation loop that
  rereads `GetTranscriptionJob` until OCI returns not found.
- Required status projection remains part of the published contract. The
  runtime projects the live OCI `TranscriptionJob` body back into the CR status
  surface, including
  `status.modelDetails.transcriptionSettings.additionalSettings` when OCI
  returns it, preserves the tracked OCID in `status.status.ocid`, and treats
  `SUCCEEDED` as the only steady success lifecycle.

## Repo-authored semantics

- Pre-create lookup is explicit. When no OCI identifier is already tracked, the
  generated runtime issues `ListTranscriptionJobs` with `compartmentId`,
  `displayName`, and optional tracked `id`; the returned collection is then
  rebound only through the exact ID or display-name matches already narrowed by
  that request shape.
- Lifecycle handling is explicit: `ACCEPTED` and `IN_PROGRESS` keep reconcile
  in provisioning, `SUCCEEDED` settles success, `FAILED` is terminal without
  requeue, `CANCELING` is a terminating breadcrumb during normal observe, and
  `CANCELED` is a canceled failure unless delete confirmation is already in
  flight.
- Delete is repo-authored behavior layered on top of the generated shell. The
  runtime does not treat `CANCELED` as completed deletion; it waits through
  `CANCELING` or `CANCELED`, reissues `DeleteTranscriptionJob` when OCI allows
  it, rereads `GetTranscriptionJob`, and releases the finalizer only after the
  job is absent.
- Mutation policy is explicit: only `displayName`, `description`,
  `freeformTags`, and `definedTags` reconcile in place. `compartmentId`
  remains replacement-only drift;
  `modelDetails.transcriptionSettings.additionalSettings` is create-only
  because `CreateTranscriptionJob` and `GetTranscriptionJob` expose it while
  `UpdateTranscriptionJobDetails` does not; and the runtime never calls the
  SDK's auxiliary `CancelTranscriptionJob` operation as a published mutator.

## Import boundary

- The pinned `terraform-provider-oci` revision in `formal/sources.lock` does
  not register an `aispeech.TranscriptionJob` resource or datasource. The
  checked-in import for this row is therefore repo-curated from the vendored
  OCI Speech SDK request and lifecycle shape rather than refreshed by
  `formal-import`, while the runtime semantics above remain the authoritative
  contract for promotion.

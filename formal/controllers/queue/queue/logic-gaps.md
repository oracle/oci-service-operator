---
schemaVersion: 1
surface: repo-authored-semantics
service: queue
slug: queue
gaps: []
---

# Logic Gaps

## Current runtime intent

- `Queue` intentionally retains a handwritten core runtime as the active manager path. The generated `newQueueServiceClient` scaffold still serves as the checked-in generatedruntime baseline for lifecycle, delete, and mutation metadata, but the live reconcile path still bypasses `generatedruntime.ServiceClient`: the shared core has no narrow seam for persisting and resuming Queue work requests, recovering the created Queue OCID from work-request resources, and preserving Queue's explicit zero and empty-string update intent without re-implementing most of the queue-local state machine anyway. Create, update, and delete persist work-request IDs in status and resume through `GetWorkRequest` until the Queue reaches a terminal observed state.
- Queue work-request polling now treats the shared async tracker as canonical. Queue SDK `OperationStatus*` values normalize into OSOK async classes, Queue `ActionType*` values normalize into create/update/delete phases, and the controller projects raw work-request status, operation type, message, percent complete, and work request ID into `status.async.current` before deriving conditions.
- Create-time identity recovery is work-request-backed. The runtime resolves the created Queue OCID from work-request resources and does not rely on display-name-only list matching as the primary binding mechanism.
- Update is field-aware and repo-authored. The runtime reads the live Queue by tracked OCID, rejects create-only drift for `compartmentId` and `retentionInSeconds`, allows mutable `UpdateQueueDetails` fields, and preserves the explicit empty-string custom-encryption-key clear path.
- Status projection is part of the checked-in contract. The runtime projects live OCI Queue fields into the published `QueueStatus` surface, uses `status.async.current` as the canonical in-flight operation tracker, and keeps `createWorkRequestId`, `updateWorkRequestId`, and `deleteWorkRequestId` stable as compatibility mirrors across requeues.
- Delete confirmation is required, not best-effort. The runtime does not report success until `GetQueue` confirms the tracked Queue is gone or exposes lifecycle state `DELETED`, even when work-request reads become unavailable.
- Secret side effects are explicit repo-authored behavior. A package-local companion writes or updates the same-name endpoint Secret only after ACTIVE when `messagesEndpoint` is present, adopts only unlabeled matching Secrets, uses guarded updates/deletes for owned Secrets, and skips missing or unowned Secrets during delete cleanup.

## Authority and scoped cleanup

- `formal/controllers/queue/queue/*` is the authoritative formal path for the
  promoted Queue runtime contract.
- `pkg/servicemanager/queue/queue/queue_serviceclient.go` still records the
  generated helper-hook baseline, but the live execution contract is owned by
  `pkg/servicemanager/queue/queue/queue_runtime.go` plus the endpoint Secret
  companion.
- The disabled `service: workrequests` row in
  `internal/generator/config/services.yaml` is a separate generator-contract
  decision. Queue's work-request-backed async path does not publish or enable a
  standalone `workrequests` API group.

## Out of scope follow-on work

- `Channel`, `Message`, `Stats`, and `WorkRequest*` remain scaffold-only
  Queue-family surfaces outside this first Queue rollout. Their future
  prune-or-promote decision is separate from the promoted `Queue` contract and
  is not implied by Queue's async adapter.

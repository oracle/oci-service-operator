---
schemaVersion: 1
surface: repo-authored-semantics
service: queue
slug: queue
gaps: []
---

# Logic Gaps

## Current runtime intent

- `Queue` intentionally retains a handwritten core runtime as the active manager path. The generated `newQueueServiceClient` scaffold is not the live reconcile path because the generic `generatedruntime` flow still cannot preserve Queue's work-request-backed identity recovery and explicit empty-string and zero-value update intent without a broader seam. Create, update, and delete persist work-request IDs in status and resume through `GetWorkRequest` until the Queue reaches a terminal observed state.
- Create-time identity recovery is work-request-backed. The runtime resolves the created Queue OCID from work-request resources and does not rely on display-name-only list matching as the primary binding mechanism.
- Update is field-aware and repo-authored. The runtime reads the live Queue by tracked OCID, rejects create-only drift for `compartmentId` and `retentionInSeconds`, allows mutable `UpdateQueueDetails` fields, and preserves the explicit empty-string custom-encryption-key clear path.
- Status projection is part of the checked-in contract. The runtime projects live OCI Queue fields into the published `QueueStatus` surface and keeps `createWorkRequestId`, `updateWorkRequestId`, and `deleteWorkRequestId` stable across requeues.
- Delete confirmation is required, not best-effort. The runtime does not report success until `GetQueue` confirms the tracked Queue is gone or exposes lifecycle state `DELETED`, even when work-request reads become unavailable.
- Secret side effects are explicit repo-authored behavior. A package-local companion writes or updates the same-name endpoint Secret only after ACTIVE when `messagesEndpoint` is present, adopts only unlabeled matching Secrets, uses guarded updates/deletes for owned Secrets, and skips missing or unowned Secrets during delete cleanup.

## Out of scope follow-on work

- `Channel`, `Message`, `Stats`, and `WorkRequest*` remain scaffolded Queue resources outside this first Queue rollout. Their formal rows and runtime semantics are follow-on work, not implied by the promoted `Queue` contract here.

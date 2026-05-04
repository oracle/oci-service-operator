---
schemaVersion: 1
surface: repo-authored-semantics
service: adm
slug: knowledgebase
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `adm/KnowledgeBase` row after the
runtime review replaced the scaffold semantics with the published
work-request-backed contract.

## Current runtime path

- `KnowledgeBase` keeps the generated controller, service-manager shell, and
  registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/adm/knowledgebase/knowledgebase_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/adm/knowledgebase/knowledgebase_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes ADM
  `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `WAITING`,
  `CANCELING`, `SUCCEEDED`, `FAILED`, and `CANCELED`) into shared async
  classes, maps `CREATE_KNOWLEDGE_BASE`, `UPDATE_KNOWLEDGE_BASE`, and
  `DELETE_KNOWLEDGE_BASE` into create/update/delete phases, and resumes
  reconciliation from that shared async tracker across requeues.
- Create-time identity recovery is work-request-backed. The runtime records the
  tracked KnowledgeBase OCID when ADM exposes it and otherwise resolves the
  created resource OCID from work-request resources before reading the
  KnowledgeBase by ID and projecting status.
- Bind resolution is bounded. When no OCI identifier is tracked, the runtime
  only attempts pre-create reuse when `spec.displayName` is non-empty and then
  adopts only a unique `ListKnowledgeBases` match on exact `compartmentId`
  plus `displayName`. Summaries in `FAILED`, `DELETING`, or `DELETED` are not
  reused, and duplicate exact-name matches fail instead of binding
  arbitrarily.
- Mutable drift is limited to `displayName`, `freeformTags`, and
  `definedTags`. The handwritten update-body builder preserves clear-to-empty
  intent for both tag maps, while `compartmentId` remains replacement-only
  drift. The provider-only `ChangeKnowledgeBaseCompartment` auxiliary operation
  stays out-of-scope for the published runtime.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and
  the published `status.id`, `status.displayName`, `status.timeCreated`,
  `status.timeUpdated`, `status.lifecycleState`, `status.compartmentId`,
  `status.freeformTags`, `status.definedTags`, and `status.systemTags`
  read-model fields when ADM returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetKnowledgeBase` or fallback
  `ListKnowledgeBases` confirms the KnowledgeBase is gone or reports
  lifecycle state `DELETED`.

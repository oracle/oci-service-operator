---
schemaVersion: 1
surface: repo-authored-semantics
service: generativeaiagent
slug: knowledgebase
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `generativeaiagent/KnowledgeBase`
row after the runtime review replaced the provisional generated scaffold
semantics with the published work-request-backed contract.

## Current runtime path

- `KnowledgeBase` keeps the generated controller, service-manager shell, and
  registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/generativeaiagent/knowledgebase/knowledgebase_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/generativeaiagent/knowledgebase/knowledgebase_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes
  Generative AI Agent `OperationStatus*` values (`ACCEPTED`, `IN_PROGRESS`,
  `WAITING`, `NEEDS_ATTENTION`, `SUCCEEDED`, `FAILED`, `CANCELING`, and
  `CANCELED`) into shared async classes, maps
  `CREATE_KNOWLEDGE_BASE`, `UPDATE_KNOWLEDGE_BASE`, and
  `DELETE_KNOWLEDGE_BASE` into create/update/delete phases, and resumes
  reconciliation from that shared async tracker across requeues.
- Create-time identity recovery is work-request-backed. The runtime prefers the
  create response body when OCI returns a `KnowledgeBase` identifier and
  otherwise resolves the created resource OCID from work-request resources
  before reading `GetKnowledgeBase` by ID and projecting status.
- Bind resolution is bounded. When no OCI identifier is tracked, the runtime
  only attempts pre-create reuse when `spec.displayName` is non-empty and then
  adopts only a unique `ListKnowledgeBases` match on exact `compartmentId`
  plus `displayName` in reusable lifecycles (`ACTIVE`, `INACTIVE`,
  `CREATING`, or `UPDATING`). Summaries in `FAILED`, `DELETING`, or `DELETED`
  are not reused, and duplicate exact-name matches fail instead of binding
  arbitrarily.
- Mutable drift is limited to `displayName`, `description`, `indexConfig`,
  `freeformTags`, and `definedTags`. The handwritten update-body builder
  preserves clear-to-empty intent for `description` and tag maps, converts
  `indexConfig` into concrete SDK polymorphic bodies, and keeps
  `compartmentId` as replacement-only drift. The provider-only
  `ChangeKnowledgeBaseCompartment` auxiliary operation stays out-of-scope for
  the published runtime.
- Reviewed lifecycle mapping treats `CREATING` as provisioning, `UPDATING` as
  updating, `ACTIVE` and `INACTIVE` as success, `DELETING` as terminating,
  `DELETED` as the delete-confirmation target, and `FAILED` as terminal
  failure without requeue.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and
  the published read-model fields when OCI returns them, including `status.id`,
  `status.displayName`, `status.description`, `status.timeCreated`,
  `status.timeUpdated`, `status.lifecycleState`, `status.lifecycleDetails`,
  `status.compartmentId`, `status.indexConfig`,
  `status.knowledgeBaseStatistics`, `status.freeformTags`,
  `status.definedTags`, and `status.systemTags`.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetKnowledgeBase` or fallback
  `ListKnowledgeBases` confirms the resource is gone or exposes lifecycle
  state `DELETED`.

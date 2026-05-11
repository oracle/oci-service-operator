---
schemaVersion: 1
surface: repo-authored-semantics
service: generativeaiagent
slug: agent
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `generativeaiagent/Agent` row after
the runtime review replaced the scaffold semantics with the published
work-request-backed contract.

## Current runtime path

- `Agent` keeps the generated controller, service-manager shell, and
  registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/generativeaiagent/agent/agent_runtime_client.go` rather
  than the generated baseline in
  `pkg/servicemanager/generativeaiagent/agent/agent_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes Generative
  AI Agent `OperationStatus*` values into shared async classes, maps
  `CREATE_AGENT`, `UPDATE_AGENT`, and `DELETE_AGENT` into create/update/delete
  phases, and resumes reconciliation from that shared async tracker across
  requeues.
- Create-time identity recovery is work-request-backed. The runtime prefers the
  create response body when OCI returns an `Agent` identifier and otherwise
  resolves the created resource OCID from work-request resources before reading
  `GetAgent` by ID and projecting status.
- Bind resolution is bounded. When no OCI identifier is tracked, the runtime
  only attempts pre-create reuse when `spec.displayName` is non-empty and then
  adopts only a unique `ListAgents` match on exact `compartmentId` plus
  `displayName` in reusable lifecycles (`ACTIVE`, `CREATING`, or `UPDATING`).
  Summaries in `FAILED`, `DELETING`, or `DELETED` are not reused, and duplicate
  exact-name matches fail instead of binding arbitrarily.
- Mutable drift is limited to `displayName`, `description`, `welcomeMessage`,
  `knowledgeBaseIds`, `llmConfig`, `freeformTags`, and `definedTags`. The
  handwritten update-body builder preserves clear-to-empty intent for the
  string fields, `knowledgeBaseIds`, and tag maps, rebuilds nested
  `llmConfig.routingLlmCustomization.llmSelection` values into concrete SDK
  polymorphic bodies, and treats an empty `llmConfig` value as omission because
  the published spec cannot distinguish clear-from-empty from omission for that
  nested object. `compartmentId` stays replacement-only drift, and the
  provider-only `ChangeAgentCompartment` auxiliary operation stays out of scope
  for the published runtime.
- Reviewed lifecycle mapping treats `CREATING` as provisioning, `UPDATING` as
  updating, `ACTIVE` as success, `DELETING` as terminating, `DELETED` as the
  delete-confirmation target, and `FAILED` as terminal failure without
  requeue.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and the
  published read-model fields when OCI returns them, including `status.id`,
  `status.displayName`, `status.description`, `status.welcomeMessage`,
  `status.knowledgeBaseIds`, `status.llmConfig`, `status.timeCreated`,
  `status.timeUpdated`, `status.lifecycleState`, `status.lifecycleDetails`,
  `status.compartmentId`, `status.freeformTags`, `status.definedTags`, and
  `status.systemTags`.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetAgent` or fallback `ListAgents`
  confirms the resource is gone or exposes lifecycle state `DELETED`.

# Generative AI Agent Onboarding Audit

This audit is the `US-105` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/generativeaiagent` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `generativeaiagent` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/generativeaiagent` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/generativeaiagent` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `KnowledgeBase`

- Full CRUD family is present: `CreateKnowledgeBase`, `GetKnowledgeBase`,
  `ListKnowledgeBases`, `UpdateKnowledgeBase`, and `DeleteKnowledgeBase`.
- Additional mutator is present: `ChangeKnowledgeBaseCompartment`.
- `GetKnowledgeBaseResponse` returns `KnowledgeBase`.
- `ListKnowledgeBasesResponse` returns `KnowledgeBaseCollection`.
- `ListKnowledgeBasesRequest` exposes `compartmentId`, `lifecycleState`, and
  `displayName`, plus page and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `INACTIVE`,
  `DELETING`, `DELETED`, and `FAILED`.
- `CreateKnowledgeBaseResponse` returns the resource body and
  `OpcWorkRequestId`.
- `UpdateKnowledgeBaseResponse` and `DeleteKnowledgeBaseResponse` expose
  `OpcWorkRequestId`; update and delete do not return a `KnowledgeBase` body.
- The package also exposes service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, and `ListWorkRequestLogs`
  helpers.

### Auxiliary Families

- Additional full CRUD families are `Agent`, `AgentEndpoint`, `DataSource`,
  `ProvisionedCapacity`, and `Tool`.
- `DataIngestionJob` is create/get/list/delete only and ships with
  `GetDataIngestionJobLogContent` as an additional support surface.
- All of those non-`KnowledgeBase` families should stay unpublished initially.

## Generator Implications For `US-115`

- `KnowledgeBase` is the requested initial kind and it looks viable as the
  first published resource in the package.
- Recommended `formalSpec` is `knowledgebase`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `KnowledgeBase` looks viable as a direct controller-backed generated rollout
  because GET/list expose lifecycle state and the service ships the
  work-request helpers needed for mutation follow-up.
- The required risk callout is explicit here: `generativeaiagent/KnowledgeBase`
  must remain clearly distinct from the existing `adm/KnowledgeBase` rollout.
  `US-115` must keep service-qualified formal rows, docs, catalog entries, and
  generated package metadata disambiguated so the new group does not overwrite
  or confuse the existing ADM surface.

## Follow-through For `US-135`

- `Agent` is now the second published controller-backed kind in the
  `generativeaiagent` package; the remaining `AgentEndpoint`, `DataSource`,
  `DataIngestionJob`, `ProvisionedCapacity`, `Tool`, and work-request helper
  families remain unpublished.
- `Agent` uses the same recommended `formalSpec`/async posture as
  `KnowledgeBase`: repo-authored formal row plus `async.strategy=workrequest`
  with `workRequest.source=service-sdk` and `create`, `update`, and `delete`
  phases.
- `CreateAgent` returns an `Agent` body plus `OpcWorkRequestId`, while
  `UpdateAgent`, `DeleteAgent`, and `ChangeAgentCompartment` return
  work-request headers only. `GetAgent` and `ListAgents` project lifecycle
  directly, so the published runtime uses service-SDK `GetWorkRequest`
  follow-up and recovers the created OCID from either the create response body
  or work-request resources.
- The published runtime keeps `ChangeAgentCompartment` out of scope. The
  controller-backed contract treats `compartmentId` as replacement-only drift
  and confines in-place mutation to `displayName`, `description`,
  `welcomeMessage`, `knowledgeBaseIds`, `llmConfig`, `freeformTags`, and
  `definedTags`.
- `Agent` needs a handwritten create/update body builder even on top of the
  generated runtime shell because `llmConfig.routingLlmCustomization` and its
  nested `llmSelection` polymorphism need concrete SDK bodies, and the CRD's
  zero-value nested structs would otherwise serialize empty objects that the
  published runtime should omit.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are
  `oci_generative_ai_agent_knowledge_base` as the resource,
  `oci_generative_ai_agent_knowledge_base` as the singular data source, and
  `oci_generative_ai_agent_knowledge_bases` as the list data source.
- The follow-through `Agent` row uses
  `oci_generative_ai_agent_agent` as the resource and singular data source plus
  `oci_generative_ai_agent_agents` as the list data source. The checked-in
  mutability docs fixture is used for that row because the live Terraform
  Registry resource page currently renders as a JavaScript shell instead of a
  deterministic argument-reference HTML page.
- Provider docs publish the same knowledge-base family as both a resource and
  singular/list data sources, so the main risk is the repo-local naming
  collision with ADM rather than missing provider coverage.

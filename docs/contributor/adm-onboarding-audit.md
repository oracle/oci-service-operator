# ADM Onboarding Audit

This audit is the `US-84` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/adm` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `adm` package in the module cache; the repo
  lacked `vendor/github.com/oracle/oci-go-sdk/v65/adm` only because nothing
  imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/adm` so `go mod vendor` keeps the package
  in the branch-local inputs.

## SDK Audit

### `KnowledgeBase`

- Full CRUD family is present: `CreateKnowledgeBase`, `GetKnowledgeBase`,
  `ListKnowledgeBases`, `UpdateKnowledgeBase`, and `DeleteKnowledgeBase`.
- Additional mutator is present: `ChangeKnowledgeBaseCompartment`.
- `GetKnowledgeBaseResponse` returns `KnowledgeBase`.
- `ListKnowledgeBasesResponse` returns `KnowledgeBaseCollection`.
- `ListKnowledgeBasesRequest` exposes `id`, `compartmentId`, `displayName`,
  and `lifecycleState`, plus page and sort controls.
- Lifecycle states are `CREATING`, `ACTIVE`, `UPDATING`, `FAILED`,
  `DELETING`, and `DELETED`.
- `CreateKnowledgeBaseResponse`, `UpdateKnowledgeBaseResponse`, and
  `DeleteKnowledgeBaseResponse` all expose `OpcWorkRequestId`.
- The CRUD responses do not project a `KnowledgeBase` body, so the selected
  kind depends on the service-local work-request APIs to recover or confirm the
  resource after mutations.

### Auxiliary Families

- Additional SDK-discovered families are
  `ApplicationDependencyRecommendation`,
  `ApplicationDependencyVulnerability`, `RemediationRecipe`,
  `RemediationRun`, `Stage`, `Vulnerability`, `VulnerabilityAudit`,
  `WorkRequest`, `WorkRequestError`, and `WorkRequestLog`.
- `RemediationRecipe`, `RemediationRun`, and `VulnerabilityAudit` each carry a
  full CRUD surface.
- `Stage`, the application-dependency families, `Vulnerability`, and the
  work-request families are read, list, or workflow auxiliaries and should
  stay unpublished initially.

## Generator Implications For `US-86`

- `KnowledgeBase` is the cleanest first published kind because the broader ADM
  workflow families build on top of knowledge-base state and add more specific
  orchestration semantics.
- Recommended `formalSpec` is `knowledgebase`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `KnowledgeBase` still looks viable as a direct controller-backed generated
  rollout without handwritten runtime work because the service ships the full
  `GetWorkRequest`, `ListWorkRequests`, `ListWorkRequestErrors`, and
  `ListWorkRequestLogs` surface needed by the shared generated work-request
  seam.
- `RemediationRecipe`, `RemediationRun`, `Stage`, `VulnerabilityAudit`, and
  the work-request auxiliaries should stay unpublished initially while the
  first `KnowledgeBase` rollout lands.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_adm_knowledge_base` as both the resource
  and singular data source, plus `oci_adm_knowledge_bases` as the list data
  source.
- The provider `oci_adm_knowledge_base` resource waits on
  `GetWorkRequest` and `ListWorkRequestErrors` for create, update, delete, and
  compartment-change flows, which matches the recommended
  `async.strategy=workrequest` baseline.

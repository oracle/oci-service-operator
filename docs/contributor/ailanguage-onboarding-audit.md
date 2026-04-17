# AI Language Onboarding Audit

This audit is the `US-149` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/ailanguage` before `services.yaml` publishes
the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.61.1`.
- `v65.61.1` already contains the `ailanguage` package in the module cache; the
  repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/ailanguage` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/ailanguage` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `Project`

- Full CRUD family is present: `CreateProject`, `GetProject`, `ListProjects`,
  `UpdateProject`, and `DeleteProject`.
- Additional mutator is present: `ChangeProjectCompartment`.
- `GetProjectResponse` returns `Project`.
- `ListProjectsResponse` returns `[]ProjectSummary`.
- `ListProjectsRequest` exposes `compartmentId` (required), `lifecycleState`,
  `displayName`, and `projectId`, plus page and sort controls.
- Lifecycle states are: `ACTIVE`, `CREATING`, `DELETING`, `DELETED`, `FAILED`,
  and `UPDATING`.
- `CreateProjectResponse`, `UpdateProjectResponse`, and
  `DeleteProjectResponse` all expose `OpcWorkRequestId`.
- `CreateProjectDetails` contains `compartmentId`, `displayName`,
  `description`, `freeformTags`, and `definedTags`.
- `UpdateProjectDetails` contains `displayName`, `description`,
  `freeformTags`, and `definedTags`; `compartmentId` moves through
  `ChangeProjectCompartment`, so no obvious create-only field remains in the
  current SDK shape.

### Auxiliary Families

- Additional SDK-discovered families are `Endpoint`, `EvaluationResult`,
  `Model`, `ModelType`, `WorkRequest`, `WorkRequestError`, and
  `WorkRequestLog`.
- `Endpoint` and `Model` each carry their own CRUD surface; `EvaluationResult`,
  `ModelType`, and the work-request families are read or list auxiliaries.

## Generator Implications For `US-150`

- No `observedState.sdkAliases` requirement is apparent for `Project`; the GET
  response already projects `Project`.
- `Project` remains the narrowest audited common denominator for the first
  shared AI rollout, even though `Endpoint` and `Model` also exist in the
  local SDK surface.
- The list surface can match on `compartmentId` plus `displayName`;
  `projectId` is also available after OCI identity is known.
- Mutable fields are `displayName`, `description`, `freeformTags`, and
  `definedTags`, with compartment changes handled by the separate
  `ChangeProjectCompartment` action.
- `Endpoint`, `EvaluationResult`, `Model`, `ModelType`, and the work-request
  families should stay unpublished initially while the first `Project` rollout
  lands.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- That pinned revision registers `oci_ai_language_project` as both a resource
  and a singular data source, and `oci_ai_language_projects` as the list data
  source.
- The same provider service also registers `oci_ai_language_endpoint`,
  `oci_ai_language_job`, and `oci_ai_language_model` resource and data-source
  surfaces.
- The current local SDK discovery surface does not expose a CRUD `Job` family
  at `v65.61.1`, so later rollout work should not assume provider and SDK
  auxiliaries already match beyond the audited `Project` baseline.
- The pinned `oci_ai_language_project` resource waits on `GetWorkRequest` for
  create and delete, but its update path still uses `WaitForUpdatedState`
  lifecycle rereads even though `UpdateProjectResponse` returns
  `OpcWorkRequestId`.
- The pinned `oci_ai_language_projects` data source schema exposes an `id`
  input, but the current implementation does not pass that field through to
  `ListProjects`; only `compartment_id`, `display_name`, and `state` are
  actively wired today.

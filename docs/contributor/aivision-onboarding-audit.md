# AI Vision Onboarding Audit

This audit is the `US-149` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/aivision` before `services.yaml` publishes
the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.61.1`.
- `v65.61.1` already contains the `aivision` package in the module cache; the
  repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/aivision` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/aivision` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `Project`

- Full CRUD family is present: `CreateProject`, `GetProject`, `ListProjects`,
  `UpdateProject`, and `DeleteProject`.
- Additional mutator is present: `ChangeProjectCompartment`.
- `GetProjectResponse` returns `Project`.
- `ListProjectsResponse` returns `[]ProjectSummary`.
- `ListProjectsRequest` exposes `compartmentId`, `lifecycleState`,
  `displayName`, and `id`, plus page and sort controls.
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

- Additional SDK-discovered families are `DocumentJob`, `ImageJob`, `Model`,
  `WorkRequest`, `WorkRequestError`, and `WorkRequestLog`.
- `Model` also carries a CRUD surface; `DocumentJob` and `ImageJob` are
  create or get auxiliaries, and the work-request families are read or list
  support surfaces.

## Generator Implications For `US-150`

- No `observedState.sdkAliases` requirement is apparent for `Project`; the GET
  response already projects `Project`.
- `Project` is still the cleanest first published kind because the same
  branch-local inputs expose complete `Project` CRUD on both AI services.
- The list surface can match on `compartmentId` plus `displayName`; `id` is
  also available after OCI identity is known.
- Mutable fields are `displayName`, `description`, `freeformTags`, and
  `definedTags`, with compartment changes handled by the separate
  `ChangeProjectCompartment` action.
- `DocumentJob`, `ImageJob`, `Model`, and the work-request families should stay
  unpublished initially while the shared `Project` rollout lands.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- That pinned revision registers `oci_ai_vision_project` as both a resource and
  a singular data source, and `oci_ai_vision_projects` as the list data
  source.
- The pinned `oci_ai_vision_project` resource waits on `GetWorkRequest` for
  create, update, and delete, uses `ListWorkRequestErrors` for failure
  surfaces, and attempts `CancelWorkRequest` when create fails after a work
  request is issued.
- The pinned `oci_ai_vision_projects` data source wires `compartment_id`,
  `display_name`, `id`, and `state` directly through to `ListProjects`.
- The same provider service also registers `oci_ai_vision_model`,
  `oci_ai_vision_stream_group`, `oci_ai_vision_stream_job`,
  `oci_ai_vision_stream_source`, and
  `oci_ai_vision_vision_private_endpoint` resource and data-source surfaces.
- Those provider auxiliaries are broader than the current local SDK discovery
  surface at `v65.61.1`, so later rollout work should keep relying on the
  audited common `Project` baseline first.

# JMS Onboarding Audit

This audit is the `US-131` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/jms` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `jms` package in the module cache; the repo
  lacked `vendor/github.com/oracle/oci-go-sdk/v65/jms` only because nothing
  imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/jms` so `go mod vendor` keeps the package
  in the branch-local inputs.

## SDK Audit

### `Fleet`

- Full CRUD family is present:
  `CreateFleet`, `GetFleet`, `ListFleets`, `UpdateFleet`, and `DeleteFleet`.
- Additional mutator is present: `ChangeFleetCompartment`.
- `GetFleetResponse` returns `Fleet`.
- `ListFleetsResponse` returns `FleetCollection`.
- `ListFleetsRequest` exposes `compartmentId`, `id`, `lifecycleState`,
  `displayName`, `displayNameContains`, page, and sort controls.
- Lifecycle states are `ACTIVE`, `CREATING`, `DELETED`, `DELETING`, `FAILED`,
  `NEEDS_ATTENTION`, and `UPDATING`.
- `CreateFleetDetails` requires `displayName`, `compartmentId`, and
  `inventoryLog`; `description`, `operationLog`, deprecated
  `isAdvancedFeaturesEnabled`, and tags are optional.
- `CreateFleetResponse`, `UpdateFleetResponse`, `DeleteFleetResponse`, and
  `ChangeFleetCompartmentResponse` all expose `OpcWorkRequestId`, but only
  `GetFleetResponse` returns the `Fleet` body.
- The package exposes service-local `GetWorkRequest`, `ListWorkRequests`,
  `ListWorkRequestErrors`, and `ListWorkRequestLogs` helpers.
- `ListWorkRequests` includes `CREATE_FLEET`, `UPDATE_FLEET`, `DELETE_FLEET`,
  and `MOVE_FLEET` operation types, so base CRUD and compartment moves are all
  represented in the service-local async metadata.
- `WorkRequest.Resources` returns `entityType`, `actionType`, `identifier`,
  and `entityUri`, which is enough to recover the affected Fleet OCID before
  the first steady-state `GetFleet`.

### Auxiliary Families

- Additional fleet-scoped family surfaces are present for
  `FleetAdvancedFeatureConfiguration`, `FleetAgentConfiguration`, `DrsFile`,
  work requests, and a larger set of analysis, export, installation-site,
  error, diagnosis, and uncorrelated-package helpers.
- `FleetAdvancedFeatureConfiguration` and `FleetAgentConfiguration` are
  get/update sidecars rather than the base Fleet CRUD contract.
- Those auxiliary families should stay unpublished initially while the first
  `Fleet` rollout lands.

## Generator Implications For `US-132`

- `Fleet` is the intended first published kind in the `jms` package.
- Recommended `formalSpec` is `fleet`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `Fleet` looks viable as a controller-backed generated rollout because
  `GetFleet` and `ListFleets` project lifecycle state directly, the service
  ships the full work-request helper surface, and the work-request resources
  expose identifiers that can recover the created object before the first
  follow-up GET.
- The main rollout risk is create-time input policy: `inventoryLog` is
  mandatory on `CreateFleet` even though the update surface keeps it optional,
  so `US-132` must model that field as required from the first published spec.
- Another rollout risk is lifecycle interpretation: `NEEDS_ATTENTION` is a
  documented Fleet state and a provider steady target, so the runtime must not
  treat it as an automatic failure or endless requeue.
- `ChangeFleetCompartment`, advanced feature configuration, agent
  configuration, DRS helpers, and the broader analysis/read-only families
  should stay unpublished initially while the first `Fleet` controller-backed
  path is reviewed.
- The approximate inventory counters are best-effort service outputs rather
  than spec-driven fields, so they are the first status fields to revisit if
  the initial rollout sees noisy observed-state churn.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_jms_fleet` as the resource,
  `oci_jms_fleet` as the singular data source, and `oci_jms_fleets` as the
  list data source.
- The pinned provider resource also requires `inventory_log`, exposes
  `ChangeFleetCompartment`, and treats `ACTIVE` plus `NEEDS_ATTENTION` as
  steady target states.
- The provider service code uses `GetWorkRequest` to recover or reread the
  Fleet on create and update, which matches the recommended
  `async.strategy=workrequest` baseline for the published runtime.
- The provider's delete handling still falls back to lifecycle polling after
  `DeleteFleet`, so the repo rollout should keep its delete semantics explicit
  rather than assuming the provider already proves every work-request edge.

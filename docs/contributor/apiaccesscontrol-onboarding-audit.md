# API Access Control Onboarding Audit

This audit is the `US-95` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/apiaccesscontrol` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `apiaccesscontrol` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/apiaccesscontrol` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/apiaccesscontrol` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `PrivilegedApiControl`

- Full CRUD family is present:
  `CreatePrivilegedApiControl`, `GetPrivilegedApiControl`,
  `ListPrivilegedApiControls`, `UpdatePrivilegedApiControl`, and
  `DeletePrivilegedApiControl`.
- Additional mutator is present: `ChangePrivilegedApiControlCompartment`.
- `GetPrivilegedApiControlResponse` returns `PrivilegedApiControl`.
- `ListPrivilegedApiControlsResponse` returns
  `PrivilegedApiControlCollection` with `[]PrivilegedApiControlSummary`.
- `ListPrivilegedApiControlsRequest` exposes `compartmentId`, `id`,
  `lifecycleState`, `displayName`, and `resourceType`, plus page and sort
  controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, `FAILED`, and `NEEDS_ATTENTION`.
- `CreatePrivilegedApiControlResponse` returns the resource body and
  `OpcWorkRequestId`.
- `UpdatePrivilegedApiControlResponse` and
  `DeletePrivilegedApiControlResponse` expose `OpcWorkRequestId`; delete does
  not return a resource body.
- The package also exposes service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, and `ListWorkRequestLogs`
  helpers.

### Auxiliary Families

- Additional SDK-discovered families are `ApiMetadata`,
  `ApiMetadataByEntityType`, `PrivilegedApiRequest`, `WorkRequest`,
  `WorkRequestError`, and `WorkRequestLog`.
- `PrivilegedApiRequest` is create/get/list only and carries separate approval,
  revoke, reject, and close action APIs, so it should stay unpublished
  initially.
- `ApiMetadata` and `ApiMetadataByEntityType` are read-only support surfaces.

## Generator Implications For `US-96`

- `PrivilegedApiControl` is the requested initial kind and the only full CRUD
  family aligned with the follow-on story.
- Recommended `formalSpec` is `privilegedapicontrol`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `PrivilegedApiControl` looks viable as a direct controller-backed generated
  rollout because GET/list expose stable lifecycle state while the SDK also
  ships the work-request helpers needed to follow mutating operations.
- `US-96` should keep `PrivilegedApiRequest`, approval actions, and metadata
  auxiliaries unpublished initially and record any unsupported follow-on
  behavior explicitly in `logic-gaps.md`.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are
  `oci_apiaccesscontrol_privileged_api_control` as the resource,
  `oci_apiaccesscontrol_privileged_api_control` as the singular data source,
  and `oci_apiaccesscontrol_privileged_api_controls` as the list data source.
- The provider docs also publish lifecycle `state` plus explicit create,
  update, and delete timeouts, which matches the SDK's workrequest-backed
  mutation shape.

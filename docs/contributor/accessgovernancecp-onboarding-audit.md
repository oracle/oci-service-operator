# Access Governance Control Plane Onboarding Audit

This audit is the `US-84` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/accessgovernancecp` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `accessgovernancecp` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/accessgovernancecp` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/accessgovernancecp` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `GovernanceInstance`

- Full CRUD family is present:
  `CreateGovernanceInstance`, `GetGovernanceInstance`,
  `ListGovernanceInstances`, `UpdateGovernanceInstance`, and
  `DeleteGovernanceInstance`.
- Additional mutator is present: `ChangeGovernanceInstanceCompartment`.
- `GetGovernanceInstanceResponse` returns `GovernanceInstance`.
- `ListGovernanceInstancesResponse` returns `GovernanceInstanceCollection`.
- `ListGovernanceInstancesRequest` exposes required `compartmentId`, plus
  `displayName`, `id`, and `lifecycleState`, plus page and sort controls.
- Lifecycle states are `CREATING`, `ACTIVE`, `DELETING`, `DELETED`, and
  `NEEDS_ATTENTION`.
- `CreateGovernanceInstanceResponse` and `UpdateGovernanceInstanceResponse`
  both return `GovernanceInstance` and expose `OpcWorkRequestId`.
  `DeleteGovernanceInstanceResponse` also exposes `OpcWorkRequestId`.
- The package does not expose service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, or `ListWorkRequestLogs`
  helpers, so the selected kind should not rely on work-request metadata for
  the first generated rollout.

### Auxiliary Families

- Additional SDK-discovered families are `GovernanceInstanceConfiguration` and
  `SenderConfig`.
- `GovernanceInstanceConfiguration` exposes get and update only.
- `SenderConfig` is update-only and should stay unpublished initially.

## Generator Implications For `US-88`

- `GovernanceInstance` is the only full CRUD family in the package and the
  clear first published kind.
- Recommended `formalSpec` is `governanceinstance`.
- Recommended async classification is `lifecycle`.
- `GovernanceInstance` looks viable as a direct controller-backed generated
  rollout without handwritten runtime work because the GET and list surfaces
  project lifecycle states directly and the create response already returns the
  resource body and identity.
- `GovernanceInstanceConfiguration` and `SenderConfig` should stay unpublished
  initially while the first `GovernanceInstance` rollout lands.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching provider resource or data-source surfaces for
  `GovernanceInstance` in the accessible provider repo layout, so
  provider-facts coverage should be treated as absent or unconfirmed for the
  current pinned source.
- `US-88` should keep `formalSpec: governanceinstance` scaffold-only and avoid
  assuming provider-helper imports or imported provider state coverage until a
  provider-backed path is proven explicitly.

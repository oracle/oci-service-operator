# Budget Onboarding Audit

This audit is the `US-84` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/budget` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `budget` package in the module cache; the
  repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/budget` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/budget` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `Budget`

- Full CRUD family is present: `CreateBudget`, `GetBudget`, `ListBudgets`,
  `UpdateBudget`, and `DeleteBudget`.
- `GetBudgetResponse` returns `Budget`.
- `ListBudgetsResponse` returns `[]BudgetSummary`.
- `ListBudgetsRequest` exposes required `compartmentId`, plus
  `displayName`, `lifecycleState`, and `targetType`, plus page and sort
  controls.
- Lifecycle states are `ACTIVE` and `INACTIVE`.
- `CreateBudgetResponse` and `UpdateBudgetResponse` both return `Budget`.
  `DeleteBudgetResponse` is header-only, and none of the CRUD responses expose
  `OpcWorkRequestId`.

### Auxiliary Families

- Additional SDK-discovered families are `AlertRule`,
  `CostAlertSubscription`, `CostAnomalyEvent`, and `CostAnomalyMonitor`.
- `AlertRule`, `CostAlertSubscription`, and `CostAnomalyMonitor` each carry
  their own CRUD surface.
- `CostAnomalyEvent` is a narrower get/list/update auxiliary and should stay
  unpublished initially.

## Generator Implications For `US-85`

- `Budget` is the only clean first published kind and already matches the
  approved story sequence.
- Recommended `formalSpec` is `budget`.
- Recommended async classification is `none`; the resource exposes no
  service-local work-request APIs and no in-flight create/update/delete
  lifecycle states.
- `Budget` looks viable as a direct controller-backed generated rollout
  without handwritten runtime work. No `observedState.sdkAliases`
  requirement is apparent from the SDK shape alone.
- `AlertRule`, `CostAlertSubscription`, `CostAnomalyEvent`, and
  `CostAnomalyMonitor` should stay unpublished initially while the first
  `Budget` rollout lands.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_budget_budget` as both the resource and
  singular data source, plus `oci_budget_budgets` as the list data source.
- The provider budget resource uses direct CRUD or lifecycle rereads instead of
  service-local work-request helpers, which matches the recommended
  `async.strategy=none` baseline.

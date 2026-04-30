# APM Synthetics Onboarding Audit

This audit is the `US-95` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/apmsynthetics` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `apmsynthetics` package in the module
  cache; the repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/apmsynthetics`
  only because nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/apmsynthetics` so `go mod vendor` keeps
  the package in the branch-local inputs.

## SDK Audit

### `Script`

- Full CRUD family is present:
  `CreateScript`, `GetScript`, `ListScripts`, `UpdateScript`, and
  `DeleteScript`.
- `GetScriptResponse`, `CreateScriptResponse`, and `UpdateScriptResponse`
  return `Script`.
- `ListScriptsResponse` returns `ScriptCollection` with `[]ScriptSummary`.
- `ListScriptsRequest` exposes required `apmDomainId`, plus `displayName`,
  `contentType`, page, and sort controls.
- `Script` does not expose a lifecycle-state enum.
- The CRUD responses do not expose work-request IDs or service-local
  work-request helper APIs.
- `DeleteScriptResponse` does not return a resource body.

### Auxiliary Families

- Additional full CRUD families are `DedicatedVantagePoint`, `Monitor`,
  `OnPremiseVantagePoint`, and `Worker`.
- `PublicVantagePoint` is list-only and `MonitorResult` is get-only.
- `Script` is still the requested first kind and is the narrowest initial
  rollout because it avoids the wider monitor, vantage-point, and result
  orchestration surfaces.

## Generator Implications For `US-100`

- `Script` is the requested initial kind and a clean standalone synchronous
  CRUD surface.
- Recommended `formalSpec` is `script`.
- Recommended async classification is `none`.
- `Script` looks viable as a direct controller-backed generated rollout
  because create/get/list/update return stable bodies and no lifecycle or
  workrequest bridge is required.
- `US-100` should still treat script content and parameter handling carefully:
  the resource carries opaque script bodies plus secret-capable parameter
  markup, so the rollout must avoid noisy parity drift or accidental projection
  of secret-marked values.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_apm_synthetics_script` as the resource
  and `oci_apm_synthetics_scripts` as the list data source.
- I did not locate a separate singular data-source page in the accessible
  provider docs, so `US-100` should keep any singular import assumptions
  explicit if that gap persists at the pinned provider revision.

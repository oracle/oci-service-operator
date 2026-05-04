# APM Config Onboarding Audit

This audit is the `US-95` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/apmconfig` before `services.yaml` publishes
the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `apmconfig` package in the module cache;
  the repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/apmconfig` only
  because nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/apmconfig` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `Config`

- Full CRUD family is present:
  `CreateConfig`, `GetConfig`, `ListConfigs`, `UpdateConfig`, and
  `DeleteConfig`.
- `GetConfigResponse`, `CreateConfigResponse`, and `UpdateConfigResponse`
  return the polymorphic `Config` interface rather than a concrete struct.
- `ListConfigsResponse` returns `ConfigCollection`, which unmarshals
  `[]ConfigSummary` polymorphically.
- `ListConfigsRequest` exposes required `apmDomainId`, plus `configType`,
  `displayName`, `optionsGroup`, tag filters, page, and sort controls.
- The root `Config` surface does not expose lifecycle-state enums.
- The root `Config` surface does not expose work-request IDs or service-local
  work-request helper APIs.
- `CreateConfigDetails` and `UpdateConfigDetails` are also polymorphic
  interfaces.

### Auxiliary Families

- The polymorphic `Config` and `ConfigSummary` roots dispatch across
  `AGENT`, `OPTIONS`, `MACS_APM_EXTENSION`, `METRIC_GROUP`, `APDEX`, and
  `SPAN_FILTER`.
- `MatchAgentsWithAttributeKey` is a separate get/update-only family and
  should stay unpublished initially.

## Generator Implications For `US-97`

- `Config` is the requested initial kind and the only top-level CRUD family in
  the package.
- Recommended `formalSpec` is `config`.
- Recommended async classification is `none`.
- `Config` is only viable as a direct controller-backed rollout if `US-97`
  explicitly handles or narrows the polymorphic root across create, update,
  get, list, and observed-state projection.
- The required risk callout is explicit here: the root `Config` and
  `ConfigSummary` shapes are polymorphic interfaces, not plain structs, so the
  later story must own subtype restriction, request-body dispatch, and any
  observed-state aliasing needed to publish one OSOK kind safely.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_apm_config_config` as both the resource
  and singular data source, plus `oci_apm_config_configs` as the list data
  source.
- The provider docs model subtype-specific arguments and filters keyed by
  `config_type`, which reinforces the SDK-side polymorphic root risk rather
  than removing it.

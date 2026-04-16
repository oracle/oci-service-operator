# Analytics Onboarding Audit

This audit is the `US-136` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/analytics` before `services.yaml` publishes
the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.61.1`.
- `v65.61.1` already contains the `analytics` package in the module cache; the
  repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/analytics` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/analytics` so
  `go mod vendor` keeps the package in the branch-local inputs.

## SDK Audit

### `AnalyticsInstance`

- Full CRUD family is present:
  `CreateAnalyticsInstance`, `GetAnalyticsInstance`,
  `ListAnalyticsInstances`, `UpdateAnalyticsInstance`,
  `DeleteAnalyticsInstance`.
- Additional mutators are present:
  `StartAnalyticsInstance`, `StopAnalyticsInstance`,
  `ScaleAnalyticsInstance`, `ChangeAnalyticsInstanceCompartment`,
  `ChangeAnalyticsInstanceNetworkEndpoint`, and `SetKmsKey`.
- `GetAnalyticsInstanceResponse` returns `AnalyticsInstance`.
- `ListAnalyticsInstancesResponse` returns `[]AnalyticsInstanceSummary`.
- Lifecycle states are:
  `ACTIVE`, `CREATING`, `DELETED`, `DELETING`, `FAILED`, `INACTIVE`,
  and `UPDATING`.
- `CreateAnalyticsInstanceResponse` and `DeleteAnalyticsInstanceResponse`
  both expose `OpcWorkRequestId`.
- `UpdateAnalyticsInstanceResponse` does not expose `OpcWorkRequestId`.
- The package also exposes service-local work-request APIs:
  `GetWorkRequest`, `DeleteWorkRequest`, `ListWorkRequests`,
  `ListWorkRequestErrors`, and `ListWorkRequestLogs`.

### Auxiliary Families

- `PrivateAccessChannel` has
  `Create`, `Get`, `Update`, and `Delete` request or response families, plus
  work-request operation enums, but no list family.
- `VanityUrl` has `Create`, `Update`, and `Delete` request or response
  families, plus work-request operation enums, but no get or list family and
  no standalone `VanityUrl` model beyond `VanityUrlDetails`.
- `WorkRequest`, `WorkRequestError`, and `WorkRequestLog` are present as
  service-local auxiliary surfaces.

## Generator Implications For `US-137`

- `AnalyticsInstance` is the only family with a complete
  create/get/list/update/delete surface and should be the initial published
  kind.
- `PrivateAccessChannel`, `VanityUrl`, and the service-local work-request
  families should stay unpublished initially.
- No `observedState.sdkAliases` requirement is apparent for
  `AnalyticsInstance`; the GET response already projects `AnalyticsInstance`.
- No hard `observedState.excludedFieldPaths` requirement is apparent from the
  SDK shape alone. `AnalyticsInstance` does include polymorphic
  `NetworkEndpointDetails` plus `PrivateAccessChannels` and
  `VanityUrlDetails` maps, so those are the first status fields to revisit if
  `US-137` hits noisy or unsupported observed-state rendering.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- That pinned revision includes analytics data sources:
  `oci_analytics_analytics_instance`,
  `oci_analytics_analytics_instance_private_access_channel`,
  and `oci_analytics_analytics_instances`.
- That pinned revision includes analytics resources:
  `oci_analytics_analytics_instance`,
  `oci_analytics_analytics_instance_private_access_channel`,
  and `oci_analytics_analytics_instance_vanity_url`.
- The provider service code waits on `GetWorkRequest` for the analytics
  instance, private access channel, and vanity URL resource flows, so the
  pinned provider facts cover both the CRUD surface and explicit
  work-request-backed asynchronous behavior.

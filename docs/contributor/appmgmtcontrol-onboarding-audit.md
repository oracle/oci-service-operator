# Appmgmt Control Onboarding Audit

This audit is the `US-149` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/appmgmtcontrol` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/appmgmtcontrol`.
- `vendor/modules.txt` and
  `vendor/github.com/oracle/oci-go-sdk/v65/appmgmtcontrol/` now carry the
  branch-local SDK surface for this service.

## SDK Audit

### `MonitoredInstance`

- The package exposes `GetMonitoredInstance` and `ListMonitoredInstances`
  only. There is no create, update, or delete family for `MonitoredInstance`.
- `GetMonitoredInstanceRequest` is keyed by `monitoredInstanceId`.
- `GetMonitoredInstanceResponse` returns `MonitoredInstance`.
- `ListMonitoredInstancesRequest` requires `compartmentId` and supports
  `displayName`, paging, `sortBy`, and `sortOrder`.
- `ListMonitoredInstancesResponse` returns `MonitoredInstanceCollection`,
  whose items are `MonitoredInstanceSummary` rather than full
  `MonitoredInstance` bodies.
- `MonitoredInstance` carries `instanceId`, `compartmentId`, `displayName`,
  `managementAgentId`, `timeCreated`, `timeUpdated`, `monitoringState`,
  `lifecycleState`, and `lifecycleDetails`.
- The SDK comments explicitly note that `displayName` is bound to a Compute
  instance and fetched from the Core service API.
- Monitoring states are `ENABLED` and `DISABLED`.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `INACTIVE`,
  `DELETING`, `DELETED`, and `FAILED`.

### Auxiliary Families

- The package also exposes out-of-scope surfaces for
  `ActivateMonitoringPlugin`, `PublishTopProcessesMetrics`, and work-request
  inspection helpers. Those mutation or action flows do not make
  `MonitoredInstance` itself a normal controller-managed CRUD resource.

## Generator Implications For `US-152`

- `MonitoredInstance` is the planned first published kind for `US-152`.
- Recommended `formalSpec` is `monitoredinstance`.
- Recommended async classification is `none`.
- The main rollout risks are explicit here: there is no SDK mutation path, list
  returns summaries rather than full bodies, and the request identity
  (`monitoredInstanceId`) does not match the returned top-level field name
  (`instanceId`). `US-152` should publish this only as bind-existing or
  observe-only and make the identity mapping explicit instead of inventing
  create, update, delete, or fake async behavior.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching checked-in provider-fact imports or local repo
  evidence for `MonitoredInstance` in this checkout, so `US-152` should treat
  provider-backed imports as absent or unconfirmed until a pinned provider
  surface is proven directly.

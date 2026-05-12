# Compute Instance Agent Onboarding Audit

This audit is the `US-149` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/computeinstanceagent` before
`services.yaml` publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/computeinstanceagent`.
- `vendor/modules.txt` and
  `vendor/github.com/oracle/oci-go-sdk/v65/computeinstanceagent/` now carry
  the branch-local SDK surface for this service.

## SDK Audit

### `InstanceAgentPlugin`

- The package exposes `GetInstanceAgentPlugin` and `ListInstanceAgentPlugins`
  only. There is no create, update, or delete family for `InstanceAgentPlugin`.
- `GetInstanceAgentPluginRequest` requires `instanceagentId`,
  `compartmentId`, and `pluginName`.
- `GetInstanceAgentPluginResponse` returns `InstanceAgentPlugin`.
- `ListInstanceAgentPluginsRequest` requires `instanceagentId` and
  `compartmentId`, and supports `name`, `status`, paging, and sort controls.
- `ListInstanceAgentPluginsResponse` returns `[]InstanceAgentPluginSummary`
  rather than full `InstanceAgentPlugin` bodies.
- `InstanceAgentPlugin` carries `name`, `status`, `timeLastUpdatedUtc`, and an
  optional `message`.
- `InstanceAgentPluginSummary` carries `name`, `status`, and
  `timeLastUpdatedUtc`, but not the full `message` field from the singular get
  response.
- Status values are `RUNNING`, `STOPPED`, `NOT_SUPPORTED`, and `INVALID`.
- The SDK comments are explicit that plugin enable or disable is handled by
  Core `UpdateInstance`, not by a compute-instance-agent mutation API.

### Auxiliary Families

- The package also exposes out-of-scope surfaces for instance-agent commands,
  command executions, available-plugin discovery, and plugin-configuration
  clients. Those families are broader than the single plugin-read contract
  targeted by this initiative.

## Generator Implications For `US-153`

- `InstanceAgentPlugin` is the planned first published kind for `US-153`.
- Recommended `formalSpec` is `instanceagentplugin`.
- Recommended async classification is `none`.
- The main rollout risks are explicit here: there is no SDK mutation path, list
  returns summary items rather than the full get body, and the resource is
  keyed by the composite identity `instanceagentId + compartmentId +
  pluginName`. `US-153` should publish this only as bind-existing or observe-
  only and keep plugin enable or disable explicitly out of scope.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching checked-in provider-fact imports or local repo
  evidence for `InstanceAgentPlugin` in this checkout, so `US-153` should treat
  provider-backed imports as absent or unconfirmed until a pinned provider
  surface is proven directly.

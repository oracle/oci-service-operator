---
schemaVersion: 1
surface: repo-authored-semantics
service: computeinstanceagent
slug: instanceagentplugin
gaps: []
---

# Logic Gaps

## Current runtime path

- `InstanceAgentPlugin` keeps the generated controller, service-manager shell,
  and registration wiring, but the reviewed runtime contract is finalized in
  `pkg/servicemanager/computeinstanceagent/instanceagentplugin/instanceagentplugin_runtime_client.go`.
- The pinned SDK exposes `GetInstanceAgentPlugin` and
  `ListInstanceAgentPlugins` only. The published runtime is therefore
  bind-existing and observe-only: reconcile requires explicit
  `spec.instanceagentId`, `spec.compartmentId`, and `spec.pluginName`, records
  that composite identity into status, and uses `ListInstanceAgentPlugins`
  only as a confirmation fallback when `GetInstanceAgentPlugin` no longer
  returns the tracked plugin.
- No OCI create, update, or delete path is published. Plugin enable or disable
  remains part of Core `UpdateInstance` and is explicitly out of scope. Delete
  is CR-local unbind only.

## Repo-authored semantics

- Status projection is required and publishes the bound compute instance OCID,
  compartment OCID, plugin name, raw SDK status, last-updated timestamp, and
  optional plugin message with no secret side effects.
- Lifecycle classification is explicit for the plugin read surface. The visible
  plugin states `RUNNING`, `STOPPED`, `NOT_SUPPORTED`, and `INVALID` are all
  stable observed states rather than create, update, or delete progress
  signals.
- Mutation policy is bind-only. `instanceagentId`, `compartmentId`, and
  `pluginName` are replacement-only identity fields, and no in-place mutable
  OCI surface is published in this story.
- Delete confirmation is best-effort because the SDK exposes no delete call.
  Once the Kubernetes object is being removed, the runtime clears controller
  state locally and releases the finalizer without waiting on a cloud-side
  terminal state.

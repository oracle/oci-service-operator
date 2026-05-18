---
schemaVersion: 1
surface: repo-authored-semantics
service: zpr
slug: configuration
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `zpr/Configuration` row after the
reviewed singleton create/get contract replaced the unpublished placeholder
surface.

## Current runtime contract

- `Configuration` keeps the generated controller, service-manager shell, and
  registration wiring, but the reviewed runtime contract is finalized in
  `pkg/servicemanager/zpr/configuration/`.
- The pinned SDK exposes `CreateConfiguration`, `GetConfiguration`, and the
  configuration-specific work-request helper seam
  (`GetZprConfigurationWorkRequest`, `ListZprConfigurationWorkRequests`,
  logs, and errors), but it does not expose list, update, or delete
  operations for the singleton configuration resource.
- The published runtime is root-compartment singleton create/get only.
  Reconcile binds or rereads through `GetConfiguration(spec.compartmentId)`,
  creates only when that read confirms absence, and refuses to invent list
  reuse semantics for the service-scoped singleton.
- Create is work-request-backed. `CreateConfiguration` seeds
  `status.status.async.current.workRequestId`, reconcile resumes through
  `GetZprConfigurationWorkRequest`, derives the create phase from the ZPR
  work-request operation type, recovers the configuration identifier from
  work-request resources, and then rereads `GetConfiguration` as the
  authoritative observed-state source.
- No in-place OCI mutation surface is published. `compartmentId` remains the
  create/bind identity, and post-create drift on `zprStatus`,
  `freeformTags`, or `definedTags` is rejected instead of inventing update or
  replacement helpers the SDK does not expose.
- Delete is CR-local cleanup only. Deleting the Kubernetes resource clears the
  finalizer immediately and never calls nonexistent ZPR delete helpers.
- Status projection is required and publishes the bound configuration identity,
  root compartment, `zprStatus`, lifecycle fields, timestamps, and tags with
  no secret side effects.

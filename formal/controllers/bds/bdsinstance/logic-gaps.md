---
schemaVersion: 1
surface: repo-authored-semantics
service: bds
slug: bdsinstance
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `bds/BdsInstance` row after the
runtime audit replaced the scaffold placeholder with the reviewed
generated-runtime contract.

## Current runtime path

- `BdsInstance` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/bds/bdsinstance/bdsinstance_runtime_client.go`.
- The handwritten runtime config binds `CreateBdsInstance`, `GetBdsInstance`,
  `ListBdsInstances`, `UpdateBdsInstance`, and `DeleteBdsInstance` through
  `generatedruntime.ServiceClient` rather than a service-local legacy adapter.
- Lifecycle handling is explicit: `CREATING` requeues as provisioning,
  `UPDATING`, `SUSPENDING`, and `RESUMING` requeue as updating, `ACTIVE`,
  `INACTIVE`, and `SUSPENDED` settle success, `FAILED` is terminal without
  requeue, and delete confirmation waits through `DELETING` until `DELETED` or
  NotFound.
- Required status projection remains part of the repo-authored contract. The
  generated runtime stamps OSOK lifecycle conditions plus the published
  `status.id`, `status.compartmentId`, `status.displayName`,
  `status.lifecycleState`, `status.nodes`, `status.numberOfNodes`,
  `status.clusterVersion`, `status.networkConfig`, `status.clusterDetails`,
  `status.cloudSqlDetails`, `status.freeformTags`, `status.definedTags`,
  `status.kmsKeyId`, and `status.clusterProfile` read-model fields when OCI
  returns them.

## Repo-authored semantics

- Pre-create lookup is explicit. When no OCI identifier is already tracked, the
  generated runtime queries `ListBdsInstances` with the identifying request
  shape `compartmentId` plus `displayName`; the OCI list request also exposes
  `lifecycleState`, `limit`, `page`, `sortBy`, and `sortOrder`, but reusable
  matching remains a repo-authored decision layered on top of those provider
  facts.
- Mutation policy is explicit: only `displayName`, `bootstrapScriptUrl`,
  `freeformTags`, `definedTags`, and `kmsKeyId` reconcile in place. The
  handwritten update-body builder keeps clear-to-empty intent for bootstrap
  script URL, tag maps, and `kmsKeyId`.
- Replacement-only drift remains explicit for `compartmentId`,
  `clusterVersion`, `isHighAvailability`, `isSecure`, `clusterProfile`,
  `networkConfig`, and `nodes`; the handwritten drift check rejects those
  changes before OCI mutation rather than silently widening the supported
  update surface.
- `clusterAdminPassword`, `clusterPublicKey`, and `kerberosRealmName` remain
  create-time inputs only. OCI does not project them back on `BdsInstance`, so
  post-create reconciles do not attempt drift detection or reapplication for
  those values.
- Provider auxiliary mutators `ChangeBdsInstanceCompartment`,
  `ExecuteBootstrapScript`, `InstallOsPatch`, `RemoveKafka`, `RemoveNode`,
  `StartBdsInstance`, and `StopBdsInstance` remain out-of-scope drift for the
  published runtime surface. The reviewed generated-runtime client records
  lifecycle breadcrumbs from plain create, update, and delete calls instead of
  driving those auxiliary operations or provider-owned work-request waiters.
- Kubernetes secret reads and writes are out of scope for `BdsInstance`; the
  row keeps `secret_side_effects = none`.

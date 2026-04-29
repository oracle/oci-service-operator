---
schemaVersion: 1
surface: repo-authored-semantics
service: analytics
slug: analyticsinstance
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `analytics/AnalyticsInstance` row
after the runtime audit replaced the provisional generated scaffold semantics
with a reviewed runtime contract.

## Current runtime path

- `AnalyticsInstance` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/analytics/analyticsinstance/analyticsinstance_runtime_client.go`.
- The vendored SDK exposes
  `Create/Get/List/Update/DeleteAnalyticsInstance` plus lifecycle states
  `CREATING`, `ACTIVE`, `UPDATING`, `DELETING`, `FAILED`, `INACTIVE`, and
  `DELETED`. The reviewed runtime keeps plain `generatedruntime` lifecycle
  handling: `CREATING` requeues as provisioning, `UPDATING` requeues as
  updating, `ACTIVE` and `INACTIVE` settle success, `FAILED` is terminal
  without requeue, and delete confirmation waits through `DELETING` until
  `DELETED` or NotFound.
- Pre-create lookup is explicit: `ListAnalyticsInstances` searches by exact
  `compartmentId`, `name`, `featureSet`, and `capacityType`, then reuses only a
  single candidate in reusable lifecycle states (`ACTIVE`, `CREATING`,
  `UPDATING`, or `INACTIVE`). Duplicate exact-name matches fail instead of
  guessing.
- Mutation policy is explicit: only `UpdateAnalyticsInstanceDetails` fields
  `description`, `emailNotification`, `licenseType`, `freeformTags`,
  `definedTags`, and `updateChannel` reconcile in place. The handwritten
  update-body builder keeps clear-to-empty intent for description, email, and
  tag maps rather than dropping empty values, and sends `updateChannel` only
  when the desired non-empty enum differs from the live value.
- Provider auxiliary mutators `ChangeAnalyticsInstanceCompartment`,
  `ChangeAnalyticsInstanceNetworkEndpoint`, `ScaleAnalyticsInstance`,
  `SetFeatureBundle`, and `SetKmsKey` remain out-of-scope drift for the
  published runtime surface. Replacement-only drift remains explicit for
  `capacity.capacityType`, `featureSet`, `name`, and `networkEndpointType`.
- `IdcsAccessToken` remains a create-time input only. OCI does not project it
  back on `AnalyticsInstance`, so post-create reconciles do not attempt drift
  detection or reapplication for that value.
- `AdminUser` follows the same create-time-only pattern as
  `IdcsAccessToken`: OCI accepts it on create but does not project it back, so
  post-create reconciles normalize it out of parity checks instead of treating
  it as unsupported update drift.
- `DomainId` is a create-time input that OCI does echo back on
  `AnalyticsInstance`, so the runtime keeps it in required status projection
  while still refusing to mutate it in place.
- `FeatureBundle` is now part of required status projection. OCI exposes a
  separate `SetFeatureBundle` auxiliary operation, but the published runtime
  does not call it and therefore treats `featureBundle` changes as explicit
  unsupported drift rather than an implicit in-place update.
- Create and delete responses expose `opc-work-request-id`, while update does
  not. The reviewed runtime records request and work-request breadcrumbs when
  OCI returns them, but it does not publish service-local work-request kinds or
  poll them directly; lifecycle projection and confirm-delete rereads remain the
  authoritative async signal.
- `NetworkEndpointDetails`, `PrivateAccessChannels`, and `VanityUrlDetails`
  stay inside required status projection for the published kind. The refreshed
  v65.110.0 fields `updateChannel`, `featureBundle`, `domainId`, and
  `systemTags` also stay inside required status projection, and the row keeps
  `secret_side_effects = none`.

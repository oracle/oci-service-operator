---
schemaVersion: 1
surface: repo-authored-semantics
service: analytics
slug: analyticsinstance
gaps: []
---

# Logic Gaps

## Current rollout boundary

- `AnalyticsInstance` is published as an API-first `crd-only` surface in
  `internal/generator/config/services.yaml`. Controller, service-manager, and
  registration rollout remain disabled here; this row seeds the formal catalog
  before the controller-backed promotion stories bind `formalSpec`.
- The initial public analytics surface intentionally excludes
  `PrivateAccessChannel`, `VanityUrl`, and the service-local `WorkRequest*`
  families. Those auxiliary families remain unpublished with explicit per-kind
  `strategy: none` overrides until a later story proves they belong in the
  supported surface.

## Seeded runtime intent

- The vendored SDK exposes a complete
  `Create/Get/List/Update/DeleteAnalyticsInstance` family plus stable lifecycle
  states `CREATING`, `ACTIVE`, `UPDATING`, `DELETING`, `FAILED`, `INACTIVE`,
  and `DELETED`.
- The repo records a provisional lifecycle async contract for the published
  kind so the selected API surface is truthful without forcing controller-backed
  rollout yet. Create and delete responses expose `OpcWorkRequestId`, but
  update does not, so later runtime stories must decide whether the emitted
  controller can stay on plain `generatedruntime` or needs a narrower
  work-request adapter before `formalSpec` is bound.
- Out-of-band mutators such as `StartAnalyticsInstance`,
  `StopAnalyticsInstance`, `ScaleAnalyticsInstance`,
  `ChangeAnalyticsInstanceCompartment`,
  `ChangeAnalyticsInstanceNetworkEndpoint`, and `SetKmsKey` are intentionally
  out of scope for this API-first onboarding. The row describes only the
  published top-level CRUD contract and the lifecycle states that later
  controller stories must reconcile honestly.

## Provider-fact boundary

- The pinned provider facts cover `oci_analytics_analytics_instance` and use
  service-local `GetWorkRequest` polling for analytics instance flows. This row
  keeps that provider behavior explicit in the imported facts without
  publishing the service-local work-request kinds themselves.
- `NetworkEndpointDetails`, `PrivateAccessChannels`, and `VanityUrlDetails`
  remain the first status fields to re-audit once the controller-backed
  scaffold exists, because this story publishes their API shapes but does not
  yet prove the final runtime projection policy for them.

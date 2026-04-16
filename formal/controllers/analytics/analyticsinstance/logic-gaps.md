---
schemaVersion: 1
surface: repo-authored-semantics
service: analytics
slug: analyticsinstance
gaps: []
---

# Logic Gaps

## Current rollout boundary

- `AnalyticsInstance` is published as a generated controller-backed scaffold in
  `internal/generator/config/services.yaml`. The generated controller,
  service-manager, and registration outputs land here as the baseline runtime
  seam that the follow-on runtime story will harden.
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
- The repo binds `formalSpec` with a provisional lifecycle async contract for
  the published kind so the generated controller-backed scaffold and formal row
  stay aligned. Create and delete responses expose `OpcWorkRequestId`, but
  update does not, so the follow-on runtime story must still decide whether the
  emitted controller can stay on plain `generatedruntime` or needs a narrower
  work-request adapter.
- Out-of-band mutators such as `StartAnalyticsInstance`,
  `StopAnalyticsInstance`, `ScaleAnalyticsInstance`,
  `ChangeAnalyticsInstanceCompartment`,
  `ChangeAnalyticsInstanceNetworkEndpoint`, and `SetKmsKey` are intentionally
  out of scope for this initial controller-backed scaffold. The row describes
  only the published top-level CRUD contract and the lifecycle states that
  later runtime stories must reconcile honestly.

## Provider-fact boundary

- The pinned provider facts cover `oci_analytics_analytics_instance` and use
  service-local `GetWorkRequest` polling for analytics instance flows. This row
  keeps that provider behavior explicit in the imported facts without
  publishing the service-local work-request kinds themselves.
- The published SDK-derived API surface does not expose provider-only fields
  such as `admin_user`, `domain_id`, `feature_bundle`, `state`, or
  `update_channel`, so those facts stay out of the checked-in mutation overlay
  inputs for this scaffolded rollout.
- `NetworkEndpointDetails`, `PrivateAccessChannels`, and `VanityUrlDetails`
  remain the first status fields to re-audit once the controller-backed
  scaffold exists, because this story publishes their API shapes but does not
  yet prove the final runtime projection policy for them.

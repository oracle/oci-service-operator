---
schemaVersion: 1
surface: repo-authored-semantics
service: generativeaiagentruntime
slug: session
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `generativeaiagentruntime/Session`
row after the runtime review replaced the provisional generated scaffold with
the published synchronous controller-backed contract.

## Current runtime contract

- `Session` keeps the generated controller, service-manager shell, and
  registration wiring, but the published runtime contract is owned by
  `pkg/servicemanager/generativeaiagentruntime/session/session_runtime_client.go`
  instead of the generated baseline in
  `pkg/servicemanager/generativeaiagentruntime/session/session_serviceclient.go`.
- The vendored SDK exposes direct `CreateSession`, `GetSession`,
  `UpdateSession`, and `DeleteSession` operations only. Create, get, and update
  return the `Session` body directly, delete returns only headers, and the
  service exposes no `ListSessions` or service-local work-request helper
  surface.
- Path identity is explicit and reviewed. Create requires
  `spec.agentEndpointId`. After OCI returns the session body, the runtime
  records `status.agentEndpointId` and `status.sessionId` alongside the shared
  tracked identifier so subsequent reread, update, and delete requests can
  address `GetSession`, `UpdateSession`, and `DeleteSession` by the required
  `agentEndpointId + sessionId` path. The published rollout does not invent
  list-based bind or search semantics.
- Mutation policy stays aligned with `UpdateSessionDetails`. The published
  runtime reconciles `displayName` and `description` in place while treating
  `agentEndpointId` as replacement-only identity drift.
- Required status projection remains part of the repo-authored contract. The
  published status read model keeps `status.id`, `status.sessionId`,
  `status.agentEndpointId`, `status.displayName`, `status.description`,
  `status.welcomeMessage`, `status.timeCreated`, and `status.timeUpdated`
  mirrored from the confirmed OCI read path.
- Create and update are synchronous reread flows. The `Session` model exposes
  no lifecycle enum, so the reviewed runtime converts a confirmed
  read-after-write response into settled `Active` status instead of inventing
  provisioning or updating lifecycle buckets.
- Delete is explicit and required. The controller retains the finalizer until
  `DeleteSession` succeeds and a follow-up `GetSession` reread returns OCI
  NotFound.

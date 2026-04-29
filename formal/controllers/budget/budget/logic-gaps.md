---
schemaVersion: 1
surface: repo-authored-semantics
service: budget
slug: budget
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `budget/Budget` row after the runtime
review replaced the provisional scaffold semantics with the published generated
contract.

## Current runtime path

- `Budget` keeps the generated controller, service-manager shell, and
  registration wiring with no handwritten runtime override. The published
  behavior comes from `internal/generator/config/services.yaml` plus the
  repo-authored formal row under `formal/`.
- The vendored SDK exposes direct `Create/Get/List/Update/DeleteBudget`
  operations and only lifecycle states `ACTIVE` and `INACTIVE`. The reviewed
  runtime treats both as steady success states, publishes no provisioning or
  updating lifecycle buckets, and keeps `async.strategy=none`.
- Delete confirmation stays explicit but synchronous: the generated runtime
  retains the finalizer after `DeleteBudget` until confirm-delete rereads stop
  finding the resource. The row does not model service-local delete lifecycle
  states or work requests.
- Mutation policy remains the provider-backed conservative surface from the
  formal import: `displayName` reconciles in place and `compartmentId` stays
  replacement-only.
- Required status projection keeps the observed Budget body mirrored on the CR,
  including `lifecycleState`, `targets`, spend counters, and tag maps. The row
  has no secret side effects.

---
schemaVersion: 1
surface: repo-authored-semantics
service: apiplatform
slug: apiplatforminstance
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `apiplatform/ApiPlatformInstance`
row after the runtime review narrowed the generated baseline into the published
contract with a package-local runtime hook override.

## Current runtime path

- `ApiPlatformInstance` keeps the generated controller, service-manager shell,
  and registration wiring, but the live published behavior is owned by
  `pkg/servicemanager/apiplatform/apiplatforminstance/apiplatforminstance_runtime_client.go`
  rather than the unmodified generated baseline.
- The vendored SDK exposes direct
  `Create/Get/List/Update/DeleteApiPlatformInstance` operations plus the
  auxiliary mutator `ChangeApiPlatformInstanceCompartment`. The reviewed
  runtime keeps lifecycle-backed generated handling: `CREATING` requeues as
  provisioning, `UPDATING` requeues as updating, `ACTIVE` settles success,
  `FAILED` is terminal without requeue, and delete confirmation waits through
  `DELETING` until OCI reports `DELETED` or NotFound.
- Pre-create reuse is bounded and explicit. Existing-before-create lookup uses
  exact `compartmentId` plus `name` matching. The package-local runtime hook
  intentionally omits `lifecycleState` from the list request and list-match
  contract so first bind-or-create does not over-constrain reusable matches.
  Only reusable lifecycle states (`ACTIVE`, `CREATING`, or `UPDATING`) can
  bind, and duplicate exact matches fail instead of guessing.
- Mutable drift stays on the direct update body surface only:
  `description`, `freeformTags`, and `definedTags` reconcile in place. The
  package-local runtime hook clears auxiliary operations from the published
  semantics, so `ChangeApiPlatformInstanceCompartment` stays out of scope and
  `compartmentId` remains replacement-only drift. `name` is also create-only.
- Create and delete responses expose `opc-work-request-id`, while update
  returns only the resource body. The generated runtime records async
  breadcrumbs when OCI returns them, but lifecycle projection plus
  confirm-delete rereads remain the authoritative async signal; no
  service-local work-request readers are published.
- Required status projection remains part of the checked-in contract. The
  runtime mirrors the observed `ApiPlatformInstance` body, including
  `id`, `name`, `compartmentId`, `description`, `idcsApp`, `uris`,
  `lifecycleState`, `lifecycleDetails`, tag maps, and timestamps, and the row
  keeps `secret_side_effects = none`.

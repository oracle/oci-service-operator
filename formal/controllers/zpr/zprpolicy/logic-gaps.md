---
schemaVersion: 1
surface: repo-authored-semantics
service: zpr
slug: zprpolicy
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `zpr/ZprPolicy` row after the
scaffold placeholder was replaced with the reviewed work-request-backed
runtime contract.

## Current runtime contract

- `ZprPolicy` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/zpr/zprpolicy/zprpolicy_runtime.go`.
- Bind resolution uses an explicit OCI identifier first. When no policy OCID is
  tracked yet, the runtime may adopt only a unique `ListZprPolicies` result
  that matches `spec.compartmentId` plus the required exact `spec.name`.
- Mutation policy is explicit: only `description`, `statements`,
  `freeformTags`, and `definedTags` reconcile in place. `compartmentId` and
  `name` remain replacement-only drift once a live policy is bound.
- Create, update, and delete are work-request-backed via the ZPR service SDK.
  The runtime stores the in-flight work request in `status.async.current`,
  resumes through `GetZprPolicyWorkRequest`, derives create/update/delete phase
  from ZPR work-request operation types, and recovers the tracked policy OCID
  from work-request resources instead of assuming a broader helper seam.
- Lifecycle handling is explicit around the shared async tracker. `ACTIVE`
  settles success, `CREATING` and `UPDATING` keep reconciliation pending,
  `DELETING` keeps the finalizer in place until confirm-delete succeeds, and
  `FAILED` or `NEEDS_ATTENTION` remain terminal attention/failure outcomes.
- Status projection is part of the checked-in contract. The runtime projects the
  live policy identity, `name`, `description`, `compartmentId`, `statements`,
  lifecycle fields, timestamps, and tag maps from OCI onto the published
  `ZprPolicyStatus` surface.

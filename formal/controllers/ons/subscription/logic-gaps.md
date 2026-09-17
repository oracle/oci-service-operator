---
schemaVersion: 1
surface: repo-authored-semantics
service: ons
slug: subscription
gaps: []
---

# Logic Gaps

## Reviewed runtime contract

- `PENDING` requeues and `ACTIVE` succeeds. Delivery policy and tags update in place; topic, protocol, endpoint, compartment, and metadata require replacement.
- Delete confirms an auth-shaped Get 404 with an empty topic-scoped list before releasing the finalizer.

## Evidence

- The checked-in ORACLE_FUNCTIONS recording proves immediate active create, update-body response handling, full readback, delete, and scoped absence.
- The vendored data-plane SDK, pinned provider, and dynamic mock agree on list response shape and lifecycle behavior.

No open formal gaps remain for this runtime surface.

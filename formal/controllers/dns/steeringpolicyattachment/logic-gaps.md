---
schemaVersion: 1
surface: repo-authored-semantics
service: dns
slug: steeringpolicyattachment
gaps: []
---

# Logic Gaps

## Reviewed runtime contract

- `CREATING` requeues and `ACTIVE` succeeds. Only display name updates in place; policy, zone, and domain identify the attachment.
- Delete resolves the parent steering-policy compartment and requires an empty policy/zone/domain-scoped list before accepting an auth-shaped Get 404.

## Evidence

- The checked-in recording proves create, display-name update, parent lookup, scoped list, and delete confirmation.
- The vendored SDK, pinned provider, and dynamic parent-aware mock agree on the request paths and response shapes.

No open formal gaps remain for this runtime surface.

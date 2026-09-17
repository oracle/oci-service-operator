---
schemaVersion: 1
surface: repo-authored-semantics
service: dns
slug: tsigkey
gaps: []
---

# Logic Gaps

## Reviewed runtime contract

- `CREATING` and `UPDATING` requeue, `ACTIVE` succeeds, and `FAILED` is terminal. Only tag fields update in place.
- The write-only secret is create-only and never enters status. Delete confirms auth-shaped 404 through an empty compartment/name/id-scoped list.

## Evidence

- The checked-in recording proves create, tag update, prolonged deletion, terminal 404, and scoped list confirmation without retaining secret material.
- The vendored SDK, pinned provider, and dynamic mock agree on lifecycle and mapping.

No open formal gaps remain for this runtime surface.

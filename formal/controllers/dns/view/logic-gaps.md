---
schemaVersion: 1
surface: repo-authored-semantics
service: dns
slug: view
gaps: []
---

# Logic Gaps

## Reviewed runtime contract

- Create can settle directly on `ACTIVE`; `UPDATING` requeues. Display name and tags update in place while compartment identity requires replacement.
- Delete retains the finalizer through `DELETING` until `DELETED` or confirmed absence.

## Evidence

- The checked-in recording proves create, update, `DELETING`, and final 404 behavior.
- The vendored SDK, pinned provider, and dynamic mock agree on the recorded resource path.

No open formal gaps remain for this runtime surface.

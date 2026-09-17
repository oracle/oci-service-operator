---
schemaVersion: 1
surface: repo-authored-semantics
service: artifacts
slug: containerrepository
gaps: []
---

# Logic Gaps

## Reviewed runtime contract

- The handwritten extension owns exact-name binding, mutable visibility/immutability/README/tag updates, and conservative delete confirmation.
- `AVAILABLE` is steady. `DELETING` retains the finalizer until `DELETED`, `REPO_ID_UNKNOWN`, or scoped absence; auth-shaped 404 alone is insufficient.

## Evidence

- The checked-in recording proves create, read, public/README/tag update, delete, and service-specific terminal 404 behavior.
- The vendored SDK and pinned provider agree on request and response shapes. The dynamic HTTP mock exercises the production SDK and service manager.

No open formal gaps remain for this runtime surface.

---
schemaVersion: 1
surface: repo-authored-semantics
service: ons
slug: topic
gaps: []
---

# Logic Gaps

## Reviewed runtime contract

- Topic uses shared OSOK status, exact compartment/name binding, mutable description/tags, and required delete confirmation.
- `CREATING` requeues, `ACTIVE` succeeds, and `DELETING` retains the finalizer until absence.

## Evidence

- The checked-in recording proves create, read, update, the long `DELETING` interval, and final 404.
- The vendored SDK, pinned provider, and dynamic mock agree on the current request surface.

No open formal gaps remain for this runtime surface.

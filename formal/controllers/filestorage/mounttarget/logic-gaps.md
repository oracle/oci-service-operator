---
schemaVersion: 1
surface: repo-authored-semantics
service: filestorage
slug: mounttarget
gaps: []
---

# Logic Gaps

## Reviewed runtime contract

- `CREATING` and `UPDATING` requeue, `ACTIVE` succeeds, `FAILED` is terminal, and delete requires `DELETED` or absence.
- Display name, tags, NSGs, and security attributes update in place; availability domain, compartment, subnet, hostname, and selected IP require replacement.

## Evidence

- The checked-in recording proves create, export-set projection, mutable update, and prolonged delete convergence.
- The vendored SDK, pinned provider, and dynamic mock agree on placement and status identity fields.

No open formal gaps remain for this runtime surface.

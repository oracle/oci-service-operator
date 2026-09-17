---
schemaVersion: 1
surface: repo-authored-semantics
service: dns
slug: zone
gaps: []
---

# Logic Gaps

## Reviewed runtime contract

- Zone binding preserves name, scope, and private view identity. `CREATING` and `UPDATING` requeue, `ACTIVE` succeeds, and `FAILED` is terminal.
- Transfer/DNSSEC/resolution and tag fields update in place; name, compartment, scope, view, and zone type require replacement.

## Evidence

- The checked-in private-zone recording proves scoped pre-create lookup, create, update convergence, and absence-confirmed delete.
- The vendored SDK, pinned provider, and dynamic mock agree on the `migrationSource=NONE` create discriminator and private-zone response.

No open formal gaps remain for this runtime surface.

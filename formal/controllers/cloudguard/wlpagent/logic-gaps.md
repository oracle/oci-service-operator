---
schemaVersion: 1
surface: repo-authored-semantics
service: cloudguard
slug: wlpagent
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `cloudguard/WlpAgent` row.

## Current runtime path

- The pinned provider importer cannot resolve this resource's older CRUD
  implementation automatically. The provider source, vendored SDK, and
  synthetic contract therefore form the explicit evidence boundary.
- OCI's WlpAgent response has no lifecycle field. Generatedruntime performs
  read-after-write, while the retained resource-local wrapper marks a readable
  resource Active without inventing cloud lifecycle states.
- Certificate signing request and tag maps reconcile in place. Compartment,
  agent version, and OS information remain replacement-only.
- Pre-create reuse is scoped to compartmentId and exact agentVersion. Delete
  retains the finalizer until a NotFound reread. There are no Secret side
  effects.

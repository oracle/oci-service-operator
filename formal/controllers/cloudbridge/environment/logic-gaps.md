---
schemaVersion: 1
surface: repo-authored-semantics
service: cloudbridge
slug: environment
gaps: []
---

# Logic Gaps

The row remains at scaffold stage until its provider-fact import is refreshed,
but the service-manager lifecycle is live-verified.

- Bind-or-create uses compartment, display name, and optional environment ID.
- Display name and tag fields update in place; compartment is replacement-only.
- Generatedruntime requeues `CREATING` and `UPDATING`, accepts `ACTIVE`, and
  retains the finalizer through `DELETING` until `DELETED` or confirmed absence.
- No Kubernetes Secret or external side effect is owned by this resource.

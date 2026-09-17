---
schemaVersion: 1
surface: repo-authored-semantics
service: cloudguard
slug: managedlist
gaps: []
---

# Logic Gaps

The row remains at scaffold stage until its provider-fact import is refreshed,
but the service-manager lifecycle is live-verified.

- Bind-or-create uses compartment, display name, and optional managed-list ID.
- Display name, description, list items, and tags update in place. Compartment,
  group, list type, and source-list identity are replacement-only.
- Generatedruntime owns lifecycle classification and confirmed delete; this
  resource does not create detector targets or any external side effect.

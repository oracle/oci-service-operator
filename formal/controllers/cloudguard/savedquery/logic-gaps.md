---
schemaVersion: 1
surface: repo-authored-semantics
service: cloudguard
slug: savedquery
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `cloudguard/SavedQuery` row.

## Current runtime path

- The pinned provider importer cannot resolve this provider resource's older
  CRUD implementation automatically. The reviewed provider source, vendored
  SDK, and reviewed OCI API behavior therefore form the explicit evidence boundary.
- Create may report `CREATING` before `ACTIVE`. Ordinary updates are direct
  body responses, while delete returns 202 and is confirmed through
  `DELETING`, `DELETED`, or a NotFound reread.
- Display name, query, description, and tag maps reconcile in place.
  Compartment movement uses a separate SDK action not published by this OSOK
  runtime, so compartmentId remains replacement-only.
- Pre-create reuse is restricted to a unique exact compartmentId plus
  displayName match. The runtime has no Secret side effects.

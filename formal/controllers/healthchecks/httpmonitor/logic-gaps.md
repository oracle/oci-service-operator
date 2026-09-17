---
schemaVersion: 1
surface: repo-authored-semantics
service: healthchecks
slug: httpmonitor
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `healthchecks/HttpMonitor` row.

## Current runtime path

- Create and update are synchronous read-after-write operations. The SDK model
  has no lifecycle field, so a readable response is the steady-state signal.
- All fields accepted by `UpdateHttpMonitorDetails` reconcile in place.
  Compartment movement uses a separate action that is not published by this
  OSOK runtime, so compartmentId remains replacement-only.
- Pre-create reuse lists every page in the requested compartment and uses an
  exact displayName, protocol, and tracked-ID identity boundary.
- Health Checks returns auth-shaped 404 responses for both absent and hidden
  resources. The retained resource-local delete guard confirms scoped list
  absence before releasing the finalizer. There are no Secret side effects.

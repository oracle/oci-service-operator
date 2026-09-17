---
schemaVersion: 1
surface: repo-authored-semantics
service: dns
slug: steeringpolicy
gaps: []
---

# Logic Gaps

## Reviewed runtime contract

- `CREATING` and the live-observed `UPDATING` value requeue, `ACTIVE` succeeds, and `DELETING` requires terminal confirmation.
- Display name, TTL, health monitor, template, tags, answers, and polymorphic rules update in place; compartment identity requires replacement.

## Evidence

- The checked-in recording proves full custom-policy creation, replace-on-update answers/rules, `UPDATING` convergence, and scoped delete confirmation.
- The dynamic mock deliberately covers the `UPDATING` value missing from the pinned SDK enum constants.

No open formal gaps remain for this runtime surface.

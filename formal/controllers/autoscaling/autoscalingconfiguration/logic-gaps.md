---
schemaVersion: 1
surface: repo-authored-semantics
service: autoscaling
slug: autoscalingconfiguration
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded
`autoscaling/AutoScalingConfiguration` row.

## Current runtime path

- The OCI SDK exposes direct synchronous create, get, list, update, and delete
  operations. Its resource response has neither a lifecycle-state field nor a
  work-request handle, so generatedruntime performs read-after-write for create
  and update and confirms deletion with a final NotFound read.
- In-place reconciliation is limited to fields accepted by
  `UpdateAutoScalingConfigurationDetails`: cooldown, display name, enabled
  state, and tag maps. Every non-noop update carries the desired enabled state
  because the live API rejects otherwise valid partial updates that omit it.
  The instance-pool resource and policy definitions are create-time
  configuration and remain replacement-only.
- The Terraform provider exposes compartment movement through a distinct
  `ChangeAutoScalingConfigurationCompartment` action. The published OSOK
  runtime does not implement that auxiliary action, so compartment drift is
  replacement-only rather than being silently treated as an ordinary update.
- Pre-create reuse is restricted to an exact unique compartmentId plus
  displayName match. The runtime has no Secret side effects.

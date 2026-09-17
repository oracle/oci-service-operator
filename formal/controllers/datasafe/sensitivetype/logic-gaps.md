---
schemaVersion: 1
surface: repo-authored-semantics
service: datasafe
slug: sensitivetype
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `datasafe/SensitiveType` row.

## Current runtime path

- OCI create and update responses carry work-request IDs, but the published
  OSOK path does not poll Data Safe work requests. Create binds the resource ID
  from its response body; subsequent reconciles use `GetSensitiveType` and map
  `CREATING`/`UPDATING` to pending and `ACTIVE` to success. Work-request IDs are
  retained only as observable breadcrumbs.
- The resource-local builders preserve the SDK's polymorphic
  `SENSITIVE_TYPE` and `SENSITIVE_CATEGORY` request models.
- Display name, short name, description, parent category, pattern fields,
  search type, default masking format, and tags reconcile in place.
  Compartment and entity subtype remain replacement-only.
- Pre-create reuse requires a unique exact compartmentId, entityType,
  displayName, and shortName match. Delete is confirmed by `DELETING`,
  `DELETED`, or NotFound. There are no Secret side effects.

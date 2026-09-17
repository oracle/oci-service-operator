---
schemaVersion: 1
surface: repo-authored-semantics
service: usageapi
slug: usagecarbonemissionsquery
gaps: []
---

# Logic Gaps

## Current runtime path

- `UsageCarbonEmissionsQuery` retains the generated controller and operation bindings, while its package-local synchronous wrapper normalizes successful create and update readback into settled `Active` status.
- Existing resources are resolved through `ListUsageCarbonEmissionsQueries` in the requested compartment and matched against the populated `queryDefinition` fields.
- Delete uses the tracked OCI identity when present and retains the finalizer until `GetUsageCarbonEmissionsQuery` confirms absence.

## Repo-authored semantics

- The SDK resource has no lifecycle field. Create and update settle through read-after-write confirmation instead of OCI lifecycle polling.
- Only `queryDefinition` updates in place; `compartmentId` remains replacement-only.
- Status projection is required, and no Kubernetes Secret side effects are part of this path.

## Why this row is seeded

- The recorded create/read/update/delete trace, current SDK model, package-local wrapper, and dynamic service-manager integration scenario now agree on the state-free contract.
- No open formal gaps remain for the current generatedruntime behavior.

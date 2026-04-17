---
schemaVersion: 1
surface: repo-authored-semantics
service: usageapi
slug: customtable
gaps: []
---

# Logic Gaps

## Current runtime path

- `CustomTable` keeps the generated `CustomTableServiceManager` scaffold and generated runtime client configuration, but the checked-in `pkg/servicemanager/usageapi/customtable/customtable_runtime_client.go` wrapper normalizes synchronous create and update success for this resource-local path.
- The generated runtime reuses an existing custom table before create by calling `ListCustomTables` with the required `compartmentId` and `savedReportId`, then matching the returned `savedCustomTable` payload on display name plus any populated row, column, tag, depth, or version fields from the desired spec.
- Delete confirmation resolves tracked OCI identities through `GetCustomTable`; when no `status.status.ocid` is recorded, the runtime falls back to the same list-lookup shape before issuing `DeleteCustomTable`.

## Repo-authored semantics

- The SDK surface does not expose lifecycle states on `CustomTable` bodies, so create and update settle through read-after-write `GetCustomTable` confirmation instead of lifecycle polling. The service-local wrapper converts that confirmed no-lifecycle reread into settled `Active` status instead of leaving the controller in retry-only provisioning or updating fallback.
- Supported in-place updates are limited to `savedCustomTable`, matching the `UpdateCustomTableDetails` body. Drift for `compartmentId` and `savedReportId` is replacement-only and is rejected before OCI mutation.
- Status projection remains required. The generated runtime merges the live OCI `CustomTable` body into the published status read-model fields while still stamping `status.status.ocid` and shared OSOK request breadcrumbs.
- Delete is explicit and required. The controller retains the finalizer until `DeleteCustomTable` succeeds and a follow-up `GetCustomTable` confirms OCI no longer returns the custom table.
- No repo-authored secret reads or writes are part of this path.

## Why this row is seeded

- The checked-in generated controller and service-manager path now has an explicit bind-or-create answer, an in-place update boundary, and required delete confirmation for `usageapi/CustomTable`.
- No open formal gaps remain for the current generatedruntime contract.

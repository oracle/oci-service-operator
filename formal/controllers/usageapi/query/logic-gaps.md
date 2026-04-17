---
schemaVersion: 1
surface: repo-authored-semantics
service: usageapi
slug: query
gaps: []
---

# Logic Gaps

## Current runtime path

- `Query` keeps the generated `QueryServiceManager` scaffold and generated runtime client configuration, but the checked-in `pkg/servicemanager/usageapi/query/query_runtime_client.go` wrapper normalizes synchronous create and update success for this resource-local path.
- The generated runtime reuses an existing query before create by calling `ListQueries` with the required `compartmentId` request scope, then matching the returned `queryDefinition` payload on display name, version, report query inputs, and any populated cost analysis UI fields from the desired spec.
- Delete confirmation resolves tracked OCI identities through `GetQuery`; when no `status.status.ocid` is recorded, the runtime falls back to the same list-lookup shape before issuing `DeleteQuery`.

## Repo-authored semantics

- The SDK surface does not expose lifecycle states on `Query` bodies, so create and update settle through read-after-write `GetQuery` confirmation instead of lifecycle polling. The service-local wrapper converts that confirmed no-lifecycle reread into settled `Active` status instead of leaving the controller in retry-only provisioning or updating fallback.
- Supported in-place updates are limited to `queryDefinition`, matching the `UpdateQueryDetails` body. Drift for `compartmentId` is replacement-only and is rejected before OCI mutation.
- Status projection remains required. The generated runtime merges the live OCI `Query` body into the published status read-model fields while still stamping `status.status.ocid` and shared OSOK request breadcrumbs.
- Delete is explicit and required. The controller retains the finalizer until `DeleteQuery` succeeds and a follow-up `GetQuery` confirms OCI no longer returns the saved query.
- No repo-authored secret reads or writes are part of this path.

## Why this row is seeded

- The checked-in generated controller and service-manager path now has an explicit bind-or-create answer, an in-place update boundary, and required delete confirmation for `usageapi/Query`.
- No open formal gaps remain for the current generatedruntime contract.

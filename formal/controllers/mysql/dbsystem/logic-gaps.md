---
schemaVersion: 1
surface: repo-authored-semantics
service: mysql
slug: dbsystem
gaps:
  - category: bind-versus-create
    status: open
    stopCondition: "Formal semantics can resolve an existing DbSystem through ListDbSystems before create, using the shared generated-runtime identity decision instead of blindly creating when no OCI ID is tracked."
  - category: list-lookup
    status: open
    stopCondition: "Formal semantics encode the current ListDbSystems filters (`compartmentId`, `configurationId`, `databaseManagement`, `dbSystemId`, `displayName`, and lifecycle state) used for bind, update, and delete confirmation."
  - category: mutation-policy
    status: open
    stopCondition: "Formal semantics classify the mysql DbSystem create and update surface into create-only, mutable, or rejected-on-update behavior before runtime promotion."
  - category: waiter-work-request
    status: open
    stopCondition: "DbSystem create, update, and delete follow-up semantics have one shared formal answer for the provider CRUD helpers and generated read-after-write/delete-confirmation flow."
---

# Logic Gaps

## Current runtime path

- `DbSystem` now uses the generated `DbSystemServiceManager` and
  `generatedruntime.ServiceClient`; there is no checked-in legacy adapter
  override.
- The current generated path creates whenever no OCI ID is tracked, then relies
  on `GetDbSystem` or `ListDbSystems` only for read-after-write and delete
  confirmation.
- Imported provider facts cover `CreateDbSystem`, `GetDbSystem`,
  `ListDbSystems`, `UpdateDbSystem`, and `DeleteDbSystem`.

## Shared generated-runtime baseline

- Use the [shared generated-runtime baseline](../../../shared/generated-runtime-baseline.md)
  as the category map for bind, lookup, waiter, mutation, status, secret, and
  delete decisions.
- Follow `identity/User` for required status projection, required delete
  confirmation, and `retain-until-confirmed-delete`.
- Use `streaming/Stream` only as the legacy reference for pre-create lookup and
  lifecycle-sensitive matching. Do not inherit its secret or best-effort delete
  behavior.

## Repo-authored semantics

- `DbSystem` has no Kubernetes secret reads or secret writes in the generated
  runtime path.
- Delete should keep the finalizer until `GetDbSystem` or `ListDbSystems`
  confirms the DB system is gone.
- The generated runtime now follows OCI create, update, and delete requests
  with read-based status projection and delete confirmation, but it still needs
  repo-authored closure for pre-create reuse and field-by-field drift policy.

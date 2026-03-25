---
schemaVersion: 1
surface: repo-authored-semantics
service: nosql
slug: table
gaps:
  - category: bind-versus-create
    status: open
    stopCondition: "Formal semantics can resolve an existing Table through ListTables before create, using the shared generated-runtime identity decision instead of blindly creating when no OCI ID is tracked."
  - category: list-lookup
    status: open
    stopCondition: "Formal semantics encode the current ListTables filters (`compartmentId`, `name`, lifecycle state) and the lifecycle-sensitive matching used for bind, update, and delete."
  - category: mutation-policy
    status: open
    stopCondition: "Formal semantics classify `name`, `compartmentId`, `ddlStatement`, `tableLimits`, `isAutoReclaimable`, and tags into create-only, mutable, or rejected-on-update behavior before runtime promotion."
  - category: waiter-work-request
    status: open
    stopCondition: "Table create and update waits have one shared formal answer for provider work-request-backed completion instead of remaining implicit in provider CRUD helpers."
---

# Logic Gaps

## Current runtime path

- `Table` already uses the generated `TableServiceManager` and
  `generatedruntime.ServiceClient`; there is no checked-in legacy adapter
  override.
- The current generated path creates whenever no OCI ID is tracked, then relies
  on `GetTable` or `ListTables` only for read-after-write and delete
  confirmation.
- Imported provider facts now cover `CreateTable`, `GetTable`, `ListTables`,
  `ChangeTableCompartment`, `UpdateTable`, and `DeleteTable`, but the
  bind-or-create decision and waiter semantics still need repo-authored
  closure.

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

- `Table` has no Kubernetes secret reads or secret writes.
- Delete should keep the finalizer until `GetTable` or `ListTables` confirms
  the table is gone.
- The provider update path combines `ChangeTableCompartment` with
  work-request-backed `UpdateTable` calls for DDL, tag, and table-limit
  changes, so the seeded baseline keeps mutation and waiter closure explicit.

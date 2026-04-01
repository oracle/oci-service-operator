---
schemaVersion: 1
surface: repo-authored-semantics
service: mysql
slug: dbsystem
gaps: []
---

# Logic Gaps

## Current runtime path

- `DbSystem` now uses the generated `DbSystemServiceManager` and
  `generatedruntime.ServiceClient`; there is no checked-in legacy adapter
  override.
- The generated request path resolves `spec.adminUsername` and
  `spec.adminPassword` from same-namespace Kubernetes secrets before OCI
  create/update calls, then mirrors those secret references into status.
- When no OCI ID is tracked, the current generated path calls
  `ListDbSystems` before create and only reuses entries that match an
  identifying reusable field such as `displayName` while OCI remains in
  reusable lifecycle states (`ACTIVE`, `CREATING`, or `UPDATING`).
- Deleting, deleted, failed, or non-identifying list matches still fall
  through to `CreateDbSystem`, and the generated path continues to rely on
  `GetDbSystem` or `ListDbSystems` for read-after-write and delete
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

- `DbSystem` reads admin credential secret references from same-namespace
  Kubernetes secrets in the generated runtime path, but it does not create or
  update Kubernetes secrets as a side effect.
- `DbSystem` omits admin credential inputs entirely when the secret references
  are unset or empty, instead of projecting empty-string OCI payload values.
- `DbSystem` records only non-empty last applied admin credential secret
  references in status so force-new checks compare references instead of
  plaintext values.
- The shared generated runtime uses `ListDbSystems` before create only when no
  OCI ID is already tracked, reuses only identifying matches in reusable
  lifecycle states, and otherwise still calls `CreateDbSystem`.
- The generated runtime rejects force-new and otherwise unsupported update drift,
  and only calls `UpdateDbSystem` when the imported mutable surface differs from
  observed state.
- Delete should keep the finalizer until `GetDbSystem` or `ListDbSystems`
  confirms the DB system is gone.
- The generated runtime follows OCI create, update, and delete requests with
  shared read-based status projection and delete confirmation semantics.

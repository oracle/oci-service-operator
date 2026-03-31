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
    stopCondition: "Formal semantics encode the current ListDbSystems filters (`compartmentId`, `displayName`, `dbSystemId`, lifecycle state) and the lifecycle-sensitive matching used for bind, update, and delete."
  - category: mutation-policy
    status: open
    stopCondition: "Formal semantics classify the generated DbSystem mutable and force-new fields into create-only, mutable, or rejected-on-update behavior before runtime promotion."
  - category: waiter-work-request
    status: open
    stopCondition: "DbSystem create, update, and delete waits have one shared formal answer for generated-runtime completion instead of remaining implicit in the current read-after-write follow-up."
---

# Logic Gaps

## Current runtime path

- `DbSystem` already uses the generated `DbSystemServiceManager` and
  `generatedruntime.ServiceClient`; there is no checked-in legacy adapter
  override.
- The current generated path creates whenever no OCI ID is tracked, then relies
  on `GetDbSystem` for read-after-write and delete confirmation.
- Imported provider facts now cover `CreateDbSystem`, `GetDbSystem`,
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

- `DbSystem` reads same-namespace Kubernetes Secrets for
  `spec.adminUsername.secret.secretName` and
  `spec.adminPassword.secret.secretName` before create or update request
  projection.
- The generated runtime mirrors the applied secret references into
  `status.adminUsername` and `status.adminPassword`, but it does not create,
  update, or delete Kubernetes Secret objects.
- Delete should keep the finalizer until `GetDbSystem` or `ListDbSystems`
  confirms the database system is gone.
- The imported mutation surface is broader than the current generated contract,
  so the seeded baseline keeps bind, lifecycle lookup, and mutation closure
  explicit before promotion.

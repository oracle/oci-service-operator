---
schemaVersion: 1
surface: repo-authored-semantics
service: psql
slug: dbsystem
gaps:
  - category: bind-versus-create
    status: open
    stopCondition: "Formal semantics can resolve an existing DbSystem through ListDbSystems before create, using the shared generated-runtime identity decision instead of blindly creating when no OCI ID is tracked."
  - category: list-lookup
    status: open
    stopCondition: "Formal semantics encode the current ListDbSystems filters (`compartmentId`, `displayName`, `id`, lifecycle state) and the lifecycle-sensitive matching used for bind, update, and delete."
  - category: mutation-policy
    status: open
    stopCondition: "Formal semantics classify `displayName`, `compartmentId`, `dbVersion`, `storageDetails`, `shape`, `networkDetails`, `credentials`, `source`, `configId`, and tags into create-only, mutable, or rejected-on-update behavior before runtime promotion."
  - category: waiter-work-request
    status: open
    stopCondition: "DbSystem create and update waits have one shared formal answer for provider-backed completion, including create-time work requests and update-time `tfresource.WaitForUpdatedState`."
---

# Logic Gaps

## Current runtime path

- `DbSystem` already uses the generated `DbSystemServiceManager` and
  `generatedruntime.ServiceClient`; there is no checked-in legacy adapter
  override.
- The current generated path creates whenever no OCI ID is tracked, then relies
  on `GetDbSystem` for read-after-write and delete confirmation.
- Imported provider facts now cover `CreateDbSystem`, `GetDbSystem`,
  `ListDbSystems`, `ChangeDbSystemCompartment`, `PatchDbSystem`,
  `ResetMasterUserPassword`, `UpdateDbSystem`, and `DeleteDbSystem`.

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

- `DbSystem` has no Kubernetes secret reads or writes; credential changes stay
  within OCI request payload fields.
- Create uses work-request-backed completion and update also exposes
  `tfresource.WaitForUpdatedState`, so the seeded baseline keeps shared waiter
  closure explicit.
- Delete should keep the finalizer until `GetDbSystem` or `ListDbSystems`
  confirms the database system is gone.

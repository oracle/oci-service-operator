---
schemaVersion: 1
surface: repo-authored-semantics
service: database
slug: autonomousdatabase
gaps:
  - category: bind-versus-create
    status: open
    stopCondition: "Formal semantics can resolve an existing AutonomousDatabase through ListAutonomousDatabases before create, using the shared generated-runtime identity decision instead of blindly creating when no OCI ID is tracked."
  - category: list-lookup
    status: open
    stopCondition: "Formal semantics encode the current ListAutonomousDatabases filters (`compartmentId`, `autonomousContainerDatabaseId`, `displayName`, lifecycle state) and the lifecycle-sensitive matching used for bind, update, and delete."
  - category: mutation-policy
    status: open
    stopCondition: "Formal semantics classify the generated AutonomousDatabase mutable and force-new fields into create-only, mutable, or rejected-on-update behavior before runtime promotion."
  - category: waiter-work-request
    status: open
    stopCondition: "AutonomousDatabase create, update, and delete waits have one shared formal answer for generated-runtime completion instead of remaining implicit in the current read-after-write follow-up."
---

# Logic Gaps

## Current runtime path

- `AutonomousDatabase` already uses the generated `AutonomousDatabaseServiceManager`
  and `generatedruntime.ServiceClient`; there is no checked-in legacy adapter
  override.
- The current generated path creates whenever no OCI ID is tracked, then relies
  on `GetAutonomousDatabase` for read-after-write and delete confirmation.
- Imported provider facts now cover `CreateAutonomousDatabase`,
  `GetAutonomousDatabase`, `ListAutonomousDatabases`,
  `ChangeAutonomousDatabaseCompartment`, `UpdateAutonomousDatabase`, and
  `DeleteAutonomousDatabase`.

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

- `AutonomousDatabase` reads same-namespace Kubernetes Secrets for
  `spec.adminPassword.secret.secretName` when the repo-authored v2 contract
  uses Kubernetes-backed admin credentials instead of OCI Vault `secretId`
  input.
- The generated runtime forwards the resolved password to OCI but does not
  create, update, or delete Kubernetes Secret objects.
- Delete should keep the finalizer until `GetAutonomousDatabase` or
  `ListAutonomousDatabases` confirms the database is gone.
- The imported mutation surface is broad, so the seeded baseline keeps bind,
  lifecycle lookup, and mutation closure explicit before promotion.

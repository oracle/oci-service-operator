# Shared Generated Runtime Baseline

This checklist seeds `nosql/Table`, `psql/DbSystem`, and `mysql/DbSystem` from
the same reference points:

- `formal/controllers/identity/user` remains the explicit backlog
  generated-runtime reference for required status projection, read-after-write,
  and finalizer retention until delete confirmation, even though
  blank/default-active runs no longer check the `User` controller and
  service-manager into the repo.
- `formal/controllers/streaming/stream` is the generated-runtime reference for
  bind-versus-create, lifecycle-sensitive list lookup, and delete-side lookup
  branching, while its ready-only secret companion still uses Stream UID
  ownership with one-time adoption of matching legacy unlabeled secrets,
  guarded full-snapshot revalidation before update or delete, and keeps
  best-effort delete semantics stream-specific.
- `mysql/DbSystem` now follows the generated-runtime path and uses the same
  category names as the shared checklist.

## Checklist

### Bind-versus-create

- Shared rule: if no OCI ID is tracked, decide whether the service can reuse an
  existing OCI resource before create. Do not copy stream-specific secret or
  best-effort delete behavior into generated-runtime rows.
- `nosql/Table`: use `ListTables` with `compartment_id`, `name`, and lifecycle
  state to decide whether create is required.
- `psql/DbSystem`: use `ListDbSystems` with `compartment_id`, `display_name`,
  `id`, and lifecycle state to decide whether create is required.
- `mysql/DbSystem`: use `ListDbSystems` with `compartmentId`,
  `configurationId`, `databaseManagement`, `dbSystemId`, `displayName`, and
  lifecycle state to decide whether create is required.
- Owner: shared generated-runtime prerequisite plus kind-specific filter
  closure for all three resources.

### List-lookup

- Shared rule: list filters and lifecycle matching are repo-authored semantics
  even when provider facts expose the datasource shape.
- `nosql/Table`: imported provider facts cover `oci_nosql_tables`.
- `psql/DbSystem`: the provider exposes `oci_psql_db_systems`; the seeded
  baseline carries that datasource shape explicitly even though the current
  formal importer does not auto-wire it.
- `mysql/DbSystem`: imported provider facts cover `oci_mysql_mysql_db_systems`.
- Owner: shared identity-resolution prerequisite plus kind-specific filter
  closure.

### Waiter handling

- Shared rule: provider-backed write completion must have one explicit formal
  answer before generated-runtime promotion.
- `nosql/Table`: provider CRUD uses work-request-backed completion for create,
  update, and delete.
- `psql/DbSystem`: create is work-request-backed and update also exposes
  `tfresource.WaitForUpdatedState`.
- `mysql/DbSystem`: the generated runtime currently uses read-after-write for
  create and update plus confirm-delete for delete, while the remaining
  `waiter-work-request` stop condition stays explicit in formal semantics.
- Owner: shared generated-runtime prerequisite for all three resources.

### Mutation policy

- Shared rule: imported mutable and force-new fields are the starting point,
  not the final OSOK contract.
- `nosql/Table`: downstream work must classify `name`, `compartmentId`,
  `ddlStatement`, `tableLimits`, `isAutoReclaimable`, and tags into
  create-only, mutable, or rejected drift.
- `psql/DbSystem`: downstream work must classify `displayName`,
  `compartmentId`, `dbVersion`, `storageDetails`, `shape`, `networkDetails`,
  `credentials`, `source`, `configId`, and tags.
- `mysql/DbSystem`: downstream work must classify the broader generated
  `DbSystem` surface, including `displayName`, `description`,
  `configurationId`, `backupPolicy`, `maintenance`, `deletionPolicy`,
  `secureConnections`, and tag fields.
- Owner: kind-specific closure for all three resources.

### Status projection

- Shared rule: preserve the `identity/User` backlog reference for required
  status projection plus OSOK lifecycle conditions even though blank/default-
  active runs no longer check in that runtime.
- `nosql/Table`, `psql/DbSystem`, and `mysql/DbSystem`: seed the
  generated-runtime baseline with required status projection.
- Owner: shared generated-runtime baseline for all three resources.

### Secret inputs or outputs

- Shared rule: make repo-authored secret behavior explicit instead of implying
  it from provider facts.
- `nosql/Table`: no repo-authored secret reads or writes.
- `psql/DbSystem`: no Kubernetes secret reads or writes; credential fields
  remain OCI payload inputs.
- `mysql/DbSystem`: generated-runtime request projection reads
  same-namespace Kubernetes secrets for `adminUsername` and `adminPassword`,
  omits those inputs entirely when the secret references are unset or empty,
  mirrors only non-empty secret references into status for force-new
  bookkeeping, and
  does not materialize endpoint or access secrets.
- Owner: no additional secret seam for `nosql/Table` or `psql/DbSystem`;
  mysql keeps explicit read-only secret semantics in the generated-runtime row.

### Delete semantics

- Shared rule: preserve the `identity/User` backlog reference for retaining the
  finalizer until `Get` or list fallback confirms deletion.
- `nosql/Table`, `psql/DbSystem`, and `mysql/DbSystem`: seed required delete
  confirmation plus `retain-until-confirmed-delete`.
- Owner: shared generated-runtime baseline for all three resources.

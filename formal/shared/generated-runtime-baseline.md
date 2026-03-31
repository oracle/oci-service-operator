# Shared Generated Runtime Baseline

This checklist seeds `nosql/Table`, `psql/DbSystem`, and
`mysql/MySqlDbSystem` from the same reference points:

- `formal/controllers/identity/user` is the generated-runtime precedent for
  required status projection, read-after-write, and finalizer retention until
  delete confirmation.
- `formal/controllers/streaming/stream` is the generated-runtime reference for
  bind-versus-create, lifecycle-sensitive list lookup, and delete-side lookup
  branching, while its ready-only secret companion and best-effort delete
  semantics stay stream-specific.
- `mysql/MySqlDbSystem` stays in the legacy-adapter batch in this issue; it
  uses the same category names as the shared checklist but does not inherit the
  generated-runtime answers yet.

## Checklist

### Bind-versus-create

- Shared rule: if no OCI ID is tracked, decide whether the service can reuse an
  existing OCI resource before create. Do not copy stream-specific secret or
  best-effort delete behavior into generated-runtime rows.
- `nosql/Table`: use `ListTables` with `compartment_id`, `name`, and lifecycle
  state to decide whether create is required.
- `psql/DbSystem`: use `ListDbSystems` with `compartment_id`, `display_name`,
  `id`, and lifecycle state to decide whether create is required.
- `mysql/MySqlDbSystem`: keep the legacy bind-by-display-name plus compartment
  lookup path explicit until the adapter blocker closes.
- Owner: shared generated-runtime prerequisite for `nosql/Table` and
  `psql/DbSystem`; legacy-only for `mysql/MySqlDbSystem`.

### List-lookup

- Shared rule: list filters and lifecycle matching are repo-authored semantics
  even when provider facts expose the datasource shape.
- `nosql/Table`: imported provider facts cover `oci_nosql_tables`.
- `psql/DbSystem`: the provider exposes `oci_psql_db_systems`; the seeded
  baseline carries that datasource shape explicitly even though the current
  formal importer does not auto-wire it.
- `mysql/MySqlDbSystem`: imported provider facts cover
  `oci_mysql_mysql_db_systems`.
- Owner: shared identity-resolution prerequisite plus kind-specific filter
  closure.

### Waiter handling

- Shared rule: provider-backed write completion must have one explicit formal
  answer before generated-runtime promotion.
- `nosql/Table`: provider CRUD uses work-request-backed completion for create,
  update, and delete.
- `psql/DbSystem`: create is work-request-backed and update also exposes
  `tfresource.WaitForUpdatedState`.
- `mysql/MySqlDbSystem`: keep the legacy adapter's behavior explicit; this
  issue does not settle mysql waiter behavior.
- Owner: shared generated-runtime prerequisite for `nosql/Table` and
  `psql/DbSystem`; legacy-only for `mysql/MySqlDbSystem`.

### Mutation policy

- Shared rule: imported mutable and force-new fields are the starting point,
  not the final OSOK contract.
- `nosql/Table`: downstream work must classify `name`, `compartmentId`,
  `ddlStatement`, `tableLimits`, `isAutoReclaimable`, and tags into
  create-only, mutable, or rejected drift.
- `psql/DbSystem`: downstream work must classify `displayName`,
  `compartmentId`, `dbVersion`, `storageDetails`, `shape`, `networkDetails`,
  `credentials`, `source`, `configId`, and tags.
- `mysql/MySqlDbSystem`: keep the handwritten narrower update surface explicit
  even though provider facts expose a much broader mutable set.
- Owner: kind-specific closure for `nosql/Table` and `psql/DbSystem`;
  legacy-only for `mysql/MySqlDbSystem`.

### Status projection

- Shared rule: `identity/User` is the generated-runtime precedent for required
  status projection plus OSOK lifecycle conditions.
- `nosql/Table` and `psql/DbSystem`: seed the generated-runtime baseline with
  required status projection.
- `mysql/MySqlDbSystem`: keep the current manual `OsokStatus`-only projection
  explicit.
- Owner: shared generated-runtime baseline for `nosql/Table` and
  `psql/DbSystem`; legacy-only for `mysql/MySqlDbSystem`.

### Secret inputs or outputs

- Shared rule: make repo-authored secret behavior explicit instead of implying
  it from provider facts.
- `nosql/Table`: no repo-authored secret reads or writes.
- `psql/DbSystem`: no Kubernetes secret reads or writes; credential fields
  remain OCI payload inputs.
- `mysql/MySqlDbSystem`: keep admin credential secret reads and ready-only
  endpoint secret writes explicit.
- Owner: no additional secret seam for `nosql/Table` or `psql/DbSystem`;
  legacy-only for `mysql/MySqlDbSystem`.

### Delete semantics

- Shared rule: `identity/User` is the generated-runtime precedent for retaining
  the finalizer until `Get` or list fallback confirms deletion.
- `nosql/Table` and `psql/DbSystem`: seed required delete confirmation plus
  `retain-until-confirmed-delete`.
- `mysql/MySqlDbSystem`: delete remains unresolved and legacy-only in this
  issue.
- Owner: shared generated-runtime baseline for `nosql/Table` and
  `psql/DbSystem`; legacy-only blocker for `mysql/MySqlDbSystem`.

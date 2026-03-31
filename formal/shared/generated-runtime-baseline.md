# Shared Generated Runtime Baseline

This checklist seeds `database/AutonomousDatabase`, `mysql/DbSystem`,
`nosql/Table`, and `psql/DbSystem` from the same reference points:

- `formal/controllers/identity/user` is the generated-runtime precedent for
  required status projection, read-after-write, and finalizer retention until
  delete confirmation.
- `formal/controllers/streaming/stream` plus the main-worktree
  `pkg/servicemanager/streams` package are the legacy reference for
  bind-versus-create and lifecycle-sensitive list lookup. Do not inherit its
  secret or best-effort delete behavior into generated-runtime rows.

## Checklist

### Bind-versus-create

- Shared rule: if no OCI ID is tracked, decide whether the service can reuse an
  existing OCI resource before create.
- `database/AutonomousDatabase`: use `ListAutonomousDatabases` with
  `compartment_id`, `autonomous_container_database_id`, `display_name`, and
  lifecycle state when formalizing the pre-create decision.
- `mysql/DbSystem`: use `ListDbSystems` with `compartment_id`, `display_name`,
  `db_system_id`, and lifecycle state when formalizing the pre-create decision.
- `nosql/Table`: use `ListTables` with `compartment_id`, `name`, and lifecycle
  state to decide whether create is required.
- `psql/DbSystem`: use `ListDbSystems` with `compartment_id`, `display_name`,
  `id`, and lifecycle state to decide whether create is required.

### List-lookup

- Shared rule: list filters and lifecycle matching are repo-authored semantics
  even when provider facts expose the datasource shape.
- `database/AutonomousDatabase`: imported provider facts cover
  `oci_database_autonomous_databases`.
- `mysql/DbSystem`: imported provider facts cover
  `oci_mysql_mysql_db_systems`.
- `nosql/Table`: imported provider facts cover `oci_nosql_tables`.
- `psql/DbSystem`: the provider exposes `oci_psql_db_systems`; the seeded
  baseline carries that datasource shape explicitly even though the current
  formal importer does not auto-wire it.

### Waiter handling

- Shared rule: provider-backed write completion must have one explicit formal
  answer before generated-runtime promotion.
- `database/AutonomousDatabase`: create, update, and delete currently perform a
  single follow-up read; formal coverage must decide whether lifecycle polling
  is required beyond that baseline.
- `mysql/DbSystem`: create, update, and delete currently perform a single
  follow-up read; formal coverage must decide whether lifecycle polling is
  required beyond that baseline.
- `nosql/Table`: provider CRUD uses work-request-backed completion for create,
  update, and delete.
- `psql/DbSystem`: create is work-request-backed and update also exposes
  `tfresource.WaitForUpdatedState`.

### Mutation policy

- Shared rule: imported mutable and force-new fields are the starting point,
  not the final OSOK contract.
- `database/AutonomousDatabase`: downstream work must classify the generated
  mutable and force-new fields into create-only, mutable, or rejected drift.
- `mysql/DbSystem`: downstream work must classify the generated mutable and
  force-new fields into create-only, mutable, or rejected drift.
- `nosql/Table`: downstream work must classify `name`, `compartmentId`,
  `ddlStatement`, `tableLimits`, `isAutoReclaimable`, and tags.
- `psql/DbSystem`: downstream work must classify `displayName`,
  `compartmentId`, `dbVersion`, `storageDetails`, `shape`, `networkDetails`,
  `credentials`, `source`, `configId`, and tags.

### Status projection

- Shared rule: `identity/User` is the generated-runtime precedent for required
  status projection plus OSOK lifecycle conditions.
- `database/AutonomousDatabase`, `mysql/DbSystem`, `nosql/Table`, and
  `psql/DbSystem` all seed the generated-runtime baseline with required status
  projection.

### Secret inputs or outputs

- Shared rule: make repo-authored secret behavior explicit instead of implying
  it from provider facts.
- `database/AutonomousDatabase`, `mysql/DbSystem`, `nosql/Table`, and
  `psql/DbSystem` have no Kubernetes secret reads or writes in the current
  generated-runtime path.

### Delete semantics

- Shared rule: `identity/User` is the generated-runtime precedent for retaining
  the finalizer until `Get` or list fallback confirms deletion.
- `database/AutonomousDatabase`, `mysql/DbSystem`, `nosql/Table`, and
  `psql/DbSystem` all seed required delete confirmation plus
  `retain-until-confirmed-delete`.

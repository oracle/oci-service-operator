---
schemaVersion: 1
surface: repo-authored-semantics
service: nosql
slug: table
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `nosql/Table` row after the
handwritten Table runtime semantics were encoded into the repo-authored formal
metadata and regenerated diagrams.

## Current runtime path

- `Table` now keeps the generated `TableServiceManager` shell but overrides the
  generated client seam with
  `pkg/servicemanager/nosql/table/table_generated_client_adapter.go`.
- The handwritten adapter resolves by tracked OCI ID first, then uses
  `ListTables` with `compartmentId`, `name`, and `LifecycleState=ALL`. It
  reuses only a single exact-name match, rereads that candidate through
  `GetTable`, and fails on ambiguous duplicate matches instead of guessing.
- The runtime projects OCI lifecycle into OSOK status, mapping `CREATING`,
  `UPDATING`, and `DELETING` into `Provisioning`, `Updating`, and
  `Terminating` with one-minute requeues, while `ACTIVE` settles success and
  `FAILED` becomes terminal without requeue.
- Delete keeps the finalizer until `GetTable` or `ListTables` confirms the
  table is deleted or no longer exists.

## Shared generated-runtime baseline

- Use the [shared generated-runtime baseline](../../../shared/generated-runtime-baseline.md)
  as the category map for bind, lookup, waiter, mutation, status, secret, and
  delete decisions.
- Follow the preserved explicit-backlog `identity/User` row for required
  status projection, required delete confirmation, and
  `retain-until-confirmed-delete`.
- Use `streaming/Stream` as the generated-runtime reference for pre-create
  lookup and lifecycle-sensitive matching. Do not inherit its secret or
  best-effort delete behavior.

## Repo-authored semantics

- `Table` has no Kubernetes secret reads or secret writes.
- Bind-before-create is explicit: tracked OCID wins, then `ListTables` searches
  the requested compartment for an exact-name match, and only zero matches open
  the `CreateTable` path.
- Mutation policy is explicit: `name` and `isAutoReclaimable` are rejected as
  replacement-only drift, `ddlStatement` remains create-only and never opens an
  implicit update path, and `compartmentId`, `tableLimits`, `freeformTags`, and
  `definedTags` are the supported mutable surface. When both compartment and
  tag/limit drift exist, `ChangeTableCompartment` runs before `UpdateTable`.
- Waiter closure is explicit: the handwritten path does not poll work requests.
  Create, compartment move, update, and delete all use read-after-write
  confirmation through `GetTable` plus list fallback when needed until OCI
  reaches `ACTIVE`, `DELETED`, or NotFound.

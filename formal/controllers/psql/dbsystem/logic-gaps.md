---
schemaVersion: 1
surface: repo-authored-semantics
service: psql
slug: dbsystem
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `psql/DbSystem` row after the
handwritten DbSystem adapter semantics were encoded into the repo-authored
formal metadata and regenerated diagrams.

## Current runtime path

- `DbSystem` now keeps the generated `DbSystemServiceManager` shell but
  overrides the generated client seam with
  `pkg/servicemanager/psql/dbsystem/dbsystem_generated_client_adapter.go`.
- The handwritten adapter resolves by tracked OCI ID first through
  `GetDbSystem`, then uses `ListDbSystems` with `compartmentId` and
  `displayName`. It reuses only a single exact match in bindable lifecycle
  states (`ACTIVE`, `CREATING`, `UPDATING`, `INACTIVE`,
  `NEEDS_ATTENTION`), rereads that candidate through `GetDbSystem`, and fails
  on ambiguous duplicate matches instead of guessing.
- Create accepts inline credentials or same-namespace `adminUsername` and
  `adminPassword` secret references, then mirrors only the resolved secret
  references into status for later drift checks.
- The runtime projects OCI lifecycle into OSOK status, mapping `CREATING`,
  `UPDATING`, and `DELETING` into `Provisioning`, `Updating`, and
  `Terminating` with one-minute requeues, while `ACTIVE`, `INACTIVE`, and
  `NEEDS_ATTENTION` settle success unless the reconcile is already in update or
  delete follow-up.
- Delete keeps the finalizer until `GetDbSystem` or `ListDbSystems` confirms
  the DbSystem is deleted or no longer exists, and it does not reissue
  `DeleteDbSystem` once OCI already reports `DELETING` or `DELETED`.
- Imported provider facts now cover `CreateDbSystem`, `GetDbSystem`,
  `ListDbSystems`, `ChangeDbSystemCompartment`, `PatchDbSystem`,
  `ResetMasterUserPassword`, `UpdateDbSystem`, and `DeleteDbSystem`.

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

- Secret inputs are explicit: create may read same-namespace admin credential
  secret refs, mirrors only non-empty secret refs into status, and never writes
  follow-up Kubernetes secrets.
- Bind-before-create is explicit: tracked OCID via `GetDbSystem` wins;
  otherwise `ListDbSystems` searches the requested compartment for an exact
  `displayName` match, only bindable lifecycle states are reusable, duplicate
  matches fail, and only zero matches open the `CreateDbSystem` path.
- Mutation policy is explicit: `displayName`, `description`,
  `storageDetails.iops`, `dbConfigurationParams.configId`,
  `dbConfigurationParams.applyConfig`, `managementPolicy`, `freeformTags`, and
  `definedTags` are the supported mutable surface. Top-level `configId`,
  `credentials`, admin secret refs, `instancesDetails`, and `source` remain
  create-time only, while `compartmentId`, `dbVersion`, `shape`, `systemType`,
  instance sizing/count, `networkDetails`, and storage placement/durability
  drift are rejected before OCI mutation.
- Waiter closure is explicit: the handwritten path does not poll work requests
  directly. Create and update rely on provider-backed completion plus
  `GetDbSystem` read-after-write projection, requeuing while OCI remains
  `CREATING`, `UPDATING`, or delete follow-up keeps `DELETING`; delete only
  releases the finalizer after `DELETED` or NotFound is observed through
  `GetDbSystem` or list fallback.

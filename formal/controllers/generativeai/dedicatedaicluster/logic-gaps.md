---
schemaVersion: 1
surface: repo-authored-semantics
service: generativeai
slug: dedicatedaicluster
gaps: []
---

# Logic Gaps

## Current runtime path

- `DedicatedAiCluster` uses the generated `DedicatedAiClusterServiceManager`
  and generated runtime client directly, with one service-local wrapper in
  `pkg/servicemanager/generativeai/dedicatedaicluster` that only narrows
  pre-create reuse.
- The service-local wrapper skips list-based reuse when
  `spec.displayName` is empty so create does not degrade to compartment-wide
  matching, and it clears stale tracked identifiers before re-entering the
  generated path when OCI no longer returns the previously recorded cluster.
- When `spec.displayName` is set and no OCI ID is tracked, the generated path
  may bind to an existing cluster by exact `compartmentId` plus `displayName`
  match before create.

## Repo-authored semantics

- `DedicatedAiCluster` uses read-after-write follow-up for create and update,
  requeues while OCI reports `CREATING`, `UPDATING`, or `DELETING`, and treats
  `ACTIVE` as the steady-state success target.
- The checked-in runtime supports in-place updates only for `displayName`,
  `description`, `unitCount`, `freeformTags`, and `definedTags`. It explicitly
  rejects replacement-only drift for `compartmentId`, `type`, and `unitShape`
  before any OCI update call.
- Provider facts also expose compartment moves through separate Terraform-side
  logic, but the checked-in OSOK baseline does not model that auxiliary path
  yet and keeps `compartmentId` replacement-only in the generated runtime.
- Status projection is part of the repo-authored contract. The generated runtime
  merges the live OCI `DedicatedAiCluster` response into published status
  fields, stamps `status.status.ocid`, and keeps OSOK lifecycle conditions
  aligned to the observed OCI lifecycle.
- Delete retains the finalizer until `GetDedicatedAiCluster` confirms `DELETED`
  or OCI no longer returns the resource. No Kubernetes secret reads or writes
  are part of this path.

---
schemaVersion: 1
surface: repo-authored-semantics
service: ocvp
slug: cluster
gaps: []
---

# Logic Gaps

## Current runtime path

- `Cluster` routes through the generated `ClusterServiceManager` and
  `generatedruntime.ServiceClient`, with a small service-local wrapper in
  `pkg/servicemanager/ocvp/cluster/cluster_runtime.go` that keeps identity
  resolution explicit for create-or-bind flows.
- The generated runtime handles create, update, delete, status projection, and
  finalizer retention directly from the checked-in `ocvp/cluster`
  controller/service-manager path; there is no handwritten OCI CRUD adapter for
  this resource.
- Because `CreateCluster` returns only a work-request header and not the new
  Cluster OCID, the repo-authored runtime requires `spec.displayName` whenever
  no tracked OCI identifier is already recorded. The wrapper rejects create or
  bind attempts without `displayName` instead of guessing from broad list
  matches.

## Shared generated-runtime baseline

- Use the [shared generated-runtime baseline](../../../shared/generated-runtime-baseline.md)
  for the common bind, lifecycle, mutation, status, and delete semantics.
- Keep delete confirmation explicit with
  `finalizer_policy = retain-until-confirmed-delete`; the finalizer only clears
  after OCI reports the Cluster missing or in `DELETED`.
- No Kubernetes secret reads or secret writes are part of this resource.

## Repo-authored semantics

- Bind lookup is explicit: when `spec.displayName` is set, the runtime matches
  `ListClusters` results on exact `sddcId` plus `displayName` and only reuses
  a unique Cluster in reusable lifecycle states (`ACTIVE`, `CREATING`, or
  `UPDATING`). Terminal, deleting, and duplicate matches are never rebound.
- Mutable drift is limited to `displayName`, `networkConfiguration`,
  `vmwareSoftwareVersion`, `esxiSoftwareVersion`, `freeformTags`, and
  `definedTags`. Fields absent from `UpdateClusterDetails` remain create-only
  drift, including `sddcId`, `computeAvailabilityDomain`, `esxiHostsCount`,
  `initialCommitment`, `initialHostShapeName`, `initialHostOcpuCount`,
  `isShieldedInstanceEnabled`, `capacityReservationId`, `workloadNetworkCidr`,
  `datastores`, and `instanceDisplayNamePrefix`.
- OCI accepts `networkConfiguration` and software-version updates for the
  Cluster object, but those values influence future ESXi host additions rather
  than rewriting the current VMware host fleet in place. The published runtime
  still treats that API-supported surface as legitimate update behavior.
- Create and delete responses expose work-request headers, while update returns
  the Cluster body directly. The reviewed generatedruntime client records those
  breadcrumbs when present and relies on lifecycle rereads plus confirm-delete
  follow-up rather than service-local work-request polling.

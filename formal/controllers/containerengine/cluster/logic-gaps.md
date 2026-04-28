---
schemaVersion: 1
surface: repo-authored-semantics
service: containerengine
slug: cluster
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `containerengine/Cluster` row after the
scaffold placeholder was replaced with repo-authored lifecycle, mutation, and
list-lookup semantics.

## Current seeded contract

- `Cluster` keeps the generated `ClusterServiceManager` shell today; this row
  seeds the controller contract that the generated-service-manager path must
  honor.
- OCI lifecycle classification is explicit: `CREATING` and `UPDATING` requeue,
  `ACTIVE` settles success, `FAILED` is terminal without requeue, and delete
  confirmation observes `DELETING` until `DELETED`.
- Mutation policy is explicit: only `UpdateClusterDetails` fields are mutable
  in place. That mutable surface is `name`, `kubernetesVersion`,
  `definedTags`, `freeformTags`, `imagePolicyConfig`, `type`,
  `options.admissionControllerOptions`, `options.persistentVolumeConfig`,
  `options.serviceLbConfig` including `backendNsgIds`,
  `options.openIdConnectTokenAuthenticationConfig`, and
  `options.openIdConnectDiscovery`. Fields omitted from
  `UpdateClusterDetails` remain create-only drift and never open implicit
  replacement.
- Read parity is explicit: `GetCluster` always sets
  `shouldIncludeOidcConfigFile=true` so the mutable OIDC auth
  `configurationFile` participates in drift detection and read-after-write
  follow-up observation instead of being silently truncated from live reads.
- Pre-create lookup semantics are explicit: `ListClusters` searches by
  `compartmentId`, `name`, and reusable lifecycle state, and only a single
  exact-name match in `ACTIVE`, `CREATING`, or `UPDATING` is safe to reuse.
  `FAILED`, `DELETING`, and `DELETED` candidates are never reusable.
- Create, update, and delete are work-request-backed at the API boundary
  because their SDK responses return `opc-work-request-id`, but the seeded
  lifecycle contract is still expressed in observed Cluster states rather than
  separate work-request status objects.
- Kubernetes secret reads and writes are out of scope for `Cluster`; the row
  keeps `secret_side_effects = none`.

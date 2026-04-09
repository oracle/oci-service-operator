---
schemaVersion: 1
surface: repo-authored-semantics
service: opensearch
slug: opensearchopensearchcluster
gaps: []
---

# Logic Gaps

## Current runtime path

- `OpensearchCluster` routes through the generated
  `OpensearchClusterServiceManager` and
  `generatedruntime.ServiceClient`; there is no legacy handwritten adapter for
  this resource.
- Non-primary OpenSearch helper kinds remain deferred in this checkout. Only
  `OpensearchCluster` is wired into controller and service-manager
  registration.
- The generated runtime now uses the published OSOK kind `OpensearchCluster`,
  matching the OCI SDK request and response types.
- Create or bind is explicit: the runtime first resolves a tracked OCI
  identifier when one is present, otherwise it lists clusters in the requested
  compartment and reuses a unique `displayName` match before issuing create.

## Shared generated-runtime baseline

- Use the [shared generated-runtime baseline](../../../shared/generated-runtime-baseline.md)
  for the common bind, lifecycle, mutation, status, and delete semantics.
- Keep delete confirmation explicit with
  `finalizer_policy = retain-until-confirmed-delete`; the finalizer only clears
  after OCI reports the cluster missing or in `DELETED`.
- No Kubernetes secret reads or secret writes are part of this resource.

## Repo-authored semantics

- Bind lookup matches on `compartmentId` plus `displayName`; lifecycle `state`
  remains a provider-fact filter so terminal or deleting clusters are not
  rebound during create or delete flows.
- Mutable drift remains narrow: only `displayName` is updated in place, while
  `compartmentId` stays create-only drift.
- Create, update, and delete all use read-after-write or confirm-delete
  follow-up, so OCI lifecycle transitions are projected back into status and
  condition state before the reconcile finishes.

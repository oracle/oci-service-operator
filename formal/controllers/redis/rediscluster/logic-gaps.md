---
schemaVersion: 1
surface: repo-authored-semantics
service: redis
slug: rediscluster
gaps: []
---

# Logic Gaps

## Repo-owned target

- `RedisCluster` is the published OSOK controller target for the OCI/provider
  resource `oci_redis_redis_cluster`.
- OCI SDK and terraform-provider-oci APIs still use `RedisRedisCluster` in
  request, response, and operation names. That provider-facing alias remains
  scaffold-only under `formal/controllers/redis/redisrediscluster`.
- No compatibility override should preserve `RedisRedisCluster` as a published
  OSOK kind because redis does not have a previously shipped GVK to keep.

## Runtime contract

- Bind resolution uses an explicit OCI identity first. When no OCI identity is
  tracked yet, the controller may adopt only a unique `ListRedisClusters`
  result that matches `spec.compartmentId` plus `spec.displayName`.
- Ambiguous list matches are treated as a reconcile error instead of silently
  binding the wrong redis cluster.
- Mutable drift is limited to `displayName`, `nodeCount`, `nodeMemoryInGBs`,
  `freeformTags`, and `definedTags`. Changes to `compartmentId`,
  `softwareVersion`, or `subnetId` require replacement semantics and are
  rejected once a live OCI resource is bound.
- Create and update rely on the standard generated-runtime read-after-write
  path with `GetRedisCluster` or `ListRedisClusters`, while delete keeps the
  finalizer until `DeleteRedisCluster` is confirmed.

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
- Create, update, and delete are work-request-backed via the Redis service SDK.
  The runtime stores the in-flight work request in `status.async.current`,
  projects raw work-request ID, status, operation type, percent complete, and a
  synthesized message for debugging, and resumes reconciliation from that
  shared tracker instead of relying on helper-name drift.
- Delete keeps the finalizer until the tracked Redis work request reaches a
  terminal state and `GetRedisCluster` confirms the resource is gone. The
  existing delete guard still blocks the initial delete request while OCI keeps
  the cluster in `CREATING` or `UPDATING`, but it no longer suppresses
  work-request polling after delete has started.

## Authority and scoped cleanup

- `formal/controllers/redis/rediscluster/*` is the authoritative formal path
  for the promoted Redis work-request adapter contract.
- `pkg/servicemanager/redis/rediscluster/rediscluster_runtime.go` and
  `pkg/servicemanager/redis/rediscluster/rediscluster_delete_guard_client.go`
  own live runtime behavior.
  `pkg/servicemanager/redis/rediscluster/rediscluster_serviceclient.go`
  still records the generated helper-hook baseline and should not be treated as
  a second execution contract.
- The disabled `service: workrequests` row in
  `internal/generator/config/services.yaml` is a separate generator-contract
  decision. Redis work-request polling does not publish or enable a standalone
  `workrequests` API group.

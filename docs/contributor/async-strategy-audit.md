# Enabled Resource Async Strategy Audit

This audit is the `oci-service-operator-6kv.1` baseline for the selected
controller-backed surface in `internal/generator/config/services.yaml`.

Current classifications used below:

- `lifecycle`: read-after-write, lifecycle-state requeue, or confirm-delete
  without explicit `GetWorkRequest` polling. Lifecycle resources may still
  capture an opening-response `OpcWorkRequestId` into the shared tracker
  without changing classification.
- `workrequest`: persisted work-request IDs plus explicit
  `GetWorkRequest` resume or polling.
- `none`: no selected kind currently proves this classification.

## Selected Surface

| Selected kind | Runtime package | Current | Target | Rationale |
| --- | --- | --- | --- | --- |
| `containerinstances/ContainerInstance` | `pkg/servicemanager/containerinstance` | `lifecycle` | `lifecycle` | The handwritten manager gates create and delete from live `LifecycleState` values and does not poll work requests. |
| `core/Instance` | `pkg/servicemanager/core/instance` | `lifecycle` | `lifecycle` | The generatedruntime config uses plain create, update, and delete hooks plus lifecycle-state lookup filters only. |
| `database/AutonomousDatabase` | `pkg/servicemanager/database/autonomousdatabase` | `lifecycle` | `lifecycle` | The generatedruntime config uses plain CRUD hooks with lifecycle-state list filtering and no work-request waiter. |
| `dataflow/Application` | `pkg/servicemanager/dataflow/application` | `lifecycle` | `lifecycle` | The selected path now uses generatedruntime for create, update, observe, and bounded delete/error hooks; only stale tracked recreate when OCI rereads report `DELETED` stays package-local. |
| `functions/Application` | `pkg/servicemanager/functions` | `lifecycle` | `lifecycle` | The manual manager stays on get or list plus delete-helper semantics without `GetWorkRequest` polling. |
| `functions/Function` | `pkg/servicemanager/functions` | `lifecycle` | `lifecycle` | The manual manager uses the same lifecycle and delete-helper pattern as `functions/Application`. |
| `identity/Compartment` | `pkg/servicemanager/identity/compartment` | `lifecycle` | `lifecycle` | The generated client is wrapped by an orphan-delete client that decides from compartment lifecycle, not work requests. |
| `keymanagement/Vault` | `pkg/servicemanager/keymanagement/vault` | `lifecycle` | `lifecycle` | The handwritten runtime wraps generatedruntime but projects Vault lifecycle, including pending-deletion handling, itself. |
| `mysql/DbSystem` | `pkg/servicemanager/mysql/dbsystem` | `lifecycle` | `lifecycle` | The selected runtime is plain generatedruntime plus credential and endpoint wrappers; no work-request adapter is active. |
| `nosql/Table` | `pkg/servicemanager/nosql/table` | `lifecycle` | `lifecycle` | The handwritten runtime models `CREATING`, `UPDATING`, `DELETING`, `FAILED`, and delete confirmation directly from table lifecycle. |
| `objectstorage/Bucket` | `pkg/servicemanager/objectstorage/bucket` | `lifecycle` | `lifecycle` | The selected runtime is plain generatedruntime plus namespace resolution only. |
| `opensearch/OpensearchCluster` | `pkg/servicemanager/opensearch/opensearchcluster` | `lifecycle` | `lifecycle` | The live path stays on generatedruntime read-after-write or confirm-delete semantics even though old create helper metadata remains on disk. |
| `psql/DbSystem` | `pkg/servicemanager/psql/dbsystem` | `lifecycle` | `lifecycle` | The handwritten adapter projects `CREATING`, `UPDATING`, and `DELETING` from live DbSystem lifecycle and confirms delete by reread. |
| `queue/Queue` | `pkg/servicemanager/queue/queue` | `workrequest` | `workrequest` | The handwritten runtime persists `CreateWorkRequestId`, `UpdateWorkRequestId`, and `DeleteWorkRequestId`, then resumes through `GetWorkRequest`. |
| `redis/RedisCluster` | `pkg/servicemanager/redis/rediscluster` | `lifecycle` | `workrequest` | The active path is still read-after-write plus delete guard, but the selected helper metadata and Redis SDK work-request APIs make this the first non-Queue adapter candidate. |
| `streaming/Stream` | `pkg/servicemanager/streaming/stream` | `lifecycle` | `lifecycle` | The generatedruntime config uses `WaitForUpdatedState` for update follow-up, not `GetWorkRequest` polling. |

No selected kind currently proves a `none` classification.

## Reference Migrations

- `queue/Queue` is the workrequest reference. The live runtime persists and
  resumes OCI work requests, recovers the created Queue OCID from work-request
  payloads, keeps delete confirmation explicit, and retains its legacy
  per-resource work-request ID fields only as a compatibility window around
  the shared tracker.
- `nosql/Table` is the lifecycle-only reference. Its handwritten runtime
  already owns read-after-write bind, lifecycle classification, and delete
  confirmation without any work-request dependency.
- `redis/RedisCluster` is the first non-Queue workrequest candidate. The
  selected runtime still behaves like lifecycle plus delete-guard handling, but
  the service SDK and checked-in helper metadata show enough work-request
  surface to justify adapter work in the epic.
- Every other selected kind remains inventory coverage for this epic unless a
  correctness bug proves it needs promotion out of lifecycle handling.

## Onboarding Defaults

- New selected resources should write the shared tracker first. When an
  opening OCI response carries `OpcWorkRequestId`, lifecycle resources may
  still project that breadcrumb into the shared tracker without becoming
  `workrequest` resources.
- Generated selected controllers inherit event-recorder RBAC by default, so
  event-only `extraRBACMarkers` are redundant rollout noise rather than an
  onboarding prerequisite.
- New selected resources should not add Queue-style per-resource work-request
  ID fields unless a later compatibility requirement explicitly proves they are
  needed.

## Drift Inventory

| Path | Drift | Scope |
| --- | --- | --- |
| `internal/generator/config/services.yaml` | The selected surface is enabled and package-owned, but no selected kind declares explicit async strategy metadata yet. | `epic-scope` |
| `pkg/shared/common_types.go` | `shared.OSOKStatus` has conditions, OCI ID, message, reason, and timestamps only; there is no generic async tracker for the shared contract. | `epic-scope` |
| `pkg/servicemanager/redis/rediscluster/rediscluster_serviceclient.go` | The selected generated metadata still advertises create-time `tfresource.WaitForWorkRequestWithErrorHandling` with `EntityType: "template"` while the live runtime stays on lifecycle plus delete guard. | `epic-scope` |
| `formal/controllers/redis/rediscluster/logic-gaps.md` | The formal contract still describes standard generatedruntime read-after-write behavior and must be updated when the Redis workrequest adapter becomes canonical. | `epic-scope` |
| `pkg/servicemanager/opensearch/opensearchcluster/opensearchcluster_serviceclient.go` | The selected generated metadata still carries create-time `tfresource.WaitForWorkRequestWithErrorHandling` even though the active runtime stays on lifecycle or confirm-delete semantics. | `epic-scope` |
| `formal/controller_manifest.tsv` | Scaffolded per-service `WorkRequest`, `WorkRequestError`, and `WorkRequestLog` rows remain on disk even though the dedicated `service: workrequests` group is disabled. | `follow-up-only` |
| `pkg/servicemanager/psql/dbsystem/dbsystem_generated_client_adapter.go` | `psql/DbSystem` remains lifecycle-only today even though the service SDK exposes work-request APIs; re-checking whether that becomes a correctness gap is follow-up work, not current epic scope. | `follow-up-only` |

## Follow-up Boundary

The audit does not promote additional adapter scope beyond Queue, Table, and
Redis. Residual lifecycle-only services that still sit on SDKs with
work-request APIs stay deferred to `oci-service-operator-0kb`,
`Audit lifecycle-only selected services that still expose OCI work-request
APIs`.

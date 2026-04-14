# API Error Coverage Registry

`pkg/errorutil/errortest/api_error_coverage_registry.go` is the reviewed
contract for OCI API-error coverage inventory. It turns
`internal/generator/config/services.yaml` into the authoritative source for
which resources need a reviewed registration and which resources need an
explicit exemption.

## Inventory Rules

- Only default-active, controller-backed services participate.
- The active selected resource set comes from `selection.includeKinds` plus
  `packageSplits.includeKinds`.
- Active controller-backed services must keep `selection.mode: explicit` for
  this contract. If a future rollout needs `selection.mode: all`, extend the
  inventory contract first instead of silently widening scope.
- Resource overrides with `controller.strategy: none` or
  `serviceManager.strategy: none` become explicit reviewed exceptions. They do
  not disappear implicitly just because the parent service is active.
- Every inventory item must land in exactly one bucket:
  reviewed registration or reviewed exception.

## Reviewed Schema

Each reviewed registration records:

- resource identity: service, group, version, kind
- effective coverage family
- supported operations
- delete not-found semantics
- retryable conflict semantics
- justified deviations for helper-heavy paths

Each reviewed exception records:

- resource identity
- explicit reason the kind is outside the active controller-backed scope

## Family Meanings

| Family | Use For | Current Examples |
| --- | --- | --- |
| `generatedruntime-plain` | shared generatedruntime CRUD with no special follow-up helper ownership | `core/Instance`, `database/AutonomousDatabase`, `mysql/DbSystem`, `objectstorage/Bucket` |
| `generatedruntime-follow-up` | generatedruntime paths whose error handling depends on follow-up helpers such as `WaitForUpdatedState` or `WaitForWorkRequestWithErrorHandling` | `containerengine/Cluster`, `opensearch/OpensearchCluster`, `streaming/Stream` |
| `generatedruntime-workrequest` | work-request-aware flows that keep explicit work-request tracking as the reviewed contract | `queue/Queue` |
| `manual-runtime` | direct handwritten runtimes whose primary OCI error handling stays in package-local create/update logic; per-resource delete/conflict semantics may still point at generatedruntime when delete is delegated there | `core/Vcn`, `core/InternetGateway`, `core/Subnet`, `core/SecurityList`, other active core-network runtimes |
| `legacy-adapter` | helper and adapter paths that still own bespoke not-found, delete-guard, orphan-delete, pending-deletion, or create-fallback behavior | `containerinstances/ContainerInstance`, `functions/Application`, `functions/Function`, `keymanagement/Vault`, `nosql/Table`, `psql/DbSystem`, `identity/Compartment` |

For the split-core parity clients, family and delete semantics are intentionally
separate reviewed fields. `core/Vcn`, `core/InternetGateway`,
`core/NatGateway`, `core/NetworkSecurityGroup`, and `core/ServiceGateway` stay
in the `manual-runtime` family because create/update ownership is handwritten,
but their registry entries record generatedruntime delete semantics because
`Delete` delegates to serviceclients with `DeleteFollowUp.Strategy =
"confirm-delete"`.

## Current Explicit Exceptions

Active selected services can still carry non-selected subresources that remain
outside the controller-backed API-error gate. The reviewed registry keeps those
exceptions explicit today:

- `keymanagement`: `Key`, `KeyVersion`, `ReplicationStatus`, `WrappingKey`
- `opensearch`: `Manifest`, `OpensearchClusterBackup`,
  `OpensearchOpensearchVersion`, `WorkRequest`, `WorkRequestError`,
  `WorkRequestLog`

Each of these stays exempt because `services.yaml` still marks the subresource
with `strategy: none`.

## Update Workflow

When `services.yaml` changes:

1. If a new kind becomes active through `selection.includeKinds` or
   `packageSplits.includeKinds`, add a reviewed registration with an explicit
   family, operation set, delete not-found semantics, retryable conflict
   semantics, and any deviation note.
2. If a kind stays on an active service but is intentionally excluded with
   `strategy: none`, add or keep an explicit reviewed exception.
3. Run `go test ./pkg/errorutil/errortest` to verify the reviewed registry still
   matches the authoritative inventory.

This contract is intentionally inventory-first. The registry can lead the test
backfill work, but it should not silently trail the selected rollout surface.

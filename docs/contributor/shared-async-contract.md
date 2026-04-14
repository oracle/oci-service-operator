# Shared Async Status Contract

This document is the `oci-service-operator-6kv.2` contract for the shared
async status surface published by OSOK CRDs.

## Canonical Shared Shape

The shared async tracker now lives at:

- `.status.status.async.current.source`
- `.status.status.async.current.phase`
- `.status.status.async.current.workRequestId`
- `.status.status.async.current.rawStatus`
- `.status.status.async.current.rawOperationType`
- `.status.status.async.current.normalizedClass`
- `.status.status.async.current.percentComplete`
- `.status.status.async.current.message`
- `.status.status.async.current.updatedAt`

The schema is OSOK-owned. Public status never exposes provider SDK enum types;
raw provider values are preserved only as plain strings in `rawStatus` and
`rawOperationType`.

## Shared Mapping Rules

The controller-owned mapper in `pkg/servicemanager` is the only supported path
from normalized async class plus phase into shared OSOK conditions:

- `pending` + `create` => `Provisioning`, requeue
- `pending` + `update` => `Updating`, requeue
- `pending` + `delete` => `Terminating`, requeue
- `succeeded` + `create|update` => `Active`, no requeue
- `succeeded` + `delete` => `Terminating`, keep requeueing until delete
  confirmation clears the finalizer path
- `failed`, `canceled`, `attention`, and `unknown` => `Failed`, no requeue by
  default

Resource runtimes may still own provider-specific raw-status normalization, but
they must not invent separate condition mappings once they have a normalized
class plus phase.

When a runtime has an explicit phase from the current OCI observation, that
phase wins over any previously persisted `status.async.current.phase`.
Persisted phase is only a fallback when the current observation cannot
determine phase directly.

## Compatibility Window

`shared.OSOKStatus.Async.Current` is canonical immediately.

During the staged migration window:

- Existing resource-local work-request ID fields may remain as compatibility
  mirrors when a runtime still needs them for persisted resume behavior.
- `queue/Queue` keeps `createWorkRequestId`, `updateWorkRequestId`, and
  `deleteWorkRequestId` for resume parity, but it must mirror the same
  in-flight operation into the shared tracker.
- New async migrations must write the shared tracker first; they must not add
  new per-resource work-request ID or raw-status fields to published status.
- Retirement of legacy resource-local async fields is owned by the follow-on
  resource children after live parity is proven.

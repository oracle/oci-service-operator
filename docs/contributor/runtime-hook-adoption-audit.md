# Runtime Hook Adoption Audit

This audit was originally checked in for `US-213`, refreshed in `US-218`
after `US-217` moved the three load-balancer proof packages onto the bounded
path-identity and nested-read seams, and refreshed again in `US-223` after
`US-222` and `US-223` moved the seven-package core networking family onto the
bounded phase-4 hook seam, and refreshed again in `US-228` after `US-227` and
`US-228` moved the four-package bind-guard family onto the bounded phase-5
`Identity` seam, and refreshed again in `US-238` after `redis/rediscluster`
moved onto the bounded async work-request seam, and refreshed again in
`US-239` after `queue/queue` moved onto the same bounded async seam.

The live residual set comes from:

```sh
rg -l "new[A-Za-z0-9]+ServiceClient\s*=" pkg/servicemanager | \
  rg -v '_serviceclient\.go$|_test\.go$'
```

That command now returns 6 live constructor rewrites. The `core` seven-package
networking family and the four-package bind-guard family are no longer in the
residual set. The remaining packages collapse into three concrete blocked
families:

| Family | Count | Packages |
| --- | --- | --- |
| Async resume and work-request state machines | 1 | `ailanguage/project` |
| Delete-confirmation and OCI-error overlays | 2 | `aispeech/transcriptionjob`, `dataflow/application` |
| Full manual runtime engines | 3 | `core/securitylist`, `nosql/table`, `psql/dbsystem` |

## Current Hook Boundary

- The generated scaffold now covers operation-field overrides,
  `BuildCreateBody`, `BuildUpdateBody`, `WrapGeneratedClient`, tracked-recreate
  clearing, bounded status/parity callbacks, and the checked-in `Identity` and
  `Read` seams for path-addressed subresources.
- That surface is now enough for the full seven-package core networking family:
  `core/vcn`, `core/internetgateway`, `core/networksecuritygroup`,
  `core/natgateway`, `core/routetable`, `core/subnet`, and
  `core/servicegateway`.
- That surface is also now enough for the full four-package bind-guard family:
  `generativeai/model`, `generativeai/dedicatedaicluster`, `ocvp/cluster`, and
  `ocvp/sddc`.
- The remaining 6 rewrites are still real blockers, but they no longer include
  the core networking wrapper family or the bind-guard family and they do not
  justify widening the checked-in runtime surface beyond the current bounded
  hooks.

## Family Inventory

### Async resume and work-request state machines

This package still owns explicit persisted async state and resume logic.

- `ailanguage/project`

The remaining shape is:

- persist a work-request ID after create, update, or delete
- resume reconcile from `GetWorkRequest(...)` rather than only from a read of
  the target resource
- recover or confirm resource identity from the work-request payload before the
  lifecycle can safely converge
- map service-local work-request statuses and actions into the shared async
  contract

### Delete-confirmation and OCI-error overlays

These packages already let generatedruntime own the steady-state create or
update path, but delete still needs a package-local overlay.

- `aispeech/transcriptionjob`
- `dataflow/application`

The common shape is:

- service-local delete confirmation instead of the stock generatedruntime
  delete-follow-up
- package-local OCI error normalization and request-ID projection
- lifecycle states whose meaning differs between normal observation and delete
  confirmation

### Full manual runtime engines

These packages still keep the whole runtime engine in handwritten code instead
of wrapping a generated delegate.

- `core/securitylist`
- `nosql/table`
- `psql/dbsystem`

The common shape is:

- package-local CRUD, status projection, and lifecycle handling stay explicit
- the live runtime owns behavior that is still broader than a thin wrapper,
  such as nested security-rule normalization, compartment moves, SDK contract
  guards, or credential-backed create details

## Exhaustive Package Classification

| Package | Primary blocked family | Current live reason |
| --- | --- | --- |
| `core/securitylist` | Full manual runtime engine | The runtime still owns full CRUD because nested rule normalization, stale optional status clearing, tracked-OCID recreate behavior, and an SDK contract guard are all package-local. |
| `ailanguage/project` | Async resume and work-request state machine | Persists and resumes create, update, and delete through `GetWorkRequest(...)`, then rebinds the OCI Project identity before convergence. |
| `aispeech/transcriptionjob` | Delete-confirmation and OCI-error overlay | The generated path still needs a handwritten delete layer because `CANCELING` and `CANCELED` mean different things for normal observation and delete confirmation. |
| `dataflow/application` | Delete-confirmation and OCI-error overlay | Uses an embedded generated client for create or update, but delete still needs package-local rereads, OCI error normalization, and request-ID projection. |
| `psql/dbsystem` | Full manual runtime engine | Keeps full create, update, delete, bind lookup, lifecycle handling, and credential-backed request construction in manual code. |
| `nosql/table` | Full manual runtime engine | Keeps full lifecycle-aware create, update, compartment move, and delete confirmation behavior in an explicit handwritten runtime. |

## Residual Design Input

With `queue/queue` off the residual list, the former repeated async-resume gap
is now isolated to one package.

Representative package:

- `ailanguage/project`

Current residual need:

- persist a work-request identifier across create, update, and delete phases
- resume reconcile from `GetWorkRequest(...)` rather than only from a direct
  read of the target resource
- recover or confirm the target OCI identity from work-request payloads before
  the lifecycle can safely converge
- map service-local work-request statuses and actions into the shared async
  contract

Why this stays residual design input instead of a new bounded hook claim:

- it spans create, update, and delete resume paths rather than a thin
  observation or parity seam
- the last remaining package still mixes async persistence with service-
  specific identity recovery and work-request payload shaping
- this audit records the remaining gap only; it does not claim a generic async
  resume hook surface

## Contract Note

`docs/api-generator-contract.md` already records the checked-in bounded hook
surface and says the remaining handwritten runtime seams stay explicit until
later rollout work closes them. This audit refresh only updates the live
residual inventory after `US-239`, so no further contract change is required
here.

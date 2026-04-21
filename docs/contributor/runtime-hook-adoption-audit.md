# Runtime Hook Adoption Audit

This audit was originally checked in for `US-213`, refreshed in `US-218`
after `US-217` moved the three load-balancer proof packages onto the bounded
path-identity and nested-read seams, and refreshed again in `US-223` after
`US-222` and `US-223` moved the seven-package core networking family onto the
bounded phase-4 hook seam.

The live residual set comes from:

```sh
rg -l "new[A-Za-z0-9]+ServiceClient\s*=" pkg/servicemanager | \
  rg -v '_serviceclient\.go$|_test\.go$'
```

That command now returns 12 live constructor rewrites. The `core` seven-package
networking family is no longer in the residual set. The remaining packages
collapse into four concrete blocked families:

| Family | Count | Packages |
| --- | --- | --- |
| Identity bind guards | 4 | `generativeai/model`, `generativeai/dedicatedaicluster`, `ocvp/cluster`, `ocvp/sddc` |
| Async resume and work-request state machines | 3 | `ailanguage/project`, `queue/queue`, `redis/rediscluster` |
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
- The remaining 12 rewrites are still real blockers, but they no longer
  include the core networking wrapper family and they do not justify widening
  the checked-in runtime surface beyond the current bounded hooks.

## Family Inventory

### Identity bind guards

These packages are still blocked on pre-bind identity logic rather than plain
generated delegate wrapping.

- `generativeai/model`
- `generativeai/dedicatedaicluster`
- `ocvp/cluster`
- `ocvp/sddc`

The common shape is:

- fail or bypass generated list-based reuse when no OCI ID is recorded and the
  runtime does not have a safe binding key yet
- require `spec.displayName` for bind safety or resolve the OCI ID through a
  package-local list query before the generated delegate runs
- clear stale recorded identity before retrying a create path that must skip
  list reuse

### Async resume and work-request state machines

These packages still own explicit persisted async state and resume logic.

- `ailanguage/project`
- `queue/queue`
- `redis/rediscluster`

The common shape is:

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
| `queue/queue` | Async resume and work-request state machine | Persists `CreateWorkRequestId`, `UpdateWorkRequestId`, and `DeleteWorkRequestId`, then resumes all phases through service-local work-request handling and Queue ID recovery. |
| `redis/rediscluster` | Async resume and work-request state machine | Persists the shared async tracker and resumes create, update, and delete from Redis work requests before lifecycle convergence. |
| `psql/dbsystem` | Full manual runtime engine | Keeps full create, update, delete, bind lookup, lifecycle handling, and credential-backed request construction in manual code. |
| `nosql/table` | Full manual runtime engine | Keeps full lifecycle-aware create, update, compartment move, and delete confirmation behavior in an explicit handwritten runtime. |
| `generativeai/model` | Identity bind guard | Clears stale tracked identity and skips list reuse when no safe binding key is present because `displayName` is optional on the create path. |
| `generativeai/dedicatedaicluster` | Identity bind guard | Keeps the same optional-`displayName` bind guard and stale-identity clearing shape as `generativeai/model`. |
| `ocvp/cluster` | Identity bind guard | Requires `spec.displayName` when no OCI identifier is recorded because the generated delegate does not have a narrower bind-safety seam yet. |
| `ocvp/sddc` | Identity bind guard | Resolves an existing OCI ID with `ListSddcs(...)` before delegating and sanitizes the read payload shape locally. |

## Residual Design Input

With the seven-package core networking family off the residual list, the
clearest remaining repeated need is the broader pre-bind identity-guard shape.

Representative packages:

- `generativeai/model`
- `generativeai/dedicatedaicluster`
- `ocvp/cluster`
- `ocvp/sddc`

Current repeated need:

- require a safe bind key before generated list reuse can run
- bypass or narrow create-time reuse when `displayName` is optional or not yet
  trustworthy
- resolve an OCI ID through a package-local list query before normal generated
  reconcile can proceed

Why this stays residual design input instead of a new bounded hook claim:

- it is earlier in the flow than the current tracked-recreate, status, and
  read-adaptation hooks
- the remaining packages mix bind safety with service-specific lookup and
  payload-shaping rules
- this audit records the repeated gap only; it does not claim a generic
  pre-bind identity guard seam

## Contract Note

`docs/api-generator-contract.md` already records the checked-in bounded hook
surface and says the remaining handwritten runtime seams stay explicit until
later rollout work closes them. This audit refresh only updates the live
residual inventory after `US-222` and `US-223`, so no further contract change
is required here.

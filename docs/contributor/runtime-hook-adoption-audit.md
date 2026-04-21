# Runtime Hook Adoption Audit

This audit was originally checked in for `US-213` and is refreshed in
`US-218` after `US-217` moved the three load-balancer proof packages onto the
bounded path-identity and nested-read seams.

The live residual set comes from:

```sh
rg -l "new[A-Za-z0-9]+ServiceClient\s*=" pkg/servicemanager | \
  rg -v '_serviceclient\.go$|_test\.go$'
```

That command now returns 19 live constructor rewrites. The remaining packages
are still real blockers, but they are no longer a flat list of one-off manual
exceptions. They collapse into five concrete blocked families:

| Family | Count | Packages |
| --- | --- | --- |
| Identity parity wrappers | 7 | `core/vcn`, `core/internetgateway`, `core/networksecuritygroup`, `core/natgateway`, `core/routetable`, `core/subnet`, `core/servicegateway` |
| Identity bind guards | 4 | `generativeai/model`, `generativeai/dedicatedaicluster`, `ocvp/cluster`, `ocvp/sddc` |
| Async resume and work-request state machines | 3 | `ailanguage/project`, `queue/queue`, `redis/rediscluster` |
| Delete-confirmation and OCI-error overlays | 2 | `aispeech/transcriptionjob`, `dataflow/application` |
| Full manual runtime engines | 3 | `core/securitylist`, `nosql/table`, `psql/dbsystem` |

## Current Hook Boundary

- The generated scaffold already exposes operation-field overrides,
  `BuildCreateBody`, `BuildUpdateBody`, `WrapGeneratedClient`, and the bounded
  `Identity` and `Read` seams for path-addressed subresources.
- That surface is enough for the packages already migrated in `US-210`,
  `US-211`, `US-212`, and the load-balancer proof migrations in `US-217`,
  where the live path only needed a bounded identity or read extension around
  the generated delegate.
- The remaining 19 rewrites all still need one of the blocked families below.
  This refresh records that inventory only. It does not widen the checked-in
  runtime surface into generic identity-repair, status-reset, async-resume, or
  OCI-error hooks.

## Family Inventory

### Identity parity wrappers

These packages still pre-read OCI by the tracked ID, clear stale identity when
OCI says the resource is gone, and then adjust the generated delegate path by
clearing projected status or injecting package-local parity behavior first.

- `core/vcn`
- `core/internetgateway`
- `core/networksecuritygroup`
- `core/natgateway`
- `core/routetable`
- `core/subnet`
- `core/servicegateway`

The common shape is:

- tracked ID lookup before the delegate runs
- `clearTrackedIdentity(...)` plus
  `generatedruntime.WithSkipExistingBeforeCreate(...)` when OCI no longer
  exposes the prior object
- package-local `clearProjectedStatus(...)` and `restoreStatus(...)` around the
  delegate when stale status would otherwise create parity drift
- local parity-only update or normalization logic that is still too specific
  for a shared hook, such as route-rule normalization, service normalization,
  mutable collection normalization, or NAT public-IP create intent tracking

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
  such as nested security-rule normalization, stale optional status clearing,
  compartment moves, SDK contract guards, or credential-backed create details

## Exhaustive Package Classification

| Package | Primary blocked family | Current live reason |
| --- | --- | --- |
| `core/vcn` | Identity parity wrapper | Clears tracked identity on OCI not found, clears projected status before the delegate, restores status on failure, and keeps create-only list parity logic local. |
| `core/internetgateway` | Identity parity wrapper | Clears projected status around the delegate and still owns parity-only update decisions for local read or update handling. |
| `core/networksecuritygroup` | Identity parity wrapper | Keeps local retry-state handling plus clear-or-restore status parity around the generated delegate. |
| `core/natgateway` | Identity parity wrapper | Tracks recreate intent, preserves `PublicIpIdCreateIntent`, and keeps local parity update behavior around the delegate. |
| `core/routetable` | Identity parity wrapper | Normalizes equivalent route rules, clears projected status before delegate create or update, and owns explicit delete confirmation. |
| `core/subnet` | Identity parity wrapper | Normalizes mutable collections and create-only parity flags, clears projected status before delegation, and keeps explicit delete confirmation local. |
| `core/servicegateway` | Identity parity wrapper | Normalizes equivalent services, clears projected status before delegation, and still owns parity-only update handling. |
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

No new narrow later-phase candidate replaces the load-balancer proof family.
With those packages off the residual list, the clearest remaining repeated need
is the broader identity parity wrapper shape.

Representative packages:

- `core/vcn`
- `core/internetgateway`
- `core/networksecuritygroup`
- `core/natgateway`
- `core/routetable`
- `core/subnet`
- `core/servicegateway`

Current repeated need:

- clear tracked identity when OCI no longer exposes the prior object
- clear and sometimes restore projected status around the generated delegate so
  stale status does not create parity drift
- keep package-local parity normalization or recreate-intent handling around
  create, update, or delete decisions

Why this stays residual design input instead of a new bounded hook claim:

- it is broader than the bounded `Identity` and `Read` seams that already
  landed for the load-balancer proof migrations
- the remaining packages still mix identity repair with package-specific
  normalization, delete confirmation, or recreate-intent semantics
- this audit records the repeated gap only; it does not claim a generic parity
  repair or projected-status-reset hook shape

## Contract Note

`docs/api-generator-contract.md` already records the checked-in `Identity` and
`Read` seams and says the remaining handwritten runtime seams stay explicit
until later rollout work closes them. This audit refreshes the residual
inventory only, so no further contract change is required in `US-218`.

# Runtime Hook Adoption Audit

This audit is the `US-213` closeout for the phase-1 runtime-hook adoption
stories (`US-210`, `US-211`, and `US-212`).

The live residual set comes from:

```sh
rg -l "new[A-Za-z0-9]+ServiceClient\s*=" pkg/servicemanager | \
  rg -v '_serviceclient\.go$|_test\.go$'
```

That command now returns 22 live constructor rewrites. The remaining packages
are still real blockers, but they are no longer a flat list of one-off manual
exceptions. They collapse into six concrete blocked families:

| Family | Count | Packages |
| --- | --- | --- |
| Identity parity wrappers | 7 | `core/vcn`, `core/internetgateway`, `core/networksecuritygroup`, `core/natgateway`, `core/routetable`, `core/subnet`, `core/servicegateway` |
| Identity bind guards | 4 | `generativeai/model`, `generativeai/dedicatedaicluster`, `ocvp/cluster`, `ocvp/sddc` |
| Load-balancer path identity adapters | 3 | `loadbalancer/backend`, `loadbalancer/backendset`, `loadbalancer/listener` |
| Async resume and work-request state machines | 3 | `ailanguage/project`, `queue/queue`, `redis/rediscluster` |
| Delete-confirmation and OCI-error overlays | 2 | `aispeech/transcriptionjob`, `dataflow/application` |
| Full manual runtime engines | 3 | `core/securitylist`, `nosql/table`, `psql/dbsystem` |

## Current Hook Boundary

- The generated scaffold already exposes operation-field overrides,
  `BuildCreateBody`, `BuildUpdateBody`, and `WrapGeneratedClient`.
- That seam is enough for the packages already migrated in `US-210`,
  `US-211`, and `US-212`, where the live path only needed a narrow generated
  delegate wrapper.
- The remaining 22 rewrites all still need one of the blocked families below.
  This story records that inventory only. It does not add `MutateConfig`, a
  generic identity-repair seam, a generic status-reset seam, a generic
  async-resume seam, or a broad OCI-error hook.

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

### Load-balancer path identity adapters

These packages are not identified by a durable standalone OCI ID in the same
way as the plain generatedruntime baseline.

- `loadbalancer/backend`
- `loadbalancer/backendset`
- `loadbalancer/listener`

The common shape is:

- resolve identity from parent-path fields such as `loadBalancerId`,
  `backendSetName`, `listenerName`, or the synthesized backend name
- persist path identity into status before delegating
- seed a synthetic tracked ID for existing or delete flows when the subresource
  does not expose a standalone OCI identifier
- for `listener`, synthesize the `Get` and `List` read model from
  `GetLoadBalancer(...)` because the nested listener surface does not match the
  default generated assumptions

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
| `loadbalancer/backend` | Load-balancer path identity adapter | Resolves identity from `loadBalancerId`, `backendSetName`, and the synthesized backend name, then seeds a synthetic tracked ID for existing and delete flows. |
| `loadbalancer/backendset` | Load-balancer path identity adapter | Persists load-balancer path identity and uses a synthetic tracked ID for existing and delete flows because the subresource is still path-addressed. |
| `loadbalancer/listener` | Load-balancer path identity adapter | Rebuilds `Get` and `List` from `GetLoadBalancer(...)`, so the generated runtime still needs a package-local nested read model. |
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

## Later-Phase Candidate API Shape

The cleanest repeated later-phase candidate is a bounded load-balancer
subresource identity seam.

Proof packages:

- `loadbalancer/backend`
- `loadbalancer/backendset`
- `loadbalancer/listener`

Current repeated need:

- persist parent-path identity into status before generated CRUD runs
- optionally seed a synthetic tracked ID for existing and delete flows
- optionally synthesize `Get` or `List` from a parent read when the nested SDK
  surface does not match the default generated assumptions

Bounded candidate shape for a later phase, not implemented here:

- a load-balancer-specific subresource identity helper on runtime hooks or
  config that can:
  - persist named path keys into status
  - seed or restore a synthetic tracked ID for path-only resources
  - plug a nested read adapter into generated `Get` or `List`

This is narrower than a generic identity-repair hook and matches the one
remaining repeated path-identity family without claiming broader runtime-hook
coverage.

## Contract Note

`docs/api-generator-contract.md` already says the checked-in generated runtime
surface exposes manual extension seams and that remaining handwritten runtime
seams stay explicit until later rollout work closes them. This audit updates
the residual inventory only, so no contract change is required in `US-213`.

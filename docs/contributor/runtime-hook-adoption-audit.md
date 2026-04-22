# Runtime Hook Adoption Audit

This audit was originally checked in for `US-213`, refreshed in `US-218`
after `US-217` moved the three load-balancer proof packages onto the bounded
path-identity and nested-read seams, and refreshed again in `US-223` after
`US-222` and `US-223` moved the seven-package core networking family onto the
bounded phase-4 hook seam, and refreshed again in `US-228` after `US-227` and
`US-228` moved the four-package bind-guard family onto the bounded phase-5
`Identity` seam, and refreshed again in `US-238` after `redis/rediscluster`
moved onto the bounded async work-request seam, and refreshed again in
`US-239` after `queue/queue` moved onto the same bounded async seam, and
refreshed again in `US-240` after `ailanguage/project` moved onto that seam,
and refreshed again in `US-246` after `dataflow/application` moved onto the
bounded delete/error seam, and refreshed again in `US-248` after
`aispeech/transcriptionjob` moved onto the bounded delete-hook seam.

The live residual set comes from:

```sh
rg -l "new[A-Za-z0-9]+ServiceClient\s*=" pkg/servicemanager | \
  rg -v '_serviceclient\.go$|_test\.go$'
```

That command now returns 3 live constructor rewrites. The `core` seven-package
networking family and the four-package bind-guard family are no longer in the
residual set, the former async-resume family is no longer in the live
inventory, and the delete-confirmation and OCI-error overlay family has now
closed. The remaining packages collapse into one concrete blocked family:

| Family | Count | Packages |
| --- | --- | --- |
| Full manual runtime engines | 3 | `core/securitylist`, `nosql/table`, `psql/dbsystem` |

## Current Hook Boundary

- The generated scaffold now covers operation-field overrides,
  `BuildCreateBody`, `BuildUpdateBody`, `WrapGeneratedClient`, tracked-recreate
  clearing, bounded status/parity callbacks, the checked-in `Identity` and
  `Read` seams for path-addressed subresources, the bounded `Async` seam for
  work-request-backed resources, and the delete-only `DeleteHooks` seam for
  confirm-delete overlays.
- That surface is now enough for the full seven-package core networking family:
  `core/vcn`, `core/internetgateway`, `core/networksecuritygroup`,
  `core/natgateway`, `core/routetable`, `core/subnet`, and
  `core/servicegateway`.
- That surface is also now enough for the full four-package bind-guard family:
  `generativeai/model`, `generativeai/dedicatedaicluster`, `ocvp/cluster`, and
  `ocvp/sddc`.
- That surface is now also enough for the former delete-confirmation and
  OCI-error overlay family: `aispeech/transcriptionjob`.
- The remaining 3 rewrites are still real blockers, but they no longer include
  the core networking wrapper family, the bind-guard family, the former
  async-resume family, or the former delete-confirmation and OCI-error overlay
  family, and they do not justify widening the checked-in runtime surface
  beyond the current bounded hooks.

## Family Inventory

### Full manual runtime engines

The delete-confirmation and OCI-error overlay family is now closed, so these
packages are the only remaining live constructor rewrites. They still keep the
whole runtime engine in handwritten code instead of wrapping a generated
delegate.

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
| `nosql/table` | Full manual runtime engine | Keeps full lifecycle-aware create, update, compartment move, and delete confirmation behavior in an explicit handwritten runtime. |
| `psql/dbsystem` | Full manual runtime engine | Keeps full create, update, delete, bind lookup, lifecycle handling, and credential-backed request construction in manual code. |

## Residual Design Input

With `ailanguage/project` off the residual list, the bounded async
work-request seam no longer has any live constructor rewrites to retire.

With `aispeech/transcriptionjob` off the residual list, the bounded
delete-only `DeleteHooks` seam no longer has any live constructor rewrites to
retire.

The remaining residual need now stays concentrated in one shape:

- full manual runtime engines whose behavior is still broader than a thin
  wrapper

This audit still records the live gaps only; it does not claim any broader
hook surface beyond the current bounded runtime seams.

## Contract Note

`docs/api-generator-contract.md` already records the checked-in bounded hook
surface and says the remaining handwritten runtime seams stay explicit until
later rollout work closes them. This audit refresh only updates the live
residual inventory after `US-247` and `US-248`, so no further contract change
is required here.

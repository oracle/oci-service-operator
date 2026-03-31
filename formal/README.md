# Formal Scaffold

Start here before designing or changing reconciliation logic. The diagrams in
this tree are part of the design input, not just generated output.

This tree is the repo-local source of truth for formal runtime inputs.

- `imports/` holds provider-fact JSON only.
- `controllers/` holds repo-authored semantics, logic gaps, and diagram metadata.
- `controller_diagrams/` holds the shared diagram strategy YAML that controller-local
  rendering derives from.
- `shared/` holds TLA modules that encode the shared reconciler and service-manager contracts.
- `shared/diagrams/` holds the shared reconcile, resolution, delete, controller-state,
  and legend `.puml` and `.svg` artifacts rendered from `controller_diagrams/*.yaml`.
- `sources.lock` records the pinned provider source boundary that `formal-import` refreshes.
- `controller_manifest.tsv` binds one controller row to exactly one import, spec, logic-gap file, and diagrams directory.

The checked-in `template` row remains scaffold-only as a schema example. Seeded rows for
`database/AutonomousDatabases`, `mysql/MySqlDbSystem`, `streaming/Stream`, and
`identity/User` exercise the first resource-specific formal corpus.

Use `make formal-scaffold` to expand scaffold-only entries from the published API
inventory in `internal/generator/config/services.yaml`, and optionally merge in
provider-discovered rows with
`FORMAL_PROVIDER_PATH=/path/to/terraform-provider-oci`.
Use `make formal-diagrams` to deterministically rerender the checked-in
shared `activity`, `sequence`, `state-machine`, and `legend` `.puml` and `.svg`
artifacts under `shared/diagrams/` plus each controller's `activity`,
`sequence`, and `state-machine` `.puml` and `.svg` artifacts from the checked-in
`controller_diagrams/*.yaml` strategy, repo-authored metadata, and imported provider facts.
The SVGs are rendered by the `plantuml` CLI, so keep `plantuml` available on `PATH`
before running `make formal-diagrams`, `make formal-scaffold`, or `make formal-verify`.
Use `make formal-scaffold-verify FORMAL_PROVIDER_PATH=/path/to/terraform-provider-oci`
to confirm the checked-in formal catalog covers the provider-backed inventory.
Use `make formal-import FORMAL_IMPORT_PROVIDER_PATH=/path/to/terraform-provider-oci` to pin the source lock and refresh non-scaffold provider-fact imports.

## Onboarding and Promotion Checklist

Use this sequence when formal coverage is part of service onboarding or
controller-backed promotion:

1. Keep service inventory and rollout metadata in
   `internal/generator/config/services.yaml`. New services start
   `packageProfile: crd-only`; add `formalSpec` at the service level or under
   `generation.resources[]` once the published kind has a matching manifest row.
2. Run `make formal-scaffold` to create or refresh scaffold-only rows for the
   published API inventory, and pass
   `FORMAL_PROVIDER_PATH=/path/to/terraform-provider-oci` when provider-wide
   coverage should be refreshed. The command also renders deterministic
   shared `.puml` and `.svg` diagrams in `formal/shared/diagrams/` and controller-local
   `activity`, `sequence`, and `state-machine` `.puml` and `.svg` files from
   `controller_diagrams/*.yaml`, `runtime-lifecycle.yaml`, `controller_manifest.tsv`,
   and controller `spec.cfg` metadata. When only the rendered artifacts need a
   refresh after editing repo-authored or imported formal inputs, run
   `make formal-diagrams`. Then update the matching
   `formal/controller_manifest.tsv` row,
   `formal/controllers/<service>/<slug>/spec.cfg`,
   `formal/controllers/<service>/<slug>/logic-gaps.md`,
   `formal/controllers/<service>/<slug>/diagrams/`, and
   `formal/imports/<service>/<slug>.json`.
3. Run `make formal-import FORMAL_IMPORT_PROVIDER_PATH=/path/to/terraform-provider-oci`.
   The provider path is a pinned external input; the repo does not assume a
   sibling checkout.
4. Run `make formal-verify` before generator or runtime-gate work. When the
   pinned provider inventory is in scope, also run
   `make formal-scaffold-verify FORMAL_PROVIDER_PATH=/path/to/terraform-provider-oci`.
   `formal-verify` now checks generated `.puml` sources plus the embedded
   PlantUML metadata inside rendered SVGs, so any import/spec/logic change that
   affects diagrams must be followed by `make formal-diagrams` or
   `make formal-scaffold`.
5. Keep every unsupported behavior in `logic-gaps.md` front matter with an
   explicit `stopCondition`, and file linked `bd` follow-up issues instead of
   leaving TODOs. Open logic gaps block formal-driven promotion.
6. After formal inputs are valid, run
   `go run ./cmd/generator --config internal/generator/config/services.yaml --all`,
   `go run ./hack/update_validator_registries.go --write`,
   `make generated-coverage-gate`, and `make generated-runtime-gate`.
7. For resources that still depend on manual runtime behavior, keep
   `_generated_client_adapter.go` shims, manual webhook files, and other legacy
   seams explicit until their stop conditions close; do not treat formal
   scaffold coverage as permission to delete those files.

## Coverage Staging

## Shared vs Controller-Local Strategy

`formal/shared/diagrams/` carries the common reconcile semantics once:

- `shared-reconcile-activity.puml` and `.svg` explain the controller-agnostic reconcile flow.
- `shared-resolution-sequence.puml` and `.svg` explain shared OCID binding, datasource lookup,
  and pagination behavior.
- `shared-delete-sequence.puml` and `.svg` explain finalizer retention, delete confirmation,
  optional wait tracking, and optional Secret cleanup.
- `shared-controller-state-machine.puml` and `.svg` explain the common controller phase model.
- `shared-legend.puml` and `.svg` explain the shared palette plus the `generated-service-manager`
  and `legacy-adapter` controller batches.

Each controller keeps generated `activity.svg`, `sequence.svg`, and
`state-machine.svg` outputs under `controllers/<service>/<slug>/diagrams/`. The
generator specializes those controller-local diagrams from the shared
`controller_diagrams/*.yaml` strategy, the controller manifest row, controller
`spec.cfg`, repo-authored runtime metadata, and imported provider facts derived
from the public `terraform-provider-oci` behavior.

`formal/` is controller-scoped, not service-scoped. The steady-state target is one
manifest row plus sibling `controllers/<service>/<slug>/...` and
`imports/<service>/<slug>.json` entries for each published top-level API kind
resolved from `internal/generator/config/services.yaml`, plus provider-backed
rows for additional `terraform-provider-oci` groups and resources that are not
yet published as OSOK APIs.

When expanding `formal/` across existing API groups:

- use `make formal-scaffold` to generate scaffold rows from the resolved OSOK
  API inventory and, when `FORMAL_PROVIDER_PATH` is set, merge in
  `terraform-provider-oci` registrations without copying upstream diagram
  artifacts;
- keep new rows at `stage=scaffold` with placeholder imports until repo-authored
  semantics and logic gaps are written; scaffold rows must pass
  `make formal-verify`, provider-backed coverage should pass
  `make formal-scaffold-verify`, and `make formal-import` will skip scaffold
  rows;
- do not add `formalSpec` to a service or resource until its row is promoted
  beyond scaffold and the required stop conditions are explicit;
- treat scaffold coverage as catalog expansion only; it does not change runtime
  ownership, legacy adapters, or controller-backed rollout by itself.

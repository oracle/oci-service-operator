# OSOK Validator Guide

This document describes the tooling under `internal/validator` and how to use the `osok-schema-validator` CLI to keep the OSOK controllers and CRDs aligned with the Oracle Cloud Infrastructure (OCI) Go SDK.

## Purpose

`osok-schema-validator` compares OSOK controller/API usage against curated OCI SDK structs so schema and controller coverage regressions are visible early.

## Objectives

- **Provider-driven**: Traverse OSOK service manager/controller code to discover OCI request struct usage.
- **OCI SDK aware**: Reflect over tracked SDK targets and capture metadata such as mandatory flags and read-only/deprecated semantics.
- **Allowlist support**: Classify intentional gaps via `validator_allowlist.yaml`.
- **CI-friendly**: Emit deterministic output and useful exit codes for build gates.

## What the Validator Covers

The CLI produces three kinds of analysis:

1. **Controller coverage** — For each curated SDK struct (create/update “Details” types), it inspects the reconciler source code and reports whether each field is actually set. Missing fields show up as `unclassified` unless the allowlist assigns a more meaningful status (e.g. `future_consideration`, `intentionally_omitted`, `potential_gap`). Controller coverage is what you use to enforce that upgrades don’t drop required fields.

2. **API coverage** — It walks the generated CRD specs for the API groups configured in `internal/generator/config/services.yaml` across `api/<group>/<version>` packages (for example `api/core/v1beta1`, `api/database/v1beta1`, `api/functions/v1beta1`, `api/mysql/v1beta1`, and `api/streaming/v1beta1`), maps them to the corresponding SDK request structs, and reports three buckets:
   - **Present fields** — Spec fields that map to the SDK payload.
   - **Missing fields** — SDK fields the spec doesn’t expose. Mandatory SDK fields default to `potential_gap`; optional ones to `future_consideration` unless the allowlist overrides them.
   - **API-only fields** — Spec fields with no matching SDK field (useful when you add CRD-only metadata).

3. **SDK upgrade diff (`--upgrade-from/--upgrade-to`)** — Diffs two OCI SDK versions and tells you which fields/operations were added or removed. Optional `--provider-path` annotates which fields the current controllers already touch and suggests allowlist entries for new fields.

API coverage and controller coverage use the same allowlist semantics so you can keep both sides in sync.

## Key Concepts

- **Allowlist (`validator_allowlist.yaml`)** — Classifies missing fields. Without it, controller gaps remain `unclassified` and API gaps fall back to `potential_gap` (mandatory) or `future_consideration` (optional).
- **Baseline (`--write-baseline` / `--baseline`)** — Snapshot of a past controller run. Use it with `--fail-on-new-actionable` to fail CI if a field regresses into `potential_gap` or `unclassified`.
- **Exit codes** — The CLI always prints the report. It only exits non-zero when you pass `--baseline … --fail-on-new-actionable` *and* a controller field became `potential_gap` or `unclassified` compared to that baseline.

## Running the Validator

Assuming you’re at the repo root (`--provider-path .`):

```bash
# Run full controller+API report (default ASCII table)
go run ./cmd/osok-schema-validator --provider-path .

# Run for a single service only (controller + API)
go run ./cmd/osok-schema-validator --provider-path . --service core

# Markdown / JSON output
go run ./cmd/osok-schema-validator --provider-path . --format markdown
go run ./cmd/osok-schema-validator --provider-path . --format json | jq .

# Write a baseline snapshot of the controller report
go run ./cmd/osok-schema-validator --provider-path . --write-baseline validator-baseline.json

# Compare against a baseline and fail on new controller gaps
go run ./cmd/osok-schema-validator \
  --provider-path . \
  --baseline validator-baseline.json \
  --fail-on-new-actionable

# SDK upgrade diff (requires both versions in GOMODCACHE)
go run ./cmd/osok-schema-validator \
  --provider-path . \
  --upgrade-from v65.61.1 \
  --upgrade-to   v65.104.0 \
  --format markdown
```

Common flags:

| Flag | Purpose |
| --- | --- |
| `--provider-path` | Path to the OSOK repo (defaults to `.`). |
| `--allowlist` | Path to the allowlist file (defaults to `validator_allowlist.yaml`). |
| `--service` | Optional service filter (for example `core`, `identity`, `psql`). |
| `--format` | `table` (default), `markdown`, or `json`. |
| `--baseline` | Load a previous controller report when diffing. |
| `--write-baseline` | Save the current controller report as JSON. |
| `--fail-on-new-actionable` | Exit non-zero if new controller gaps appear vs. baseline. |
| `--upgrade-from` / `--upgrade-to` | Run the SDK upgrade diff instead of coverage reports. |

## Package Layout

```text
internal/validator/
  allowlist/   # YAML loader for field classifications
  config/      # Option parsing/validation
  provider/    # AST-based analyzer for OSOK provider usage
  sdk/         # Reflection/source-based OCI SDK analyzer
  diff/        # Comparison engine and coverage stats
  report/      # Renderers for table/markdown/json
  apispec/     # CRD spec vs SDK coverage mapping
  upgrade/     # SDK version diff analysis
  run.go       # Main coverage orchestration
  upgrade_runner.go
cmd/osok-schema-validator/
  main.go      # CLI wrapper
```

## Makefile Integration

Run validator directly through the repo `Makefile`:

```bash
# Full report for all services, written to validator-report.json
make schema-validator

# Service-specific report
make schema-validator SCHEMA_VALIDATOR_SERVICE=core

# Custom output file + format
make schema-validator \
  SCHEMA_VALIDATOR_SERVICE=identity \
  SCHEMA_VALIDATOR_FORMAT=json \
  SCHEMA_VALIDATOR_REPORT=identity-validator-report.json
```

`schema-validator` target variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `SCHEMA_VALIDATOR_PROVIDER_PATH` | `.` | Provider path passed to the CLI. |
| `SCHEMA_VALIDATOR_SERVICE` | empty | Optional service filter. |
| `SCHEMA_VALIDATOR_FORMAT` | `json` | Output format passed to the CLI. |
| `SCHEMA_VALIDATOR_REPORT` | `validator-report.json` | File path to write CLI output. |

## Generated Snapshot Coverage Report

For generator work, use the generated-output coverage workflow instead of
rewriting the checked-in tree by hand. This command creates a snapshot workspace,
renders selected services into that snapshot, restores the checked-in
non-generated API, controller, and service-manager companion Go files for the
selected services, refreshes the validator registries inside the snapshot,
regenerates deepcopy code for the generated API groups, and then runs
`osok-schema-validator` from that snapshot so the report reflects the generated
API types plus the checked-in handwritten runtime seams. This workflow is
focused on API and validator coverage only; generated controller and
service-manager compilation is covered by the runtime gate below.

```bash
# Full generated-output baseline report
make generated-coverage-report

# Limit the snapshot run to one service and keep the snapshot for inspection
make generated-coverage-report \
  GENERATED_COVERAGE_SERVICE=functions \
  GENERATED_COVERAGE_REPORT=/tmp/functions-generated-coverage.json \
  GENERATED_COVERAGE_SNAPSHOT_DIR=/tmp/osok-functions-snapshot

# Direct CLI usage (stdout is JSON)
go run ./cmd/osok-generated-coverage --all > before.json
go run ./cmd/osok-generated-coverage --service functions > after.json
```

Common variables and flags:

| Variable / Flag | Default | Purpose |
| --- | --- | --- |
| `--config` | `internal/generator/config/services.yaml` | Generator config used for the snapshot coverage run. The Makefile passes the repo's effective generator config. |
| `GENERATED_COVERAGE_SERVICE` / `--service` | empty | Run the snapshot report for one configured service. |
| `--all` | false | Report all configured services. |
| `GENERATED_COVERAGE_REPORT` / `--report-out` | `generated-coverage-report.json` for the Makefile target | Write the generated coverage summary JSON. |
| `GENERATED_COVERAGE_TOP` / `--top` | `10` | Number of top offenders to keep per category. Use `0` for all. |
| `GENERATED_COVERAGE_SNAPSHOT_DIR` / `--snapshot-dir` | empty | Keep the generated snapshot at a specific path instead of using an auto-cleaned temp dir. |
| `GENERATED_COVERAGE_KEEP_SNAPSHOT` / `--keep-snapshot` | empty / false | Keep an automatically created temp snapshot after a successful run. |
| `GENERATED_COVERAGE_VALIDATOR_JSON` / `--validator-json-out` | empty | Optional path for the full raw validator JSON from the snapshot run. |
| `GENERATED_COVERAGE_BASELINE` / `--baseline` | `internal/generator/config/generated_coverage_baseline.json` | Baseline file used by the regression gate. |
| `--write-baseline` | empty | Refresh the baseline file intentionally from the current generated snapshot report. |
| `--fail-on-regression` | false | Exit non-zero if coverage scope or metrics regress compared to `--baseline`. |
| `--controller-gen` | `<repo>/bin/controller-gen` | `controller-gen` binary used to regenerate deepcopy code inside the snapshot. |

The summary JSON includes:

- `summary.aggregate` — aggregate counts and overall / mandatory coverage percentages.
- `summary.services[]` — per-service rollups for tracked mappings, missing fields,
  mandatory missing fields, extra spec-only fields, and coverage percentages.
- `summary.scopeBreakdown[]` — planning slices derived from the raw API report:
  `desiredState` for the spec + `*Details` subset, `statusParity` for status-surface
  mappings, `broadening` for tracked spec mappings against non-`*Details` SDK
  structs, and `responseBody` for the preserved response-body special cases.
  Each scope also includes its own per-service rollup.
- `summary.topOffenders` — the worst individual spec-to-SDK mappings for missing
  fields, mandatory missing fields, and extra spec-only fields, including field names.
- `snapshot.root` — present only when the snapshot is retained.

Typical before/after comparison flow:

```bash
go run ./cmd/osok-generated-coverage --all > before.json
# make generator changes
go run ./cmd/osok-generated-coverage --all > after.json

jq '.summary.aggregate' before.json
jq '.summary.aggregate' after.json
jq '.summary.scopeBreakdown' after.json
jq '.summary.topOffenders.missingFields[:5]' after.json
```

Direct `go run` invocations use the same snapshot behavior as the Make targets.
If you need to inspect the retained snapshot from a direct CLI run, keep it
explicitly and read the path back from the JSON report:

```bash
go run ./cmd/osok-generated-coverage --all --keep-snapshot > report.json
jq -r '.snapshot.root' report.json
```

## Generated Coverage Gate

The checked-in baseline for the generated-output workflow lives at:

```text
internal/generator/config/generated_coverage_baseline.json
```

Use the gate target when you want CI-style protection against regressions:

```bash
# Fail if generated coverage regresses compared to the checked-in baseline
make generated-coverage-gate

# Keep the full summary and raw validator JSON for inspection on failure
make generated-coverage-gate \
  GENERATED_COVERAGE_REPORT=/tmp/generated-coverage-gate.json \
  GENERATED_COVERAGE_VALIDATOR_JSON=/tmp/generated-coverage-gate-validator.json
```

`generated-coverage-gate` and `generated-coverage-baseline` always operate on
the full configured service set. Use `make generated-coverage-report
GENERATED_COVERAGE_SERVICE=<service>` for targeted local inspection, but keep
the checked-in baseline scoped to `--all`.

The snapshot workflow always restores the checked-in non-generated API,
controller, and service-manager companion Go files for the selected services
before the validator runs, so the gate measures the same mixed generated and
handwritten contract as the checked-in tree.

The gate checks two classes of failure:

- **Scope changes** — service inventory, spec counts, or mapping counts changed from the baseline.
- **Metric regressions** — tracked mappings dropped, untracked mappings increased, missing or extra fields increased, or coverage percentages decreased in aggregate or for a specific service.

When the gate fails, the command groups those failures separately and includes
current top-offender summaries so you can jump straight to the worst mappings.
If you also set `GENERATED_COVERAGE_REPORT` or `GENERATED_COVERAGE_VALIDATOR_JSON`,
the failure output points to those files directly.

When a scope change or baseline shift is intentional, refresh the baseline explicitly:

```bash
make generated-coverage-baseline
```

That target reruns the generated snapshot report and overwrites
`internal/generator/config/generated_coverage_baseline.json`. Treat the baseline
update as part of the reviewed change: if the metrics moved for a good reason,
the baseline diff documents the new expected floor for future regressions.

## Generated Runtime Gate

Use the runtime gate when generator work changes controller, service-manager, or
registration templates. By default it uses
`internal/generator/config/services.yaml`, creates an isolated snapshot repo,
generates runtime-enabled outputs there, restores the checked-in non-generated
API, controller, and service-manager companion Go files for the selected
services, regenerates deepcopy code for the selected API groups, verifies that
each selected `internal/registrations/<group>_generated.go` output exists, and
then compile-checks the generated controller, service-manager, and registration
packages.

When future rollout work needs a pre-promotion snapshot, override
`GENERATED_RUNTIME_CONFIG` or `--config` with an alternate config explicitly.

```bash
# Write a runtime validation summary JSON
make generated-runtime-report

# CI-style gate for the full runtime config
make generated-runtime-gate

# Keep the snapshot for inspection
make generated-runtime-gate \
  GENERATED_RUNTIME_SNAPSHOT_DIR=/tmp/osok-generated-runtime \
  GENERATED_RUNTIME_REPORT=/tmp/generated-runtime-report.json

# Direct CLI usage for all configured services
go run ./cmd/osok-generated-runtime-check --all

# Direct CLI usage for one service with a retained snapshot
go run ./cmd/osok-generated-runtime-check \
  --service mysql \
  --snapshot-dir /tmp/osok-generated-runtime-mysql
```

Common variables and flags:

| Variable / Flag | Default | Purpose |
| --- | --- | --- |
| `GENERATED_RUNTIME_CONFIG` / `--config` | `internal/generator/config/services.yaml` | Generator config used for the runtime snapshot. Override for alternate rollout configs. |
| `GENERATED_RUNTIME_SERVICE` / `--service` | empty | Run the runtime check for one configured service. |
| `--all` | false | Validate all services in the selected runtime config. |
| `GENERATED_RUNTIME_REPORT` / `--report-out` | `generated-runtime-report.json` for the Makefile target | Write the generated runtime summary JSON. |
| `GENERATED_RUNTIME_SNAPSHOT_DIR` / `--snapshot-dir` | empty | Keep the runtime snapshot at a specific path instead of using an auto-cleaned temp dir. |
| `GENERATED_RUNTIME_KEEP_SNAPSHOT` / `--keep-snapshot` | empty / false | Keep an automatically created temp snapshot after a successful run. |
| `--controller-gen` | `<repo>/bin/controller-gen` | `controller-gen` binary used to regenerate deepcopy code inside the snapshot. |

The runtime summary JSON includes:

- `build.controllerPackages` — generated controller packages compiled from the snapshot.
- `build.serviceManagerPackages` — generated service-manager packages compiled from the snapshot.
- `build.registrationPackages` — the shared registration package compiled after the selected `<group>_generated.go` outputs are present in the snapshot.
- `snapshot.root` — present only when the snapshot is retained.

The runtime snapshot always restores the checked-in non-generated API,
controller, and service-manager companion Go files for the selected services
before compiling the generated packages, so the gate checks the current mixed
runtime seams rather than a fully isolated generated tree.

`generated-runtime-gate` does not use a baseline file. The build either
compiles or it fails, which makes it a straightforward regression tripwire for
runtime template changes.

## Registry Generation Workflow

The validator registries can now be generated automatically:

```bash
# Preview only (no file changes)
go run ./hack/update_validator_registries.go

# Regenerate and write both registries
go run ./hack/update_validator_registries.go --write
```

`internal/generator/config/services.yaml` is the source-of-truth inventory for
this workflow. The generator scans the configured API groups/versions from that
file, preserves existing explicit SDK mappings, and keeps specs with no inferred
SDK payloads in the API registry so coverage reports can mark them as
`untracked` instead of silently omitting them.

This script updates:

- `internal/validator/apispec/registry.go`
- `internal/validator/sdk/registry.go`

Recommended workflow when APIs/SDK change:

```bash
go run ./hack/update_validator_registries.go --write
go test ./hack ./internal/validator/apispec ./internal/validator/sdk
make schema-validator
```

## Interpreting the Controller Report

Each SDK struct shows a coverage percentage based on the tracked fields. Field statuses:

- `used` — The controller sets this field (references are listed).
- `deprecated` / `read_only` — Marked by the SDK; usually safe to skip.
- `intentionally_omitted`, `future_consideration`, `potential_gap` — Assigned by the allowlist to document why a field isn’t used yet (mandatory fields should generally be `potential_gap` if missing).
- `unclassified` — No allowlist entry *and* the controller doesn’t set it. If you downgrade a previously used field, this is what triggers `--fail-on-new-actionable`.

If you run without a baseline, missing fields are highlighted but the command still exits 0. Add a baseline + `--fail-on-new-actionable` in CI to break builds when new gaps appear.

## Interpreting the API Report

For each spec ↔ SDK pairing you’ll see three lists:

- **Present fields** — Spec fields exposed to users. They show up as `used` (mandatory fields are flagged).
- **Missing fields** — SDK fields not visible in the spec. Review these to decide whether to add them or classify them in the allowlist (`future_consideration`, `intentionally_omitted`, etc.).
- **API-only fields** — Spec fields with no matching SDK field. Useful for catching spec-only metadata or typos.
- **Untracked mappings** — Generated API specs that exist in `api/<group>/<version>`
  but still have no mapped SDK payloads in `internal/validator/apispec/registry.go`.
  These rows render as `untracked` so aggregate coverage no longer drops them on
  the floor.

This report uses the SDK’s `mandatory` tags to default missing mandatory fields to `potential_gap` and optional ones to `future_consideration`. You can use the same allowlist to adjust those statuses.

## SDK Upgrade Diff

`--upgrade-from/--upgrade-to` compares two SDK versions (e.g. `v65.61.1` → `v65.104.0`) and shows:

- Added/removed/changed fields for each tracked struct (with controller usage if you provide `--provider-path`).
- Draft allowlist suggestions for new fields (mandatory ones default to `potential_gap`, optional to `future_consideration`).
- Optional service operation differences if you add operation targets in `internal/validator/sdk/registry.go`.

This mode ignores baseline/allowlist flags; it’s a standalone helper for SDK bumps.

## Where the Metadata Lives

- `internal/validator/sdk/registry.go` lists the SDK structs we track per service.
- `internal/validator/apispec/registry.go` maps each CRD spec type to relevant SDK structs, including explicit `untracked` entries when a generated spec still lacks an SDK mapping.
- `hack/update_validator_registries.go` generates/reorders both registry files from `internal/generator/config/services.yaml`, generated API specs, and vendored SDK types.
- `validator_allowlist.yaml` (repo root) documents intentional gaps and feeds both controller and API reports.

## Troubleshooting & Tips

- Missing allowlist? The CLI will still run, but all unused fields show as `unclassified` (controller) or default to `future_consideration` / `potential_gap` (API). Add the allowlist to make the output more meaningful.
- If a new generated resource shows up as `untracked`, rerun `go run ./hack/update_validator_registries.go --write`. If it is still untracked after regeneration, add or adjust the `SDKStructs` mapping in `internal/validator/apispec/registry.go` so the validator knows which payloads to compare.
- `--fail-on-new-actionable` does nothing without `--baseline`: it needs a “before” snapshot.
- JSON output returns raw objects; use tools like `jq` to explore the report:
  ```bash
  go run ./cmd/osok-schema-validator --format json | jq '.api.structs[] | select(.sdkStruct=="streaming.CreateStreamDetails")'
  ```
- Keep the allowlist in version control so everyone shares the same classification.

## Summary

Use the validator whenever you:

- Touch controller logic (to ensure you don’t drop SDK fields).
- Expand the CRD surface (API report shows what’s still missing).
- Upgrade the OCI SDK (upgrade mode lists new fields/operations and suggests allowlist updates).

With baselines and the fail flag wired into CI, the combination of controller and API coverage catches regressions early and documents intentional gaps, keeping OSOK aligned with OCI.

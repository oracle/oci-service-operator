# OSOK Validator Guide

This document describes the tooling under `pkg/validator` and how to use the `osok-schema-validator` CLI to keep the OSOK controllers and CRDs aligned with the Oracle Cloud Infrastructure (OCI) Go SDK.

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

2. **API coverage** — It walks the CRD specs in grouped API packages (`api/database/v1beta1`, `api/mysql/v1beta1`, `api/streaming/v1beta1`), maps them to the corresponding SDK request structs, and reports three buckets:
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
| `--format` | `table` (default), `markdown`, or `json`. |
| `--baseline` | Load a previous controller report when diffing. |
| `--write-baseline` | Save the current controller report as JSON. |
| `--fail-on-new-actionable` | Exit non-zero if new controller gaps appear vs. baseline. |
| `--upgrade-from` / `--upgrade-to` | Run the SDK upgrade diff instead of coverage reports. |

## Package Layout

```text
pkg/validator/
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

This report uses the SDK’s `mandatory` tags to default missing mandatory fields to `potential_gap` and optional ones to `future_consideration`. You can use the same allowlist to adjust those statuses.

## SDK Upgrade Diff

`--upgrade-from/--upgrade-to` compares two SDK versions (e.g. `v65.61.1` → `v65.104.0`) and shows:

- Added/removed/changed fields for each tracked struct (with controller usage if you provide `--provider-path`).
- Draft allowlist suggestions for new fields (mandatory ones default to `potential_gap`, optional to `future_consideration`).
- Optional service operation differences if you add operation targets in `pkg/validator/sdk/registry.go`.

This mode ignores baseline/allowlist flags; it’s a standalone helper for SDK bumps.

## Where the Metadata Lives

- `pkg/validator/sdk/registry.go` lists the SDK structs we track (create/update payloads for each service). When you add a new CRD/controller, add its create/update structs here.
- `pkg/validator/apispec/registry.go` maps each CRD spec type to the relevant SDK structs so the API validator knows what to compare.
- `validator_allowlist.yaml` (repo root) documents intentional gaps and feeds both controller and API reports.

## Troubleshooting & Tips

- Missing allowlist? The CLI will still run, but all unused fields show as `unclassified` (controller) or default to `future_consideration` / `potential_gap` (API). Add the allowlist to make the output more meaningful.
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

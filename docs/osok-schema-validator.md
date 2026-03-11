# OSOK Schema Validator

## Purpose

`osok-schema-validator` compares the OSOK provider's SDK usage against the capabilities exposed by the OCI Go SDK. The tool flags fields that the provider never sets as well as new SDK fields that require attention when the dependency is upgraded.

## Objectives

- **Provider-driven**: Traverse the OSOK service manager packages and controllers to discover how OCI request structs are instantiated and mutated.
- **OCI SDK awareness**: Reflect over a curated list of SDK struct targets and capture metadata such as mandatory flags, documentation, and read-only semantics.
- **Allowlist support**: Permit intentional deviations such as deprecated or server-managed fields via a YAML allowlist.
- **Maklib integration**: Provide a `schema.validate` target so CI can run the validator before publishing new schemas.

## Inputs & Outputs

| Input | Description |
| --- | --- |
| Provider root | Absolute path to the OSOK repository. |
| Allowlist | Optional YAML file classifying known gaps. |

| Output | Description |
| --- | --- |
| Go structs (`diff.Report`) | Coverage summary per SDK struct. |
| Text renderer | Human-readable summary for consoles/logs. |
| Exit code | Non-zero when new potential gaps appear (suitable for CI). |

## Package Layout

```
pkg/validator/
  allowlist/   # YAML loader for field classifications
  config/      # Option validation
  provider/    # AST-based analyzer for OSOK provider usage
  sdk/         # Reflection-based OCI SDK analyzer
  diff/        # Comparison engine and coverage stats
  report/      # Text rendering helpers
  run.go       # Orchestration entrypoint
cmd/osok-schema-validator/
  main.go      # CLI wrapper
```

## CLI Usage

Run the validator from the repository root:

```
go run ./cmd/osok-schema-validator \
  --provider-path . \
  --allowlist validator_allowlist.yaml \
  --format table|markdown|json \
  --baseline baseline.json \
  --write-baseline baseline.json \
  --fail-on-new-actionable
```

- `--format` controls the stdout rendering (table by default).  
- `--baseline` loads a previous report and annotates fields with their prior status.  
- `--write-baseline` writes the current report to JSON regardless of the stdout format.  
- `--fail-on-new-actionable` returns exit code 1 when a field transitions into `unclassified` or `potential_gap`.

### SDK Upgrade Diffs

Use the upgrade helper to compare two OCI Go SDK releases and identify new fields that may require OSOK support:

```
go run ./cmd/osok-schema-validator \
  --upgrade-from v65.61.1 \
  --upgrade-to v65.104.0 \
  --provider-path . \
  --format markdown
```

The upgrade report:

- Lists added/removed/changed fields for every tracked SDK struct.
- Indicates whether the current OSOK controllers already reference each field.
- Emits draft allowlist suggestions for new fields (mandatory fields default to `potential_gap`, optional fields to `future_consideration`).

Both versions must be present in your module cache (`$GOMODCACHE/github.com/oracle/oci-go-sdk/v65@<version>`).

### API Spec Coverage

Every run now also compares the OSOK CRD specs against the curated SDK structs. Missing SDK fields are grouped by spec (`AutonomousDatabases`, `MySqlDbSystem`, `Stream`) and annotated with the allowlist status so you can see which fields are intentionally omitted versus genuine gaps. The summary appears after the controller coverage in table/markdown formats and under the `api` key in JSON output.

You can reuse the existing allowlist to classify API omissions (for example, `intentionally_omitted`, `future_consideration`, `potential_gap`).

## Maklib Integration

A Maklib fragment (`makelib/schema.validate.mk`) exposes `make schema.validate`, which runs `go test ./pkg/validator/...`. CI should set `PROVIDER_PATH`/`GOFLAGS` as needed so the validator can locate the OSOK repository and vendor cache.

## Next Steps

1. Expand the `sdk` registry with additional OCI services as OSOK adds new coverage.
2. Populate an allowlist capturing fields intentionally omitted today.
3. Wire the CLI into automated jobs (pre-commit or CI) to block regressions.

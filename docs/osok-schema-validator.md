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

## Maklib Integration

A Maklib fragment (`makelib/schema.validate.mk`) exposes `make schema.validate`, which runs `go test ./pkg/validator/...`. CI should set `PROVIDER_PATH`/`GOFLAGS` as needed so the validator can locate the OSOK repository and vendor cache.

## Next Steps

1. Expand the `sdk` registry with additional OCI services as OSOK adds new coverage.
2. Populate an allowlist capturing fields intentionally omitted today.
3. Wire the CLI into automated jobs (pre-commit or CI) to block regressions.

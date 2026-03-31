Implemented the remaining generator-contract cleanup for `admin-osok#3`.

Changes:
- Removed the checked-in `compatibility.existingKinds` contract from `internal/generator/config/services.yaml`, the generator config model, and the discovery pipeline.
- Added a normal per-resource `generation.resources[].sdkName` mapping so renamed published kinds such as `database/AutonomousDatabases` and `mysql/MySqlDbSystem` stay in standard generator config without compatibility-only branches.
- Generalized `--preserve-existing-spec-surface` to preserve any existing checked-in sample/package artifacts for selected services, and updated generator docs/help text/tests to reflect the new contract and to fail if parity or compatibility config returns.

Validation:
- `GOCACHE=$PWD/.cache/go-build go test ./internal/generator`
- `GOCACHE=$PWD/.cache/go-build go test ./internal/generatorcmd`
- `GOCACHE=$PWD/.cache/go-build go test ./cmd/osok-generated-coverage`
- `GOCACHE=$PWD/.cache/go-build go test ./hack`

Outcome:
- The checked-in generator contract now encodes `database`, `mysql`, and `streaming` as normal generator-owned services without parity/compatibility settings, while the preserve-existing-spec-surface workflow still protects the checked-in surfaces that must remain stable.

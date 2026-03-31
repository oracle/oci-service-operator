Updated `packages/README.md` to remove the last stale compatibility-style wording from the checked-in package contract.

The README now describes `database`, `mysql`, and `streaming` as normal `controller-backed` services whose rollout metadata is declared directly in `internal/generator/config/services.yaml`, and it documents `--preserve-existing-spec-surface` as the generic checked-in artifact preservation flow for naming and observed-state overrides.

Validation:
- `rg -n 'compatibility-locked|existingKinds|parity-only|osok-api-generator|api-generate|api-refresh' packages/README.md docs/api-generator-contract.md internal/generator/config/services.yaml Makefile cmd internal/generator -S`
- `git diff --check -- packages/README.md`

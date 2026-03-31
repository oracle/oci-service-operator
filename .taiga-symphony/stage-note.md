Implemented the review follow-up for `admin-osok#3`.

Changes:
- Replaced the file-scoped `nolint` directives in `internal/generator/generator_test.go`, `internal/generator/config_test.go`, and `internal/generator/compatibility_surface_test.go` with shared helper-based assertions and smaller test-specific helpers so the generator contract tests stay explicit without broad waivers.
- Refactored the generated-file AST/comment normalization helpers in `internal/generator/generator_test.go` to keep the contract-test utilities under the repository complexity thresholds.

Validation:
- `GOCACHE=$PWD/.cache/go-build go test ./internal/generator`
- `GOCACHE=$PWD/.cache/go-build GOLANGCI_LINT_CACHE=$PWD/.cache/golangci-lint make lint`

Outcome:
- The review-blocking file-scoped generator test suppressions are removed, the generator contract tests still pass, and the branch is lint-clean again.

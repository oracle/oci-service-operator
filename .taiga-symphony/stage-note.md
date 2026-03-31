Implemented the remaining lint-fix follow-up for `admin-osok#3`.

Changes:
- Refactored `internal/generator/discovery.go` to split resource discovery, runtime binding, request field synthesis, and formal attachment into smaller helpers, and removed the two dead legacy field-merge helpers.
- Simplified helper-type name selection in `internal/generator/type_synthesis.go` so scoped helper naming keeps the same behavior while staying under the repository complexity thresholds.

Validation:
- `XDG_CACHE_HOME=$PWD/.cache HOME=$PWD/.tmp/home GOCACHE=$PWD/.cache/go-build GOLANGCI_LINT_CACHE=$PWD/.cache/golangci-lint make lint`
- `XDG_CACHE_HOME=$PWD/.cache HOME=$PWD/.tmp/home GOCACHE=$PWD/.cache/go-build go test ./internal/generator`
- `XDG_CACHE_HOME=$PWD/.cache HOME=$PWD/.tmp/home GOCACHE=$PWD/.cache/go-build go test ./internal/generatorcmd ./hack ./cmd/osok-generated-coverage`

Outcome:
- The remaining seven lint failures in `internal/generator/discovery.go` and `internal/generator/type_synthesis.go` are resolved, and targeted generator/tooling validation passes.

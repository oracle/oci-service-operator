Refactored `internal/generator/names.go` to split `singularize` and `splitCamel` into smaller helpers, clearing the last `gocyclo` violations in the generator cleanup branch without adding broad lint suppressions.

Added focused `internal/generator/names_test.go` coverage for camel-token splitting and lower-camel conversion so acronym and digit-boundary behavior stays locked down with the helper-based implementation.

Validation:
- `GOCACHE=/tmp/admin-osok_3-go-build GOLANGCI_LINT_CACHE=/tmp/admin-osok_3-golangci-lint make lint`
- `GOCACHE=/tmp/admin-osok_3-go-build go test ./internal/generator/...`

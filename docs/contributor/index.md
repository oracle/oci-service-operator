# Contributor Docs

This section is for maintainers and contributors working on OSOK generation,
validation, and repository-owned documentation flows.

## Maintainer References

- [Generator Contract](../api-generator-contract.md)
- [Validator Guide](../validator-guide.md)
- [GitHub Pages Handoff](github-pages-handoff.md)

## Local Docs Commands

- `make docs-generate`
- `make docs-build`
- `make docs-serve`
- `make docs-verify`

Phase 1 keeps missing public description coverage as warnings. When the public
spec-field backlog is resolved, set
`DOCS_VERIFY_STRICT_PUBLIC_DESCRIPTIONS=true` in CI to promote those warnings to
hard failures.

The main customer quickstart remains the [User Guide](../user-guide.md).

# LINT Stage Guidance

- This is a verification gate. Do not edit or commit code here.
- Run `make lint`.
- The lint gate enforces cyclomatic complexity `10` and cognitive complexity `15`.
- Prefer refactoring over suppressions. Treat new `nolint` directives as suspect unless they are narrow, justified, and story-specific.
- The repo lint config is branch-oriented and excludes vendor and generated code; do not treat excluded generated files as a reason to skip linting handwritten code.

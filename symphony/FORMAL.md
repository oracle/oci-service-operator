# FORMAL Stage Guidance

- This is a verification gate. Do not edit or commit code here.
- Treat `formal/` as the design source of truth for reconciliation semantics.
- When controller or service-manager behavior changes, require matching TLA+ updates and matching UML artifacts: PlantUML `.puml` plus rendered `.svg` in the relevant shared or controller-local diagram directories.
- Run `make formal-verify`.
- If the branch changed diagram-affecting formal inputs, require that implementation already reran `make formal-diagrams`; if artifacts are stale or missing, return `needs_changes` instead of fixing them here.
- Call out missing `spec.cfg`, `logic-gaps.md`, diagram, or shared-contract updates precisely in the stage note.

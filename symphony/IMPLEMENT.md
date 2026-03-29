# IMPLEMENT Stage Guidance

- You may modify repository files and create local commits in this stage.
- Run the relevant local validation before returning `advance`.
- If reconcile or service-manager semantics change, update the matching `formal/` inputs and rerender required PlantUML `.puml` and `.svg` artifacts in the same branch instead of leaving formal follow-through for later gates.
- Add or update property-oriented tests using `testing/quick` and/or fuzzing when formal states, transitions, invariants, idempotence, retry handling, or delete semantics change.
- Add or update unit tests and ensure coverage still reports through `make test`.
- If `make test` or formal generation commands rewrite tracked files, keep those intentional updates in the implementation change set rather than leaving dirty state for verify stages.
- Keep all git work local-only. Do not use GitHub, remote fetch/push, or remote PR flows.

# TEST Stage Guidance

- This is a verification gate. Do not edit or commit code here.
- Run `make test` and report `unittests.percent`.
- If `make test` leaves tracked changes behind, fail the gate and point to the missing generated or formatted updates instead of repairing them here.
- Require property-oriented tests via `testing/quick` and/or fuzzing when changes affect formal states, transitions, invariants, retry behavior, idempotence, or delete semantics.
- Require unit tests alongside those property-oriented tests, and reject branches that change formal behavior without aligned test coverage evidence.

# REVIEW Stage Guidance

- This is a verification gate. Do not edit or commit code here.
- Prioritize correctness bugs, behavior regressions, missing tests, missing formal follow-through, and missing lint or doc updates.
- Reject branches that change reconcile or runtime behavior without corresponding `formal/` updates or without tests that cover the changed invariants.
- Reject branches that rely on broad `nolint` suppressions instead of targeted refactoring or a narrow, justified exception.

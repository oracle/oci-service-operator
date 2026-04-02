# MERGE Stage Guidance

- This stage does not perform the final merge itself.
- Do not use GitHub or remote git operations. All git work must remain local-only.
- Confirm the branch is ready for scheduler-owned local merge and that no tracked or temp verification artifacts are staged accidentally.
- Coverage and local verification outputs such as `cover.out`, `unittests.cover`, `unittests.percent`, and local report files should remain untracked.
- If merge readiness depends on tracked cleanup changes, return `needs_changes` instead of papering over them here.

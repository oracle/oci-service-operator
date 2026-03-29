# Taiga-Symphony Guidance

This repository intentionally does not track `WORKFLOW.md`.

Keep the taiga-symphony workflow file local and untracked because it will
carry secrets or other machine-local configuration. The root `.gitignore`
already ignores `WORKFLOW.md` and `.taiga-symphony/`.

When the local workflow uses the standard upstream stage names below,
taiga-symphony will auto-load the matching guidance file from this directory:

- `plan` -> `symphony/PLAN.md`
- `implement` -> `symphony/IMPLEMENT.md`
- `review` -> `symphony/REVIEW.md`
- `formal` -> `symphony/FORMAL.md`
- `lint` -> `symphony/LINT.md`
- `test` -> `symphony/TEST.md`
- `security` -> `symphony/SECURITY.md`
- `e2e` -> `symphony/E2E.md`
- `doc` -> `symphony/DOC.md`
- `merge` -> `symphony/MERGE.md`

If a local workflow uses different stage names, set
`stages[].guidance_file` in that untracked workflow to point at the desired
relative file path inside this repository.

All git work must remain local-only. Do not use GitHub, remote fetch/push, or
other remote git interactions from taiga-symphony stages.

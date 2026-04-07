# GitHub Pages Handoff

This repository now carries the checked-in MkDocs configuration, generated
reference pages, and GitHub Actions workflows needed to publish the customer
docs site. The remaining public GitHub steps are intentionally manual and are
out of scope for local issue execution.

## Post-Merge GitHub Steps

1. Confirm `Verify Docs` passes on the default branch after the docs changes
   merge.
2. In the public GitHub repository settings, enable GitHub Pages with
   `GitHub Actions` as the build and deployment source.
3. Re-run `Publish Docs Pages` on the default branch if the first post-merge
   push landed before Pages was enabled.
4. Verify the published site loads at
   `https://oracle.github.io/oci-service-operator/` and that the main nav,
   Supported Resources, API Reference, and User Guide pages render correctly.
5. If the public repository owner or repo name differs from the expected Pages
   path, update `mkdocs.yml` `site_url` before the publish step is repeated.

## Local Maintainer Commands

- `make docs-generate` refreshes checked-in `docs/reference/` outputs from repo
  metadata and CRD schemas.
- `make docs-build` renders the MkDocs site locally under `site/`.
- `make docs-serve` starts a local preview server.
- `make docs-verify` runs CRD regeneration, generated-doc drift checks, MkDocs
  strict build, rendered-link validation, and description coverage checks.

Phase 1 keeps missing public description coverage as warnings. When the public
spec-field description backlog is ready for enforcement, set
`DOCS_VERIFY_STRICT_PUBLIC_DESCRIPTIONS=true` in CI.

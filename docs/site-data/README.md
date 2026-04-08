# Site Data Contract

`docs/site-data/` is the hand-maintained docs input layer for the planned
`cmd/sitegen` generator. It keeps customer-facing labels, summaries, visibility
flags, and release history in checked-in data files instead of scattering that
state across Markdown pages.

## Ownership

- `catalog.yaml` stores customer-facing package metadata and optional per-kind
  copy overrides that should not be inferred from rollout state alone.
- `releases/*.yaml` stores versioned package release history. A manifest may
  intentionally list only a subset of packages so partial releases remain
  representable.
- `internal/generator/config/services.yaml` remains the default source of truth
  for generated service inventory, group versions, and rollout posture.
- `packages/*/metadata.env` remains the source of truth for package identity,
  namespaces, and split-package CRD filters.
- `.github/workflows/publish-service-packages.yml` remains the source of truth
  for GHCR image naming and the default publish batch.
- Package-only exceptions that are not currently modeled in `services.yaml`,
  such as `apigateway`, still belong in this checked-in site-data layer so the
  public docs can reflect the actual package surface in the repo.

## Catalog Fields

- `schemaVersion`: schema marker for the file format.
- `packages[].package`: package directory name under `packages/`.
- `packages[].displayName`: customer-facing package label.
- `packages[].summary`: short package summary used by generated overview pages.
- `packages[].supportStatus`: customer-facing support label such as `preview`
  or `ga`.
- `packages[].customerVisible`: whether the package should appear in generated
  public docs before a release manifest is considered.
- `packages[].guidePath`: repo-root-relative path to the primary checked-in
  guide for that package.
- `packages[].packageNotes[]`: optional customer-facing notes that clarify
  split packages, focused bundles, or rollout exceptions.
- `packages[].kindOverrides[]`: optional `(group, kind)` overrides for summary
  or description text when generated inventory needs repo-authored copy.

## Release Manifest Fields

- `schemaVersion`: schema marker for the file format.
- `version`: released package version recorded by the manifest.
- `publishedAt`: RFC3339 timestamp chosen as the checked-in release anchor for
  that version.
- `commit`: git commit SHA associated with that checked-in release anchor.
- `notes[]`: optional maintainer notes about partial releases or placeholder
  behavior.
- `packages[].package`: package directory name under `packages/`.
- `packages[].controllerImage`: controller image reference for that package and
  version.
- `packages[].bundleImage`: OLM bundle image reference for that package and
  version.
- `packages[].groups[]`: API groups and versions exposed by that package.
- `packages[].groups[].kinds[]`: top-level kinds exposed for that group in that
  package version.

## Update Rules

1. Update `packages/*/metadata.env` or `internal/generator/config/services.yaml`
   first when package identity or inventory changes.
2. Update `catalog.yaml` when customer-facing labels, summaries, guide links,
   or visibility rules change.
3. Add or update `releases/<version>.yaml` when a package version is published
   or when the checked-in release history needs correction.
4. Do not infer released versions from git tags alone. The release manifest is
   the checked-in source of truth for package publication history.

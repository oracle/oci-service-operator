This directory contains per-group packaging scaffolding.

Each subdirectory under `packages/` stages a future package install overlay for one
service-owned API group while the shared monolith remains authoritative.

Expected layout:

- `metadata.env`: package metadata used by the Makefile and `hack/package.sh`.
- `install/`: kustomize overlay for the installable manifests for the group.
- `install/generated/`: generated group-specific CRDs and manager RBAC.

Current workflow:

1. `make package-generate GROUP=<group>` generates CRDs and controller RBAC into
   `packages/<group>/install/generated/`.
2. `make package-install GROUP=<group>` renders a single install YAML into
   `dist/packages/<group>/install.yaml`.

Out of scope for this scaffold:

- Per-group OLM bundles
- Per-group bundle images
- Per-group catalogs

Those remain explicit follow-on work after the shared monolith package layout is proven.

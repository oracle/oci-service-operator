This directory contains per-group packaging scaffolding.

Each subdirectory under `packages/` stages a future package install overlay for one
service-owned API group while the shared monolith remains authoritative.

Expected layout:

- `metadata.env`: package metadata used by the Makefile and `hack/package.sh`.
- `install/`: kustomize overlay for the installable manifests for the group.
- `install/generated/`: generated group-specific CRDs and, for controller-backed
  groups, manager RBAC.

Package profiles:

- `controller-backed`: used by the current `database`, `mysql`, and `streaming`
  groups, which already have hand-maintained controllers wired into the shared
  manager.
- `crd-only`: used by generator-first API groups that ship CRDs and samples
  before controller support exists.

Current workflow:

1. `make api-generate API_SERVICE=<service>` or `make api-generate API_ALL=true`
   writes generated API packages, sample manifests, sample kustomization, and
   per-group package scaffolding from
   `internal/generator/config/services.yaml`.
2. `make api-refresh ...` runs the API generator first, then refreshes
   `zz_generated.deepcopy.go` and `config/crd/` artifacts with
   `controller-gen`.
3. `make package-generate GROUP=<group>` generates CRDs and optional controller
   RBAC into `packages/<group>/install/generated/`.
4. `make package-install GROUP=<group>` renders a single install YAML into
   `dist/packages/<group>/install.yaml` for either package profile.

Package profile behavior:

- `controller-backed` overlays include generated CRDs, generated controller
  RBAC, and the shared manager install overlay.
- `crd-only` overlays render only generated CRDs until controller support
  exists for that group.

Out of scope for this scaffold:

- Per-group OLM bundles
- Per-group bundle images
- Per-group catalogs

Those remain explicit follow-on work after the shared monolith package layout is proven.

For the full generator-owned package and API contract, see
`docs/api-generator-contract.md`.

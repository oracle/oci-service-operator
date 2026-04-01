This directory contains per-group packaging scaffolding.

Each subdirectory under `packages/` stages an install overlay for one
service-owned API group while the shared monolith remains authoritative.
Runtime rollout for controllers, service managers, and registrations is declared
in `internal/generator/config/services.yaml` under each service's `generation`
block.

Expected layout:

- `metadata.env`: package metadata used by the Makefile and `hack/package.sh`
- `install/`: kustomize overlay for the installable manifests for the group
- `install/generated/`: generated group-specific CRDs and, for
  `controller-backed` groups, manager RBAC

Package profiles:

- `controller-backed`: used by groups that participate in the shared manager
  install. Every service in the checked-in `services.yaml` currently uses this
  profile, even when a group still mixes generated and handwritten runtime
  seams during rollout.
- `crd-only`: reserved for staged or alternate configs that need generated CRDs
  and samples before runtime rollout is enabled. The checked-in config does not
  currently use this profile.

Current workflow:

1. `go run ./cmd/generator --config internal/generator/config/services.yaml --service <service>`
   or `go run ./cmd/generator --config internal/generator/config/services.yaml --all`
   writes generated API packages, sample manifests, sample kustomization,
   per-group package scaffolding, and the configured controller,
   service-manager, and registration outputs from
   `internal/generator/config/services.yaml`.
2. Run `make generator-refresh GENERATOR_SERVICE=<service>` or
   `make generator-refresh GENERATOR_ALL=true` when the same refresh also needs
   `zz_generated.deepcopy.go` and `config/crd/` artifacts updated.
3. `make package-generate GROUP=<group>` generates CRDs and optional controller
   RBAC into `packages/<group>/install/generated/`.
4. `make package-install GROUP=<group>` renders a single install YAML into
   `dist/packages/<group>/install.yaml` for either package profile.

Runtime rollout defaults:

- Services without a `generation` block default to controller, service-manager,
  and registration rollout `none`, while webhook ownership remains `manual`.
- Services that need observed-state overrides, `package.extraResources`,
  package-path overrides, or webhook carve-outs keep those mappings in
  `services.yaml`; generator-owned spec/helper/sample/package artifacts always
  regenerate from the current v2 contract.

Package profile behavior:

- `controller-backed` overlays include generated CRDs, generated controller
  RBAC, and the shared manager install overlay for every checked-in service.
- `crd-only` overlays render only generated CRDs until a staged config opts the
  service into runtime rollout.

Runtime seam ownership stays explicit in checked-in files rather than being
implied by the package profile. The checked-in `database` and `mysql` runtime
surfaces are generator-owned, while `streaming/Stream` keeps the only
repo-authored companion through the endpoint-secret client layered onto the
generated service-manager path.

Package profile transitions:

- New services can start as `crd-only` in a staged or alternate config.
- Move a service to `controller-backed` only when shared-manager integration,
  validator coverage, and generated-runtime gates are ready.
- Runtime gates consume rollout metadata directly from `services.yaml`. Use
  `GENERATED_RUNTIME_CONFIG` to point them at an alternate config when staging
  future promotions.

Out of scope for this scaffold:

- Per-group OLM bundles
- Per-group bundle images
- Per-group catalogs
- Webhook generation
- Direct `main.go` edits

Those remain explicit follow-on work after the shared monolith package layout
is proven.

For the full generator-owned package and API contract, see
`docs/api-generator-contract.md`.

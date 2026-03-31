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
  profile.
- `crd-only`: reserved for staged or alternate configs that need generated CRDs
  and samples before runtime rollout is enabled. The checked-in config does not
  currently use this profile.

Current workflow:

1. `make generator-generate GENERATOR_SERVICE=<service>` or
   `make generator-generate GENERATOR_ALL=true` writes generated API packages,
   sample manifests, sample kustomization, per-group package scaffolding, and
   the configured controller, service-manager, and registration outputs from
   `internal/generator/config/services.yaml`.
2. `make generator-refresh ...` runs the generator first, then refreshes
   `zz_generated.deepcopy.go` and `config/crd/` artifacts with `controller-gen`.
3. `make package-generate GROUP=<group>` generates CRDs and optional controller
   RBAC into `packages/<group>/install/generated/`.
4. `make package-install GROUP=<group>` renders a single install YAML into
   `dist/packages/<group>/install.yaml` for either package profile.

Compatibility aliases:

- `make api-generate ...` forwards to `make generator-generate ...`
- `make api-refresh ...` forwards to `make generator-refresh ...`

Runtime rollout defaults:

- Services without a `generation` block default to controller, service-manager,
  and registration rollout `none`; webhook ownership stays with checked-in
  `*_webhook.go` files unless explicitly disabled.
- The checked-in config now enables generated controller, service-manager, and
  registration rollout for every service. Historical parity resources rely on
  resource-level package-path and webhook carve-outs instead of service-level
  runtime exceptions.

Package profile behavior:

- `controller-backed` overlays include generated CRDs, generated controller
  RBAC, and the shared manager install overlay for every checked-in service.
- `crd-only` overlays render only generated CRDs until a staged config opts the
  service into runtime rollout.

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

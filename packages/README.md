This directory contains per-group packaging scaffolding.

Each subdirectory under `packages/` stages a future package install overlay for one
service-owned API group while the shared monolith remains authoritative.
Runtime rollout for controllers, service managers, and generated registrations is
declared in `internal/generator/config/services.yaml` under each service's
`generation` block.

Expected layout:

- `metadata.env`: package metadata used by the Makefile and `hack/package.sh`.
- `install/`: kustomize overlay for the installable manifests for the group.
- `install/generated/`: generated group-specific CRDs and, for controller-backed
  groups, manager RBAC.

Package profiles:

- `controller-backed`: used by groups that already participate in the shared
  manager install. `database`, `mysql`, and `streaming` already live here with
  generated controller, service-manager, and registration outputs declared
  directly in `internal/generator/config/services.yaml`.
- `crd-only`: used by generator-first API groups that ship CRDs and samples
  before controller, service-manager, and registration rollout is enabled.

Being `controller-backed` does not require every runtime seam to stay manual.
`database` and `mysql` still retain explicit parity adapters, while
`streaming/Stream` now runs on the generated runtime path directly except for
the endpoint-secret companion.

Current workflow:

1. `go run ./cmd/generator --config internal/generator/config/services.yaml --service <service>`
   or `go run ./cmd/generator --config internal/generator/config/services.yaml --all`
   writes generated API packages, sample manifests, sample kustomization,
   per-group package scaffolding, and any opt-in controller, service-manager,
   or registration outputs from `internal/generator/config/services.yaml`.
2. If the regenerated APIs also need deepcopy and CRD refresh, run
   `make generate` and `make manifests` after the generator command.
3. `make package-generate GROUP=<group>` generates CRDs and optional controller
   RBAC into `packages/<group>/install/generated/`.
4. `make package-install GROUP=<group>` renders a single install YAML into
   `dist/packages/<group>/install.yaml` for either package profile.

Runtime rollout defaults:

- Services without a `generation` block default to controller, service-manager,
  and registration rollout `none`, while webhook ownership remains `manual`.
- Existing parity services keep their rollout metadata in `services.yaml`.
  `database`, `mysql`, and `streaming` are already `controller-backed`, with
  generated controller, service-manager, and registration rollout enabled.
  Remaining handwritten seams stay explicit in repo-owned files instead of
  being implied by the package profile.

Package profile behavior:

- `controller-backed` overlays include generated CRDs, generated controller
  RBAC, and the shared manager install overlay, regardless of whether the
  runtime pieces are currently manual or generated.
- `crd-only` overlays render only generated CRDs until controller support
  exists for that group.

Package profile transitions:

- New services stay `crd-only` until controller, service-manager, and
  registration rollout is explicitly enabled in `services.yaml`.
- A group only moves to `controller-backed` when shared manager integration is
  ready; the generator should consume generated registration surfaces rather
  than rewrite `main.go` directly.
- All currently generated services now carry their shared-manager rollout
  metadata directly in `services.yaml`. If a future service needs a staged
  pre-promotion runtime snapshot, use `GENERATED_RUNTIME_CONFIG` to point the
  runtime gate at an alternate config.

Out of scope for this scaffold:

- Per-group OLM bundles
- Per-group bundle images
- Per-group catalogs
- Webhook generation
- Direct `main.go` edits

Those remain explicit follow-on work after the shared monolith package layout is proven.

For the full generator-owned package and API contract, see
`docs/api-generator-contract.md`.

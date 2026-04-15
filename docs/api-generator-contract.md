# OSOK Generator Contract

This document defines the v2 source-of-truth mapping and output contract for the
OSOK generator epic (`oci-service-operator-9q0*`). The canonical user-facing
entrypoint is `./cmd/generator`. The contract now covers API, controller,
service-manager, and runtime-registration surfaces.

## Source of Truth

`internal/generator/config/services.yaml` is the only hand-maintained mapping
for OCI SDK services to OSOK API groups and generator rollout metadata.

Each service record defines:

| Field | Meaning |
| --- | --- |
| `service` | OCI Go SDK service package name. |
| `sdkPackage` | Full OCI Go SDK import path used by the generator pipeline. |
| `group` | OSOK API group name and directory segment under `api/`. |
| `version` | API version. Default is `v1beta1` unless a record overrides it. |
| `phase` | Current rollout bucket used by the generator epic. |
| `sampleOrder` | Optional deterministic ordering hint for generated sample entries. |
| `packageProfile` | Install posture for the group: `controller-backed` or `crd-only`. |
| `package.extraResources` | Optional extra package overlay resources to include in the generated install kustomization. |
| `selection.enabled` | Whether the service participates in the default active generator surface. |
| `selection.mode` | Default selection contract for the service: `all` or `explicit`. |
| `selection.includeKinds` | Optional non-empty kind list used only when `selection.mode=explicit`. |
| `async.strategy` | Optional service-level default for published async behavior: `none`, `lifecycle`, or `workrequest`. |
| `async.runtime` | Optional service-level default naming whether the active runtime owner is `generatedruntime` or a handwritten service package. |
| `async.formalClassification` | Optional service-level default that keeps `formal/` classification aligned with the checked-in async posture. |
| `formalSpec` | Optional controller slug from `formal/controller_manifest.tsv` when one formal row covers the service-level runtime contract. |
| `observedState.sdkAliases` | Optional observed-state SDK struct aliases keyed by the discovered SDK resource family when status synthesis must read a differently named response model. |
| `observedState.excludedFieldPaths` | Optional dot-separated observed-state field paths keyed by the discovered SDK resource family when sensitive or unsupported SDK fields must be omitted from generated status surfaces. |
| `generation.controller.strategy` | Service-wide controller rollout: `none`, `manual`, or `generated`. |
| `generation.serviceManager.strategy` | Service-wide service-manager rollout: `none`, `manual`, or `generated`. |
| `generation.registration.strategy` | Group-level runtime registration rollout: `none`, `manual`, or `generated`. |
| `generation.webhooks.strategy` | Webhook ownership seam: `manual` or `none`. |
| `generation.resources[]` | Per-kind overrides keyed by the current OSOK kind from the v2 contract. |
| `generation.resources[].formalSpec` | Optional per-kind controller slug from `formal/controller_manifest.tsv` when only selected resources are formally promoted. |
| `generation.resources[].async.strategy` | Optional per-kind async override when the selected kind's published behavior differs from the service default. |
| `generation.resources[].async.runtime` | Optional per-kind runtime owner classification, typically `generatedruntime` or `handwritten`. |
| `generation.resources[].async.formalClassification` | Optional per-kind formal async classification for the matching controller row. |
| `generation.resources[].controller.strategy` | Optional per-kind controller rollout override: `none`, `manual`, or `generated`. When omitted, the kind inherits the service-level controller strategy. |
| `generation.resources[].controller.maxConcurrentReconciles` | Optional controller concurrency override for one kind. |
| `generation.resources[].controller.extraRBACMarkers` | Optional non-default additional kubebuilder RBAC marker payloads for one kind. Generated controllers already include their API resource verbs plus `events create;patch`. |
| `generation.resources[].serviceManager.strategy` | Optional per-kind service-manager rollout override: `none`, `manual`, or `generated`. When omitted, the kind inherits the service-level service-manager strategy. |
| `generation.resources[].serviceManager.packagePath` | Optional existing package path relative to `pkg/servicemanager/` when a manual layout must be preserved. |
| `generation.resources[].serviceManager.needsCredentialClient` | Optional flag that threads credential-client plumbing into a generated service-manager seam when repo-authored secret-backed fields need it. |
| `generation.resources[].webhooks.strategy` | Optional per-kind webhook ownership seam: `manual` or `none`. When omitted, the kind inherits the service-level webhook strategy. |
| `generation.resources[].specFields` | Optional per-kind spec field overrides keyed by generated Go field name. Overrides may replace field type, tag, comments, or markers when the repo-authored v2 contract intentionally differs from the imported SDK surface. |
| `generation.resources[].statusFields` | Optional per-kind status field overrides keyed by generated Go field name. Overrides may replace or add repo-authored observed-state or status-mirror fields. |
| `generation.resources[].sample` | Optional per-kind sample override. `body` replaces the rendered sample wholesale, while `metadataName` and `spec` refine the generated defaults. |

Rules:

- Service-to-group mapping is 1:1.
- `group` stays equal to the OCI SDK package basename unless the mapping file
  explicitly says otherwise.
- `selection.enabled=false` keeps the service out of the default active
  surface, but the service remains addressable through explicit generator
  selectors such as `--service`.
- `selection.mode=all` requires an empty `selection.includeKinds`.
- `selection.mode=explicit` requires a non-empty `selection.includeKinds` list
  of current OSOK kinds.
- Enabled selected kinds must resolve to explicit async metadata either from
  service-level `async.*` defaults or resource-level
  `generation.resources[].async.*` overrides.
- Omitted `generation` fields default to controller, service-manager, and
  registration rollout `none`, with webhooks defaulting to `manual`.
- `generation.resources[].kind` uses the current OSOK kind from the v2
  contract.
- Resource-level rollout inherits the corresponding service-level strategy when
  the override omits `strategy`.
- `generation.resources[]` may keep selected published kinds API-only even when
  the rest of the service group uses generated runtime rollout.
- Service-specific rollout behavior and repo-authored field/sample overrides
  belong in the mapping file, not in hardcoded generator branches.
- Legacy overlay files and kind-remap layers are not part of the current
  generator contract.

## Async Strategy Closeout

The checked-in async contract is now explicit on the selected surface:

- Selected kinds with lifecycle async metadata are
  `containerengine/Cluster`, `containerinstances/ContainerInstance`,
  `core/Instance`, `database/AutonomousDatabase`,
  `functions/Application`, `functions/Function`, `identity/Compartment`,
  `keymanagement/Vault`, `mysql/DbSystem`, `nosql/Table`,
  `objectstorage/Bucket`, `opensearch/OpensearchCluster`,
  `psql/DbSystem`, and `streaming/Stream`.
- Selected kinds with workrequest async metadata are `queue/Queue` and
  `redis/RedisCluster`.
- `status.async.current` is the canonical in-flight tracker for the shared
  async contract and for the reference migrations that already project it in
  runtime today. Within the embedded shared OSOK status object, the canonical
  field is `status.async.current.workRequestId`; on the CR it is exposed at
  `.status.status.async.current.workRequestId`.
- `status.opcRequestId` is the canonical shared OCI request-correlation field
  for controller-backed resources. On the CR it is exposed at
  `.status.status.opcRequestId`.
- Lifecycle resources may seed that shared breadcrumb from opening create,
  update, or delete responses carrying `OpcWorkRequestId` without changing
  `async.strategy` to `workrequest`.
- Generated service-manager scaffolds that use `generatedruntime` inherit
  `status.opcRequestId` capture from the shared runtime automatically.
  Handwritten runtimes must publish the same field explicitly from mutating
  OCI response headers and surfaced OCI service errors; they must not invent
  resource-local replacements.
- `nosql/Table` is the lifecycle-only reference migration. `queue/Queue` and
  `redis/RedisCluster` are the workrequest-backed reference migrations.
- `queue/Queue` keeps its legacy work-request ID mirrors only for the current
  compatibility window; new selected resources should not add Queue-style
  compatibility fields by default.
- Remaining lifecycle/manual selected kinds that still expose OCI
  work-request APIs, including `psql/DbSystem`, are re-audited separately
  under `oci-service-operator-0kb`; the metadata classification does not, by
  itself, claim that those handwritten runtimes already project the Table
  reference semantics or the shared tracker identically.
- The disabled top-level `service: workrequests` row in
  `internal/generator/config/services.yaml` is a separate rollout decision.
  Setting `async.strategy=workrequest` on a published kind does not implicitly
  enable or publish a standalone `workrequests` API group.
- Scaffolded per-service `WorkRequest`, `WorkRequestError`, and
  `WorkRequestLog` rows in `formal/controller_manifest.tsv` remain catalog-only
  `stage=scaffold` entries until `oci-service-operator-9s2` resolves their
  prune-or-promote path. They do not authorize `formalSpec`,
  controller-backed runtime ownership, or package publication by themselves.

## Output Ownership

The contract now covers controller and service-manager outputs, but rollout
remains opt-in and non-destructive.

| Path or artifact | Owner | Notes |
| --- | --- | --- |
| `internal/generator/config/services.yaml` | Manual source-of-truth | Edited by hand when onboarding, reclassifying, or rolling out services. |
| `cmd/generator/**` | Manual implementation | Canonical user-facing generator entrypoint. |
| `internal/generator/**` Go code | Manual implementation | Reads the service map, discovers SDK resources, and renders outputs. |
| `internal/generator/generated/mutability_overlay/<service>/*.json` | generator | Conservative AST+docs mutability artifacts for the selected formal-backed resources. Emitted only by `cmd/generator`, not by manual edits. |
| `internal/generator/generated/vap_update_policy/<service>/*.json` | generator | ValidatingAdmissionPolicy-oriented update-policy inputs derived only from the merged mutability artifact `finalPolicy` surface. Refreshed only by `cmd/generator`, not by manual edits. |
| `api/<group>/<version>/groupversion_info.go` | generator | Derived from `group`, `version`, and repo domain. |
| `api/<group>/<version>/*_types.go` | generator | Top-level kinds, spec/status types, kubebuilder markers, and imports. |
| `api/<group>/<version>/zz_generated.deepcopy.go` | `controller-gen` | Rebuilt after generated API types change. This file is generated but not hand-authored and not directly owned by the generator. |
| `api/<group>/<version>/*_webhook.go` | Manual when present | Webhook code stays outside generator ownership; `generation.webhooks` only records the seam. |
| `api/<group>/<version>/webhook_suite_test.go` | Manual when present | Kept only for groups that still carry checked-in webhook code. |
| `controllers/<group>/*_controller.go` | Controller generator when `generation.controller.strategy=generated` | Manual files stay authoritative until rollout switches from `manual` or `none`. |
| `controllers/<group>/*_controller_test.go` | Manual for now | Controller test code is not generated by this contract. |
| `controllers/suite_test.go` | Manual | Shared test harness stays outside generator scope. |
| `pkg/servicemanager/**` | Service-manager generator when `generation.serviceManager.strategy=generated` | Generated scaffolds are baseline implementations; existing manual layouts may be referenced through `generation.resources[].serviceManager.packagePath`. |
| `internal/registrations/*.go` | Manual runtime bridge | Handwritten bridge code can adapt groups that still retain any remaining manual runtime seams to the shared registration/runtime contract until generated registration rollout lands. |
| `internal/registrations/<group>_generated.go` | Registration generator when `generation.registration.strategy=generated` | Contains scheme, controller, and service-manager registries for one group. |
| `main.go` imports and per-group wiring | Manual consumer | The generator does not rewrite `main.go`; later runtime integration consumes generated registrations instead of bespoke per-group edits. |
| `config/crd/bases/*.yaml` | `controller-gen` | Generated from API packages via `make manifests`. |
| `config/crd/kustomization.yaml` | Workflow-managed sync | `make manifests` refreshes the shared CRD resource list from `config/crd/bases/*.yaml` while preserving any remaining manual webhook and CA-injection patch sections used by the shared install path and bundle generation. |
| `config/samples/<group>_<version>_<kind-lower>.yaml` | generator | One sample manifest per generated top-level kind. |
| `config/samples/kustomization.yaml` sample entries | generator | The resource list is generator-owned and rewritten from the current generated sample set; the existing kubebuilder scaffold marker stays intact. |
| `packages/<group>/metadata.env` | generator | Derived from group identity and package profile. |
| `packages/<group>/install/kustomization.yaml` | generator | Generated static overlay for the group package profile. |
| `packages/<group>/install/generated/**` | Package workflow | Refreshed by `make package-generate` or `make package-install`, not by hand and not directly by the generator. |

Manual carve-outs are still deliberate. Webhooks, suite-test harnesses, and
direct `main.go` edits remain outside generator ownership even when controller
and service-manager generation is enabled.

## Naming and Derivation Rules

### Group and version

- Group DNS name: `<group>.oracle.com`
- Go package path: `api/<group>/<version>`
- Go package name inside that directory: `<version>`
- `groupversion_info.go` must set:
  `schema.GroupVersion{Group: "<group>.oracle.com", Version: "<version>"}`
- Default version is `v1beta1` for every service in the current epic scope.

### Kind discovery

- A generated kind comes from an OCI SDK resource family discovered from the
  service package, not from the service name itself.
- The generator pipeline should normalize OCI SDK CRUD request families
  (`Create*`, `Get*`, `List*`, `Update*`, `Delete*`) into a shared resource
  stem and render that stem as the OSOK kind.
- Checked-in kinds now follow the current v2 contract directly; the generator
  does not preserve older published GVK names through remap layers.

### Controller outputs

- Service-level controller rollout defaults to `none`.
- `generation.resources[].controller.strategy` overrides the service-wide
  controller rollout for one published kind; omitted kinds inherit the service
  default.
- Generated controller files live at:
  `controllers/<group>/<file-stem>_controller.go`
- Controller package name matches the group directory segment.
- Default reconciler type name is `<kind>Reconciler`.
- Default RBAC markers derive from the generated group, plural resource name,
  status subresource, finalizers, and event-recorder access
  (`events create;patch`).
- `generation.resources[].controller.extraRBACMarkers` appends additional
  kubebuilder RBAC marker payloads for one kind when non-default access such
  as secret reads or writes is still needed.
- `generation.resources[].controller.maxConcurrentReconciles` overrides the
  generated controller option only when explicitly set.
- New selected generated controllers should inherit event emission through the
  shared default markers; event-only `extraRBACMarkers` are redundant.
- A service may remain `manual` at the service level while recording resource
  overrides that later migration work will consume.

### Service-manager outputs

- Service-level service-manager rollout defaults to `none`.
- `generation.resources[].serviceManager.strategy` overrides the service-wide
  service-manager rollout for one published kind; omitted kinds inherit the
  service default.
- Generated service-manager packages live at:
  `pkg/servicemanager/<group>/<file-stem>/` by default.
- Default service-manager package name is the generated file stem.
- Generated baseline files live beside one another in that directory and must
  satisfy `pkg/servicemanager/interfaces.go`.
- Generated baseline packages contain `<file-stem>_serviceclient.go` and
  `<file-stem>_servicemanager.go`.
- `<Kind>ServiceManager` exposes `New<Kind>ServiceManagerWithDeps(...)` plus a
  convenience constructor that expands the same runtime contract fields.
- `<Kind>ServiceClient` is the handwritten extension seam. Generated
  `WithClient(...)`, typed conversion, and `GetCrdStatus(...)` helpers stay in
  the scaffold so manual logic can live in separate files without editing the
  generated baseline.
- `generation.resources[].serviceManager.packagePath` may override the default
  path with an existing relative path beneath `pkg/servicemanager/` when manual
  layouts must be preserved during migration.
- Generated scaffolds must expose clear manual extension seams rather than
  pretending the current handwritten implementations are fully templated.

### Registration surfaces

- Service-level registration rollout defaults to `none`.
- Generated registration files live at:
  `internal/registrations/<group>_generated.go`.
- Registration surfaces aggregate scheme registration, controller setup hooks,
  and service-manager factory metadata for one group.
- Generated registration rollout expects matching generated controller and
  service-manager outputs for each published kind whose effective runtime
  rollout resolves to `generated`.
- Generated registration files still add the full API group to the scheme even
  when some published kinds in that group remain API-only.
- `main.go` consumes registration surfaces; the generator never writes raw
  per-group wiring into `main.go` directly.

### Runtime dependency contract

- Shared service-manager constructor inputs live in
  `pkg/servicemanager/runtime.go`.
- `servicemanager.RuntimeDeps` carries OCI provider, credential client, scheme,
  logger, and optional runtime extras such as metrics.
- Group registration setup consumes `internal/registrations.Context`, which
  combines the controller-runtime manager/client/recorder seam with
  `servicemanager.RuntimeDeps`.
- Groups with manual webhook or other remaining runtime seams can still bridge
  into this contract through handwritten files under `internal/registrations/`
  without changing the shared manager bootstrap shape in `main.go`.

### Webhook ownership

- Webhook ownership defaults to `manual`.
- Webhook code remains under `api/<group>/<version>/*_webhook.go`.
- Resource overrides may pin individual kinds to `manual` so handwritten
  webhooks stay explicit when needed.
- Webhook generation is not part of this epic.

### Kubebuilder markers

Every top-level generated kind must carry these markers:

- `+kubebuilder:object:root=true`
- `+kubebuilder:subresource:status`

Default print columns are derived as follows:

- Primary display column:
  `DisplayName` on `.spec.displayName` when that field exists.
- Fallback primary display column:
  `Name` on `.spec.name` when `displayName` does not exist.
- Shared status column:
  `Status` on `.status.status.conditions[-1].type`
- Shared OCI identifier column:
  `Ocid` on `.status.status.ocid`
- Shared age column:
  `Age` on `.metadata.creationTimestamp`

Additional validation, default, enum, or minimum/maximum markers should come
from the OCI schema model when the generator can infer them. Any exception
should be expressed as a structured override in generator inputs, not as a
one-off file edit under `api/`.

### Structured field and sample overrides

- `generation.resources[].specFields` and `generation.resources[].statusFields`
  match fields by generated Go name, with JSON tag fallback for anonymous or
  embedded cases.
- Field overrides may set `type`, `tag`, `comments`, and `markers`.
- Omitted comments and markers inherit from the discovered field model; explicit
  values should be supplied when the repo-authored v2 contract intentionally
  diverges from the imported SDK field semantics, such as secret-backed
  credential inputs.
- `generation.resources[].sample.body` replaces the generated sample manifest
  for one kind. `metadataName` and `spec` allow narrower sample customization
  without forking the full body.

### Samples

- Sample file path:
  `config/samples/<group>_<version>_<strings.ToLower(kind)>.yaml`
- Sample `apiVersion`:
  `<group>.oracle.com/<version>`
- Sample `kind`:
  generated OSOK kind for that resource
- `config/samples/kustomization.yaml` must include every generated sample file
  in deterministic service/kind order and must not preserve preexisting
  entries that are not generated from the current config.

### Package metadata

`packages/<group>/metadata.env` is derived from the service record:

- `PACKAGE_NAME=oci-service-operator-<group>`
- `PACKAGE_NAMESPACE=oci-service-operator-<group>-system`
- `PACKAGE_NAME_PREFIX=oci-service-operator-<group>-`
- `CRD_PATHS=./api/<group>/...`
- `DEFAULT_CONTROLLER_IMAGE=iad.ocir.io/oracle/oci-service-operator:latest`

`RBAC_PATHS` depends on the package profile:

- `controller-backed`:
  `RBAC_PATHS=./controllers/<group>/...`
- `crd-only`:
  no controller RBAC path is generated until controller support exists

## Package Profiles

Two package profiles remain part of the contract, but they describe install
posture rather than whether the runtime pieces are handwritten or generated.

### `controller-backed`

Use this for groups that participate in the shared manager install and package
wiring. The exact set is declared in
`internal/generator/config/services.yaml`, and controller-backed groups may
still mix generated and handwritten seams during rollout.

Generated package scaffolding must include:

- `packages/<group>/metadata.env` with `RBAC_PATHS=./controllers/<group>/...`
- `packages/<group>/install/kustomization.yaml` referencing:
  - `generated/crd`
  - `generated/rbac`
  - `../../../config/manager`
  - shared leader-election and role-binding manifests
  - group-specific editor and viewer roles when they exist

### `crd-only`

Use this for generator-first groups that do not yet participate in the shared
manager install.

Generated package scaffolding must include:

- `packages/<group>/metadata.env` without a controller RBAC path
- `packages/<group>/install/kustomization.yaml` that references only
  `generated/crd`

### Transition rules

- New services start as `crd-only` and normally omit the `generation` block.
- Existing controller-backed services stay `controller-backed`; the v2
  generator no longer carries legacy overlay or kind-preservation behavior.
- A service moves from `crd-only` to `controller-backed` only when controller,
  service-manager, and registration rollout is explicitly enabled and the
  shared manager/package integration issue lands.
- Generated services now move into the shared-manager rollout directly in
  `services.yaml` once controller, service-manager, registration, validator,
  and webhook prerequisites are ready.
- `make generated-runtime-gate` uses that source-of-truth config by default. If
  future rollout work needs a pre-promotion snapshot, point
  `GENERATED_RUNTIME_CONFIG` at an alternate config explicitly.

### Formal onboarding and promotion flow

Formal coverage is resource-scoped. One `formal/controller_manifest.tsv` row
maps to one published OSOK kind, even when several rows belong to the same API
group.

Use this flow when onboarding a new service or promoting an existing resource
from scaffold coverage into generated runtime:

1. Keep service inventory and rollout metadata in
   `internal/generator/config/services.yaml`. New services start as
   `packageProfile: crd-only`; add `formalSpec` at the service level or under
   `generation.resources[]` once the published OSOK kind has a matching formal
   manifest row.
2. Run `make formal-scaffold` to create or refresh scaffold-only rows for the
   published default-active API surface. Pass
   `FORMAL_PROVIDER_PATH=/path/to/terraform-provider-oci` when matching
   provider facts for that selected surface should also be refreshed. The scaffold flow keeps
   `runtime-lifecycle.yaml` as structured metadata, generates shared
   `activity`, `sequence`, `state-machine`, and `legend` `.puml` and `.svg`
   artifacts under `formal/shared/diagrams/`, and renders deterministic
   controller-local `activity`, `sequence`, and `state-machine` `.puml` and
   `.svg` files under each controller diagram directory. The SVGs come from the
   `plantuml` CLI, so keep `plantuml` available on `PATH` before running the
   formal targets. Then update the matching `formal/` artifacts:
   `formal/controller_manifest.tsv`,
   `formal/controllers/<service>/<slug>/spec.cfg`,
   `formal/controllers/<service>/<slug>/logic-gaps.md`,
   `formal/controllers/<service>/<slug>/diagrams/`, and
   `formal/imports/<service>/<slug>.json`. `make formal-scaffold` is also the
   authoritative cleanup path for generator-owned formal catalog artifacts: if
   the active published surface shrinks, the scaffold refresh prunes stale
   `formal/controllers/<service>/<slug>/...` and
   `formal/imports/<service>/<slug>.json` entries that are no longer referenced
   by `formal/controller_manifest.tsv`.
3. Refresh provider facts with an explicit pinned provider source:

   ```sh
   make formal-import FORMAL_IMPORT_PROVIDER_PATH=/path/to/terraform-provider-oci
   ```

   `FORMAL_IMPORT_PROVIDER_PATH` is an operator-supplied external input. The
   repo does not assume a sibling checkout or a writable provider tree.
4. Run formal validation before generation:

   ```sh
   make formal-verify
   ```

   When provider-backed coverage is in scope, also run:

   ```sh
   make formal-scaffold-verify FORMAL_PROVIDER_PATH=/path/to/terraform-provider-oci
   ```

   `formal-verify` checks generated `.puml` sources plus the embedded PlantUML
   metadata inside rendered SVGs, so diagram-affecting formal changes must be
   followed by `make formal-diagrams` or `make formal-scaffold`. It also
   rejects orphan controller directories and import JSON files that are not
   referenced by `formal/controller_manifest.tsv`.

5. Record every unsupported or legacy-only behavior in `logic-gaps.md` front
   matter with an explicit `stopCondition`, and file follow-up `bd` issues for
   promotion blockers instead of hidden TODOs. Open logic gaps block
   formal-driven promotion for that resource.
6. Regenerate and gate the promoted path:

   ```sh
   go run ./cmd/generator --config internal/generator/config/services.yaml --all
   go run ./hack/update_validator_registries.go --write
   make generated-coverage-gate
   make generated-runtime-gate
   make generated-mutability-gate
   ```

7. Move a service from `crd-only` to `controller-backed` in `services.yaml`
   only after controller, service-manager, registration, validator, webhook,
   and package prerequisites land. When a resource still depends on
   handwritten runtime seams, keep those files explicit until the
   corresponding `logic-gaps.md` stop conditions are closed.

Scaffold coverage alone does not move a group into generated runtime or replace
remaining handwritten seams. Runtime ownership still follows `generation.*`
rollout and explicit `formalSpec` references.

Workflow integration for both package profiles uses the same targets:

- `make package-generate GROUP=<group>` refreshes package-local generated CRDs
  and emits controller RBAC only when `RBAC_PATHS` is configured.
- `make package-install GROUP=<group>` renders an install YAML for either
  package profile while the shared `config/manager` deployment remains the
  authoritative controller install path.

## Regeneration and Validation Flow

The supported generator interface is direct invocation of `./cmd/generator`:

```sh
go run ./cmd/generator --config internal/generator/config/services.yaml --all
```

`--all` means every service with `selection.enabled: true` in the selected
config. When one of those services uses `selection.mode=explicit`, only its
configured kind subset reaches package-model construction and downstream
renderers. Use `--service <name>` to bypass the default-active filter and
regenerate one configured service explicitly, including backlog or disabled
services.

Single-service regeneration uses the same interface:

```sh
go run ./cmd/generator --config internal/generator/config/services.yaml --service mysql
```

Expected regeneration and validation flow:

1. Update `internal/generator/config/services.yaml` if service scope,
   resource `kind`, runtime rollout, or `formalSpec` bindings changed.
2. If formal scope changed, run `make formal-scaffold` to refresh scaffold rows
   and rendered diagram artifacts. Use
   `FORMAL_PROVIDER_PATH=/path/to/terraform-provider-oci` when provider-backed
   coverage changed. Update any non-scaffold `logic-gaps.md` entries, run
   `make formal-import FORMAL_IMPORT_PROVIDER_PATH=/path/to/terraform-provider-oci`
   for non-scaffold rows, then run `make formal-verify` and, when provider
   coverage is expected, `make formal-scaffold-verify
   FORMAL_PROVIDER_PATH=/path/to/terraform-provider-oci`.
3. Run `go run ./cmd/generator ...` to emit API packages, sample manifests,
   sample kustomization, package scaffolding, the pinned mutability overlay
   artifacts, and the derived VAP update-policy input artifacts for the
   selected services.
   Generator-owned spec, helper, sample, and package artifacts regenerate
   directly from `services.yaml` and the current v2 contract; there is no
   legacy-preservation mode in the generator. With `--all --overwrite`, the
   generator performs a full-sync cleanup that removes stale generator-owned
   outputs under `api/`, `controllers/`, `pkg/servicemanager/`,
   `internal/generator/generated/mutability_overlay/`,
   `internal/generator/generated/vap_update_policy/`,
   `internal/registrations/`, `packages/`, and `config/samples/` when they no
   longer belong to the active surface.
4. When deepcopy output and CRD manifests also need refresh, run `make generate`
   and `make manifests` after the generator command. That flow also syncs the shared
   `config/crd/kustomization.yaml` resource list from `config/crd/bases/*.yaml`.
5. Run `go run ./hack/update_validator_registries.go --write` so validator
   coverage targets stay aligned with `services.yaml` and the generated API
   packages. Any generated spec that still has no mapped SDK payload will remain
   visible as `untracked` in validator output until its `SDKStructs` mapping is
   filled in.
6. Run `make generated-coverage-gate` to keep the generated API snapshot and
   validator coverage baseline from regressing.
7. Run `make generated-runtime-gate` to compile-check the generated
   controller/service-manager/registration snapshot defined by the active
   generator config (by default `internal/generator/config/services.yaml`).
   Override `GENERATED_RUNTIME_CONFIG` when staging an alternate rollout config
   ahead of promotion.
8. Run `make generated-mutability-gate` to keep the pinned mutability overlay
   and derived VAP update-policy decisions from regressing on the
   fixture-backed validation surface defined in
   `internal/generator/config/mutability_validation_services.yaml`. When the
   pinned Terraform docs version changes, refresh the checked-in docs fixtures
   first with `make mutability-docs-refresh`, review the new mutability report,
   and only then refresh
   `internal/generator/config/generated_mutability_baseline.json` intentionally.
9. Run `make fmt`.
10. Run `make vet`.
11. Run `make test`.
12. Run `make build`.

The checked-in generator emits API/package outputs for all enabled services in
the default active surface and can also emit controller, service-manager, and
registration outputs for services explicitly switched to
`generation.*.strategy=generated`. Use `--service <name>` when local work needs
to regenerate a configured service that is currently inactive by default.
Manual groups stay authoritative until later rollout work changes their
strategy.

Package regeneration stays separate from the API generation step:

- `make package-generate GROUP=<group>` refreshes package-local generated CRDs
  and optional controller RBAC for either package profile.
- `make package-install GROUP=<group>` renders an install YAML for either
  package profile.

## Current Non-Goals

This contract intentionally leaves these areas for later issues:

- direct edits to `main.go`
- webhook generation
- full replacement of any remaining handwritten runtime implementations,
  including the narrow `streaming` endpoint-secret companion

Those follow-on items should conform to this contract rather than redefining
service mapping, naming rules, rollout vocabulary, or package profiles.

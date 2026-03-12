# OSOK API Generator Contract

This document defines the source-of-truth mapping and output contract for the
OSOK API generator epic (`oci-service-operator-0cd*`). It is the design input
for the later generator implementation under `cmd/` and `internal/`.

## Source of Truth

`internal/generator/config/services.yaml` is the only hand-maintained mapping
for OCI SDK services to OSOK API groups.

Each service record defines:

| Field | Meaning |
| --- | --- |
| `service` | OCI Go SDK service package name. |
| `sdkPackage` | Full OCI Go SDK import path used by the generator pipeline. |
| `group` | OSOK API group name and directory segment under `api/`. |
| `version` | API version. Default is `v1beta1` unless a record overrides it. |
| `phase` | Current rollout bucket used by the generator epic. |
| `packageProfile` | `controller-backed` for groups with existing shared-manager controllers, `crd-only` for generated API-only groups. |
| `compatibility.existingKinds` | Stable kinds already published by this repo that later generator output must preserve unless a follow-up issue explicitly changes them. |

Rules:

- Service-to-group mapping is 1:1.
- `group` stays equal to the OCI SDK package basename unless the mapping file
  explicitly says otherwise.
- Service-specific compatibility behavior belongs in the mapping file, not in
  hardcoded generator branches.
- The existing manual groups with locked compatibility kinds are:
  `database/AutonomousDatabases`, `mysql/MySqlDbSystem`, and
  `streaming/Stream`.

## Output Ownership

The generator contract is intentionally narrower than the full operator repo.
Generator-owned outputs stop at API packages, samples, and package scaffolding.

| Path or artifact | Owner | Notes |
| --- | --- | --- |
| `internal/generator/config/services.yaml` | Manual source-of-truth | Edited by hand when onboarding or reclassifying services. |
| `cmd/osok-api-generator/**` | Manual implementation | Added by follow-on implementation work; this contract only fixes its interface and inputs. |
| `internal/generator/**` Go code | Manual implementation | Reads the service map, discovers SDK resources, and renders outputs. |
| `api/<group>/<version>/groupversion_info.go` | API generator | Derived from `group`, `version`, and repo domain. |
| `api/<group>/<version>/*_types.go` | API generator | Top-level kinds, spec/status types, kubebuilder markers, and imports. |
| `api/<group>/<version>/zz_generated.deepcopy.go` | `controller-gen` | Rebuilt after generated API types change. This file is generated but not hand-authored and not directly owned by the API generator. |
| `api/<group>/<version>/*_webhook.go` | Manual | Webhook code stays manual until a later automation issue exists. |
| `api/<group>/<version>/webhook_suite_test.go` | Manual | Remains manual with webhook code. |
| `controllers/<group>/**` | Manual | Reconcile logic and controller tests stay hand-maintained. |
| `controllers/suite_test.go` | Manual | Shared test harness stays outside generator scope. |
| `main.go` imports, `AddToScheme`, and controller setup | Manual | Scheme wiring and controller registration remain explicit until a follow-on automation issue exists. |
| `config/crd/bases/*.yaml` | `controller-gen` | Generated from API packages via `make manifests`. |
| `config/crd/kustomization.yaml` | Manual for now | The shared CRD aggregator stays manual until workflow integration work teaches regeneration to keep it in sync. |
| `config/samples/<group>_<version>_<kind-lower>.yaml` | API generator | One sample manifest per generated top-level kind. |
| `config/samples/kustomization.yaml` sample entries | API generator | The resource list is generator-owned; the existing kubebuilder scaffold marker stays intact. |
| `packages/<group>/metadata.env` | API generator | Derived from group identity and package profile. |
| `packages/<group>/install/kustomization.yaml` | API generator | Generated static overlay for the group package profile. |
| `packages/<group>/install/generated/**` | Package workflow | Refreshed by `make package-generate` or `make package-install`, not by hand and not directly by the API generator. |

Manual carve-outs are deliberate. Controllers, webhooks, suite tests, and
`main.go` remain outside generator scope in the initial rollout.

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
- If the discovered kind would break an existing published OSOK GVK, the
  compatibility override belongs in `internal/generator/config/services.yaml`
  rather than in special-case Go code.
- Existing locked compatibility kinds are:
  `AutonomousDatabases`, `MySqlDbSystem`, and `Stream`.

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

### Samples

- Sample file path:
  `config/samples/<group>_<version>_<strings.ToLower(kind)>.yaml`
- Sample `apiVersion`:
  `<group>.oracle.com/<version>`
- Sample `kind`:
  generated OSOK kind for that resource
- `config/samples/kustomization.yaml` must include every generated sample file
  in deterministic service/kind order.

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

Two package profiles are part of the contract.

### `controller-backed`

Use this only for groups already served by the shared manager. In the current
repo that is `database`, `mysql`, and `streaming`.

Generated package scaffolding must include:

- `packages/<group>/metadata.env` with `RBAC_PATHS=./controllers/<group>/...`
- `packages/<group>/install/kustomization.yaml` referencing:
  - `generated/crd`
  - `generated/rbac`
  - `../../../config/manager`
  - shared leader-election and role-binding manifests
  - group-specific editor and viewer roles when they exist

### `crd-only`

Use this for generator-first groups that do not yet have controllers or
controller RBAC.

Generated package scaffolding must include:

- `packages/<group>/metadata.env` without a controller RBAC path
- `packages/<group>/install/kustomization.yaml` that references only
  `generated/crd`

Workflow integration for both package profiles now uses the same targets:

- `make package-generate GROUP=<group>` refreshes package-local generated CRDs
  and emits controller RBAC only when `RBAC_PATHS` is configured.
- `make package-install GROUP=<group>` renders an install YAML for either
  package profile while the shared `config/manager` deployment remains the
  authoritative controller install path.

## Regeneration and Validation Flow

The generator supports direct invocation:

```sh
go run ./cmd/osok-api-generator --config internal/generator/config/services.yaml --all
```

Single-service regeneration must also be supported:

```sh
go run ./cmd/osok-api-generator --config internal/generator/config/services.yaml --service mysql
```

The scripted repo entrypoints are:

```sh
make api-generate API_ALL=true
make api-generate API_SERVICE=mysql API_OVERWRITE=true
make api-refresh API_SERVICE=mysql API_OVERWRITE=true
```

Expected regeneration and validation flow:

1. Update `internal/generator/config/services.yaml` if service scope or
   compatibility behavior changed.
2. Run `make api-generate ...` to emit API packages, sample manifests, sample
   kustomization, and package scaffolding for the selected services.
3. Run `make api-refresh ...` when deepcopy output and CRD manifests also need
   to be refreshed in the same step.
4. Run `make fmt`.
5. Run `make vet`.
6. Run `make test`.
7. Run `make build`.

Package regeneration stays separate from the API generation step:

- `make package-generate GROUP=<group>` refreshes package-local generated CRDs
  and optional controller RBAC for either package profile.
- `make package-install GROUP=<group>` renders an install YAML for either
  package profile.

## Follow-on Constraints

This contract intentionally leaves these areas for later issues:

- parity review against the current `database`, `mysql`, and `streaming` API
  groups
- automatic updates to `config/crd/kustomization.yaml`
- any automation for controllers, webhooks, suite tests, or `main.go`

Those follow-on items should conform to this contract rather than redefining
service mapping, naming rules, or package profiles.

# Generator Parity Strategy

This note records the parity-overlay strategy used for the historical
`database`, `mysql`, and `streaming` groups.

The current `database`, `mysql`, and `streaming` groups are parity-reviewed for
published kinds and API/package artifacts. Their existing published resources
still carry explicit parity overlays, but generator discovery now emits the
full service resource set for each group.

## Generator-owned parity scope

For the three parity groups, the checked-in generator remains responsible for:

- `api/database/v1beta1/**`
- `api/mysql/v1beta1/**`
- `api/streaming/v1beta1/**`
- the generated samples for every discovered resource in those groups,
  including the historical `AutonomousDatabases`, `MySqlDbSystem`, and
  `Stream` samples
- the matching entries in `config/samples/kustomization.yaml`
- `controllers/database/**`
- `controllers/mysql/**`
- `controllers/streaming/**`
- `pkg/servicemanager/autonomousdatabases/adb/**`
- `pkg/servicemanager/mysql/dbsystem/**`
- `pkg/servicemanager/streaming/stream/**`
- the additional generated service-manager packages for every other discovered
  resource in those groups
- `internal/registrations/database_generated.go`
- `internal/registrations/mysql_generated.go`
- `internal/registrations/streaming_generated.go`
- `packages/<group>/metadata.env`
- `packages/<group>/install/kustomization.yaml`

## Overlay plan

The parity groups now stay on the generated-runtime path while preserving the
legacy published resources through targeted parity overlays:

- `internal/generator/config/services.yaml` keeps
  `generation.controller.strategy=generated`,
  `generation.serviceManager.strategy=generated`, and
  `generation.registration.strategy=generated` for `database`, `mysql`, and
  `streaming`.
- `parity/*.yaml` overlays only the historical published resources instead of
  filtering the service down to one resource.
- `compatibility.existingKinds` preserves the published kinds
  `AutonomousDatabases`, `MySqlDbSystem`, and `Stream`.
- `generation.resources[].serviceManager.packagePath` preserves the established
  package layouts for the historical resources:
  `pkg/servicemanager/autonomousdatabases/adb`,
  `pkg/servicemanager/mysql/dbsystem`, and
  `pkg/servicemanager/streaming/stream`.
- legacy behavior for those historical resources is preserved through
  handwritten client-adapter files that plug the generated service-manager
  scaffolds into the existing manual implementations.
- `database` webhook setup stays manual through
  `api/database/v1beta1/*_webhook.go`.

## Why adapters still exist

The generated foundations are still baseline-only for most resources. The
historical parity resources retain handwritten OCI behavior that is materially
richer than the current generated scaffolds:

- `database` retains wallet handling, secret-reference reconciliation, and
  webhook-specific validation behavior.
- `mysql` retains admin-secret ingestion, bind-versus-create branching, and
  custom update or retry handling.
- `streaming` retains metrics emission, secret materialization, and custom
  status or retry behavior.

Those differences are the reason the current strategy is explicit opt-out
rather than pretending the new generated scaffolds are already feature-parity
replacements.

## Validation

The parity regression path now validates that:

- generated API, sample, package, controller, service-manager, and registration
  outputs for the three parity groups still match the checked-in artifacts
- the parity overlays preserve the historical published resources while the
  rest of each service still generates from full SDK discovery
- registration tests keep manual webhooks explicit and reject duplicate group
  registrations

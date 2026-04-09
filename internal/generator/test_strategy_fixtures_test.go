package generator

import (
	"path/filepath"
	"testing"
)

const generatorTestActivityStrategy = `schemaVersion: 1
family: activity
title: Activity
sharedDiagram: shared/diagrams/shared-reconcile-activity.svg
summary:
  - Start from the shared reconcile activity and specialize provider-backed CRUD, wait, and status-projection details here.
  - Controller-local activity diagrams add list-lookup hints, secret policies, delete handling, and open formal gaps.
archetypeBatches:
  generated-service-manager: Generated service-manager controllers specialize the shared reconcile path with formal runtime ownership.
  legacy-adapter: Legacy-adapter controllers specialize the shared reconcile path while explicit gaps still block generated promotion.
`

const generatorTestSequenceStrategy = `schemaVersion: 1
family: sequence
title: Sequence
sharedDiagram: shared/diagrams/shared-resolution-sequence.svg
sharedDeleteDiagram: shared/diagrams/shared-delete-sequence.svg
summary:
  - Start from the shared resolution sequence for OCID binding, datasource pagination, and create-or-update dispatch.
  - Layer the shared delete sequence when finalizer retention, delete confirmation, work-request polling, or Secret cleanup apply.
participants:
  - Kubernetes
  - Controller
  - BaseReconciler
  - ServiceManager
  - OCI
  - SecretStore
archetypeBatches:
  generated-service-manager: Generated service-manager controllers follow the shared resolution and delete sequences with imported waits and controller-local policy.
  legacy-adapter: Legacy-adapter controllers follow the shared resolution and delete sequences but preserve handwritten stop conditions until promotion is safe.
`

const generatorTestStateMachineStrategy = `schemaVersion: 1
family: state-machine
title: State Machine
sharedDiagram: shared/diagrams/shared-controller-state-machine.svg
summary:
  - Start from the shared controller phase model and specialize imported provider states, delete targets, and failure exits here.
baseStates:
  - provisioning
  - active
  - updating
  - terminating
  - failed
archetypeBatches:
  generated-service-manager: Generated service-manager controllers map provider states into the shared controller phases with explicit finalizer and requeue policies.
  legacy-adapter: Legacy-adapter controllers map provider states into the shared controller phases while open gaps keep additional transitions explicit.
`

const generatorTestSharedStrategy = `schemaVersion: 1
diagrams:
  - file: shared/diagrams/shared-reconcile-activity.svg
    title: Shared Reconcile Activity
    subtitle: Common reconcile flow specialized by controller-local activity diagrams
    lines:
      - Read desired spec and existing status from Kubernetes.
      - Resolve current OCI identity through GET or datasource list fallback before choosing bind, create, update, or delete.
      - Apply controller-local spec, secret, and delete policies while keeping BaseReconciler invariants intact.
      - Project OSOK status, finalizer state, and retryable conditions back to the reconciler.
  - file: shared/diagrams/shared-resolution-sequence.svg
    title: Shared Resolution Sequence
    subtitle: Shared ID binding, datasource pagination, and create-update dispatch
    lines:
      - BaseReconciler asks the service manager to resolve the controller's current OCI identity.
      - Controller-local specialization supplies datasource filters, pagination behavior, and bind-versus-create policy.
      - Imported waits and read-after-write behavior keep retryable states explicit.
  - file: shared/diagrams/shared-delete-sequence.svg
    title: Shared Delete Sequence
    subtitle: Finalizer discipline, delete confirmation, work requests, and optional Secret cleanup
    lines:
      - Keep the OSOK finalizer until delete confirmation or an explicit not-supported policy says otherwise.
      - Controller-local specialization supplies delete confirmation via GET, datasource list fallback, or provider wait helpers.
      - Optional Secret cleanup remains explicit and only happens when repo-authored policy allows it.
  - file: shared/diagrams/shared-controller-state-machine.svg
    title: Shared Controller State Machine
    subtitle: Common controller phases specialized by provider-backed lifecycle states
    lines:
      - Provisioning, Active, Updating, Terminating, and Failed are the shared controller phases.
      - Controller-local state machines map imported provider states into those phases and keep delete targets explicit.
      - Open gaps and legacy adapters stay visible instead of being hidden behind a generic steady-state path.
`

const generatorTestLegendStrategy = `schemaVersion: 1
title: Shared Diagram Legend
subtitle: Diagram palette and controller archetype batches used by the formal generator strategy
palette:
  shared-contract:
    color: "#0f172a"
    label: Shared Contract
    description: Shared reconcile semantics and common controller invariants.
  controller-local:
    color: "#1d4ed8"
    label: Controller Local
    description: Per-controller specialization derived from manifest rows and spec settings.
  provider-facts:
    color: "#0f766e"
    label: Provider Facts
    description: Imported terraform-provider-oci operations, waits, datasource behavior, and lifecycle states.
  repo-authored:
    color: "#b45309"
    label: Repo Authored
    description: OSOK-only status, finalizer, Secret, and delete semantics captured in formal specs and gaps.
  open-gap:
    color: "#b91c1c"
    label: Open Gap
    description: Explicit stop conditions that still block generated promotion or adapter removal.
archetypeBatches:
  generated-service-manager:
    color: "#1d4ed8"
    label: Generated Service Manager Batch
    description: Controller-backed services where generated runtime owns CRUD, waits, delete confirmation, and status projection.
  legacy-adapter:
    color: "#b91c1c"
    label: Legacy Adapter Batch
    description: Controllers that still route through handwritten adapters until open gaps close.
`

func writeGeneratorDiagramStrategyFixtures(t *testing.T, formalRoot string) {
	t.Helper()

	writeGeneratorTestFile(t, filepath.Join(formalRoot, "controller_diagrams", "activity.yaml"), generatorTestActivityStrategy)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "controller_diagrams", "sequence.yaml"), generatorTestSequenceStrategy)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "controller_diagrams", "state-machine.yaml"), generatorTestStateMachineStrategy)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "controller_diagrams", "shared.yaml"), generatorTestSharedStrategy)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "controller_diagrams", "legend.yaml"), generatorTestLegendStrategy)
}

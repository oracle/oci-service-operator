package formal

import "strings"

// RuntimeLifecycleSpec exposes the parsed repo-authored runtime-lifecycle.yaml
// metadata so generator/runtime consumers can reuse the same merged semantics
// that formal diagrams already render.
type RuntimeLifecycleSpec = diagramSpec

// RuntimeLifecycleRepoAuthoredSemantics exposes the optional repo-authored
// runtime overrides embedded in runtime-lifecycle.yaml.
type RuntimeLifecycleRepoAuthoredSemantics = diagramRepoAuthoredSemantics

// RuntimeLifecycleProviderLifecycle exposes repo-authored lifecycle-state
// overrides embedded in runtime-lifecycle.yaml.
type RuntimeLifecycleProviderLifecycle = diagramProviderLifecycle

// RuntimeLifecycleListLookupSemantics exposes repo-authored list-lookup
// overrides embedded in runtime-lifecycle.yaml.
type RuntimeLifecycleListLookupSemantics = diagramListLookupSemantics

// RuntimeLifecycleMutationSemantics exposes repo-authored merged mutation
// allowlists embedded in runtime-lifecycle.yaml.
type RuntimeLifecycleMutationSemantics = diagramMutationSemantics

// RuntimeLifecycleOperationSemantics exposes repo-authored effective provider
// operation subsets embedded in runtime-lifecycle.yaml.
type RuntimeLifecycleOperationSemantics = diagramOperationSemantics

// RuntimeLifecycleHookSemantics exposes repo-authored helper-hook overrides
// embedded in runtime-lifecycle.yaml.
type RuntimeLifecycleHookSemantics = diagramHookSemantics

// RuntimeLifecycleFollowUpSemantics exposes repo-authored follow-up strategy
// overrides embedded in runtime-lifecycle.yaml.
type RuntimeLifecycleFollowUpSemantics = diagramFollowUpSemantics

// LoadRuntimeLifecycle parses one repo-authored runtime-lifecycle.yaml file.
func LoadRuntimeLifecycle(path string) (RuntimeLifecycleSpec, error) {
	return loadDiagram(path)
}

// EffectiveRuntimeLifecycleCreateOperations returns the imported create
// operations selected by repo-authored runtime semantics.
func EffectiveRuntimeLifecycleCreateOperations(
	spec *RuntimeLifecycleSpec,
	imported []OperationBinding,
) []OperationBinding {
	return effectiveRuntimeLifecycleOperations(spec, imported, func(operations *diagramOperationSemantics) []string {
		return operations.Create
	})
}

// EffectiveRuntimeLifecycleUpdateOperations returns the imported update
// operations selected by repo-authored runtime semantics.
func EffectiveRuntimeLifecycleUpdateOperations(
	spec *RuntimeLifecycleSpec,
	imported []OperationBinding,
) []OperationBinding {
	return effectiveRuntimeLifecycleOperations(spec, imported, func(operations *diagramOperationSemantics) []string {
		return operations.Update
	})
}

// EffectiveRuntimeLifecycleDeleteOperations returns the imported delete
// operations selected by repo-authored runtime semantics.
func EffectiveRuntimeLifecycleDeleteOperations(
	spec *RuntimeLifecycleSpec,
	imported []OperationBinding,
) []OperationBinding {
	return effectiveRuntimeLifecycleOperations(spec, imported, func(operations *diagramOperationSemantics) []string {
		return operations.Delete
	})
}

// An omitted phase subset preserves every imported provider operation. An
// explicit subset, including an explicit empty list, keeps only the named
// operations in declaration order.
func effectiveRuntimeLifecycleOperations(
	spec *RuntimeLifecycleSpec,
	imported []OperationBinding,
	selection func(*diagramOperationSemantics) []string,
) []OperationBinding {
	if spec == nil || spec.RepoAuthored == nil || spec.RepoAuthored.Operations == nil {
		return append([]OperationBinding(nil), imported...)
	}
	subset := selection(spec.RepoAuthored.Operations)
	if subset == nil {
		return append([]OperationBinding(nil), imported...)
	}

	byName := make(map[string]OperationBinding, len(imported))
	for _, operation := range imported {
		byName[strings.TrimSpace(operation.Operation)] = operation
	}

	effective := make([]OperationBinding, 0, len(subset))
	for _, name := range subset {
		if operation, ok := byName[strings.TrimSpace(name)]; ok {
			effective = append(effective, operation)
		}
	}
	return effective
}

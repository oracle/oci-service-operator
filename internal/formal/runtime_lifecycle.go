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

// EffectiveRuntimeLifecycleUpdateOperations returns the imported update
// operations selected by repo-authored runtime semantics. An omitted update
// subset preserves every imported provider operation; an explicit subset keeps
// only the named operations in declaration order.
func EffectiveRuntimeLifecycleUpdateOperations(
	spec *RuntimeLifecycleSpec,
	imported []OperationBinding,
) []OperationBinding {
	if spec == nil || spec.RepoAuthored == nil || spec.RepoAuthored.Operations == nil || spec.RepoAuthored.Operations.Update == nil {
		return append([]OperationBinding(nil), imported...)
	}

	byName := make(map[string]OperationBinding, len(imported))
	for _, operation := range imported {
		byName[strings.TrimSpace(operation.Operation)] = operation
	}

	effective := make([]OperationBinding, 0, len(spec.RepoAuthored.Operations.Update))
	for _, name := range spec.RepoAuthored.Operations.Update {
		if operation, ok := byName[strings.TrimSpace(name)]; ok {
			effective = append(effective, operation)
		}
	}
	return effective
}

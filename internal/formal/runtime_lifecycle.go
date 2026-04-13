package formal

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

// LoadRuntimeLifecycle parses one repo-authored runtime-lifecycle.yaml file.
func LoadRuntimeLifecycle(path string) (RuntimeLifecycleSpec, error) {
	return loadDiagram(path)
}

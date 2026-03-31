Implemented the remaining MySQL DbSystem rename follow-through for the v2-only cleanup.

- Renamed the checked-in MySQL end-user RBAC snippets from `mysqldbsystem_*` to `dbsystem_*` and updated the role names/resources to `dbsystems`.
- Renamed the MySQL CRD webhook and cert-manager patch snippets to `dbsystems` and updated `config/crd/kustomization.yaml` to reference the new filenames.
- Removed the leftover `API_GENERATOR_*` / `API_*` Makefile alias variables so the generator surface is driven only by the current `GENERATOR_*` contract.
- Repaired the current branch’s `internal/generator` helper/test regressions that blocked validation after the v2-only cleanup, and updated the remaining stale MySQL generator expectations to the current `DbSystem` contract.

Validation:

- `go test ./internal/generator -run 'TestLoadConfigIncludesGenerationRolloutAndOverrides|TestGeneratePreservesCheckedInPackageArtifactsInOutputRoot|TestCheckedInConfigLeavesMySQLObservedStateUnconfigured' -count=1`
- `go test ./hack -run TestMakefileDoesNotExposeLegacyGeneratorAliases -count=1`
- `make test`

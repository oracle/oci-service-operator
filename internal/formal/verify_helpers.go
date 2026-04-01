package formal

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type manifestResolvedPaths struct {
	SpecPath   string
	LogicPath  string
	ImportPath string
	DiagramDir string
}

type manifestRowArtifacts struct {
	SpecModel SpecModel
	LogicGaps logicGapsFrontMatter
	ImportDoc importFile
	HasSpec   bool
	HasLogic  bool
	HasImport bool
}

type operationBindingGroup struct {
	name     string
	bindings []operationBinding
}

func normalizeFormalRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", &ValidationError{Problems: []string{"formal root must not be empty"}}
	}
	return filepath.Abs(root)
}

func validateFormalDirectories(root string) []string {
	var problems []string
	for _, requiredDir := range []string{"shared", sharedDiagramDir, controllerDiagramTemplateDir, "controllers", "imports"} {
		if err := requireDirectory(filepath.Join(root, requiredDir)); err != nil {
			problems = append(problems, err.Error())
		}
	}
	return problems
}

func loadVerifiedSourceIndex(formalRoot string) (int, map[string]sourceLockEntry, []string) {
	lockFile, sourceIndex, err := loadSourceLock(filepath.Join(formalRoot, "sources.lock"))
	if err != nil {
		return 0, nil, []string{err.Error()}
	}
	return len(lockFile.Sources), sourceIndex, nil
}

func loadVerifiedManifest(formalRoot string) ([]manifestRow, int, []string) {
	rows, err := loadManifest(filepath.Join(formalRoot, "controller_manifest.tsv"))
	if err != nil {
		return nil, 0, []string{err.Error()}
	}
	return rows, len(rows), nil
}

func validateManifestOwnedArtifacts(formalRoot string, rows []manifestRow) []string {
	desiredControllerRoots := make(map[string]struct{}, len(rows))
	desiredImportPaths := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		desiredControllerRoots[filepath.Clean(filepath.Dir(filepath.FromSlash(row.SpecPath)))] = struct{}{}
		desiredImportPaths[filepath.Clean(filepath.FromSlash(row.ImportPath))] = struct{}{}
	}

	problems := validateOrphanedControllerArtifacts(formalRoot, desiredControllerRoots)
	problems = append(problems, validateOrphanedImportArtifacts(formalRoot, desiredImportPaths)...)
	return problems
}

func validateManifestRows(formalRoot string, rows []manifestRow, sourceIndex map[string]sourceLockEntry, strategy diagramStrategy) (int, []plantUMLPair, []string) {
	var problems []string
	renderedPairs := make([]plantUMLPair, 0, len(rows))
	seenRows := map[string]int{}
	diagramCount := 0

	for _, row := range rows {
		rowKey := strings.Join([]string{row.Service, row.Slug, row.Kind}, "/")
		if previous, ok := seenRows[rowKey]; ok {
			problems = append(problems, fmt.Sprintf("controller_manifest.tsv line %d duplicates controller row from line %d for %s", row.Line, previous, rowKey))
			continue
		}
		seenRows[rowKey] = row.Line
		nextCount, nextPairs, rowProblems := validateManifestRow(formalRoot, row, sourceIndex, strategy)
		diagramCount += nextCount
		renderedPairs = append(renderedPairs, nextPairs...)
		problems = append(problems, rowProblems...)
	}

	return diagramCount, renderedPairs, problems
}

func validateOrphanedControllerArtifacts(formalRoot string, desired map[string]struct{}) []string {
	controllersRoot := filepath.Join(formalRoot, "controllers")
	stale, err := staleControllerArtifactRoots(formalRoot, controllersRoot, desired)
	if err != nil {
		return []string{fmt.Sprintf("controllers: %v", err)}
	}

	problems := make([]string, 0, len(stale))
	for _, rel := range stale {
		problems = append(problems, fmt.Sprintf("%s: stale controller artifacts are not referenced by controller_manifest.tsv", filepath.ToSlash(rel)))
	}
	return problems
}

func staleControllerArtifactRoots(formalRoot string, controllersRoot string, desired map[string]struct{}) ([]string, error) {
	if _, err := os.Stat(controllersRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	stale := map[string]struct{}{}
	if err := filepath.WalkDir(controllersRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, ok, skipDir, err := controllerArtifactRoot(formalRoot, path, d)
		if err != nil {
			return err
		}
		if ok {
			if _, desiredRoot := desired[rel]; !desiredRoot {
				stale[rel] = struct{}{}
			}
		}
		if skipDir {
			return fs.SkipDir
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return sortedStringSet(stale), nil
}

func controllerArtifactRoot(formalRoot string, path string, d fs.DirEntry) (string, bool, bool, error) {
	if d.IsDir() {
		if d.Name() != "diagrams" {
			return "", false, false, nil
		}
		rel, err := filepath.Rel(formalRoot, filepath.Dir(path))
		if err != nil {
			return "", false, false, err
		}
		return filepath.Clean(rel), true, true, nil
	}

	if d.Name() != "spec.cfg" && d.Name() != "logic-gaps.md" {
		return "", false, false, nil
	}

	rel, err := filepath.Rel(formalRoot, filepath.Dir(path))
	if err != nil {
		return "", false, false, err
	}
	return filepath.Clean(rel), true, false, nil
}

func sortedStringSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func validateOrphanedImportArtifacts(formalRoot string, desired map[string]struct{}) []string {
	importsRoot := filepath.Join(formalRoot, "imports")
	if _, err := os.Stat(importsRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []string{fmt.Sprintf("imports: %v", err)}
	}

	var stale []string
	if err := filepath.WalkDir(importsRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".json" {
			return nil
		}

		rel, err := filepath.Rel(formalRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.Clean(rel)
		if _, ok := desired[rel]; !ok {
			stale = append(stale, rel)
		}
		return nil
	}); err != nil {
		return []string{fmt.Sprintf("imports: %v", err)}
	}

	sort.Strings(stale)
	problems := make([]string, 0, len(stale))
	for _, rel := range stale {
		problems = append(problems, fmt.Sprintf("%s: stale import artifact is not referenced by controller_manifest.tsv", filepath.ToSlash(rel)))
	}
	return problems
}

func validateSourceLockSchema(lockFile sourceLockFile) error {
	if lockFile.SchemaVersion != currentSchemaVersion {
		return fmt.Errorf("sources.lock: schemaVersion=%d, want %d", lockFile.SchemaVersion, currentSchemaVersion)
	}
	if len(lockFile.Sources) == 0 {
		return fmt.Errorf("sources.lock: expected at least one provider source entry")
	}
	return nil
}

func validateSourceLockEntry(label string, source sourceLockEntry) error {
	if strings.TrimSpace(source.Name) == "" {
		return fmt.Errorf("%s: name must not be empty", label)
	}
	if !allowedProviderSurfaces.has(source.Surface) {
		return fmt.Errorf("%s: surface=%q is not allowed", label, source.Surface)
	}
	if !allowedSourceStatuses.has(source.Status) {
		return fmt.Errorf("%s: status=%q is not allowed", label, source.Status)
	}
	if source.Status == "pinned" && (strings.TrimSpace(source.Path) == "" || strings.TrimSpace(source.Revision) == "") {
		return fmt.Errorf("%s: pinned sources require non-empty path and revision", label)
	}
	return nil
}

func buildSourceLockIndex(lockFile sourceLockFile) (map[string]sourceLockEntry, error) {
	index := make(map[string]sourceLockEntry, len(lockFile.Sources))
	for i, source := range lockFile.Sources {
		label := fmt.Sprintf("sources.lock source %d", i+1)
		if err := validateSourceLockEntry(label, source); err != nil {
			return nil, err
		}
		if _, exists := index[source.Name]; exists {
			return nil, fmt.Errorf("sources.lock: duplicate source name %q", source.Name)
		}
		index[source.Name] = source
	}
	return index, nil
}

func manifestRowLabel(row manifestRow) string {
	return fmt.Sprintf("controller_manifest.tsv line %d", row.Line)
}

func validateManifestRowMetadata(row manifestRow) []string {
	rowLabel := manifestRowLabel(row)
	var problems []string

	for _, check := range []struct {
		label string
		value string
	}{
		{label: "service", value: row.Service},
		{label: "slug", value: row.Slug},
		{label: "kind", value: row.Kind},
	} {
		if strings.TrimSpace(check.value) == "" {
			problems = append(problems, fmt.Sprintf("%s: %s must not be empty", rowLabel, check.label))
		}
	}
	if !allowedStages.has(row.Stage) {
		problems = append(problems, fmt.Sprintf("%s: stage=%q is not allowed", rowLabel, row.Stage))
	}
	if !allowedRepoSurfaces.has(row.Surface) {
		problems = append(problems, fmt.Sprintf("%s: surface=%q is not allowed", rowLabel, row.Surface))
	}

	return problems
}

func resolveManifestRowPaths(root string, row manifestRow) (manifestResolvedPaths, []string) {
	rowLabel := manifestRowLabel(row)
	paths := manifestResolvedPaths{}
	var problems []string

	for _, target := range []struct {
		relative string
		dest     *string
	}{
		{relative: row.SpecPath, dest: &paths.SpecPath},
		{relative: row.LogicPath, dest: &paths.LogicPath},
		{relative: row.ImportPath, dest: &paths.ImportPath},
		{relative: row.DiagramDir, dest: &paths.DiagramDir},
	} {
		resolved, err := resolveWithinRoot(root, target.relative)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", rowLabel, err))
			continue
		}
		*target.dest = resolved
	}

	return paths, problems
}

func loadManifestRowArtifacts(root string, row manifestRow, paths manifestResolvedPaths, sourceIndex map[string]sourceLockEntry) (manifestRowArtifacts, []string) {
	var problems []string
	artifacts := manifestRowArtifacts{}

	specModel, hasSpec, specProblems := loadManifestRowSpec(root, row, paths.SpecPath)
	artifacts.SpecModel = specModel
	artifacts.HasSpec = hasSpec
	problems = append(problems, specProblems...)

	logicGaps, hasLogic, logicProblems := loadManifestRowLogicGaps(row, paths.LogicPath)
	artifacts.LogicGaps = logicGaps
	artifacts.HasLogic = hasLogic
	problems = append(problems, logicProblems...)

	importDoc, hasImport, importProblems := loadManifestRowImport(row, paths.ImportPath, sourceIndex)
	artifacts.ImportDoc = importDoc
	artifacts.HasImport = hasImport
	problems = append(problems, importProblems...)

	return artifacts, problems
}

func loadManifestRowSpec(root string, row manifestRow, path string) (SpecModel, bool, []string) {
	if path == "" {
		return SpecModel{}, false, nil
	}

	specValues, err := loadSpec(path)
	if err != nil {
		return SpecModel{}, false, []string{fmt.Sprintf("%s: spec %s: %v", manifestRowLabel(row), filepath.ToSlash(row.SpecPath), err)}
	}

	problems := validateSpec(root, row, specValues)
	specModel, err := parseSpecModel(row.SpecPath, specValues)
	if err != nil {
		problems = append(problems, err.Error())
		return SpecModel{}, false, problems
	}

	return specModel, true, problems
}

func loadManifestRowLogicGaps(row manifestRow, path string) (logicGapsFrontMatter, bool, []string) {
	if path == "" {
		return logicGapsFrontMatter{}, false, nil
	}

	logicGaps, err := loadLogicGaps(path)
	if err != nil {
		return logicGapsFrontMatter{}, false, []string{fmt.Sprintf("%s: logic gaps %s: %v", manifestRowLabel(row), filepath.ToSlash(row.LogicPath), err)}
	}

	return logicGaps, true, validateLogicGaps(row, logicGaps)
}

func loadManifestRowImport(row manifestRow, path string, sourceIndex map[string]sourceLockEntry) (importFile, bool, []string) {
	if path == "" {
		return importFile{}, false, nil
	}

	importDoc, err := loadImport(path)
	if err != nil {
		return importFile{}, false, []string{fmt.Sprintf("%s: import %s: %v", manifestRowLabel(row), filepath.ToSlash(row.ImportPath), err)}
	}

	return importDoc, true, validateImport(row, importDoc, sourceIndex)
}

func (artifacts manifestRowArtifacts) binding(row manifestRow) *ControllerBinding {
	if !artifacts.HasSpec || !artifacts.HasLogic || !artifacts.HasImport {
		return nil
	}

	return &ControllerBinding{
		Manifest:  row,
		Spec:      artifacts.SpecModel,
		LogicGaps: append([]LogicGap(nil), artifacts.LogicGaps.Gaps...),
		Import:    artifacts.ImportDoc,
	}
}

func validateSpecRequiredKeys(specPath string, values map[string]string) []string {
	requiredKeys := []string{
		"schema_version",
		"surface",
		"service",
		"slug",
		"kind",
		"stage",
		"import",
		"shared_contracts",
		"status_projection",
		"success_condition",
		"requeue_conditions",
		"delete_confirmation",
		"finalizer_policy",
		"secret_side_effects",
	}

	var problems []string
	for _, key := range requiredKeys {
		if strings.TrimSpace(values[key]) == "" {
			problems = append(problems, fmt.Sprintf("%s: missing required key %q", specPath, key))
		}
	}
	return problems
}

func validateSpecScalarFields(specPath string, row manifestRow, values map[string]string) []string {
	var problems []string

	if values["schema_version"] != fmt.Sprintf("%d", currentSchemaVersion) {
		problems = append(problems, fmt.Sprintf("%s: schema_version=%q, want %d", specPath, values["schema_version"], currentSchemaVersion))
	}
	for _, check := range []struct {
		key   string
		value string
		want  string
		label string
	}{
		{key: "service", value: values["service"], want: row.Service, label: "manifest service"},
		{key: "slug", value: values["slug"], want: row.Slug, label: "manifest slug"},
		{key: "kind", value: values["kind"], want: row.Kind, label: "manifest kind"},
		{key: "stage", value: values["stage"], want: row.Stage, label: "manifest stage"},
		{key: "import", value: values["import"], want: row.ImportPath, label: "manifest import"},
	} {
		if check.value != check.want {
			problems = append(problems, fmt.Sprintf("%s: %s=%q does not match %s %q", specPath, check.key, check.value, check.label, check.want))
		}
	}

	return problems
}

func validateSpecAllowedValues(specPath string, values map[string]string) []string {
	var problems []string

	for _, check := range []struct {
		key     string
		value   string
		allowed stringSet
	}{
		{key: "surface", value: values["surface"], allowed: allowedRepoSurfaces},
		{key: "status_projection", value: values["status_projection"], allowed: allowedStatusProjection},
		{key: "success_condition", value: values["success_condition"], allowed: allowedSuccessConditions},
		{key: "delete_confirmation", value: values["delete_confirmation"], allowed: allowedDeleteConfirmation},
		{key: "finalizer_policy", value: values["finalizer_policy"], allowed: allowedFinalizerPolicies},
		{key: "secret_side_effects", value: values["secret_side_effects"], allowed: allowedSecretSideEffects},
	} {
		if !check.allowed.has(check.value) {
			problems = append(problems, fmt.Sprintf("%s: %s=%q is not allowed", specPath, check.key, check.value))
		}
	}

	return problems
}

func validateSpecRequeueConditions(specPath string, values map[string]string) []string {
	requeueConditions := splitList(values["requeue_conditions"])
	if len(requeueConditions) == 0 {
		return []string{fmt.Sprintf("%s: requeue_conditions must contain at least one retryable condition", specPath)}
	}

	var problems []string
	for _, condition := range requeueConditions {
		if !allowedRequeueConditions.has(condition) {
			problems = append(problems, fmt.Sprintf("%s: requeue condition %q is not allowed", specPath, condition))
		}
	}
	return problems
}

func validateSpecSharedContracts(root, specPath string, values map[string]string) []string {
	sharedContracts := splitList(values["shared_contracts"])
	var problems []string

	for _, required := range requiredSharedContracts {
		if !containsString(sharedContracts, required.Path) {
			problems = append(problems, fmt.Sprintf("%s: shared_contracts must include %q", specPath, required.Path))
		}
	}
	for _, contract := range sharedContracts {
		if _, err := resolveWithinRoot(root, contract); err != nil {
			problems = append(problems, fmt.Sprintf("%s: shared contract %q: %v", specPath, contract, err))
		}
	}

	return problems
}

func validateImportMetadata(importPath string, row manifestRow, doc importFile, sourceIndex map[string]sourceLockEntry) []string {
	var problems []string

	if doc.SchemaVersion != currentSchemaVersion {
		problems = append(problems, fmt.Sprintf("%s: schemaVersion=%d, want %d", importPath, doc.SchemaVersion, currentSchemaVersion))
	}
	if !allowedProviderSurfaces.has(doc.Surface) {
		problems = append(problems, fmt.Sprintf("%s: surface=%q is not allowed", importPath, doc.Surface))
	}
	for _, check := range []struct {
		key   string
		value string
		want  string
		label string
	}{
		{key: "service", value: doc.Service, want: row.Service, label: "manifest service"},
		{key: "slug", value: doc.Slug, want: row.Slug, label: "manifest slug"},
		{key: "kind", value: doc.Kind, want: row.Kind, label: "manifest kind"},
	} {
		if check.value != check.want {
			problems = append(problems, fmt.Sprintf("%s: %s=%q does not match %s %q", importPath, check.key, check.value, check.label, check.want))
		}
	}
	for _, check := range []struct {
		field string
		value string
	}{
		{field: "providerResource", value: doc.ProviderResource},
		{field: "sourceRef", value: doc.SourceRef},
	} {
		if strings.TrimSpace(check.value) == "" {
			problems = append(problems, fmt.Sprintf("%s: %s must not be empty", importPath, check.field))
		}
	}

	source, ok := sourceIndex[doc.SourceRef]
	if !ok {
		problems = append(problems, fmt.Sprintf("%s: sourceRef=%q is not present in sources.lock", importPath, doc.SourceRef))
	} else if source.Surface != doc.Surface {
		problems = append(problems, fmt.Sprintf("%s: sourceRef=%q has surface %q in sources.lock, want %q", importPath, doc.SourceRef, source.Surface, doc.Surface))
	}

	return problems
}

func validateImportBoundary(importPath string, row manifestRow, boundary importBoundary) []string {
	var problems []string

	if !boundary.ProviderFactsOnly {
		problems = append(problems, fmt.Sprintf("%s: boundary.providerFactsOnly must be true", importPath))
	}
	if boundary.RepoAuthoredSpecPath != row.SpecPath {
		problems = append(problems, fmt.Sprintf("%s: boundary.repoAuthoredSpecPath=%q does not match manifest spec %q", importPath, boundary.RepoAuthoredSpecPath, row.SpecPath))
	}
	if boundary.RepoAuthoredLogicGapsPath != row.LogicPath {
		problems = append(problems, fmt.Sprintf("%s: boundary.repoAuthoredLogicGapsPath=%q does not match manifest logic_gaps %q", importPath, boundary.RepoAuthoredLogicGapsPath, row.LogicPath))
	}
	if len(boundary.ExcludedSemantics) == 0 {
		problems = append(problems, fmt.Sprintf("%s: boundary.excludedSemantics must document at least one repo-authored behavior", importPath))
	}

	return problems
}

func operationBindingGroups(ops operations) []operationBindingGroup {
	return []operationBindingGroup{
		{name: "create", bindings: ops.Create},
		{name: "get", bindings: ops.Get},
		{name: "list", bindings: ops.List},
		{name: "update", bindings: ops.Update},
		{name: "delete", bindings: ops.Delete},
	}
}

func validateOperationBindingGroup(importPath, groupName string, bindings []operationBinding) ([]string, int) {
	bindingCount := 0
	var problems []string

	for _, binding := range bindings {
		bindingCount++
		if strings.TrimSpace(binding.Operation) == "" || strings.TrimSpace(binding.RequestType) == "" || strings.TrimSpace(binding.ResponseType) == "" {
			problems = append(problems, fmt.Sprintf("%s: %s bindings require non-empty operation, requestType, and responseType", importPath, groupName))
			continue
		}

		expectedRequestType := qualifiedOperationType(binding.RequestType, binding.Operation, "Request")
		if binding.RequestType != expectedRequestType {
			problems = append(problems, fmt.Sprintf("%s: %s binding requestType=%q, want %q", importPath, binding.Operation, binding.RequestType, expectedRequestType))
		}
		expectedResponseType := qualifiedOperationType(binding.ResponseType, binding.Operation, "Response")
		if binding.ResponseType != expectedResponseType {
			problems = append(problems, fmt.Sprintf("%s: %s binding responseType=%q, want %q", importPath, binding.Operation, binding.ResponseType, expectedResponseType))
		}
	}

	return problems, bindingCount
}

func validateImportBindings(importPath string, ops operations) []string {
	bindingCount := 0
	var problems []string

	for _, group := range operationBindingGroups(ops) {
		groupProblems, nextCount := validateOperationBindingGroup(importPath, group.name, group.bindings)
		bindingCount += nextCount
		problems = append(problems, groupProblems...)
	}
	if bindingCount == 0 {
		problems = append(problems, fmt.Sprintf("%s: expected at least one OCI operation binding", importPath))
	}

	return problems
}

func validateImportLifecycle(importPath string, doc importFile) []string {
	var problems []string

	if len(doc.Lifecycle.Create.Target) == 0 {
		problems = append(problems, fmt.Sprintf("%s: lifecycle.create.target must contain at least one state", importPath))
	}
	if len(doc.DeleteConfirmation.Target) == 0 {
		problems = append(problems, fmt.Sprintf("%s: deleteConfirmation.target must contain at least one state", importPath))
	}

	return problems
}

func validateImportListLookup(importPath string, listBindings []operationBinding, lookup *listLookup) []string {
	if len(listBindings) == 0 {
		return nil
	}
	if lookup == nil {
		return []string{fmt.Sprintf("%s: listLookup must be present when list bindings exist", importPath)}
	}

	var problems []string
	for _, check := range []struct {
		field string
		value string
	}{
		{field: "datasource", value: lookup.Datasource},
		{field: "collectionField", value: lookup.CollectionField},
		{field: "responseItemsField", value: lookup.ResponseItemsField},
	} {
		if strings.TrimSpace(check.value) == "" {
			problems = append(problems, fmt.Sprintf("%s: listLookup.%s must not be empty", importPath, check.field))
		}
	}

	return problems
}

func validateImportHooks(importPath string, hooks hooks) []string {
	var problems []string

	for _, hookList := range [][]hook{hooks.Create, hooks.Update, hooks.Delete} {
		for _, nextHook := range hookList {
			if strings.TrimSpace(nextHook.Helper) == "" {
				problems = append(problems, fmt.Sprintf("%s: hook helper must not be empty", importPath))
			}
		}
	}

	return problems
}

func validateImportMutation(importPath string, mutation mutation) []string {
	if mutation.ConflictsWith == nil {
		return []string{fmt.Sprintf("%s: mutation.conflictsWith must be present", importPath)}
	}
	return nil
}

func loadDiagramEntries(path, displayPath string) (map[string]os.DirEntry, []string) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: %v", filepath.ToSlash(displayPath), err)}
	}
	if !info.IsDir() {
		return nil, []string{fmt.Sprintf("%s is not a directory", filepath.ToSlash(displayPath))}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: %v", filepath.ToSlash(displayPath), err)}
	}

	present := map[string]os.DirEntry{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		present[entry.Name()] = entry
	}

	return present, nil
}

func requireDiagramEntry(path, displayDir, name string, present map[string]os.DirEntry) (string, bool, []string) {
	entry, ok := present[name]
	if !ok {
		return "", false, []string{fmt.Sprintf("%s: missing required diagram artifact %q", filepath.ToSlash(displayDir), name)}
	}
	if entry.IsDir() {
		return "", false, []string{fmt.Sprintf("%s: %q must be a file", filepath.ToSlash(displayDir), name)}
	}

	return filepath.Join(path, name), true, nil
}

func validateDiagramArtifact(row manifestRow, name, fullPath string, binding *ControllerBinding) (diagramSpec, []string) {
	displayPath := filepath.Join(row.DiagramDir, name)

	switch {
	case name == runtimeLifecycleDiagramName:
		diagram, err := loadDiagram(fullPath)
		if err != nil {
			return diagramSpec{}, []string{fmt.Sprintf("%s: %v", filepath.ToSlash(displayPath), err)}
		}
		return diagram, validateDiagram(row, displayPath, diagram, binding)
	case strings.HasSuffix(name, ".puml"):
		return diagramSpec{}, validatePlantUML(displayPath, fullPath)
	case strings.HasSuffix(name, ".svg"):
		return diagramSpec{}, validateSVG(displayPath, fullPath)
	default:
		return diagramSpec{}, nil
	}
}

func validateRequiredDiagramArtifacts(path string, row manifestRow, present map[string]os.DirEntry, binding *ControllerBinding) (int, diagramSpec, []string) {
	count := 0
	var runtimeDiagram diagramSpec
	var problems []string

	for _, name := range requiredDiagramFiles {
		fullPath, ok, entryProblems := requireDiagramEntry(path, row.DiagramDir, name, present)
		problems = append(problems, entryProblems...)
		if !ok {
			continue
		}

		count++
		diagram, artifactProblems := validateDiagramArtifact(row, name, fullPath, binding)
		if name == runtimeLifecycleDiagramName {
			runtimeDiagram = diagram
		}
		problems = append(problems, artifactProblems...)
	}

	return count, runtimeDiagram, problems
}

func shouldValidateRenderedControllerArtifacts(binding *ControllerBinding, runtimeDiagram diagramSpec, strategy diagramStrategy) bool {
	return binding != nil && runtimeDiagram.SchemaVersion != 0 && strategy.Activity.SchemaVersion != 0
}

func validateRenderedControllerArtifact(root string, present map[string]os.DirEntry, expected plantUMLArtifact) (plantUMLPair, bool, []string) {
	displayPath := filepath.ToSlash(expected.SourcePath)
	fullExpectedPath := filepath.Join(root, filepath.FromSlash(expected.SourcePath))
	actual, err := os.ReadFile(fullExpectedPath)
	if err != nil {
		return plantUMLPair{}, false, []string{fmt.Sprintf("%s: %v", displayPath, err)}
	}

	var problems []string
	if !bytes.Equal(actual, expected.SourceData) {
		problems = append(problems, fmt.Sprintf("%s: stale PlantUML source; run `make formal-diagrams`", displayPath))
	}
	if _, ok := present[filepath.Base(filepath.FromSlash(expected.SourcePath))]; !ok {
		return plantUMLPair{}, false, problems
	}
	if _, ok := present[filepath.Base(filepath.FromSlash(expected.RenderedPath))]; !ok {
		return plantUMLPair{}, false, problems
	}

	return plantUMLPair{
		SourcePath:   expected.SourcePath,
		RenderedPath: expected.RenderedPath,
	}, true, problems
}

func validateRenderedControllerArtifacts(root string, present map[string]os.DirEntry, binding *ControllerBinding, strategy diagramStrategy) ([]plantUMLPair, []string) {
	artifacts, err := renderDiagramArtifacts(root, *binding, strategy)
	if err != nil {
		return nil, []string{err.Error()}
	}

	var problems []string
	var renderedPairs []plantUMLPair
	for _, expected := range artifacts.plantUMLArtifacts() {
		pair, ok, artifactProblems := validateRenderedControllerArtifact(root, present, expected)
		problems = append(problems, artifactProblems...)
		if ok {
			renderedPairs = append(renderedPairs, pair)
		}
	}

	return renderedPairs, problems
}

func requireRegularFile(displayPath, fullPath string) (bool, []string) {
	info, err := os.Stat(fullPath)
	if err != nil {
		return false, []string{fmt.Sprintf("%s: %v", filepath.ToSlash(displayPath), err)}
	}
	if info.IsDir() {
		return false, []string{fmt.Sprintf("%s: must be a file", filepath.ToSlash(displayPath))}
	}
	return true, nil
}

func validateDiagramTemplateFiles(root string) []string {
	var problems []string

	for _, required := range requiredControllerDiagramTemplateFiles {
		displayPath := filepath.Join(controllerDiagramTemplateDir, required)
		fullPath := filepath.Join(root, displayPath)
		_, nextProblems := requireRegularFile(displayPath, fullPath)
		problems = append(problems, nextProblems...)
	}

	return problems
}

func validateSharedDiagramArtifact(root string, expected sharedDiagramArtifact) (plantUMLPair, bool, []string) {
	sourcePath := filepath.Join(root, filepath.FromSlash(expected.SourcePath))
	if ok, problems := requireRegularFile(expected.SourcePath, sourcePath); !ok {
		return plantUMLPair{}, false, problems
	}

	renderedPath := filepath.Join(root, filepath.FromSlash(expected.RenderedPath))
	if ok, problems := requireRegularFile(expected.RenderedPath, renderedPath); !ok {
		return plantUMLPair{}, false, problems
	}

	actual, err := os.ReadFile(sourcePath)
	if err != nil {
		return plantUMLPair{}, false, []string{fmt.Sprintf("%s: %v", filepath.ToSlash(expected.SourcePath), err)}
	}

	var problems []string
	if !bytes.Equal(actual, expected.SourceData) {
		problems = append(problems, fmt.Sprintf("%s: stale PlantUML source; run `make formal-diagrams`", filepath.ToSlash(expected.SourcePath)))
	}
	problems = append(problems, validatePlantUML(expected.SourcePath, sourcePath)...)
	problems = append(problems, validateSVG(expected.RenderedPath, renderedPath)...)

	return plantUMLPair{
		SourcePath:   expected.SourcePath,
		RenderedPath: expected.RenderedPath,
	}, true, problems
}

func validateSharedDiagramArtifacts(root string, strategy diagramStrategy) (int, []plantUMLPair, []string) {
	count := 0
	var problems []string
	var renderedPairs []plantUMLPair

	for _, expected := range strategy.renderSharedDiagramArtifacts() {
		pair, ok, artifactProblems := validateSharedDiagramArtifact(root, expected)
		problems = append(problems, artifactProblems...)
		if !ok {
			continue
		}
		count++
		renderedPairs = append(renderedPairs, pair)
	}

	return count, renderedPairs, problems
}

func validateDiagramMetadata(row manifestRow, path string, diagram diagramSpec) []string {
	var problems []string

	if diagram.SchemaVersion != currentSchemaVersion {
		problems = append(problems, fmt.Sprintf("%s: schemaVersion=%d, want %d", filepath.ToSlash(path), diagram.SchemaVersion, currentSchemaVersion))
	}
	if !allowedRepoSurfaces.has(diagram.Surface) {
		problems = append(problems, fmt.Sprintf("%s: surface=%q is not allowed", filepath.ToSlash(path), diagram.Surface))
	}
	for _, check := range []struct {
		key   string
		value string
		want  string
		label string
	}{
		{key: "service", value: diagram.Service, want: row.Service, label: "manifest service"},
		{key: "slug", value: diagram.Slug, want: row.Slug, label: "manifest slug"},
		{key: "kind", value: diagram.Kind, want: row.Kind, label: "manifest kind"},
	} {
		if check.value != check.want {
			problems = append(problems, fmt.Sprintf("%s: %s=%q does not match %s %q", filepath.ToSlash(path), check.key, check.value, check.label, check.want))
		}
	}
	if !allowedDiagramArchetypes.has(diagram.Archetype) {
		problems = append(problems, fmt.Sprintf("%s: archetype=%q is not allowed", filepath.ToSlash(path), diagram.Archetype))
	}
	if len(diagram.States) == 0 {
		problems = append(problems, fmt.Sprintf("%s: states must contain at least one lifecycle state", filepath.ToSlash(path)))
	}

	return problems
}

func expectedDiagramArchetype(binding *ControllerBinding) string {
	if binding != nil && hasUnresolvedGap(binding.LogicGaps, "legacy-adapter") {
		return "legacy-adapter"
	}
	return "generated-service-manager"
}

func validateDiagramBinding(path string, diagram diagramSpec, binding *ControllerBinding) []string {
	if binding == nil {
		return nil
	}

	var problems []string
	expectedArchetype := expectedDiagramArchetype(binding)
	if diagram.Archetype != expectedArchetype {
		problems = append(problems, fmt.Sprintf("%s: archetype=%q does not match formal runtime contract %q", filepath.ToSlash(path), diagram.Archetype, expectedArchetype))
	}
	if !containsIgnoreCase(diagram.States, binding.Spec.SuccessCondition) {
		problems = append(problems, fmt.Sprintf("%s: states must include success_condition %q", filepath.ToSlash(path), binding.Spec.SuccessCondition))
	}
	for _, condition := range binding.Spec.RequeueConditions {
		if !containsIgnoreCase(diagram.States, condition) {
			problems = append(problems, fmt.Sprintf("%s: states must include requeue condition %q", filepath.ToSlash(path), condition))
		}
	}
	if binding.Spec.DeleteConfirmation == "not-supported" && containsIgnoreCase(diagram.States, "terminating") {
		problems = append(problems, fmt.Sprintf("%s: states must not include terminating when delete_confirmation is not-supported", filepath.ToSlash(path)))
	}

	return problems
}

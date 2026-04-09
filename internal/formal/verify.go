package formal

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const currentSchemaVersion = 1

var manifestHeader = []string{
	"service",
	"slug",
	"kind",
	"stage",
	"surface",
	"import",
	"spec",
	"logic_gaps",
	"diagram_dir",
}

var requiredSharedContracts = []sharedContractSpec{
	{Path: "shared/BaseReconcilerContract.tla", Module: "BaseReconcilerContract"},
	{Path: "shared/ControllerLifecycleSpec.tla", Module: "ControllerLifecycleSpec"},
	{Path: "shared/OSOKServiceManagerContract.tla", Module: "OSOKServiceManagerContract"},
	{Path: "shared/SecretSideEffectsContract.tla", Module: "SecretSideEffectsContract"},
}

var allowedStages = stringSet{
	"scaffold":   {},
	"seeded":     {},
	"promotable": {},
}

var allowedRepoSurfaces = stringSet{
	"repo-authored-semantics": {},
}

var allowedProviderSurfaces = stringSet{
	"provider-facts": {},
}

var allowedSourceStatuses = stringSet{
	"scaffold": {},
	"pinned":   {},
}

var allowedStatusProjection = stringSet{
	"required": {},
	"manual":   {},
}

var allowedSuccessConditions = stringSet{
	"active":   {},
	"terminal": {},
}

var allowedRequeueConditions = stringSet{
	"provisioning": {},
	"updating":     {},
	"terminating":  {},
}

var allowedDeleteConfirmation = stringSet{
	"required":      {},
	"best-effort":   {},
	"not-supported": {},
}

var allowedFinalizerPolicies = stringSet{
	"retain-until-confirmed-delete": {},
	"none":                          {},
}

var allowedSecretSideEffects = stringSet{
	"none":       {},
	"ready-only": {},
}

var allowedDiagramArchetypes = stringSet{
	"generated-service-manager": {},
	"legacy-adapter":            {},
}

var requiredDiagramFiles = []string{
	runtimeLifecycleDiagramName,
	"activity.puml",
	"activity.svg",
	"sequence.puml",
	"sequence.svg",
	"state-machine.puml",
	"state-machine.svg",
}

var allowedLogicGapCategories = stringSet{
	"bind-versus-create":       {},
	"collection-equivalence":   {},
	"delete-confirmation":      {},
	"drift-guard":              {},
	"endpoint-materialization": {},
	"legacy-adapter":           {},
	"list-lookup":              {},
	"manual-webhook":           {},
	"mutation-policy":          {},
	"seed-corpus":              {},
	"secret-input":             {},
	"secret-output":            {},
	"status-projection":        {},
	"update-guard":             {},
	"waiter-work-request":      {},
}

var allowedLogicGapStatuses = stringSet{
	"open":     {},
	"blocked":  {},
	"resolved": {},
}

type stringSet map[string]struct{}

func (s stringSet) has(value string) bool {
	_, ok := s[value]
	return ok
}

type sharedContractSpec struct {
	Path   string
	Module string
}

type Report struct {
	Root            string
	SharedContracts int
	SharedDiagrams  int
	Sources         int
	Controllers     int
	DiagramFiles    int
}

func (r Report) String() string {
	var b strings.Builder
	b.WriteString("formal verification passed\n")
	fmt.Fprintf(&b, "- root: %s\n", filepath.ToSlash(r.Root))
	fmt.Fprintf(&b, "- shared contracts: %d\n", r.SharedContracts)
	fmt.Fprintf(&b, "- shared diagrams: %d\n", r.SharedDiagrams)
	fmt.Fprintf(&b, "- sources: %d\n", r.Sources)
	fmt.Fprintf(&b, "- controller rows: %d\n", r.Controllers)
	fmt.Fprintf(&b, "- diagram files: %d\n", r.DiagramFiles)
	return b.String()
}

type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	if len(e.Problems) == 0 {
		return "formal verification failed"
	}
	var b strings.Builder
	b.WriteString("formal verification failed:\n")
	for _, problem := range e.Problems {
		b.WriteString("- ")
		b.WriteString(problem)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

type sourceLockFile struct {
	SchemaVersion int               `json:"schemaVersion"`
	Sources       []sourceLockEntry `json:"sources"`
}

type sourceLockEntry struct {
	Name     string   `json:"name"`
	Surface  string   `json:"surface"`
	Status   string   `json:"status"`
	Path     string   `json:"path,omitempty"`
	Revision string   `json:"revision,omitempty"`
	Notes    []string `json:"notes,omitempty"`
}

type manifestRow struct {
	Line       int
	Service    string
	Slug       string
	Kind       string
	Stage      string
	Surface    string
	ImportPath string
	SpecPath   string
	LogicPath  string
	DiagramDir string
}

type logicGapsFrontMatter struct {
	SchemaVersion int        `yaml:"schemaVersion"`
	Surface       string     `yaml:"surface"`
	Service       string     `yaml:"service"`
	Slug          string     `yaml:"slug"`
	Gaps          []logicGap `yaml:"gaps"`
}

type logicGap struct {
	Category      string `yaml:"category"`
	Status        string `yaml:"status"`
	StopCondition string `yaml:"stopCondition"`
}

type importFile struct {
	SchemaVersion      int            `json:"schemaVersion"`
	Surface            string         `json:"surface"`
	Service            string         `json:"service"`
	Slug               string         `json:"slug"`
	Kind               string         `json:"kind"`
	SourceRef          string         `json:"sourceRef"`
	ProviderResource   string         `json:"providerResource"`
	Operations         operations     `json:"operations"`
	Lifecycle          lifecycle      `json:"lifecycle"`
	Mutation           mutation       `json:"mutation"`
	Hooks              hooks          `json:"hooks,omitempty"`
	DeleteConfirmation lifecyclePhase `json:"deleteConfirmation"`
	ListLookup         *listLookup    `json:"listLookup,omitempty"`
	Boundary           importBoundary `json:"boundary"`
	Notes              []string       `json:"notes,omitempty"`
}

type operations struct {
	Create []operationBinding `json:"create,omitempty"`
	Get    []operationBinding `json:"get,omitempty"`
	List   []operationBinding `json:"list,omitempty"`
	Update []operationBinding `json:"update,omitempty"`
	Delete []operationBinding `json:"delete,omitempty"`
}

type operationBinding struct {
	Operation    string `json:"operation"`
	RequestType  string `json:"requestType"`
	ResponseType string `json:"responseType"`
}

type lifecycle struct {
	Create lifecyclePhase `json:"create"`
	Update lifecyclePhase `json:"update,omitempty"`
}

type lifecyclePhase struct {
	Pending []string `json:"pending,omitempty"`
	Target  []string `json:"target,omitempty"`
}

type mutation struct {
	Mutable       []string            `json:"mutable"`
	ForceNew      []string            `json:"forceNew"`
	ConflictsWith map[string][]string `json:"conflictsWith"`
}

type hooks struct {
	Create []hook `json:"create,omitempty"`
	Update []hook `json:"update,omitempty"`
	Delete []hook `json:"delete,omitempty"`
}

type hook struct {
	Helper     string `json:"helper"`
	EntityType string `json:"entityType,omitempty"`
	Action     string `json:"action,omitempty"`
}

type listLookup struct {
	Datasource         string   `json:"datasource"`
	CollectionField    string   `json:"collectionField"`
	ResponseItemsField string   `json:"responseItemsField"`
	FilterFields       []string `json:"filterFields,omitempty"`
}

type importBoundary struct {
	ProviderFactsOnly         bool     `json:"providerFactsOnly"`
	RepoAuthoredSpecPath      string   `json:"repoAuthoredSpecPath"`
	RepoAuthoredLogicGapsPath string   `json:"repoAuthoredLogicGapsPath"`
	ExcludedSemantics         []string `json:"excludedSemantics"`
}

func Verify(root string) (Report, error) {
	report := Report{Root: filepath.Clean(root)}
	formalRoot, err := normalizeFormalRoot(root)
	if err != nil {
		return report, err
	}
	report.Root = formalRoot

	if err := requireDirectory(formalRoot); err != nil {
		return report, err
	}

	problems := validateFormalDirectories(formalRoot)
	report.SharedContracts, problems = validateSharedContracts(formalRoot, problems)
	strategy, sharedDiagrams, renderedPairs, strategyProblems := validateDiagramStrategy(formalRoot)
	report.SharedDiagrams = sharedDiagrams
	problems = append(problems, strategyProblems...)

	sourceCount, sourceIndex, sourceProblems := loadVerifiedSourceIndex(formalRoot)
	report.Sources = sourceCount
	problems = append(problems, sourceProblems...)

	rows, controllerCount, manifestProblems := loadVerifiedManifest(formalRoot)
	report.Controllers = controllerCount
	problems = append(problems, manifestProblems...)
	if len(manifestProblems) == 0 {
		problems = append(problems, validateManifestOwnedArtifacts(formalRoot, rows)...)
	}

	diagramCount, rowPairs, rowProblems := validateManifestRows(formalRoot, rows, sourceIndex, strategy)
	report.DiagramFiles = diagramCount
	renderedPairs = append(renderedPairs, rowPairs...)
	problems = append(problems, rowProblems...)
	problems = append(problems, validateRenderedPlantUMLArtifacts(formalRoot, renderedPairs)...)

	if len(problems) > 0 {
		sort.Strings(problems)
		return report, &ValidationError{Problems: problems}
	}
	return report, nil
}

func validateSharedContracts(root string, problems []string) (int, []string) {
	count := 0
	for _, contract := range requiredSharedContracts {
		path, err := resolveWithinRoot(root, contract.Path)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		moduleName, err := tlaModuleName(path)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		if moduleName != contract.Module {
			problems = append(problems, fmt.Sprintf("%s declares TLA module %q, want %q", filepath.ToSlash(contract.Path), moduleName, contract.Module))
			continue
		}
		count++
	}
	return count, problems
}

func loadSourceLock(path string) (sourceLockFile, map[string]sourceLockEntry, error) {
	lockFile, err := decodeJSONFile[sourceLockFile](path)
	if err != nil {
		return sourceLockFile{}, nil, fmt.Errorf("sources.lock: %w", err)
	}
	if err := validateSourceLockSchema(lockFile); err != nil {
		return sourceLockFile{}, nil, err
	}

	index, err := buildSourceLockIndex(lockFile)
	if err != nil {
		return sourceLockFile{}, nil, err
	}

	return lockFile, index, nil
}

func loadManifest(path string) ([]manifestRow, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(bytes.NewReader(contents))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("controller_manifest.tsv: file is empty")
	}
	if !equalStrings(records[0], manifestHeader) {
		return nil, fmt.Errorf("controller_manifest.tsv: header %q does not match expected %q", records[0], manifestHeader)
	}
	if len(records) == 1 {
		return nil, fmt.Errorf("controller_manifest.tsv: expected at least one controller row")
	}

	rows := make([]manifestRow, 0, len(records)-1)
	for i, record := range records[1:] {
		line := i + 2
		if len(record) != len(manifestHeader) {
			return nil, fmt.Errorf("controller_manifest.tsv line %d: expected %d columns, got %d", line, len(manifestHeader), len(record))
		}
		rows = append(rows, manifestRow{
			Line:       line,
			Service:    strings.TrimSpace(record[0]),
			Slug:       strings.TrimSpace(record[1]),
			Kind:       strings.TrimSpace(record[2]),
			Stage:      strings.TrimSpace(record[3]),
			Surface:    strings.TrimSpace(record[4]),
			ImportPath: strings.TrimSpace(record[5]),
			SpecPath:   strings.TrimSpace(record[6]),
			LogicPath:  strings.TrimSpace(record[7]),
			DiagramDir: strings.TrimSpace(record[8]),
		})
	}

	return rows, nil
}

func validateManifestRow(root string, row manifestRow, sourceIndex map[string]sourceLockEntry, strategy diagramStrategy) (int, []plantUMLPair, []string) {
	problems := validateManifestRowMetadata(row)
	paths, pathProblems := resolveManifestRowPaths(root, row)
	problems = append(problems, pathProblems...)

	artifacts, artifactProblems := loadManifestRowArtifacts(root, row, paths, sourceIndex)
	problems = append(problems, artifactProblems...)

	if paths.DiagramDir == "" {
		return 0, nil, problems
	}

	diagramCount, renderedPairs, diagramProblems := validateDiagramDir(root, paths.DiagramDir, row, artifacts.binding(row), strategy)
	problems = append(problems, diagramProblems...)
	return diagramCount, renderedPairs, problems
}

func loadSpec(path string) (map[string]string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	values := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		parts := strings.SplitN(text, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d is not a key=value binding", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			return nil, fmt.Errorf("line %d contains an empty key or value", line)
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("line %d duplicates key %q", line, key)
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func validateSpec(root string, row manifestRow, values map[string]string) []string {
	specPath := filepath.ToSlash(row.SpecPath)
	problems := validateSpecRequiredKeys(specPath, values)
	if len(problems) > 0 {
		return problems
	}

	problems = append(problems, validateSpecScalarFields(specPath, row, values)...)
	problems = append(problems, validateSpecAllowedValues(specPath, values)...)
	problems = append(problems, validateSpecRequeueConditions(specPath, values)...)
	problems = append(problems, validateSpecSharedContracts(root, specPath, values)...)
	return problems
}

func loadLogicGaps(path string) (logicGapsFrontMatter, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return logicGapsFrontMatter{}, err
	}

	frontMatter, ok := splitFrontMatter(string(contents))
	if !ok {
		return logicGapsFrontMatter{}, fmt.Errorf("expected YAML front matter delimited by ---")
	}

	var doc logicGapsFrontMatter
	decoder := yaml.NewDecoder(strings.NewReader(frontMatter))
	decoder.KnownFields(true)
	if err := decoder.Decode(&doc); err != nil {
		return logicGapsFrontMatter{}, err
	}
	return doc, nil
}

func validateLogicGaps(row manifestRow, doc logicGapsFrontMatter) []string {
	var problems []string

	if doc.SchemaVersion != currentSchemaVersion {
		problems = append(problems, fmt.Sprintf("%s: schemaVersion=%d, want %d", filepath.ToSlash(row.LogicPath), doc.SchemaVersion, currentSchemaVersion))
	}
	if !allowedRepoSurfaces.has(doc.Surface) {
		problems = append(problems, fmt.Sprintf("%s: surface=%q is not allowed", filepath.ToSlash(row.LogicPath), doc.Surface))
	}
	if doc.Service != row.Service {
		problems = append(problems, fmt.Sprintf("%s: service=%q does not match manifest service %q", filepath.ToSlash(row.LogicPath), doc.Service, row.Service))
	}
	if doc.Slug != row.Slug {
		problems = append(problems, fmt.Sprintf("%s: slug=%q does not match manifest slug %q", filepath.ToSlash(row.LogicPath), doc.Slug, row.Slug))
	}

	seen := map[string]struct{}{}
	for _, gap := range doc.Gaps {
		if !allowedLogicGapCategories.has(gap.Category) {
			problems = append(problems, fmt.Sprintf("%s: unsupported logic gap category %q", filepath.ToSlash(row.LogicPath), gap.Category))
		}
		if !allowedLogicGapStatuses.has(gap.Status) {
			problems = append(problems, fmt.Sprintf("%s: logic gap status %q is not allowed", filepath.ToSlash(row.LogicPath), gap.Status))
		}
		if strings.TrimSpace(gap.StopCondition) == "" {
			problems = append(problems, fmt.Sprintf("%s: logic gap category %q requires a stopCondition", filepath.ToSlash(row.LogicPath), gap.Category))
		}
		if _, exists := seen[gap.Category]; exists {
			problems = append(problems, fmt.Sprintf("%s: duplicate logic gap category %q", filepath.ToSlash(row.LogicPath), gap.Category))
		}
		seen[gap.Category] = struct{}{}
	}

	return problems
}

func loadImport(path string) (importFile, error) {
	doc, err := decodeJSONFile[importFile](path)
	if err != nil {
		return importFile{}, err
	}
	return doc, nil
}

func validateImport(row manifestRow, doc importFile, sourceIndex map[string]sourceLockEntry) []string {
	importPath := filepath.ToSlash(row.ImportPath)
	var problems []string

	problems = append(problems, validateImportMetadata(importPath, row, doc, sourceIndex)...)
	problems = append(problems, validateImportBoundary(importPath, row, doc.Boundary)...)
	problems = append(problems, validateImportBindings(importPath, doc.Operations)...)
	problems = append(problems, validateImportLifecycle(importPath, doc)...)
	problems = append(problems, validateImportListLookup(importPath, doc.Operations.List, doc.ListLookup)...)
	problems = append(problems, validateImportHooks(importPath, doc.Hooks)...)
	problems = append(problems, validateImportMutation(importPath, doc.Mutation)...)
	return problems
}

func qualifiedOperationType(boundType, operation, suffix string) string {
	prefix := ""
	if idx := strings.LastIndex(boundType, "."); idx >= 0 {
		prefix = boundType[:idx+1]
	}
	return prefix + operation + suffix
}

func validateDiagramDir(root, path string, row manifestRow, binding *ControllerBinding, strategy diagramStrategy) (int, []plantUMLPair, []string) {
	present, problems := loadDiagramEntries(path, row.DiagramDir)
	if len(problems) > 0 {
		return 0, nil, problems
	}

	count, runtimeDiagram, artifactProblems := validateRequiredDiagramArtifacts(path, row, present, binding)
	problems = append(problems, artifactProblems...)
	if !shouldValidateRenderedControllerArtifacts(binding, runtimeDiagram, strategy) {
		return count, nil, problems
	}

	renderedPairs, renderProblems := validateRenderedControllerArtifacts(root, present, binding, strategy)
	problems = append(problems, renderProblems...)
	return count, renderedPairs, problems
}

func validateDiagramStrategy(root string) (diagramStrategy, int, []plantUMLPair, []string) {
	strategy, err := loadDiagramStrategy(root)
	if err != nil {
		return diagramStrategy{}, 0, nil, []string{err.Error()}
	}

	problems := validateDiagramTemplateFiles(root)
	count, renderedPairs, sharedProblems := validateSharedDiagramArtifacts(root, strategy)
	problems = append(problems, sharedProblems...)
	return strategy, count, renderedPairs, problems
}

func validateDiagram(row manifestRow, path string, diagram diagramSpec, binding *ControllerBinding) []string {
	problems := validateDiagramMetadata(row, path, diagram)
	problems = append(problems, validateDiagramBinding(path, diagram, binding)...)
	return problems
}

func validatePlantUML(displayPath, fullPath string) []string {
	contents, err := os.ReadFile(fullPath)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", filepath.ToSlash(displayPath), err)}
	}
	text := strings.TrimSpace(string(contents))
	if text == "" {
		return []string{fmt.Sprintf("%s: file must not be empty", filepath.ToSlash(displayPath))}
	}
	var problems []string
	if !strings.Contains(text, "@startuml") {
		problems = append(problems, fmt.Sprintf("%s: expected @startuml", filepath.ToSlash(displayPath)))
	}
	if !strings.Contains(text, "@enduml") {
		problems = append(problems, fmt.Sprintf("%s: expected @enduml", filepath.ToSlash(displayPath)))
	}
	return problems
}

func validateSVG(displayPath, fullPath string) []string {
	contents, err := os.ReadFile(fullPath)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", filepath.ToSlash(displayPath), err)}
	}
	text := strings.TrimSpace(string(contents))
	if text == "" {
		return []string{fmt.Sprintf("%s: file must not be empty", filepath.ToSlash(displayPath))}
	}
	if !strings.Contains(text, "<svg") {
		return []string{fmt.Sprintf("%s: expected <svg root element", filepath.ToSlash(displayPath))}
	}
	return nil
}

func requireDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func resolveWithinRoot(root, relative string) (string, error) {
	relative = strings.TrimSpace(relative)
	if relative == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("path %q must be relative to the formal root", relative)
	}
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q escapes the formal root", relative)
	}
	path := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q escapes the formal root", relative)
	}
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func tlaModuleName(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, "MODULE") {
			continue
		}
		fields := strings.Fields(line)
		for i, field := range fields {
			if field == "MODULE" && i+1 < len(fields) {
				return fields[i+1], nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("%s: expected a TLA module declaration", filepath.ToSlash(path))
}

func splitFrontMatter(contents string) (string, bool) {
	if !strings.HasPrefix(contents, "---\n") {
		return "", false
	}
	rest := strings.TrimPrefix(contents, "---\n")
	index := strings.Index(rest, "\n---\n")
	if index < 0 {
		return "", false
	}
	return rest[:index], true
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func containsIgnoreCase(values []string, candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), candidate) {
			return true
		}
	}
	return false
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func decodeJSONFile[T any](path string) (T, error) {
	var doc T
	contents, err := os.ReadFile(path)
	if err != nil {
		return doc, err
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return doc, err
	}
	return doc, nil
}

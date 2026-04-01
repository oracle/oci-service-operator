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
	var problems []string

	root = strings.TrimSpace(root)
	if root == "" {
		return report, &ValidationError{Problems: []string{"formal root must not be empty"}}
	}

	formalRoot, err := filepath.Abs(root)
	if err != nil {
		return report, err
	}
	report.Root = formalRoot

	if err := requireDirectory(formalRoot); err != nil {
		return report, err
	}

	for _, requiredDir := range []string{"shared", sharedDiagramDir, controllerDiagramTemplateDir, "controllers", "imports"} {
		if err := requireDirectory(filepath.Join(formalRoot, requiredDir)); err != nil {
			problems = append(problems, err.Error())
		}
	}

	report.SharedContracts, problems = validateSharedContracts(formalRoot, problems)
	strategy, sharedDiagrams, sharedPairs, strategyProblems := validateDiagramStrategy(formalRoot)
	report.SharedDiagrams = sharedDiagrams
	problems = append(problems, strategyProblems...)

	sourceLock, sourceIndex, err := loadSourceLock(filepath.Join(formalRoot, "sources.lock"))
	if err != nil {
		problems = append(problems, err.Error())
	} else {
		report.Sources = len(sourceLock.Sources)
	}

	rows, err := loadManifest(filepath.Join(formalRoot, "controller_manifest.tsv"))
	if err != nil {
		problems = append(problems, err.Error())
	} else {
		report.Controllers = len(rows)
	}

	renderedPairs := append([]plantUMLPair(nil), sharedPairs...)
	seenRows := map[string]int{}
	for _, row := range rows {
		rowKey := strings.Join([]string{row.Service, row.Slug, row.Kind}, "/")
		if previous, ok := seenRows[rowKey]; ok {
			problems = append(problems, fmt.Sprintf("controller_manifest.tsv line %d duplicates controller row from line %d for %s", row.Line, previous, rowKey))
			continue
		}
		seenRows[rowKey] = row.Line
		diagramCount, rowPairs, rowProblems := validateManifestRow(formalRoot, row, sourceIndex, strategy)
		report.DiagramFiles += diagramCount
		renderedPairs = append(renderedPairs, rowPairs...)
		problems = append(problems, rowProblems...)
	}
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
	if lockFile.SchemaVersion != currentSchemaVersion {
		return sourceLockFile{}, nil, fmt.Errorf("sources.lock: schemaVersion=%d, want %d", lockFile.SchemaVersion, currentSchemaVersion)
	}
	if len(lockFile.Sources) == 0 {
		return sourceLockFile{}, nil, fmt.Errorf("sources.lock: expected at least one provider source entry")
	}

	index := make(map[string]sourceLockEntry, len(lockFile.Sources))
	for i, source := range lockFile.Sources {
		label := fmt.Sprintf("sources.lock source %d", i+1)
		if strings.TrimSpace(source.Name) == "" {
			return sourceLockFile{}, nil, fmt.Errorf("%s: name must not be empty", label)
		}
		if !allowedProviderSurfaces.has(source.Surface) {
			return sourceLockFile{}, nil, fmt.Errorf("%s: surface=%q is not allowed", label, source.Surface)
		}
		if !allowedSourceStatuses.has(source.Status) {
			return sourceLockFile{}, nil, fmt.Errorf("%s: status=%q is not allowed", label, source.Status)
		}
		if source.Status == "pinned" {
			if strings.TrimSpace(source.Path) == "" || strings.TrimSpace(source.Revision) == "" {
				return sourceLockFile{}, nil, fmt.Errorf("%s: pinned sources require non-empty path and revision", label)
			}
		}
		if _, exists := index[source.Name]; exists {
			return sourceLockFile{}, nil, fmt.Errorf("sources.lock: duplicate source name %q", source.Name)
		}
		index[source.Name] = source
	}

	return lockFile, index, nil
}

func loadManifest(path string) ([]manifestRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
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
	var problems []string
	var renderedPairs []plantUMLPair
	rowLabel := fmt.Sprintf("controller_manifest.tsv line %d", row.Line)

	if row.Service == "" {
		problems = append(problems, fmt.Sprintf("%s: service must not be empty", rowLabel))
	}
	if row.Slug == "" {
		problems = append(problems, fmt.Sprintf("%s: slug must not be empty", rowLabel))
	}
	if row.Kind == "" {
		problems = append(problems, fmt.Sprintf("%s: kind must not be empty", rowLabel))
	}
	if !allowedStages.has(row.Stage) {
		problems = append(problems, fmt.Sprintf("%s: stage=%q is not allowed", rowLabel, row.Stage))
	}
	if !allowedRepoSurfaces.has(row.Surface) {
		problems = append(problems, fmt.Sprintf("%s: surface=%q is not allowed", rowLabel, row.Surface))
	}

	specPath, err := resolveWithinRoot(root, row.SpecPath)
	if err != nil {
		problems = append(problems, fmt.Sprintf("%s: %v", rowLabel, err))
	}
	logicPath, err := resolveWithinRoot(root, row.LogicPath)
	if err != nil {
		problems = append(problems, fmt.Sprintf("%s: %v", rowLabel, err))
	}
	importPath, err := resolveWithinRoot(root, row.ImportPath)
	if err != nil {
		problems = append(problems, fmt.Sprintf("%s: %v", rowLabel, err))
	}
	diagramDir, err := resolveWithinRoot(root, row.DiagramDir)
	if err != nil {
		problems = append(problems, fmt.Sprintf("%s: %v", rowLabel, err))
	}

	var (
		specValues map[string]string
		specModel  SpecModel
		logicGaps  logicGapsFrontMatter
		importDoc  importFile
		hasSpec    bool
		hasLogic   bool
		hasImport  bool
	)

	if specPath != "" {
		specValues, err = loadSpec(specPath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: spec %s: %v", rowLabel, filepath.ToSlash(row.SpecPath), err))
		} else {
			problems = append(problems, validateSpec(root, row, specValues)...)
			specModel, err = parseSpecModel(row.SpecPath, specValues)
			if err != nil {
				problems = append(problems, err.Error())
			} else {
				hasSpec = true
			}
		}
	}

	if logicPath != "" {
		logicGaps, err = loadLogicGaps(logicPath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: logic gaps %s: %v", rowLabel, filepath.ToSlash(row.LogicPath), err))
		} else {
			problems = append(problems, validateLogicGaps(row, logicGaps)...)
			hasLogic = true
		}
	}

	if importPath != "" {
		importDoc, err = loadImport(importPath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: import %s: %v", rowLabel, filepath.ToSlash(row.ImportPath), err))
		} else {
			problems = append(problems, validateImport(row, importDoc, sourceIndex)...)
			hasImport = true
		}
	}

	diagramCount := 0
	if diagramDir != "" {
		var diagramProblems []string
		var binding *ControllerBinding
		if hasSpec && hasLogic && hasImport {
			binding = &ControllerBinding{
				Manifest:  row,
				Spec:      specModel,
				LogicGaps: append([]LogicGap(nil), logicGaps.Gaps...),
				Import:    importDoc,
			}
		}
		diagramCount, renderedPairs, diagramProblems = validateDiagramDir(root, diagramDir, row, binding, strategy)
		problems = append(problems, diagramProblems...)
	}

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
	var problems []string
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
	for _, key := range requiredKeys {
		if strings.TrimSpace(values[key]) == "" {
			problems = append(problems, fmt.Sprintf("%s: missing required key %q", filepath.ToSlash(row.SpecPath), key))
		}
	}
	if len(problems) > 0 {
		return problems
	}

	if values["schema_version"] != fmt.Sprintf("%d", currentSchemaVersion) {
		problems = append(problems, fmt.Sprintf("%s: schema_version=%q, want %d", filepath.ToSlash(row.SpecPath), values["schema_version"], currentSchemaVersion))
	}
	if !allowedRepoSurfaces.has(values["surface"]) {
		problems = append(problems, fmt.Sprintf("%s: surface=%q is not allowed", filepath.ToSlash(row.SpecPath), values["surface"]))
	}
	if values["service"] != row.Service {
		problems = append(problems, fmt.Sprintf("%s: service=%q does not match manifest service %q", filepath.ToSlash(row.SpecPath), values["service"], row.Service))
	}
	if values["slug"] != row.Slug {
		problems = append(problems, fmt.Sprintf("%s: slug=%q does not match manifest slug %q", filepath.ToSlash(row.SpecPath), values["slug"], row.Slug))
	}
	if values["kind"] != row.Kind {
		problems = append(problems, fmt.Sprintf("%s: kind=%q does not match manifest kind %q", filepath.ToSlash(row.SpecPath), values["kind"], row.Kind))
	}
	if values["stage"] != row.Stage {
		problems = append(problems, fmt.Sprintf("%s: stage=%q does not match manifest stage %q", filepath.ToSlash(row.SpecPath), values["stage"], row.Stage))
	}
	if values["import"] != row.ImportPath {
		problems = append(problems, fmt.Sprintf("%s: import=%q does not match manifest import %q", filepath.ToSlash(row.SpecPath), values["import"], row.ImportPath))
	}
	if !allowedStatusProjection.has(values["status_projection"]) {
		problems = append(problems, fmt.Sprintf("%s: status_projection=%q is not allowed", filepath.ToSlash(row.SpecPath), values["status_projection"]))
	}
	if !allowedSuccessConditions.has(values["success_condition"]) {
		problems = append(problems, fmt.Sprintf("%s: success_condition=%q is not allowed", filepath.ToSlash(row.SpecPath), values["success_condition"]))
	}
	if !allowedDeleteConfirmation.has(values["delete_confirmation"]) {
		problems = append(problems, fmt.Sprintf("%s: delete_confirmation=%q is not allowed", filepath.ToSlash(row.SpecPath), values["delete_confirmation"]))
	}
	if !allowedFinalizerPolicies.has(values["finalizer_policy"]) {
		problems = append(problems, fmt.Sprintf("%s: finalizer_policy=%q is not allowed", filepath.ToSlash(row.SpecPath), values["finalizer_policy"]))
	}
	if !allowedSecretSideEffects.has(values["secret_side_effects"]) {
		problems = append(problems, fmt.Sprintf("%s: secret_side_effects=%q is not allowed", filepath.ToSlash(row.SpecPath), values["secret_side_effects"]))
	}

	requeueConditions := splitList(values["requeue_conditions"])
	if len(requeueConditions) == 0 {
		problems = append(problems, fmt.Sprintf("%s: requeue_conditions must contain at least one retryable condition", filepath.ToSlash(row.SpecPath)))
	}
	for _, condition := range requeueConditions {
		if !allowedRequeueConditions.has(condition) {
			problems = append(problems, fmt.Sprintf("%s: requeue condition %q is not allowed", filepath.ToSlash(row.SpecPath), condition))
		}
	}

	sharedContracts := splitList(values["shared_contracts"])
	for _, required := range requiredSharedContracts {
		if !containsString(sharedContracts, required.Path) {
			problems = append(problems, fmt.Sprintf("%s: shared_contracts must include %q", filepath.ToSlash(row.SpecPath), required.Path))
		}
	}
	for _, contract := range sharedContracts {
		if _, err := resolveWithinRoot(root, contract); err != nil {
			problems = append(problems, fmt.Sprintf("%s: shared contract %q: %v", filepath.ToSlash(row.SpecPath), contract, err))
		}
	}

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
	var problems []string

	if doc.SchemaVersion != currentSchemaVersion {
		problems = append(problems, fmt.Sprintf("%s: schemaVersion=%d, want %d", filepath.ToSlash(row.ImportPath), doc.SchemaVersion, currentSchemaVersion))
	}
	if !allowedProviderSurfaces.has(doc.Surface) {
		problems = append(problems, fmt.Sprintf("%s: surface=%q is not allowed", filepath.ToSlash(row.ImportPath), doc.Surface))
	}
	if doc.Service != row.Service {
		problems = append(problems, fmt.Sprintf("%s: service=%q does not match manifest service %q", filepath.ToSlash(row.ImportPath), doc.Service, row.Service))
	}
	if doc.Slug != row.Slug {
		problems = append(problems, fmt.Sprintf("%s: slug=%q does not match manifest slug %q", filepath.ToSlash(row.ImportPath), doc.Slug, row.Slug))
	}
	if doc.Kind != row.Kind {
		problems = append(problems, fmt.Sprintf("%s: kind=%q does not match manifest kind %q", filepath.ToSlash(row.ImportPath), doc.Kind, row.Kind))
	}
	if strings.TrimSpace(doc.ProviderResource) == "" {
		problems = append(problems, fmt.Sprintf("%s: providerResource must not be empty", filepath.ToSlash(row.ImportPath)))
	}
	if strings.TrimSpace(doc.SourceRef) == "" {
		problems = append(problems, fmt.Sprintf("%s: sourceRef must not be empty", filepath.ToSlash(row.ImportPath)))
	}
	source, ok := sourceIndex[doc.SourceRef]
	if !ok {
		problems = append(problems, fmt.Sprintf("%s: sourceRef=%q is not present in sources.lock", filepath.ToSlash(row.ImportPath), doc.SourceRef))
	} else if source.Surface != doc.Surface {
		problems = append(problems, fmt.Sprintf("%s: sourceRef=%q has surface %q in sources.lock, want %q", filepath.ToSlash(row.ImportPath), doc.SourceRef, source.Surface, doc.Surface))
	}

	if !doc.Boundary.ProviderFactsOnly {
		problems = append(problems, fmt.Sprintf("%s: boundary.providerFactsOnly must be true", filepath.ToSlash(row.ImportPath)))
	}
	if doc.Boundary.RepoAuthoredSpecPath != row.SpecPath {
		problems = append(problems, fmt.Sprintf("%s: boundary.repoAuthoredSpecPath=%q does not match manifest spec %q", filepath.ToSlash(row.ImportPath), doc.Boundary.RepoAuthoredSpecPath, row.SpecPath))
	}
	if doc.Boundary.RepoAuthoredLogicGapsPath != row.LogicPath {
		problems = append(problems, fmt.Sprintf("%s: boundary.repoAuthoredLogicGapsPath=%q does not match manifest logic_gaps %q", filepath.ToSlash(row.ImportPath), doc.Boundary.RepoAuthoredLogicGapsPath, row.LogicPath))
	}
	if len(doc.Boundary.ExcludedSemantics) == 0 {
		problems = append(problems, fmt.Sprintf("%s: boundary.excludedSemantics must document at least one repo-authored behavior", filepath.ToSlash(row.ImportPath)))
	}

	bindings := map[string][]operationBinding{
		"create": doc.Operations.Create,
		"get":    doc.Operations.Get,
		"list":   doc.Operations.List,
		"update": doc.Operations.Update,
		"delete": doc.Operations.Delete,
	}
	bindingCount := 0
	for operation, operationBindings := range bindings {
		for _, binding := range operationBindings {
			bindingCount++
			if strings.TrimSpace(binding.Operation) == "" || strings.TrimSpace(binding.RequestType) == "" || strings.TrimSpace(binding.ResponseType) == "" {
				problems = append(problems, fmt.Sprintf("%s: %s bindings require non-empty operation, requestType, and responseType", filepath.ToSlash(row.ImportPath), operation))
				continue
			}
			expectedRequestType := qualifiedOperationType(binding.RequestType, binding.Operation, "Request")
			if binding.RequestType != expectedRequestType {
				problems = append(problems, fmt.Sprintf("%s: %s binding requestType=%q, want %q", filepath.ToSlash(row.ImportPath), binding.Operation, binding.RequestType, expectedRequestType))
			}
			expectedResponseType := qualifiedOperationType(binding.ResponseType, binding.Operation, "Response")
			if binding.ResponseType != expectedResponseType {
				problems = append(problems, fmt.Sprintf("%s: %s binding responseType=%q, want %q", filepath.ToSlash(row.ImportPath), binding.Operation, binding.ResponseType, expectedResponseType))
			}
		}
	}
	if bindingCount == 0 {
		problems = append(problems, fmt.Sprintf("%s: expected at least one OCI operation binding", filepath.ToSlash(row.ImportPath)))
	}
	if len(doc.Lifecycle.Create.Target) == 0 {
		problems = append(problems, fmt.Sprintf("%s: lifecycle.create.target must contain at least one state", filepath.ToSlash(row.ImportPath)))
	}
	if len(doc.Operations.List) > 0 {
		if doc.ListLookup == nil {
			problems = append(problems, fmt.Sprintf("%s: listLookup must be present when list bindings exist", filepath.ToSlash(row.ImportPath)))
		} else {
			if strings.TrimSpace(doc.ListLookup.Datasource) == "" {
				problems = append(problems, fmt.Sprintf("%s: listLookup.datasource must not be empty", filepath.ToSlash(row.ImportPath)))
			}
			if strings.TrimSpace(doc.ListLookup.CollectionField) == "" {
				problems = append(problems, fmt.Sprintf("%s: listLookup.collectionField must not be empty", filepath.ToSlash(row.ImportPath)))
			}
			if strings.TrimSpace(doc.ListLookup.ResponseItemsField) == "" {
				problems = append(problems, fmt.Sprintf("%s: listLookup.responseItemsField must not be empty", filepath.ToSlash(row.ImportPath)))
			}
		}
	}
	if len(doc.DeleteConfirmation.Target) == 0 {
		problems = append(problems, fmt.Sprintf("%s: deleteConfirmation.target must contain at least one state", filepath.ToSlash(row.ImportPath)))
	}

	for _, hookList := range [][]hook{
		doc.Hooks.Create,
		doc.Hooks.Update,
		doc.Hooks.Delete,
	} {
		for _, nextHook := range hookList {
			if strings.TrimSpace(nextHook.Helper) == "" {
				problems = append(problems, fmt.Sprintf("%s: hook helper must not be empty", filepath.ToSlash(row.ImportPath)))
			}
		}
	}
	if doc.Mutation.ConflictsWith == nil {
		problems = append(problems, fmt.Sprintf("%s: mutation.conflictsWith must be present", filepath.ToSlash(row.ImportPath)))
	}

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
	var problems []string

	info, err := os.Stat(path)
	if err != nil {
		return 0, nil, []string{fmt.Sprintf("%s: %v", filepath.ToSlash(row.DiagramDir), err)}
	}
	if !info.IsDir() {
		return 0, nil, []string{fmt.Sprintf("%s is not a directory", filepath.ToSlash(row.DiagramDir))}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, nil, []string{fmt.Sprintf("%s: %v", filepath.ToSlash(row.DiagramDir), err)}
	}

	present := map[string]os.DirEntry{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		present[entry.Name()] = entry
	}

	count := 0
	var renderedPairs []plantUMLPair
	var runtimeDiagram diagramSpec
	for _, name := range requiredDiagramFiles {
		entry, ok := present[name]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: missing required diagram artifact %q", filepath.ToSlash(row.DiagramDir), name))
			continue
		}
		if entry.IsDir() {
			problems = append(problems, fmt.Sprintf("%s: %q must be a file", filepath.ToSlash(row.DiagramDir), name))
			continue
		}

		count++
		fullPath := filepath.Join(path, name)
		switch {
		case name == runtimeLifecycleDiagramName:
			diagram, err := loadDiagram(fullPath)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", filepath.ToSlash(filepath.Join(row.DiagramDir, name)), err))
				continue
			}
			runtimeDiagram = diagram
			problems = append(problems, validateDiagram(row, filepath.Join(row.DiagramDir, name), diagram, binding)...)
		case strings.HasSuffix(name, ".puml"):
			problems = append(problems, validatePlantUML(filepath.Join(row.DiagramDir, name), fullPath)...)
		case strings.HasSuffix(name, ".svg"):
			problems = append(problems, validateSVG(filepath.Join(row.DiagramDir, name), fullPath)...)
		}
	}

	if binding != nil && runtimeDiagram.SchemaVersion != 0 && strategy.Activity.SchemaVersion != 0 {
		artifacts, err := renderDiagramArtifacts(root, *binding, strategy)
		if err != nil {
			problems = append(problems, err.Error())
		} else {
			for _, expected := range artifacts.plantUMLArtifacts() {
				displayPath := filepath.ToSlash(expected.SourcePath)
				fullExpectedPath := filepath.Join(root, filepath.FromSlash(expected.SourcePath))
				actual, err := os.ReadFile(fullExpectedPath)
				if err != nil {
					problems = append(problems, fmt.Sprintf("%s: %v", displayPath, err))
					continue
				}
				if !bytes.Equal(actual, expected.SourceData) {
					problems = append(problems, fmt.Sprintf("%s: stale PlantUML source; run `make formal-diagrams`", displayPath))
				}
				if _, ok := present[filepath.Base(filepath.FromSlash(expected.SourcePath))]; ok {
					if _, ok := present[filepath.Base(filepath.FromSlash(expected.RenderedPath))]; ok {
						renderedPairs = append(renderedPairs, plantUMLPair{
							SourcePath:   expected.SourcePath,
							RenderedPath: expected.RenderedPath,
						})
					}
				}
			}
		}
	}

	return count, renderedPairs, problems
}

func validateDiagramStrategy(root string) (diagramStrategy, int, []plantUMLPair, []string) {
	var problems []string

	strategy, err := loadDiagramStrategy(root)
	if err != nil {
		return diagramStrategy{}, 0, nil, []string{err.Error()}
	}

	count := 0
	var renderedPairs []plantUMLPair
	for _, required := range requiredControllerDiagramTemplateFiles {
		fullPath := filepath.Join(root, controllerDiagramTemplateDir, required)
		info, err := os.Stat(fullPath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", filepath.ToSlash(filepath.Join(controllerDiagramTemplateDir, required)), err))
			continue
		}
		if info.IsDir() {
			problems = append(problems, fmt.Sprintf("%s: must be a file", filepath.ToSlash(filepath.Join(controllerDiagramTemplateDir, required))))
		}
	}

	for _, expected := range strategy.renderSharedDiagramArtifacts() {
		sourcePath := filepath.Join(root, filepath.FromSlash(expected.SourcePath))
		info, err := os.Stat(sourcePath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", filepath.ToSlash(expected.SourcePath), err))
			continue
		}
		if info.IsDir() {
			problems = append(problems, fmt.Sprintf("%s: must be a file", filepath.ToSlash(expected.SourcePath)))
			continue
		}
		renderedPath := filepath.Join(root, filepath.FromSlash(expected.RenderedPath))
		renderedInfo, err := os.Stat(renderedPath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", filepath.ToSlash(expected.RenderedPath), err))
			continue
		}
		if renderedInfo.IsDir() {
			problems = append(problems, fmt.Sprintf("%s: must be a file", filepath.ToSlash(expected.RenderedPath)))
			continue
		}
		count++
		actual, err := os.ReadFile(sourcePath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", filepath.ToSlash(expected.SourcePath), err))
			continue
		}
		if !bytes.Equal(actual, expected.SourceData) {
			problems = append(problems, fmt.Sprintf("%s: stale PlantUML source; run `make formal-diagrams`", filepath.ToSlash(expected.SourcePath)))
		}
		problems = append(problems, validatePlantUML(expected.SourcePath, sourcePath)...)
		problems = append(problems, validateSVG(expected.RenderedPath, renderedPath)...)
		renderedPairs = append(renderedPairs, plantUMLPair{
			SourcePath:   expected.SourcePath,
			RenderedPath: expected.RenderedPath,
		})
	}

	return strategy, count, renderedPairs, problems
}

func validateDiagram(row manifestRow, path string, diagram diagramSpec, binding *ControllerBinding) []string {
	var problems []string

	if diagram.SchemaVersion != currentSchemaVersion {
		problems = append(problems, fmt.Sprintf("%s: schemaVersion=%d, want %d", filepath.ToSlash(path), diagram.SchemaVersion, currentSchemaVersion))
	}
	if !allowedRepoSurfaces.has(diagram.Surface) {
		problems = append(problems, fmt.Sprintf("%s: surface=%q is not allowed", filepath.ToSlash(path), diagram.Surface))
	}
	if diagram.Service != row.Service {
		problems = append(problems, fmt.Sprintf("%s: service=%q does not match manifest service %q", filepath.ToSlash(path), diagram.Service, row.Service))
	}
	if diagram.Slug != row.Slug {
		problems = append(problems, fmt.Sprintf("%s: slug=%q does not match manifest slug %q", filepath.ToSlash(path), diagram.Slug, row.Slug))
	}
	if diagram.Kind != row.Kind {
		problems = append(problems, fmt.Sprintf("%s: kind=%q does not match manifest kind %q", filepath.ToSlash(path), diagram.Kind, row.Kind))
	}
	if !allowedDiagramArchetypes.has(diagram.Archetype) {
		problems = append(problems, fmt.Sprintf("%s: archetype=%q is not allowed", filepath.ToSlash(path), diagram.Archetype))
	}
	if len(diagram.States) == 0 {
		problems = append(problems, fmt.Sprintf("%s: states must contain at least one lifecycle state", filepath.ToSlash(path)))
	}
	if binding != nil {
		expectedArchetype := "generated-service-manager"
		if hasUnresolvedGap(binding.LogicGaps, "legacy-adapter") {
			expectedArchetype = "legacy-adapter"
		}
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
	}

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

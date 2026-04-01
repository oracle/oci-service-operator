package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/oracle/oci-service-operator/internal/generator"
)

type specTarget struct {
	Service     string
	Group       string
	Spec        string
	Status      string
	Name        string
	SDKMappings []sdkMapping
}

type sdkMapping struct {
	SDKStruct  string
	APISurface string
	Exclude    bool
	Reason     string
}

type apiTypeInfo struct {
	Spec   string
	Status string
}

type sdkTarget struct {
	Group string
	Type  string
}

type configuredService struct {
	Service       string
	Group         string
	Version       string
	SelectedKinds []string
}

var (
	reSDKStruct = regexp.MustCompile(`(?m)^type\s+([A-Za-z0-9]+)\s+struct\b`)

	reSDKTarget = regexp.MustCompile(`newTarget\("([a-z0-9]+)",\s*"([A-Za-z0-9]+)"`)
)

func main() {
	write := flag.Bool("write", false, "write changes to registry files")
	serviceName := flag.String("service", "", "refresh validator registries for a single configured service")
	all := flag.Bool("all", false, "refresh validator registries for the default active generator surface")
	flag.Parse()

	root, err := findRepoRoot()
	if err != nil {
		die(err)
	}

	apispecPath := filepath.Join(root, "internal", "validator", "apispec", "registry.go")
	sdkPath := filepath.Join(root, "internal", "validator", "sdk", "registry.go")

	existingAPI, err := parseExistingAPITargets(apispecPath)
	if err != nil {
		die(err)
	}
	existingSDK, err := parseExistingSDKTargets(sdkPath)
	if err != nil {
		die(err)
	}

	apiOut, sdkOut, err := generateRegistryOutputs(root, *serviceName, *all, existingAPI, existingSDK)
	if err != nil {
		die(err)
	}

	if !*write {
		reportDiff(apispecPath, apiOut)
		reportDiff(sdkPath, sdkOut)
		fmt.Println("Run with --write to apply changes.")
		return
	}

	if err := os.WriteFile(apispecPath, apiOut, 0o644); err != nil {
		die(err)
	}
	if err := os.WriteFile(sdkPath, sdkOut, 0o644); err != nil {
		die(err)
	}

	fmt.Printf("Updated %s\n", rel(root, apispecPath))
	fmt.Printf("Updated %s\n", rel(root, sdkPath))
}

func generateRegistryOutputs(root string, serviceName string, all bool, existingAPI map[string]specTarget, existingSDK []sdkTarget) ([]byte, []byte, error) {
	services, err := loadConfiguredServices(root, serviceName, all)
	if err != nil {
		return nil, nil, err
	}

	apiSpecs, err := scanConfiguredAPISpecs(root, services)
	if err != nil {
		return nil, nil, err
	}

	targets, err := buildTargets(root, services, apiSpecs, existingAPI)
	if err != nil {
		return nil, nil, err
	}
	apiOut, err := renderAPIRegistry(targets)
	if err != nil {
		return nil, nil, err
	}

	sdkTargets := buildSDKTargets(targets, existingSDK, services)
	sdkOut, err := renderSDKRegistry(sdkTargets)
	if err != nil {
		return nil, nil, err
	}

	return apiOut, sdkOut, nil
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return r
}

func parseExistingAPITargets(path string) (map[string]specTarget, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	m := make(map[string]specTarget)
	for _, declaration := range parsed.Decls {
		if err := collectExistingAPITargets(m, declaration); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func collectExistingAPITargets(targets map[string]specTarget, declaration ast.Decl) error {
	genDecl, ok := declaration.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return nil
	}

	for _, spec := range genDecl.Specs {
		composite, ok := targetsComposite(spec)
		if !ok {
			continue
		}
		if err := appendExistingAPITargets(targets, composite); err != nil {
			return err
		}
	}
	return nil
}

func targetsComposite(spec ast.Spec) (*ast.CompositeLit, bool) {
	valueSpec, ok := spec.(*ast.ValueSpec)
	if !ok || len(valueSpec.Names) != 1 || valueSpec.Names[0].Name != "targets" || len(valueSpec.Values) != 1 {
		return nil, false
	}

	composite, ok := valueSpec.Values[0].(*ast.CompositeLit)
	return composite, ok
}

func appendExistingAPITargets(targets map[string]specTarget, composite *ast.CompositeLit) error {
	for _, element := range composite.Elts {
		targetLit, ok := element.(*ast.CompositeLit)
		if !ok {
			continue
		}

		target, err := parseExistingAPITarget(targetLit)
		if err != nil {
			return err
		}
		key := target.Group + "." + target.Spec
		targets[key] = target
	}
	return nil
}

func parseExistingAPITarget(targetLit *ast.CompositeLit) (specTarget, error) {
	target := specTarget{}
	for _, element := range targetLit.Elts {
		field, value, ok := keyValueField(element)
		if !ok {
			continue
		}
		if err := applyExistingAPITargetField(&target, field, value); err != nil {
			return specTarget{}, err
		}
	}

	return target, nil
}

func keyValueField(element ast.Expr) (string, ast.Expr, bool) {
	keyValue, ok := element.(*ast.KeyValueExpr)
	if !ok {
		return "", nil, false
	}
	key, ok := keyValue.Key.(*ast.Ident)
	if !ok {
		return "", nil, false
	}
	return key.Name, keyValue.Value, true
}

func applyExistingAPITargetField(target *specTarget, field string, value ast.Expr) error {
	switch field {
	case "Name":
		return applyExistingAPITargetName(target, value)
	case "SpecType":
		return applyExistingAPITargetSpecType(target, value)
	case "StatusType":
		return applyExistingAPITargetStatusType(target, value)
	case "SDKStructs":
		return applyExistingAPITargetLegacySDKMappings(target, value)
	case "SDKMappings":
		return applyExistingAPITargetSDKMappings(target, value)
	}
	return nil
}

func applyExistingAPITargetName(target *specTarget, value ast.Expr) error {
	name, err := stringLiteralValue(value)
	if err != nil {
		return err
	}
	target.Name = name
	return nil
}

func applyExistingAPITargetSpecType(target *specTarget, value ast.Expr) error {
	group, typeName, err := parseReflectTypeSelector(value)
	if err != nil {
		return err
	}
	target.Group = group
	target.Spec = strings.TrimSuffix(typeName, "Spec")
	return nil
}

func applyExistingAPITargetStatusType(target *specTarget, value ast.Expr) error {
	_, typeName, err := parseReflectTypeSelector(value)
	if err != nil {
		return err
	}
	target.Status = typeName
	return nil
}

func applyExistingAPITargetLegacySDKMappings(target *specTarget, value ast.Expr) error {
	mappings, err := parseLegacySDKStructs(value)
	if err != nil {
		return err
	}
	target.SDKMappings = mappings
	return nil
}

func applyExistingAPITargetSDKMappings(target *specTarget, value ast.Expr) error {
	mappings, err := parseSDKMappings(value)
	if err != nil {
		return err
	}
	target.SDKMappings = mappings
	return nil
}

func parseReflectTypeSelector(expr ast.Expr) (string, string, error) {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok || len(callExpr.Args) != 1 {
		return "", "", fmt.Errorf("unexpected reflect.TypeOf expression %T", expr)
	}

	composite, ok := callExpr.Args[0].(*ast.CompositeLit)
	if !ok {
		return "", "", fmt.Errorf("unexpected reflect.TypeOf argument %T", callExpr.Args[0])
	}

	selector, ok := composite.Type.(*ast.SelectorExpr)
	if !ok {
		return "", "", fmt.Errorf("unexpected reflect.TypeOf selector %T", composite.Type)
	}

	groupIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", "", fmt.Errorf("unexpected reflect.TypeOf package selector %T", selector.X)
	}

	group := strings.TrimSuffix(groupIdent.Name, "v1beta1")
	return group, selector.Sel.Name, nil
}

func parseLegacySDKStructs(expr ast.Expr) ([]sdkMapping, error) {
	composite, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, fmt.Errorf("unexpected SDKStructs expression %T", expr)
	}

	mappings := make([]sdkMapping, 0, len(composite.Elts))
	for _, element := range composite.Elts {
		sdkStruct, err := stringLiteralValue(element)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, sdkMapping{SDKStruct: sdkStruct})
	}
	return mappings, nil
}

func parseSDKMappings(expr ast.Expr) ([]sdkMapping, error) {
	composite, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, fmt.Errorf("unexpected SDKMappings expression %T", expr)
	}

	mappings := make([]sdkMapping, 0, len(composite.Elts))
	for _, element := range composite.Elts {
		mappingLit, ok := element.(*ast.CompositeLit)
		if !ok {
			return nil, fmt.Errorf("unexpected SDKMapping element %T", element)
		}

		mapping, err := parseSDKMapping(mappingLit)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
	}
	return mappings, nil
}

func parseSDKMapping(mappingLit *ast.CompositeLit) (sdkMapping, error) {
	mapping := sdkMapping{}
	for _, field := range mappingLit.Elts {
		name, value, ok := keyValueField(field)
		if !ok {
			continue
		}
		if err := applySDKMappingField(&mapping, name, value); err != nil {
			return sdkMapping{}, err
		}
	}
	return mapping, nil
}

func applySDKMappingField(mapping *sdkMapping, field string, value ast.Expr) error {
	switch field {
	case "SDKStruct":
		sdkStruct, err := stringLiteralValue(value)
		if err != nil {
			return err
		}
		mapping.SDKStruct = sdkStruct
	case "APISurface":
		apiSurface, err := stringLiteralValue(value)
		if err != nil {
			return err
		}
		mapping.APISurface = apiSurface
	case "Exclude":
		exclude, err := boolLiteralValue(value)
		if err != nil {
			return err
		}
		mapping.Exclude = exclude
	case "Reason":
		reason, err := stringLiteralValue(value)
		if err != nil {
			return err
		}
		mapping.Reason = reason
	}
	return nil
}

func stringLiteralValue(expr ast.Expr) (string, error) {
	basicLit, ok := expr.(*ast.BasicLit)
	if !ok || basicLit.Kind != token.STRING {
		return "", fmt.Errorf("unexpected string literal expression %T", expr)
	}
	return strconv.Unquote(basicLit.Value)
}

func boolLiteralValue(expr ast.Expr) (bool, error) {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false, fmt.Errorf("unexpected bool literal expression %T", expr)
	}
	switch ident.Name {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("unexpected bool literal %q", ident.Name)
	}
}

func parseExistingSDKTargets(path string) ([]sdkTarget, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := make([]sdkTarget, 0)
	for _, m := range reSDKTarget.FindAllSubmatch(data, -1) {
		out = append(out, sdkTarget{Group: string(m[1]), Type: string(m[2])})
	}
	return out, nil
}

func loadConfiguredServices(root string, serviceName string, all bool) ([]configuredService, error) {
	configPath := filepath.Join(root, "internal", "generator", "config", "services.yaml")
	cfg, err := generator.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	serviceName, all, err = normalizeConfiguredServiceSelection(serviceName, all)
	if err != nil {
		return nil, err
	}
	selectedServices, err := cfg.SelectServices(serviceName, all)
	if err != nil {
		return nil, err
	}

	services := make([]configuredService, 0, len(selectedServices))
	for _, service := range selectedServices {
		sdkPackageBase := path.Base(strings.TrimSpace(service.SDKPackage))
		if sdkPackageBase != service.Service {
			return nil, fmt.Errorf("service %q sdkPackage %q does not match SDK package basename %q", service.Service, service.SDKPackage, sdkPackageBase)
		}
		services = append(services, configuredService{
			Service:       service.Service,
			Group:         service.Group,
			Version:       service.VersionOrDefault(cfg.DefaultVersion),
			SelectedKinds: service.SelectedKinds(),
		})
	}

	sort.SliceStable(services, func(i, j int) bool {
		gi, gj := groupOrder(services[i].Group), groupOrder(services[j].Group)
		if gi != gj {
			return gi < gj
		}
		return services[i].Group < services[j].Group
	})

	return services, nil
}

func normalizeConfiguredServiceSelection(serviceName string, all bool) (string, bool, error) {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName != "" && all {
		return "", false, fmt.Errorf("use either --all or --service, not both")
	}
	if serviceName == "" && !all {
		return "", true, nil
	}
	return serviceName, all, nil
}

func scanConfiguredAPISpecs(root string, services []configuredService) (map[string][]apiTypeInfo, error) {
	out := make(map[string][]apiTypeInfo)
	for _, service := range services {
		specs, err := scanAPISpecDir(filepath.Join(root, "api", service.Group, service.Version))
		if err != nil {
			return nil, fmt.Errorf("scan API specs for group %q: %w", service.Group, err)
		}
		specs, err = filterConfiguredAPISpecs(service, specs)
		if err != nil {
			return nil, err
		}
		if len(specs) == 0 {
			return nil, fmt.Errorf("configured API group %q has no selected spec types under api/%s/%s", service.Group, service.Group, service.Version)
		}
		out[service.Group] = specs
	}

	return out, nil
}

func filterConfiguredAPISpecs(service configuredService, specs []apiTypeInfo) ([]apiTypeInfo, error) {
	if len(service.SelectedKinds) == 0 {
		return specs, nil
	}

	selected := make(map[string]struct{}, len(service.SelectedKinds))
	for _, kind := range service.SelectedKinds {
		selected[kind] = struct{}{}
	}

	filtered := make([]apiTypeInfo, 0, len(specs))
	for _, spec := range specs {
		if _, ok := selected[spec.Spec]; !ok {
			continue
		}
		filtered = append(filtered, spec)
		delete(selected, spec.Spec)
	}

	if len(selected) == 0 {
		return filtered, nil
	}

	missing := make([]string, 0, len(selected))
	for kind := range selected {
		missing = append(missing, kind)
	}
	sort.Strings(missing)
	return nil, fmt.Errorf(
		"configured service %q selected kinds %s were not found under api/%s/%s",
		service.Service,
		strings.Join(missing, ", "),
		service.Group,
		service.Version,
	)
}

func scanAPISpecDir(dir string) ([]apiTypeInfo, error) {
	specs := make(map[string]apiTypeInfo)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		return collectAPISpecsFromEntry(path, d, walkErr, specs)
	})
	if err != nil {
		return nil, err
	}

	return sortedAPITypeInfo(specs), nil
}

func collectAPISpecsFromEntry(path string, d fs.DirEntry, walkErr error, specs map[string]apiTypeInfo) error {
	if walkErr != nil {
		return walkErr
	}
	if d.IsDir() || !strings.HasSuffix(path, "_types.go") {
		return nil
	}
	return collectAPISpecsFromFile(path, specs)
}

func collectAPISpecsFromFile(path string, specs map[string]apiTypeInfo) error {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return err
	}

	for _, declaration := range parsed.Decls {
		collectAPISpecsFromDecl(specs, declaration)
	}
	return nil
}

func collectAPISpecsFromDecl(specs map[string]apiTypeInfo, declaration ast.Decl) {
	genDecl, ok := declaration.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.TYPE {
		return
	}

	for _, spec := range genDecl.Specs {
		apiType, ok := apiTypeInfoFromSpec(spec)
		if !ok {
			continue
		}
		specs[apiType.Spec] = apiType
	}
}

func apiTypeInfoFromSpec(spec ast.Spec) (apiTypeInfo, bool) {
	typeSpec, ok := spec.(*ast.TypeSpec)
	if !ok {
		return apiTypeInfo{}, false
	}
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return apiTypeInfo{}, false
	}
	specType, statusType := resourceSurfaceTypes(structType)
	if specType == "" || !strings.HasSuffix(specType, "Spec") {
		return apiTypeInfo{}, false
	}

	specName := strings.TrimSuffix(specType, "Spec")
	return apiTypeInfo{
		Spec:   specName,
		Status: statusType,
	}, true
}

func sortedAPITypeInfo(specs map[string]apiTypeInfo) []apiTypeInfo {
	if len(specs) == 0 {
		return nil
	}

	names := make([]string, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]apiTypeInfo, 0, len(names))
	for _, name := range names {
		out = append(out, specs[name])
	}
	return out
}

func resourceSurfaceTypes(structType *ast.StructType) (string, string) {
	specType := ""
	statusType := ""
	for _, field := range structType.Fields.List {
		if len(field.Names) != 1 {
			continue
		}
		switch field.Names[0].Name {
		case "Spec":
			specType = exprTypeName(field.Type)
		case "Status":
			statusType = exprTypeName(field.Type)
		}
	}
	if specType == "" {
		return "", ""
	}
	return specType, statusType
}

func exprTypeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typed.Sel.Name
	default:
		return ""
	}
}

func buildTargets(root string, services []configuredService, apiSpecs map[string][]apiTypeInfo, existing map[string]specTarget) ([]specTarget, error) {
	out := make([]specTarget, 0)
	for _, service := range services {
		serviceTargets, err := buildServiceTargets(root, service, apiSpecs[service.Group], existing)
		if err != nil {
			return nil, err
		}
		out = append(out, serviceTargets...)
	}

	sortSpecTargets(out)
	return out, nil
}

func buildServiceTargets(root string, service configuredService, specs []apiTypeInfo, existing map[string]specTarget) ([]specTarget, error) {
	sdkStructs, err := sdkStructNamesForService(root, service)
	if err != nil {
		return nil, err
	}

	targets := make([]specTarget, 0, len(specs))
	for _, specInfo := range specs {
		targets = append(targets, buildServiceTarget(service, specInfo, sdkStructs, existing))
	}
	return targets, nil
}

func sdkStructNamesForService(root string, service configuredService) (map[string]bool, error) {
	sdkDir := filepath.Join(root, "vendor", "github.com", "oracle", "oci-go-sdk", "v65", service.Service)
	stat, err := os.Stat(sdkDir)
	if err != nil || !stat.IsDir() {
		return nil, fmt.Errorf("configured service %q SDK package dir %q not found", service.Service, sdkDir)
	}
	return scanSDKStructNames(sdkDir), nil
}

func buildServiceTarget(service configuredService, specInfo apiTypeInfo, sdkStructs map[string]bool, existing map[string]specTarget) specTarget {
	key := service.Group + "." + specInfo.Spec
	existingTarget := existing[key]
	targetName := makeTargetName(service.Group, specInfo.Spec)
	name := targetName
	if strings.TrimSpace(existingTarget.Name) != "" {
		name = existingTarget.Name
	}

	return specTarget{
		Service:     service.Service,
		Group:       service.Group,
		Spec:        specInfo.Spec,
		Status:      resolveStatusType(specInfo, false, existingTarget),
		Name:        name,
		SDKMappings: buildSDKMappings(service.Service, specInfo.Spec, sdkCandidatesForTarget(service, specInfo.Spec, targetName, sdkStructs, existingTarget), false, existingTarget),
	}
}

func sdkCandidatesForTarget(service configuredService, spec string, targetName string, sdkStructs map[string]bool, existing specTarget) []string {
	candidates := deriveSDKTypes(service.Service, spec, targetName, sdkStructs)
	candidates = appendExistingSDKCandidates(service.Service, candidates, existing.SDKMappings)
	candidates = uniqueByOrder(candidates)
	sortSDKTypeNames(candidates)
	return candidates
}

func appendExistingSDKCandidates(service string, candidates []string, mappings []sdkMapping) []string {
	for _, mapping := range mappings {
		typeName, ok := unqualifiedSDKType(mapping.SDKStruct, service)
		if !ok {
			continue
		}
		candidates = append(candidates, typeName)
	}
	return candidates
}

func sortSDKTypeNames(names []string) {
	sort.SliceStable(names, func(i, j int) bool {
		ai, aj := sdkTypeOrder(names[i]), sdkTypeOrder(names[j])
		if ai != aj {
			return ai < aj
		}
		return names[i] < names[j]
	})
}

func sortSpecTargets(targets []specTarget) {
	sort.SliceStable(targets, func(i, j int) bool {
		gi, gj := groupOrder(targets[i].Group), groupOrder(targets[j].Group)
		if gi != gj {
			return gi < gj
		}
		if targets[i].Group != targets[j].Group {
			return targets[i].Group < targets[j].Group
		}
		return targets[i].Name < targets[j].Name
	})
}

func buildSDKTargets(targets []specTarget, _ []sdkTarget, _ []configuredService) []sdkTarget {
	set := make(map[string]sdkTarget)
	for _, t := range targets {
		for _, mapping := range t.SDKMappings {
			parts := strings.Split(mapping.SDKStruct, ".")
			if len(parts) != 2 {
				continue
			}
			k := mapping.SDKStruct
			set[k] = sdkTarget{Group: parts[0], Type: parts[1]}
		}
	}
	out := make([]sdkTarget, 0, len(set))
	for _, v := range set {
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool {
		gi, gj := groupOrder(out[i].Group), groupOrder(out[j].Group)
		if gi != gj {
			return gi < gj
		}
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		ai, aj := sdkTypeOrder(out[i].Type), sdkTypeOrder(out[j].Type)
		if ai != aj {
			return ai < aj
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func buildSDKMappings(service, spec string, candidates []string, _ bool, existing specTarget) []sdkMapping {
	override := explicitAPITargetOverrides[service+"."+spec]
	existingByType := existingSDKMappingsByType(service, existing.SDKMappings)
	order := sdkMappingOrder(service, candidates, existing.SDKMappings, override)

	mappings := make([]sdkMapping, 0, len(order))
	for _, typeName := range order {
		mapping := defaultSDKMapping(service, typeName, existingByType)
		mappings = append(mappings, applySDKMappingOverride(mapping, override, typeName))
	}
	return mappings
}

func existingSDKMappingsByType(service string, mappings []sdkMapping) map[string]sdkMapping {
	existingByType := make(map[string]sdkMapping, len(mappings))
	for _, mapping := range mappings {
		typeName, ok := unqualifiedSDKType(mapping.SDKStruct, service)
		if !ok {
			continue
		}
		existingByType[typeName] = mapping
	}
	return existingByType
}

func sdkMappingOrder(service string, candidates []string, existing []sdkMapping, override apiTargetOverride) []string {
	overrideTypes := make([]string, 0, len(override.MappingOverrides))
	for typeName := range override.MappingOverrides {
		overrideTypes = append(overrideTypes, typeName)
	}
	sort.Strings(overrideTypes)

	order := make([]string, 0, len(override.SDKTypes)+len(candidates)+len(existing)+len(overrideTypes))
	order = append(order, override.SDKTypes...)
	order = append(order, overrideTypes...)
	order = append(order, candidates...)
	order = append(order, existingSDKTypeNames(service, existing)...)
	order = uniqueByOrder(order)
	sortSDKTypeNames(order)
	return order
}

func existingSDKTypeNames(service string, mappings []sdkMapping) []string {
	out := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		typeName, ok := unqualifiedSDKType(mapping.SDKStruct, service)
		if !ok {
			continue
		}
		out = append(out, typeName)
	}
	return out
}

func defaultSDKMapping(service string, typeName string, existingByType map[string]sdkMapping) sdkMapping {
	if existingMapping, ok := existingByType[typeName]; ok {
		return existingMapping
	}
	return sdkMapping{SDKStruct: service + "." + typeName}
}

func applySDKMappingOverride(mapping sdkMapping, override apiTargetOverride, typeName string) sdkMapping {
	if override.UseStatus && strings.TrimSpace(mapping.APISurface) == "" {
		mapping.APISurface = "status"
	}

	overrideMapping, ok := override.MappingOverrides[typeName]
	if !ok {
		return mapping
	}
	if strings.TrimSpace(overrideMapping.APISurface) != "" {
		mapping.APISurface = overrideMapping.APISurface
	}
	if overrideMapping.Exclude {
		mapping.Exclude = true
	}
	if strings.TrimSpace(overrideMapping.Reason) != "" {
		mapping.Reason = overrideMapping.Reason
	}
	return mapping
}

func unqualifiedSDKType(sdkStruct, service string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(sdkStruct), ".")
	if len(parts) != 2 || parts[0] != service {
		return "", false
	}
	return parts[1], true
}

func scanSDKStructNames(dir string) map[string]bool {
	out := make(map[string]bool)
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, m := range reSDKStruct.FindAllSubmatch(data, -1) {
			name := string(m[1])
			if name == "" {
				continue
			}
			if name[0] < 'A' || name[0] > 'Z' {
				continue
			}
			out[name] = true
		}
		return nil
	})
	return out
}

type apiTargetOverride struct {
	SDKTypes         []string
	UseStatus        bool
	MappingOverrides map[string]mappingOverride
}

type mappingOverride struct {
	APISurface string
	Exclude    bool
	Reason     string
}

func mappingOverridesForSurface(apiSurface string, sdkTypes ...string) map[string]mappingOverride {
	overrides := make(map[string]mappingOverride, len(sdkTypes))
	for _, sdkType := range sdkTypes {
		overrides[sdkType] = mappingOverride{APISurface: apiSurface}
	}
	return overrides
}

func specMappingOverrides(sdkTypes ...string) map[string]mappingOverride {
	return mappingOverridesForSurface("spec", sdkTypes...)
}

func statusMappingOverrides(sdkTypes ...string) map[string]mappingOverride {
	return mappingOverridesForSurface("status", sdkTypes...)
}

func excludedMappingOverrides(reason string, sdkTypes ...string) map[string]mappingOverride {
	overrides := make(map[string]mappingOverride, len(sdkTypes))
	for _, sdkType := range sdkTypes {
		overrides[sdkType] = mappingOverride{Exclude: true, Reason: reason}
	}
	return overrides
}

func mergeMappingOverrides(sets ...map[string]mappingOverride) map[string]mappingOverride {
	total := 0
	for _, set := range sets {
		total += len(set)
	}
	merged := make(map[string]mappingOverride, total)
	for _, set := range sets {
		for sdkType, override := range set {
			merged[sdkType] = override
		}
	}
	return merged
}

const (
	psqlTransportWrapperExcludedReason = "Intentionally untracked: transport wrapper type does not correspond to a top-level CRD spec or status surface."
	collectionResponseExcludedReason   = "Intentionally untracked: collection responses do not map to a singular resource status surface."
	narrowUpdateExcludedReason         = "Intentionally untracked: patch-style update payload only covers a narrow subset of the generated desired-state surface; create payload parity remains tracked on the CRD spec."
	statusDetailsExcludedReason        = "Intentionally untracked: detailed read-model broadens status coverage beyond the generated CRD status surface; primary status parity remains tracked on the resource and summary types."
	noMeaningfulStatusExcludedReason   = "Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking."
)

// Explicit overrides cover specs whose API surface or SDK names do not follow the common generator conventions.
var explicitAPITargetOverrides = map[string]apiTargetOverride{
	"artifacts.ContainerImage": {
		MappingOverrides: excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"ContainerImageCollection",
		),
	},
	"artifacts.ContainerImageSignature": {
		MappingOverrides: mergeMappingOverrides(
			statusMappingOverrides(
				"ContainerImageSignature",
				"ContainerImageSignatureSummary",
			),
			excludedMappingOverrides(
				"Intentionally untracked: collection responses do not map to a singular resource status surface.",
				"ContainerImageSignatureCollection",
			),
		),
	},
	"artifacts.ContainerRepository": {
		MappingOverrides: excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"ContainerRepositoryCollection",
		),
	},
	"artifacts.GenericArtifact": {
		MappingOverrides: excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"GenericArtifactCollection",
		),
	},
	"artifacts.Repository": {
		SDKTypes: []string{"GenericRepository", "ContainerRepository"},
		MappingOverrides: mergeMappingOverrides(
			statusMappingOverrides("GenericRepository"),
			excludedMappingOverrides(
				"Intentionally untracked: ArtifactsRepository status represents generic repositories; container repository parity is tracked on ArtifactsContainerRepository.",
				"ContainerRepository",
			),
			excludedMappingOverrides(
				"Intentionally untracked: collection responses do not map to a singular resource status surface.",
				"RepositoryCollection",
			),
		),
	},
	"certificates.CertificateAuthorityBundleVersion": {
		MappingOverrides: excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"CertificateAuthorityBundleVersionCollection",
		),
	},
	"certificates.CertificateBundleVersion": {
		MappingOverrides: excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"CertificateBundleVersionCollection",
		),
	},
	"certificatesmanagement.Association": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"Association",
			"AssociationSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"AssociationCollection",
		),
	)},
	"certificatesmanagement.CaBundle": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"CaBundle",
			"CaBundleSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"CaBundleCollection",
		),
	)},
	"certificatesmanagement.Certificate": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"Certificate",
			"CertificateSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: version summaries belong to the dedicated CertificatesManagementCertificateVersion status surface.",
			"CertificateVersionSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"CertificateCollection",
		),
	)},
	"certificatesmanagement.CertificateAuthority": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"CertificateAuthority",
			"CertificateAuthoritySummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: version summaries belong to the dedicated CertificatesManagementCertificateAuthorityVersion status surface.",
			"CertificateAuthorityVersionSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"CertificateAuthorityCollection",
		),
	)},
	"certificatesmanagement.CertificateAuthorityVersion": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"CertificateAuthorityVersion",
			"CertificateAuthorityVersionSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"CertificateAuthorityVersionCollection",
		),
	)},
	"certificatesmanagement.CertificateVersion": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"CertificateVersion",
			"CertificateVersionSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"CertificateVersionCollection",
		),
	)},
	"containerengine.Cluster": {MappingOverrides: statusMappingOverrides(
		"Cluster",
		"ClusterSummary",
	)},
	"containerengine.ClusterEndpointConfig": {MappingOverrides: specMappingOverrides(
		"ClusterEndpointConfig",
	)},
	"containerengine.ClusterOption":   {SDKTypes: []string{"ClusterOptions"}},
	"containerengine.Kubeconfig":      {SDKTypes: []string{"CreateClusterKubeconfigContentDetails"}},
	"containerengine.NodePool":        {MappingOverrides: statusMappingOverrides("NodePool", "NodePoolSummary")},
	"containerengine.NodePoolOption":  {SDKTypes: []string{"NodePoolOptions"}},
	"containerengine.VirtualNodePool": {MappingOverrides: statusMappingOverrides("VirtualNodePool", "VirtualNodePoolSummary")},
	"database.AutonomousContainerDatabase": {MappingOverrides: mergeMappingOverrides(
		excludedMappingOverrides(
			narrowUpdateExcludedReason,
			"UpdateAutonomousContainerDatabaseDetails",
		),
		excludedMappingOverrides(
			"Intentionally untracked: version summaries belong to the dedicated AutonomousContainerDatabaseVersion status surface.",
			"AutonomousContainerDatabaseVersionSummary",
		),
	)},
	"database.AutonomousDatabaseRegionalWallet": {MappingOverrides: excludedMappingOverrides(
		noMeaningfulStatusExcludedReason,
		"AutonomousDatabaseWallet",
	)},
	"database.BackupDestination": {
		SDKTypes: []string{
			"CreateNfsBackupDestinationDetails",
			"CreateRecoveryApplianceBackupDestinationDetails",
		},
		MappingOverrides: mergeMappingOverrides(
			excludedMappingOverrides(
				narrowUpdateExcludedReason,
				"UpdateBackupDestinationDetails",
			),
			excludedMappingOverrides(
				statusDetailsExcludedReason,
				"BackupDestinationDetails",
			),
		),
	},
	"database.CloudVmCluster": {MappingOverrides: excludedMappingOverrides(
		narrowUpdateExcludedReason,
		"UpdateCloudVmClusterDetails",
	)},
	"database.CloudVmClusterIormConfig": {MappingOverrides: excludedMappingOverrides(
		noMeaningfulStatusExcludedReason,
		"ExadataIormConfig",
	)},
	"database.ConsoleHistory": {MappingOverrides: excludedMappingOverrides(
		collectionResponseExcludedReason,
		"ConsoleHistoryCollection",
	)},
	"database.DataGuardAssociation": {
		SDKTypes: []string{
			"CreateDataGuardAssociationToExistingDbSystemDetails",
			"CreateDataGuardAssociationToExistingVmClusterDetails",
			"CreateDataGuardAssociationWithNewDbSystemDetails",
		},
		MappingOverrides: excludedMappingOverrides(
			narrowUpdateExcludedReason,
			"UpdateDataGuardAssociationDetails",
		),
	},
	"database.DbServer": {MappingOverrides: excludedMappingOverrides(
		statusDetailsExcludedReason,
		"DbServerDetails",
	)},
	"database.FlexComponent": {MappingOverrides: excludedMappingOverrides(
		collectionResponseExcludedReason,
		"FlexComponentCollection",
	)},
	"database.SystemVersion": {MappingOverrides: excludedMappingOverrides(
		collectionResponseExcludedReason,
		"SystemVersionCollection",
	)},
	"database.VmClusterUpdate": {MappingOverrides: excludedMappingOverrides(
		statusDetailsExcludedReason,
		"VmClusterUpdateDetails",
	)},
	"core.AllDrgAttachment":                   {SDKTypes: []string{"DrgAttachmentInfo"}, UseStatus: true},
	"core.AllowedPeerRegionsForRemotePeering": {SDKTypes: []string{"PeerRegionForRemotePeering"}, UseStatus: true},
	"core.AppCatalogListingAgreement":         {SDKTypes: []string{"AppCatalogListingResourceVersionAgreements"}},
	"core.BootVolume":                         {MappingOverrides: statusMappingOverrides("BootVolume")},
	"core.BootVolumeKmsKey":                   {MappingOverrides: specMappingOverrides("BootVolumeKmsKey")},
	"core.ByoipAllocatedRange": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("ByoipAllocatedRangeSummary"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"ByoipAllocatedRangeCollection",
		),
	)},
	"core.ByoipRange": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("ByoipRange", "ByoipRangeSummary"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"ByoipRangeCollection",
		),
	)},
	"core.CaptureFilter":          {MappingOverrides: statusMappingOverrides("CaptureFilter")},
	"core.ClusterNetwork":         {MappingOverrides: statusMappingOverrides("ClusterNetwork", "ClusterNetworkSummary")},
	"core.ClusterNetworkInstance": {SDKTypes: []string{"InstanceSummary"}, UseStatus: true},
	"core.ComputeCapacityReport":  {MappingOverrides: statusMappingOverrides("ComputeCapacityReport")},
	"core.ComputeCapacityReservation": {MappingOverrides: statusMappingOverrides(
		"ComputeCapacityReservation",
		"ComputeCapacityReservationSummary",
	)},
	"core.ComputeCapacityReservationInstance": {SDKTypes: []string{"CapacityReservationInstanceSummary"}, UseStatus: true},
	"core.ComputeCapacityTopology": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("ComputeCapacityTopology", "ComputeCapacityTopologySummary"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"ComputeCapacityTopologyCollection",
		),
	)},
	"core.ComputeCapacityTopologyComputeBareMetalHost": {SDKTypes: []string{"ComputeBareMetalHostCollection"}, MappingOverrides: excludedMappingOverrides(
		"Intentionally untracked: collection responses do not map to a singular resource status surface.",
		"ComputeBareMetalHostCollection",
	)},
	"core.ComputeCapacityTopologyComputeHpcIsland": {SDKTypes: []string{"ComputeHpcIslandCollection"}, MappingOverrides: excludedMappingOverrides(
		"Intentionally untracked: collection responses do not map to a singular resource status surface.",
		"ComputeHpcIslandCollection",
	)},
	"core.ComputeCapacityTopologyComputeNetworkBlock": {SDKTypes: []string{"ComputeNetworkBlockCollection"}, MappingOverrides: excludedMappingOverrides(
		"Intentionally untracked: collection responses do not map to a singular resource status surface.",
		"ComputeNetworkBlockCollection",
	)},
	"core.ComputeCluster": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("ComputeCluster", "ComputeClusterSummary"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"ComputeClusterCollection",
		),
	)},
	"core.ComputeImageCapabilitySchema": {MappingOverrides: statusMappingOverrides(
		"ComputeImageCapabilitySchema",
		"ComputeImageCapabilitySchemaSummary",
	)},
	"core.ConsoleHistoryContent":         {UseStatus: true},
	"core.Cpe":                           {MappingOverrides: statusMappingOverrides("Cpe")},
	"core.CpeDeviceConfigContent":        {UseStatus: true},
	"core.CrossConnect":                  {MappingOverrides: statusMappingOverrides("CrossConnect")},
	"core.CrossConnectGroup":             {MappingOverrides: statusMappingOverrides("CrossConnectGroup")},
	"core.CrossConnectLetterOfAuthority": {SDKTypes: []string{"LetterOfAuthority"}, UseStatus: true},
	"core.DhcpOption":                    {MappingOverrides: statusMappingOverrides("DhcpOptions")},
	"core.Drg":                           {MappingOverrides: statusMappingOverrides("Drg")},
	"core.DrgAttachment":                 {MappingOverrides: statusMappingOverrides("DrgAttachment")},
	"core.DrgRouteDistribution":          {MappingOverrides: statusMappingOverrides("DrgRouteDistribution")},
	"core.DrgRouteDistributionStatement": {MappingOverrides: statusMappingOverrides("DrgRouteDistributionStatement")},
	"core.DrgRouteTable":                 {MappingOverrides: statusMappingOverrides("DrgRouteTable")},
	"core.FastConnectProviderVirtualCircuitBandwidthShape": {SDKTypes: []string{"VirtualCircuitBandwidthShape"}, UseStatus: true},
	"core.IPSecConnection":                                 {MappingOverrides: statusMappingOverrides("IpSecConnection")},
	"core.IPSecConnectionTunnelRoute":                      {SDKTypes: []string{"TunnelRouteSummary"}, UseStatus: true},
	"core.IPSecConnectionTunnelSecurityAssociation":        {SDKTypes: []string{"TunnelSecurityAssociationSummary"}, UseStatus: true},
	"core.IPSecConnectionTunnelSharedSecret":               {MappingOverrides: specMappingOverrides("IpSecConnectionTunnelSharedSecret")},
	"core.Instance": {
		MappingOverrides: map[string]mappingOverride{
			"Instance":        {APISurface: "status"},
			"InstanceSummary": {APISurface: "status"},
		},
	},
	"core.InstanceConfiguration": {MappingOverrides: statusMappingOverrides(
		"InstanceConfiguration",
		"InstanceConfigurationSummary",
	)},
	"core.InstanceDevice":                   {SDKTypes: []string{"Device"}, UseStatus: true},
	"core.InstancePool":                     {MappingOverrides: statusMappingOverrides("InstancePool", "InstancePoolSummary")},
	"core.InternetGateway":                  {MappingOverrides: statusMappingOverrides("InternetGateway")},
	"core.IpsecCpeDeviceConfigContent":      {UseStatus: true},
	"core.NatGateway":                       {MappingOverrides: statusMappingOverrides("NatGateway")},
	"core.NetworkSecurityGroup":             {MappingOverrides: statusMappingOverrides("NetworkSecurityGroup")},
	"core.NetworkSecurityGroupSecurityRule": {SDKTypes: []string{"SecurityRule"}},
	"core.PrivateIp":                        {MappingOverrides: statusMappingOverrides("PrivateIp")},
	"core.PublicIpPool": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("PublicIpPool", "PublicIpPoolSummary"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"PublicIpPoolCollection",
		),
	)},
	"core.RouteTable":                        {MappingOverrides: statusMappingOverrides("RouteTable")},
	"core.SecurityList":                      {MappingOverrides: statusMappingOverrides("SecurityList")},
	"core.ServiceGateway":                    {MappingOverrides: statusMappingOverrides("ServiceGateway")},
	"core.Subnet":                            {MappingOverrides: statusMappingOverrides("Subnet")},
	"core.TunnelCpeDeviceConfigContent":      {UseStatus: true},
	"core.VirtualCircuit":                    {MappingOverrides: statusMappingOverrides("VirtualCircuit")},
	"core.Vlan":                              {MappingOverrides: statusMappingOverrides("Vlan")},
	"core.Volume":                            {MappingOverrides: statusMappingOverrides("Volume")},
	"core.VolumeBackupPolicy":                {MappingOverrides: statusMappingOverrides("VolumeBackupPolicy")},
	"core.VolumeBackupPolicyAssetAssignment": {SDKTypes: []string{"VolumeBackupPolicyAssignment"}},
	"core.VolumeGroup":                       {MappingOverrides: statusMappingOverrides("VolumeGroup")},
	"core.VolumeKmsKey":                      {MappingOverrides: specMappingOverrides("VolumeKmsKey")},
	"core.Vtap":                              {MappingOverrides: statusMappingOverrides("Vtap")},
	"core.WindowsInstanceInitialCredential":  {SDKTypes: []string{"InstanceCredentials"}, UseStatus: true},
	"dns.DomainRecord":                       {SDKTypes: []string{"Record"}},
	"dns.RRSet":                              {MappingOverrides: specMappingOverrides("RrSet")},
	"dns.ResolverEndpoint": {
		SDKTypes:         []string{"ResolverVnicEndpoint", "ResolverVnicEndpointSummary"},
		MappingOverrides: statusMappingOverrides("ResolverVnicEndpoint", "ResolverVnicEndpointSummary"),
	},
	"dns.SteeringPolicy": {MappingOverrides: statusMappingOverrides(
		"SteeringPolicy",
		"SteeringPolicySummary",
	)},
	"dns.TsigKey": {MappingOverrides: statusMappingOverrides(
		"TsigKey",
		"TsigKeySummary",
	)},
	"dns.ZoneContent":      {UseStatus: true},
	"dns.ZoneFromZoneFile": {SDKTypes: []string{"Zone"}, UseStatus: true},
	"dns.ZoneRecord":       {SDKTypes: []string{"Record"}},
	"events.Rule":          {MappingOverrides: statusMappingOverrides("Rule", "RuleSummary")},
	"functions.Application": {MappingOverrides: statusMappingOverrides(
		"Application",
		"ApplicationSummary",
	)},
	"functions.Function": {MappingOverrides: statusMappingOverrides(
		"Function",
		"FunctionSummary",
	)},
	"functions.PbfListing": {
		MappingOverrides: mergeMappingOverrides(
			statusMappingOverrides(
				"PbfListing",
				"PbfListingSummary",
			),
			excludedMappingOverrides(
				"Intentionally untracked: version summaries belong to the dedicated PbfListingVersion status surface.",
				"PbfListingVersionSummary",
			),
		),
	},
	"identity.AuthenticationPolicy": {MappingOverrides: statusMappingOverrides("AuthenticationPolicy")},
	"identity.CostTrackingTag":      {SDKTypes: []string{"Tag"}, UseStatus: true},
	"identity.DynamicGroup":         {MappingOverrides: statusMappingOverrides("DynamicGroup")},
	"identity.Group":                {MappingOverrides: statusMappingOverrides("Group")},
	"identity.IdentityProvider": {
		SDKTypes:         []string{"Saml2IdentityProvider"},
		MappingOverrides: statusMappingOverrides("Saml2IdentityProvider"),
	},
	"identity.NetworkSource": {MappingOverrides: statusMappingOverrides("NetworkSources")},
	"identity.OAuthClientCredential": {
		MappingOverrides: excludedMappingOverrides(
			"Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			"OAuth2ClientCredential",
			"OAuth2ClientCredentialSummary",
		),
	},
	"identity.OrResetUIPassword": {SDKTypes: []string{"UiPassword"}, UseStatus: true},
	"identity.Policy":            {MappingOverrides: statusMappingOverrides("Policy")},
	"identity.StandardTagNamespace": {SDKTypes: []string{
		"StandardTagNamespaceTemplate",
		"StandardTagNamespaceTemplateSummary",
	}},
	"identity.StandardTagTemplate":       {SDKTypes: []string{"StandardTagDefinitionTemplate"}},
	"identity.Tag":                       {MappingOverrides: statusMappingOverrides("Tag", "TagSummary")},
	"identity.TagNamespace":              {MappingOverrides: statusMappingOverrides("TagNamespace", "TagNamespaceSummary")},
	"identity.UserCapability":            {MappingOverrides: specMappingOverrides("UserCapabilities")},
	"identity.UserState":                 {SDKTypes: []string{"User"}, UseStatus: true},
	"identity.UserUIPasswordInformation": {SDKTypes: []string{"UiPasswordInformation"}},
	"keymanagement.EkmsPrivateEndpoint": {MappingOverrides: statusMappingOverrides(
		"EkmsPrivateEndpoint",
		"EkmsPrivateEndpointSummary",
	)},
	"keymanagement.HsmCluster": {
		MappingOverrides: mergeMappingOverrides(
			statusMappingOverrides(
				"HsmCluster",
				"HsmClusterSummary",
			),
			excludedMappingOverrides(
				"Intentionally untracked: collection responses do not map to a singular resource status surface.",
				"HsmClusterCollection",
			),
		),
	},
	"keymanagement.HsmPartition": {
		MappingOverrides: mergeMappingOverrides(
			statusMappingOverrides(
				"HsmPartition",
				"HsmPartitionSummary",
			),
			excludedMappingOverrides(
				"Intentionally untracked: collection responses do not map to a singular resource status surface.",
				"HsmPartitionCollection",
			),
		),
	},
	"keymanagement.Key": {
		MappingOverrides: mergeMappingOverrides(
			statusMappingOverrides(
				"Key",
				"KeySummary",
			),
			excludedMappingOverrides(
				"Intentionally untracked: key version summaries belong to the dedicated KeyManagementKeyVersion status surface.",
				"KeyVersionSummary",
			),
		),
	},
	"keymanagement.PreCoUserCredential": {SDKTypes: []string{"PreCoUserCredentials"}},
	"keymanagement.ReplicationStatus": {
		MappingOverrides: excludedMappingOverrides(
			"Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			"ReplicationStatusDetails",
		),
	},
	"keymanagement.VaultReplica": {
		MappingOverrides: mergeMappingOverrides(
			statusMappingOverrides("VaultReplicaSummary"),
			excludedMappingOverrides(
				"Intentionally untracked: replica detail payload is nested under KeyManagementVault status via replicaDetails.",
				"VaultReplicaDetails",
			),
		),
	},
	"limits.Quota": {MappingOverrides: statusMappingOverrides(
		"Quota",
		"QuotaSummary",
	)},
	"loadbalancer.Backend":              {MappingOverrides: statusMappingOverrides("Backend")},
	"loadbalancer.BackendSet":           {MappingOverrides: specMappingOverrides("BackendSet")},
	"loadbalancer.Certificate":          {MappingOverrides: specMappingOverrides("Certificate")},
	"loadbalancer.HealthChecker":        {MappingOverrides: specMappingOverrides("HealthChecker")},
	"loadbalancer.Hostname":             {MappingOverrides: specMappingOverrides("Hostname")},
	"loadbalancer.Listener":             {MappingOverrides: specMappingOverrides("Listener")},
	"loadbalancer.LoadBalancer":         {MappingOverrides: statusMappingOverrides("LoadBalancer")},
	"loadbalancer.NetworkSecurityGroup": {SDKTypes: []string{"UpdateNetworkSecurityGroupsDetails"}},
	"loadbalancer.PathRouteSet":         {MappingOverrides: specMappingOverrides("PathRouteSet")},
	"loadbalancer.Policy": {MappingOverrides: excludedMappingOverrides(
		"Intentionally untracked: policy catalog entries are read-only reference data and this CRD does not expose a meaningful singular status surface.",
		"LoadBalancerPolicy",
	)},
	"loadbalancer.Protocol": {MappingOverrides: excludedMappingOverrides(
		"Intentionally untracked: protocol catalog entries are read-only reference data and this CRD does not expose a meaningful singular status surface.",
		"LoadBalancerProtocol",
	)},
	"loadbalancer.RoutingPolicy":  {MappingOverrides: specMappingOverrides("RoutingPolicy")},
	"loadbalancer.RuleSet":        {MappingOverrides: specMappingOverrides("RuleSet")},
	"loadbalancer.SSLCipherSuite": {MappingOverrides: specMappingOverrides("SslCipherSuite")},
	"loadbalancer.Shape": {
		MappingOverrides: map[string]mappingOverride{
			"LoadBalancerShape": {
				Exclude: true,
				Reason:  "Intentionally untracked: shape catalog entries are read-only reference data; load balancer shape mutation parity is tracked on LoadBalancerLoadBalancerShape.",
			},
			"ShapeDetails": {
				Exclude: true,
				Reason:  "Intentionally untracked: shape catalog entries are read-only reference data; load balancer shape mutation parity is tracked on LoadBalancerLoadBalancerShape.",
			},
			"UpdateLoadBalancerShapeDetails": {
				Exclude: true,
				Reason:  "Intentionally untracked: duplicate desired-state payload is already tracked on LoadBalancerLoadBalancerShape.",
			},
		},
	},
	"logging.Log": {MappingOverrides: statusMappingOverrides("LogSummary")},
	"logging.LogGroup": {MappingOverrides: statusMappingOverrides(
		"LogGroup",
		"LogGroupSummary",
	)},
	"logging.LogSavedSearch": {MappingOverrides: statusMappingOverrides(
		"LogSavedSearch",
		"LogSavedSearchSummary",
	)},
	"logging.UnifiedAgentConfiguration": {MappingOverrides: statusMappingOverrides(
		"UnifiedAgentConfiguration",
	)},
	"monitoring.Alarm": {MappingOverrides: statusMappingOverrides(
		"Alarm",
		"AlarmSummary",
	)},
	"monitoring.AlarmHistory": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("AlarmHistoryCollection"),
		excludedMappingOverrides(
			"Intentionally untracked: alarm history entries are represented as nested elements under AlarmHistory.status.entries, not a top-level reusable status surface.",
			"AlarmHistoryEntry",
		),
	)},
	"monitoring.AlarmSuppression": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"AlarmSuppression",
			"AlarmSuppressionSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"AlarmSuppressionCollection",
		),
	)},
	"monitoring.Metric": {MappingOverrides: statusMappingOverrides("Metric")},
	"nosql.Index": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"Index",
			"IndexSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"IndexCollection",
		),
	)},
	"nosql.Table": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"Table",
			"TableSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"TableCollection",
		),
	)},
	"nosql.TableUsage": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("TableUsageSummary"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"TableUsageCollection",
		),
	)},
	"nosql.WorkRequest": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"WorkRequest",
			"WorkRequestSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"WorkRequestCollection",
		),
	)},
	"nosql.WorkRequestError": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("WorkRequestError"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"WorkRequestErrorCollection",
		),
	)},
	"nosql.WorkRequestLog": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("WorkRequestLogEntry"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"WorkRequestLogEntryCollection",
		),
	)},
	"networkloadbalancer.Backend": {MappingOverrides: mergeMappingOverrides(
		specMappingOverrides(
			"Backend",
			"BackendSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"BackendCollection",
		),
	)},
	"networkloadbalancer.BackendSet": {MappingOverrides: mergeMappingOverrides(
		specMappingOverrides(
			"BackendSet",
			"BackendSetSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"BackendSetCollection",
		),
	)},
	"networkloadbalancer.HealthChecker": {MappingOverrides: specMappingOverrides("HealthChecker")},
	"networkloadbalancer.Listener": {MappingOverrides: mergeMappingOverrides(
		specMappingOverrides("Listener", "ListenerSummary"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"ListenerCollection",
		),
	)},
	"networkloadbalancer.NetworkLoadBalancer": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"NetworkLoadBalancer",
			"NetworkLoadBalancerSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"NetworkLoadBalancerCollection",
		),
	)},
	"networkloadbalancer.NetworkLoadBalancerHealth": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"NetworkLoadBalancerHealth",
			"NetworkLoadBalancerHealthSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"NetworkLoadBalancerHealthCollection",
		),
	)},
	"networkloadbalancer.NetworkLoadBalancersPolicy": {MappingOverrides: excludedMappingOverrides(
		"Intentionally untracked: collection responses do not map to a singular resource status surface.",
		"NetworkLoadBalancersPolicyCollection",
	)},
	"networkloadbalancer.NetworkLoadBalancersProtocol": {MappingOverrides: excludedMappingOverrides(
		"Intentionally untracked: collection responses do not map to a singular resource status surface.",
		"NetworkLoadBalancersProtocolCollection",
	)},
	"networkloadbalancer.WorkRequest": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"WorkRequest",
			"WorkRequestSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"WorkRequestCollection",
		),
	)},
	"networkloadbalancer.WorkRequestError": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("WorkRequestError"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"WorkRequestErrorCollection",
		),
	)},
	"networkloadbalancer.WorkRequestLog": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("WorkRequestLogEntry"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"WorkRequestLogEntryCollection",
		),
	)},
	"networkloadbalancer.NetworkSecurityGroup": {SDKTypes: []string{"UpdateNetworkSecurityGroupsDetails"}},
	"objectstorage.Bucket": {MappingOverrides: statusMappingOverrides(
		"Bucket",
		"BucketSummary",
	)},
	"objectstorage.Namespace": {
		SDKTypes: []string{"NamespaceMetadata"},
		MappingOverrides: excludedMappingOverrides(
			"Intentionally untracked: Namespace returns the namespace string in the response body; namespace metadata parity is tracked on ObjectStorageNamespaceMetadata.",
			"NamespaceMetadata",
		),
	},
	"objectstorage.NamespaceMetadata": {MappingOverrides: statusMappingOverrides("NamespaceMetadata")},
	"objectstorage.Object": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("ObjectSummary"),
		excludedMappingOverrides(
			"Intentionally untracked: version summaries belong to the dedicated ObjectStorageObjectVersion status surface.",
			"ObjectVersionSummary",
		),
	)},
	"objectstorage.ObjectVersion": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("ObjectVersionSummary"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"ObjectVersionCollection",
		),
	)},
	"objectstorage.PreauthenticatedRequest": {MappingOverrides: statusMappingOverrides(
		"PreauthenticatedRequest",
		"PreauthenticatedRequestSummary",
	)},
	"objectstorage.RetentionRule": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"RetentionRule",
			"RetentionRuleDetails",
			"RetentionRuleSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"RetentionRuleCollection",
		),
	)},
	"objectstorage.WorkRequestLog": {MappingOverrides: statusMappingOverrides("WorkRequestLogEntry")},
	"ons.ConfirmSubscription":      {SDKTypes: []string{"ConfirmationResult"}, UseStatus: true},
	"ons.Subscription": {MappingOverrides: statusMappingOverrides(
		"Subscription",
		"SubscriptionSummary",
	)},
	"ons.Topic": {
		MappingOverrides: excludedMappingOverrides(
			"Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			"NotificationTopic",
			"NotificationTopicSummary",
		),
	},
	"ons.Unsubscription": {UseStatus: true},
	"psql.Backup": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"Backup",
			"BackupSummary",
		),
		excludedMappingOverrides(
			psqlTransportWrapperExcludedReason,
			"BackupCollection",
		),
	)},
	"psql.Configuration": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"Configuration",
			"ConfigurationSummary",
		),
		excludedMappingOverrides(
			psqlTransportWrapperExcludedReason,
			"ConfigurationCollection",
			"ConfigurationDetails",
		),
	)},
	"psql.DbSystem": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"DbSystem",
			"DbSystemSummary",
		),
		excludedMappingOverrides(
			psqlTransportWrapperExcludedReason,
			"DbSystemCollection",
		),
	)},
	"psql.DefaultConfiguration": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"DefaultConfiguration",
			"DefaultConfigurationSummary",
		),
		excludedMappingOverrides(
			psqlTransportWrapperExcludedReason,
			"DefaultConfigurationCollection",
			"DefaultConfigurationDetails",
		),
	)},
	"psql.PrimaryDbInstance": {MappingOverrides: statusMappingOverrides("PrimaryDbInstanceDetails")},
	"psql.Shape": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("ShapeSummary"),
		excludedMappingOverrides(
			psqlTransportWrapperExcludedReason,
			"ShapeCollection",
		),
	)},
	"psql.WorkRequest": {MappingOverrides: statusMappingOverrides(
		"WorkRequest",
		"WorkRequestSummary",
	)},
	"psql.WorkRequestError": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("WorkRequestError"),
		excludedMappingOverrides(
			psqlTransportWrapperExcludedReason,
			"WorkRequestErrorCollection",
		),
	)},
	"psql.WorkRequestLog": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("WorkRequestLogEntry"),
		excludedMappingOverrides(
			psqlTransportWrapperExcludedReason,
			"WorkRequestLogEntryCollection",
		),
	)},
	"queue.Channel": {MappingOverrides: excludedMappingOverrides(
		"Intentionally untracked: collection responses do not map to a singular resource status surface.",
		"ChannelCollection",
	)},
	"queue.Queue": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("Queue", "QueueSummary"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"QueueCollection",
		),
	)},
	"queue.WorkRequestError": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("WorkRequestError"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"WorkRequestErrorCollection",
		),
	)},
	"queue.WorkRequestLog": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides("WorkRequestLogEntry"),
		excludedMappingOverrides(
			"Intentionally untracked: collection responses do not map to a singular resource status surface.",
			"WorkRequestLogEntryCollection",
		),
	)},
	"mysql.WorkRequestLog": {MappingOverrides: excludedMappingOverrides(
		noMeaningfulStatusExcludedReason,
		"WorkRequestLogEntry",
	)},
	"streaming.ConnectHarness": {MappingOverrides: excludedMappingOverrides(
		narrowUpdateExcludedReason,
		"UpdateConnectHarnessDetails",
	)},
	"streaming.Stream": {
		MappingOverrides: excludedMappingOverrides(
			"Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			"Stream",
			"StreamSummary",
		),
	},
	"streaming.StreamPool": {MappingOverrides: excludedMappingOverrides(
		narrowUpdateExcludedReason,
		"UpdateStreamPoolDetails",
	)},
	"vault.Secret": {MappingOverrides: mergeMappingOverrides(
		statusMappingOverrides(
			"Secret",
			"SecretSummary",
		),
		excludedMappingOverrides(
			"Intentionally untracked: secret version summaries belong to the dedicated VaultSecretVersion status surface.",
			"SecretVersionSummary",
		),
	)},
}

func deriveSDKTypes(service, spec, targetName string, structs map[string]bool) []string {
	out := make([]string, 0)
	for _, override := range explicitAPITargetOverrides[service+"."+spec].SDKTypes {
		addIf(&out, structs, override)
	}

	bases := candidateBases(spec, targetName)
	for _, base := range bases {
		for _, candidate := range primarySDKTypeCandidates(base) {
			addIf(&out, structs, candidate)
		}
	}

	for _, base := range bases {
		for _, candidate := range fallbackSDKTypeCandidates(base) {
			addIf(&out, structs, candidate)
		}
		for _, alias := range pluralAliases(base) {
			addIf(&out, structs, alias)
		}
	}

	return uniqueByOrder(out)
}

func candidateBases(spec, targetName string) []string {
	inputs := []string{spec}
	if targetName != "" && targetName != spec {
		inputs = append(inputs, targetName)
	}

	out := make([]string, 0, len(inputs)*2)
	for _, input := range inputs {
		out = append(out, specVariants(input)...)
		if strings.HasSuffix(input, "ByName") {
			out = append(out, specVariants(strings.TrimSuffix(input, "ByName"))...)
		}
	}

	return uniqueByOrder(out)
}

func primarySDKTypeCandidates(base string) []string {
	return []string{
		"Create" + base + "Details",
		"Update" + base + "Details",
		base,
		base + "Summary",
		base + "VersionSummary",
		base + "PublicOnly",
	}
}

func fallbackSDKTypeCandidates(base string) []string {
	return []string{
		base + "Details",
		"Get" + base + "Details",
		base + "Collection",
		base + "Entry",
		base + "EntryCollection",
	}
}

func pluralAliases(base string) []string {
	aliases := []string{base + "s"}
	if strings.HasSuffix(base, "y") {
		aliases = append(aliases, strings.TrimSuffix(base, "y")+"ies")
	}
	if strings.HasSuffix(base, "ies") {
		aliases = append(aliases, strings.TrimSuffix(base, "ies")+"y")
	}
	return uniqueByOrder(aliases)
}

func addIf(out *[]string, set map[string]bool, name string) {
	if set[name] {
		*out = append(*out, name)
	}
}

func resolveStatusType(specInfo apiTypeInfo, _ bool, existing specTarget) string {
	if strings.TrimSpace(specInfo.Status) != "" {
		return specInfo.Status
	}
	return existing.Status
}

func specVariants(spec string) []string {
	variants := []string{spec}
	repl := []struct{ old, new string }{
		{"IPSec", "IpSec"},
		{"CPE", "Cpe"},
		{"VCN", "Vcn"},
		{"VLAN", "Vlan"},
		{"VNIC", "Vnic"},
		{"NAT", "Nat"},
		{"DRG", "Drg"},
		{"KMS", "Kms"},
		{"SSL", "Ssl"},
		{"UI", "Ui"},
		{"Crossconnect", "CrossConnect"},
		{"RR", "Rr"},
		{"OAuth", "OAuth2"},
	}
	v := spec
	for _, r := range repl {
		v = strings.ReplaceAll(v, r.old, r.new)
	}
	if v != spec {
		variants = append(variants, v)
	}
	return uniqueByOrder(variants)
}

func uniqueByOrder(in []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func uniqueSorted(in []string) []string {
	in = uniqueByOrder(in)
	sort.Strings(in)
	return in
}

func sdkTypeOrder(t string) int {
	switch {
	case strings.HasPrefix(t, "Create") && strings.HasSuffix(t, "Details"):
		return 0
	case strings.HasPrefix(t, "Update") && strings.HasSuffix(t, "Details"):
		return 1
	case strings.HasSuffix(t, "Details"):
		return 2
	case strings.HasSuffix(t, "VersionSummary"):
		return 4
	case strings.HasSuffix(t, "Summary"):
		return 5
	default:
		return 3
	}
}

func renderAPIRegistry(targets []specTarget) ([]byte, error) {
	apiGroups := apiRegistryGroups(targets)
	var b strings.Builder
	renderAPIRegistryPreamble(&b, apiGroups)
	renderAPITargets(&b, targets)
	renderAPIRegistryClone(&b)

	return format.Source([]byte(b.String()))
}

func apiRegistryGroups(targets []specTarget) []string {
	groups := make(map[string]bool)
	for _, t := range targets {
		groups[t.Group] = true
	}
	apiGroups := make([]string, 0, len(groups))
	for g := range groups {
		apiGroups = append(apiGroups, g)
	}
	sort.SliceStable(apiGroups, func(i, j int) bool {
		gi, gj := groupOrder(apiGroups[i]), groupOrder(apiGroups[j])
		if gi != gj {
			return gi < gj
		}
		return apiGroups[i] < apiGroups[j]
	})
	return apiGroups
}

func renderAPIRegistryPreamble(b *strings.Builder, apiGroups []string) {
	b.WriteString("package apispec\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"reflect\"\n\n")
	for _, g := range apiGroups {
		fmt.Fprintf(b, "\t%sv1beta1 \"github.com/oracle/oci-service-operator/api/%s/v1beta1\"\n", g, g)
	}
	b.WriteString(")\n\n")
	b.WriteString("type SDKMapping struct {\n\tSDKStruct  string\n\tAPISurface string\n\tExclude    bool\n\tReason     string\n}\n\n")
	b.WriteString("type Target struct {\n\tName        string\n\tSpecType    reflect.Type\n\tStatusType  reflect.Type\n\tSDKMappings []SDKMapping\n}\n\n")
	b.WriteString("var targets = []Target{\n")
}

func renderAPITargets(b *strings.Builder, targets []specTarget) {
	for _, target := range targets {
		renderAPITarget(b, target)
	}
	b.WriteString("}\n\n")
}

func renderAPITarget(b *strings.Builder, target specTarget) {
	b.WriteString("\t{\n")
	fmt.Fprintf(b, "\t\tName:     %q,\n", target.Name)
	fmt.Fprintf(b, "\t\tSpecType: reflect.TypeOf(%sv1beta1.%sSpec{}),\n", target.Group, target.Spec)
	if strings.TrimSpace(target.Status) != "" {
		fmt.Fprintf(b, "\t\tStatusType: reflect.TypeOf(%sv1beta1.%s{}),\n", target.Group, target.Status)
	}
	b.WriteString("\t\tSDKMappings: []SDKMapping{\n")
	for _, mapping := range target.SDKMappings {
		renderAPISDKMapping(b, mapping)
	}
	b.WriteString("\t\t},\n")
	b.WriteString("\t},\n")
}

func renderAPISDKMapping(b *strings.Builder, mapping sdkMapping) {
	b.WriteString("\t\t\t{\n")
	fmt.Fprintf(b, "\t\t\t\tSDKStruct: %q,\n", mapping.SDKStruct)
	if strings.TrimSpace(mapping.APISurface) != "" {
		fmt.Fprintf(b, "\t\t\t\tAPISurface: %q,\n", mapping.APISurface)
	}
	if mapping.Exclude {
		b.WriteString("\t\t\t\tExclude: true,\n")
	}
	if strings.TrimSpace(mapping.Reason) != "" {
		fmt.Fprintf(b, "\t\t\t\tReason: %q,\n", mapping.Reason)
	}
	b.WriteString("\t\t\t},\n")
}

func renderAPIRegistryClone(b *strings.Builder) {
	b.WriteString("func Targets() []Target {\n")
	b.WriteString("\tresult := make([]Target, len(targets))\n")
	b.WriteString("\tfor i := range targets {\n")
	b.WriteString("\t\tresult[i] = targets[i]\n")
	b.WriteString("\t\tif len(targets[i].SDKMappings) > 0 {\n")
	b.WriteString("\t\t\tresult[i].SDKMappings = append([]SDKMapping(nil), targets[i].SDKMappings...)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn result\n")
	b.WriteString("}\n")
}

func renderSDKRegistry(targets []sdkTarget) ([]byte, error) {
	sdkGroups := sdkRegistryGroups(targets)
	byGroup := sdkTargetsByGroup(targets)
	var b strings.Builder
	renderSDKRegistryPreamble(&b, sdkGroups)
	renderSDKSeedTargets(&b, sdkGroups, byGroup)
	renderSDKRegistryHelpers(&b)

	return format.Source([]byte(b.String()))
}

func sdkRegistryGroups(targets []sdkTarget) []string {
	groups := make(map[string]bool)
	for _, t := range targets {
		groups[t.Group] = true
	}
	groups["mysql"] = true // required for interfaceImplementations map below.

	sdkGroups := make([]string, 0, len(groups))
	for g := range groups {
		sdkGroups = append(sdkGroups, g)
	}
	sort.SliceStable(sdkGroups, func(i, j int) bool {
		gi, gj := groupOrder(sdkGroups[i]), groupOrder(sdkGroups[j])
		if gi != gj {
			return gi < gj
		}
		return sdkGroups[i] < sdkGroups[j]
	})
	return sdkGroups
}

func sdkTargetsByGroup(targets []sdkTarget) map[string][]string {
	byGroup := make(map[string][]string)
	for _, t := range targets {
		byGroup[t.Group] = append(byGroup[t.Group], t.Type)
	}
	for g := range byGroup {
		byGroup[g] = uniqueSorted(byGroup[g])
		sortSDKTypeNames(byGroup[g])
	}
	return byGroup
}

func renderSDKRegistryPreamble(b *strings.Builder, sdkGroups []string) {
	b.WriteString("package sdk\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"path\"\n")
	b.WriteString("\t\"reflect\"\n\n")
	for _, g := range sdkGroups {
		fmt.Fprintf(b, "\t\"github.com/oracle/oci-go-sdk/v65/%s\"\n", g)
	}
	b.WriteString(")\n\n")
	b.WriteString("const (\n\tmodulePath    = \"github.com/oracle/oci-go-sdk/v65\"\n\tmoduleVersion = \"v65.61.1\"\n)\n\n")
	b.WriteString("var seedTargets = []Target{\n")
}

func renderSDKSeedTargets(b *strings.Builder, sdkGroups []string, byGroup map[string][]string) {
	for _, group := range sdkGroups {
		types := byGroup[group]
		if len(types) == 0 {
			continue
		}
		fmt.Fprintf(b, "\t// %s\n", serviceComment(group))
		for _, typeName := range types {
			fmt.Fprintf(b, "\tnewTarget(%q, %q, reflect.TypeOf(%s.%s{})),\n", group, typeName, group, typeName)
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n\n")
}

func renderSDKRegistryHelpers(b *strings.Builder) {
	b.WriteString("var interfaceImplementations = map[string][]reflect.Type{\n")
	b.WriteString("\tqualifiedTypeName(reflect.TypeOf((*mysql.CreateDbSystemSourceDetails)(nil)).Elem()): {\n")
	b.WriteString("\t\treflect.TypeOf(mysql.CreateDbSystemSourceFromBackupDetails{}),\n")
	b.WriteString("\t\treflect.TypeOf(mysql.CreateDbSystemSourceFromNoneDetails{}),\n")
	b.WriteString("\t\treflect.TypeOf(mysql.CreateDbSystemSourceFromPitrDetails{}),\n")
	b.WriteString("\t\treflect.TypeOf(mysql.CreateDbSystemSourceImportFromUrlDetails{}),\n")
	b.WriteString("\t},\n")
	b.WriteString("}\n\n")

	b.WriteString("func SeedTargets() []Target {\n")
	b.WriteString("\tresult := make([]Target, len(seedTargets))\n")
	b.WriteString("\tcopy(result, seedTargets)\n")
	b.WriteString("\treturn result\n")
	b.WriteString("}\n\n")

	b.WriteString("func TargetByName(qualifiedName string) (Target, bool) {\n")
	b.WriteString("\tfor _, target := range seedTargets {\n")
	b.WriteString("\t\tif target.QualifiedName == qualifiedName {\n")
	b.WriteString("\t\t\treturn target, true\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn Target{}, false\n")
	b.WriteString("}\n\n")

	b.WriteString("func knownInterfaceImplementations(interfaceType reflect.Type) []reflect.Type {\n")
	b.WriteString("\tknown := interfaceImplementations[qualifiedTypeName(interfaceType)]\n")
	b.WriteString("\tresult := make([]reflect.Type, len(known))\n")
	b.WriteString("\tcopy(result, known)\n")
	b.WriteString("\treturn result\n")
	b.WriteString("}\n\n")

	b.WriteString("func newTarget(packageName string, typeName string, typeRef reflect.Type) Target {\n")
	b.WriteString("\treturn Target{\n")
	b.WriteString("\t\tQualifiedName: packageName + \".\" + typeName,\n")
	b.WriteString("\t\tPackageName:   packageName,\n")
	b.WriteString("\t\tTypeName:      typeName,\n")
	b.WriteString("\t\tImportPath:    typeRef.PkgPath(),\n")
	b.WriteString("\t\tReflectType:   typeRef,\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")

	b.WriteString("func qualifiedTypeName(typeRef reflect.Type) string {\n")
	b.WriteString("\treturn path.Base(typeRef.PkgPath()) + \".\" + typeRef.Name()\n")
	b.WriteString("}\n")
}

func reportDiff(path string, next []byte) {
	cur, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("%s: unreadable (%v)\n", path, err)
		return
	}
	if bytes.Equal(cur, next) {
		fmt.Printf("%s: up to date\n", path)
		return
	}
	fmt.Printf("%s: would change\n", path)
}

func makeTargetName(group, spec string) string {
	prefix := map[string]string{
		"database":               "",
		"mysql":                  "MySql",
		"streaming":              "",
		"queue":                  "",
		"functions":              "Functions",
		"nosql":                  "NoSQL",
		"objectstorage":          "ObjectStorage",
		"ons":                    "Notification",
		"logging":                "Logging",
		"psql":                   "PSQL",
		"events":                 "Events",
		"monitoring":             "Monitoring",
		"dns":                    "DNS",
		"loadbalancer":           "LoadBalancer",
		"networkloadbalancer":    "NetworkLoadBalancer",
		"artifacts":              "Artifacts",
		"certificates":           "Certificates",
		"certificatesmanagement": "CertificatesManagement",
		"containerengine":        "ContainerEngine",
		"identity":               "Identity",
		"keymanagement":          "KeyManagement",
		"limits":                 "Limits",
		"secrets":                "Secrets",
		"vault":                  "Vault",
		"core":                   "Core",
	}
	p, ok := prefix[group]
	if !ok {
		p = pascal(group)
	}
	if p == "" {
		return spec
	}
	return p + spec
}

func pascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' })
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func serviceComment(group string) string {
	labels := map[string]string{
		"database":               "Autonomous Database",
		"mysql":                  "MySQL DB System",
		"streaming":              "Streaming",
		"queue":                  "Queue",
		"functions":              "Functions",
		"nosql":                  "NoSQL",
		"objectstorage":          "Object Storage",
		"ons":                    "Notifications (ONS)",
		"logging":                "Logging",
		"psql":                   "PostgreSQL",
		"events":                 "Events",
		"monitoring":             "Monitoring",
		"dns":                    "DNS",
		"loadbalancer":           "Load Balancer",
		"networkloadbalancer":    "Network Load Balancer",
		"artifacts":              "Artifacts",
		"certificates":           "Certificates",
		"certificatesmanagement": "Certificates Management",
		"containerengine":        "Container Engine",
		"identity":               "Identity",
		"keymanagement":          "Key Management",
		"limits":                 "Limits",
		"secrets":                "Secrets",
		"vault":                  "Vault",
		"core":                   "Core VCN",
	}
	if v, ok := labels[group]; ok {
		return v + " CRD support"
	}
	return pascal(group) + " CRD support"
}

func groupOrder(group string) int {
	order := []string{
		"database", "mysql", "streaming", "queue", "functions", "nosql", "objectstorage", "ons", "logging", "psql", "events", "monitoring", "dns", "loadbalancer", "networkloadbalancer", "artifacts", "certificates", "certificatesmanagement", "containerengine", "identity", "keymanagement", "limits", "secrets", "vault", "core",
	}
	for i, g := range order {
		if g == group {
			return i
		}
	}
	return len(order) + 100
}

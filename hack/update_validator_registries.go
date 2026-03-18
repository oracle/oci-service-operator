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
	Service string
	Group   string
	Version string
}

var (
	reSDKStruct = regexp.MustCompile(`(?m)^type\s+([A-Za-z0-9]+)\s+struct\b`)

	reSDKTarget = regexp.MustCompile(`newTarget\("([a-z0-9]+)",\s*"([A-Za-z0-9]+)"`)
)

func main() {
	write := flag.Bool("write", false, "write changes to registry files")
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

	apiOut, sdkOut, err := generateRegistryOutputs(root, existingAPI, existingSDK)
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

func generateRegistryOutputs(root string, existingAPI map[string]specTarget, existingSDK []sdkTarget) ([]byte, []byte, error) {
	services, err := loadConfiguredServices(root)
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
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || valueSpec.Names[0].Name != "targets" || len(valueSpec.Values) != 1 {
				continue
			}

			composite, ok := valueSpec.Values[0].(*ast.CompositeLit)
			if !ok {
				continue
			}

			for _, element := range composite.Elts {
				targetLit, ok := element.(*ast.CompositeLit)
				if !ok {
					continue
				}

				target, err := parseExistingAPITarget(targetLit)
				if err != nil {
					return nil, err
				}
				key := target.Group + "." + target.Spec
				m[key] = target
			}
		}
	}
	return m, nil
}

func parseExistingAPITarget(targetLit *ast.CompositeLit) (specTarget, error) {
	target := specTarget{}
	for _, element := range targetLit.Elts {
		keyValue, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := keyValue.Key.(*ast.Ident)
		if !ok {
			continue
		}

		switch key.Name {
		case "Name":
			value, err := stringLiteralValue(keyValue.Value)
			if err != nil {
				return specTarget{}, err
			}
			target.Name = value
		case "SpecType":
			group, typeName, err := parseReflectTypeSelector(keyValue.Value)
			if err != nil {
				return specTarget{}, err
			}
			target.Group = group
			target.Spec = strings.TrimSuffix(typeName, "Spec")
		case "StatusType":
			_, typeName, err := parseReflectTypeSelector(keyValue.Value)
			if err != nil {
				return specTarget{}, err
			}
			target.Status = typeName
		case "SDKStructs":
			mappings, err := parseLegacySDKStructs(keyValue.Value)
			if err != nil {
				return specTarget{}, err
			}
			target.SDKMappings = mappings
		case "SDKMappings":
			mappings, err := parseSDKMappings(keyValue.Value)
			if err != nil {
				return specTarget{}, err
			}
			target.SDKMappings = mappings
		}
	}

	return target, nil
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

		mapping := sdkMapping{}
		for _, field := range mappingLit.Elts {
			keyValue, ok := field.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := keyValue.Key.(*ast.Ident)
			if !ok {
				continue
			}

			switch key.Name {
			case "SDKStruct":
				value, err := stringLiteralValue(keyValue.Value)
				if err != nil {
					return nil, err
				}
				mapping.SDKStruct = value
			case "APISurface":
				value, err := stringLiteralValue(keyValue.Value)
				if err != nil {
					return nil, err
				}
				mapping.APISurface = value
			case "Exclude":
				value, err := boolLiteralValue(keyValue.Value)
				if err != nil {
					return nil, err
				}
				mapping.Exclude = value
			case "Reason":
				value, err := stringLiteralValue(keyValue.Value)
				if err != nil {
					return nil, err
				}
				mapping.Reason = value
			}
		}
		mappings = append(mappings, mapping)
	}
	return mappings, nil
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

func loadConfiguredServices(root string) ([]configuredService, error) {
	configPath := filepath.Join(root, "internal", "generator", "config", "services.yaml")
	cfg, err := generator.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	services := make([]configuredService, 0, len(cfg.Services))
	for _, service := range cfg.Services {
		sdkPackageBase := path.Base(strings.TrimSpace(service.SDKPackage))
		if sdkPackageBase != service.Service {
			return nil, fmt.Errorf("service %q sdkPackage %q does not match SDK package basename %q", service.Service, service.SDKPackage, sdkPackageBase)
		}
		services = append(services, configuredService{
			Service: service.Service,
			Group:   service.Group,
			Version: service.VersionOrDefault(cfg.DefaultVersion),
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

func scanConfiguredAPISpecs(root string, services []configuredService) (map[string][]apiTypeInfo, error) {
	out := make(map[string][]apiTypeInfo)
	for _, service := range services {
		specs, err := scanAPISpecDir(filepath.Join(root, "api", service.Group, service.Version))
		if err != nil {
			return nil, fmt.Errorf("scan API specs for group %q: %w", service.Group, err)
		}
		if len(specs) == 0 {
			return nil, fmt.Errorf("configured API group %q has no spec types under api/%s/%s", service.Group, service.Group, service.Version)
		}
		out[service.Group] = specs
	}

	return out, nil
}

func scanAPISpecDir(dir string) ([]apiTypeInfo, error) {
	specs := make(map[string]apiTypeInfo)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, "_types.go") {
			return nil
		}
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		for _, declaration := range parsed.Decls {
			genDecl, ok := declaration.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				specType, statusType := resourceSurfaceTypes(structType)
				if specType == "" || !strings.HasSuffix(specType, "Spec") {
					continue
				}
				specName := strings.TrimSuffix(specType, "Spec")
				specs[specName] = apiTypeInfo{
					Spec:   specName,
					Status: statusType,
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(specs) == 0 {
		return nil, nil
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
	return out, nil
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
		specs := apiSpecs[service.Group]
		sdkDir := filepath.Join(root, "vendor", "github.com", "oracle", "oci-go-sdk", "v65", service.Service)
		stat, err := os.Stat(sdkDir)
		if err != nil || !stat.IsDir() {
			return nil, fmt.Errorf("configured service %q SDK package dir %q not found", service.Service, sdkDir)
		}

		sdkStructs := scanSDKStructNames(sdkDir)
		for _, specInfo := range specs {
			key := service.Group + "." + specInfo.Spec
			existingTarget, hasExisting := existing[key]
			targetName := makeTargetName(service.Group, specInfo.Spec)
			candidates := deriveSDKTypes(service.Service, specInfo.Spec, targetName, sdkStructs)
			if hasExisting {
				for _, mapping := range existingTarget.SDKMappings {
					parts := strings.Split(mapping.SDKStruct, ".")
					if len(parts) == 2 && parts[0] == service.Service {
						candidates = append(candidates, parts[1])
					}
				}
			}
			candidates = uniqueByOrder(candidates)
			sort.SliceStable(candidates, func(i, j int) bool {
				ai, aj := sdkTypeOrder(candidates[i]), sdkTypeOrder(candidates[j])
				if ai != aj {
					return ai < aj
				}
				return candidates[i] < candidates[j]
			})

			name := targetName
			if hasExisting && strings.TrimSpace(existingTarget.Name) != "" {
				name = existingTarget.Name
			}
			statusType := resolveStatusType(specInfo, hasExisting, existingTarget)
			mappings := buildSDKMappings(service.Service, specInfo.Spec, candidates, hasExisting, existingTarget)
			out = append(out, specTarget{
				Service:     service.Service,
				Group:       service.Group,
				Spec:        specInfo.Spec,
				Status:      statusType,
				Name:        name,
				SDKMappings: mappings,
			})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		gi, gj := groupOrder(out[i].Group), groupOrder(out[j].Group)
		if gi != gj {
			return gi < gj
		}
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].Name < out[j].Name
	})

	return out, nil
}

func buildSDKTargets(targets []specTarget, existing []sdkTarget, services []configuredService) []sdkTarget {
	set := make(map[string]sdkTarget)
	allowedServices := make(map[string]struct{}, len(services))
	for _, service := range services {
		allowedServices[service.Service] = struct{}{}
	}
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
	for _, e := range existing {
		if _, ok := allowedServices[e.Group]; !ok {
			continue
		}
		k := e.Group + "." + e.Type
		if _, ok := set[k]; !ok {
			set[k] = e
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

func buildSDKMappings(service, spec string, candidates []string, hasExisting bool, existing specTarget) []sdkMapping {
	override := explicitAPITargetOverrides[service+"."+spec]

	existingByType := make(map[string]sdkMapping, len(existing.SDKMappings))
	for _, mapping := range existing.SDKMappings {
		typeName, ok := unqualifiedSDKType(mapping.SDKStruct, service)
		if !ok {
			continue
		}
		existingByType[typeName] = mapping
	}

	overrideTypes := make([]string, 0, len(override.MappingOverrides))
	for typeName := range override.MappingOverrides {
		overrideTypes = append(overrideTypes, typeName)
	}
	sort.Strings(overrideTypes)

	order := make([]string, 0, len(override.SDKTypes)+len(candidates)+len(existingByType)+len(overrideTypes))
	order = append(order, override.SDKTypes...)
	order = append(order, overrideTypes...)
	order = append(order, candidates...)
	if hasExisting {
		for _, mapping := range existing.SDKMappings {
			typeName, ok := unqualifiedSDKType(mapping.SDKStruct, service)
			if !ok {
				continue
			}
			order = append(order, typeName)
		}
	}
	order = uniqueByOrder(order)
	sort.SliceStable(order, func(i, j int) bool {
		ai, aj := sdkTypeOrder(order[i]), sdkTypeOrder(order[j])
		if ai != aj {
			return ai < aj
		}
		return order[i] < order[j]
	})

	mappings := make([]sdkMapping, 0, len(order))
	for _, typeName := range order {
		mapping := sdkMapping{SDKStruct: service + "." + typeName}
		if existingMapping, ok := existingByType[typeName]; ok {
			mapping = existingMapping
		}
		if override.UseStatus && strings.TrimSpace(mapping.APISurface) == "" {
			mapping.APISurface = "status"
		}
		if overrideMapping, ok := override.MappingOverrides[typeName]; ok {
			if strings.TrimSpace(overrideMapping.APISurface) != "" {
				mapping.APISurface = overrideMapping.APISurface
			}
			mapping.Exclude = overrideMapping.Exclude
			if strings.TrimSpace(overrideMapping.Reason) != "" {
				mapping.Reason = overrideMapping.Reason
			} else if !mapping.Exclude {
				mapping.Reason = ""
			}
		}
		mappings = append(mappings, mapping)
	}
	return mappings
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

// Explicit overrides cover specs whose API surface or SDK names do not follow the common generator conventions.
var explicitAPITargetOverrides = map[string]apiTargetOverride{
	"artifacts.Repository":                                 {SDKTypes: []string{"GenericRepository", "ContainerRepository"}},
	"containerengine.ClusterOption":                        {SDKTypes: []string{"ClusterOptions"}},
	"containerengine.Kubeconfig":                           {SDKTypes: []string{"CreateClusterKubeconfigContentDetails"}},
	"containerengine.NodePoolOption":                       {SDKTypes: []string{"NodePoolOptions"}},
	"core.AllDrgAttachment":                                {SDKTypes: []string{"DrgAttachmentInfo"}, UseStatus: true},
	"core.AllowedPeerRegionsForRemotePeering":              {SDKTypes: []string{"PeerRegionForRemotePeering"}, UseStatus: true},
	"core.AppCatalogListingAgreement":                      {SDKTypes: []string{"AppCatalogListingResourceVersionAgreements"}},
	"core.ClusterNetworkInstance":                          {SDKTypes: []string{"InstanceSummary"}, UseStatus: true},
	"core.ComputeCapacityReservationInstance":              {SDKTypes: []string{"CapacityReservationInstanceSummary"}, UseStatus: true},
	"core.ComputeCapacityTopologyComputeBareMetalHost":     {SDKTypes: []string{"ComputeBareMetalHostCollection"}, UseStatus: true},
	"core.ComputeCapacityTopologyComputeHpcIsland":         {SDKTypes: []string{"ComputeHpcIslandCollection"}, UseStatus: true},
	"core.ComputeCapacityTopologyComputeNetworkBlock":      {SDKTypes: []string{"ComputeNetworkBlockCollection"}, UseStatus: true},
	"core.ConsoleHistoryContent":                           {UseStatus: true},
	"core.CpeDeviceConfigContent":                          {UseStatus: true},
	"core.CrossConnectLetterOfAuthority":                   {SDKTypes: []string{"LetterOfAuthority"}, UseStatus: true},
	"core.FastConnectProviderVirtualCircuitBandwidthShape": {SDKTypes: []string{"VirtualCircuitBandwidthShape"}, UseStatus: true},
	"core.IPSecConnectionTunnelRoute":                      {SDKTypes: []string{"TunnelRouteSummary"}, UseStatus: true},
	"core.IPSecConnectionTunnelSecurityAssociation":        {SDKTypes: []string{"TunnelSecurityAssociationSummary"}, UseStatus: true},
	"core.InstanceDevice":                                  {SDKTypes: []string{"Device"}, UseStatus: true},
	"core.Instance": {
		MappingOverrides: map[string]mappingOverride{
			"Instance":        {APISurface: "status"},
			"InstanceSummary": {APISurface: "status"},
		},
	},
	"core.IpsecCpeDeviceConfigContent":       {UseStatus: true},
	"core.NetworkSecurityGroupSecurityRule":  {SDKTypes: []string{"SecurityRule"}},
	"core.TunnelCpeDeviceConfigContent":      {UseStatus: true},
	"core.VolumeBackupPolicyAssetAssignment": {SDKTypes: []string{"VolumeBackupPolicyAssignment"}},
	"core.WindowsInstanceInitialCredential":  {SDKTypes: []string{"InstanceCredentials"}, UseStatus: true},
	"dns.DomainRecord":                       {SDKTypes: []string{"Record"}},
	"dns.ResolverEndpoint":                   {SDKTypes: []string{"ResolverVnicEndpoint", "ResolverVnicEndpointSummary"}},
	"dns.ZoneContent":                        {UseStatus: true},
	"dns.ZoneFromZoneFile":                   {SDKTypes: []string{"Zone"}, UseStatus: true},
	"dns.ZoneRecord":                         {SDKTypes: []string{"Record"}},
	"identity.CostTrackingTag":               {SDKTypes: []string{"Tag"}, UseStatus: true},
	"identity.IdentityProvider":              {SDKTypes: []string{"Saml2IdentityProvider"}},
	"identity.OAuthClientCredential": {
		MappingOverrides: map[string]mappingOverride{
			"OAuth2ClientCredential":        {APISurface: "status"},
			"OAuth2ClientCredentialSummary": {APISurface: "status"},
		},
	},
	"identity.OrResetUIPassword":         {SDKTypes: []string{"UiPassword"}, UseStatus: true},
	"identity.StandardTagNamespace":      {SDKTypes: []string{"StandardTagNamespaceTemplate", "StandardTagNamespaceTemplateSummary"}},
	"identity.StandardTagTemplate":       {SDKTypes: []string{"StandardTagDefinitionTemplate"}},
	"identity.UserState":                 {SDKTypes: []string{"User"}, UseStatus: true},
	"identity.UserUIPasswordInformation": {SDKTypes: []string{"UiPasswordInformation"}},
	"keymanagement.PreCoUserCredential":  {SDKTypes: []string{"PreCoUserCredentials"}},
	"loadbalancer.NetworkSecurityGroup":  {SDKTypes: []string{"UpdateNetworkSecurityGroupsDetails"}},
	"loadbalancer.Shape": {
		MappingOverrides: map[string]mappingOverride{
			"UpdateLoadBalancerShapeDetails": {
				Exclude: true,
				Reason:  "Intentionally untracked: duplicate desired-state payload is already tracked on LoadBalancerLoadBalancerShape.",
			},
		},
	},
	"networkloadbalancer.NetworkSecurityGroup": {SDKTypes: []string{"UpdateNetworkSecurityGroupsDetails"}},
	"objectstorage.Namespace":                  {SDKTypes: []string{"NamespaceMetadata"}},
	"ons.ConfirmSubscription":                  {SDKTypes: []string{"ConfirmationResult"}, UseStatus: true},
	"ons.Topic": {
		MappingOverrides: map[string]mappingOverride{
			"NotificationTopic":        {APISurface: "status"},
			"NotificationTopicSummary": {APISurface: "status"},
		},
	},
	"ons.Unsubscription": {UseStatus: true},
	"streaming.Stream": {
		MappingOverrides: map[string]mappingOverride{
			"Stream":        {APISurface: "status"},
			"StreamSummary": {APISurface: "status"},
		},
	},
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

func resolveStatusType(specInfo apiTypeInfo, hasExisting bool, existing specTarget) string {
	if strings.TrimSpace(specInfo.Status) != "" {
		return specInfo.Status
	}
	if hasExisting {
		return existing.Status
	}
	return ""
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

	var b strings.Builder
	b.WriteString("package apispec\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"reflect\"\n\n")
	for _, g := range apiGroups {
		fmt.Fprintf(&b, "\t%sv1beta1 \"github.com/oracle/oci-service-operator/api/%s/v1beta1\"\n", g, g)
	}
	b.WriteString(")\n\n")
	b.WriteString("type SDKMapping struct {\n\tSDKStruct  string\n\tAPISurface string\n\tExclude    bool\n\tReason     string\n}\n\n")
	b.WriteString("type Target struct {\n\tName        string\n\tSpecType    reflect.Type\n\tStatusType  reflect.Type\n\tSDKMappings []SDKMapping\n}\n\n")
	b.WriteString("var targets = []Target{\n")
	for _, t := range targets {
		b.WriteString("\t{\n")
		fmt.Fprintf(&b, "\t\tName:     %q,\n", t.Name)
		fmt.Fprintf(&b, "\t\tSpecType: reflect.TypeOf(%sv1beta1.%sSpec{}),\n", t.Group, t.Spec)
		if strings.TrimSpace(t.Status) != "" {
			fmt.Fprintf(&b, "\t\tStatusType: reflect.TypeOf(%sv1beta1.%s{}),\n", t.Group, t.Status)
		}
		b.WriteString("\t\tSDKMappings: []SDKMapping{\n")
		for _, mapping := range t.SDKMappings {
			b.WriteString("\t\t\t{\n")
			fmt.Fprintf(&b, "\t\t\t\tSDKStruct: %q,\n", mapping.SDKStruct)
			if strings.TrimSpace(mapping.APISurface) != "" {
				fmt.Fprintf(&b, "\t\t\t\tAPISurface: %q,\n", mapping.APISurface)
			}
			if mapping.Exclude {
				b.WriteString("\t\t\t\tExclude: true,\n")
			}
			if strings.TrimSpace(mapping.Reason) != "" {
				fmt.Fprintf(&b, "\t\t\t\tReason: %q,\n", mapping.Reason)
			}
			b.WriteString("\t\t\t},\n")
		}
		b.WriteString("\t\t},\n")
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n\n")
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

	return format.Source([]byte(b.String()))
}

func renderSDKRegistry(targets []sdkTarget) ([]byte, error) {
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

	byGroup := make(map[string][]string)
	for _, t := range targets {
		byGroup[t.Group] = append(byGroup[t.Group], t.Type)
	}
	for g := range byGroup {
		byGroup[g] = uniqueSorted(byGroup[g])
		sort.SliceStable(byGroup[g], func(i, j int) bool {
			ai, aj := sdkTypeOrder(byGroup[g][i]), sdkTypeOrder(byGroup[g][j])
			if ai != aj {
				return ai < aj
			}
			return byGroup[g][i] < byGroup[g][j]
		})
	}

	var b strings.Builder
	b.WriteString("package sdk\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"path\"\n")
	b.WriteString("\t\"reflect\"\n\n")
	for _, g := range sdkGroups {
		fmt.Fprintf(&b, "\t\"github.com/oracle/oci-go-sdk/v65/%s\"\n", g)
	}
	b.WriteString(")\n\n")
	b.WriteString("const (\n\tmodulePath    = \"github.com/oracle/oci-go-sdk/v65\"\n\tmoduleVersion = \"v65.61.1\"\n)\n\n")
	b.WriteString("var seedTargets = []Target{\n")
	for _, g := range sdkGroups {
		types := byGroup[g]
		if len(types) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\t// %s\n", serviceComment(g))
		for _, t := range types {
			fmt.Fprintf(&b, "\tnewTarget(%q, %q, reflect.TypeOf(%s.%s{})),\n", g, t, g, t)
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n\n")

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

	return format.Source([]byte(b.String()))
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

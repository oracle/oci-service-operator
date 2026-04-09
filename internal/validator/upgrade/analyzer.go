package upgrade

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/oracle/oci-service-operator/internal/validator/provider"
	"github.com/oracle/oci-service-operator/internal/validator/sdk"
)

const modulePath = "github.com/oracle/oci-go-sdk/v65"

type Analyzer struct {
	GOMODCACHE string
}

func NewAnalyzer() (*Analyzer, error) {
	gomodcache, err := resolveGoModCache()
	if err != nil {
		return nil, err
	}
	return &Analyzer{GOMODCACHE: gomodcache}, nil
}

func resolveGoModCache() (string, error) {
	if value := strings.TrimSpace(os.Getenv("GOMODCACHE")); value != "" {
		return value, nil
	}
	command := exec.Command("go", "env", "GOMODCACHE")
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("resolve GOMODCACHE: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (a *Analyzer) moduleDir(version string) (string, error) {
	dir := filepath.Join(a.GOMODCACHE, modulePath+"@"+version)
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}

	command := exec.Command("go", "mod", "download", modulePath+"@"+version)
	if output, err := command.CombinedOutput(); err != nil {
		return "", fmt.Errorf("download %s@%s: %w (%s)", modulePath, version, err, strings.TrimSpace(string(output)))
	}
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("resolve module dir for %s after download: %w", version, err)
	}
	return dir, nil
}

func (a *Analyzer) Analyze(fromVersion, toVersion, providerPath string) (Report, error) {
	fromDir, err := a.moduleDir(fromVersion)
	if err != nil {
		return Report{}, err
	}
	toDir, err := a.moduleDir(toVersion)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		FromVersion:          fromVersion,
		ToVersion:            toVersion,
		ComparedToOSOK:       strings.TrimSpace(providerPath) != "",
		Structs:              []StructDiff{},
		AllowlistSuggestions: []AllowlistSuggestion{},
	}

	fieldUsageIndex := map[string]map[string][]string{}
	if strings.TrimSpace(providerPath) != "" {
		providerAnalysis, err := provider.NewAnalyzer(providerPath).Analyze()
		if err != nil {
			return Report{}, err
		}
		fieldUsageIndex = buildFieldUsageIndex(providerAnalysis)
	}

	for _, target := range sdk.SeedTargets() {
		fromFields, err := parseStructFields(fromDir, target.PackageName, target.TypeName)
		if err != nil {
			return Report{}, err
		}
		toFields, err := parseStructFields(toDir, target.PackageName, target.TypeName)
		if err != nil {
			return Report{}, err
		}
		diff := diffStruct(target.QualifiedName, fromFields, toFields, fieldUsageIndex[target.QualifiedName])
		if len(diff.AddedFields) == 0 && len(diff.RemovedFields) == 0 && len(diff.ChangedFields) == 0 {
			continue
		}
		report.Structs = append(report.Structs, diff)
		report.AllowlistSuggestions = append(report.AllowlistSuggestions, fieldSuggestions(target.QualifiedName, diff.AddedFields, fromVersion, toVersion)...)
	}

	sort.Slice(report.Structs, func(i, j int) bool { return report.Structs[i].StructType < report.Structs[j].StructType })
	sort.Slice(report.AllowlistSuggestions, func(i, j int) bool { return report.AllowlistSuggestions[i].Path < report.AllowlistSuggestions[j].Path })
	return report, nil
}

func parseStructFields(moduleDir, packageName, typeName string) (map[string]FieldInfo, error) {
	packageDir := filepath.Join(moduleDir, packageName)
	parsed, err := parser.ParseDir(token.NewFileSet(), packageDir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	fields := map[string]FieldInfo{}
	for _, pkg := range parsed {
		for _, fileNode := range pkg.Files {
			for _, decl := range fileNode.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || typeSpec.Name.Name != typeName {
						continue
					}
					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range structType.Fields.List {
						if len(field.Names) == 0 {
							continue
						}
						for _, name := range field.Names {
							if !name.IsExported() {
								continue
							}
							doc := fieldDocumentation(field)
							fields[name.Name] = FieldInfo{
								Name:       name.Name,
								JSONName:   tagValue(field, "json"),
								Mandatory:  tagValue(field, "mandatory") == "true",
								Deprecated: strings.Contains(strings.ToLower(doc), "deprecated"),
								ReadOnly:   isReadOnlyDocumentation(doc),
							}
						}
					}
				}
			}
		}
	}
	return fields, nil
}

func diffStruct(structType string, fromFields, toFields map[string]FieldInfo, currentUsage map[string][]string) StructDiff {
	diff := StructDiff{StructType: structType}
	for name, toField := range toFields {
		fromField, found := fromFields[name]
		if !found {
			toField.UsedByOSOK = len(currentUsage[name]) > 0
			toField.References = append([]string(nil), currentUsage[name]...)
			diff.AddedFields = append(diff.AddedFields, toField)
			continue
		}
		if !sameFieldMetadata(fromField, toField) {
			diff.ChangedFields = append(diff.ChangedFields, FieldChange{
				FieldName:  name,
				From:       fromField,
				To:         toField,
				UsedByOSOK: len(currentUsage[name]) > 0,
				References: append([]string(nil), currentUsage[name]...),
			})
		}
	}
	for name, fromField := range fromFields {
		if _, found := toFields[name]; found {
			continue
		}
		fromField.UsedByOSOK = len(currentUsage[name]) > 0
		fromField.References = append([]string(nil), currentUsage[name]...)
		diff.RemovedFields = append(diff.RemovedFields, fromField)
	}
	sort.Slice(diff.AddedFields, func(i, j int) bool { return diff.AddedFields[i].Name < diff.AddedFields[j].Name })
	sort.Slice(diff.RemovedFields, func(i, j int) bool { return diff.RemovedFields[i].Name < diff.RemovedFields[j].Name })
	sort.Slice(diff.ChangedFields, func(i, j int) bool { return diff.ChangedFields[i].FieldName < diff.ChangedFields[j].FieldName })
	return diff
}

func sameFieldMetadata(left, right FieldInfo) bool {
	return left.Name == right.Name &&
		left.JSONName == right.JSONName &&
		left.Mandatory == right.Mandatory &&
		left.Deprecated == right.Deprecated &&
		left.ReadOnly == right.ReadOnly
}

func fieldSuggestions(structType string, addedFields []FieldInfo, fromVersion, toVersion string) []AllowlistSuggestion {
	suggestions := make([]AllowlistSuggestion, 0, len(addedFields))
	for _, field := range addedFields {
		status := "future_consideration"
		if field.Mandatory {
			status = "potential_gap"
		}
		suggestions = append(suggestions, AllowlistSuggestion{
			Path:   structType + ".fields." + field.Name,
			Status: status,
			Reason: fmt.Sprintf("Added in OCI Go SDK %s compared to %s; review OSOK coverage and update the allowlist as needed.", toVersion, fromVersion),
		})
	}
	return suggestions
}

func buildFieldUsageIndex(analysis provider.Analysis) map[string]map[string][]string {
	index := map[string]map[string][]string{}
	for _, usage := range analysis.Usages {
		structFields := index[usage.StructType]
		if structFields == nil {
			structFields = map[string][]string{}
			index[usage.StructType] = structFields
		}
		ref := usage.File + ":" + strconv.Itoa(usage.Line)
		if !containsString(structFields[usage.FieldName], ref) {
			structFields[usage.FieldName] = append(structFields[usage.FieldName], ref)
		}
	}
	for _, structFields := range index {
		for fieldName := range structFields {
			sort.Strings(structFields[fieldName])
		}
	}
	return index
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func tagValue(field *ast.Field, name string) string {
	if field.Tag == nil {
		return ""
	}
	tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
	value := tag.Get(name)
	if name == "json" && value != "" {
		return strings.Split(value, ",")[0]
	}
	return value
}

func fieldDocumentation(field *ast.Field) string {
	if field.Doc != nil {
		return strings.TrimSpace(field.Doc.Text())
	}
	if field.Comment != nil {
		return strings.TrimSpace(field.Comment.Text())
	}
	return ""
}

func isReadOnlyDocumentation(doc string) bool {
	lower := strings.ToLower(doc)
	return strings.Contains(lower, "read-only") ||
		strings.Contains(lower, "read only") ||
		strings.Contains(lower, "server-managed") ||
		strings.Contains(lower, "server managed") ||
		strings.Contains(lower, "output only")
}

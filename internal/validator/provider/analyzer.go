package provider

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/validator/sdk"
	"golang.org/x/tools/go/packages"
)

type Analyzer struct {
	SourcePath string
}

func NewAnalyzer(sourcePath string) *Analyzer {
	return &Analyzer{SourcePath: sourcePath}
}

func (analyzer *Analyzer) Analyze() (Analysis, error) {
	if err := analyzer.validate(); err != nil {
		return Analysis{}, err
	}

	loadedPackages, err := analyzer.loadPackages()
	if err != nil {
		return Analysis{}, err
	}
	usages := analyzer.collectUsages(loadedPackages, seedTargets())
	sortFieldUsages(usages)
	return Analysis{SourcePath: analyzer.SourcePath, Usages: usages}, nil
}

func (analyzer *Analyzer) validate() error {
	if strings.TrimSpace(analyzer.SourcePath) == "" {
		return errors.New("provider source path must not be empty")
	}
	return nil
}

func (analyzer *Analyzer) loadPackages() ([]*packages.Package, error) {
	config := &packages.Config{
		Dir: analyzer.SourcePath,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Tests: false,
	}
	loadedPackages, err := packages.Load(config,
		"./pkg/servicemanager/...",
		"./controllers/...",
	)
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(loadedPackages) > 0 {
		return nil, fmt.Errorf("failed to load OSOK packages from %s", analyzer.SourcePath)
	}
	return loadedPackages, nil
}

func seedTargets() map[string]struct{} {
	targets := map[string]struct{}{}
	for _, target := range sdk.SeedTargets() {
		targets[target.QualifiedName] = struct{}{}
	}
	return targets
}

func (analyzer *Analyzer) collectUsages(loadedPackages []*packages.Package, targets map[string]struct{}) []FieldUsage {
	var usages []FieldUsage
	for _, loadedPackage := range loadedPackages {
		usages = append(usages, analyzer.packageUsages(loadedPackage, targets)...)
	}
	return usages
}

func (analyzer *Analyzer) packageUsages(loadedPackage *packages.Package, targets map[string]struct{}) []FieldUsage {
	var usages []FieldUsage
	for fileIndex, fileNode := range loadedPackage.Syntax {
		usages = append(usages, analyzer.fileUsages(loadedPackage, fileIndex, fileNode, targets)...)
	}
	return usages
}

func (analyzer *Analyzer) fileUsages(
	loadedPackage *packages.Package,
	fileIndex int,
	fileNode *ast.File,
	targets map[string]struct{},
) []FieldUsage {
	filePath := normalizeFilePath(analyzer.SourcePath, filePathFor(loadedPackage, fileIndex))
	if shouldSkipFile(filePath) {
		return nil
	}

	var usages []FieldUsage
	ast.Inspect(fileNode, func(node ast.Node) bool {
		usages = appendNodeUsages(usages, loadedPackage, filePath, targets, node)
		return true
	})
	return usages
}

func appendNodeUsages(
	usages []FieldUsage,
	loadedPackage *packages.Package,
	filePath string,
	targets map[string]struct{},
	node ast.Node,
) []FieldUsage {
	switch typedNode := node.(type) {
	case *ast.CompositeLit:
		return append(usages, compositeLiteralUsages(loadedPackage, filePath, targets, typedNode)...)
	case *ast.AssignStmt:
		return append(usages, assignmentUsages(loadedPackage, filePath, targets, typedNode)...)
	default:
		return usages
	}
}

func compositeLiteralUsages(
	loadedPackage *packages.Package,
	filePath string,
	targets map[string]struct{},
	composite *ast.CompositeLit,
) []FieldUsage {
	var usages []FieldUsage
	if qualifiedTypeName(loadedPackage.TypesInfo.TypeOf(composite)) == "generatedruntime.Operation" {
		usages = append(usages, generatedOperationUsages(loadedPackage.TypesInfo, loadedPackage.Fset, filePath, targets, composite)...)
	}

	structType := seedStructName(loadedPackage.TypesInfo.TypeOf(composite), targets)
	if structType == "" {
		return usages
	}

	for _, element := range composite.Elts {
		keyValue, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		fieldIdentifier, ok := keyValue.Key.(*ast.Ident)
		if !ok {
			continue
		}
		position := loadedPackage.Fset.Position(fieldIdentifier.Pos())
		usages = append(usages, FieldUsage{
			StructType: structType,
			FieldName:  fieldIdentifier.Name,
			File:       filePath,
			Line:       position.Line,
			Kind:       UsageKindCompositeLiteral,
		})
	}
	return usages
}

func assignmentUsages(
	loadedPackage *packages.Package,
	filePath string,
	targets map[string]struct{},
	assign *ast.AssignStmt,
) []FieldUsage {
	var usages []FieldUsage
	for _, leftHandSide := range assign.Lhs {
		selector, ok := leftHandSide.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		structType := seedStructName(loadedPackage.TypesInfo.TypeOf(selector.X), targets)
		if structType == "" {
			continue
		}
		position := loadedPackage.Fset.Position(selector.Sel.Pos())
		usages = append(usages, FieldUsage{
			StructType: structType,
			FieldName:  selector.Sel.Name,
			File:       filePath,
			Line:       position.Line,
			Kind:       UsageKindPostLiteralAssignment,
		})
	}
	return usages
}

func sortFieldUsages(usages []FieldUsage) {
	sort.Slice(usages, func(i, j int) bool {
		left := usages[i]
		right := usages[j]
		if left.StructType != right.StructType {
			return left.StructType < right.StructType
		}
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.FieldName != right.FieldName {
			return left.FieldName < right.FieldName
		}
		return left.Kind < right.Kind
	})
}

func seedStructName(typeRef types.Type, targets map[string]struct{}) string {
	qualifiedName := qualifiedTypeName(typeRef)
	if qualifiedName == "" {
		return ""
	}
	if _, ok := targets[qualifiedName]; !ok {
		return ""
	}
	return qualifiedName
}

func qualifiedTypeName(typeRef types.Type) string {
	if typeRef == nil {
		return ""
	}
	for {
		pointerType, ok := typeRef.(*types.Pointer)
		if !ok {
			break
		}
		typeRef = pointerType.Elem()
	}
	namedType, ok := typeRef.(*types.Named)
	if !ok {
		return ""
	}
	object := namedType.Obj()
	if object == nil || object.Pkg() == nil {
		return ""
	}
	return path.Base(object.Pkg().Path()) + "." + object.Name()
}

func generatedOperationUsages(
	typesInfo *types.Info,
	fileSet *token.FileSet,
	filePath string,
	targets map[string]struct{},
	operation *ast.CompositeLit,
) []FieldUsage {
	requestType := generatedOperationRequestType(typesInfo, operation)
	if requestType == "" {
		return nil
	}
	fieldsLiteral := generatedOperationFieldsLiteral(operation)
	if fieldsLiteral == nil {
		return nil
	}
	return generatedRequestFieldUsages(typesInfo, fileSet, filePath, requestType, fieldsLiteral)
}

func generatedOperationRequestType(typesInfo *types.Info, operation *ast.CompositeLit) string {
	requestFactory := generatedOperationRequestFactory(operation)
	if requestFactory == nil {
		return ""
	}
	return returnedTypeName(typesInfo, requestFactory)
}

func generatedOperationFieldsLiteral(operation *ast.CompositeLit) *ast.CompositeLit {
	return compositeFieldValue(operation, "Fields")
}

func generatedOperationRequestFactory(operation *ast.CompositeLit) *ast.FuncLit {
	for _, element := range operation.Elts {
		keyValue, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		fieldIdentifier, ok := keyValue.Key.(*ast.Ident)
		if !ok || fieldIdentifier.Name != "NewRequest" {
			continue
		}
		requestFactory, ok := keyValue.Value.(*ast.FuncLit)
		if ok {
			return requestFactory
		}
	}
	return nil
}

func generatedRequestFieldUsages(
	typesInfo *types.Info,
	fileSet *token.FileSet,
	filePath string,
	requestType string,
	fieldsLiteral *ast.CompositeLit,
) []FieldUsage {
	var usages []FieldUsage
	for _, fieldElement := range fieldsLiteral.Elts {
		fieldLiteral, ok := fieldElement.(*ast.CompositeLit)
		if !ok {
			continue
		}
		if qualifiedTypeName(typesInfo.TypeOf(fieldLiteral)) != "generatedruntime.RequestField" {
			continue
		}
		fieldName, fieldPos := generatedRequestFieldName(fileSet, fieldLiteral)
		if fieldName == "" {
			continue
		}
		usages = append(usages, FieldUsage{
			StructType: requestType,
			FieldName:  fieldName,
			File:       filePath,
			Line:       fieldPos.Line,
			Kind:       UsageKindGeneratedRequestField,
		})
	}
	return usages
}

func returnedTypeName(typesInfo *types.Info, requestFactory *ast.FuncLit) string {
	for _, statement := range requestFactory.Body.List {
		returnStmt, ok := statement.(*ast.ReturnStmt)
		if !ok {
			continue
		}
		for _, result := range returnStmt.Results {
			if requestType := qualifiedTypeName(typesInfo.TypeOf(result)); requestType != "" {
				return requestType
			}
		}
	}
	return ""
}

func compositeFieldValue(literal *ast.CompositeLit, fieldName string) *ast.CompositeLit {
	for _, element := range literal.Elts {
		keyValue, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		fieldIdentifier, ok := keyValue.Key.(*ast.Ident)
		if !ok || fieldIdentifier.Name != fieldName {
			continue
		}
		value, ok := keyValue.Value.(*ast.CompositeLit)
		if ok {
			return value
		}
	}
	return nil
}

func generatedRequestFieldName(fileSet *token.FileSet, requestField *ast.CompositeLit) (string, token.Position) {
	for _, element := range requestField.Elts {
		keyValue, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		fieldIdentifier, ok := keyValue.Key.(*ast.Ident)
		if !ok || fieldIdentifier.Name != "FieldName" {
			continue
		}
		fieldValue, ok := keyValue.Value.(*ast.BasicLit)
		if !ok || fieldValue.Kind != token.STRING {
			return "", token.Position{}
		}
		fieldName := strings.Trim(fieldValue.Value, `"`)
		if fieldName == "" {
			return "", token.Position{}
		}
		return fieldName, fileSet.Position(fieldValue.Pos())
	}
	return "", token.Position{}
}

func filePathFor(loadedPackage *packages.Package, fileIndex int) string {
	if fileIndex < len(loadedPackage.CompiledGoFiles) {
		return loadedPackage.CompiledGoFiles[fileIndex]
	}
	if fileIndex < len(loadedPackage.GoFiles) {
		return loadedPackage.GoFiles[fileIndex]
	}
	return ""
}

func shouldSkipFile(filePath string) bool {
	if strings.HasSuffix(filePath, "_test.go") {
		return true
	}
	pathWithSlashes := strings.ReplaceAll(filePath, "\\", "/")
	return strings.Contains(pathWithSlashes, "/mock_") ||
		strings.Contains(pathWithSlashes, "/mocks/") ||
		strings.Contains(pathWithSlashes, "/test/") ||
		strings.HasSuffix(pathWithSlashes, "/clients_mock.go")
}

func normalizeFilePath(rootPath string, filePath string) string {
	relativePath, err := filepath.Rel(rootPath, filePath)
	if err != nil || strings.HasPrefix(relativePath, "..") {
		return filepath.ToSlash(filePath)
	}
	return filepath.ToSlash(relativePath)
}

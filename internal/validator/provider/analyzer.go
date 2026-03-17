package provider

import (
	"errors"
	"fmt"
	"go/ast"
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
	if strings.TrimSpace(analyzer.SourcePath) == "" {
		return Analysis{}, errors.New("provider source path must not be empty")
	}

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
		return Analysis{}, err
	}
	if packages.PrintErrors(loadedPackages) > 0 {
		return Analysis{}, fmt.Errorf("failed to load OSOK packages from %s", analyzer.SourcePath)
	}

	targets := map[string]struct{}{}
	for _, target := range sdk.SeedTargets() {
		targets[target.QualifiedName] = struct{}{}
	}

	var usages []FieldUsage
	for _, loadedPackage := range loadedPackages {
		for fileIndex, fileNode := range loadedPackage.Syntax {
			filePath := normalizeFilePath(analyzer.SourcePath, filePathFor(loadedPackage, fileIndex))
			if shouldSkipFile(filePath) {
				continue
			}
			ast.Inspect(fileNode, func(node ast.Node) bool {
				switch typedNode := node.(type) {
				case *ast.CompositeLit:
					structType := seedStructName(loadedPackage.TypesInfo.TypeOf(typedNode), targets)
					if structType == "" {
						return true
					}
					for _, element := range typedNode.Elts {
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
				case *ast.AssignStmt:
					for _, leftHandSide := range typedNode.Lhs {
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
				}
				return true
			})
		}
	}

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

	return Analysis{SourcePath: analyzer.SourcePath, Usages: usages}, nil
}

func seedStructName(typeRef types.Type, targets map[string]struct{}) string {
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
	qualifiedName := path.Base(object.Pkg().Path()) + "." + object.Name()
	if _, ok := targets[qualifiedName]; !ok {
		return ""
	}
	return qualifiedName
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

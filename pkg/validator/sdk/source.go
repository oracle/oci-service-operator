package sdk

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type sourceFieldMetadata struct {
	Documentation string
	Deprecated    bool
	ReadOnly      bool
}

type sourceIndex struct {
	moduleDir string
	packages  map[string]map[string]map[string]sourceFieldMetadata
}

func newSourceIndex(moduleDir string) *sourceIndex {
	return &sourceIndex{
		moduleDir: moduleDir,
		packages:  map[string]map[string]map[string]sourceFieldMetadata{},
	}
}

func (index *sourceIndex) fieldMetadata(importPath string, typeName string, fieldName string) (sourceFieldMetadata, error) {
	if err := index.loadPackage(importPath); err != nil {
		return sourceFieldMetadata{}, err
	}
	packageFields := index.packages[importPath]
	if packageFields == nil {
		return sourceFieldMetadata{}, nil
	}
	typeFields := packageFields[typeName]
	if typeFields == nil {
		return sourceFieldMetadata{}, nil
	}
	return typeFields[fieldName], nil
}

func (index *sourceIndex) loadPackage(importPath string) error {
	if _, ok := index.packages[importPath]; ok {
		return nil
	}

	packageDir := filepath.Join(index.moduleDir, strings.TrimPrefix(importPath, modulePath+"/"))
	parsedFiles, err := parser.ParseDir(token.NewFileSet(), packageDir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return err
	}

	packageFields := map[string]map[string]sourceFieldMetadata{}
	for _, parsedPackage := range parsedFiles {
		for _, fileNode := range parsedPackage.Files {
			for _, declaration := range fileNode.Decls {
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
					fieldMap := packageFields[typeSpec.Name.Name]
					if fieldMap == nil {
						fieldMap = map[string]sourceFieldMetadata{}
						packageFields[typeSpec.Name.Name] = fieldMap
					}
					for _, field := range structType.Fields.List {
						documentation := fieldDocumentation(field)
						metadata := sourceFieldMetadata{
							Documentation: documentation,
							Deprecated:    strings.Contains(strings.ToLower(documentation), "deprecated"),
							ReadOnly:      isReadOnlyDocumentation(documentation),
						}
						for _, name := range field.Names {
							fieldMap[name.Name] = metadata
						}
					}
				}
			}
		}
	}

	index.packages[importPath] = packageFields
	return nil
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

func isReadOnlyDocumentation(documentation string) bool {
	lower := strings.ToLower(documentation)
	return strings.Contains(lower, "read-only") ||
		strings.Contains(lower, "read only") ||
		strings.Contains(lower, "server-side") ||
		strings.Contains(lower, "server side") ||
		strings.Contains(lower, "output only")
}

package sdk

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/oracle/oci-service-operator/internal/ocisdk"
)

type Analyzer struct {
	metadataIndex *ocisdk.Index
}

func NewAnalyzer(moduleRoot string) (*Analyzer, error) {
	resolver, err := sdkPackageDirResolver(moduleRoot)
	if err != nil {
		return nil, err
	}
	return &Analyzer{metadataIndex: ocisdk.NewIndex(resolver)}, nil
}

func (analyzer *Analyzer) AnalyzeAll() ([]SDKStruct, error) {
	results := make([]SDKStruct, 0, len(seedTargets))
	for _, target := range SeedTargets() {
		result, err := analyzer.AnalyzeTarget(target.QualifiedName)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (analyzer *Analyzer) AnalyzeTarget(qualifiedName string) (SDKStruct, error) {
	target, ok := TargetByName(qualifiedName)
	if !ok {
		return SDKStruct{}, fmt.Errorf("unknown SDK target %q", qualifiedName)
	}
	fields, err := analyzer.fieldsForType(target.ReflectType, true)
	if err != nil {
		return SDKStruct{}, err
	}
	return SDKStruct{
		QualifiedName: target.QualifiedName,
		PackageName:   target.PackageName,
		TypeName:      target.TypeName,
		ImportPath:    target.ImportPath,
		Fields:        fields,
	}, nil
}

func (analyzer *Analyzer) fieldsForType(typeRef reflect.Type, allowNested bool) ([]SDKField, error) {
	metadataByName, err := analyzer.fieldMetadata(typeRef.PkgPath(), typeRef.Name())
	if err != nil {
		return nil, err
	}

	fields := make([]SDKField, 0, typeRef.NumField())
	for i := 0; i < typeRef.NumField(); i++ {
		structField := typeRef.Field(i)
		if !structField.IsExported() {
			continue
		}

		field := SDKField{
			Name:          structField.Name,
			Type:          structField.Type.String(),
			JSONName:      jsonFieldName(structField),
			Mandatory:     structField.Tag.Get("mandatory") == "true",
			Deprecated:    metadataByName[structField.Name].Deprecated,
			ReadOnly:      metadataByName[structField.Name].ReadOnly,
			Documentation: metadataByName[structField.Name].Documentation,
			Kind:          SDKFieldKindScalar,
		}
		if metadata, ok := metadataByName[structField.Name]; ok {
			if metadata.JSONName != "" {
				field.JSONName = metadata.JSONName
			}
			field.Mandatory = metadata.Mandatory
			field.Kind = sdkFieldKind(metadata.Kind)
		}

		if interfaceType := interfaceFieldType(structField.Type); interfaceType != nil {
			field.Kind = SDKFieldKindInterface
			for _, implementationType := range knownInterfaceImplementations(interfaceType) {
				nested, err := analyzer.fieldsForType(implementationType, false)
				if err != nil {
					return nil, err
				}
				field.InterfaceImplementations = append(field.InterfaceImplementations, SDKImplementation{
					QualifiedName: qualifiedTypeName(implementationType),
					PackageName:   path.Base(implementationType.PkgPath()),
					TypeName:      implementationType.Name(),
					ImportPath:    implementationType.PkgPath(),
					Fields:        nested,
				})
			}
		} else if allowNested {
			if metadata, ok := metadataByName[structField.Name]; ok && field.Kind == SDKFieldKindStruct {
				field.Kind = SDKFieldKindStruct
				field.NestedFields = convertMetadataFields(metadata.NestedFields, false)
			} else {
				nestedType := nestedStructType(structField.Type)
				if nestedType != nil {
					field.Kind = SDKFieldKindStruct
					nestedFields, err := analyzer.fieldsForType(nestedType, false)
					if err != nil {
						return nil, err
					}
					field.NestedFields = nestedFields
				}
			}
		}

		fields = append(fields, field)
	}
	return fields, nil
}

func (analyzer *Analyzer) fieldMetadata(importPath string, typeName string) (map[string]ocisdk.Field, error) {
	structMeta, ok, err := analyzer.metadataIndex.Struct(context.Background(), importPath, typeName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return map[string]ocisdk.Field{}, nil
	}

	metadataByName := make(map[string]ocisdk.Field, len(structMeta.Fields))
	for _, field := range structMeta.Fields {
		metadataByName[field.Name] = field
	}
	return metadataByName, nil
}

func sdkPackageDirResolver(moduleRoot string) (ocisdk.ResolveDirFunc, error) {
	moduleDir, err := resolveModuleDir(moduleRoot)
	if err != nil {
		return nil, err
	}

	return func(_ context.Context, importPath string) (string, error) {
		if importPath == modulePath {
			return moduleDir, nil
		}
		if !strings.HasPrefix(importPath, modulePath+"/") {
			return "", fmt.Errorf("import path %q is outside OCI SDK module %q", importPath, modulePath)
		}
		return filepath.Join(moduleDir, strings.TrimPrefix(importPath, modulePath+"/")), nil
	}, nil
}

func sdkFieldKind(kind ocisdk.FieldKind) SDKFieldKind {
	switch kind {
	case ocisdk.FieldKindStruct:
		return SDKFieldKindStruct
	case ocisdk.FieldKindInterface:
		return SDKFieldKindInterface
	default:
		return SDKFieldKindScalar
	}
}

func convertMetadataFields(fields []ocisdk.Field, allowNested bool) []SDKField {
	converted := make([]SDKField, 0, len(fields))
	for _, field := range fields {
		convertedField := SDKField{
			Name:          field.Name,
			Type:          field.Type,
			JSONName:      field.JSONName,
			Mandatory:     field.Mandatory,
			Deprecated:    field.Deprecated,
			ReadOnly:      field.ReadOnly,
			Documentation: field.Documentation,
			Kind:          sdkFieldKind(field.Kind),
		}
		if allowNested && field.Kind == ocisdk.FieldKindStruct {
			convertedField.NestedFields = convertMetadataFields(field.NestedFields, false)
		}
		converted = append(converted, convertedField)
	}
	return converted
}

func resolveModuleDir(moduleRoot string) (string, error) {
	if moduleRoot == "" {
		moduleRoot = autoModuleRoot()
	}
	if moduleRoot != "" {
		vendorPath := filepath.Join(moduleRoot, "vendor", modulePath)
		if info, err := os.Stat(vendorPath); err == nil && info.IsDir() {
			return vendorPath, nil
		}
	}

	gomodcache := strings.TrimSpace(os.Getenv("GOMODCACHE"))
	if gomodcache == "" {
		gopath := strings.TrimSpace(os.Getenv("GOPATH"))
		if gopath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			gopath = filepath.Join(home, "go")
		}
		gomodcache = filepath.Join(gopath, "pkg", "mod")
	}
	moduleDir := filepath.Join(gomodcache, modulePath+"@"+moduleVersion)
	if _, err := os.Stat(moduleDir); err != nil {
		return "", fmt.Errorf("resolve OCI SDK module dir: %w", err)
	}
	return moduleDir, nil
}

func autoModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func nestedStructType(typeRef reflect.Type) reflect.Type {
	candidate := dereferenceType(typeRef)
	if candidate.Kind() != reflect.Struct {
		return nil
	}
	if candidate.Name() == "" {
		return nil
	}
	return candidate
}

func interfaceFieldType(typeRef reflect.Type) reflect.Type {
	candidate := dereferenceType(typeRef)
	if candidate.Kind() == reflect.Interface {
		return candidate
	}
	if candidate.Kind() == reflect.Slice {
		element := dereferenceType(candidate.Elem())
		if element.Kind() == reflect.Interface {
			return element
		}
	}
	return nil
}

func dereferenceType(typeRef reflect.Type) reflect.Type {
	for typeRef.Kind() == reflect.Pointer {
		typeRef = typeRef.Elem()
	}
	return typeRef
}

func jsonFieldName(structField reflect.StructField) string {
	jsonTag := structField.Tag.Get("json")
	if jsonTag == "" {
		return ""
	}
	jsonName := strings.Split(jsonTag, ",")[0]
	if jsonName == "-" {
		return ""
	}
	return jsonName
}

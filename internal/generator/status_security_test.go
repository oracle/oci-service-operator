/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var sensitiveStatusFieldNames = map[string]struct{}{
	"BgpMd5AuthKey":           {},
	"ChapSecret":              {},
	"InstanceCredentials":     {},
	"Password":                {},
	"PrivateKeyPem":           {},
	"PrivateKeyPemPassphrase": {},
	"Secret":                  {},
	"SecretBundleContent":     {},
	"SharedSecret":            {},
	"SwiftPassword":           {},
	"Token":                   {},
}

var sensitiveStatusPropertyNames = map[string]struct{}{
	"bgpMd5AuthKey":           {},
	"chapSecret":              {},
	"instanceCredentials":     {},
	"password":                {},
	"privateKeyPem":           {},
	"privateKeyPemPassphrase": {},
	"secret":                  {},
	"secretBundleContent":     {},
	"sharedSecret":            {},
	"swiftPassword":           {},
	"token":                   {},
}

func TestCheckedInGeneratedAPIsDoNotExposeSensitiveStatusFields(t *testing.T) {
	root := repoRoot(t)
	paths, err := filepath.Glob(filepath.Join(root, "api", "*", "*", "*_types.go"))
	if err != nil {
		t.Fatalf("Glob(api/*/*/*_types.go) error = %v", err)
	}

	var offenders []string
	for _, path := range paths {
		offenders = append(offenders, collectSensitiveStatusFieldOffenders(t, root, path)...)
	}

	assertNoSensitiveStatusOffenders(t, offenders, "generated API status field")
}

func TestCheckedInPublishedCRDsDoNotExposeSensitiveStatusProperties(t *testing.T) {
	root := repoRoot(t)
	patterns := []string{
		filepath.Join(root, "config", "crd", "bases", "*.yaml"),
		filepath.Join(root, "packages", "*", "install", "generated", "crd", "bases", "*.yaml"),
	}

	var paths []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("Glob(%q) error = %v", pattern, err)
		}
		paths = append(paths, matches...)
	}

	var offenders []string
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}

		var document map[string]any
		if err := yaml.Unmarshal(content, &document); err != nil {
			t.Fatalf("yaml.Unmarshal(%q) error = %v", path, err)
		}

		for _, version := range crdVersions(t, document, path) {
			statusSchema, ok := nestedMap(version, "schema", "openAPIV3Schema", "properties", "status")
			if !ok {
				continue
			}
			collectSensitiveStatusProperties(path, []string{"status"}, statusSchema, &offenders)
		}
	}

	assertNoSensitiveStatusOffenders(t, offenders, "published CRD status property")
}

func assertNoSensitiveStatusOffenders(t *testing.T, offenders []string, target string) {
	t.Helper()

	if len(offenders) == 0 {
		return
	}

	sort.Strings(offenders)
	t.Fatalf("found %s offender(s):\n%s", target, strings.Join(offenders, "\n"))
}

func TestCollectSensitiveStatusPropertiesAllowsSecretNameReferenceWrappers(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{
			"adminPassword": map[string]any{
				"properties": map[string]any{
					"secret": map[string]any{
						"properties": map[string]any{
							"secretName": map[string]any{
								"type": "string",
							},
						},
						"type": "object",
					},
				},
				"type": "object",
			},
		},
	}

	var offenders []string
	collectSensitiveStatusProperties("test.yaml", []string{"status"}, schema, &offenders)
	if len(offenders) != 0 {
		t.Fatalf("collectSensitiveStatusProperties() offenders = %v, want none", offenders)
	}
}

func TestCollectSensitiveStatusPropertiesRejectsSensitiveLeafProperties(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{
			"source": map[string]any{
				"properties": map[string]any{
					"password": map[string]any{
						"type": "string",
					},
				},
				"type": "object",
			},
		},
	}

	var offenders []string
	collectSensitiveStatusProperties("test.yaml", []string{"status"}, schema, &offenders)
	if len(offenders) != 1 || offenders[0] != "test.yaml:status.source.password" {
		t.Fatalf("collectSensitiveStatusProperties() offenders = %v, want [test.yaml:status.source.password]", offenders)
	}
}

func isGeneratedStatusType(name string) bool {
	return strings.HasSuffix(name, "Status") || strings.HasSuffix(name, "ObservedState")
}

func collectSensitiveStatusFieldOffenders(t *testing.T, root string, path string) []string {
	t.Helper()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%q) error = %v", path, err)
	}

	var offenders []string
	for _, typeSpec := range generatedStatusTypeSpecs(file) {
		offenders = append(offenders, sensitiveStatusStructFieldOffenders(relativeToRepo(root, path), typeSpec)...)
	}

	return offenders
}

func generatedStatusTypeSpecs(file *ast.File) []*ast.TypeSpec {
	var specs []*ast.TypeSpec
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && isGeneratedStatusType(typeSpec.Name.Name) {
				specs = append(specs, typeSpec)
			}
		}
	}
	return specs
}

func sensitiveStatusStructFieldOffenders(path string, typeSpec *ast.TypeSpec) []string {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok || structType.Fields == nil {
		return nil
	}

	var offenders []string
	for _, field := range structType.Fields.List {
		offenders = append(offenders, sensitiveFieldNameOffenders(path, typeSpec.Name.Name, field.Names)...)
	}
	return offenders
}

func sensitiveFieldNameOffenders(path string, typeName string, names []*ast.Ident) []string {
	offenders := make([]string, 0, len(names))
	for _, name := range names {
		if _, sensitive := sensitiveStatusFieldNames[name.Name]; sensitive {
			offenders = append(offenders, fmt.Sprintf("%s:%s.%s", path, typeName, name.Name))
		}
	}
	return offenders
}

func relativeToRepo(root string, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return rel
	}
	return path
}

func crdVersions(t *testing.T, document map[string]any, path string) []map[string]any {
	t.Helper()

	spec, ok := nestedMap(document, "spec")
	if !ok {
		t.Fatalf("%s is missing spec", path)
	}

	rawVersions, ok := spec["versions"]
	if !ok {
		return nil
	}

	items, ok := rawVersions.([]any)
	if !ok {
		t.Fatalf("%s spec.versions = %T, want []any", path, rawVersions)
	}

	versions := make([]map[string]any, 0, len(items))
	for _, item := range items {
		version, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("%s spec.versions item = %T, want map[string]any", path, item)
		}
		versions = append(versions, version)
	}
	return versions
}

func collectSensitiveStatusProperties(path string, prefix []string, schema map[string]any, offenders *[]string) {
	properties, ok := nestedMap(schema, "properties")
	if ok {
		for name, rawChild := range properties {
			child, childOK := rawChild.(map[string]any)
			if _, sensitive := sensitiveStatusPropertyNames[name]; sensitive && !isSafeSecretReferenceProperty(name, child) {
				*offenders = append(*offenders, fmt.Sprintf("%s:%s", path, strings.Join(append(prefix, name), ".")))
			}

			if !childOK {
				continue
			}
			collectSensitiveStatusProperties(path, append(prefix, name), child, offenders)
		}
	}

	if items, ok := nestedMap(schema, "items"); ok {
		collectSensitiveStatusProperties(path, append(prefix, "[]"), items, offenders)
	}

	if additionalProperties, ok := nestedMap(schema, "additionalProperties"); ok {
		collectSensitiveStatusProperties(path, append(prefix, "{}"), additionalProperties, offenders)
	}
}

func isSafeSecretReferenceProperty(name string, schema map[string]any) bool {
	if name != "secret" || schema == nil {
		return false
	}

	properties, ok := nestedMap(schema, "properties")
	if !ok || len(properties) != 1 {
		return false
	}

	secretNameSchema, ok := properties["secretName"].(map[string]any)
	if !ok {
		return false
	}

	typeName, _ := secretNameSchema["type"].(string)
	return typeName == "" || typeName == "string"
}

func nestedMap(root map[string]any, path ...string) (map[string]any, bool) {
	current := root
	for _, segment := range path {
		value, ok := current[segment]
		if !ok {
			return nil, false
		}
		next, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

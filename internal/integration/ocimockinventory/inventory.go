/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Package ocimockinventory derives mock-integration groups from the checked-in
// generated runtime and formal catalog.
package ocimockinventory

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Group is a mutually exclusive generated-resource coverage group.
type Group string

const (
	GroupImmediate    Group = "s1-immediate-top-level"
	GroupUnclassified Group = "needs-contract-classification"
	GroupLifecycle    Group = "s2-lifecycle-polled"
	GroupComposite    Group = "s3-composite-path"
	GroupWorkRequest  Group = "async-work-request-or-special"
)

// Resource describes one generated CRUD resource and its evidence attributes.
type Resource struct {
	Service         string `json:"service"`
	Kind            string `json:"kind"`
	PackagePath     string `json:"packagePath"`
	Group           Group  `json:"group"`
	RuntimeOverride bool   `json:"runtimeOverride"`
	Formal          bool   `json:"formal"`
	MockIntegration bool   `json:"mockIntegration"`
}

// Report is the generated CRUD mock-integration inventory.
type Report struct {
	GeneratedCRUD               int           `json:"generatedCrud"`
	SynchronousCRUD             int           `json:"synchronousCrud"`
	WorkRequestCRUD             int           `json:"workRequestCrud"`
	RuntimeOverrides            int           `json:"runtimeOverrides"`
	FormalResources             int           `json:"formalResources"`
	MockIntegration             int           `json:"mockIntegration"`
	SynchronousRuntimeOverrides int           `json:"synchronousRuntimeOverrides"`
	SynchronousFormalResources  int           `json:"synchronousFormalResources"`
	SynchronousMockIntegration  int           `json:"synchronousMockIntegration"`
	Groups                      map[Group]int `json:"groups"`
	Resources                   []Resource    `json:"resources"`
}

var (
	operationPattern         = regexp.MustCompile(`(?m)^\s*(Create|Get|List|Update|Delete):\s+&generatedruntime\.Operation\{`)
	kindPattern              = regexp.MustCompile(`(?m)^type\s+([A-Za-z0-9]+)RuntimeHooks\s+struct\s*\{`)
	workRequestPattern       = regexp.MustCompile(`Strategy:\s+"workrequest"`)
	lifecyclePattern         = regexp.MustCompile(`(?s)(ProvisioningStates|UpdatingStates|PendingStates):\s*\[\]string\{[^}]`)
	lifecycleStrategyPattern = regexp.MustCompile(`Strategy:\s+"lifecycle"`)
	explicitSemanticsPattern = regexp.MustCompile(`(?m)(new[A-Za-z0-9]+RuntimeSemantics\(\)|hooks\.Semantics\s*=|return\s+&generatedruntime\.Semantics\{)`)
	compositePathPattern     = regexp.MustCompile(`Contribution:\s*"path",\s*PreferResourceID:\s*false`)
)

// Audit derives all generated CRUD resource groups from a repository root.
func Audit(root string) (Report, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("resolve repository root: %w", err)
	}
	formal, err := formalResources(root)
	if err != nil {
		return Report{}, err
	}

	report := Report{Groups: map[Group]int{}}
	hooksRoot := filepath.Join(root, "pkg", "servicemanager")
	err = filepath.WalkDir(hooksRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_runtimehooks_generated.go") {
			return nil
		}
		hooks, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !hasFullCRUD(hooks) {
			return nil
		}
		kindMatch := kindPattern.FindSubmatch(hooks)
		if len(kindMatch) != 2 {
			return fmt.Errorf("derive runtime kind from %s", filepath.ToSlash(path))
		}
		packageDir := filepath.Dir(path)
		production, runtimeOverride, err := productionSource(packageDir)
		if err != nil {
			return err
		}
		relativePackage, err := filepath.Rel(root, packageDir)
		if err != nil {
			return err
		}
		serviceRelative, err := filepath.Rel(hooksRoot, packageDir)
		if err != nil {
			return err
		}
		service := strings.Split(filepath.ToSlash(serviceRelative), "/")[0]
		kind := string(kindMatch[1])
		group := classify(hooks, production)
		key := inventoryKey(service, kind)
		item := Resource{
			Service:         service,
			Kind:            kind,
			PackagePath:     filepath.ToSlash(relativePackage),
			Group:           group,
			RuntimeOverride: runtimeOverride,
			Formal:          formal[key],
			MockIntegration: hasMockIntegrationTest(packageDir),
		}
		report.Resources = append(report.Resources, item)
		report.GeneratedCRUD++
		report.Groups[group]++
		if group == GroupWorkRequest {
			report.WorkRequestCRUD++
		} else {
			report.SynchronousCRUD++
			if item.MockIntegration {
				report.SynchronousMockIntegration++
			}
			if runtimeOverride {
				report.SynchronousRuntimeOverrides++
			}
			if item.Formal {
				report.SynchronousFormalResources++
			}
		}
		if runtimeOverride {
			report.RuntimeOverrides++
		}
		if item.Formal {
			report.FormalResources++
		}
		if item.MockIntegration {
			report.MockIntegration++
		}
		return nil
	})
	if err != nil {
		return Report{}, fmt.Errorf("walk generated runtime hooks: %w", err)
	}
	sort.Slice(report.Resources, func(i, j int) bool {
		return inventoryKey(report.Resources[i].Service, report.Resources[i].Kind) < inventoryKey(report.Resources[j].Service, report.Resources[j].Kind)
	})
	return report, nil
}

// MissingMockIntegration returns deterministically ordered resources in one
// group that do not yet have a package-local typed mock scenario.
func MissingMockIntegration(report Report, group Group) []Resource {
	var missing []Resource
	for _, resource := range report.Resources {
		if resource.Group == group && !resource.MockIntegration {
			missing = append(missing, resource)
		}
	}
	return missing
}

// MissingFormal returns deterministically ordered resources in one group that
// do not have a registered formal catalog row.
func MissingFormal(report Report, group Group) []Resource {
	var missing []Resource
	for _, resource := range report.Resources {
		if resource.Group == group && !resource.Formal {
			missing = append(missing, resource)
		}
	}
	return missing
}

func formalResources(root string) (map[string]bool, error) {
	path := filepath.Join(root, "formal", "controller_manifest.tsv")
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read formal controller manifest: %w", err)
	}
	reader := csv.NewReader(bytes.NewReader(content))
	reader.Comma = '\t'
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("decode formal controller manifest: %w", err)
	}
	result := map[string]bool{}
	for index, record := range records {
		if index == 0 || len(record) < 3 || record[0] == "template" {
			continue
		}
		result[inventoryKey(record[0], record[2])] = true
	}
	return result, nil
}

func productionSource(packageDir string) ([]byte, bool, error) {
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return nil, false, err
	}
	var combined []byte
	runtimeOverride := false
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(packageDir, entry.Name()))
		if err != nil {
			return nil, false, err
		}
		combined = append(combined, content...)
		combined = append(combined, '\n')
		if !bytes.Contains(content, []byte("Code generated by generator. DO NOT EDIT.")) {
			runtimeOverride = true
		}
	}
	return combined, runtimeOverride, nil
}

func hasMockIntegrationTest(packageDir string) bool {
	matches, err := filepath.Glob(filepath.Join(packageDir, "*_mock_integration_test.go"))
	return err == nil && len(matches) > 0
}

func hasFullCRUD(source []byte) bool {
	seen := map[string]bool{}
	for _, match := range operationPattern.FindAllSubmatch(source, -1) {
		name := string(match[1])
		if name == "Get" || name == "List" {
			name = "Read"
		}
		seen[name] = true
	}
	return seen["Create"] && seen["Read"] && seen["Update"] && seen["Delete"]
}

func classify(hooks []byte, production []byte) Group {
	if workRequestPattern.Match(production) {
		return GroupWorkRequest
	}
	if compositePathPattern.Match(hooks) {
		return GroupComposite
	}
	if lifecycleStrategyPattern.Match(production) || lifecyclePattern.Match(production) {
		return GroupLifecycle
	}
	if explicitSemanticsPattern.Match(production) {
		return GroupImmediate
	}
	return GroupUnclassified
}

func inventoryKey(service string, kind string) string {
	return strings.ToLower(strings.TrimSpace(service)) + "/" + strings.ToLower(strings.TrimSpace(kind))
}

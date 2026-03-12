/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// Renderer writes the generator's intermediate model into Go source files.
type Renderer struct{}

// NewRenderer returns the default package renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// ErrTargetExists is returned when a service output directory already exists.
type ErrTargetExists struct {
	Path string
}

func (e ErrTargetExists) Error() string {
	return fmt.Sprintf("target output %q already exists", e.Path)
}

func (r *Renderer) RenderPackage(root string, pkg *PackageModel, overwrite bool) (string, error) {
	outputDir := targetOutputDir(root, pkg)
	if _, err := os.Stat(outputDir); err == nil && !overwrite {
		return "", ErrTargetExists{Path: outputDir}
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat output dir %q: %w", outputDir, err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir %q: %w", outputDir, err)
	}

	groupVersionContent, err := renderGroupVersionFile(pkg)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "groupversion_info.go"), []byte(groupVersionContent), 0o644); err != nil {
		return "", fmt.Errorf("write groupversion_info.go for %s: %w", pkg.Service.Service, err)
	}

	for _, resource := range pkg.Resources {
		resourceContent, err := renderResourceFile(pkg, resource)
		if err != nil {
			return "", err
		}
		filePath := filepath.Join(outputDir, resource.FileStem+"_types.go")
		if err := os.WriteFile(filePath, []byte(resourceContent), 0o644); err != nil {
			return "", fmt.Errorf("write %s for %s: %w", filepath.Base(filePath), pkg.Service.Service, err)
		}
	}

	return outputDir, nil
}

func (r *Renderer) RenderPackageOutputs(root string, pkg *PackageModel) error {
	if !pkg.PackageOutput.Generate {
		return nil
	}

	packageDir := filepath.Join(root, "packages", pkg.Service.Group)
	installDir := filepath.Join(packageDir, "install")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("create package install dir %q: %w", installDir, err)
	}

	metadataContent, err := renderPackageMetadata(pkg.PackageOutput.Metadata)
	if err != nil {
		return fmt.Errorf("render package metadata for %s: %w", pkg.Service.Service, err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, "metadata.env"), []byte(metadataContent), 0o644); err != nil {
		return fmt.Errorf("write metadata.env for %s: %w", pkg.Service.Service, err)
	}

	installContent, err := renderInstallKustomization(pkg.PackageOutput.Install)
	if err != nil {
		return fmt.Errorf("render install kustomization for %s: %w", pkg.Service.Service, err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "kustomization.yaml"), []byte(installContent), 0o644); err != nil {
		return fmt.Errorf("write install/kustomization.yaml for %s: %w", pkg.Service.Service, err)
	}

	return nil
}

func (r *Renderer) RenderSamples(root string, packages []*PackageModel) error {
	type sampleEntry struct {
		order    int
		fileName string
		body     string
		groupDNS string
		version  string
		kind     string
		metadata string
		spec     string
	}

	var samples []sampleEntry
	for _, pkg := range packages {
		for _, resource := range pkg.Resources {
			if strings.TrimSpace(resource.Sample.FileName) == "" {
				continue
			}
			samples = append(samples, sampleEntry{
				order:    pkg.SampleOrder,
				fileName: resource.Sample.FileName,
				body:     resource.Sample.Body,
				groupDNS: pkg.GroupDNSName,
				version:  pkg.Version,
				kind:     resource.Kind,
				metadata: resource.Sample.MetadataName,
				spec:     resource.Sample.Spec,
			})
		}
	}

	if len(samples) == 0 {
		return nil
	}

	sort.Slice(samples, func(i, j int) bool {
		if samples[i].order == samples[j].order {
			return samples[i].fileName < samples[j].fileName
		}
		return samples[i].order < samples[j].order
	})

	samplesDir := filepath.Join(root, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		return fmt.Errorf("create samples dir %q: %w", samplesDir, err)
	}

	if err := cleanupGeneratedSampleFiles(samplesDir, packages); err != nil {
		return err
	}

	resourceNames := make([]string, 0, len(samples))
	for _, sample := range samples {
		content, err := renderSampleFile(sample.body, sample.groupDNS, sample.version, sample.kind, sample.metadata, sample.spec)
		if err != nil {
			return fmt.Errorf("render sample %s: %w", sample.fileName, err)
		}
		if err := os.WriteFile(filepath.Join(samplesDir, sample.fileName), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write sample %s: %w", sample.fileName, err)
		}
		resourceNames = append(resourceNames, sample.fileName)
	}

	orderedResources, err := orderedSampleResources(samplesDir, resourceNames)
	if err != nil {
		return err
	}

	kustomizationContent, err := renderSamplesKustomization(orderedResources)
	if err != nil {
		return fmt.Errorf("render samples kustomization: %w", err)
	}
	if err := os.WriteFile(filepath.Join(samplesDir, "kustomization.yaml"), []byte(kustomizationContent), 0o644); err != nil {
		return fmt.Errorf("write samples kustomization: %w", err)
	}

	return nil
}

func renderGroupVersionFile(pkg *PackageModel) (string, error) {
	data := struct {
		Group   string
		Version string
		Domain  string
	}{
		Group:   pkg.Service.Group,
		Version: pkg.Version,
		Domain:  pkg.Domain,
	}

	content, err := executeTemplate(groupVersionTemplate, data)
	if err != nil {
		return "", fmt.Errorf("render groupversion_info.go for %s: %w", pkg.Service.Service, err)
	}
	return formatGoSource(content)
}

func renderResourceFile(pkg *PackageModel, resource ResourceModel) (string, error) {
	data := struct {
		Version string
		ResourceModel
	}{
		Version:       pkg.Version,
		ResourceModel: resource,
	}

	content, err := executeTemplate(resourceTemplate, data)
	if err != nil {
		return "", fmt.Errorf("render %s_types.go for %s: %w", resource.FileStem, pkg.Service.Service, err)
	}
	return formatGoSource(content)
}

func renderPackageMetadata(metadata PackageMetadataModel) (string, error) {
	return executeTemplate(packageMetadataTemplate, metadata)
}

func renderInstallKustomization(install InstallKustomizationModel) (string, error) {
	return executeTemplate(installKustomizationTemplate, install)
}

func renderSampleFile(body string, groupDNS string, version string, kind string, metadataName string, spec string) (string, error) {
	data := struct {
		Body         string
		GroupDNSName string
		Version      string
		Kind         string
		MetadataName string
		Spec         string
	}{
		Body:         body,
		GroupDNSName: groupDNS,
		Version:      version,
		Kind:         kind,
		MetadataName: metadataName,
		Spec:         spec,
	}

	return executeTemplate(sampleTemplate, data)
}

func renderSamplesKustomization(resources []string) (string, error) {
	data := struct {
		Resources []string
	}{
		Resources: resources,
	}

	return executeTemplate(samplesKustomizationTemplate, data)
}

func cleanupGeneratedSampleFiles(samplesDir string, packages []*PackageModel) error {
	prefixes := make([]string, 0, len(packages))
	for _, pkg := range packages {
		prefixes = append(prefixes, fmt.Sprintf("%s_%s_", pkg.Service.Group, pkg.Version))
	}

	entries, err := os.ReadDir(samplesDir)
	if err != nil {
		return fmt.Errorf("read samples dir %q: %w", samplesDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "kustomization.yaml" || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		if !matchesSamplePrefix(entry.Name(), prefixes) {
			continue
		}

		path := filepath.Join(samplesDir, entry.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove sample %q: %w", path, err)
		}
	}

	return nil
}

func orderedSampleResources(samplesDir string, generatedOrder []string) ([]string, error) {
	existingOrder, err := readSampleKustomizationOrder(filepath.Join(samplesDir, "kustomization.yaml"))
	if err != nil {
		return nil, err
	}

	currentFiles, err := listSampleFiles(samplesDir)
	if err != nil {
		return nil, err
	}

	remaining := make(map[string]struct{}, len(currentFiles))
	for _, name := range currentFiles {
		remaining[name] = struct{}{}
	}

	ordered := make([]string, 0, len(currentFiles))
	for _, name := range existingOrder {
		if _, ok := remaining[name]; !ok {
			continue
		}
		ordered = append(ordered, name)
		delete(remaining, name)
	}

	for _, name := range generatedOrder {
		if _, ok := remaining[name]; !ok {
			continue
		}
		ordered = append(ordered, name)
		delete(remaining, name)
	}

	var leftovers []string
	for name := range remaining {
		leftovers = append(leftovers, name)
	}
	sort.Strings(leftovers)
	ordered = append(ordered, leftovers...)

	return ordered, nil
}

func readSampleKustomizationOrder(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read sample kustomization %q: %w", path, err)
	}

	lines := strings.Split(string(content), "\n")
	resources := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		resources = append(resources, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
	}
	return resources, nil
}

func listSampleFiles(samplesDir string) ([]string, error) {
	entries, err := os.ReadDir(samplesDir)
	if err != nil {
		return nil, fmt.Errorf("read samples dir %q: %w", samplesDir, err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "kustomization.yaml" || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	return files, nil
}

func matchesSamplePrefix(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func executeTemplate(content string, data any) (string, error) {
	tmpl, err := template.New("generator").Funcs(template.FuncMap{
		"comment":      commentLine,
		"marker":       markerLine,
		"fieldDecl":    fieldDecl,
		"printColumn":  printColumnMarker,
		"hasComments":  hasComments,
		"hasSpecValue": hasSpecValue,
	}).Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buffer.String(), nil
}

func formatGoSource(content string) (string, error) {
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return "", fmt.Errorf("format generated Go source: %w", err)
	}
	return string(formatted), nil
}

func commentLine(text string) string {
	if strings.TrimSpace(text) == "" {
		return "//"
	}
	return "// " + text
}

func markerLine(text string) string {
	if strings.TrimSpace(text) == "" {
		return "//"
	}
	return "// " + text
}

func fieldDecl(field FieldModel) string {
	if field.Embedded {
		return fmt.Sprintf("%s `%s`", field.Type, field.Tag)
	}
	return fmt.Sprintf("%s %s `%s`", field.Name, field.Type, field.Tag)
}

func printColumnMarker(column PrintColumnModel) string {
	parts := []string{
		fmt.Sprintf(`+kubebuilder:printcolumn:name="%s"`, column.Name),
		fmt.Sprintf(`type="%s"`, column.Type),
		fmt.Sprintf(`JSONPath="%s"`, column.JSONPath),
	}
	if column.Description != "" {
		parts = append(parts, fmt.Sprintf(`description="%s"`, column.Description))
	}
	if column.Priority != nil {
		parts = append(parts, fmt.Sprintf("priority=%d", *column.Priority))
	}
	return strings.Join(parts, ",")
}

func hasSpecValue(spec string) bool {
	return strings.TrimSpace(spec) != ""
}

func hasComments(comments []string) bool {
	return len(comments) > 0
}

const groupVersionTemplate = `/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Code generated by osok-api-generator. DO NOT EDIT.

// Package {{ .Version }} contains API Schema definitions for the {{ .Group }} {{ .Version }} API group.
// +kubebuilder:object:generate=true
// +groupName={{ .Group }}.{{ .Domain }}
package {{ .Version }}

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "{{ .Group }}.{{ .Domain }}", Version: "{{ .Version }}"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
`

const resourceTemplate = `/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Code generated by osok-api-generator. DO NOT EDIT.

package {{ .Version }}

import (
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

{{ range .LeadingComments }}
{{ comment . }}
{{ end }}
{{ if hasComments .LeadingComments }}

{{ end }}
{{- range .SpecComments }}
{{ comment . }}
{{- end }}
type {{ .Kind }}Spec struct {
{{- range .SpecFields }}
{{- range .Comments }}
	{{ comment . }}
{{- end }}
{{- range .Markers }}
	{{ marker . }}
{{- end }}
	{{ fieldDecl . }}
{{- end }}
}

{{- range .HelperTypes }}

{{- range .Comments }}
{{ comment . }}
{{- end }}
type {{ .Name }} struct {
{{- range .Fields }}
{{- range .Comments }}
	{{ comment . }}
{{- end }}
{{- range .Markers }}
	{{ marker . }}
{{- end }}
	{{ fieldDecl . }}
{{- end }}
}
{{- end }}

{{- if .StatusComments }}

{{- range .StatusComments }}
{{ comment . }}
{{- end }}
{{- end }}
type {{ .StatusTypeName }} struct {
{{- range .StatusFields }}
{{- range .Comments }}
	{{ comment . }}
{{- end }}
{{- range .Markers }}
	{{ marker . }}
{{- end }}
	{{ fieldDecl . }}
{{- end }}
}

{{ marker "+kubebuilder:object:root=true" }}
{{ marker "+kubebuilder:subresource:status" }}
{{- range .PrintColumns }}
{{ marker (printColumn .) }}
{{- end }}

{{- range .ObjectComments }}
{{ comment . }}
{{- end }}
type {{ .Kind }} struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Spec   {{ .Kind }}Spec   ` + "`json:\"spec,omitempty\"`" + `
	Status {{ .StatusTypeName }} ` + "`json:\"status,omitempty\"`" + `
}

{{ marker "+kubebuilder:object:root=true" }}

{{- range .ListComments }}
{{ comment . }}
{{- end }}
type {{ .Kind }}List struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items           []{{ .Kind }} ` + "`json:\"items\"`" + `
}

func init() {
	SchemeBuilder.Register(&{{ .Kind }}{}, &{{ .Kind }}List{})
}
`

const sampleTemplate = `#
# Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
# Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
#
{{- if hasSpecValue .Body }}

{{ .Body }}
{{- else }}

apiVersion: {{ .GroupDNSName }}/{{ .Version }}
kind: {{ .Kind }}
metadata:
  name: {{ .MetadataName }}
{{- if hasSpecValue .Spec }}
spec:
{{ .Spec }}
{{- else }}
spec: {}
{{- end }}
{{- end }}
`

const samplesKustomizationTemplate = `#
# Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
# Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
#

## Append samples you want in your CSV to this file as resources ##
resources:
{{- range .Resources }}
- {{ . }}
{{- end }}
# +kubebuilder:scaffold:manifestskustomizesamples
`

const packageMetadataTemplate = `PACKAGE_NAME={{ .PackageName }}
PACKAGE_NAMESPACE={{ .PackageNamespace }}
PACKAGE_NAME_PREFIX={{ .PackageNamePrefix }}
CRD_PATHS={{ .CRDPaths }}
{{- if .RBACPaths }}
RBAC_PATHS={{ .RBACPaths }}
{{- end }}
DEFAULT_CONTROLLER_IMAGE={{ .DefaultControllerImage }}
`

const installKustomizationTemplate = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
{{- if .Namespace }}

namespace: {{ .Namespace }}
{{- end }}
{{- if .NamePrefix }}
namePrefix: {{ .NamePrefix }}
{{- end }}

resources:
{{- range .Resources }}
- {{ . }}
{{- end }}
{{- if .PatchPath }}

patches:
- path: {{ .PatchPath }}
  target:
    kind: {{ .PatchTarget }}
{{- end }}
`

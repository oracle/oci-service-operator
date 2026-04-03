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

//nolint:gocyclo // Package rendering fans out across multiple optional generated surfaces.
func (r *Renderer) RenderPackage(root string, pkg *PackageModel, overwrite bool) (string, error) {
	outputDir, err := preparePackageOutputDir(root, pkg, overwrite)
	if err != nil {
		return "", err
	}
	if err := writeGroupVersionFile(outputDir, pkg); err != nil {
		return "", err
	}
	if err := writePackageResourceFiles(outputDir, pkg); err != nil {
		return "", err
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

func (r *Renderer) RenderControllers(root string, pkg *PackageModel, overwrite bool) error {
	if len(pkg.Controller.Resources) == 0 {
		return nil
	}

	outputDir := controllerOutputDir(root, pkg)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create controller dir %q: %w", outputDir, err)
	}

	for _, controller := range pkg.Controller.Resources {
		content, err := renderControllerFile(pkg, controller)
		if err != nil {
			return err
		}

		path := filepath.Join(outputDir, controller.FileStem+"_controller.go")
		if err := writeGeneratedFile(path, content, overwrite); err != nil {
			return err
		}
	}

	return nil
}

func (r *Renderer) RenderRegistrations(root string, pkg *PackageModel, overwrite bool) error {
	if len(pkg.Registration.Resources) == 0 {
		return nil
	}

	outputDir := registrationOutputDir(root)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create registration dir %q: %w", outputDir, err)
	}

	content, err := renderRegistrationFile(pkg.Registration)
	if err != nil {
		return err
	}

	path := filepath.Join(outputDir, pkg.Registration.Group+"_generated.go")
	if err := writeGeneratedFile(path, content, overwrite); err != nil {
		return err
	}

	return nil
}

func (r *Renderer) RenderServiceManagers(root string, pkg *PackageModel, overwrite bool) error {
	if len(pkg.ServiceManagers) == 0 {
		return nil
	}

	if err := preflightServiceManagerDirs(root, pkg.ServiceManagers, overwrite); err != nil {
		return err
	}

	for _, serviceManager := range pkg.ServiceManagers {
		outputDir := serviceManagerOutputDir(root, serviceManager)
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("create service-manager dir %q: %w", outputDir, err)
		}

		serviceClientContent, err := renderServiceClientFile(serviceManager)
		if err != nil {
			return fmt.Errorf("render %s for %s: %w", serviceManager.ServiceClientFileName, pkg.Service.Service, err)
		}
		serviceClientPath := filepath.Join(outputDir, serviceManager.ServiceClientFileName)
		if err := writeGeneratedFile(serviceClientPath, serviceClientContent, overwrite); err != nil {
			return err
		}

		serviceManagerContent, err := renderServiceManagerFile(serviceManager)
		if err != nil {
			return fmt.Errorf("render %s for %s: %w", serviceManager.ServiceManagerFileName, pkg.Service.Service, err)
		}
		serviceManagerPath := filepath.Join(outputDir, serviceManager.ServiceManagerFileName)
		if err := writeGeneratedFile(serviceManagerPath, serviceManagerContent, overwrite); err != nil {
			return err
		}
	}

	return nil
}

//nolint:gocognit,gocyclo // Sample rendering preserves ordering and kustomization updates across package groups.
func (r *Renderer) RenderSamples(root string, packages []*PackageModel) error {
	samples := collectSamples(packages)
	if len(samples) == 0 {
		samplesDir := filepath.Join(root, "config", "samples")
		if _, err := os.Stat(samplesDir); os.IsNotExist(err) {
			return nil
		} else if err != nil {
			return fmt.Errorf("stat samples dir %q: %w", samplesDir, err)
		}
		return writeSamplesKustomizationFile(samplesDir, nil)
	}
	sortSamples(samples)
	samplesDir, err := ensureSamplesDir(root)
	if err != nil {
		return err
	}
	if err := cleanupGeneratedSampleFiles(samplesDir, packages); err != nil {
		return err
	}
	resourceNames, err := writeSampleFiles(samplesDir, samples)
	if err != nil {
		return err
	}
	return writeSamplesKustomizationFile(samplesDir, resourceNames)
}

func preparePackageOutputDir(root string, pkg *PackageModel, overwrite bool) (string, error) {
	outputDir := targetOutputDir(root, pkg)
	if _, err := os.Stat(outputDir); err == nil && !overwrite {
		return "", ErrTargetExists{Path: outputDir}
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat output dir %q: %w", outputDir, err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir %q: %w", outputDir, err)
	}
	return outputDir, nil
}

func writeGroupVersionFile(outputDir string, pkg *PackageModel) error {
	groupVersionContent, err := renderGroupVersionFile(pkg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "groupversion_info.go"), []byte(groupVersionContent), 0o644); err != nil {
		return fmt.Errorf("write groupversion_info.go for %s: %w", pkg.Service.Service, err)
	}
	return nil
}

func writePackageResourceFiles(outputDir string, pkg *PackageModel) error {
	for _, resource := range pkg.Resources {
		resourceContent, err := renderResourceFile(pkg, resource)
		if err != nil {
			return err
		}
		filePath := filepath.Join(outputDir, resource.FileStem+"_types.go")
		if err := os.WriteFile(filePath, []byte(resourceContent), 0o644); err != nil {
			return fmt.Errorf("write %s for %s: %w", filepath.Base(filePath), pkg.Service.Service, err)
		}
	}
	return nil
}

func collectSamples(packages []*PackageModel) []sampleEntry {
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
	return samples
}

func sortSamples(samples []sampleEntry) {
	sort.Slice(samples, func(i, j int) bool {
		if samples[i].order == samples[j].order {
			return samples[i].fileName < samples[j].fileName
		}
		return samples[i].order < samples[j].order
	})
}

func ensureSamplesDir(root string) (string, error) {
	samplesDir := filepath.Join(root, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		return "", fmt.Errorf("create samples dir %q: %w", samplesDir, err)
	}
	return samplesDir, nil
}

func writeSampleFiles(samplesDir string, samples []sampleEntry) ([]string, error) {
	resourceNames := make([]string, 0, len(samples))
	for _, sample := range samples {
		content, err := renderSampleFile(sample.body, sample.groupDNS, sample.version, sample.kind, sample.metadata, sample.spec)
		if err != nil {
			return nil, fmt.Errorf("render sample %s: %w", sample.fileName, err)
		}
		if err := os.WriteFile(filepath.Join(samplesDir, sample.fileName), []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("write sample %s: %w", sample.fileName, err)
		}
		resourceNames = append(resourceNames, sample.fileName)
	}
	return resourceNames, nil
}

func writeSamplesKustomizationFile(samplesDir string, resourceNames []string) error {
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

func renderControllerFile(pkg *PackageModel, controller ControllerModel) (string, error) {
	data := struct {
		PackageName    string
		APIImportAlias string
		Group          string
		Version        string
		ControllerModel
	}{
		PackageName:     pkg.Service.Group,
		APIImportAlias:  fmt.Sprintf("%s%s", pkg.Service.Group, pkg.Version),
		Group:           pkg.Service.Group,
		Version:         pkg.Version,
		ControllerModel: controller,
	}

	content, err := executeTemplate(controllerTemplate, data)
	if err != nil {
		return "", fmt.Errorf("render %s_controller.go for %s: %w", controller.FileStem, pkg.Service.Service, err)
	}
	return formatGoSource(content)
}

func renderRegistrationFile(registration RegistrationOutputModel) (string, error) {
	content, err := executeTemplate(registrationTemplate, registration)
	if err != nil {
		return "", fmt.Errorf("render %s_generated.go: %w", registration.Group, err)
	}
	return formatGoSource(content)
}

func renderPackageMetadata(metadata PackageMetadataModel) (string, error) {
	return executeTemplate(packageMetadataTemplate, metadata)
}

func renderInstallKustomization(install InstallKustomizationModel) (string, error) {
	return executeTemplate(installKustomizationTemplate, install)
}

func renderServiceClientFile(serviceManager ServiceManagerModel) (string, error) {
	content, err := executeTemplate(serviceClientTemplate, serviceManager)
	if err != nil {
		return "", err
	}
	return formatGoSource(content)
}

func renderServiceManagerFile(serviceManager ServiceManagerModel) (string, error) {
	content, err := executeTemplate(serviceManagerTemplate, serviceManager)
	if err != nil {
		return "", err
	}
	return formatGoSource(content)
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
		prefixes = append(prefixes, sampleGroupPrefix(pkg.Service.Group))
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
		"comment":               commentLine,
		"marker":                markerLine,
		"fieldDecl":             fieldDecl,
		"printColumn":           printColumnMarker,
		"hasComments":           hasComments,
		"hasSpecValue":          hasSpecValue,
		"stringSliceLiteral":    goStringSliceLiteral,
		"stringSliceMapLiteral": goStringSliceMapLiteral,
		"requestFieldsLiteral":  requestFieldsLiteral,
		"runtimeHooksLiteral":   runtimeHooksLiteral,
		"runtimeAuxOpsLiteral":  runtimeAuxiliaryOperationsLiteral,
		"runtimeGapsLiteral":    runtimeGapsLiteral,
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

func preflightServiceManagerDirs(root string, serviceManagers []ServiceManagerModel, overwrite bool) error {
	if overwrite {
		return nil
	}

	seen := make(map[string]struct{}, len(serviceManagers))
	for _, serviceManager := range serviceManagers {
		outputDir := serviceManagerOutputDir(root, serviceManager)
		if _, ok := seen[outputDir]; ok {
			continue
		}
		seen[outputDir] = struct{}{}

		if _, err := os.Stat(outputDir); err == nil {
			return ErrTargetExists{Path: outputDir}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat service-manager dir %q: %w", outputDir, err)
		}
	}

	return nil
}

func serviceManagerOutputDir(root string, serviceManager ServiceManagerModel) string {
	return filepath.Join(root, "pkg", "servicemanager", filepath.FromSlash(serviceManager.PackagePath))
}

func controllerOutputDir(root string, pkg *PackageModel) string {
	return filepath.Join(root, "controllers", pkg.Service.Group)
}

func registrationOutputDir(root string) string {
	return filepath.Join(root, "internal", "registrations")
}

func writeGeneratedFile(path string, content string, overwrite bool) error {
	if _, err := os.Stat(path); err == nil && !overwrite {
		return ErrTargetExists{Path: path}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat generated file %q: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write generated file %q: %w", path, err)
	}
	return nil
}

const groupVersionTemplate = `/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Code generated by generator. DO NOT EDIT.

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

// Code generated by generator. DO NOT EDIT.

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

const controllerTemplate = `/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Code generated by generator. DO NOT EDIT.

package {{ .PackageName }}

import (
	"context"

	{{ .APIImportAlias }} "github.com/oracle/oci-service-operator/api/{{ .Group }}/{{ .Version }}"
	"github.com/oracle/oci-service-operator/pkg/core"
{{- if .HasMaxConcurrentReconciles }}
	"sigs.k8s.io/controller-runtime/pkg/controller"
{{- end }}
	ctrl "sigs.k8s.io/controller-runtime"
)

// {{ .ReconcilerType }} reconciles a {{ .Kind }} object.
type {{ .ReconcilerType }} struct {
	Reconciler *core.BaseReconciler
}

{{- range .RBACMarkers }}
{{ marker . }}
{{- end }}

// Reconcile is part of the main Kubernetes reconciliation loop.
func (r *{{ .ReconcilerType }}) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	{{ .ResourceVariable }} := &{{ .APIImportAlias }}.{{ .Kind }}{}
	return r.Reconciler.Reconcile(ctx, req, {{ .ResourceVariable }})
}

// SetupWithManager sets up the controller with the Manager.
func (r *{{ .ReconcilerType }}) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&{{ .APIImportAlias }}.{{ .Kind }}{})
{{- if .HasMaxConcurrentReconciles }}
	builder = builder.WithOptions(controller.Options{MaxConcurrentReconciles: {{ .MaxConcurrentReconciles }}})
{{- end }}
	return builder.
		WithEventFilter(core.ReconcilePredicate()).
		Complete(r)
}
`

const registrationTemplate = `/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Code generated by generator. DO NOT EDIT.

package registrations

import (
	"fmt"

	{{ .APIImportAlias }} "{{ .APIImportPath }}"
	{{ .ControllerImportAlias }} "{{ .ControllerImportPath }}"
{{- range .Resources }}
	{{ .ServiceManagerImportAlias }} "{{ .ServiceManagerImportPath }}"
{{- end }}
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
)

func init() {
	registerGeneratedGroup(GroupRegistration{
		Group:       "{{ .Group }}",
		AddToScheme: {{ .APIImportAlias }}.AddToScheme,
		SetupWithManager: func(ctx Context) error {
{{- range .Resources }}
			if err := (&{{ $.ControllerImportAlias }}.{{ .ReconcilerType }}{
				Reconciler: NewBaseReconciler(
					ctx,
					"{{ .ComponentName }}",
					func(deps servicemanager.RuntimeDeps) servicemanager.OSOKServiceManager {
						return {{ .ServiceManagerImportAlias }}.{{ .WithDepsConstructor }}(deps)
					},
				),
			}).SetupWithManager(ctx.Manager); err != nil {
				return fmt.Errorf("setup {{ .Kind }} controller: %w", err)
			}
{{- end }}
			return nil
		},
	})
}
`

const serviceClientTemplate = `/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Code generated by generator. DO NOT EDIT.

package {{ .PackageName }}

import (
	"context"
	"fmt"

	{{ .SDKImportAlias }} "{{ .SDKImportPath }}"
	{{ .APIImportAlias }} "{{ .APIImportPath }}"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// {{ .ClientInterfaceName }} is the handwritten extension seam for {{ .Kind }} runtime behavior.
// Add a manual file in this package that implements the interface and wire it through
// (*{{ .ManagerTypeName }}).WithClient.
type {{ .ClientInterfaceName }} interface {
	CreateOrUpdate(context.Context, *{{ .APIImportAlias }}.{{ .Kind }}, ctrl.Request) (servicemanager.OSOKResponse, error)
	Delete(context.Context, *{{ .APIImportAlias }}.{{ .Kind }}) (bool, error)
}

type {{ .DefaultClientTypeName }} struct {
	generatedruntime.ServiceClient[*{{ .APIImportAlias }}.{{ .Kind }}]
}

var _ {{ .ClientInterfaceName }} = {{ .DefaultClientTypeName }}{}

var new{{ .Kind }}ServiceClient = func(manager *{{ .ManagerTypeName }}) {{ .ClientInterfaceName }} {
{{- if eq .SDKClientConstructorKind "provider" }}
	sdkClient, err := {{ .SDKImportAlias }}.{{ .SDKClientConstructor }}(manager.Provider)
{{- else }}
	var (
		sdkClient {{ .SDKImportAlias }}.{{ .SDKClientTypeName }}
		err error
	)
{{- if eq .SDKClientConstructorKind "provider_endpoint" }}
	err = fmt.Errorf("{{ .SDKImportAlias }}.{{ .SDKClientConstructor }} requires an explicit service endpoint")
{{- else }}
	err = fmt.Errorf("unsupported SDK client constructor signature for {{ .SDKImportAlias }}.{{ .SDKClientConstructor }}")
{{- end }}
{{- end }}
	config := generatedruntime.Config[*{{ .APIImportAlias }}.{{ .Kind }}]{
		Kind:             "{{ .Kind }}",
		SDKName:          "{{ .SDKName }}",
		Log:              manager.Log,
{{- if or .UsesCredentialClient .NeedsCredentialClient }}
		CredentialClient: manager.CredentialClient,
{{- end }}
{{- if .Semantics }}
		Semantics: &generatedruntime.Semantics{
			FormalService:     "{{ .Semantics.FormalService }}",
			FormalSlug:        "{{ .Semantics.FormalSlug }}",
			StatusProjection:  "{{ .Semantics.StatusProjection }}",
			SecretSideEffects: "{{ .Semantics.SecretSideEffects }}",
			FinalizerPolicy:   "{{ .Semantics.FinalizerPolicy }}",
			Lifecycle: generatedruntime.LifecycleSemantics{
				ProvisioningStates: {{ stringSliceLiteral .Semantics.Lifecycle.ProvisioningStates }},
				UpdatingStates:     {{ stringSliceLiteral .Semantics.Lifecycle.UpdatingStates }},
				ActiveStates:       {{ stringSliceLiteral .Semantics.Lifecycle.ActiveStates }},
			},
			Delete: generatedruntime.DeleteSemantics{
				Policy:         "{{ .Semantics.Delete.Policy }}",
				PendingStates:  {{ stringSliceLiteral .Semantics.Delete.PendingStates }},
				TerminalStates: {{ stringSliceLiteral .Semantics.Delete.TerminalStates }},
			},
{{- if .Semantics.List }}
			List: &generatedruntime.ListSemantics{
				ResponseItemsField: "{{ .Semantics.List.ResponseItemsField }}",
				MatchFields:        {{ stringSliceLiteral .Semantics.List.MatchFields }},
			},
{{- end }}
			Mutation: generatedruntime.MutationSemantics{
				Mutable:       {{ stringSliceLiteral .Semantics.Mutation.Mutable }},
				ForceNew:      {{ stringSliceLiteral .Semantics.Mutation.ForceNew }},
				ConflictsWith: {{ stringSliceMapLiteral .Semantics.Mutation.ConflictsWith }},
			},
			Hooks: generatedruntime.HookSet{
				Create: {{ runtimeHooksLiteral .Semantics.Hooks.Create }},
				Update: {{ runtimeHooksLiteral .Semantics.Hooks.Update }},
				Delete: {{ runtimeHooksLiteral .Semantics.Hooks.Delete }},
			},
			CreateFollowUp: generatedruntime.FollowUpSemantics{
				Strategy: "{{ .Semantics.CreateFollowUp.Strategy }}",
				Hooks:    {{ runtimeHooksLiteral .Semantics.CreateFollowUp.Hooks }},
			},
			UpdateFollowUp: generatedruntime.FollowUpSemantics{
				Strategy: "{{ .Semantics.UpdateFollowUp.Strategy }}",
				Hooks:    {{ runtimeHooksLiteral .Semantics.UpdateFollowUp.Hooks }},
			},
			DeleteFollowUp: generatedruntime.FollowUpSemantics{
				Strategy: "{{ .Semantics.DeleteFollowUp.Strategy }}",
				Hooks:    {{ runtimeHooksLiteral .Semantics.DeleteFollowUp.Hooks }},
			},
			AuxiliaryOperations: {{ runtimeAuxOpsLiteral .Semantics.AuxiliaryOperations }},
			Unsupported:         {{ runtimeGapsLiteral .Semantics.OpenGaps }},
		},
{{- end }}
{{- if .CreateOperation }}
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &{{ $.SDKImportAlias }}.{{ .CreateOperation.RequestTypeName }}{} },
			Call: func(ctx context.Context, request any) (any, error) {
{{- if .CreateOperation.UsesRequest }}
				return sdkClient.{{ .CreateOperation.MethodName }}(ctx, *request.(*{{ $.SDKImportAlias }}.{{ .CreateOperation.RequestTypeName }}))
{{- else }}
				return sdkClient.{{ .CreateOperation.MethodName }}(ctx)
{{- end }}
			},
{{- if $.Semantics }}
			Fields: {{ requestFieldsLiteral .CreateOperation.RequestFields }},
{{- end }}
		},
{{- end }}
{{- if .GetOperation }}
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &{{ $.SDKImportAlias }}.{{ .GetOperation.RequestTypeName }}{} },
			Call: func(ctx context.Context, request any) (any, error) {
{{- if .GetOperation.UsesRequest }}
				return sdkClient.{{ .GetOperation.MethodName }}(ctx, *request.(*{{ $.SDKImportAlias }}.{{ .GetOperation.RequestTypeName }}))
{{- else }}
				return sdkClient.{{ .GetOperation.MethodName }}(ctx)
{{- end }}
			},
{{- if $.Semantics }}
			Fields: {{ requestFieldsLiteral .GetOperation.RequestFields }},
{{- end }}
		},
{{- end }}
{{- if .ListOperation }}
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &{{ $.SDKImportAlias }}.{{ .ListOperation.RequestTypeName }}{} },
			Call: func(ctx context.Context, request any) (any, error) {
{{- if .ListOperation.UsesRequest }}
				return sdkClient.{{ .ListOperation.MethodName }}(ctx, *request.(*{{ $.SDKImportAlias }}.{{ .ListOperation.RequestTypeName }}))
{{- else }}
				return sdkClient.{{ .ListOperation.MethodName }}(ctx)
{{- end }}
			},
{{- if $.Semantics }}
			Fields: {{ requestFieldsLiteral .ListOperation.RequestFields }},
{{- end }}
		},
{{- end }}
{{- if .UpdateOperation }}
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &{{ $.SDKImportAlias }}.{{ .UpdateOperation.RequestTypeName }}{} },
			Call: func(ctx context.Context, request any) (any, error) {
{{- if .UpdateOperation.UsesRequest }}
				return sdkClient.{{ .UpdateOperation.MethodName }}(ctx, *request.(*{{ $.SDKImportAlias }}.{{ .UpdateOperation.RequestTypeName }}))
{{- else }}
				return sdkClient.{{ .UpdateOperation.MethodName }}(ctx)
{{- end }}
			},
{{- if $.Semantics }}
			Fields: {{ requestFieldsLiteral .UpdateOperation.RequestFields }},
{{- end }}
		},
{{- end }}
{{- if .DeleteOperation }}
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &{{ $.SDKImportAlias }}.{{ .DeleteOperation.RequestTypeName }}{} },
			Call: func(ctx context.Context, request any) (any, error) {
{{- if .DeleteOperation.UsesRequest }}
				return sdkClient.{{ .DeleteOperation.MethodName }}(ctx, *request.(*{{ $.SDKImportAlias }}.{{ .DeleteOperation.RequestTypeName }}))
{{- else }}
				return sdkClient.{{ .DeleteOperation.MethodName }}(ctx)
{{- end }}
			},
{{- if $.Semantics }}
			Fields: {{ requestFieldsLiteral .DeleteOperation.RequestFields }},
{{- end }}
		},
{{- end }}
	}
	if err != nil {
		config.InitError = fmt.Errorf("initialize {{ .Kind }} OCI client: %w", err)
	}
	return {{ .DefaultClientTypeName }}{
		ServiceClient: generatedruntime.NewServiceClient[*{{ .APIImportAlias }}.{{ .Kind }}](config),
	}
}
`

const serviceManagerTemplate = `/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Code generated by generator. DO NOT EDIT.

package {{ .PackageName }}

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	{{ .APIImportAlias }} "{{ .APIImportPath }}"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type {{ .ManagerTypeName }} struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics

	client {{ .ClientInterfaceName }}
}

var _ servicemanager.OSOKServiceManager = (*{{ .ManagerTypeName }})(nil)

func {{ .WithDepsConstructor }}(deps servicemanager.RuntimeDeps) *{{ .ManagerTypeName }} {
	manager := &{{ .ManagerTypeName }}{
		Provider:         deps.Provider,
		CredentialClient: deps.CredentialClient,
		Scheme:           deps.Scheme,
		Log:              deps.Log,
		Metrics:          deps.Metrics,
	}
	manager.client = new{{ .Kind }}ServiceClient(manager)
	return manager
}

func {{ .Constructor }}(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger, metrics *metrics.Metrics) *{{ .ManagerTypeName }} {
	return {{ .WithDepsConstructor }}(servicemanager.RuntimeDeps{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
		Metrics:          metrics,
	})
}

// WithClient overrides the default generated client with handwritten runtime behavior.
func (c *{{ .ManagerTypeName }}) WithClient(client {{ .ClientInterfaceName }}) *{{ .ManagerTypeName }} {
	if client != nil {
		c.client = client
	}
	return c
}

func (c *{{ .ManagerTypeName }}) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	resource, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return c.client.CreateOrUpdate(ctx, resource, req)
}

func (c *{{ .ManagerTypeName }}) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	resource, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return false, err
	}

	return c.client.Delete(ctx, resource)
}

func (c *{{ .ManagerTypeName }}) GetCrdStatus(obj runtime.Object) (*shared.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}

	return c.status(resource), nil
}

func (c *{{ .ManagerTypeName }}) convert(obj runtime.Object) (*{{ .APIImportAlias }}.{{ .Kind }}, error) {
	resource, ok := obj.(*{{ .APIImportAlias }}.{{ .Kind }})
	if !ok {
		return nil, fmt.Errorf("expected *{{ .APIImportAlias }}.{{ .Kind }}, got %T", obj)
	}
	return resource, nil
}

func (c *{{ .ManagerTypeName }}) status(resource *{{ .APIImportAlias }}.{{ .Kind }}) *shared.OSOKStatus {
	return &resource.Status.OsokStatus
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
{{- if .Patches }}

patches:
{{- range .Patches }}
- path: {{ .Path }}
{{- if .Target }}
  target:
    kind: {{ .Target }}
{{- end }}
{{- end }}
{{- end }}
`

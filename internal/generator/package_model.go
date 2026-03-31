/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

func buildPackageModel(cfg *Config, service ServiceConfig, discovered []ResourceModel) (*PackageModel, error) {
	version := service.VersionOrDefault(cfg.DefaultVersion)
	resources := assignHelperTypeNames(discovered)
	resources = assignStatusTypeNames(resources)
	resources = applyDefaultSamples(service, version, resources)
	controllerOutput := buildControllerOutputModel(service, cfg.Domain, resources)
	serviceManagers, err := buildServiceManagerModels(service, version, resources)
	if err != nil {
		return nil, err
	}
	registrationOutput, err := buildRegistrationOutputModel(service, version, resources, controllerOutput, serviceManagers)
	if err != nil {
		return nil, err
	}

	return &PackageModel{
		Service:         service,
		Domain:          cfg.Domain,
		Version:         version,
		GroupDNSName:    service.GroupDNSName(cfg.Domain),
		SampleOrder:     service.SampleOrder,
		Resources:       resources,
		Controller:      controllerOutput,
		Registration:    registrationOutput,
		PackageOutput:   buildPackageOutputModel(service, resources),
		ServiceManagers: serviceManagers,
	}, nil
}

func buildParityResources(service ServiceConfig, version string, discovered []ResourceModel) ([]ResourceModel, error) {
	discoveredBySource := make(map[string]ResourceModel, len(discovered))
	for _, resource := range discovered {
		discoveredBySource[resource.SDKName] = resource
	}

	overridesBySource := make(map[string]ParityResource, len(service.Parity.Resources))
	for _, override := range service.Parity.Resources {
		if _, ok := discoveredBySource[override.SourceResource]; !ok {
			return nil, fmt.Errorf("parity resource %q for service %q was not found in SDK discovery", override.SourceResource, service.Service)
		}
		overridesBySource[override.SourceResource] = override
	}

	resources := make([]ResourceModel, 0, len(discovered))
	for _, resource := range discovered {
		override, ok := overridesBySource[resource.SDKName]
		if ok {
			resource = buildParityResourceModel(service, version, resource, override)
		}
		resources = append(resources, resource)
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Kind < resources[j].Kind
	})

	return resources, nil
}

func buildParityResourceModel(service ServiceConfig, version string, discoveredResource ResourceModel, override ParityResource) ResourceModel {
	if override.MergeWithDiscovered {
		return mergeParityResourceModel(service, version, discoveredResource, override)
	}

	fileStem := override.FileStem
	if strings.TrimSpace(fileStem) == "" {
		fileStem = strings.ToLower(override.Kind)
	}

	printColumns := convertPrintColumns(override.PrintColumns)
	if len(printColumns) == 0 {
		printColumns = discoveredResource.PrintColumns
	}

	statusFields := convertFields(override.StatusFields)
	if len(statusFields) == 0 {
		statusFields = defaultStatusFields()
	}

	return ResourceModel{
		SDKName:         discoveredResource.SDKName,
		Kind:            override.Kind,
		FileStem:        fileStem,
		KindPlural:      strings.ToLower(pluralize(override.Kind)),
		Operations:      discoveredResource.Operations,
		Runtime:         discoveredResource.Runtime,
		LeadingComments: override.LeadingComments,
		SpecComments:    override.SpecComments,
		HelperTypes:     convertHelperTypes(override.HelperTypes),
		SpecFields:      convertFields(override.SpecFields),
		StatusTypeName:  defaultStatusTypeName(override.Kind),
		StatusComments:  override.StatusComments,
		StatusFields:    statusFields,
		PrintColumns:    printColumns,
		ObjectComments:  override.ObjectComments,
		ListComments:    override.ListComments,
		Sample: SampleModel{
			Body:         override.Sample.Body,
			FileName:     sampleFileName(service.Group, version, fileStem),
			MetadataName: override.Sample.MetadataName,
			Spec:         override.Sample.Spec,
		},
		PrimaryDisplayField: discoveredResource.PrimaryDisplayField,
		CompatibilityLocked: true,
	}
}

//nolint:gocyclo // Partial parity merges must handle each optional overlay surface explicitly.
func mergeParityResourceModel(service ServiceConfig, version string, discoveredResource ResourceModel, override ParityResource) ResourceModel {
	resource := discoveredResource
	resource.Kind = override.Kind
	if strings.TrimSpace(override.FileStem) != "" {
		resource.FileStem = override.FileStem
	}
	if resource.Kind != discoveredResource.Kind || strings.TrimSpace(resource.KindPlural) == "" {
		resource.KindPlural = strings.ToLower(pluralize(resource.Kind))
	}
	if len(override.LeadingComments) > 0 {
		resource.LeadingComments = override.LeadingComments
	}
	if len(override.SpecComments) > 0 {
		resource.SpecComments = override.SpecComments
	}
	if len(override.HelperTypes) > 0 {
		resource.HelperTypes = mergeHelperTypeOverrides(resource.HelperTypes, override.HelperTypes)
	}
	if len(override.SpecFields) > 0 {
		resource.SpecFields = mergeFieldOverrides(resource.SpecFields, override.SpecFields)
	}
	if len(override.StatusComments) > 0 {
		resource.StatusComments = override.StatusComments
	}
	if len(override.StatusFields) > 0 {
		resource.StatusFields = mergeFieldOverrides(resource.StatusFields, override.StatusFields)
	}
	if len(override.PrintColumns) > 0 {
		resource.PrintColumns = convertPrintColumns(override.PrintColumns)
	}
	if len(override.ObjectComments) > 0 {
		resource.ObjectComments = override.ObjectComments
	}
	if len(override.ListComments) > 0 {
		resource.ListComments = override.ListComments
	}
	resource.Sample = mergeSampleOverride(service, version, resource.FileStem, resource.Sample, override.Sample)
	resource.CompatibilityLocked = true
	return resource
}

func buildPackageOutputModel(service ServiceConfig, resources []ResourceModel) PackageOutputModel {
	output := PackageOutputModel{
		Generate: true,
		Metadata: PackageMetadataModel{
			PackageName:            fmt.Sprintf("oci-service-operator-%s", service.Group),
			PackageNamespace:       fmt.Sprintf("oci-service-operator-%s-system", service.Group),
			PackageNamePrefix:      fmt.Sprintf("oci-service-operator-%s-", service.Group),
			CRDPaths:               fmt.Sprintf("./api/%s/...", service.Group),
			DefaultControllerImage: "iad.ocir.io/oracle/oci-service-operator:latest",
		},
	}

	switch service.PackageProfile {
	case PackageProfileControllerBacked:
		output.Metadata.RBACPaths = fmt.Sprintf("./controllers/%s/...", service.Group)
		output.Install.Namespace = fmt.Sprintf("oci-service-operator-%s-system", service.Group)
		output.Install.NamePrefix = fmt.Sprintf("oci-service-operator-%s-", service.Group)
		output.Install.PatchPath = "../../../config/default/manager_config_patch.yaml"
		output.Install.PatchTarget = "Deployment"
		output.Install.Resources = append(output.Install.Resources,
			"generated/crd",
			"generated/rbac",
			"../../../config/manager",
			"../../../config/rbac/role_binding.yaml",
			"../../../config/rbac/leader_election_role.yaml",
			"../../../config/rbac/leader_election_role_binding.yaml",
		)
	case PackageProfileCRDOnly:
		output.Install.Resources = append(output.Install.Resources, "generated/crd")
	default:
		output.Generate = false
	}

	return output
}

//nolint:gocognit,gocyclo // Registration generation validates several coupled controller and service-manager cases.
func buildRegistrationOutputModel(
	service ServiceConfig,
	version string,
	resources []ResourceModel,
	controllerOutput ControllerOutputModel,
	serviceManagers []ServiceManagerModel,
) (RegistrationOutputModel, error) {
	if service.RegistrationGenerationStrategy() != GenerationStrategyGenerated {
		return RegistrationOutputModel{}, nil
	}
	if service.PackageProfile != PackageProfileControllerBacked {
		return RegistrationOutputModel{}, fmt.Errorf(
			"service %q registration strategy %q requires packageProfile %q",
			service.Service,
			GenerationStrategyGenerated,
			PackageProfileControllerBacked,
		)
	}

	controllersByKind := make(map[string]ControllerModel, len(controllerOutput.Resources))
	for _, controller := range controllerOutput.Resources {
		controllersByKind[controller.Kind] = controller
	}

	serviceManagersByKind := make(map[string]ServiceManagerModel, len(serviceManagers))
	for _, serviceManager := range serviceManagers {
		serviceManagersByKind[serviceManager.Kind] = serviceManager
	}

	output := RegistrationOutputModel{
		Group:                 service.Group,
		APIImportPath:         fmt.Sprintf("github.com/oracle/oci-service-operator/api/%s/%s", service.Group, version),
		APIImportAlias:        fmt.Sprintf("%s%s", service.Group, version),
		ControllerImportPath:  fmt.Sprintf("github.com/oracle/oci-service-operator/controllers/%s", service.Group),
		ControllerImportAlias: service.Group + "controllers",
		Resources:             make([]RegistrationResourceModel, 0, len(resources)),
	}

	for _, resource := range resources {
		controllerStrategy := service.ControllerGenerationStrategyFor(resource.Kind)
		serviceManagerStrategy := service.ServiceManagerGenerationStrategyFor(resource.Kind)
		if controllerStrategy != GenerationStrategyGenerated && serviceManagerStrategy != GenerationStrategyGenerated {
			continue
		}
		if controllerStrategy != GenerationStrategyGenerated {
			return RegistrationOutputModel{}, fmt.Errorf(
				"service %q registration strategy %q requires generated controller output for kind %q",
				service.Service,
				GenerationStrategyGenerated,
				resource.Kind,
			)
		}
		if serviceManagerStrategy != GenerationStrategyGenerated {
			return RegistrationOutputModel{}, fmt.Errorf(
				"service %q registration strategy %q requires generated service-manager output for kind %q",
				service.Service,
				GenerationStrategyGenerated,
				resource.Kind,
			)
		}

		controller, ok := controllersByKind[resource.Kind]
		if !ok {
			return RegistrationOutputModel{}, fmt.Errorf(
				"service %q registration strategy %q requires generated controller output for kind %q",
				service.Service,
				GenerationStrategyGenerated,
				resource.Kind,
			)
		}

		serviceManager, ok := serviceManagersByKind[resource.Kind]
		if !ok {
			return RegistrationOutputModel{}, fmt.Errorf(
				"service %q registration strategy %q requires generated service-manager output for kind %q",
				service.Service,
				GenerationStrategyGenerated,
				resource.Kind,
			)
		}

		output.Resources = append(output.Resources, RegistrationResourceModel{
			Kind:                      resource.Kind,
			ComponentName:             resource.Kind,
			ReconcilerType:            controller.ReconcilerType,
			ServiceManagerImportPath:  fmt.Sprintf("github.com/oracle/oci-service-operator/pkg/servicemanager/%s", serviceManager.PackagePath),
			ServiceManagerImportAlias: registrationServiceManagerImportAlias(service.Group, serviceManager.FileStem),
			WithDepsConstructor:       serviceManager.WithDepsConstructor,
		})
	}

	if len(output.Resources) == 0 {
		return RegistrationOutputModel{}, fmt.Errorf(
			"service %q registration strategy %q requires at least one generated resource",
			service.Service,
			GenerationStrategyGenerated,
		)
	}

	return output, nil
}

func buildControllerOutputModel(service ServiceConfig, domain string, resources []ResourceModel) ControllerOutputModel {
	output := ControllerOutputModel{
		Resources: make([]ControllerModel, 0, len(resources)),
	}

	groupDNSName := service.GroupDNSName(domain)
	for _, resource := range resources {
		controllerConfig := service.ControllerGenerationConfigFor(resource.Kind)
		if controllerConfig.Strategy != GenerationStrategyGenerated {
			continue
		}

		kindPlural := resource.KindPlural
		if strings.TrimSpace(kindPlural) == "" {
			kindPlural = strings.ToLower(pluralize(resource.Kind))
		}

		rbacMarkers := append(
			defaultControllerRBACMarkers(groupDNSName, kindPlural),
			extraControllerRBACMarkers(controllerConfig.ExtraRBACMarkers)...,
		)

		output.Resources = append(output.Resources, ControllerModel{
			Kind:                       resource.Kind,
			FileStem:                   resource.FileStem,
			ReconcilerType:             fmt.Sprintf("%sReconciler", resource.Kind),
			ResourceVariable:           lowerCamel(resource.Kind),
			MaxConcurrentReconciles:    controllerConfig.MaxConcurrentReconciles,
			HasMaxConcurrentReconciles: controllerConfig.MaxConcurrentReconciles > 0,
			RBACMarkers:                rbacMarkers,
		})
	}

	return output
}

func registrationServiceManagerImportAlias(group string, fileStem string) string {
	return group + fileStem + "servicemanager"
}

func buildServiceManagerModels(service ServiceConfig, version string, resources []ResourceModel) ([]ServiceManagerModel, error) {
	serviceManagers := make([]ServiceManagerModel, 0, len(resources))
	for _, resource := range resources {
		if service.ServiceManagerGenerationStrategyFor(resource.Kind) != GenerationStrategyGenerated {
			continue
		}
		if resource.Runtime == nil {
			return nil, fmt.Errorf(
				"service %q generated service-manager resource %q is missing SDK runtime metadata",
				service.Service,
				resource.Kind,
			)
		}

		packagePath := service.ServiceManagerPackagePathFor(resource.Kind, resource.FileStem)
		serviceManagers = append(serviceManagers, ServiceManagerModel{
			Kind:                     resource.Kind,
			SDKName:                  resource.SDKName,
			FileStem:                 resource.FileStem,
			Formal:                   resource.Formal,
			Semantics:                resource.Runtime.Semantics,
			PackagePath:              packagePath,
			PackageName:              path.Base(packagePath),
			APIImportPath:            fmt.Sprintf("github.com/oracle/oci-service-operator/api/%s/%s", service.Group, version),
			APIImportAlias:           fmt.Sprintf("%s%s", service.Group, version),
			SDKImportPath:            service.SDKPackage,
			SDKImportAlias:           service.Group + "sdk",
			ManagerTypeName:          fmt.Sprintf("%sServiceManager", resource.Kind),
			WithDepsConstructor:      fmt.Sprintf("New%sServiceManagerWithDeps", resource.Kind),
			Constructor:              fmt.Sprintf("New%sServiceManager", resource.Kind),
			ClientInterfaceName:      fmt.Sprintf("%sServiceClient", resource.Kind),
			DefaultClientTypeName:    fmt.Sprintf("default%sServiceClient", resource.Kind),
			SDKClientTypeName:        resource.Runtime.ClientType,
			SDKClientConstructor:     resource.Runtime.ClientConstructor,
			SDKClientConstructorKind: resource.Runtime.ClientConstructorKind,
			NeedsCredentialClient:    service.ServiceManagerNeedsCredentialClientFor(resource.Kind),
			CreateOperation:          resource.Runtime.Create,
			GetOperation:             resource.Runtime.Get,
			ListOperation:            resource.Runtime.List,
			UpdateOperation:          resource.Runtime.Update,
			DeleteOperation:          resource.Runtime.Delete,
			ServiceClientFileName:    fmt.Sprintf("%s_serviceclient.go", resource.FileStem),
			ServiceManagerFileName:   fmt.Sprintf("%s_servicemanager.go", resource.FileStem),
		})
	}

	sort.Slice(serviceManagers, func(i, j int) bool {
		if serviceManagers[i].PackagePath == serviceManagers[j].PackagePath {
			return serviceManagers[i].Kind < serviceManagers[j].Kind
		}
		return serviceManagers[i].PackagePath < serviceManagers[j].PackagePath
	})

	return serviceManagers, nil
}

func defaultControllerRBACMarkers(groupDNSName string, kindPlural string) []string {
	return []string{
		fmt.Sprintf("+kubebuilder:rbac:groups=%s,resources=%s,verbs=get;list;watch;create;update;patch;delete", groupDNSName, kindPlural),
		fmt.Sprintf("+kubebuilder:rbac:groups=%s,resources=%s/status,verbs=get;update;patch", groupDNSName, kindPlural),
		fmt.Sprintf("+kubebuilder:rbac:groups=%s,resources=%s/finalizers,verbs=update", groupDNSName, kindPlural),
	}
}

func extraControllerRBACMarkers(markers []string) []string {
	normalized := make([]string, 0, len(markers))
	for _, marker := range markers {
		marker = strings.TrimSpace(marker)
		if marker == "" {
			continue
		}
		if strings.HasPrefix(marker, "+kubebuilder:") {
			normalized = append(normalized, marker)
			continue
		}
		normalized = append(normalized, "+kubebuilder:rbac:"+marker)
	}
	return normalized
}

func applyDefaultSamples(service ServiceConfig, version string, resources []ResourceModel) []ResourceModel {
	updated := make([]ResourceModel, 0, len(resources))
	for _, resource := range resources {
		if strings.TrimSpace(resource.Sample.FileName) == "" {
			resource.Sample.FileName = sampleFileName(service.Group, version, resource.FileStem)
		}
		if strings.TrimSpace(resource.Sample.MetadataName) == "" {
			resource.Sample.MetadataName = resource.FileStem + "-sample"
		}
		updated = append(updated, resource)
	}
	return updated
}

func appendUniqueStrings(existing []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(extras))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range extras {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		existing = append(existing, value)
	}
	return existing
}

func convertHelperTypes(overrides []TypeOverride) []TypeModel {
	types := make([]TypeModel, 0, len(overrides))
	for _, override := range overrides {
		types = append(types, TypeModel{
			Name:     override.Name,
			Comments: override.Comments,
			Fields:   convertFields(override.Fields),
		})
	}
	return types
}

func convertFields(overrides []FieldOverride) []FieldModel {
	fields := make([]FieldModel, 0, len(overrides))
	for _, override := range overrides {
		fields = append(fields, FieldModel{
			Name:     override.Name,
			Type:     override.Type,
			Tag:      override.Tag,
			Comments: override.Comments,
			Markers:  override.Markers,
			Embedded: strings.TrimSpace(override.Name) == "",
		})
	}
	return fields
}

func mergeHelperTypeOverrides(existing []TypeModel, overrides []TypeOverride) []TypeModel {
	merged := append([]TypeModel(nil), existing...)
	indexByName := make(map[string]int, len(existing))
	for index, helperType := range existing {
		indexByName[helperType.Name] = index
	}

	for _, override := range overrides {
		converted := TypeModel{
			Name:     override.Name,
			Comments: override.Comments,
			Fields:   convertFields(override.Fields),
		}
		if index, ok := indexByName[override.Name]; ok {
			merged[index] = converted
			continue
		}
		indexByName[override.Name] = len(merged)
		merged = append(merged, converted)
	}

	return merged
}

func mergeFieldOverrides(existing []FieldModel, overrides []FieldOverride) []FieldModel {
	merged := append([]FieldModel(nil), existing...)
	indexByKey := make(map[string]int, len(existing))
	for index, field := range existing {
		indexByKey[fieldMergeKey(field.Name, field.Type, field.Tag)] = index
	}

	for _, override := range overrides {
		converted := FieldModel{
			Name:     override.Name,
			Type:     override.Type,
			Tag:      override.Tag,
			Comments: override.Comments,
			Markers:  override.Markers,
			Embedded: strings.TrimSpace(override.Name) == "",
		}
		key := fieldMergeKey(override.Name, override.Type, override.Tag)
		if index, ok := indexByKey[key]; ok {
			merged[index] = converted
			continue
		}
		indexByKey[key] = len(merged)
		merged = append(merged, converted)
	}

	return merged
}

func fieldMergeKey(name string, typ string, tag string) string {
	if strings.TrimSpace(name) != "" {
		return "name:" + name
	}
	if jsonName := jsonTagName(tag); jsonName != "" {
		return "json:" + jsonName
	}
	return "type:" + strings.TrimSpace(typ) + "|tag:" + strings.TrimSpace(tag)
}

func jsonTagName(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.Trim(tag, "`")
	if tag == "" {
		return ""
	}
	const prefix = `json:"`
	start := strings.Index(tag, prefix)
	if start == -1 {
		return ""
	}
	tag = tag[start+len(prefix):]
	end := strings.Index(tag, `"`)
	if end == -1 {
		return ""
	}
	tag = tag[:end]
	if tag == "" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

func convertPrintColumns(overrides []PrintColumnOverride) []PrintColumnModel {
	printColumns := make([]PrintColumnModel, 0, len(overrides))
	for _, override := range overrides {
		printColumns = append(printColumns, PrintColumnModel(override))
	}
	return printColumns
}

func mergeSampleOverride(service ServiceConfig, version string, fileStem string, existing SampleModel, override SampleOverride) SampleModel {
	sample := existing
	if strings.TrimSpace(sample.FileName) == "" {
		sample.FileName = sampleFileName(service.Group, version, fileStem)
	}
	if strings.TrimSpace(override.Body) != "" {
		sample.Body = override.Body
	}
	if strings.TrimSpace(override.MetadataName) != "" {
		sample.MetadataName = override.MetadataName
	}
	if strings.TrimSpace(override.Spec) != "" {
		sample.Spec = override.Spec
	}
	return sample
}

func sampleFileName(group string, version string, fileStem string) string {
	return fmt.Sprintf("%s_%s_%s.yaml", group, version, fileStem)
}

func defaultStatusFields() []FieldModel {
	return []FieldModel{
		{
			Name: "OsokStatus",
			Type: "shared.OSOKStatus",
			Tag:  `json:"status"`,
		},
	}
}

//nolint:gocognit // Helper renaming coordinates reserved names, rewrites, and comment carryover in one pass.
func assignHelperTypeNames(resources []ResourceModel) []ResourceModel {
	reservedNames := make(map[string]struct{}, len(resources))
	for _, resource := range resources {
		reservedNames[resource.Kind] = struct{}{}
	}

	usedHelperNames := make(map[string]struct{}, len(resources))
	updated := make([]ResourceModel, 0, len(resources))
	for _, resource := range resources {
		renames := make(map[string]string, len(resource.HelperTypes))
		for _, helperType := range resource.HelperTypes {
			name := helperType.Name
			if nameConflicts(name, reservedNames, usedHelperNames) {
				name = uniqueHelperTypeName(helperType.Name, reservedNames, usedHelperNames)
			}
			if name != helperType.Name {
				renames[helperType.Name] = name
			}
			usedHelperNames[name] = struct{}{}
		}

		if len(renames) > 0 {
			resource.SpecFields = rewriteFieldTypes(resource.SpecFields, renames)
			resource.StatusFields = rewriteFieldTypes(resource.StatusFields, renames)
		}

		helperTypes := make([]TypeModel, 0, len(resource.HelperTypes))
		for _, helperType := range resource.HelperTypes {
			if renamed, ok := renames[helperType.Name]; ok {
				helperType = renameHelperType(helperType, renamed)
			}
			if len(renames) > 0 {
				helperType.Fields = rewriteFieldTypes(helperType.Fields, renames)
			}
			helperTypes = append(helperTypes, helperType)
		}
		resource.HelperTypes = helperTypes
		updated = append(updated, resource)
	}

	return updated
}

func rewriteFieldTypes(fields []FieldModel, renames map[string]string) []FieldModel {
	rewritten := make([]FieldModel, 0, len(fields))
	for _, field := range fields {
		field.Type = rewriteFieldType(field.Type, renames)
		rewritten = append(rewritten, field)
	}
	return rewritten
}

func rewriteFieldType(typeExpr string, renames map[string]string) string {
	trimmed := strings.TrimSpace(typeExpr)
	switch {
	case trimmed == "":
		return trimmed
	case strings.HasPrefix(trimmed, "*"):
		return "*" + rewriteFieldType(strings.TrimPrefix(trimmed, "*"), renames)
	case strings.HasPrefix(trimmed, "[]"):
		return "[]" + rewriteFieldType(strings.TrimPrefix(trimmed, "[]"), renames)
	case strings.HasPrefix(trimmed, "map[string]"):
		return "map[string]" + rewriteFieldType(strings.TrimPrefix(trimmed, "map[string]"), renames)
	default:
		if renamed, ok := renames[trimmed]; ok {
			return renamed
		}
		return trimmed
	}
}

func renameHelperType(helperType TypeModel, newName string) TypeModel {
	oldName := helperType.Name
	helperType.Name = newName
	for index, comment := range helperType.Comments {
		prefix := oldName + " defines nested fields for "
		if strings.HasPrefix(comment, prefix) {
			helperType.Comments[index] = newName + " defines nested fields for " + strings.TrimPrefix(comment, prefix)
		}
	}
	return helperType
}

func assignStatusTypeNames(resources []ResourceModel) []ResourceModel {
	reservedNames := make(map[string]struct{}, len(resources))
	for _, resource := range resources {
		reservedNames[resource.Kind] = struct{}{}
		for _, helperType := range resource.HelperTypes {
			reservedNames[helperType.Name] = struct{}{}
		}
	}

	usedStatusNames := make(map[string]struct{}, len(resources))
	updated := make([]ResourceModel, 0, len(resources))
	for _, resource := range resources {
		statusTypeName := resource.StatusTypeName
		if strings.TrimSpace(statusTypeName) == "" {
			statusTypeName = defaultStatusTypeName(resource.Kind)
		}

		if nameConflicts(statusTypeName, reservedNames, usedStatusNames) {
			statusTypeName = uniqueStatusTypeName(resource.Kind, reservedNames, usedStatusNames)
		}

		if usesDefaultStatusComment(resource.StatusComments, resource.StatusTypeName, resource.Kind) {
			resource.StatusComments = []string{fmt.Sprintf("%s defines the observed state of %s.", statusTypeName, resource.Kind)}
		}
		resource.StatusTypeName = statusTypeName
		usedStatusNames[statusTypeName] = struct{}{}
		updated = append(updated, resource)
	}

	return updated
}

func defaultStatusTypeName(kind string) string {
	if strings.HasSuffix(kind, "Status") || strings.HasSuffix(kind, "Stats") {
		return kind + "ObservedState"
	}
	return kind + "Status"
}

func usesDefaultStatusComment(comments []string, statusTypeName string, kind string) bool {
	if len(comments) != 1 {
		return false
	}
	if strings.TrimSpace(statusTypeName) == "" {
		statusTypeName = defaultStatusTypeName(kind)
	}
	return comments[0] == fmt.Sprintf("%s defines the observed state of %s.", statusTypeName, kind)
}

func uniqueHelperTypeName(name string, reservedNames map[string]struct{}, usedHelperNames map[string]struct{}) string {
	candidates := []string{
		name + "Fields",
		name + "Details",
	}

	for _, candidate := range candidates {
		if !nameConflicts(candidate, reservedNames, usedHelperNames) {
			return candidate
		}
	}

	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%sFields%d", name, index)
		if !nameConflicts(candidate, reservedNames, usedHelperNames) {
			return candidate
		}
	}
}

func uniqueStatusTypeName(kind string, reservedNames map[string]struct{}, usedStatusNames map[string]struct{}) string {
	candidates := []string{
		kind + "ObservedState",
		kind + "StatusDetails",
	}

	for _, candidate := range candidates {
		if !nameConflicts(candidate, reservedNames, usedStatusNames) {
			return candidate
		}
	}

	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%sObservedState%d", kind, index)
		if !nameConflicts(candidate, reservedNames, usedStatusNames) {
			return candidate
		}
	}
}

func nameConflicts(candidate string, reservedNames map[string]struct{}, usedStatusNames map[string]struct{}) bool {
	if _, exists := reservedNames[candidate]; exists {
		return true
	}
	if _, exists := usedStatusNames[candidate]; exists {
		return true
	}
	return false
}

func defaultPrintColumns(kind string, primaryDisplayField string) []PrintColumnModel {
	var printColumns []PrintColumnModel
	switch primaryDisplayField {
	case "DisplayName":
		printColumns = append(printColumns, PrintColumnModel{
			Name:     "DisplayName",
			Type:     "string",
			JSONPath: ".spec.displayName",
			Priority: intPtr(1),
		})
	case "Name":
		printColumns = append(printColumns, PrintColumnModel{
			Name:     "Name",
			Type:     "string",
			JSONPath: ".spec.name",
			Priority: intPtr(1),
		})
	}

	printColumns = append(printColumns,
		PrintColumnModel{
			Name:        "Status",
			Type:        "string",
			JSONPath:    ".status.status.conditions[-1].type",
			Description: fmt.Sprintf("status of the %s", kind),
			Priority:    intPtr(0),
		},
		PrintColumnModel{
			Name:        "Ocid",
			Type:        "string",
			JSONPath:    ".status.status.ocid",
			Description: fmt.Sprintf("Ocid of the %s", kind),
			Priority:    intPtr(1),
		},
		PrintColumnModel{
			Name:     "Age",
			Type:     "date",
			JSONPath: ".metadata.creationTimestamp",
			Priority: intPtr(0),
		},
	)

	return printColumns
}

func intPtr(value int) *int {
	return &value
}

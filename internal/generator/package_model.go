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
	resources := discovered
	resources = assignHelperTypeNames(resources)
	resources = assignStatusTypeNames(resources)
	resources = applyResourceGenerationOverrides(service, version, resources)
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

func buildPackageOutputModel(service ServiceConfig, resources []ResourceModel) PackageOutputModel {
	defaultControllerImage := fmt.Sprintf("iad.ocir.io/oracle/oci-service-operator-%s:latest", service.Group)
	managerOverlay := fmt.Sprintf("../../../config/manager/%s", service.Group)

	output := PackageOutputModel{
		Generate: true,
		Metadata: PackageMetadataModel{
			PackageName:            fmt.Sprintf("oci-service-operator-%s", service.Group),
			PackageNamespace:       fmt.Sprintf("oci-service-operator-%s-system", service.Group),
			PackageNamePrefix:      fmt.Sprintf("oci-service-operator-%s-", service.Group),
			CRDPaths:               fmt.Sprintf("./api/%s/...", service.Group),
			DefaultControllerImage: defaultControllerImage,
		},
	}

	switch service.PackageProfile {
	case PackageProfileControllerBacked:
		output.Metadata.RBACPaths = fmt.Sprintf("./controllers/%s/...", service.Group)
		output.Install.Namespace = fmt.Sprintf("oci-service-operator-%s-system", service.Group)
		output.Install.NamePrefix = fmt.Sprintf("oci-service-operator-%s-", service.Group)
		output.Install.Resources = append(output.Install.Resources,
			"generated/crd",
			"generated/rbac",
			managerOverlay,
			"../../../config/rbac/role_binding.yaml",
			"../../../config/rbac/leader_election_role.yaml",
			"../../../config/rbac/leader_election_role_binding.yaml",
		)
		output.Install.Resources = appendUniqueStrings(output.Install.Resources, service.Package.ExtraResources...)
		if service.WebhookGenerationStrategy() == GenerationStrategyManual {
			output.Install.Resources = appendUniqueStrings(output.Install.Resources,
				"../../../config/webhook",
				"../../../config/certmanager",
			)
			output.Install.Patches = append(output.Install.Patches,
				InstallPatchModel{Path: "../../../config/default/manager_webhook_patch.yaml", Target: "Deployment"},
				InstallPatchModel{Path: "../../../config/default/webhookcainjection_patch.yaml"},
			)
		}
		output.Install.Patches = append(output.Install.Patches, InstallPatchModel{
			Path:   "../../../config/default/manager_config_patch.yaml",
			Target: "Deployment",
		})
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
	if err := validateGeneratedRegistrationService(service); err != nil {
		return RegistrationOutputModel{}, err
	}

	controllersByKind := controllerModelsByKind(controllerOutput.Resources)
	serviceManagersByKind := serviceManagerModelsByKind(serviceManagers)
	output := newRegistrationOutputModel(service, version, len(resources))

	for _, resource := range resources {
		registrationResource, include, err := buildRegistrationResourceModel(service, resource, controllersByKind, serviceManagersByKind)
		if err != nil {
			return RegistrationOutputModel{}, err
		}
		if !include {
			continue
		}
		output.Resources = append(output.Resources, registrationResource)
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

func validateGeneratedRegistrationService(service ServiceConfig) error {
	if service.PackageProfile == PackageProfileControllerBacked {
		return nil
	}
	return fmt.Errorf(
		"service %q registration strategy %q requires packageProfile %q",
		service.Service,
		GenerationStrategyGenerated,
		PackageProfileControllerBacked,
	)
}

func newRegistrationOutputModel(service ServiceConfig, version string, resourceCount int) RegistrationOutputModel {
	return RegistrationOutputModel{
		Group:                 service.Group,
		APIImportPath:         fmt.Sprintf("github.com/oracle/oci-service-operator/api/%s/%s", service.Group, version),
		APIImportAlias:        fmt.Sprintf("%s%s", service.Group, version),
		ControllerImportPath:  fmt.Sprintf("github.com/oracle/oci-service-operator/controllers/%s", service.Group),
		ControllerImportAlias: service.Group + "controllers",
		Resources:             make([]RegistrationResourceModel, 0, resourceCount),
	}
}

func controllerModelsByKind(controllers []ControllerModel) map[string]ControllerModel {
	controllersByKind := make(map[string]ControllerModel, len(controllers))
	for _, controller := range controllers {
		controllersByKind[controller.Kind] = controller
	}
	return controllersByKind
}

func serviceManagerModelsByKind(serviceManagers []ServiceManagerModel) map[string]ServiceManagerModel {
	serviceManagersByKind := make(map[string]ServiceManagerModel, len(serviceManagers))
	for _, serviceManager := range serviceManagers {
		serviceManagersByKind[serviceManager.Kind] = serviceManager
	}
	return serviceManagersByKind
}

func buildRegistrationResourceModel(
	service ServiceConfig,
	resource ResourceModel,
	controllersByKind map[string]ControllerModel,
	serviceManagersByKind map[string]ServiceManagerModel,
) (RegistrationResourceModel, bool, error) {
	controllerStrategy := service.ControllerGenerationStrategyFor(resource.Kind)
	serviceManagerStrategy := service.ServiceManagerGenerationStrategyFor(resource.Kind)
	if controllerStrategy != GenerationStrategyGenerated && serviceManagerStrategy != GenerationStrategyGenerated {
		return RegistrationResourceModel{}, false, nil
	}
	if controllerStrategy != GenerationStrategyGenerated {
		return RegistrationResourceModel{}, false, fmt.Errorf(
			"service %q registration strategy %q requires generated controller output for kind %q",
			service.Service,
			GenerationStrategyGenerated,
			resource.Kind,
		)
	}
	if serviceManagerStrategy != GenerationStrategyGenerated {
		return RegistrationResourceModel{}, false, fmt.Errorf(
			"service %q registration strategy %q requires generated service-manager output for kind %q",
			service.Service,
			GenerationStrategyGenerated,
			resource.Kind,
		)
	}

	controller, ok := controllersByKind[resource.Kind]
	if !ok {
		return RegistrationResourceModel{}, false, fmt.Errorf(
			"service %q registration strategy %q requires generated controller output for kind %q",
			service.Service,
			GenerationStrategyGenerated,
			resource.Kind,
		)
	}

	serviceManager, ok := serviceManagersByKind[resource.Kind]
	if !ok {
		return RegistrationResourceModel{}, false, fmt.Errorf(
			"service %q registration strategy %q requires generated service-manager output for kind %q",
			service.Service,
			GenerationStrategyGenerated,
			resource.Kind,
		)
	}

	return RegistrationResourceModel{
		Kind:                      resource.Kind,
		ComponentName:             resource.Kind,
		ReconcilerType:            controller.ReconcilerType,
		ServiceManagerImportPath:  fmt.Sprintf("github.com/oracle/oci-service-operator/pkg/servicemanager/%s", serviceManager.PackagePath),
		ServiceManagerImportAlias: registrationServiceManagerImportAlias(service.Group, serviceManager.FileStem),
		WithDepsConstructor:       serviceManager.WithDepsConstructor,
	}, true, nil
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
			UsesCredentialClient:     resourceUsesCredentialClient(resource),
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

func applyResourceGenerationOverrides(service ServiceConfig, version string, resources []ResourceModel) []ResourceModel {
	updated := make([]ResourceModel, 0, len(resources))
	for _, resource := range resources {
		override, ok := service.resourceGenerationOverride(resource.Kind)
		if ok {
			resource.SpecFields = mergeFieldOverrides(resource.SpecFields, override.SpecFields)
			resource.StatusFields = mergeFieldOverrides(resource.StatusFields, override.StatusFields)
			resource.Sample = mergeSampleOverride(service, version, resource.FileStem, resource.Sample, override.Sample)
		}
		updated = append(updated, resource)
	}
	return updated
}
func resourceUsesCredentialClient(resource ResourceModel) bool {
	helperIndex := make(map[string]TypeModel, len(resource.HelperTypes))
	for _, helper := range resource.HelperTypes {
		helperIndex[helper.Name] = helper
	}

	seenHelpers := make(map[string]struct{}, len(resource.HelperTypes))
	for _, field := range resource.SpecFields {
		if fieldTypeUsesCredentialClient(field.Type, helperIndex, seenHelpers) {
			return true
		}
	}

	return false
}

func fieldTypeUsesCredentialClient(typeExpr string, helperIndex map[string]TypeModel, seenHelpers map[string]struct{}) bool {
	switch underlyingTypeName(typeExpr) {
	case "shared.PasswordSource", "shared.UsernameSource":
		return true
	}

	helper, ok := helperIndex[underlyingTypeName(typeExpr)]
	if !ok {
		return false
	}
	if _, seen := seenHelpers[helper.Name]; seen {
		return false
	}

	seenHelpers[helper.Name] = struct{}{}
	for _, field := range helper.Fields {
		if fieldTypeUsesCredentialClient(field.Type, helperIndex, seenHelpers) {
			return true
		}
	}
	return false
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

func mergeFieldOverrides(existing []FieldModel, overrides []FieldOverride) []FieldModel {
	if len(overrides) == 0 {
		return append([]FieldModel(nil), existing...)
	}

	merged := append([]FieldModel(nil), existing...)
	indexByKey := make(map[string]int, len(existing))
	for index, field := range existing {
		indexByKey[fieldMergeKey(field.Name, field.Type, field.Tag)] = index
	}

	for _, override := range overrides {
		key := fieldMergeKey(override.Name, override.Type, override.Tag)
		if index, ok := indexByKey[key]; ok {
			converted := merged[index]
			converted.Name = override.Name
			converted.Type = override.Type
			converted.Tag = override.Tag
			converted.Embedded = strings.TrimSpace(override.Name) == ""
			if len(override.Comments) > 0 {
				converted.Comments = append([]string(nil), override.Comments...)
			}
			if len(override.Markers) > 0 {
				converted.Markers = append([]string(nil), override.Markers...)
			}
			merged[index] = converted
			continue
		}
		converted := FieldModel{
			Name:     override.Name,
			Type:     override.Type,
			Tag:      override.Tag,
			Comments: append([]string(nil), override.Comments...),
			Markers:  append([]string(nil), override.Markers...),
			Embedded: strings.TrimSpace(override.Name) == "",
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
	return sampleFilePrefix(group, version) + fileStem + ".yaml"
}

func sampleGroupPrefix(group string) string {
	return fmt.Sprintf("%s_", group)
}

func sampleFilePrefix(group string, version string) string {
	return sampleGroupPrefix(group) + version + "_"
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
	reservedNames := reservedResourceNames(resources)
	usedHelperNames := make(map[string]struct{}, len(resources))
	updated := make([]ResourceModel, 0, len(resources))
	for _, resource := range resources {
		updated = append(updated, assignResourceHelperTypeNames(resource, reservedNames, usedHelperNames))
	}

	return updated
}

func reservedResourceNames(resources []ResourceModel) map[string]struct{} {
	reservedNames := make(map[string]struct{}, len(resources))
	for _, resource := range resources {
		reservedNames[resource.Kind] = struct{}{}
	}
	return reservedNames
}

func assignResourceHelperTypeNames(
	resource ResourceModel,
	reservedNames map[string]struct{},
	usedHelperNames map[string]struct{},
) ResourceModel {
	renames := helperTypeRenames(resource.HelperTypes, reservedNames, usedHelperNames)
	if len(renames) == 0 {
		return resource
	}

	resource.SpecFields = rewriteFieldTypes(resource.SpecFields, renames)
	resource.StatusFields = rewriteFieldTypes(resource.StatusFields, renames)
	resource.HelperTypes = renamedHelperTypes(resource.HelperTypes, renames)
	return resource
}

func helperTypeRenames(
	helperTypes []TypeModel,
	reservedNames map[string]struct{},
	usedHelperNames map[string]struct{},
) map[string]string {
	renames := make(map[string]string, len(helperTypes))
	for _, helperType := range helperTypes {
		name := helperType.Name
		if nameConflicts(name, reservedNames, usedHelperNames) {
			name = uniqueHelperTypeName(helperType.Name, reservedNames, usedHelperNames)
		}
		if name != helperType.Name {
			renames[helperType.Name] = name
		}
		usedHelperNames[name] = struct{}{}
	}
	return renames
}

func renamedHelperTypes(helperTypes []TypeModel, renames map[string]string) []TypeModel {
	updated := make([]TypeModel, 0, len(helperTypes))
	for _, helperType := range helperTypes {
		if renamed, ok := renames[helperType.Name]; ok {
			helperType = renameHelperType(helperType, renamed)
		}
		helperType.Fields = rewriteFieldTypes(helperType.Fields, renames)
		updated = append(updated, helperType)
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

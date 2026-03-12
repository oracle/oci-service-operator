/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"fmt"
	"sort"
	"strings"
)

func buildPackageModel(cfg *Config, service ServiceConfig, discovered []ResourceModel) (*PackageModel, error) {
	version := service.VersionOrDefault(cfg.DefaultVersion)
	resources := discovered
	if service.Parity != nil {
		var err error
		resources, err = buildParityResources(service, version, discovered)
		if err != nil {
			return nil, err
		}
	}
	resources = assignStatusTypeNames(resources)
	resources = applyDefaultSamples(service, version, resources)

	return &PackageModel{
		Service:       service,
		Domain:        cfg.Domain,
		Version:       version,
		GroupDNSName:  service.GroupDNSName(cfg.Domain),
		SampleOrder:   service.SampleOrder,
		Resources:     resources,
		PackageOutput: buildPackageOutputModel(service, resources),
	}, nil
}

func buildParityResources(service ServiceConfig, version string, discovered []ResourceModel) ([]ResourceModel, error) {
	discoveredBySource := make(map[string]ResourceModel, len(discovered))
	for _, resource := range discovered {
		discoveredBySource[resource.SDKName] = resource
	}

	resources := make([]ResourceModel, 0, len(service.Parity.Resources))
	for _, override := range service.Parity.Resources {
		discoveredResource, ok := discoveredBySource[override.SourceResource]
		if !ok {
			return nil, fmt.Errorf("parity resource %q for service %q was not found in SDK discovery", override.SourceResource, service.Service)
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

		resources = append(resources, ResourceModel{
			SDKName:         discoveredResource.SDKName,
			Kind:            override.Kind,
			FileStem:        fileStem,
			KindPlural:      strings.ToLower(pluralize(override.Kind)),
			Operations:      discoveredResource.Operations,
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
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Kind < resources[j].Kind
	})

	return resources, nil
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
		output.Install.Resources = appendUniqueStrings(output.Install.Resources, packageRoleResources(resources)...)
		if service.Parity != nil {
			output.Install.Resources = appendUniqueStrings(output.Install.Resources, service.Parity.Package.ExtraResources...)
		}
	case PackageProfileCRDOnly:
		output.Install.Resources = append(output.Install.Resources, "generated/crd")
	default:
		output.Generate = false
	}

	return output
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

func packageRoleResources(resources []ResourceModel) []string {
	roleResources := make([]string, 0, len(resources)*2)
	seen := make(map[string]struct{}, len(resources))
	for _, resource := range resources {
		if _, ok := seen[resource.FileStem]; ok {
			continue
		}
		seen[resource.FileStem] = struct{}{}
		roleResources = append(roleResources,
			fmt.Sprintf("../../../config/rbac/%s_editor_role.yaml", resource.FileStem),
			fmt.Sprintf("../../../config/rbac/%s_viewer_role.yaml", resource.FileStem),
		)
	}
	sort.Strings(roleResources)
	return roleResources
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

func convertPrintColumns(overrides []PrintColumnOverride) []PrintColumnModel {
	printColumns := make([]PrintColumnModel, 0, len(overrides))
	for _, override := range overrides {
		printColumns = append(printColumns, PrintColumnModel{
			Name:        override.Name,
			Type:        override.Type,
			JSONPath:    override.JSONPath,
			Description: override.Description,
			Priority:    override.Priority,
		})
	}
	return printColumns
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

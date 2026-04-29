package validator

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/oracle/oci-service-operator/internal/generator"
	"github.com/oracle/oci-service-operator/internal/validator/apispec"
	"github.com/oracle/oci-service-operator/internal/validator/config"
	"github.com/oracle/oci-service-operator/internal/validator/diff"
	"github.com/oracle/oci-service-operator/internal/validator/sdk"
)

type selectedValidatorSurface struct {
	services []generator.ServiceConfig
	structs  map[string]struct{}
}

func loadSelectedValidatorSurface(options config.Options) (selectedValidatorSurface, error) {
	configPath := resolveValidatorConfigPath(options.ProviderPath, options.ConfigPath)
	cfg, err := generator.LoadConfig(configPath)
	if err != nil {
		return selectedValidatorSurface{}, err
	}
	services, err := cfg.SelectDefaultActiveOrExplicitServices(options.Service, options.All)
	if err != nil {
		return selectedValidatorSurface{}, err
	}
	return selectedValidatorSurface{
		services: services,
		structs:  selectedValidatorStructs(services),
	}, nil
}

func loadUpgradeSelectedStructs(options config.Options) (map[string]struct{}, error) {
	surface, err := loadSelectedValidatorSurface(options)
	if err != nil {
		return nil, err
	}
	return surface.structs, nil
}

func resolveValidatorConfigPath(providerPath string, configPath string) string {
	root := strings.TrimSpace(providerPath)
	if root == "" {
		root = "."
	}

	configPath = strings.TrimSpace(configPath)
	switch {
	case configPath == "":
		return filepath.Clean(filepath.Join(root, "internal", "generator", "config", "services.yaml"))
	case filepath.IsAbs(configPath):
		return filepath.Clean(configPath)
	default:
		return filepath.Clean(filepath.Join(root, configPath))
	}
}

func selectedValidatorStructs(services []generator.ServiceConfig) map[string]struct{} {
	selected := make(map[string]struct{})
	allTargetsByService := make(map[string][]sdk.Target)
	for _, target := range sdk.SeedTargets() {
		allTargetsByService[target.PackageName] = append(allTargetsByService[target.PackageName], target)
	}

	registryTargets := apispec.Targets()
	for _, service := range services {
		if !service.HasSelectedKinds() {
			for _, target := range allTargetsByService[service.Service] {
				selected[target.QualifiedName] = struct{}{}
			}
			continue
		}

		kinds := make(map[string]struct{}, len(service.SelectedKinds()))
		for _, kind := range service.SelectedKinds() {
			kinds[kind] = struct{}{}
		}
		for _, target := range registryTargets {
			if validatorTargetService(target) != service.Service {
				continue
			}
			if _, ok := kinds[validatorTargetKind(target)]; !ok {
				continue
			}
			for _, mapping := range target.SDKMappings {
				selected[mapping.SDKStruct] = struct{}{}
			}
		}
	}

	return selected
}

func validatorTargetService(target apispec.Target) string {
	if len(target.SDKMappings) > 0 {
		parts := strings.SplitN(target.SDKMappings[0].SDKStruct, ".", 2)
		if len(parts) == 2 {
			return parts[0]
		}
	}
	return path.Base(path.Dir(target.SpecType.PkgPath()))
}

func validatorTargetKind(target apispec.Target) string {
	return strings.TrimSuffix(target.SpecType.Name(), "Spec")
}

func filterControllerReportByStructs(report diff.Report, allowed map[string]struct{}) diff.Report {
	filtered := diff.Report{Structs: make([]diff.StructReport, 0, len(report.Structs))}
	for _, structReport := range report.Structs {
		if _, ok := allowed[structReport.StructType]; ok {
			filtered.Structs = append(filtered.Structs, structReport)
		}
	}
	return filtered
}

func filterAPIReportBySelectedServices(report apispec.Report, services []generator.ServiceConfig) apispec.Report {
	selectedServices := make(map[string]struct{}, len(services))
	selectedKinds := make(map[string]map[string]struct{}, len(services))
	for _, service := range services {
		if !service.HasSelectedKinds() {
			selectedServices[service.Service] = struct{}{}
			continue
		}
		kinds := make(map[string]struct{}, len(service.SelectedKinds()))
		for _, kind := range service.SelectedKinds() {
			kinds[kind] = struct{}{}
		}
		selectedKinds[service.Service] = kinds
	}

	filtered := apispec.Report{Structs: make([]apispec.StructReport, 0, len(report.Structs))}
	for _, structReport := range report.Structs {
		if _, ok := selectedServices[structReport.Service]; ok {
			filtered.Structs = append(filtered.Structs, structReport)
			continue
		}
		kinds, ok := selectedKinds[structReport.Service]
		if !ok {
			continue
		}
		if _, ok := kinds[structReport.Spec]; ok {
			filtered.Structs = append(filtered.Structs, structReport)
		}
	}
	return filtered
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/oracle/oci-service-operator/internal/formal"
	"sigs.k8s.io/yaml"
)

// Config is the source-of-truth configuration consumed by the generator.
type Config struct {
	SchemaVersion       string                    `yaml:"schemaVersion"`
	Domain              string                    `yaml:"domain"`
	DefaultVersion      string                    `yaml:"defaultVersion"`
	GeneratorEntrypoint string                    `yaml:"generatorEntrypoint"`
	PackageProfiles     map[string]PackageProfile `yaml:"packageProfiles"`
	Services            []ServiceConfig           `yaml:"services"`
	configDir           string                    `yaml:"-"`
}

const (
	PackageProfileControllerBacked = "controller-backed"
	PackageProfileCRDOnly          = "crd-only"

	GenerationStrategyNone      = "none"
	GenerationStrategyManual    = "manual"
	GenerationStrategyGenerated = "generated"
)

// PackageProfile describes how generated service outputs integrate with packaging.
type PackageProfile struct {
	Description string `yaml:"description"`
}

// GenerationConfig defines controller/service-manager/runtime rollout for a service.
type GenerationConfig struct {
	Controller     GenerationSurfaceConfig      `yaml:"controller,omitempty"`
	ServiceManager GenerationSurfaceConfig      `yaml:"serviceManager,omitempty"`
	Registration   GenerationSurfaceConfig      `yaml:"registration,omitempty"`
	Webhooks       GenerationSurfaceConfig      `yaml:"webhooks,omitempty"`
	Resources      []ResourceGenerationOverride `yaml:"resources,omitempty"`
}

// GenerationSurfaceConfig tracks one generator-owned surface rollout.
type GenerationSurfaceConfig struct {
	Strategy string `yaml:"strategy,omitempty"`
}

// ResourceGenerationOverride captures per-kind rollout and override metadata.
type ResourceGenerationOverride struct {
	Kind           string                           `yaml:"kind"`
	FormalSpec     string                           `yaml:"formalSpec,omitempty"`
	Controller     ControllerGenerationOverride     `yaml:"controller,omitempty"`
	ServiceManager ServiceManagerGenerationOverride `yaml:"serviceManager,omitempty"`
	Webhooks       GenerationSurfaceConfig          `yaml:"webhooks,omitempty"`
}

// ControllerGenerationOverride captures per-kind controller-specific settings.
type ControllerGenerationOverride struct {
	Strategy                string   `yaml:"strategy,omitempty"`
	MaxConcurrentReconciles int      `yaml:"maxConcurrentReconciles,omitempty"`
	ExtraRBACMarkers        []string `yaml:"extraRBACMarkers,omitempty"`
}

// ServiceManagerGenerationOverride captures per-kind service-manager settings.
type ServiceManagerGenerationOverride struct {
	Strategy    string `yaml:"strategy,omitempty"`
	PackagePath string `yaml:"packagePath,omitempty"`
}

// ServiceConfig identifies one OCI SDK service and its OSOK output group.
type ServiceConfig struct {
	Service        string              `yaml:"service"`
	SDKPackage     string              `yaml:"sdkPackage"`
	Group          string              `yaml:"group"`
	Version        string              `yaml:"version"`
	Phase          string              `yaml:"phase"`
	SampleOrder    int                 `yaml:"sampleOrder,omitempty"`
	PackageProfile string              `yaml:"packageProfile"`
	FormalSpec     string              `yaml:"formalSpec,omitempty"`
	ParityFile     string              `yaml:"parityFile,omitempty"`
	Compatibility  CompatibilityConfig `yaml:"compatibility,omitempty"`
	ObservedState  ObservedStateConfig `yaml:"observedState,omitempty"`
	Generation     GenerationConfig    `yaml:"generation,omitempty"`
	Parity         *ParityConfig       `yaml:"-"`
}

// CompatibilityConfig holds explicit backwards-compatibility hints for published kinds.
type CompatibilityConfig struct {
	ExistingKinds []string `yaml:"existingKinds,omitempty"`
}

// ObservedStateConfig tunes how read-model fields are synthesized into status types.
type ObservedStateConfig struct {
	SDKAliases map[string][]string `yaml:"sdkAliases,omitempty"`
}

// LoadConfig reads and validates the generator config file.
func LoadConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read generator config %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.UnmarshalStrict(content, &cfg); err != nil {
		return nil, fmt.Errorf("decode generator config %q: %w", path, err)
	}
	cfg.configDir = filepath.Dir(path)
	if err := cfg.loadParity(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("load generator config %q parity inputs: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate generator config %q: %w", path, err)
	}

	return &cfg, nil
}

func (c *Config) loadParity(baseDir string) error {
	for i := range c.Services {
		parityFile := strings.TrimSpace(c.Services[i].ParityFile)
		if parityFile == "" {
			continue
		}

		parity, err := loadParityConfig(ResolveParityFile(baseDir, parityFile))
		if err != nil {
			return fmt.Errorf("service %q: %w", c.Services[i].Service, err)
		}
		c.Services[i].Parity = parity
	}

	return nil
}

// Validate ensures the config is coherent before generation begins.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.SchemaVersion) == "" {
		return fmt.Errorf("schemaVersion is required")
	}
	if strings.TrimSpace(c.Domain) == "" {
		return fmt.Errorf("domain is required")
	}
	if strings.TrimSpace(c.DefaultVersion) == "" {
		return fmt.Errorf("defaultVersion is required")
	}
	if len(c.Services) == 0 {
		return fmt.Errorf("at least one service is required")
	}

	servicesByName := make(map[string]struct{}, len(c.Services))
	groupsByName := make(map[string]struct{}, len(c.Services))
	for _, service := range c.Services {
		if strings.TrimSpace(service.Service) == "" {
			return fmt.Errorf("service name is required")
		}
		if strings.TrimSpace(service.SDKPackage) == "" {
			return fmt.Errorf("service %q is missing sdkPackage", service.Service)
		}
		if strings.TrimSpace(service.Group) == "" {
			return fmt.Errorf("service %q is missing group", service.Service)
		}
		if strings.TrimSpace(service.PackageProfile) == "" {
			return fmt.Errorf("service %q is missing packageProfile", service.Service)
		}
		if _, ok := c.PackageProfiles[service.PackageProfile]; !ok {
			return fmt.Errorf("service %q references unknown packageProfile %q", service.Service, service.PackageProfile)
		}
		if err := validateFormalSpec(fmt.Sprintf("service %q formalSpec", service.Service), service.FormalSpec); err != nil {
			return err
		}
		if err := service.Generation.Validate(service.Service); err != nil {
			return err
		}
		for rawName, aliases := range service.ObservedState.SDKAliases {
			if strings.TrimSpace(rawName) == "" {
				return fmt.Errorf("service %q observedState sdkAliases contains a blank resource name", service.Service)
			}
			for _, alias := range aliases {
				if strings.TrimSpace(alias) == "" {
					return fmt.Errorf("service %q observedState sdkAliases[%q] contains a blank SDK alias", service.Service, rawName)
				}
			}
		}
		if _, exists := servicesByName[service.Service]; exists {
			return fmt.Errorf("duplicate service %q", service.Service)
		}
		if _, exists := groupsByName[service.Group]; exists {
			return fmt.Errorf("duplicate group %q", service.Group)
		}
		servicesByName[service.Service] = struct{}{}
		groupsByName[service.Group] = struct{}{}
	}

	return nil
}

// Validate ensures runtime rollout metadata is coherent before generation begins.
func (g GenerationConfig) Validate(serviceName string) error {
	if err := validateGenerationStrategy(
		fmt.Sprintf("service %q generation.controller.strategy", serviceName),
		g.Controller.Strategy,
	); err != nil {
		return err
	}
	if err := validateGenerationStrategy(
		fmt.Sprintf("service %q generation.serviceManager.strategy", serviceName),
		g.ServiceManager.Strategy,
	); err != nil {
		return err
	}
	if err := validateGenerationStrategy(
		fmt.Sprintf("service %q generation.registration.strategy", serviceName),
		g.Registration.Strategy,
	); err != nil {
		return err
	}
	if err := validateWebhookStrategy(
		fmt.Sprintf("service %q generation.webhooks.strategy", serviceName),
		g.Webhooks.Strategy,
	); err != nil {
		return err
	}

	resourceKinds := make(map[string]struct{}, len(g.Resources))
	for _, resource := range g.Resources {
		kind := strings.TrimSpace(resource.Kind)
		if kind == "" {
			return fmt.Errorf("service %q generation.resources kind is required", serviceName)
		}
		if _, exists := resourceKinds[kind]; exists {
			return fmt.Errorf("service %q generation.resources contains duplicate kind %q", serviceName, kind)
		}
		resourceKinds[kind] = struct{}{}

		if err := validateGenerationStrategy(
			fmt.Sprintf("service %q generation.resources[%q].controller.strategy", serviceName, kind),
			resource.Controller.Strategy,
		); err != nil {
			return err
		}
		if err := validateGenerationStrategy(
			fmt.Sprintf("service %q generation.resources[%q].serviceManager.strategy", serviceName, kind),
			resource.ServiceManager.Strategy,
		); err != nil {
			return err
		}
		if err := validateWebhookStrategy(
			fmt.Sprintf("service %q generation.resources[%q].webhooks.strategy", serviceName, kind),
			resource.Webhooks.Strategy,
		); err != nil {
			return err
		}
		if err := validateFormalSpec(
			fmt.Sprintf("service %q generation.resources[%q].formalSpec", serviceName, kind),
			resource.FormalSpec,
		); err != nil {
			return err
		}
		if resource.Controller.MaxConcurrentReconciles < 0 {
			return fmt.Errorf(
				"service %q generation.resources[%q].controller.maxConcurrentReconciles must be >= 0",
				serviceName,
				kind,
			)
		}
		for _, marker := range resource.Controller.ExtraRBACMarkers {
			if strings.TrimSpace(marker) == "" {
				return fmt.Errorf(
					"service %q generation.resources[%q].controller.extraRBACMarkers contains a blank marker",
					serviceName,
					kind,
				)
			}
		}
		if packagePath := strings.TrimSpace(resource.ServiceManager.PackagePath); packagePath != "" {
			cleaned := path.Clean(packagePath)
			if strings.HasPrefix(packagePath, "/") || cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned != packagePath {
				return fmt.Errorf(
					"service %q generation.resources[%q].serviceManager.packagePath must be a clean relative path beneath pkg/servicemanager",
					serviceName,
					kind,
				)
			}
		}
		if !resource.hasOverrides() {
			return fmt.Errorf(
				"service %q generation.resources[%q] does not override any runtime output",
				serviceName,
				kind,
			)
		}
	}

	return nil
}

func validateGenerationStrategy(field string, strategy string) error {
	switch strings.TrimSpace(strategy) {
	case "", GenerationStrategyNone, GenerationStrategyManual, GenerationStrategyGenerated:
		return nil
	default:
		return fmt.Errorf(
			"%s %q must be one of %q, %q, or %q",
			field,
			strategy,
			GenerationStrategyNone,
			GenerationStrategyManual,
			GenerationStrategyGenerated,
		)
	}
}

func validateWebhookStrategy(field string, strategy string) error {
	switch strings.TrimSpace(strategy) {
	case "", GenerationStrategyNone, GenerationStrategyManual:
		return nil
	default:
		return fmt.Errorf(
			"%s %q must be one of %q or %q",
			field,
			strategy,
			GenerationStrategyNone,
			GenerationStrategyManual,
		)
	}
}

func (r ResourceGenerationOverride) hasOverrides() bool {
	return strings.TrimSpace(r.FormalSpec) != "" ||
		r.Controller.hasOverrides() ||
		r.ServiceManager.hasOverrides() ||
		strings.TrimSpace(r.Webhooks.Strategy) != ""
}

func (c ControllerGenerationOverride) hasOverrides() bool {
	return strings.TrimSpace(c.Strategy) != "" || c.MaxConcurrentReconciles != 0 || len(c.ExtraRBACMarkers) > 0
}

func (s ServiceManagerGenerationOverride) hasOverrides() bool {
	return strings.TrimSpace(s.Strategy) != "" || strings.TrimSpace(s.PackagePath) != ""
}

func validateFormalSpec(field string, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.ContainsAny(value, `/\`) {
		return fmt.Errorf("%s %q must be a single formal slug, not a path", field, value)
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || cleaned != value {
		return fmt.Errorf("%s %q must be a clean formal slug", field, value)
	}
	return nil
}

// SelectServices resolves the requested services from the config.
func (c *Config) SelectServices(serviceName string, all bool) ([]ServiceConfig, error) {
	if all && strings.TrimSpace(serviceName) != "" {
		return nil, fmt.Errorf("use either --all or --service, not both")
	}
	if !all && strings.TrimSpace(serviceName) == "" {
		return nil, fmt.Errorf("either --all or --service must be set")
	}
	if all {
		selected := make([]ServiceConfig, len(c.Services))
		copy(selected, c.Services)
		return selected, nil
	}

	for _, service := range c.Services {
		if service.Service == serviceName {
			return []ServiceConfig{service}, nil
		}
	}

	return nil, fmt.Errorf("service %q was not found in the generator config", serviceName)
}

// FormalRoot returns the repo-local formal catalog root implied by the config location.
func (c *Config) FormalRoot() string {
	if strings.TrimSpace(c.configDir) == "" {
		return ""
	}
	return filepath.Clean(filepath.Join(c.configDir, "..", "..", "..", "formal"))
}

// HasFormalSpecs reports whether the config includes any formalSpec references.
func (c *Config) HasFormalSpecs() bool {
	if c == nil {
		return false
	}
	for _, service := range c.Services {
		if service.HasFormalSpecs() {
			return true
		}
	}
	return false
}

// VerifyFormalInputs validates the repo-local formal catalog when the config references formal specs.
func (c *Config) VerifyFormalInputs() error {
	if !c.HasFormalSpecs() {
		return nil
	}

	formalRoot := c.FormalRoot()
	if strings.TrimSpace(formalRoot) == "" {
		return fmt.Errorf("formal root is unknown for configs with formalSpec references")
	}
	if _, err := formal.Verify(formalRoot); err != nil {
		return fmt.Errorf("verify formal inputs %q: %w", formalRoot, err)
	}
	return nil
}

// VersionOrDefault returns the configured version or the config default.
func (s ServiceConfig) VersionOrDefault(defaultVersion string) string {
	if strings.TrimSpace(s.Version) != "" {
		return s.Version
	}
	return defaultVersion
}

// GroupDNSName returns the Kubernetes API group DNS name for the service.
func (s ServiceConfig) GroupDNSName(domain string) string {
	return fmt.Sprintf("%s.%s", s.Group, domain)
}

// IsControllerBacked reports whether the service expects shared-manager controller assets.
func (s ServiceConfig) IsControllerBacked() bool {
	return s.PackageProfile == PackageProfileControllerBacked
}

// ControllerGenerationStrategy returns the controller rollout strategy for the service.
func (s ServiceConfig) ControllerGenerationStrategy() string {
	return generationStrategyOrDefault(s.Generation.Controller.Strategy, GenerationStrategyNone)
}

// ControllerGenerationStrategyFor returns the resource-specific controller rollout strategy.
func (s ServiceConfig) ControllerGenerationStrategyFor(kind string) string {
	if override, ok := s.resourceGenerationOverride(kind); ok {
		return generationStrategyOrDefault(override.Controller.Strategy, s.ControllerGenerationStrategy())
	}
	return s.ControllerGenerationStrategy()
}

// ControllerGenerationConfigFor resolves one resource's effective controller generation config.
func (s ServiceConfig) ControllerGenerationConfigFor(kind string) ControllerGenerationOverride {
	config := ControllerGenerationOverride{
		Strategy: s.ControllerGenerationStrategyFor(kind),
	}

	if override, ok := s.resourceGenerationOverride(kind); ok {
		config.MaxConcurrentReconciles = override.Controller.MaxConcurrentReconciles
		config.ExtraRBACMarkers = append([]string(nil), override.Controller.ExtraRBACMarkers...)
	}

	return config
}

// ServiceManagerGenerationStrategy returns the service-manager rollout strategy for the service.
func (s ServiceConfig) ServiceManagerGenerationStrategy() string {
	return generationStrategyOrDefault(s.Generation.ServiceManager.Strategy, GenerationStrategyNone)
}

// ServiceManagerGenerationStrategyFor returns the resource-specific service-manager rollout strategy.
func (s ServiceConfig) ServiceManagerGenerationStrategyFor(kind string) string {
	if override, ok := s.resourceGenerationOverride(kind); ok {
		return generationStrategyOrDefault(override.ServiceManager.Strategy, s.ServiceManagerGenerationStrategy())
	}
	return s.ServiceManagerGenerationStrategy()
}

// ServiceManagerPackagePathFor resolves the package path for one generated service-manager resource.
func (s ServiceConfig) ServiceManagerPackagePathFor(kind string, fileStem string) string {
	if override, ok := s.resourceGenerationOverride(kind); ok && strings.TrimSpace(override.ServiceManager.PackagePath) != "" {
		return override.ServiceManager.PackagePath
	}
	return path.Join(s.Group, fileStem)
}

// FormalSpecFor resolves the effective formal slug for one generated resource.
func (s ServiceConfig) FormalSpecFor(kind string) string {
	if override, ok := s.resourceGenerationOverride(kind); ok && strings.TrimSpace(override.FormalSpec) != "" {
		return strings.TrimSpace(override.FormalSpec)
	}
	return strings.TrimSpace(s.FormalSpec)
}

// HasFormalSpecs reports whether the service or any resource override uses formal specs.
func (s ServiceConfig) HasFormalSpecs() bool {
	if strings.TrimSpace(s.FormalSpec) != "" {
		return true
	}
	for _, resource := range s.Generation.Resources {
		if strings.TrimSpace(resource.FormalSpec) != "" {
			return true
		}
	}
	return false
}

// RegistrationGenerationStrategy returns the runtime-registration rollout strategy for the service.
func (s ServiceConfig) RegistrationGenerationStrategy() string {
	return generationStrategyOrDefault(s.Generation.Registration.Strategy, GenerationStrategyNone)
}

// WebhookGenerationStrategy returns the webhook ownership strategy for the service.
func (s ServiceConfig) WebhookGenerationStrategy() string {
	return generationStrategyOrDefault(s.Generation.Webhooks.Strategy, GenerationStrategyManual)
}

func generationStrategyOrDefault(strategy string, defaultStrategy string) string {
	strategy = strings.TrimSpace(strategy)
	if strategy == "" {
		return defaultStrategy
	}
	return strategy
}

func (s ServiceConfig) resourceGenerationOverride(kind string) (ResourceGenerationOverride, bool) {
	for _, override := range s.Generation.Resources {
		if override.Kind == kind {
			return override, true
		}
	}
	return ResourceGenerationOverride{}, false
}

// ObservedStateStructCandidates returns the read-model structs that should feed status synthesis.
func (s ServiceConfig) ObservedStateStructCandidates(rawName string) []string {
	rawName = strings.TrimSpace(rawName)
	if rawName == "" {
		return nil
	}

	candidates := appendUniqueStrings(nil, rawName, rawName+"Summary")
	indexByKey := make(map[string]int, len(candidates))
	for i, candidate := range candidates {
		indexByKey[normalizedTypeKey(candidate)] = i
	}
	for _, alias := range s.ObservedState.SDKAliases[rawName] {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		key := normalizedTypeKey(alias)
		if index, ok := indexByKey[key]; ok {
			candidates[index] = alias
			continue
		}
		indexByKey[key] = len(candidates)
		candidates = append(candidates, alias)
	}

	return candidates
}

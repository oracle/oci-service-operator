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

	SelectionModeAll      = "all"
	SelectionModeExplicit = "explicit"
)

// PackageProfile describes how generated service outputs integrate with packaging.
type PackageProfile struct {
	Description string `yaml:"description"`
}

// PackageConfig describes service-scoped package overlay details.
type PackageConfig struct {
	ExtraResources []string `yaml:"extraResources,omitempty"`
}

// SelectionConfig declares whether a service participates in the default active surface.
type SelectionConfig struct {
	Enabled      *bool    `yaml:"enabled"`
	Mode         string   `yaml:"mode"`
	IncludeKinds []string `yaml:"includeKinds,omitempty"`
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
	SpecFields     []FieldOverride                  `yaml:"specFields,omitempty"`
	StatusFields   []FieldOverride                  `yaml:"statusFields,omitempty"`
	Sample         SampleOverride                   `yaml:"sample,omitempty"`
}

// ControllerGenerationOverride captures per-kind controller-specific settings.
type ControllerGenerationOverride struct {
	Strategy                string   `yaml:"strategy,omitempty"`
	MaxConcurrentReconciles int      `yaml:"maxConcurrentReconciles,omitempty"`
	ExtraRBACMarkers        []string `yaml:"extraRBACMarkers,omitempty"`
}

// ServiceManagerGenerationOverride captures per-kind service-manager settings.
type ServiceManagerGenerationOverride struct {
	Strategy              string `yaml:"strategy,omitempty"`
	PackagePath           string `yaml:"packagePath,omitempty"`
	NeedsCredentialClient bool   `yaml:"needsCredentialClient,omitempty"`
}

// FieldOverride captures one generated resource field override sourced from services.yaml.
type FieldOverride struct {
	Name     string   `yaml:"name,omitempty"`
	Type     string   `yaml:"type,omitempty"`
	Tag      string   `yaml:"tag,omitempty"`
	Comments []string `yaml:"comments,omitempty"`
	Markers  []string `yaml:"markers,omitempty"`
}

// SampleOverride captures a generated sample override sourced from services.yaml.
type SampleOverride struct {
	Body         string `yaml:"body,omitempty"`
	MetadataName string `yaml:"metadataName,omitempty"`
	Spec         string `yaml:"spec,omitempty"`
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
	Package        PackageConfig       `yaml:"package,omitempty"`
	Selection      SelectionConfig     `yaml:"selection"`
	FormalSpec     string              `yaml:"formalSpec,omitempty"`
	ObservedState  ObservedStateConfig `yaml:"observedState,omitempty"`
	Generation     GenerationConfig    `yaml:"generation,omitempty"`
	selectedKinds  []string            `yaml:"-"`
}

// ObservedStateConfig tunes how read-model fields are synthesized into status types.
type ObservedStateConfig struct {
	SDKAliases         map[string][]string `yaml:"sdkAliases,omitempty"`
	ExcludedFieldPaths map[string][]string `yaml:"excludedFieldPaths,omitempty"`
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
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate generator config %q: %w", path, err)
	}

	return &cfg, nil
}

// Validate ensures the config is coherent before generation begins.
//
//nolint:gocognit,gocyclo // Validation intentionally collects all config problems in one pass.
func (c *Config) Validate() error {
	if err := c.validateMetadata(); err != nil {
		return err
	}
	servicesByName := make(map[string]struct{}, len(c.Services))
	groupsByName := make(map[string]struct{}, len(c.Services))
	for _, service := range c.Services {
		if err := c.validateService(service, servicesByName, groupsByName); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) validateMetadata() error {
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
	return nil
}

func (c *Config) validateService(
	service ServiceConfig,
	servicesByName map[string]struct{},
	groupsByName map[string]struct{},
) error {
	if err := validateServiceIdentity(service, c.PackageProfiles); err != nil {
		return err
	}
	if err := validatePackageExtraResources(service); err != nil {
		return err
	}
	if err := service.Selection.Validate(service.Service); err != nil {
		return err
	}
	if err := validateFormalSpec(fmt.Sprintf("service %q formalSpec", service.Service), service.FormalSpec); err != nil {
		return err
	}
	if err := service.Generation.Validate(service.Service); err != nil {
		return err
	}
	if err := validateObservedStateConfig(service); err != nil {
		return err
	}
	if err := validateUniqueServiceKeys(service, servicesByName, groupsByName); err != nil {
		return err
	}

	servicesByName[service.Service] = struct{}{}
	groupsByName[service.Group] = struct{}{}
	return nil
}

func validateServiceIdentity(service ServiceConfig, packageProfiles map[string]PackageProfile) error {
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
	if _, ok := packageProfiles[service.PackageProfile]; !ok {
		return fmt.Errorf("service %q references unknown packageProfile %q", service.Service, service.PackageProfile)
	}
	return nil
}

func validatePackageExtraResources(service ServiceConfig) error {
	for _, extraResource := range service.Package.ExtraResources {
		if strings.TrimSpace(extraResource) == "" {
			return fmt.Errorf("service %q package.extraResources contains a blank path", service.Service)
		}
	}
	return nil
}

func validateObservedStateConfig(service ServiceConfig) error {
	if err := validateObservedStateAliases(service); err != nil {
		return err
	}
	return validateObservedStateExcludedFieldPaths(service)
}

func validateObservedStateAliases(service ServiceConfig) error {
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
	return nil
}

func validateObservedStateExcludedFieldPaths(service ServiceConfig) error {
	for rawName, paths := range service.ObservedState.ExcludedFieldPaths {
		if strings.TrimSpace(rawName) == "" {
			return fmt.Errorf("service %q observedState excludedFieldPaths contains a blank resource name", service.Service)
		}
		for _, fieldPath := range paths {
			if _, err := normalizeObservedStateFieldPath(fieldPath); err != nil {
				return fmt.Errorf(
					"service %q observedState excludedFieldPaths[%q] %w",
					service.Service,
					rawName,
					err,
				)
			}
		}
	}
	return nil
}

func validateUniqueServiceKeys(
	service ServiceConfig,
	servicesByName map[string]struct{},
	groupsByName map[string]struct{},
) error {
	if _, exists := servicesByName[service.Service]; exists {
		return fmt.Errorf("duplicate service %q", service.Service)
	}
	if _, exists := groupsByName[service.Group]; exists {
		return fmt.Errorf("duplicate group %q", service.Group)
	}
	return nil
}

// Validate ensures service selection metadata is coherent before generation begins.
func (s SelectionConfig) Validate(serviceName string) error {
	if s.Enabled == nil {
		return fmt.Errorf("service %q selection.enabled is required", serviceName)
	}

	mode := strings.TrimSpace(s.Mode)
	switch mode {
	case SelectionModeAll:
		if len(s.IncludeKinds) > 0 {
			return fmt.Errorf(
				"service %q selection.includeKinds must be empty when selection.mode is %q",
				serviceName,
				SelectionModeAll,
			)
		}
	case SelectionModeExplicit:
		if len(s.IncludeKinds) == 0 {
			return fmt.Errorf(
				"service %q selection.includeKinds must list at least one kind when selection.mode is %q",
				serviceName,
				SelectionModeExplicit,
			)
		}

		seen := make(map[string]struct{}, len(s.IncludeKinds))
		for index, rawKind := range s.IncludeKinds {
			kind := strings.TrimSpace(rawKind)
			if kind == "" {
				return fmt.Errorf("service %q selection.includeKinds[%d] must not be blank", serviceName, index)
			}
			if _, exists := seen[kind]; exists {
				return fmt.Errorf("service %q selection.includeKinds contains duplicate kind %q", serviceName, kind)
			}
			seen[kind] = struct{}{}
		}
	default:
		return fmt.Errorf(
			"service %q selection.mode %q must be one of %q or %q",
			serviceName,
			s.Mode,
			SelectionModeAll,
			SelectionModeExplicit,
		)
	}

	return nil
}

// Validate ensures runtime rollout metadata is coherent before generation begins.
//
//nolint:gocognit,gocyclo // Generation validation checks multiple rollout surfaces and overrides together.
func (g GenerationConfig) Validate(serviceName string) error {
	if err := validateGenerationSurfaceStrategies(serviceName, g); err != nil {
		return err
	}
	return validateResourceGenerationOverrides(serviceName, g.Resources)
}

func validateGenerationSurfaceStrategies(serviceName string, g GenerationConfig) error {
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
	return validateWebhookStrategy(
		fmt.Sprintf("service %q generation.webhooks.strategy", serviceName),
		g.Webhooks.Strategy,
	)
}

//nolint:gocognit,gocyclo // Validation walks all supported resource override surfaces and returns the first precise error.
func validateResourceGenerationOverrides(serviceName string, resources []ResourceGenerationOverride) error {
	resourceKinds := make(map[string]struct{}, len(resources))
	for _, resource := range resources {
		kind, err := validateResourceGenerationKind(serviceName, resource, resourceKinds)
		if err != nil {
			return err
		}
		if err := validateResourceGenerationStrategies(serviceName, kind, resource); err != nil {
			return err
		}
		if err := validateFormalSpec(
			fmt.Sprintf("service %q generation.resources[%q].formalSpec", serviceName, kind),
			resource.FormalSpec,
		); err != nil {
			return err
		}
		if err := validateFieldOverrides(
			fmt.Sprintf("service %q generation.resources[%q].specFields", serviceName, kind),
			resource.SpecFields,
		); err != nil {
			return err
		}
		if err := validateFieldOverrides(
			fmt.Sprintf("service %q generation.resources[%q].statusFields", serviceName, kind),
			resource.StatusFields,
		); err != nil {
			return err
		}
		if err := validateSampleOverride(
			fmt.Sprintf("service %q generation.resources[%q].sample", serviceName, kind),
			resource.Sample,
		); err != nil {
			return err
		}
		if err := validateControllerGenerationOverride(serviceName, kind, resource.Controller); err != nil {
			return err
		}
		if err := validateServiceManagerGenerationOverride(serviceName, kind, resource.ServiceManager); err != nil {
			return err
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

func validateResourceGenerationKind(
	serviceName string,
	resource ResourceGenerationOverride,
	resourceKinds map[string]struct{},
) (string, error) {
	kind := strings.TrimSpace(resource.Kind)
	if kind == "" {
		return "", fmt.Errorf("service %q generation.resources kind is required", serviceName)
	}
	if _, exists := resourceKinds[kind]; exists {
		return "", fmt.Errorf("service %q generation.resources contains duplicate kind %q", serviceName, kind)
	}
	resourceKinds[kind] = struct{}{}
	return kind, nil
}

func validateResourceGenerationStrategies(serviceName, kind string, resource ResourceGenerationOverride) error {
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
	return validateWebhookStrategy(
		fmt.Sprintf("service %q generation.resources[%q].webhooks.strategy", serviceName, kind),
		resource.Webhooks.Strategy,
	)
}

func validateControllerGenerationOverride(serviceName, kind string, controller ControllerGenerationOverride) error {
	if controller.MaxConcurrentReconciles < 0 {
		return fmt.Errorf(
			"service %q generation.resources[%q].controller.maxConcurrentReconciles must be >= 0",
			serviceName,
			kind,
		)
	}
	for _, marker := range controller.ExtraRBACMarkers {
		if strings.TrimSpace(marker) == "" {
			return fmt.Errorf(
				"service %q generation.resources[%q].controller.extraRBACMarkers contains a blank marker",
				serviceName,
				kind,
			)
		}
	}
	return nil
}

func validateServiceManagerGenerationOverride(serviceName, kind string, serviceManager ServiceManagerGenerationOverride) error {
	packagePath := strings.TrimSpace(serviceManager.PackagePath)
	if packagePath == "" {
		return nil
	}

	cleaned := path.Clean(packagePath)
	if strings.HasPrefix(packagePath, "/") || cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned != packagePath {
		return fmt.Errorf(
			"service %q generation.resources[%q].serviceManager.packagePath must be a clean relative path beneath pkg/servicemanager",
			serviceName,
			kind,
		)
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

func validateFieldOverrides(field string, overrides []FieldOverride) error {
	for index, override := range overrides {
		overrideField := fmt.Sprintf("%s[%d]", field, index)
		if strings.TrimSpace(override.Name) == "" {
			return fmt.Errorf("%s.name is required", overrideField)
		}
		if strings.TrimSpace(override.Type) == "" {
			return fmt.Errorf("%s.type is required", overrideField)
		}
		if strings.TrimSpace(override.Tag) == "" {
			return fmt.Errorf("%s.tag is required", overrideField)
		}
	}
	return nil
}

func validateSampleOverride(field string, sample SampleOverride) error {
	if strings.TrimSpace(sample.Body) != "" && (strings.TrimSpace(sample.MetadataName) != "" || strings.TrimSpace(sample.Spec) != "") {
		return fmt.Errorf("%s.body cannot be combined with metadataName or spec", field)
	}
	return nil
}

func (r ResourceGenerationOverride) hasOverrides() bool {
	return strings.TrimSpace(r.FormalSpec) != "" ||
		r.Controller.hasOverrides() ||
		r.ServiceManager.hasOverrides() ||
		strings.TrimSpace(r.Webhooks.Strategy) != "" ||
		len(r.SpecFields) > 0 ||
		len(r.StatusFields) > 0 ||
		r.Sample.hasOverride()
}

func (c ControllerGenerationOverride) hasOverrides() bool {
	return strings.TrimSpace(c.Strategy) != "" || c.MaxConcurrentReconciles != 0 || len(c.ExtraRBACMarkers) > 0
}

func (s ServiceManagerGenerationOverride) hasOverrides() bool {
	return strings.TrimSpace(s.Strategy) != "" || strings.TrimSpace(s.PackagePath) != "" || s.NeedsCredentialClient
}

func (s SampleOverride) hasOverride() bool {
	return strings.TrimSpace(s.Body) != "" || strings.TrimSpace(s.MetadataName) != "" || strings.TrimSpace(s.Spec) != ""
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
// `--all` returns only the default-active services and applies any explicit
// default kind subset before package-model construction. `--service` returns
// the named service without applying default-active filtering so backlog and
// disabled services remain addressable explicitly.
func (c *Config) SelectServices(serviceName string, all bool) ([]ServiceConfig, error) {
	if all && strings.TrimSpace(serviceName) != "" {
		return nil, fmt.Errorf("use either --all or --service, not both")
	}
	if !all && strings.TrimSpace(serviceName) == "" {
		return nil, fmt.Errorf("either --all or --service must be set")
	}
	if all {
		defaultActive := c.DefaultActiveServices()
		selected := make([]ServiceConfig, 0, len(defaultActive))
		for _, service := range defaultActive {
			selected = append(selected, service.defaultSelectedSurface())
		}
		return selected, nil
	}

	for _, service := range c.Services {
		if service.Service == serviceName {
			return []ServiceConfig{service.withSelectedKinds(nil)}, nil
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
	return c.verifyFormalRoot()
}

// VerifyFormalInputsForServices validates the repo-local formal catalog only
// when the selected service surface references formal specs.
func (c *Config) VerifyFormalInputsForServices(services []ServiceConfig) error {
	for _, service := range services {
		if !service.HasSelectedFormalSpecs() {
			continue
		}
		return c.verifyFormalRoot()
	}
	return nil
}

func (c *Config) verifyFormalRoot() error {
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

// IsDefaultActive reports whether the service is part of the default active surface.
func (s ServiceConfig) IsDefaultActive() bool {
	return s.Selection.Enabled != nil && *s.Selection.Enabled
}

// DefaultSelectionMode returns the validated default selection mode for the service.
func (s ServiceConfig) DefaultSelectionMode() string {
	return strings.TrimSpace(s.Selection.Mode)
}

// DefaultIncludeKinds returns the validated explicit default kind subset for the service.
func (s ServiceConfig) DefaultIncludeKinds() []string {
	if len(s.Selection.IncludeKinds) == 0 {
		return nil
	}

	kinds := make([]string, 0, len(s.Selection.IncludeKinds))
	for _, rawKind := range s.Selection.IncludeKinds {
		kind := strings.TrimSpace(rawKind)
		if kind == "" {
			continue
		}
		kinds = append(kinds, kind)
	}

	if len(kinds) == 0 {
		return nil
	}
	return kinds
}

// SelectedKinds returns the effective kind subset applied to this service for
// the current generator run. A nil slice means the full discovered service
// surface is in scope.
func (s ServiceConfig) SelectedKinds() []string {
	if len(s.selectedKinds) == 0 {
		return nil
	}
	return append([]string(nil), s.selectedKinds...)
}

// HasSelectedKinds reports whether the current generator run narrowed this
// service to an explicit kind subset.
func (s ServiceConfig) HasSelectedKinds() bool {
	return len(s.selectedKinds) > 0
}

// DefaultActiveServices returns the services marked active in the default generator surface.
func (c *Config) DefaultActiveServices() []ServiceConfig {
	if c == nil {
		return nil
	}

	selected := make([]ServiceConfig, 0, len(c.Services))
	for _, service := range c.Services {
		if service.IsDefaultActive() {
			selected = append(selected, service)
		}
	}

	return selected
}

func (s ServiceConfig) defaultSelectedSurface() ServiceConfig {
	if s.DefaultSelectionMode() != SelectionModeExplicit {
		return s.withSelectedKinds(nil)
	}
	return s.withSelectedKinds(s.DefaultIncludeKinds())
}

func (s ServiceConfig) withSelectedKinds(kinds []string) ServiceConfig {
	selected := s
	if len(kinds) == 0 {
		selected.selectedKinds = nil
		return selected
	}
	selected.selectedKinds = append([]string(nil), kinds...)
	return selected
}

func (s ServiceConfig) includesSelectedKind(kind string) bool {
	if len(s.selectedKinds) == 0 {
		return true
	}
	for _, selectedKind := range s.selectedKinds {
		if selectedKind == kind {
			return true
		}
	}
	return false
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

// ServiceManagerNeedsCredentialClientFor reports whether the generated service client
// should wire the runtime credential client for one resource.
func (s ServiceConfig) ServiceManagerNeedsCredentialClientFor(kind string) bool {
	if override, ok := s.resourceGenerationOverride(kind); ok {
		return override.ServiceManager.NeedsCredentialClient
	}
	return false
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

// HasSelectedFormalSpecs reports whether the effective service surface for the
// current run references any formal specs.
func (s ServiceConfig) HasSelectedFormalSpecs() bool {
	if strings.TrimSpace(s.FormalSpec) != "" {
		return true
	}
	for _, resource := range s.Generation.Resources {
		if !s.includesSelectedKind(resource.Kind) {
			continue
		}
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

// ObservedStateExcludedFieldPaths returns the configured observed-state field paths that must not be published in status.
func (s ServiceConfig) ObservedStateExcludedFieldPaths(rawName string) map[string]struct{} {
	rawName = strings.TrimSpace(rawName)
	if rawName == "" {
		return nil
	}

	configured := s.ObservedState.ExcludedFieldPaths[rawName]
	if len(configured) == 0 {
		return nil
	}

	paths := make(map[string]struct{}, len(configured))
	for _, path := range configured {
		normalized, err := normalizeObservedStateFieldPath(path)
		if err != nil {
			continue
		}
		paths[normalized] = struct{}{}
	}
	if len(paths) == 0 {
		return nil
	}
	return paths
}

func normalizeObservedStateFieldPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("contains a blank field path")
	}

	parts := strings.Split(path, ".")
	return observedStateFieldPathKey(parts)
}

func observedStateFieldPathKey(parts []string) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("contains a blank field path")
	}

	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", fmt.Errorf("contains a blank path segment")
		}
		key := normalizedTypeKey(part)
		if key == "" {
			return "", fmt.Errorf("contains a blank path segment")
		}
		keys = append(keys, key)
	}
	return strings.Join(keys, "."), nil
}

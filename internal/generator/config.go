/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// Config is the source-of-truth configuration consumed by the API generator.
type Config struct {
	SchemaVersion       string                    `yaml:"schemaVersion"`
	Domain              string                    `yaml:"domain"`
	DefaultVersion      string                    `yaml:"defaultVersion"`
	GeneratorEntrypoint string                    `yaml:"generatorEntrypoint"`
	PackageProfiles     map[string]PackageProfile `yaml:"packageProfiles"`
	Services            []ServiceConfig           `yaml:"services"`
}

const (
	PackageProfileControllerBacked = "controller-backed"
	PackageProfileCRDOnly          = "crd-only"
)

// PackageProfile describes how generated service outputs integrate with packaging.
type PackageProfile struct {
	Description string `yaml:"description"`
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
	ParityFile     string              `yaml:"parityFile,omitempty"`
	Compatibility  CompatibilityConfig `yaml:"compatibility,omitempty"`
	ObservedState  ObservedStateConfig `yaml:"observedState,omitempty"`
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

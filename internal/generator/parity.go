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

// ParityConfig captures explicit service-scoped overrides needed to match current OSOK artifacts.
type ParityConfig struct {
	Resources []ParityResource `yaml:"resources"`
	Package   PackageOverride  `yaml:"package"`
}

// ParityResource describes one resource that should be emitted for a parity-reviewed service.
type ParityResource struct {
	SourceResource  string                `yaml:"sourceResource"`
	Kind            string                `yaml:"kind"`
	FileStem        string                `yaml:"fileStem,omitempty"`
	LeadingComments []string              `yaml:"leadingComments,omitempty"`
	SpecComments    []string              `yaml:"specComments,omitempty"`
	HelperTypes     []TypeOverride        `yaml:"helperTypes,omitempty"`
	SpecFields      []FieldOverride       `yaml:"specFields"`
	StatusComments  []string              `yaml:"statusComments,omitempty"`
	StatusFields    []FieldOverride       `yaml:"statusFields,omitempty"`
	PrintColumns    []PrintColumnOverride `yaml:"printColumns,omitempty"`
	ObjectComments  []string              `yaml:"objectComments,omitempty"`
	ListComments    []string              `yaml:"listComments,omitempty"`
	Sample          SampleOverride        `yaml:"sample"`
}

// TypeOverride defines one helper type rendered in a parity types file.
type TypeOverride struct {
	Name     string          `yaml:"name"`
	Comments []string        `yaml:"comments,omitempty"`
	Fields   []FieldOverride `yaml:"fields,omitempty"`
}

// FieldOverride defines one field in a parity-rendered type.
type FieldOverride struct {
	Name     string   `yaml:"name,omitempty"`
	Type     string   `yaml:"type"`
	Tag      string   `yaml:"tag"`
	Comments []string `yaml:"comments,omitempty"`
	Markers  []string `yaml:"markers,omitempty"`
}

// PrintColumnOverride defines one kubebuilder printcolumn marker in parity mode.
type PrintColumnOverride struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	JSONPath    string `yaml:"jsonPath"`
	Description string `yaml:"description,omitempty"`
	Priority    *int   `yaml:"priority,omitempty"`
}

// SampleOverride defines the repo-owned sample YAML body for a parity resource.
type SampleOverride struct {
	Body         string `yaml:"body,omitempty"`
	MetadataName string `yaml:"metadataName"`
	Spec         string `yaml:"spec,omitempty"`
}

// PackageOverride defines parity package overlay details for a service.
type PackageOverride struct {
	ExtraResources []string `yaml:"extraResources,omitempty"`
}

func loadParityConfig(path string) (*ParityConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read parity config %q: %w", path, err)
	}

	var cfg ParityConfig
	if err := yaml.UnmarshalStrict(content, &cfg); err != nil {
		return nil, fmt.Errorf("decode parity config %q: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate parity config %q: %w", path, err)
	}

	return &cfg, nil
}

// ResolveParityFile resolves a parity config path relative to the main generator config file.
func ResolveParityFile(baseDir string, parityFile string) string {
	if filepath.IsAbs(parityFile) {
		return parityFile
	}
	return filepath.Join(baseDir, parityFile)
}

// Validate ensures the parity config is coherent.
func (p *ParityConfig) Validate() error {
	if len(p.Resources) == 0 {
		return fmt.Errorf("at least one parity resource is required")
	}

	for _, resource := range p.Resources {
		if strings.TrimSpace(resource.SourceResource) == "" {
			return fmt.Errorf("parity resource sourceResource is required")
		}
		if strings.TrimSpace(resource.Kind) == "" {
			return fmt.Errorf("parity resource kind is required")
		}
		if len(resource.SpecFields) == 0 {
			return fmt.Errorf("parity resource %q requires at least one spec field", resource.Kind)
		}
		if strings.TrimSpace(resource.Sample.MetadataName) == "" {
			return fmt.Errorf("parity resource %q requires sample.metadataName", resource.Kind)
		}
	}

	return nil
}

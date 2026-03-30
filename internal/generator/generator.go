/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"context"
	"errors"
	"fmt"
)

// Options controls how generator outputs are emitted.
type Options struct {
	OutputRoot                      string
	Overwrite                       bool
	SkipExisting                    bool
	PreserveExistingSpecSurfaceRoot string
}

// ServiceResult describes the outcome for one generated or skipped service.
type ServiceResult struct {
	Service       string
	Group         string
	OutputDir     string
	ResourceCount int
	Reason        string
}

// RunResult summarizes a generation run.
type RunResult struct {
	Generated []ServiceResult
	Skipped   []ServiceResult
}

// Generator orchestrates SDK discovery and package rendering.
type Generator struct {
	discoverer *Discoverer
	renderer   *Renderer
}

// New returns the default generator pipeline.
func New() *Generator {
	return &Generator{
		discoverer: NewDiscoverer(),
		renderer:   NewRenderer(),
	}
}

// Generate renders the requested services into OSOK API packages.
func (g *Generator) Generate(ctx context.Context, cfg *Config, services []ServiceConfig, options Options) (RunResult, error) {
	result := RunResult{}
	generatedPackages := make([]*PackageModel, 0, len(services))
	preservedArtifacts := make(map[string]preservedPackageArtifacts, len(services))
	for _, service := range services {
		pkg, err := g.discoverer.BuildPackageModel(ctx, cfg, service)
		if err != nil {
			return result, fmt.Errorf("build package model for service %q: %w", service.Service, err)
		}
		if err := preservePackageSpecSurfaces(options.PreserveExistingSpecSurfaceRoot, pkg); err != nil {
			return result, fmt.Errorf("preserve existing spec surface for service %q: %w", service.Service, err)
		}
		preserved, err := loadPreservedPackageArtifacts(options.PreserveExistingSpecSurfaceRoot, pkg)
		if err != nil {
			return result, fmt.Errorf("load preserved artifacts for service %q: %w", service.Service, err)
		}

		outputDir, err := g.renderer.RenderPackage(options.OutputRoot, pkg, options.Overwrite)
		if err != nil {
			var existsErr ErrTargetExists
			if errors.As(err, &existsErr) && options.SkipExisting {
				result.Skipped = append(result.Skipped, ServiceResult{
					Service:       service.Service,
					Group:         service.Group,
					OutputDir:     targetOutputDir(options.OutputRoot, pkg),
					ResourceCount: len(pkg.Resources),
					Reason:        err.Error(),
				})
				continue
			}
			return result, fmt.Errorf("render service %q: %w", service.Service, err)
		}
		if err := g.renderer.RenderPackageOutputs(options.OutputRoot, pkg); err != nil {
			return result, fmt.Errorf("render package outputs for service %q: %w", service.Service, err)
		}
		if err := g.renderer.RenderControllers(options.OutputRoot, pkg, options.Overwrite); err != nil {
			return result, fmt.Errorf("render controller outputs for service %q: %w", service.Service, err)
		}
		if err := g.renderer.RenderRegistrations(options.OutputRoot, pkg, options.Overwrite); err != nil {
			return result, fmt.Errorf("render registration outputs for service %q: %w", service.Service, err)
		}
		if err := g.renderer.RenderServiceManagers(options.OutputRoot, pkg, options.Overwrite); err != nil {
			return result, fmt.Errorf("render service-manager outputs for service %q: %w", service.Service, err)
		}
		if err := preserved.applyPackageFiles(options.OutputRoot); err != nil {
			return result, fmt.Errorf("preserve checked-in artifacts for service %q: %w", service.Service, err)
		}

		result.Generated = append(result.Generated, ServiceResult{
			Service:       service.Service,
			Group:         service.Group,
			OutputDir:     outputDir,
			ResourceCount: len(pkg.Resources),
		})
		generatedPackages = append(generatedPackages, pkg)
		preservedArtifacts[pkg.Service.Group] = preserved
	}

	if err := g.renderer.RenderSamples(options.OutputRoot, generatedPackages); err != nil {
		return result, fmt.Errorf("render sample outputs: %w", err)
	}
	for _, pkg := range generatedPackages {
		if err := preservedArtifacts[pkg.Service.Group].applySampleFiles(options.OutputRoot); err != nil {
			return result, fmt.Errorf("preserve checked-in sample artifacts for service %q: %w", pkg.Service.Service, err)
		}
	}

	return result, nil
}

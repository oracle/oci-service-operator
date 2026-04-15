package generator

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type MutabilityFixtureRefreshResult struct {
	ResourceCount int      `json:"resourceCount"`
	FixturePaths  []string `json:"fixturePaths"`
}

func (g *Generator) RefreshMutabilityOverlayFixtures(
	ctx context.Context,
	cfg *Config,
	services []ServiceConfig,
	fixtureRoot string,
) (MutabilityFixtureRefreshResult, error) {
	if cfg == nil {
		return MutabilityFixtureRefreshResult{}, fmt.Errorf("config is required")
	}

	packages := make([]*PackageModel, 0, len(services))
	for _, service := range services {
		pkg, err := g.discoverer.BuildPackageModel(ctx, cfg, service)
		if err != nil {
			return MutabilityFixtureRefreshResult{}, fmt.Errorf("build package model for service %q: %w", service.Service, err)
		}
		packages = append(packages, pkg)
	}

	contract, err := newMutabilityOverlayDocsContract(g.mutabilityOverlayTerraformDocsVersion())
	if err != nil {
		return MutabilityFixtureRefreshResult{}, err
	}
	if strings.TrimSpace(fixtureRoot) == "" {
		fixtureRoot = g.mutabilityOverlayDocsFixtureRoot(cfg)
	}
	fixtureRoot = strings.TrimSpace(fixtureRoot)
	if fixtureRoot == "" {
		return MutabilityFixtureRefreshResult{}, fmt.Errorf("fixture root is required")
	}

	targets := make([]mutabilityOverlayRegistryPageTarget, 0)
	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		for _, resource := range pkg.Resources {
			if resource.Formal == nil {
				continue
			}
			if !mutabilityOverlayNeedsDocs(mutabilityOverlayASTFields(resource)) {
				continue
			}
			target, err := resolveMutabilityOverlayRegistryPageTarget(pkg.Service.Service, resource, contract, nil)
			if err != nil {
				return MutabilityFixtureRefreshResult{}, err
			}
			targets = append(targets, target)
		}
	}
	if len(targets) == 0 {
		return MutabilityFixtureRefreshResult{}, nil
	}

	fetcher := g.mutabilityOverlayDocsFetcher
	if fetcher == nil {
		fetcher = newMutabilityOverlayHTTPDocsFetcher(nil)
	}
	inputs, err := refreshMutabilityOverlayDocsFixtures(ctx, fixtureRoot, targets, fetcher)
	if err != nil {
		return MutabilityFixtureRefreshResult{}, err
	}

	result := MutabilityFixtureRefreshResult{
		ResourceCount: len(inputs),
		FixturePaths:  make([]string, 0, len(inputs)),
	}
	for _, input := range inputs {
		if rel := strings.TrimSpace(input.Metadata.FixtureBodyPath); rel != "" {
			result.FixturePaths = append(result.FixturePaths, filepath.ToSlash(rel))
		}
	}
	sort.Strings(result.FixturePaths)
	return result, nil
}

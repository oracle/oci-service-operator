package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/generator"
)

const (
	defaultConfigPath  = "internal/generator/config/mutability_validation_services.yaml"
	defaultFixtureRoot = "internal/generator/testdata/mutability_overlay/docs"
)

type options struct {
	configPath  string
	service     string
	all         bool
	fixtureRoot string
}

func main() {
	opts := options{}
	flag.StringVar(&opts.configPath, "config", defaultConfigPath, "Path to the mutability fixture refresh generator config.")
	flag.StringVar(&opts.service, "service", "", "Refresh fixtures for a single service from config, even if it is inactive by default.")
	flag.BoolVar(&opts.all, "all", false, "Explicitly refresh fixtures for the enabled default-active service surface from the selected config.")
	flag.StringVar(&opts.fixtureRoot, "fixture-root", defaultFixtureRoot, "Root directory where checked-in mutability docs fixtures are written.")
	flag.Parse()

	if err := run(context.Background(), opts); err != nil {
		fmt.Fprintf(os.Stderr, "osok-mutability-docs-refresh: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, opts options) error {
	serviceName, all, err := generator.NormalizeDefaultActiveSelection(opts.service, opts.all)
	if err != nil {
		return err
	}
	opts.service = serviceName
	opts.all = all

	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	configPath := absFromRoot(repoRoot, opts.configPath)
	fixtureRoot := absFromRoot(repoRoot, opts.fixtureRoot)

	cfg, services, err := loadSelectedServices(configPath, opts.service, opts.all)
	if err != nil {
		return err
	}

	pipeline := generator.New()
	result, err := pipeline.RefreshMutabilityOverlayFixtures(ctx, cfg, services, fixtureRoot)
	if err != nil {
		return fmt.Errorf("%s", formatRefreshFailure(err, opts))
	}

	fmt.Fprintf(os.Stdout, "refreshed %d mutability docs fixtures under %s\n", result.ResourceCount, fixtureRoot)
	paths := append([]string(nil), result.FixturePaths...)
	sort.Strings(paths)
	for _, path := range paths {
		fmt.Fprintf(os.Stdout, "fixture %s\n", path)
	}
	return nil
}

func loadSelectedServices(configPath string, serviceName string, all bool) (*generator.Config, []generator.ServiceConfig, error) {
	cfg, err := generator.LoadConfig(configPath)
	if err != nil {
		return nil, nil, err
	}
	services, err := cfg.SelectDefaultActiveOrExplicitServices(serviceName, all)
	if err != nil {
		return nil, nil, err
	}
	if err := cfg.VerifyFormalInputsForServices(services); err != nil {
		return nil, nil, err
	}
	return cfg, services, nil
}

func formatRefreshFailure(err error, opts options) string {
	groups := generator.ClassifyMutabilityValidationError(err)
	if len(groups) == 0 {
		return err.Error()
	}

	var b strings.Builder
	b.WriteString("mutability docs refresh failed\n")
	for _, group := range groups {
		fmt.Fprintf(&b, "\n%s:\n", group.Class)
		for _, reason := range group.Reasons {
			fmt.Fprintf(&b, "  - %s (%d)\n", reason.Reason, reason.Count)
			limit := len(reason.Examples)
			if limit > 3 {
				limit = 3
			}
			for _, example := range reason.Examples[:limit] {
				fmt.Fprintf(&b, "    - %s\n", example)
			}
		}
	}
	if strings.TrimSpace(opts.service) == "" {
		b.WriteString("\nReview the selected service surface in the validation config or retry with --service <name>.\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func absFromRoot(root, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(root, p)
}

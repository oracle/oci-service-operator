package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/generator"
)

const defaultConfigPath = "internal/generator/config/mutability_validation_services.yaml"

type options struct {
	configPath       string
	service          string
	all              bool
	snapshotDir      string
	keepSnapshot     bool
	reportOut        string
	baselinePath     string
	writeBaseline    string
	failOnRegression bool
}

type outputReport struct {
	Config   outputConfig                          `json:"config"`
	Snapshot outputSnapshot                        `json:"snapshot"`
	Summary  generator.MutabilityValidationSummary `json:"summary"`
}

type outputConfig struct {
	ConfigPath        string   `json:"configPath"`
	Service           string   `json:"service,omitempty"`
	All               bool     `json:"all"`
	GeneratedServices []string `json:"generatedServices"`
}

type outputSnapshot struct {
	Retained bool   `json:"retained"`
	Root     string `json:"root,omitempty"`
}

type snapshot struct {
	root     string
	retained bool
	auto     bool
}

func main() {
	opts := options{}
	flag.StringVar(&opts.configPath, "config", defaultConfigPath, "Path to the mutability validation generator config.")
	flag.StringVar(&opts.service, "service", "", "Generate and validate a single service from config, even if it is inactive by default.")
	flag.BoolVar(&opts.all, "all", false, "Explicitly validate the enabled default-active service surface from the selected config.")
	flag.StringVar(&opts.snapshotDir, "snapshot-dir", "", "Optional path to keep the generated mutability snapshot workspace.")
	flag.BoolVar(&opts.keepSnapshot, "keep-snapshot", false, "Keep the generated mutability snapshot workspace when using an automatic temp directory.")
	flag.StringVar(&opts.reportOut, "report-out", "", "Optional path to write the generated mutability summary JSON.")
	flag.StringVar(&opts.baselinePath, "baseline", "", "Optional mutability baseline JSON to compare against.")
	flag.StringVar(&opts.writeBaseline, "write-baseline", "", "Write the current mutability summary baseline to the given path.")
	flag.BoolVar(&opts.failOnRegression, "fail-on-regression", false, "Exit with a non-zero status when the mutability summary regresses compared to --baseline.")
	flag.Parse()

	if err := run(context.Background(), opts); err != nil {
		fmt.Fprintf(os.Stderr, "osok-generated-mutability-check: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, opts options) (err error) {
	opts, err = normalizeOptions(opts)
	if err != nil {
		return err
	}
	if err := validateOptions(opts); err != nil {
		return err
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	configPath := absFromRoot(repoRoot, opts.configPath)
	cfg, services, err := loadSelectedServices(configPath, opts.service, opts.all)
	if err != nil {
		return err
	}

	snapshotDir, err := prepareSnapshot(opts.snapshotDir, opts.keepSnapshot)
	if err != nil {
		return err
	}
	defer cleanupSnapshot(&err, snapshotDir)

	pipeline := generator.New()
	if _, err := pipeline.Generate(ctx, cfg, services, generator.Options{
		OutputRoot:              snapshotDir.root,
		Overwrite:               true,
		FullSync:                opts.all,
		EnableMutabilityOverlay: true,
	}); err != nil {
		return fmt.Errorf("%s", formatGenerationFailure(err, opts))
	}

	report, err := generator.LoadMutabilityValidationReport(snapshotDir.root)
	if err != nil {
		return err
	}

	output := outputReport{
		Config: outputConfig{
			ConfigPath:        configPath,
			Service:           opts.service,
			All:               opts.all,
			GeneratedServices: serviceNames(services),
		},
		Snapshot: outputSnapshot{
			Retained: snapshotDir.retained,
		},
		Summary: report.Summary,
	}
	if snapshotDir.retained {
		output.Snapshot.Root = snapshotDir.root
	}

	rendered, err := renderOutput(output)
	if err != nil {
		return err
	}
	if err := writeReport(rendered, opts.reportOut); err != nil {
		return err
	}
	if strings.TrimSpace(opts.writeBaseline) != "" {
		if err := generator.WriteMutabilityValidationBaseline(opts.writeBaseline, report.Summary); err != nil {
			return err
		}
	}
	return checkRegression(opts, report.Summary)
}

func normalizeOptions(opts options) (options, error) {
	serviceName, all, err := generator.NormalizeDefaultActiveSelection(opts.service, opts.all)
	if err != nil {
		return opts, err
	}
	opts.service = serviceName
	opts.all = all
	return opts, nil
}

func validateOptions(opts options) error {
	if opts.failOnRegression && strings.TrimSpace(opts.baselinePath) == "" {
		return fmt.Errorf("--baseline is required when --fail-on-regression is set")
	}
	if strings.TrimSpace(opts.writeBaseline) != "" && (strings.TrimSpace(opts.baselinePath) != "" || opts.failOnRegression) {
		return fmt.Errorf("--write-baseline cannot be combined with --baseline or --fail-on-regression")
	}
	if (strings.TrimSpace(opts.writeBaseline) != "" || opts.failOnRegression) && strings.TrimSpace(opts.service) != "" {
		return fmt.Errorf("--write-baseline and --fail-on-regression only support default-active-surface runs; remove --service and target the default active surface")
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

func checkRegression(opts options, summary generator.MutabilityValidationSummary) error {
	if !opts.failOnRegression {
		return nil
	}
	baseline, err := generator.LoadMutabilityValidationBaseline(opts.baselinePath)
	if err != nil {
		return err
	}
	comparison := generator.CompareMutabilityValidationSummary(summary, baseline)
	if comparison.HasFailures() {
		return fmt.Errorf("%s", formatRegressionMessage(comparison, summary, opts.reportOut, opts.baselinePath))
	}
	return nil
}

func renderOutput(output outputReport) ([]byte, error) {
	var rendered bytes.Buffer
	encoder := json.NewEncoder(&rendered)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return nil, err
	}
	return rendered.Bytes(), nil
}

func writeReport(rendered []byte, path string) error {
	if strings.TrimSpace(path) == "" {
		_, err := os.Stdout.Write(rendered)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, rendered, 0o644)
}

func prepareSnapshot(snapshotDir string, keepSnapshot bool) (snapshot, error) {
	if strings.TrimSpace(snapshotDir) == "" {
		tempRoot, err := os.MkdirTemp("", "osok-generated-mutability-")
		if err != nil {
			return snapshot{}, err
		}
		return snapshot{
			root:     tempRoot,
			retained: keepSnapshot,
			auto:     true,
		}, nil
	}

	absSnapshot, err := filepath.Abs(snapshotDir)
	if err != nil {
		return snapshot{}, err
	}
	if err := os.MkdirAll(absSnapshot, 0o755); err != nil {
		return snapshot{}, err
	}
	entries, err := os.ReadDir(absSnapshot)
	if err != nil {
		return snapshot{}, err
	}
	if len(entries) > 0 {
		return snapshot{}, fmt.Errorf("snapshot dir %q must be empty", absSnapshot)
	}
	return snapshot{
		root:     absSnapshot,
		retained: true,
	}, nil
}

func cleanupSnapshot(runErr *error, snapshotDir snapshot) {
	if *runErr == nil {
		if !snapshotDir.retained {
			_ = os.RemoveAll(snapshotDir.root)
		}
		return
	}
	if snapshotDir.auto && !snapshotDir.retained {
		*runErr = fmt.Errorf("%w (snapshot kept at %s)", *runErr, snapshotDir.root)
	}
}

func formatGenerationFailure(err error, opts options) string {
	groups := generator.ClassifyMutabilityValidationError(err)
	if len(groups) == 0 {
		return err.Error()
	}

	var b strings.Builder
	b.WriteString("generated mutability validation failed\n")
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
	b.WriteString("\nIf the pinned docs fixtures need refresh, run:\n")
	b.WriteString("  go run ./cmd/osok-mutability-docs-refresh")
	if strings.TrimSpace(opts.configPath) != "" {
		fmt.Fprintf(&b, " --config %s", opts.configPath)
	}
	if strings.TrimSpace(opts.service) != "" {
		fmt.Fprintf(&b, " --service %s", opts.service)
	} else {
		b.WriteString(" --all")
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatRegressionMessage(
	comparison generator.MutabilityValidationComparison,
	summary generator.MutabilityValidationSummary,
	reportPath string,
	baselinePath string,
) string {
	var b strings.Builder
	fmt.Fprintf(&b, "generated mutability gate failed against baseline %s\n", baselinePath)
	if len(comparison.ScopeChanges) > 0 {
		b.WriteString("\nScope changes:\n")
		for _, change := range comparison.ScopeChanges {
			fmt.Fprintf(&b, "  - %s\n", change)
		}
	}
	if len(comparison.Regressions) > 0 {
		b.WriteString("\nRegressions:\n")
		for _, regression := range comparison.Regressions {
			fmt.Fprintf(&b, "  - %s\n", regression)
		}
	}

	b.WriteString("\nCurrent summary:\n")
	fmt.Fprintf(&b, "  - resources: %d\n", summary.Aggregate.Resources)
	fmt.Fprintf(&b, "  - fields: %d\n", summary.Aggregate.Fields)
	fmt.Fprintf(&b, "  - allowInPlaceCount: %d\n", summary.Aggregate.AllowInPlaceCount)
	fmt.Fprintf(&b, "  - unknownPolicyCount: %d\n", summary.Aggregate.UnknownPolicyCount)
	fmt.Fprintf(&b, "  - docsDeniedCount: %d\n", summary.Aggregate.DocsDeniedCount)
	fmt.Fprintf(&b, "  - unresolvedJoinCount: %d\n", summary.Aggregate.UnresolvedJoinCount)
	fmt.Fprintf(&b, "  - ambiguousJoinCount: %d\n", summary.Aggregate.AmbiguousJoinCount)
	fmt.Fprintf(&b, "  - mergeConflictCount: %d\n", summary.Aggregate.MergeConflictCount)

	appendTopResourceSection(&b, "unknownPolicyCount", summary.Resources, func(resource generator.MutabilityValidationResourceSummary) int {
		return resource.UnknownPolicyCount
	})
	appendTopResourceSection(&b, "docsDeniedCount", summary.Resources, func(resource generator.MutabilityValidationResourceSummary) int {
		return len(resource.DocsDeniedFields)
	})
	appendTopResourceSection(&b, "mergeConflictCount", summary.Resources, func(resource generator.MutabilityValidationResourceSummary) int {
		return len(resource.MergeConflicts)
	})

	if strings.TrimSpace(reportPath) != "" {
		fmt.Fprintf(&b, "\nGenerated mutability report: %s\n", reportPath)
	} else {
		b.WriteString("\nGenerated mutability report was written to stdout.\n")
	}
	b.WriteString("If the update-surface change is intentional, refresh the baseline with:\n")
	b.WriteString("  make generated-mutability-baseline\n")
	return strings.TrimRight(b.String(), "\n")
}

func appendTopResourceSection(
	b *strings.Builder,
	label string,
	resources []generator.MutabilityValidationResourceSummary,
	score func(generator.MutabilityValidationResourceSummary) int,
) {
	type scoredResource struct {
		resource generator.MutabilityValidationResourceSummary
		score    int
	}

	scored := make([]scoredResource, 0, len(resources))
	for _, resource := range resources {
		if value := score(resource); value > 0 {
			scored = append(scored, scoredResource{resource: resource, score: value})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].resource.Service != scored[j].resource.Service {
			return scored[i].resource.Service < scored[j].resource.Service
		}
		if scored[i].resource.Kind != scored[j].resource.Kind {
			return scored[i].resource.Kind < scored[j].resource.Kind
		}
		return scored[i].resource.FormalSlug < scored[j].resource.FormalSlug
	})

	fmt.Fprintf(b, "\nTop %s resources:\n", label)
	if len(scored) == 0 {
		b.WriteString("  - none\n")
		return
	}
	limit := len(scored)
	if limit > 3 {
		limit = 3
	}
	for _, item := range scored[:limit] {
		fmt.Fprintf(
			b,
			"  - %s/%s (%s): %d\n",
			item.resource.Service,
			item.resource.Kind,
			item.resource.FormalSlug,
			item.score,
		)
	}
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

func serviceNames(services []generator.ServiceConfig) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, service.Service)
	}
	sort.Strings(names)
	return names
}

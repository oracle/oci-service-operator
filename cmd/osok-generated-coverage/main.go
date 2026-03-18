package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/generator"
	"github.com/oracle/oci-service-operator/internal/validator/apispec"
	"github.com/oracle/oci-service-operator/internal/validator/metrics"
)

type options struct {
	configPath       string
	service          string
	all              bool
	top              int
	snapshotDir      string
	keepSnapshot     bool
	controllerGen    string
	reportOut        string
	validatorJSONOut string
	baselinePath     string
	writeBaseline    string
	failOnRegression bool
	preserveSpec     bool
}

type validatorEnvelope struct {
	API apispec.Report `json:"api"`
}

type outputReport struct {
	Config    outputConfig       `json:"config"`
	Snapshot  outputSnapshot     `json:"snapshot"`
	Validator outputValidator    `json:"validator"`
	Summary   metrics.APISummary `json:"summary"`
}

type outputConfig struct {
	ConfigPath                  string   `json:"configPath"`
	Service                     string   `json:"service,omitempty"`
	All                         bool     `json:"all"`
	GeneratedServices           []string `json:"generatedServices"`
	GeneratedGroups             []string `json:"generatedGroups"`
	Top                         int      `json:"top"`
	PreserveExistingSpecSurface bool     `json:"preserveExistingSpecSurface,omitempty"`
}

type outputSnapshot struct {
	Retained bool   `json:"retained"`
	Root     string `json:"root,omitempty"`
}

type outputValidator struct {
	JSONPath string `json:"jsonPath,omitempty"`
}

type snapshot struct {
	root     string
	retained bool
	auto     bool
}

func main() {
	opts := options{}
	flag.StringVar(&opts.configPath, "config", "internal/generator/config/services.yaml", "Path to the OSOK API generator config file.")
	flag.StringVar(&opts.service, "service", "", "Generate and report a single configured service.")
	flag.BoolVar(&opts.all, "all", false, "Generate and report all configured services.")
	flag.IntVar(&opts.top, "top", 10, "Number of top offenders to include per category. Use 0 for all.")
	flag.StringVar(&opts.snapshotDir, "snapshot-dir", "", "Optional path to keep the generated snapshot workspace.")
	flag.BoolVar(&opts.keepSnapshot, "keep-snapshot", false, "Keep the generated snapshot workspace when using an automatic temp directory.")
	flag.StringVar(&opts.controllerGen, "controller-gen", "", "Path to the controller-gen binary. Defaults to <repo>/bin/controller-gen.")
	flag.StringVar(&opts.reportOut, "report-out", "", "Optional path to write the generated coverage summary JSON.")
	flag.StringVar(&opts.validatorJSONOut, "validator-json-out", "", "Optional path to also write the raw validator JSON report.")
	flag.StringVar(&opts.baselinePath, "baseline", "", "Optional generated coverage baseline JSON to compare against.")
	flag.StringVar(&opts.writeBaseline, "write-baseline", "", "Write the current generated coverage baseline to the given path.")
	flag.BoolVar(&opts.failOnRegression, "fail-on-regression", false, "Exit with a non-zero status when the generated coverage report regresses compared to --baseline.")
	flag.BoolVar(&opts.preserveSpec, "preserve-existing-spec-surface", false, "Preserve the current checked-in spec/helper surface for selected packages while regenerating status/read-model outputs in the snapshot.")
	flag.Parse()

	if err := run(context.Background(), opts); err != nil {
		fmt.Fprintf(os.Stderr, "osok-generated-coverage: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, opts options) (err error) {
	if err := validateOptions(opts); err != nil {
		return err
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}

	configPath := absFromRoot(repoRoot, opts.configPath)
	cfg, err := generator.LoadConfig(configPath)
	if err != nil {
		return err
	}
	services, err := cfg.SelectServices(opts.service, opts.all)
	if err != nil {
		return err
	}

	controllerGenPath := opts.controllerGen
	if strings.TrimSpace(controllerGenPath) == "" {
		controllerGenPath = filepath.Join(repoRoot, "bin", "controller-gen")
	}
	if _, statErr := os.Stat(controllerGenPath); statErr != nil {
		return fmt.Errorf("controller-gen not found at %q; run `make controller-gen` or pass --controller-gen: %w", controllerGenPath, statErr)
	}

	selectedGroups := serviceGroups(services)
	snapshotDir, err := prepareSnapshot(repoRoot, selectedGroups, opts.snapshotDir, opts.keepSnapshot)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			if !snapshotDir.retained {
				_ = os.RemoveAll(snapshotDir.root)
			}
			return
		}
		if snapshotDir.auto && !snapshotDir.retained {
			err = fmt.Errorf("%w (snapshot kept at %s)", err, snapshotDir.root)
		}
	}()

	pipeline := generator.New()
	preserveRoot := ""
	if opts.preserveSpec {
		preserveRoot = repoRoot
	}
	if _, err = pipeline.Generate(ctx, cfg, services, generator.Options{
		OutputRoot:                      snapshotDir.root,
		Overwrite:                       true,
		PreserveExistingSpecSurfaceRoot: preserveRoot,
	}); err != nil {
		return fmt.Errorf("generate selected services into snapshot: %w", err)
	}

	snapshotEnv := snapshotCommandEnv(snapshotDir.root)
	if err = runCommand(snapshotDir.root, snapshotEnv, "go", "run", "./hack/update_validator_registries.go", "--write"); err != nil {
		return fmt.Errorf("refresh validator registries in snapshot: %w", err)
	}
	if err = runCommand(snapshotDir.root, snapshotEnv, controllerGenPath, "object:headerFile=hack/boilerplate.go.txt", "paths="+controllerGenPaths(selectedGroups)); err != nil {
		return fmt.Errorf("generate deepcopy code in snapshot: %w", err)
	}

	validatorJSON, err := runCommandOutput(snapshotDir.root, snapshotEnv, "go", "run", "./cmd/osok-schema-validator", "--provider-path", ".", "--format", "json")
	if err != nil {
		return fmt.Errorf("run validator in snapshot: %w", err)
	}

	if strings.TrimSpace(opts.validatorJSONOut) != "" {
		if err = writeFile(opts.validatorJSONOut, validatorJSON); err != nil {
			return err
		}
	}

	envelope := validatorEnvelope{}
	if err = json.Unmarshal(validatorJSON, &envelope); err != nil {
		return fmt.Errorf("decode validator JSON: %w", err)
	}

	output := outputReport{
		Config: outputConfig{
			ConfigPath:                  configPath,
			Service:                     opts.service,
			All:                         opts.all,
			GeneratedServices:           serviceNames(services),
			GeneratedGroups:             selectedGroups,
			Top:                         opts.top,
			PreserveExistingSpecSurface: opts.preserveSpec,
		},
		Snapshot: outputSnapshot{
			Retained: snapshotDir.retained,
		},
		Validator: outputValidator{
			JSONPath: opts.validatorJSONOut,
		},
		Summary: metrics.SummarizeAPI(envelope.API, opts.top),
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
		if err := metrics.WriteBaseline(opts.writeBaseline, output.Summary); err != nil {
			return err
		}
	}
	if opts.failOnRegression {
		baseline, err := metrics.LoadBaseline(opts.baselinePath)
		if err != nil {
			return err
		}
		comparison := metrics.CompareSummary(output.Summary, baseline)
		if comparison.HasFailures() {
			return fmt.Errorf("%s", formatRegressionMessage(comparison, output.Summary, opts.reportOut, opts.validatorJSONOut, opts.baselinePath))
		}
	}
	return nil
}

func validateOptions(opts options) error {
	if opts.failOnRegression && strings.TrimSpace(opts.baselinePath) == "" {
		return fmt.Errorf("--baseline is required when --fail-on-regression is set")
	}
	if strings.TrimSpace(opts.writeBaseline) != "" && (strings.TrimSpace(opts.baselinePath) != "" || opts.failOnRegression) {
		return fmt.Errorf("--write-baseline cannot be combined with --baseline or --fail-on-regression")
	}
	if (strings.TrimSpace(opts.writeBaseline) != "" || opts.failOnRegression) && strings.TrimSpace(opts.service) != "" {
		return fmt.Errorf("--write-baseline and --fail-on-regression only support all-service runs; remove --service and use --all")
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
	return writeFile(path, rendered)
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func formatRegressionMessage(comparison metrics.Comparison, summary metrics.APISummary, reportPath string, validatorPath string, baselinePath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "generated coverage gate failed against baseline %s\n", baselinePath)
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
	b.WriteString("\nCurrent top offenders:\n")
	appendOffenderSection(&b, "missingFields", summary.TopOffenders.MissingFields, 3)
	appendOffenderSection(&b, "mandatoryMissingFields", summary.TopOffenders.MandatoryMissingFields, 3)
	appendOffenderSection(&b, "extraSpecFields", summary.TopOffenders.ExtraSpecFields, 3)

	if strings.TrimSpace(reportPath) != "" {
		fmt.Fprintf(&b, "\nGenerated coverage report: %s\n", reportPath)
	} else {
		b.WriteString("\nGenerated coverage report was written to stdout.\n")
	}
	if strings.TrimSpace(validatorPath) != "" {
		fmt.Fprintf(&b, "Raw validator JSON: %s\n", validatorPath)
	}
	b.WriteString("If the metric or scope change is intentional, refresh the baseline with:\n")
	b.WriteString("  make generated-coverage-baseline\n")
	return strings.TrimRight(b.String(), "\n")
}

func appendOffenderSection(b *strings.Builder, label string, offenders []metrics.Offender, limit int) {
	fmt.Fprintf(b, "  %s:\n", label)
	if len(offenders) == 0 {
		b.WriteString("    - none\n")
		return
	}
	count := len(offenders)
	if count > limit {
		count = limit
	}
	for _, offender := range offenders[:count] {
		specLabel := offender.Spec
		if surface := strings.TrimSpace(offender.APISurface); surface != "" && surface != "spec" {
			specLabel = specLabel + " (" + surface + ")"
		}
		fmt.Fprintf(b, "    - %s/%s -> %s (%d)\n", offender.Service, specLabel, offender.SDKStruct, offender.Count)
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

func serviceGroups(services []generator.ServiceConfig) []string {
	set := make(map[string]struct{}, len(services))
	for _, service := range services {
		set[service.Group] = struct{}{}
	}

	groups := make([]string, 0, len(set))
	for group := range set {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	return groups
}

func controllerGenPaths(groups []string) string {
	paths := make([]string, 0, len(groups))
	for _, group := range groups {
		paths = append(paths, "./api/"+group+"/...")
	}
	return strings.Join(paths, ";")
}

func prepareSnapshot(repoRoot string, selectedGroups []string, snapshotDir string, keepSnapshot bool) (snapshot, error) {
	if strings.TrimSpace(snapshotDir) == "" {
		tempRoot, err := os.MkdirTemp("", "osok-generated-coverage-")
		if err != nil {
			return snapshot{}, err
		}
		snap := snapshot{
			root:     tempRoot,
			retained: keepSnapshot,
			auto:     true,
		}
		if err := populateSnapshot(repoRoot, tempRoot, selectedGroups); err != nil {
			return snapshot{}, err
		}
		return snap, nil
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
	if err := populateSnapshot(repoRoot, absSnapshot, selectedGroups); err != nil {
		return snapshot{}, err
	}
	return snapshot{
		root:     absSnapshot,
		retained: true,
	}, nil
}

func populateSnapshot(repoRoot, snapshotRoot string, selectedGroups []string) error {
	for _, entry := range []string{"go.mod", "go.sum", "cmd", "controllers", "hack", "pkg", "vendor", "validator_allowlist.yaml"} {
		if err := symlink(filepath.Join(repoRoot, entry), filepath.Join(snapshotRoot, entry)); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Join(snapshotRoot, "config", "samples"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(snapshotRoot, "packages"), 0o755); err != nil {
		return err
	}

	if err := populateInternal(repoRoot, snapshotRoot); err != nil {
		return err
	}
	return populateAPI(repoRoot, snapshotRoot, selectedGroups)
}

func populateInternal(repoRoot, snapshotRoot string) error {
	internalRoot := filepath.Join(snapshotRoot, "internal")
	if err := os.MkdirAll(internalRoot, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(filepath.Join(repoRoot, "internal"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(repoRoot, "internal", entry.Name())
		dst := filepath.Join(internalRoot, entry.Name())
		if entry.Name() == "validator" {
			if err := copyTree(src, dst); err != nil {
				return err
			}
			continue
		}
		if err := symlink(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func populateAPI(repoRoot, snapshotRoot string, selectedGroups []string) error {
	selected := make(map[string]struct{}, len(selectedGroups))
	for _, group := range selectedGroups {
		selected[group] = struct{}{}
	}

	apiRoot := filepath.Join(snapshotRoot, "api")
	if err := os.MkdirAll(apiRoot, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(filepath.Join(repoRoot, "api"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, ok := selected[entry.Name()]; ok {
			continue
		}
		if err := symlink(filepath.Join(repoRoot, "api", entry.Name()), filepath.Join(apiRoot, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func symlink(src, dst string) error {
	return os.Symlink(src, dst)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return output.Close()
}

func snapshotCommandEnv(snapshotRoot string) []string {
	env := append([]string{}, os.Environ()...)
	env = setEnv(env, "GOCACHE", filepath.Join(snapshotRoot, ".gocache"))
	env = setEnv(env, "GOMODCACHE", filepath.Join(snapshotRoot, ".gomodcache"))
	env = setEnv(env, "GOFLAGS", appendBuildVCSFlag(os.Getenv("GOFLAGS")))
	return env
}

func appendBuildVCSFlag(current string) string {
	if strings.Contains(current, "-buildvcs=") || strings.Contains(current, "-buildvcs ") || strings.Contains(current, "-buildvcs") {
		return strings.TrimSpace(current)
	}
	if strings.TrimSpace(current) == "" {
		return "-buildvcs=false"
	}
	return strings.TrimSpace(current) + " -buildvcs=false"
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func runCommand(dir string, env []string, name string, args ...string) error {
	_, err := runCommandOutput(dir, env, name, args...)
	return err
}

func runCommandOutput(dir string, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message != "" {
			return nil, fmt.Errorf("%s: %w\n%s", strings.Join(append([]string{name}, args...), " "), err, message)
		}
		return nil, fmt.Errorf("%s: %w", strings.Join(append([]string{name}, args...), " "), err)
	}
	return stdout.Bytes(), nil
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/generator"
)

const defaultConfigPath = "internal/generator/config/services.yaml"

type options struct {
	configPath    string
	service       string
	all           bool
	snapshotDir   string
	keepSnapshot  bool
	controllerGen string
	reportOut     string
}

type outputReport struct {
	Config   outputConfig   `json:"config"`
	Snapshot outputSnapshot `json:"snapshot"`
	Build    outputBuild    `json:"build"`
}

type outputConfig struct {
	ConfigPath        string   `json:"configPath"`
	Service           string   `json:"service,omitempty"`
	All               bool     `json:"all"`
	GeneratedServices []string `json:"generatedServices"`
	GeneratedGroups   []string `json:"generatedGroups"`
}

type outputSnapshot struct {
	Retained bool   `json:"retained"`
	Root     string `json:"root,omitempty"`
}

type outputBuild struct {
	ControllerPackages     []string `json:"controllerPackages"`
	ServiceManagerPackages []string `json:"serviceManagerPackages"`
	RegistrationPackages   []string `json:"registrationPackages,omitempty"`
}

type snapshot struct {
	root     string
	retained bool
	auto     bool
}

type runtimeCheckRunInputs struct {
	repoRoot                    string
	configPath                  string
	cfg                         *generator.Config
	services                    []generator.ServiceConfig
	packageModels               []*generator.PackageModel
	controllerGenPath           string
	selectedGroups              []string
	selectedServiceManagerRoots []string
}

func main() {
	opts := options{}
	flag.StringVar(&opts.configPath, "config", defaultConfigPath, "Path to the generator config file used for the runtime snapshot.")
	flag.StringVar(&opts.service, "service", "", "Generate and validate a single configured service.")
	flag.BoolVar(&opts.all, "all", false, "Generate and validate all configured services.")
	flag.StringVar(&opts.snapshotDir, "snapshot-dir", "", "Optional path to keep the generated runtime snapshot workspace.")
	flag.BoolVar(&opts.keepSnapshot, "keep-snapshot", false, "Keep the generated runtime snapshot workspace when using an automatic temp directory.")
	flag.StringVar(&opts.controllerGen, "controller-gen", "", "Path to the controller-gen binary. Defaults to <repo>/bin/controller-gen.")
	flag.StringVar(&opts.reportOut, "report-out", "", "Optional path to write the generated runtime summary JSON.")
	flag.Parse()

	if err := run(context.Background(), opts); err != nil {
		fmt.Fprintf(os.Stderr, "osok-generated-runtime-check: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, opts options) (err error) {
	inputs, err := loadRuntimeCheckRunInputs(ctx, opts)
	if err != nil {
		return err
	}
	snapshotDir, err := prepareSnapshot(
		inputs.repoRoot,
		inputs.selectedGroups,
		inputs.selectedServiceManagerRoots,
		opts.snapshotDir,
		opts.keepSnapshot,
	)
	if err != nil {
		return err
	}
	defer cleanupRuntimeCheckSnapshot(&err, snapshotDir)

	if err := generateRuntimeCheckSnapshot(ctx, inputs, snapshotDir.root); err != nil {
		return err
	}

	snapshotEnv := snapshotCommandEnv(snapshotDir.root)
	if err = runCommand(snapshotDir.root, snapshotEnv, inputs.controllerGenPath, "object:headerFile=hack/boilerplate.go.txt", "paths="+controllerGenPaths(inputs.selectedGroups)); err != nil {
		return fmt.Errorf("generate deepcopy code in runtime snapshot: %w", err)
	}

	build, err := collectBuildPlan(snapshotDir.root, inputs.selectedGroups, inputs.selectedServiceManagerRoots)
	if err != nil {
		return err
	}
	if err := compileGeneratedPackages(snapshotDir.root, snapshotEnv, build); err != nil {
		return err
	}

	output := outputReport{
		Config: outputConfig{
			ConfigPath:        inputs.configPath,
			Service:           opts.service,
			All:               opts.all,
			GeneratedServices: serviceNames(inputs.services),
			GeneratedGroups:   inputs.selectedGroups,
		},
		Snapshot: outputSnapshot{
			Retained: snapshotDir.retained,
		},
		Build: build,
	}
	if snapshotDir.retained {
		output.Snapshot.Root = snapshotDir.root
	}

	rendered, err := renderOutput(output)
	if err != nil {
		return err
	}
	return writeReport(rendered, opts.reportOut)
}

func loadRuntimeCheckRunInputs(ctx context.Context, opts options) (runtimeCheckRunInputs, error) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return runtimeCheckRunInputs{}, err
	}

	configPath := absFromRoot(repoRoot, opts.configPath)
	cfg, err := generator.LoadConfig(configPath)
	if err != nil {
		return runtimeCheckRunInputs{}, err
	}
	services, err := cfg.SelectServices(opts.service, opts.all)
	if err != nil {
		return runtimeCheckRunInputs{}, err
	}
	if err := cfg.VerifyFormalInputs(); err != nil {
		return runtimeCheckRunInputs{}, err
	}
	packageModels, err := buildPackageModels(ctx, cfg, services)
	if err != nil {
		return runtimeCheckRunInputs{}, err
	}
	controllerGenPath, err := resolveRuntimeCheckControllerGenPath(repoRoot, opts.controllerGen)
	if err != nil {
		return runtimeCheckRunInputs{}, err
	}

	return runtimeCheckRunInputs{
		repoRoot:                    repoRoot,
		configPath:                  configPath,
		cfg:                         cfg,
		services:                    services,
		packageModels:               packageModels,
		controllerGenPath:           controllerGenPath,
		selectedGroups:              serviceGroups(services),
		selectedServiceManagerRoots: serviceManagerRoots(services),
	}, nil
}

func resolveRuntimeCheckControllerGenPath(repoRoot string, controllerGenPath string) (string, error) {
	if strings.TrimSpace(controllerGenPath) == "" {
		controllerGenPath = filepath.Join(repoRoot, "bin", "controller-gen")
	}
	if _, statErr := os.Stat(controllerGenPath); statErr != nil {
		return "", fmt.Errorf("controller-gen not found at %q; run `make controller-gen` or pass --controller-gen: %w", controllerGenPath, statErr)
	}
	return controllerGenPath, nil
}

func cleanupRuntimeCheckSnapshot(runErr *error, snapshotDir snapshot) {
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

func generateRuntimeCheckSnapshot(ctx context.Context, inputs runtimeCheckRunInputs, snapshotRoot string) error {
	pipeline := generator.New()
	if _, err := pipeline.Generate(ctx, inputs.cfg, inputs.services, generator.Options{
		OutputRoot:                      snapshotRoot,
		Overwrite:                       true,
		PreserveExistingSpecSurfaceRoot: inputs.repoRoot,
	}); err != nil {
		return fmt.Errorf("generate selected services into runtime snapshot: %w", err)
	}
	if err := preserveCheckedInCompanionFiles(inputs.repoRoot, snapshotRoot, inputs.packageModels); err != nil {
		return fmt.Errorf("preserve checked-in companion files in runtime snapshot: %w", err)
	}
	return nil
}

func compileGeneratedPackages(snapshotRoot string, snapshotEnv []string, build outputBuild) error {
	if err := compilePackageSet(snapshotRoot, snapshotEnv, "generated controller packages", build.ControllerPackages); err != nil {
		return err
	}
	if err := compilePackageSet(snapshotRoot, snapshotEnv, "generated service-manager packages", build.ServiceManagerPackages); err != nil {
		return err
	}
	if len(build.RegistrationPackages) == 0 {
		return nil
	}
	return compilePackageSet(snapshotRoot, snapshotEnv, "generated registration packages", build.RegistrationPackages)
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

func buildPackageModels(ctx context.Context, cfg *generator.Config, services []generator.ServiceConfig) ([]*generator.PackageModel, error) {
	discoverer := generator.NewDiscoverer()
	packages := make([]*generator.PackageModel, 0, len(services))
	for _, service := range services {
		pkg, err := discoverer.BuildPackageModel(ctx, cfg, service)
		if err != nil {
			return nil, fmt.Errorf("build package model for service %q: %w", service.Service, err)
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}

func controllerGenPaths(groups []string) string {
	paths := make([]string, 0, len(groups))
	for _, group := range groups {
		paths = append(paths, "./api/"+group+"/...")
	}
	return strings.Join(paths, ";")
}

func prepareSnapshot(repoRoot string, selectedGroups []string, selectedServiceManagerRoots []string, snapshotDir string, keepSnapshot bool) (snapshot, error) {
	if strings.TrimSpace(snapshotDir) == "" {
		tempRoot, err := os.MkdirTemp("", "osok-generated-runtime-")
		if err != nil {
			return snapshot{}, err
		}
		snap := snapshot{
			root:     tempRoot,
			retained: keepSnapshot,
			auto:     true,
		}
		if err := populateSnapshot(repoRoot, tempRoot, selectedGroups, selectedServiceManagerRoots); err != nil {
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
	if err := populateSnapshot(repoRoot, absSnapshot, selectedGroups, selectedServiceManagerRoots); err != nil {
		return snapshot{}, err
	}
	return snapshot{
		root:     absSnapshot,
		retained: true,
	}, nil
}

func populateSnapshot(repoRoot, snapshotRoot string, selectedGroups []string, selectedServiceManagerRoots []string) error {
	for _, entry := range []string{"go.mod", "go.sum", "hack", "vendor"} {
		if err := symlinkIfPresent(filepath.Join(repoRoot, entry), filepath.Join(snapshotRoot, entry)); err != nil {
			return err
		}
	}
	if err := symlinkIfPresent(filepath.Join(repoRoot, "formal"), filepath.Join(snapshotRoot, "formal")); err != nil {
		return err
	}

	if err := populateAPI(repoRoot, snapshotRoot, selectedGroups); err != nil {
		return err
	}
	if err := populateControllers(repoRoot, snapshotRoot, selectedGroups); err != nil {
		return err
	}
	if err := populatePkg(repoRoot, snapshotRoot, selectedServiceManagerRoots); err != nil {
		return err
	}
	if err := populateRegistrations(repoRoot, snapshotRoot, selectedGroups); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(snapshotRoot, "config", "samples"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(snapshotRoot, "packages"), 0o755); err != nil {
		return err
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

func populateControllers(repoRoot, snapshotRoot string, selectedGroups []string) error {
	selected := make(map[string]struct{}, len(selectedGroups))
	for _, group := range selectedGroups {
		selected[group] = struct{}{}
	}

	controllersRoot := filepath.Join(snapshotRoot, "controllers")
	if err := os.MkdirAll(controllersRoot, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(filepath.Join(repoRoot, "controllers"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if _, ok := selected[entry.Name()]; ok {
				continue
			}
		}
		if err := symlink(filepath.Join(repoRoot, "controllers", entry.Name()), filepath.Join(controllersRoot, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func populatePkg(repoRoot, snapshotRoot string, selectedServiceManagerRoots []string) error {
	pkgRoot := filepath.Join(snapshotRoot, "pkg")
	if err := os.MkdirAll(pkgRoot, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(filepath.Join(repoRoot, "pkg"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == "servicemanager" {
			continue
		}
		if err := symlink(filepath.Join(repoRoot, "pkg", entry.Name()), filepath.Join(pkgRoot, entry.Name())); err != nil {
			return err
		}
	}

	return populateServiceManagerRoot(repoRoot, snapshotRoot, selectedServiceManagerRoots)
}

func populateServiceManagerRoot(repoRoot, snapshotRoot string, selectedRoots []string) error {
	selected := make(map[string]struct{}, len(selectedRoots))
	for _, root := range selectedRoots {
		selected[root] = struct{}{}
	}

	serviceManagerRoot := filepath.Join(snapshotRoot, "pkg", "servicemanager")
	if err := os.MkdirAll(serviceManagerRoot, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(filepath.Join(repoRoot, "pkg", "servicemanager"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if _, ok := selected[entry.Name()]; ok {
				continue
			}
		}
		if err := symlink(filepath.Join(repoRoot, "pkg", "servicemanager", entry.Name()), filepath.Join(serviceManagerRoot, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func populateRegistrations(repoRoot, snapshotRoot string, selectedGroups []string) error {
	selected := make(map[string]struct{}, len(selectedGroups))
	for _, group := range selectedGroups {
		selected[group] = struct{}{}
	}

	registrationsRoot := filepath.Join(snapshotRoot, "internal", "registrations")
	if err := os.MkdirAll(registrationsRoot, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(filepath.Join(repoRoot, "internal", "registrations"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if shouldSkipGeneratedRegistration(entry.Name(), selected) {
			continue
		}
		if err := symlink(filepath.Join(repoRoot, "internal", "registrations", entry.Name()), filepath.Join(registrationsRoot, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func collectBuildPlan(snapshotRoot string, selectedGroups []string, selectedServiceManagerRoots []string) (outputBuild, error) {
	build := outputBuild{}

	controllerPackages, err := collectControllerPackages(snapshotRoot, selectedGroups)
	if err != nil {
		return build, err
	}
	serviceManagerPackages, err := collectServiceManagerPackages(snapshotRoot, selectedServiceManagerRoots)
	if err != nil {
		return build, err
	}
	registrationPackages, err := collectRegistrationPackages(snapshotRoot)
	if err != nil {
		return build, err
	}

	build.ControllerPackages = controllerPackages
	build.ServiceManagerPackages = serviceManagerPackages
	build.RegistrationPackages = registrationPackages

	if len(build.ControllerPackages) == 0 {
		return build, fmt.Errorf("no generated controller packages detected in snapshot")
	}
	if len(build.ServiceManagerPackages) == 0 {
		return build, fmt.Errorf("no generated service-manager packages detected in snapshot")
	}

	return build, nil
}

func collectControllerPackages(snapshotRoot string, selectedGroups []string) ([]string, error) {
	packages := make([]string, 0, len(selectedGroups))
	for _, group := range selectedGroups {
		groupDir := filepath.Join(snapshotRoot, "controllers", group)
		entries, err := os.ReadDir(groupDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || !strings.HasSuffix(entry.Name(), "_controller.go") {
				continue
			}
			packages = append(packages, "./controllers/"+group)
			break
		}
	}

	sort.Strings(packages)
	return packages, nil
}

func collectServiceManagerPackages(snapshotRoot string, selectedRoots []string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, root := range selectedRoots {
		if err := collectServiceManagerPackagesUnderRoot(snapshotRoot, root, seen); err != nil {
			return nil, err
		}
	}

	packages := make([]string, 0, len(seen))
	for pkg := range seen {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	return packages, nil
}

func collectServiceManagerPackagesUnderRoot(snapshotRoot string, root string, seen map[string]struct{}) error {
	rootDir := filepath.Join(snapshotRoot, "pkg", "servicemanager", root)
	if _, err := os.Stat(rootDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return filepath.WalkDir(rootDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isGeneratedServiceManagerFile(entry.Name()) {
			return nil
		}

		rel, err := filepath.Rel(snapshotRoot, filepath.Dir(path))
		if err != nil {
			return err
		}
		seen["./"+filepath.ToSlash(rel)] = struct{}{}
		return nil
	})
}

func isGeneratedServiceManagerFile(name string) bool {
	if filepath.Ext(name) != ".go" {
		return false
	}
	return strings.HasSuffix(name, "_servicemanager.go") || strings.HasSuffix(name, "_serviceclient.go")
}

func serviceManagerRoots(services []generator.ServiceConfig) []string {
	seen := make(map[string]struct{}, len(services))
	for _, service := range services {
		seen[service.Group] = struct{}{}
		for _, override := range service.Generation.Resources {
			packagePath := strings.TrimSpace(override.ServiceManager.PackagePath)
			if packagePath == "" {
				continue
			}
			root := strings.Split(filepath.ToSlash(packagePath), "/")[0]
			if strings.TrimSpace(root) == "" {
				continue
			}
			seen[root] = struct{}{}
		}
	}

	roots := make([]string, 0, len(seen))
	for root := range seen {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	return roots
}

func collectRegistrationPackages(snapshotRoot string) ([]string, error) {
	registrationsRoot := filepath.Join(snapshotRoot, "internal", "registrations")
	entries, err := os.ReadDir(registrationsRoot)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || !strings.HasSuffix(entry.Name(), "_generated.go") {
			continue
		}
		return []string{"./internal/registrations"}, nil
	}
	return nil, nil
}

func preserveCheckedInCompanionFiles(repoRoot, snapshotRoot string, packages []*generator.PackageModel) error {
	for _, pkg := range packages {
		if err := preserveRuntimeCheckPackageCompanionFiles(repoRoot, snapshotRoot, pkg); err != nil {
			return err
		}
	}
	for root, generatedPaths := range runtimeCheckGeneratedServiceManagerPaths(packages, snapshotRoot) {
		if err := preserveServiceManagerCompanionGoFiles(
			filepath.Join(repoRoot, "pkg", "servicemanager", root),
			filepath.Join(snapshotRoot, "pkg", "servicemanager", root),
			generatedPaths,
		); err != nil {
			return fmt.Errorf("preserve service-manager companion files for %s: %w", root, err)
		}
	}
	return nil
}

func preserveRuntimeCheckPackageCompanionFiles(repoRoot, snapshotRoot string, pkg *generator.PackageModel) error {
	version := pkg.Version
	if err := preserveCompanionGoFiles(
		filepath.Join(repoRoot, "api", pkg.Service.Group, version),
		filepath.Join(snapshotRoot, "api", pkg.Service.Group, version),
		isGeneratedAPIFile,
	); err != nil {
		return fmt.Errorf("preserve api companion files for %s/%s: %w", pkg.Service.Group, version, err)
	}
	if err := preserveCompanionGoFiles(
		filepath.Join(repoRoot, "controllers", pkg.Service.Group),
		filepath.Join(snapshotRoot, "controllers", pkg.Service.Group),
		isGeneratedControllerFile,
	); err != nil {
		return fmt.Errorf("preserve controller companion files for %s: %w", pkg.Service.Group, err)
	}
	return nil
}

func runtimeCheckGeneratedServiceManagerPaths(packages []*generator.PackageModel, snapshotRoot string) map[string]map[string]struct{} {
	pathsByRoot := make(map[string]map[string]struct{})
	for _, pkg := range packages {
		for _, serviceManager := range pkg.ServiceManagers {
			root := strings.Split(filepath.ToSlash(serviceManager.PackagePath), "/")[0]
			if strings.TrimSpace(root) == "" {
				continue
			}
			if pathsByRoot[root] == nil {
				pathsByRoot[root] = make(map[string]struct{})
			}
			serviceManagerDir := filepath.Join(snapshotRoot, "pkg", "servicemanager", filepath.FromSlash(serviceManager.PackagePath))
			pathsByRoot[root][filepath.Join(serviceManagerDir, serviceManager.ServiceClientFileName)] = struct{}{}
			pathsByRoot[root][filepath.Join(serviceManagerDir, serviceManager.ServiceManagerFileName)] = struct{}{}
		}
	}
	return pathsByRoot
}

func preserveCompanionGoFiles(sourceRoot, destRoot string, isGenerated func(string) bool) error {
	return preserveRuntimeCheckCompanionGoFiles(sourceRoot, destRoot, func(destPath string, entry fs.DirEntry) bool {
		return filepath.Ext(entry.Name()) == ".go" && !isGenerated(entry.Name())
	})
}

func preserveServiceManagerCompanionGoFiles(sourceRoot, destRoot string, generatedPaths map[string]struct{}) error {
	return preserveRuntimeCheckCompanionGoFiles(sourceRoot, destRoot, func(destPath string, entry fs.DirEntry) bool {
		if filepath.Ext(entry.Name()) != ".go" {
			return false
		}
		_, generated := generatedPaths[destPath]
		return !generated
	})
}

func preserveRuntimeCheckCompanionGoFiles(
	sourceRoot, destRoot string,
	shouldLink func(destPath string, entry fs.DirEntry) bool,
) error {
	if _, err := os.Stat(sourceRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		destPath, err := runtimeCheckCompanionDestPath(sourceRoot, destRoot, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}
		if !shouldLink(destPath, entry) {
			return nil
		}
		return runtimeCheckSymlinkIfMissing(path, destPath)
	})
}

func runtimeCheckCompanionDestPath(sourceRoot, destRoot, path string) (string, error) {
	rel, err := filepath.Rel(sourceRoot, path)
	if err != nil {
		return "", err
	}
	return filepath.Join(destRoot, rel), nil
}

func runtimeCheckSymlinkIfMissing(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return symlink(src, dst)
}

func isGeneratedAPIFile(name string) bool {
	return name == "groupversion_info.go" ||
		name == "zz_generated.deepcopy.go" ||
		strings.HasSuffix(name, "_types.go")
}

func isGeneratedControllerFile(name string) bool {
	return strings.HasSuffix(name, "_controller.go")
}

func shouldSkipGeneratedRegistration(name string, selected map[string]struct{}) bool {
	if filepath.Ext(name) != ".go" || !strings.HasSuffix(name, "_generated.go") {
		return false
	}
	group := strings.TrimSuffix(name, "_generated.go")
	_, ok := selected[group]
	return ok
}

func compilePackageSet(dir string, env []string, label string, packages []string) error {
	if len(packages) == 0 {
		return nil
	}

	args := append([]string{"test", "-run", "^$"}, packages...)
	if err := runCommand(dir, env, "go", args...); err != nil {
		return fmt.Errorf("compile %s: %w", label, err)
	}
	return nil
}

func symlink(src, dst string) error {
	return os.Symlink(src, dst)
}

func symlinkIfPresent(src, dst string) error {
	if _, err := os.Lstat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return symlink(src, dst)
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

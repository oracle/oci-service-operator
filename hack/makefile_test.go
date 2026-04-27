package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMakeGenerateUsesScopedDeepcopyPaths(t *testing.T) {
	output := runMakeDryRun(t, "generate", nil)

	if !strings.Contains(output, `paths="./api/...;./pkg/shared"`) {
		t.Fatalf("make -n generate output did not contain scoped deepcopy paths:\n%s", output)
	}
	if strings.Contains(output, `paths="./..."`) {
		t.Fatalf("make -n generate output still contains a blanket ./... package walk:\n%s", output)
	}
}

func TestMakeGenerateUsesControllerGenCompatibilityRunner(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	output := runMakeDryRun(t, "generate", nil)
	expected := filepath.Join(root, "hack", "with-controller-gen-godebug.sh")
	if !strings.Contains(output, expected) {
		t.Fatalf("make -n generate output did not invoke the controller-gen compatibility runner %q:\n%s", expected, output)
	}
}

func TestMakeManifestsUsesControllerGenCompatibilityRunner(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	output := runMakeDryRun(t, "manifests", nil)
	expected := filepath.Join(root, "hack", "with-controller-gen-godebug.sh")
	if !strings.Contains(output, expected) {
		t.Fatalf("make -n manifests output did not invoke the controller-gen compatibility runner %q:\n%s", expected, output)
	}
}

func TestMakeGeneratedCoverageUsesControllerGenCompatibilityRunner(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	output := runMakeDryRun(t, "generated-coverage-report", nil)
	expected := filepath.Join(root, "hack", "with-controller-gen-godebug.sh")
	if !strings.Contains(output, expected) {
		t.Fatalf("make -n generated-coverage-report output did not invoke the controller-gen compatibility runner %q:\n%s", expected, output)
	}
}

func TestMakeSchemaValidatorSupportedSelectionUsesEffectiveGeneratorConfig(t *testing.T) {
	output := runMakeDryRun(t, "schema-validator", []string{
		"SCHEMA_VALIDATOR_SELECTION=supported",
	})

	if !strings.Contains(output, "go run ./cmd/osok-schema-validator --provider-path . --config internal/generator/config/services.yaml --all") {
		t.Fatalf("make -n schema-validator output did not invoke cmd/osok-schema-validator with the effective generator config and default-active selection:\n%s", output)
	}
	if !strings.Contains(output, "--format json > \"$tmp_report\"") {
		t.Fatalf("make -n schema-validator output did not keep the default coverage format at json:\n%s", output)
	}
}

func TestMakeSchemaValidatorUpgradeUsesEffectiveGeneratorConfig(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "validator-root")
	output := runMakeDryRun(t, "schema-validator", []string{
		"TMPDIR=" + tmpDir,
		"SCHEMA_VALIDATOR_UPGRADE_FROM=v65.61.1",
		"SCHEMA_VALIDATOR_UPGRADE_TO=v65.104.0",
	})

	expectedReport := filepath.Join(tmpDir, "validator-upgrade-report.md")
	if !strings.Contains(output, "go run ./cmd/osok-schema-validator --provider-path . --config internal/generator/config/services.yaml --all") {
		t.Fatalf("make -n schema-validator output did not scope the upgrade run to the effective generator config:\n%s", output)
	}
	if !strings.Contains(output, "--upgrade-from v65.61.1 --upgrade-to v65.104.0 --format markdown > \"$tmp_report\"") {
		t.Fatalf("make -n schema-validator output did not invoke cmd/osok-schema-validator with the effective generator config and default-active selection:\n%s", output)
	}
	if !strings.Contains(output, "report=\""+expectedReport+"\"") {
		t.Fatalf("make -n schema-validator output did not use the temp-based default report path %q:\n%s", expectedReport, output)
	}
	if strings.Contains(output, filepath.Join(findRepoRootForTest(t), "validator-upgrade-report.md")) {
		t.Fatalf("make -n schema-validator output still points the default report into the repo root:\n%s", output)
	}
}

func TestMakeFormalScaffoldUsesEffectiveGeneratorConfig(t *testing.T) {
	output := runMakeDryRun(t, "formal-scaffold", nil)

	if !strings.Contains(output, "go run ./cmd/formal-scaffold --root formal --config internal/generator/config/services.yaml") {
		t.Fatalf("make -n formal-scaffold output did not invoke cmd/formal-scaffold with the effective generator config:\n%s", output)
	}
}

func TestMakeFormalDiagramsUsesFormalRoot(t *testing.T) {
	output := runMakeDryRun(t, "formal-diagrams", nil)

	if !strings.Contains(output, "go run ./cmd/formal-diagrams --root formal") {
		t.Fatalf("make -n formal-diagrams output did not invoke cmd/formal-diagrams with the formal root:\n%s", output)
	}
}

func TestMakeFormalScaffoldVerifyUsesProviderPath(t *testing.T) {
	output := runMakeDryRun(t, "formal-scaffold-verify", []string{"FORMAL_PROVIDER_PATH=/tmp/provider"})

	if !strings.Contains(output, "go run ./cmd/formal-scaffold-verify --root formal --config internal/generator/config/services.yaml --provider-path /tmp/provider") {
		t.Fatalf("make -n formal-scaffold-verify output did not invoke cmd/formal-scaffold-verify with the provider path:\n%s", output)
	}
}

func TestMakefileDoesNotExposeLegacyGeneratorAliases(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error = %v", err)
	}

	rendered := string(content)
	for _, forbidden := range []string{
		"API_GENERATOR_CONFIG ?=",
		"API_GENERATOR_OUTPUT_ROOT ?=",
		"API_SERVICE ?=",
		"API_ALL ?=",
		"API_OVERWRITE ?=",
		"API_PRESERVE_EXISTING_SPEC_SURFACE ?=",
		"api-generate:",
		"api-refresh:",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("Makefile still exposes legacy generator alias %q:\n%s", forbidden, rendered)
		}
	}
}

func TestMakeTestKeepsEnvtestOutsideRepoByDefault(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	tmpDir := filepath.Join(t.TempDir(), "envtest-root")
	output := runMakeDryRun(t, "test", []string{"TMPDIR=" + tmpDir})

	if strings.Contains(output, filepath.Join(root, ".envtest-home", "home")) {
		t.Fatalf("make -n test output still points ENVTEST_HOME into the repo:\n%s", output)
	}
	if strings.Contains(output, filepath.Join(root, "testbin")) {
		t.Fatalf("make -n test output still points ENVTEST_ASSETS_DIR into the repo:\n%s", output)
	}

	expectedRoot := filepath.Join(tmpDir, "oci-service-operator-envtest")
	if !strings.Contains(output, filepath.Join(expectedRoot, "home")) {
		t.Fatalf("make -n test output did not use temp-based ENVTEST_HOME %q:\n%s", filepath.Join(expectedRoot, "home"), output)
	}
	if !strings.Contains(output, filepath.Join(expectedRoot, "testbin")) {
		t.Fatalf("make -n test output did not use temp-based ENVTEST_ASSETS_DIR under %q:\n%s", filepath.Join(expectedRoot, "testbin"), output)
	}
}

func TestMakeTestSetupEnvtestUsesIsolatedGoPath(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "envtest-root")
	output := runMakeDryRun(t, "test", []string{"TMPDIR=" + tmpDir})

	if !strings.Contains(output, "env -u GOMODCACHE") {
		t.Fatalf("make -n test output did not unset inherited GOMODCACHE for setup-envtest:\n%s", output)
	}

	expectedGoPath := filepath.Join(tmpDir, "oci-service-operator-envtest", "gopath")
	if !strings.Contains(output, "GOPATH="+expectedGoPath) {
		t.Fatalf("make -n test output did not use isolated setup-envtest GOPATH %q:\n%s", expectedGoPath, output)
	}
}

func TestMakeTestPrefersPreseededEnvtestAssets(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "envtest-root")
	output := runMakeDryRun(t, "test", []string{"TMPDIR=" + tmpDir})

	expectedAssetsRoot := filepath.Join(tmpDir, "oci-service-operator-envtest", "testbin", runtime.GOOS+"-"+runtime.GOARCH)
	if !strings.Contains(output, filepath.Join(expectedAssetsRoot, "kube-apiserver")) {
		t.Fatalf("make -n test output did not check for a preseeded kube-apiserver in %q:\n%s", expectedAssetsRoot, output)
	}
	if !strings.Contains(output, filepath.Join(expectedAssetsRoot, "etcd")) {
		t.Fatalf("make -n test output did not check for a preseeded etcd in %q:\n%s", expectedAssetsRoot, output)
	}
}

func TestMakeTestDocumentsPreseedAndEnvOverrides(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "envtest-root")
	output := runMakeDryRun(t, "test", []string{"TMPDIR=" + tmpDir})

	if !strings.Contains(output, "Run make envtest while network access is available") {
		t.Fatalf("make -n test output did not explain how to preseed envtest assets:\n%s", output)
	}
	if !strings.Contains(output, "ENVTEST_USE_ENV=true") || !strings.Contains(output, "KUBEBUILDER_ASSETS=/path") {
		t.Fatalf("make -n test output did not explain how to reuse an external envtest asset bundle:\n%s", output)
	}
}

func TestMakeEnvtestUsesTempBasedEnvtestRoot(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	tmpDir := filepath.Join(t.TempDir(), "envtest-root")
	output := runMakeDryRun(t, "envtest", []string{"TMPDIR=" + tmpDir})
	expectedRoot := filepath.Join(tmpDir, "oci-service-operator-envtest")

	if !strings.Contains(output, filepath.Join(expectedRoot, "testbin")) {
		t.Fatalf("make -n envtest output did not use temp-based ENVTEST_ASSETS_DIR under %q:\n%s", filepath.Join(expectedRoot, "testbin"), output)
	}
	if !strings.Contains(output, filepath.Join(expectedRoot, "gopath")) {
		t.Fatalf("make -n envtest output did not use temp-based setup-envtest GOPATH %q:\n%s", filepath.Join(expectedRoot, "gopath"), output)
	}
	if !strings.Contains(output, "setup-envtest@v0.0.0-20240812162837-9557f1031fe4") {
		t.Fatalf("make -n envtest output did not use the pinned setup-envtest revision:\n%s", output)
	}
	if !strings.Contains(output, filepath.Join(root, ".envtest-home", ".gomodcache")) {
		t.Fatalf("make -n envtest output did not clean the legacy repo-local envtest GOMODCACHE path:\n%s", output)
	}
}

func runMakeDryRun(t *testing.T, target string, extraEnv []string) string {
	t.Helper()

	root := findRepoRootForTest(t)

	cmd := exec.Command("make", "-n", target)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), extraEnv...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n %s failed: %v\n%s", target, err, output)
	}

	return string(output)
}

func findRepoRootForTest(t *testing.T) string {
	t.Helper()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}
	return root
}

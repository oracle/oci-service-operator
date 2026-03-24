package main

import (
	"os"
	"os/exec"
	"path/filepath"
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

func TestMakeTestKeepsEnvtestOutsideRepoByDefault(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	tmpDir := filepath.Join(t.TempDir(), "envtest-root")
	output := runMakeDryRun(t, "test", []string{"TMPDIR=" + tmpDir})

	if strings.Contains(output, filepath.Join(root, ".envtest-home")) {
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

func runMakeDryRun(t *testing.T, target string, extraEnv []string) string {
	t.Helper()

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	cmd := exec.Command("make", "-n", target)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), extraEnv...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n %s failed: %v\n%s", target, err, output)
	}

	return string(output)
}

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

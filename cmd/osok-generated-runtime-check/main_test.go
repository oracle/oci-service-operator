package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/oracle/oci-service-operator/internal/generator"
)

func TestCollectBuildPlanFindsGeneratedRuntimePackages(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "controllers", "functions"), 0o755); err != nil {
		t.Fatalf("MkdirAll(controllers) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "pkg", "servicemanager", "functions", "application"), 0o755); err != nil {
		t.Fatalf("MkdirAll(service manager) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "registrations"), 0o755); err != nil {
		t.Fatalf("MkdirAll(registrations) error = %v", err)
	}

	for path, content := range map[string]string{
		filepath.Join(root, "controllers", "functions", "application_controller.go"):                              "package functions\n",
		filepath.Join(root, "pkg", "servicemanager", "functions", "application", "application_servicemanager.go"): "package application\n",
		filepath.Join(root, "pkg", "servicemanager", "functions", "application", "application_serviceclient.go"):  "package application\n",
		filepath.Join(root, "internal", "registrations", "functions_generated.go"):                                "package registrations\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}

	build, err := collectBuildPlan(root, []string{"functions"})
	if err != nil {
		t.Fatalf("collectBuildPlan() error = %v", err)
	}

	if !slices.Equal(build.ControllerPackages, []string{"./controllers/functions"}) {
		t.Fatalf("ControllerPackages = %v, want %v", build.ControllerPackages, []string{"./controllers/functions"})
	}
	if !slices.Equal(build.ServiceManagerPackages, []string{"./pkg/servicemanager/functions/application"}) {
		t.Fatalf("ServiceManagerPackages = %v, want %v", build.ServiceManagerPackages, []string{"./pkg/servicemanager/functions/application"})
	}
	if !slices.Equal(build.RegistrationPackages, []string{"./internal/registrations"}) {
		t.Fatalf("RegistrationPackages = %v, want %v", build.RegistrationPackages, []string{"./internal/registrations"})
	}
}

func TestCollectBuildPlanRejectsMissingRuntimePackages(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "registrations"), 0o755); err != nil {
		t.Fatalf("MkdirAll(registrations) error = %v", err)
	}

	if _, err := collectBuildPlan(root, []string{"functions"}); err == nil {
		t.Fatal("collectBuildPlan() error = nil, want missing package error")
	}
}

func TestPreserveManualWebhookFilesLinksSelectedWebhookFiles(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	sourceDir := filepath.Join(repoRoot, "api", "database", "v1beta1")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", sourceDir, err)
	}

	webhookPath := filepath.Join(sourceDir, "autonomousdatabases_webhook.go")
	if err := os.WriteFile(webhookPath, []byte("package v1beta1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", webhookPath, err)
	}
	typesPath := filepath.Join(sourceDir, "autonomousdatabases_types.go")
	if err := os.WriteFile(typesPath, []byte("package v1beta1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", typesPath, err)
	}

	services := []generator.ServiceConfig{
		{Group: "database"},
	}
	if err := preserveManualWebhookFiles(repoRoot, snapshotRoot, "v1beta1", services); err != nil {
		t.Fatalf("preserveManualWebhookFiles() error = %v", err)
	}

	snapshotWebhookPath := filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabases_webhook.go")
	info, err := os.Lstat(snapshotWebhookPath)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", snapshotWebhookPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", snapshotWebhookPath, info.Mode())
	}

	target, err := os.Readlink(snapshotWebhookPath)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", snapshotWebhookPath, err)
	}
	if target != webhookPath {
		t.Fatalf("Readlink(%q) = %q, want %q", snapshotWebhookPath, target, webhookPath)
	}

	snapshotTypesPath := filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabases_types.go")
	if _, err := os.Stat(snapshotTypesPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", snapshotTypesPath, err)
	}
}

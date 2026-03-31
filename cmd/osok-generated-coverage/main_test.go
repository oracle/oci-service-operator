package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oracle/oci-service-operator/internal/generator"
)

func TestPopulateSnapshotKeepsSelectedOutputsWritable(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	for _, dir := range []string{
		filepath.Join(repoRoot, "api"),
		filepath.Join(repoRoot, "cmd"),
		filepath.Join(repoRoot, "controllers"),
		filepath.Join(repoRoot, "hack"),
		filepath.Join(repoRoot, "formal"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "database"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "identity"),
		filepath.Join(repoRoot, "internal", "registrations"),
		filepath.Join(repoRoot, "internal", "validator"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
	}
	for path, content := range map[string]string{
		filepath.Join(repoRoot, "go.mod"):                                             "module example.com/test\n",
		filepath.Join(repoRoot, "go.sum"):                                             "",
		filepath.Join(repoRoot, "validator_allowlist.yaml"):                           "{}\n",
		filepath.Join(repoRoot, "cmd", "main.go"):                                     "package main\n",
		filepath.Join(repoRoot, "internal", "validator", "doc.go"):                    "package validator\n",
		filepath.Join(repoRoot, "internal", "registrations", "database_generated.go"): "package registrations\n",
		filepath.Join(repoRoot, "internal", "registrations", "events_generated.go"):   "package registrations\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}

	if err := populateSnapshot(repoRoot, snapshotRoot, []string{"database"}, []string{"database"}); err != nil {
		t.Fatalf("populateSnapshot() error = %v", err)
	}

	formalPath := filepath.Join(snapshotRoot, "formal")
	formalInfo, err := os.Lstat(formalPath)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", formalPath, err)
	}
	if formalInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", formalPath, formalInfo.Mode())
	}

	if _, err := os.Lstat(filepath.Join(snapshotRoot, "internal", "registrations", "database_generated.go")); !os.IsNotExist(err) {
		t.Fatalf("Lstat(selected registration) error = %v, want not exist", err)
	}
	eventsPath := filepath.Join(snapshotRoot, "internal", "registrations", "events_generated.go")
	eventsInfo, err := os.Lstat(eventsPath)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", eventsPath, err)
	}
	if eventsInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", eventsPath, eventsInfo.Mode())
	}

	if _, err := os.Lstat(filepath.Join(snapshotRoot, "pkg", "servicemanager", "database")); !os.IsNotExist(err) {
		t.Fatalf("Lstat(selected service-manager root) error = %v, want not exist", err)
	}
	identityPath := filepath.Join(snapshotRoot, "pkg", "servicemanager", "identity")
	identityInfo, err := os.Lstat(identityPath)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", identityPath, err)
	}
	if identityInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", identityPath, identityInfo.Mode())
	}
}

func TestPreserveCheckedInCompanionFilesLinksManualCompanions(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	apiSourceDir := filepath.Join(repoRoot, "api", "database", "v1beta1")
	if err := os.MkdirAll(apiSourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", apiSourceDir, err)
	}
	webhookPath := filepath.Join(apiSourceDir, "autonomousdatabase_webhook.go")
	if err := os.WriteFile(webhookPath, []byte("package v1beta1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", webhookPath, err)
	}
	typesPath := filepath.Join(apiSourceDir, "autonomousdatabase_types.go")
	if err := os.WriteFile(typesPath, []byte("package v1beta1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", typesPath, err)
	}

	serviceManagerSourceDir := filepath.Join(repoRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	if err := os.MkdirAll(serviceManagerSourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", serviceManagerSourceDir, err)
	}
	manualHelperPath := filepath.Join(serviceManagerSourceDir, "manual_helper.go")
	if err := os.WriteFile(manualHelperPath, []byte("package autonomousdatabase\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", manualHelperPath, err)
	}
	generatedServiceClientPath := filepath.Join(serviceManagerSourceDir, "autonomousdatabase_serviceclient.go")
	if err := os.WriteFile(generatedServiceClientPath, []byte("package autonomousdatabase\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", generatedServiceClientPath, err)
	}

	snapshotServiceManagerDir := filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	if err := os.MkdirAll(snapshotServiceManagerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", snapshotServiceManagerDir, err)
	}
	snapshotGeneratedServiceClientPath := filepath.Join(snapshotServiceManagerDir, "autonomousdatabase_serviceclient.go")
	if err := os.WriteFile(snapshotGeneratedServiceClientPath, []byte("package autonomousdatabase\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", snapshotGeneratedServiceClientPath, err)
	}

	services := []generator.ServiceConfig{
		{
			Group: "database",
		},
	}
	if err := preserveCheckedInCompanionFiles(repoRoot, snapshotRoot, "v1beta1", services); err != nil {
		t.Fatalf("preserveCheckedInCompanionFiles() error = %v", err)
	}

	snapshotWebhookPath := filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabase_webhook.go")
	webhookInfo, err := os.Lstat(snapshotWebhookPath)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", snapshotWebhookPath, err)
	}
	if webhookInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", snapshotWebhookPath, webhookInfo.Mode())
	}

	snapshotManualHelperPath := filepath.Join(snapshotServiceManagerDir, "manual_helper.go")
	manualHelperInfo, err := os.Lstat(snapshotManualHelperPath)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", snapshotManualHelperPath, err)
	}
	if manualHelperInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", snapshotManualHelperPath, manualHelperInfo.Mode())
	}

	generatedInfo, err := os.Lstat(snapshotGeneratedServiceClientPath)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", snapshotGeneratedServiceClientPath, err)
	}
	if generatedInfo.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%q mode = %v, want regular file", snapshotGeneratedServiceClientPath, generatedInfo.Mode())
	}

	snapshotTypesPath := filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabase_types.go")
	if _, err := os.Stat(snapshotTypesPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", snapshotTypesPath, err)
	}
}

package crdsync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncFileRefreshesSharedCRDResources(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	basesDir := filepath.Join(root, "config", "crd", "bases")
	if err := os.MkdirAll(basesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	for _, name := range []string{
		"zeta.oracle.com_examples.yaml",
		"alpha.oracle.com_examples.yaml",
		"kustomization.yaml",
		"ignored.txt",
	} {
		if err := os.WriteFile(filepath.Join(basesDir, name), []byte("kind: test\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(basesDir, "nested"), 0o755); err != nil {
		t.Fatalf("Mkdir(nested) error = %v", err)
	}

	kustomizationPath := filepath.Join(root, "config", "crd", "kustomization.yaml")
	input := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- bases/stale.yaml
# +kubebuilder:scaffold:crdkustomizeresource

# patches:
- patches/webhook_in_custom.yaml
configurations:
- kustomizeconfig.yaml
`
	if err := os.WriteFile(kustomizationPath, []byte(input), 0o644); err != nil {
		t.Fatalf("WriteFile(kustomization.yaml) error = %v", err)
	}

	if err := SyncFile(kustomizationPath, basesDir); err != nil {
		t.Fatalf("SyncFile() error = %v", err)
	}

	got, err := os.ReadFile(kustomizationPath)
	if err != nil {
		t.Fatalf("ReadFile(kustomization.yaml) error = %v", err)
	}

	want := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- bases/alpha.oracle.com_examples.yaml
- bases/zeta.oracle.com_examples.yaml
# +kubebuilder:scaffold:crdkustomizeresource

# patches:
- patches/webhook_in_custom.yaml
configurations:
- kustomizeconfig.yaml
`
	if string(got) != want {
		t.Fatalf("SyncFile() mismatch (-want +got):\nwant:\n%s\ngot:\n%s", want, string(got))
	}
}

func TestSyncResourcesBlockRequiresScaffoldMarker(t *testing.T) {
	t.Helper()

	_, err := syncResourcesBlock([]byte("resources:\n- bases/example.yaml\n"), []string{"bases/next.yaml"})
	if err == nil {
		t.Fatal("syncResourcesBlock() error = nil, want scaffold marker error")
	}
}

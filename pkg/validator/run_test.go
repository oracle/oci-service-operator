package validator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/validator"
	"github.com/oracle/oci-service-operator/pkg/validator/config"
)

func moduleRoot(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("unable to locate module root from %s", dir)
		}
		dir = parent
	}
}

func TestRunGeneratesReport(t *testing.T) {
	opts := config.Options{ProviderPath: moduleRoot(t)}
	result, rendered, err := validator.Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Structs) == 0 {
		t.Fatalf("expected struct reports")
	}
	if rendered == "" {
		t.Fatalf("expected rendered output")
	}
}

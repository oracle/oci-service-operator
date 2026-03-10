package provider_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/validator/provider"
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

func TestAnalyzerFindsAutonomousDatabaseUsage(t *testing.T) {
	analyzer := provider.NewAnalyzer(moduleRoot(t))
	analysis, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	found := false
	for _, usage := range analysis.Usages {
		if usage.StructType == "database.CreateAutonomousDatabaseDetails" && usage.FieldName == "CompartmentId" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find CreateAutonomousDatabaseDetails.CompartmentId usage")
	}
}

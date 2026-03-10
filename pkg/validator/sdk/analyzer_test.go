package sdk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/validator/sdk"
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

func TestAnalyzerIncludesCreateAutonomousDatabaseDetails(t *testing.T) {
	analyzer, err := sdk.NewAnalyzer(moduleRoot(t))
	if err != nil {
		t.Fatalf("NewAnalyzer: %v", err)
	}
	structs, err := analyzer.AnalyzeAll()
	if err != nil {
		t.Fatalf("AnalyzeAll: %v", err)
	}
	found := false
	for _, strct := range structs {
		if strct.QualifiedName == "database.CreateAutonomousDatabaseDetails" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected CreateAutonomousDatabaseDetails to be analyzed")
	}
}

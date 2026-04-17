package sdk_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/oracle/oci-service-operator/internal/validator/sdk"
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

func TestAnalyzerExpandsDatabasetoolsInterfaceImplementations(t *testing.T) {
	analyzer, err := sdk.NewAnalyzer(moduleRoot(t))
	if err != nil {
		t.Fatalf("NewAnalyzer: %v", err)
	}
	structs, err := analyzer.AnalyzeAll()
	if err != nil {
		t.Fatalf("AnalyzeAll: %v", err)
	}

	var target *sdk.SDKStruct
	for i := range structs {
		if structs[i].QualifiedName == "databasetools.CreateDatabaseToolsConnectionOracleDatabaseDetails" {
			target = &structs[i]
			break
		}
	}
	if target == nil {
		t.Fatal("expected databasetools.CreateDatabaseToolsConnectionOracleDatabaseDetails to be analyzed")
	}

	userPassword := findSDKField(t, target.Fields, "UserPassword")
	if got, want := implementationNames(userPassword), []string{"databasetools.DatabaseToolsUserPasswordSecretIdDetails"}; !slices.Equal(got, want) {
		t.Fatalf("UserPassword implementations = %v, want %v", got, want)
	}

	proxyClient := findSDKField(t, target.Fields, "ProxyClient")
	if got, want := implementationNames(proxyClient), []string{
		"databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientNoProxyDetails",
		"databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientUserNameDetails",
	}; !slices.Equal(got, want) {
		t.Fatalf("ProxyClient implementations = %v, want %v", got, want)
	}

	if got, want := userPassword.Type, "databasetools.DatabaseToolsUserPasswordDetails"; got != want {
		t.Fatalf("UserPassword type = %q, want %q", got, want)
	}
	if got, want := proxyClient.Type, "databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientDetails"; got != want {
		t.Fatalf("ProxyClient type = %q, want %q", got, want)
	}
}

func findSDKField(t *testing.T, fields []sdk.SDKField, name string) sdk.SDKField {
	t.Helper()

	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}
	t.Fatalf("field %q not found", name)
	return sdk.SDKField{}
}

func implementationNames(field sdk.SDKField) []string {
	names := make([]string, 0, len(field.InterfaceImplementations))
	for _, impl := range field.InterfaceImplementations {
		names = append(names, impl.QualifiedName)
	}
	slices.Sort(names)
	return names
}

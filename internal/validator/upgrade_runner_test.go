package validator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oracle/oci-service-operator/internal/validator/config"
	upgradepkg "github.com/oracle/oci-service-operator/internal/validator/upgrade"
)

func TestLoadUpgradeSelectedStructsDefaultsBlankRunToDefaultActiveSurface(t *testing.T) {
	t.Helper()

	configPath := writeUpgradeSelectionConfig(t)
	selected, err := loadUpgradeSelectedStructs(config.Options{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("loadUpgradeSelectedStructs() error = %v", err)
	}

	assertUpgradeStructSelected(t, selected, "core.UpdateInstanceDetails")
	assertUpgradeStructSelected(t, selected, "core.Instance")
	assertUpgradeStructSelected(t, selected, "core.CreateVcnDetails")
	assertUpgradeStructSelected(t, selected, "core.Vcn")
	assertUpgradeStructSelected(t, selected, "mysql.CreateDbSystemDetails")
	assertUpgradeStructSelected(t, selected, "mysql.DbSystem")

	assertUpgradeStructNotSelected(t, selected, "core.CreateSubnetDetails")
	assertUpgradeStructNotSelected(t, selected, "core.Subnet")
	assertUpgradeStructNotSelected(t, selected, "identity.CreateCompartmentDetails")
	assertUpgradeStructNotSelected(t, selected, "identity.Compartment")
}

func TestLoadUpgradeSelectedStructsAllowsExplicitServiceScope(t *testing.T) {
	t.Helper()

	configPath := writeUpgradeSelectionConfig(t)
	selected, err := loadUpgradeSelectedStructs(config.Options{
		ConfigPath: configPath,
		Service:    "identity",
	})
	if err != nil {
		t.Fatalf("loadUpgradeSelectedStructs(identity) error = %v", err)
	}

	assertUpgradeStructSelected(t, selected, "identity.CreateCompartmentDetails")
	assertUpgradeStructSelected(t, selected, "identity.Compartment")
	assertUpgradeStructNotSelected(t, selected, "mysql.CreateDbSystemDetails")
	assertUpgradeStructNotSelected(t, selected, "core.UpdateInstanceDetails")
}

func TestLoadUpgradeSelectedStructsDefaultsConfigPathFromProviderPath(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	configPath := filepath.Join(root, "internal", "generator", "config", "services.yaml")
	writeUpgradeSelectionConfigAtPath(t, configPath)

	selected, err := loadUpgradeSelectedStructs(config.Options{
		ProviderPath: root,
		All:          true,
	})
	if err != nil {
		t.Fatalf("loadUpgradeSelectedStructs(provider-root default config) error = %v", err)
	}

	assertUpgradeStructSelected(t, selected, "core.UpdateInstanceDetails")
	assertUpgradeStructSelected(t, selected, "mysql.CreateDbSystemDetails")
	assertUpgradeStructNotSelected(t, selected, "identity.Compartment")
}

func TestFilterUpgradeReportByStructsFiltersAllowlistSuggestions(t *testing.T) {
	report := upgradepkg.Report{
		Structs: []upgradepkg.StructDiff{
			{StructType: "core.Instance"},
			{StructType: "core.Subnet"},
		},
		AllowlistSuggestions: []upgradepkg.AllowlistSuggestion{
			{Path: "core.Instance.fields.DisplayName"},
			{Path: "core.Subnet.fields.CidrBlock"},
		},
	}

	filtered := filterUpgradeReportByStructs(report, map[string]struct{}{
		"core.Instance": {},
	})
	if len(filtered.Structs) != 1 {
		t.Fatalf("filterUpgradeReportByStructs() struct count = %d, want 1", len(filtered.Structs))
	}
	if filtered.Structs[0].StructType != "core.Instance" {
		t.Fatalf("filterUpgradeReportByStructs() kept struct %q, want %q", filtered.Structs[0].StructType, "core.Instance")
	}
	if len(filtered.AllowlistSuggestions) != 1 {
		t.Fatalf("filterUpgradeReportByStructs() suggestion count = %d, want 1", len(filtered.AllowlistSuggestions))
	}
	if filtered.AllowlistSuggestions[0].Path != "core.Instance.fields.DisplayName" {
		t.Fatalf("filterUpgradeReportByStructs() kept suggestion %q, want %q", filtered.AllowlistSuggestions[0].Path, "core.Instance.fields.DisplayName")
	}
}

func writeUpgradeSelectionConfig(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	configPath := filepath.Join(root, "services.yaml")
	writeUpgradeSelectionConfigAtPath(t, configPath)
	return configPath
}

func writeUpgradeSelectionConfigAtPath(t *testing.T, configPath string) {
	t.Helper()

	const selectionConfig = `schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
services:
  - service: core
    sdkPackage: example.com/core
    group: core
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - Instance
        - Vcn
    async:
      strategy: lifecycle
      runtime: generatedruntime
      formalClassification: lifecycle
  - service: mysql
    sdkPackage: example.com/mysql
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: all
  - service: identity
    sdkPackage: example.com/identity
    group: identity
    packageProfile: controller-backed
    selection:
      enabled: false
      mode: all
`

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(configPath), err)
	}
	if err := os.WriteFile(configPath, []byte(selectionConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", configPath, err)
	}
}

func assertUpgradeStructSelected(t *testing.T, selected map[string]struct{}, name string) {
	t.Helper()

	if _, ok := selected[name]; !ok {
		t.Fatalf("selected upgrade structs did not contain %q", name)
	}
}

func assertUpgradeStructNotSelected(t *testing.T, selected map[string]struct{}, name string) {
	t.Helper()

	if _, ok := selected[name]; ok {
		t.Fatalf("selected upgrade structs unexpectedly contained %q", name)
	}
}

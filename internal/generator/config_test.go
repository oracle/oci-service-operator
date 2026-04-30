/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadConfigRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "services.yaml")
	content := `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: manual controllers
services:
  - service: mysql
    sdkPackage: github.com/oracle/oci-go-sdk/v65/mysql
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: false
      mode: all
    unknownField: nope
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "unknownField") {
		t.Fatalf("LoadConfig() error = %v, want unknownField failure", err)
	}
}

func TestLoadConfigRejectsBlankObservedStateExcludedFieldPath(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "services.yaml")
	content := `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: manual controllers
services:
  - service: mysql
    sdkPackage: github.com/oracle/oci-go-sdk/v65/mysql
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: false
      mode: all
    observedState:
      excludedFieldPaths:
        DbSystem:
          - Source..SourceUrl
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), `observedState excludedFieldPaths["DbSystem"]`) {
		t.Fatalf("LoadConfig() error = %v, want excludedFieldPaths failure", err)
	}
}

func TestSelectServices(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "manual controllers"},
		},
		Services: []ServiceConfig{
			{
				Service:        "database",
				SDKPackage:     "example/database",
				Group:          "database",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "AutonomousDatabase"),
			},
			{
				Service:        "mysql",
				SDKPackage:     "example/mysql",
				Group:          "mysql",
				PackageProfile: "controller-backed",
				Selection:      selectionAll(true),
			},
			{
				Service:        "identity",
				SDKPackage:     "example/identity",
				Group:          "identity",
				PackageProfile: "controller-backed",
				Selection:      selectionAll(false),
			},
		},
	}

	tests := []struct {
		name        string
		serviceName string
		all         bool
		wantCount   int
		wantErr     string
	}{
		{
			name:      "all services",
			all:       true,
			wantCount: 2,
		},
		{
			name:        "single service",
			serviceName: "mysql",
			wantCount:   1,
		},
		{
			name:        "disabled service explicit",
			serviceName: "identity",
			wantCount:   1,
		},
		{
			name:    "missing selector",
			wantErr: "either --all or --service must be set",
		},
		{
			name:        "both selectors",
			serviceName: "mysql",
			all:         true,
			wantErr:     "use either --all or --service",
		},
		{
			name:        "unknown service",
			serviceName: "vault",
			wantErr:     `service "vault" was not found`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assertSelectServicesResult(t, cfg, test.serviceName, test.all, test.wantCount, test.wantErr)
		})
	}
}

func TestSelectServicesAllAppliesDefaultKindSubsets(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "database",
				SDKPackage:     "example/database",
				Group:          "database",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "AutonomousDatabase"),
			},
			{
				Service:        "mysql",
				SDKPackage:     "example/mysql",
				Group:          "mysql",
				PackageProfile: "controller-backed",
				Selection:      selectionAll(true),
			},
			{
				Service:        "identity",
				SDKPackage:     "example/identity",
				Group:          "identity",
				PackageProfile: "controller-backed",
				Selection:      selectionAll(false),
			},
		},
	}

	services := assertSelectServicesResult(t, cfg, "", true, 2, "")
	selected := make(map[string]ServiceConfig, len(services))
	for _, service := range services {
		selected[service.Service] = service
	}

	assertSelectedKinds(t, selected["database"], []string{"AutonomousDatabase"})
	assertSelectedKinds(t, selected["mysql"], nil)
	if _, ok := selected["identity"]; ok {
		t.Fatal("SelectServices(--all) unexpectedly included disabled identity service")
	}
}

func TestSelectServicesExplicitServicePreservesDefaultKindSubset(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "database",
				SDKPackage:     "example/database",
				Group:          "database",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "AutonomousDatabase"),
			},
		},
	}

	services := assertSelectServicesResult(t, cfg, "database", false, 1, "")
	assertSelectedKinds(t, services[0], []string{"AutonomousDatabase"})
}

func TestSelectServicesExplicitServiceIncludesPackageSplitKinds(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "core",
				SDKPackage:     "example/core",
				Group:          "core",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "Instance"),
				PackageSplits: []PackageSplitConfig{
					{
						Name:         "core-network",
						IncludeKinds: []string{"Subnet", "Vcn"},
					},
				},
			},
		},
	}

	services := assertSelectServicesResult(t, cfg, "core", false, 1, "")
	assertSelectedKinds(t, services[0], []string{"Instance", "Subnet", "Vcn"})
}

func TestNormalizeDefaultActiveSelection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serviceName string
		all         bool
		wantService string
		wantAll     bool
		wantErr     string
	}{
		{
			name:    "blank defaults to default active surface",
			wantAll: true,
		},
		{
			name:        "explicit service is preserved",
			serviceName: "mysql",
			wantService: "mysql",
		},
		{
			name:        "conflicting selectors fail",
			serviceName: "mysql",
			all:         true,
			wantErr:     "use either --all or --service",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assertNormalizeDefaultActiveSelection(
				t,
				test.serviceName,
				test.all,
				test.wantService,
				test.wantAll,
				test.wantErr,
			)
		})
	}
}

func TestSelectDefaultActiveOrExplicitServices(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "database",
				SDKPackage:     "example/database",
				Group:          "database",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "AutonomousDatabase"),
			},
			{
				Service:        "mysql",
				SDKPackage:     "example/mysql",
				Group:          "mysql",
				PackageProfile: "controller-backed",
				Selection:      selectionAll(true),
			},
			{
				Service:        "identity",
				SDKPackage:     "example/identity",
				Group:          "identity",
				PackageProfile: "controller-backed",
				Selection:      selectionAll(false),
			},
		},
	}

	services, err := cfg.SelectDefaultActiveOrExplicitServices("", false)
	if err != nil {
		t.Fatalf("SelectDefaultActiveOrExplicitServices() error = %v", err)
	}
	if got := serviceNames(services); !slices.Equal(got, []string{"database", "mysql"}) {
		t.Fatalf("SelectDefaultActiveOrExplicitServices() services = %v, want %v", got, []string{"database", "mysql"})
	}
	assertSelectedKinds(t, services[0], []string{"AutonomousDatabase"})
	assertSelectedKinds(t, services[1], nil)

	explicit, err := cfg.SelectDefaultActiveOrExplicitServices("identity", false)
	if err != nil {
		t.Fatalf("SelectDefaultActiveOrExplicitServices(identity) error = %v", err)
	}
	if len(explicit) != 1 || explicit[0].Service != "identity" {
		t.Fatalf("SelectDefaultActiveOrExplicitServices(identity) = %#v, want identity only", explicit)
	}
	assertSelectedKinds(t, explicit[0], nil)

	explicit, err = cfg.SelectDefaultActiveOrExplicitServices("database", false)
	if err != nil {
		t.Fatalf("SelectDefaultActiveOrExplicitServices(database) error = %v", err)
	}
	if len(explicit) != 1 || explicit[0].Service != "database" {
		t.Fatalf("SelectDefaultActiveOrExplicitServices(database) = %#v, want database only", explicit)
	}
	assertSelectedKinds(t, explicit[0], []string{"AutonomousDatabase"})
}

func TestLoadConfigIncludesSelectionMetadata(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "services.yaml")
	content := `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
services:
  - service: containerengine
    sdkPackage: github.com/oracle/oci-go-sdk/v65/containerengine
    group: containerengine
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: all
  - service: database
    sdkPackage: github.com/oracle/oci-go-sdk/v65/database
    group: database
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - AutonomousDatabase
    async:
      strategy: lifecycle
      runtime: generatedruntime
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	services := requireServices(t, cfg, "containerengine", "database")
	assertServiceSelection(t, services["containerengine"], true, SelectionModeAll, nil)
	assertServiceSelection(t, services["database"], true, SelectionModeExplicit, []string{"AutonomousDatabase"})

	activeServices := serviceNames(cfg.DefaultActiveServices())
	if !slices.Equal(activeServices, []string{"containerengine", "database"}) {
		t.Fatalf("DefaultActiveServices() = %v, want containerengine,database", activeServices)
	}
}

func TestValidatePackageSplits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		splits  []PackageSplitConfig
		wantErr string
	}{
		{
			name: "blank split name",
			splits: []PackageSplitConfig{{
				Name:         " ",
				IncludeKinds: []string{"Subnet"},
			}},
			wantErr: `packageSplits name is required`,
		},
		{
			name: "duplicate split names",
			splits: []PackageSplitConfig{
				{Name: "core-network", IncludeKinds: []string{"Subnet"}},
				{Name: "core-network", IncludeKinds: []string{"Vcn"}},
			},
			wantErr: `packageSplit "core-network" is duplicated`,
		},
		{
			name: "blank included kind",
			splits: []PackageSplitConfig{{
				Name:         "core-network",
				IncludeKinds: []string{"Subnet", " "},
			}},
			wantErr: `packageSplit "core-network" contains a blank kind`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				SchemaVersion:  "v1alpha1",
				Domain:         "oracle.com",
				DefaultVersion: "v1beta1",
				PackageProfiles: map[string]PackageProfile{
					"controller-backed": {Description: "runtime-integrated groups"},
				},
				Services: []ServiceConfig{
					{
						Service:        "core",
						SDKPackage:     "example/core",
						Group:          "core",
						PackageProfile: "controller-backed",
						PackageSplits:  test.splits,
						Selection:      selectionAll(false),
					},
				},
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() unexpectedly succeeded")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestLoadConfigIncludesObservedStateAliases(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "services.yaml")
	content := `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  crd-only:
    description: generated APIs
services:
  - service: containerengine
    sdkPackage: github.com/oracle/oci-go-sdk/v65/containerengine
    group: containerengine
    packageProfile: crd-only
    selection:
      enabled: false
      mode: all
    observedState:
      sdkAliases:
        WorkRequestLog:
          - WorkRequestLogEntry
  - service: psql
    sdkPackage: github.com/oracle/oci-go-sdk/v65/psql
    group: psql
    packageProfile: crd-only
    selection:
      enabled: false
      mode: all
    observedState:
      sdkAliases:
        PrimaryDbInstance:
          - PrimaryDbInstanceDetails
        WorkRequestLog:
          - WorkRequestLogEntry
  - service: identity
    sdkPackage: github.com/oracle/oci-go-sdk/v65/identity
    group: identity
    packageProfile: crd-only
    selection:
      enabled: false
      mode: all
    observedState:
      sdkAliases:
        CostTrackingTag:
          - Tag
        UserState:
          - User
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.Services) != 3 {
		t.Fatalf("len(cfg.Services) = %d, want 3", len(cfg.Services))
	}

	containerEngineService := cfg.Services[0]
	if !slices.Equal(containerEngineService.ObservedState.SDKAliases["WorkRequestLog"], []string{"WorkRequestLogEntry"}) {
		t.Fatalf("containerengine WorkRequestLog aliases = %v, want WorkRequestLogEntry", containerEngineService.ObservedState.SDKAliases["WorkRequestLog"])
	}

	psqlService := cfg.Services[1]
	if !slices.Equal(psqlService.ObservedState.SDKAliases["PrimaryDbInstance"], []string{"PrimaryDbInstanceDetails"}) {
		t.Fatalf("PrimaryDbInstance aliases = %v, want PrimaryDbInstanceDetails", psqlService.ObservedState.SDKAliases["PrimaryDbInstance"])
	}
	if !slices.Equal(psqlService.ObservedState.SDKAliases["WorkRequestLog"], []string{"WorkRequestLogEntry"}) {
		t.Fatalf("WorkRequestLog aliases = %v, want WorkRequestLogEntry", psqlService.ObservedState.SDKAliases["WorkRequestLog"])
	}

	identityService := cfg.Services[2]
	if !slices.Equal(identityService.ObservedState.SDKAliases["CostTrackingTag"], []string{"Tag"}) {
		t.Fatalf("CostTrackingTag aliases = %v, want Tag", identityService.ObservedState.SDKAliases["CostTrackingTag"])
	}
	if !slices.Equal(identityService.ObservedState.SDKAliases["UserState"], []string{"User"}) {
		t.Fatalf("UserState aliases = %v, want User", identityService.ObservedState.SDKAliases["UserState"])
	}
}

//nolint:gocyclo // This fixture-based config parser test intentionally checks multiple rollout surfaces in one YAML example.
func TestLoadConfigIncludesGenerationRolloutAndOverrides(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "services.yaml")
	content := `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
  crd-only:
    description: generated APIs
services:
  - service: mysql
    sdkPackage: github.com/oracle/oci-go-sdk/v65/mysql
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - DbSystem
    async:
      strategy: lifecycle
      runtime: generatedruntime
    generation:
      controller:
        strategy: manual
      serviceManager:
        strategy: generated
      registration:
        strategy: generated
      webhooks:
        strategy: none
      resources:
        - kind: DbSystem
          controller:
            maxConcurrentReconciles: 3
            extraRBACMarkers:
              - groups="",resources=secrets,verbs=get;list;watch;create;update;delete
          specFields:
            - name: AdminUsername
              type: shared.UsernameSource
              tag: 'json:"adminUsername,omitempty,omitzero"'
          statusFields:
            - name: AdminPassword
              type: shared.PasswordSource
              tag: 'json:"adminPassword,omitempty,omitzero"'
          sample:
            body: |-
              apiVersion: mysql.oracle.com/v1beta1
              kind: DbSystem
              metadata:
                name: dbsystem-sample
              spec:
                adminUsername:
                  secret:
                    secretName: admin-secret
          serviceManager:
            packagePath: mysql/dbsystem
            needsCredentialClient: true
  - service: core
    sdkPackage: github.com/oracle/oci-go-sdk/v65/core
    group: core
    packageProfile: crd-only
    selection:
      enabled: false
      mode: all
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.Services) != 2 {
		t.Fatalf("len(cfg.Services) = %d, want 2", len(cfg.Services))
	}

	services := requireServices(t, cfg, "mysql", "core")
	mysqlService := services["mysql"]
	assertServiceGenerationStrategies(t, mysqlService, generationStrategyExpectations{
		controller:     GenerationStrategyManual,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, mysqlService, 1)
	assertMySQLGenerationOverride(t, mysqlService.Generation.Resources[0], mysqlSecretRBACMarkers())
	override := mysqlService.Generation.Resources[0]
	if override.Kind != "DbSystem" {
		t.Fatalf("mysql override kind = %q, want %q", override.Kind, "DbSystem")
	}
	if override.Controller.MaxConcurrentReconciles != 3 {
		t.Fatalf("mysql maxConcurrentReconciles = %d, want 3", override.Controller.MaxConcurrentReconciles)
	}
	if !slices.Equal(override.Controller.ExtraRBACMarkers, mysqlSecretRBACMarkers()) {
		t.Fatalf("mysql extra RBAC markers = %v, want secret read and write markers", override.Controller.ExtraRBACMarkers)
	}
	if override.ServiceManager.PackagePath != "mysql/dbsystem" {
		t.Fatalf("mysql packagePath = %q, want %q", override.ServiceManager.PackagePath, "mysql/dbsystem")
	}
	if len(override.SpecFields) != 1 || override.SpecFields[0].Name != "AdminUsername" {
		t.Fatalf("mysql specFields = %#v, want AdminUsername override", override.SpecFields)
	}
	if len(override.StatusFields) != 1 || override.StatusFields[0].Name != "AdminPassword" {
		t.Fatalf("mysql statusFields = %#v, want AdminPassword override", override.StatusFields)
	}
	if !strings.Contains(override.Sample.Body, "secretName: admin-secret") {
		t.Fatalf("mysql sample override = %q, want secret-backed body", override.Sample.Body)
	}
	assertServiceGenerationStrategies(t, services["core"], generationStrategyExpectations{
		controller:     GenerationStrategyNone,
		serviceManager: GenerationStrategyNone,
		registration:   GenerationStrategyNone,
		webhook:        GenerationStrategyManual,
	})
}

func TestLoadConfigIncludesFormalSpecReferences(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "services.yaml")
	content := `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
services:
  - service: mysql
    sdkPackage: github.com/oracle/oci-go-sdk/v65/mysql
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: all
    formalSpec: dbsystem
    generation:
      resources:
        - kind: Widget
          formalSpec: widget
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.Services) != 1 {
		t.Fatalf("len(cfg.Services) = %d, want 1", len(cfg.Services))
	}

	service := cfg.Services[0]
	if service.FormalSpec != "dbsystem" {
		t.Fatalf("service formalSpec = %q, want %q", service.FormalSpec, "dbsystem")
	}
	if got := service.FormalSpecFor("Widget"); got != "widget" {
		t.Fatalf("FormalSpecFor(Widget) = %q, want %q", got, "widget")
	}
	if got := service.FormalSpecFor("DbSystem"); got != "dbsystem" {
		t.Fatalf("FormalSpecFor(DbSystem) = %q, want %q", got, "dbsystem")
	}
	if got := filepath.ToSlash(cfg.FormalRoot()); got != "formal" && !strings.HasSuffix(got, "/formal") {
		t.Fatalf("FormalRoot() = %q, want a formal/ path", got)
	}
}

func TestConfigVerifyFormalInputsSkipsConfigsWithoutFormalSpecs(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Services: []ServiceConfig{
			{Service: "core"},
		},
	}

	if err := cfg.VerifyFormalInputs(); err != nil {
		t.Fatalf("VerifyFormalInputs() error = %v", err)
	}
}

func TestConfigVerifyFormalInputsRejectsMissingFormalRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := &Config{
		configDir: filepath.Join(root, "internal", "generator", "config"),
		Services: []ServiceConfig{
			{Service: "identity", FormalSpec: "user"},
		},
	}

	err := cfg.VerifyFormalInputs()
	if err == nil {
		t.Fatal("VerifyFormalInputs() error = nil, want missing formal root failure")
	}
	if !strings.Contains(err.Error(), filepath.ToSlash(filepath.Join(root, "formal"))) {
		t.Fatalf("VerifyFormalInputs() error = %v, want formal root path", err)
	}
}

func TestCheckedInConfigVerifyFormalInputs(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	if err := cfg.VerifyFormalInputs(); err != nil {
		t.Fatalf("VerifyFormalInputs() error = %v", err)
	}
}

func TestServiceConfigControllerGenerationConfigFor(t *testing.T) {
	t.Parallel()

	service := ServiceConfig{
		Service: "mysql",
		Group:   "mysql",
		Generation: GenerationConfig{
			Controller: GenerationSurfaceConfig{Strategy: GenerationStrategyManual},
			Resources: []ResourceGenerationOverride{
				{
					Kind: "DbSystem",
					Controller: ControllerGenerationOverride{
						Strategy:                GenerationStrategyGenerated,
						MaxConcurrentReconciles: 3,
						ExtraRBACMarkers: []string{
							`groups="",resources=secrets,verbs=get;list;watch`,
						},
					},
				},
			},
		},
	}

	if got := service.ControllerGenerationStrategyFor("DbSystem"); got != GenerationStrategyGenerated {
		t.Fatalf("ControllerGenerationStrategyFor(DbSystem) = %q, want %q", got, GenerationStrategyGenerated)
	}

	config := service.ControllerGenerationConfigFor("DbSystem")
	if config.Strategy != GenerationStrategyGenerated {
		t.Fatalf("ControllerGenerationConfigFor(DbSystem).Strategy = %q, want %q", config.Strategy, GenerationStrategyGenerated)
	}
	if config.MaxConcurrentReconciles != 3 {
		t.Fatalf("ControllerGenerationConfigFor(DbSystem).MaxConcurrentReconciles = %d, want 3", config.MaxConcurrentReconciles)
	}
	if !slices.Equal(config.ExtraRBACMarkers, []string{`groups="",resources=secrets,verbs=get;list;watch`}) {
		t.Fatalf("ControllerGenerationConfigFor(DbSystem).ExtraRBACMarkers = %v", config.ExtraRBACMarkers)
	}

	if got := service.ControllerGenerationStrategyFor("Widget"); got != GenerationStrategyManual {
		t.Fatalf("ControllerGenerationStrategyFor(Widget) = %q, want %q", got, GenerationStrategyManual)
	}
	if got := service.ControllerGenerationConfigFor("Widget"); got.Strategy != GenerationStrategyManual {
		t.Fatalf("ControllerGenerationConfigFor(Widget).Strategy = %q, want %q", got.Strategy, GenerationStrategyManual)
	}
}

func TestValidateAllowsResourceFormalSpecWithoutRuntimeOverride(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "mysql",
				SDKPackage:     "example/mysql",
				Group:          "mysql",
				PackageProfile: "controller-backed",
				Selection:      selectionAll(true),
				Generation: GenerationConfig{
					Resources: []ResourceGenerationOverride{
						{
							Kind:       "Widget",
							FormalSpec: "widget",
						},
					},
				},
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsInvalidSelectionConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "missing selection block",
			mutate: func(cfg *Config) {
				cfg.Services[0].Selection = SelectionConfig{}
			},
			wantErr: "selection.enabled is required",
		},
		{
			name: "missing selection mode",
			mutate: func(cfg *Config) {
				cfg.Services[0].Selection.Mode = ""
			},
			wantErr: `selection.mode ""`,
		},
		{
			name: "invalid selection mode",
			mutate: func(cfg *Config) {
				cfg.Services[0].Selection.Mode = "subset"
			},
			wantErr: `selection.mode "subset"`,
		},
		{
			name: "all mode includes kinds",
			mutate: func(cfg *Config) {
				cfg.Services[0].Selection = selectionExplicit(true, "DbSystem")
				cfg.Services[0].Selection.Mode = SelectionModeAll
			},
			wantErr: `selection.includeKinds must be empty when selection.mode is "all"`,
		},
		{
			name: "explicit mode without kinds",
			mutate: func(cfg *Config) {
				cfg.Services[0].Selection = SelectionConfig{
					Enabled: boolPtr(true),
					Mode:    SelectionModeExplicit,
				}
			},
			wantErr: `selection.includeKinds must list at least one kind when selection.mode is "explicit"`,
		},
		{
			name: "explicit mode blank kind",
			mutate: func(cfg *Config) {
				cfg.Services[0].Selection = SelectionConfig{
					Enabled:      boolPtr(true),
					Mode:         SelectionModeExplicit,
					IncludeKinds: []string{"DbSystem", " "},
				}
			},
			wantErr: "selection.includeKinds[1] must not be blank",
		},
		{
			name: "explicit mode duplicate kind",
			mutate: func(cfg *Config) {
				cfg.Services[0].Selection = selectionExplicit(true, "DbSystem", "DbSystem")
			},
			wantErr: `selection.includeKinds contains duplicate kind "DbSystem"`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				SchemaVersion:  "v1alpha1",
				Domain:         "oracle.com",
				DefaultVersion: "v1beta1",
				PackageProfiles: map[string]PackageProfile{
					"controller-backed": {Description: "runtime-integrated groups"},
				},
				Services: []ServiceConfig{
					{
						Service:        "mysql",
						SDKPackage:     "example/mysql",
						Group:          "mysql",
						PackageProfile: "controller-backed",
						Selection:      selectionAll(true),
					},
				},
			}

			test.mutate(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() unexpectedly succeeded")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestValidateRejectsInvalidGenerationConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "invalid controller strategy",
			mutate: func(cfg *Config) {
				cfg.Services[0].Generation.Controller.Strategy = "auto"
			},
			wantErr: `generation.controller.strategy "auto"`,
		},
		{
			name: "invalid webhook strategy",
			mutate: func(cfg *Config) {
				cfg.Services[0].Generation.Webhooks.Strategy = GenerationStrategyGenerated
			},
			wantErr: `generation.webhooks.strategy "generated"`,
		},
		{
			name: "duplicate resource override",
			mutate: func(cfg *Config) {
				cfg.Services[0].Generation.Resources = []ResourceGenerationOverride{
					{Kind: "DbSystem", Controller: ControllerGenerationOverride{Strategy: GenerationStrategyManual}},
					{Kind: "DbSystem", ServiceManager: ServiceManagerGenerationOverride{PackagePath: "mysql/dbsystem"}},
				}
			},
			wantErr: `duplicate kind "DbSystem"`,
		},
		{
			name: "blank extra rbac marker",
			mutate: func(cfg *Config) {
				cfg.Services[0].Generation.Resources = []ResourceGenerationOverride{
					{
						Kind: "DbSystem",
						Controller: ControllerGenerationOverride{
							ExtraRBACMarkers: []string{" "},
						},
					},
				}
			},
			wantErr: "extraRBACMarkers contains a blank marker",
		},
		{
			name: "invalid package path",
			mutate: func(cfg *Config) {
				cfg.Services[0].Generation.Resources = []ResourceGenerationOverride{
					{
						Kind: "DbSystem",
						ServiceManager: ServiceManagerGenerationOverride{
							PackagePath: "../mysql/dbsystem",
						},
					},
				}
			},
			wantErr: "packagePath must be a clean relative path beneath pkg/servicemanager",
		},
		{
			name: "empty resource override",
			mutate: func(cfg *Config) {
				cfg.Services[0].Generation.Resources = []ResourceGenerationOverride{
					{Kind: "DbSystem"},
				}
			},
			wantErr: `generation.resources["DbSystem"] does not override any runtime output`,
		},
		{
			name: "invalid service formal spec",
			mutate: func(cfg *Config) {
				cfg.Services[0].FormalSpec = "mysql/widget"
			},
			wantErr: `formalSpec "mysql/widget" must be a single formal slug`,
		},
		{
			name: "invalid resource formal spec",
			mutate: func(cfg *Config) {
				cfg.Services[0].Generation.Resources = []ResourceGenerationOverride{
					{
						Kind:       "DbSystem",
						FormalSpec: "../dbsystem",
					},
				}
			},
			wantErr: `formalSpec "../dbsystem" must be a single formal slug`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				SchemaVersion:  "v1alpha1",
				Domain:         "oracle.com",
				DefaultVersion: "v1beta1",
				PackageProfiles: map[string]PackageProfile{
					"controller-backed": {Description: "runtime-integrated groups"},
				},
				Services: []ServiceConfig{
					{
						Service:        "mysql",
						SDKPackage:     "example/mysql",
						Group:          "mysql",
						PackageProfile: "controller-backed",
						Selection:      selectionAll(true),
					},
				},
			}

			test.mutate(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() unexpectedly succeeded")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestCheckedInConfigIncludesDefaultActiveSelectionMetadata(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)

	activeServices := serviceNames(cfg.DefaultActiveServices())
	wantActiveServices := []string{
		"accessgovernancecp",
		"adm",
		"aidocument",
		"ailanguage",
		"aispeech",
		"aivision",
		"analytics",
		"apiaccesscontrol",
		"apiplatform",
		"apmcontrolplane",
		"bds",
		"budget",
		"clusterplacementgroups",
		"containerengine",
		"containerinstances",
		"core",
		"dataflow",
		"database",
		"databasetools",
		"databasemigration",
		"datalabelingservice",
		"datascience",
		"email",
		"functions",
		"generativeai",
		"identity",
		"keymanagement",
		"loadbalancer",
		"logging",
		"monitoring",
		"marketplace",
		"mysql",
		"nosql",
		"objectstorage",
		"ocvp",
		"oda",
		"opensearch",
		"psql",
		"queue",
		"redis",
		"streaming",
		"usageapi",
	}
	if !slices.Equal(activeServices, wantActiveServices) {
		t.Fatalf("DefaultActiveServices() = %v, want %v", activeServices, wantActiveServices)
	}

	services := requireServices(
		t,
		cfg,
		"accessgovernancecp",
		"adm",
		"aidocument",
		"ailanguage",
		"aispeech",
		"aivision",
		"analytics",
		"apiaccesscontrol",
		"apiplatform",
		"apmcontrolplane",
		"bds",
		"budget",
		"clusterplacementgroups",
		"containerengine",
		"containerinstances",
		"core",
		"dataflow",
		"database",
		"databasemigration",
		"databasetools",
		"datalabelingservice",
		"datascience",
		"email",
		"functions",
		"generativeai",
		"identity",
		"keymanagement",
		"loadbalancer",
		"logging",
		"monitoring",
		"marketplace",
		"mysql",
		"nosql",
		"objectstorage",
		"ocvp",
		"oda",
		"opensearch",
		"psql",
		"queue",
		"redis",
		"streaming",
		"usageapi",
		"vault",
	)
	assertServiceSelection(t, services["accessgovernancecp"], true, SelectionModeExplicit, []string{"GovernanceInstance"})
	assertServiceSelection(t, services["aidocument"], true, SelectionModeExplicit, []string{"Project"})
	assertServiceSelection(t, services["ailanguage"], true, SelectionModeExplicit, []string{"Project"})
	assertServiceSelection(t, services["aispeech"], true, SelectionModeExplicit, []string{"TranscriptionJob"})
	assertServiceSelection(t, services["aivision"], true, SelectionModeExplicit, []string{"Project"})
	assertServiceSelection(t, services["analytics"], true, SelectionModeExplicit, []string{"AnalyticsInstance"})
	assertServiceSelection(t, services["apiaccesscontrol"], true, SelectionModeExplicit, []string{"PrivilegedApiControl"})
	assertServiceSelection(t, services["bds"], true, SelectionModeExplicit, []string{"BdsInstance"})
	assertServiceSelection(t, services["budget"], true, SelectionModeExplicit, []string{"Budget"})
	assertServiceSelection(t, services["clusterplacementgroups"], true, SelectionModeExplicit, []string{"ClusterPlacementGroup"})
	assertServiceSelection(t, services["containerengine"], true, SelectionModeExplicit, []string{"Cluster", "NodePool"})
	assertServiceSelection(t, services["containerinstances"], true, SelectionModeExplicit, []string{"ContainerInstance"})
	assertServiceSelection(t, services["core"], true, SelectionModeExplicit, []string{"Instance"})
	assertServiceSelection(t, services["dataflow"], true, SelectionModeExplicit, []string{"Application"})
	assertServiceSelection(t, services["database"], true, SelectionModeExplicit, []string{"AutonomousDatabase"})
	assertServiceSelection(t, services["databasemigration"], true, SelectionModeExplicit, []string{"Connection"})
	assertServiceSelection(t, services["databasetools"], true, SelectionModeExplicit, []string{"DatabaseToolsConnection"})
	assertServiceSelection(t, services["datalabelingservice"], true, SelectionModeExplicit, []string{"Dataset"})
	assertServiceSelection(t, services["datascience"], true, SelectionModeExplicit, []string{"Project"})
	assertServiceSelection(t, services["email"], true, SelectionModeExplicit, []string{"Dkim", "EmailDomain", "Sender", "Suppression"})
	assertServiceSelection(t, services["functions"], true, SelectionModeExplicit, []string{"Application", "Function"})
	assertServiceSelection(t, services["generativeai"], true, SelectionModeExplicit, []string{"DedicatedAiCluster", "Endpoint", "Model"})
	assertServiceSelection(t, services["identity"], true, SelectionModeExplicit, []string{"Compartment"})
	assertServiceSelection(t, services["keymanagement"], true, SelectionModeExplicit, []string{"Vault"})
	assertServiceSelection(t, services["loadbalancer"], true, SelectionModeExplicit, []string{"Backend", "BackendSet", "Certificate", "Hostname", "Listener", "LoadBalancer", "PathRouteSet", "RoutingPolicy", "RuleSet", "SSLCipherSuite"})
	assertServiceSelection(t, services["logging"], true, SelectionModeExplicit, []string{"Log", "LogGroup", "LogSavedSearch", "UnifiedAgentConfiguration"})
	assertServiceSelection(t, services["monitoring"], true, SelectionModeExplicit, []string{"Alarm", "AlarmSuppression"})
	assertServiceSelection(t, services["marketplace"], true, SelectionModeExplicit, []string{"AcceptedAgreement", "Publication"})
	assertServiceSelection(t, services["mysql"], true, SelectionModeExplicit, []string{"DbSystem"})
	assertServiceSelection(t, services["nosql"], true, SelectionModeExplicit, []string{"Table"})
	assertServiceSelection(t, services["objectstorage"], true, SelectionModeExplicit, []string{"Bucket"})
	assertServiceSelection(t, services["ocvp"], true, SelectionModeExplicit, []string{"Cluster", "EsxiHost", "Sddc"})
	assertServiceSelection(t, services["oda"], true, SelectionModeExplicit, []string{"AuthenticationProvider", "Channel", "DigitalAssistant", "ImportedPackage", "OdaInstance", "OdaInstanceAttachment", "OdaPrivateEndpoint", "OdaPrivateEndpointAttachment", "OdaPrivateEndpointScanProxy", "Skill", "SkillParameter", "Translator"})
	assertServiceSelection(t, services["opensearch"], true, SelectionModeExplicit, []string{"OpensearchCluster"})
	assertServiceSelection(t, services["psql"], true, SelectionModeExplicit, []string{"DbSystem"})
	assertServiceSelection(t, services["queue"], true, SelectionModeExplicit, []string{"Queue"})
	assertServiceSelection(t, services["redis"], true, SelectionModeExplicit, []string{"RedisCluster"})
	assertServiceSelection(t, services["streaming"], true, SelectionModeExplicit, []string{"Stream"})
	assertServiceSelection(t, services["usageapi"], true, SelectionModeExplicit, []string{"CustomTable", "Query", "Schedule", "UsageCarbonEmissionsQuery"})
	assertServiceSelection(t, services["vault"], false, SelectionModeAll, nil)
}

func TestCheckedInConfigIncludesRuntimeRolloutMetadata(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	services := serviceConfigsByName(t, cfg, "aidocument", "ailanguage", "aispeech", "aivision", "bds", "containerengine", "containerinstances", "core", "dataflow", "database", "databasemigration", "databasetools", "datalabelingservice", "datascience", "functions", "identity", "keymanagement", "mysql", "nosql", "ocvp", "psql", "redis", "streaming")
	assertAIDocumentRuntimeRolloutMetadata(t, services["aidocument"])
	assertAILanguageRuntimeRolloutMetadata(t, services["ailanguage"])
	assertAISpeechRuntimeRolloutMetadata(t, services["aispeech"])
	assertAIVisionRuntimeRolloutMetadata(t, services["aivision"])
	assertBDSRuntimeRolloutMetadata(t, services["bds"])
	assertDatabaseMigrationRuntimeRolloutMetadata(t, services["databasemigration"])
	assertDatabaseToolsRuntimeRolloutMetadata(t, services["databasetools"])
	assertDataScienceRuntimeRolloutMetadata(t, services["datascience"])

	assertServiceGenerationStrategies(t, services["dataflow"], generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertServiceGenerationStrategies(t, services["database"], generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertServiceGenerationStrategies(t, services["databasemigration"], generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertServiceGenerationStrategies(t, services["mysql"], generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertServiceGenerationStrategies(t, services["datalabelingservice"], generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertServiceGenerationStrategies(t, services["streaming"], generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyManual,
	})

	assertContainerengineRuntimeRolloutMetadata(t, services["containerengine"])
	assertDatabaseRuntimeRolloutMetadata(t, services["database"])
	assertContainerInstancesRuntimeRolloutMetadata(t, services["containerinstances"])
	assertFunctionsRuntimeRolloutMetadata(t, services["functions"])
	assertDataflowRuntimeRolloutMetadata(t, services["dataflow"])
	assertMySQLRuntimeRolloutMetadata(t, services["mysql"])
	assertNoSQLRuntimeRolloutMetadata(t, services["nosql"])
	assertPSQLRuntimeRolloutMetadata(t, services["psql"])
	assertStreamingRuntimeRolloutMetadata(t, services["streaming"])
	assertCoreRuntimeRolloutMetadata(t, services["core"])
	assertIdentityRuntimeRolloutMetadata(t, services["identity"])
	assertOCVPRuntimeRolloutMetadata(t, services["ocvp"])
	assertRedisRuntimeRolloutMetadata(t, services["redis"])
}

func TestCheckedInConfigSelectServicesPreservesCorePackageSplitKinds(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)

	services := assertSelectServicesResult(t, cfg, "core", false, 1, "")
	assertSelectedKinds(t, services[0], []string{
		"Instance",
		"Drg",
		"InternetGateway",
		"NatGateway",
		"NetworkSecurityGroup",
		"RouteTable",
		"SecurityList",
		"ServiceGateway",
		"Subnet",
		"Vcn",
	})
}

func TestCheckedInConfigPromotesFormalSpecReferences(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	services := serviceConfigsByName(t, cfg, "aidocument", "ailanguage", "aispeech", "aivision", "analytics", "apiaccesscontrol", "bds", "containerengine", "containerinstances", "core", "database", "databasemigration", "databasetools", "datalabelingservice", "datascience", "dataflow", "identity", "mysql", "objectstorage", "ocvp", "opensearch", "psql", "redis", "streaming")
	assertFormalSpecFor(t, services["aidocument"], "Project", "project")
	assertFormalSpecFor(t, services["ailanguage"], "Project", "project")
	assertFormalSpecFor(t, services["aispeech"], "TranscriptionJob", "transcriptionjob")
	assertFormalSpecFor(t, services["aivision"], "Project", "project")
	assertFormalSpecFor(t, services["analytics"], "AnalyticsInstance", "analyticsinstance")
	assertFormalSpecFor(t, services["apiaccesscontrol"], "PrivilegedApiControl", "privilegedapicontrol")
	assertFormalSpecFor(t, services["bds"], "BdsInstance", "bdsinstance")
	assertFormalSpecFor(t, services["containerengine"], "Cluster", "cluster")
	assertFormalSpecFor(t, services["containerengine"], "NodePool", "nodepool")
	assertFormalSpecFor(t, services["containerinstances"], "ContainerInstance", "")
	assertFormalSpecFor(t, services["databasemigration"], "Connection", "connection")
	assertFormalSpecFor(t, services["databasetools"], "DatabaseToolsConnection", "databasetoolsconnection")
	assertFormalSpecFor(t, services["datalabelingservice"], "Dataset", "dataset")
	assertFormalSpecFor(t, services["identity"], "Compartment", "compartment")
	for _, formal := range []struct {
		kind string
		slug string
	}{
		{kind: "Instance", slug: "instance"},
		{kind: "InternetGateway", slug: "internetgateway"},
		{kind: "NatGateway", slug: "natgateway"},
		{kind: "NetworkSecurityGroup", slug: "networksecuritygroup"},
		{kind: "RouteTable", slug: "routetable"},
		{kind: "SecurityList", slug: "securitylist"},
		{kind: "ServiceGateway", slug: "servicegateway"},
		{kind: "Subnet", slug: "subnet"},
		{kind: "Vcn", slug: "vcn"},
	} {
		assertFormalSpecFor(t, services["core"], formal.kind, formal.slug)
	}
	assertFormalSpecFor(t, services["dataflow"], "Application", "application")
	assertFormalSpecFor(t, services["database"], "AutonomousDatabase", "databaseautonomousdatabase")
	assertFormalSpecFor(t, services["datascience"], "Project", "project")
	assertFormalSpecFor(t, services["mysql"], "DbSystem", "dbsystem")
	assertFormalSpecFor(t, services["objectstorage"], "Bucket", "objectstoragebucket")
	assertFormalSpecFor(t, services["ocvp"], "Cluster", "cluster")
	assertFormalSpecFor(t, services["ocvp"], "Sddc", "sddc")
	assertFormalSpecFor(t, services["opensearch"], "OpensearchCluster", "opensearchopensearchcluster")
	assertFormalSpecFor(t, services["psql"], "DbSystem", "dbsystem")
	assertFormalSpecFor(t, services["redis"], "RedisCluster", "rediscluster")
	assertFormalSpecFor(t, services["streaming"], "Stream", "stream")
}

func TestCheckedInConfigCoordinatesPrimaryPortPackagePaths(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	services := serviceConfigsByName(t, cfg, "analytics", "containerengine", "containerinstances", "core", "dataflow", "database", "identity", "keymanagement", "mysql", "objectstorage", "ocvp", "opensearch", "psql", "redis")

	assertPrimaryPortOverride(t, services["analytics"], "AnalyticsInstance", "analyticsinstance", "analytics/analyticsinstance")
	assertContainerengineRuntimeRolloutMetadata(t, services["containerengine"])
	assertContainerInstancesRuntimeRolloutMetadata(t, services["containerinstances"])
	assertPrimaryPortOverride(t, services["core"], "Instance", "instance", "core/instance")
	assertPrimaryPortOverride(t, services["dataflow"], "Application", "application", "dataflow/application")
	assertDatabaseRuntimeRolloutMetadata(t, services["database"])
	assertPrimaryPortOverride(t, services["identity"], "Compartment", "compartment", "identity/compartment")
	assertPrimaryPortOverride(t, services["keymanagement"], "Vault", "", "keymanagement/vault")
	assertMySQLRuntimeRolloutMetadata(t, services["mysql"])
	assertPrimaryPortOverride(t, services["objectstorage"], "Bucket", "objectstoragebucket", "objectstorage/bucket")
	assertOCVPRuntimeRolloutMetadata(t, services["ocvp"])
	assertOpensearchRuntimeRolloutMetadata(t, services["opensearch"])
	assertPSQLRuntimeRolloutMetadata(t, services["psql"])
	assertPrimaryPortOverride(t, services["redis"], "RedisCluster", "rediscluster", "redis/rediscluster")
}

func TestCheckedInConfigOptsOutEndpointBasedGeneratedRuntimeResources(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)

	wantKinds := map[string][]string{
		"keymanagement": {"Key", "KeyVersion", "ReplicationStatus", "WrappingKey"},
	}

	for serviceName, kinds := range wantKinds {
		assertGeneratedRuntimeOptOutKinds(t, cfg, serviceName, kinds)
	}
}

type generationStrategyExpectations struct {
	controller     string
	serviceManager string
	registration   string
	webhook        string
}

func assertNormalizeDefaultActiveSelection(
	t *testing.T,
	serviceName string,
	all bool,
	wantService string,
	wantAll bool,
	wantErr string,
) {
	t.Helper()

	gotService, gotAll, err := NormalizeDefaultActiveSelection(serviceName, all)
	if wantErr != "" {
		if err == nil {
			t.Fatalf("NormalizeDefaultActiveSelection() error = nil, want %q", wantErr)
		}
		if !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("NormalizeDefaultActiveSelection() error = %v, want substring %q", err, wantErr)
		}
		return
	}
	if err != nil {
		t.Fatalf("NormalizeDefaultActiveSelection() error = %v", err)
	}
	if gotService != wantService {
		t.Fatalf("NormalizeDefaultActiveSelection() service = %q, want %q", gotService, wantService)
	}
	if gotAll != wantAll {
		t.Fatalf("NormalizeDefaultActiveSelection() all = %t, want %t", gotAll, wantAll)
	}
}

func assertSelectServicesResult(t *testing.T, cfg *Config, serviceName string, all bool, wantCount int, wantErr string) []ServiceConfig {
	t.Helper()

	services, err := cfg.SelectServices(serviceName, all)
	if wantErr != "" {
		if err == nil {
			t.Fatalf("SelectServices() error = nil, want %q", wantErr)
		}
		if !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("SelectServices() error = %v, want substring %q", err, wantErr)
		}
		return nil
	}
	if err != nil {
		t.Fatalf("SelectServices() error = %v", err)
	}
	if len(services) != wantCount {
		t.Fatalf("SelectServices() returned %d services, want %d", len(services), wantCount)
	}
	return services
}

func assertServiceSelection(t *testing.T, service *ServiceConfig, wantEnabled bool, wantMode string, wantKinds []string) {
	t.Helper()

	if got := service.IsDefaultActive(); got != wantEnabled {
		t.Fatalf("%s default active = %t, want %t", service.Service, got, wantEnabled)
	}
	if got := service.DefaultSelectionMode(); got != wantMode {
		t.Fatalf("%s selection mode = %q, want %q", service.Service, got, wantMode)
	}
	if got := service.DefaultIncludeKinds(); !slices.Equal(got, wantKinds) {
		t.Fatalf("%s includeKinds = %v, want %v", service.Service, got, wantKinds)
	}
}

func assertSelectedKinds(t *testing.T, service ServiceConfig, want []string) {
	t.Helper()

	if got := service.SelectedKinds(); !slices.Equal(got, want) {
		t.Fatalf("%s selectedKinds = %v, want %v", service.Service, got, want)
	}
}

func selectionAll(enabled bool) SelectionConfig {
	return SelectionConfig{
		Enabled: boolPtr(enabled),
		Mode:    SelectionModeAll,
	}
}

func TestValidateSelectedAsyncMetadataRequiresStrategyForOptedInSelectedKind(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "mysql",
				SDKPackage:     "example/mysql",
				Group:          "mysql",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "DbSystem"),
				Generation: GenerationConfig{
					Resources: []ResourceGenerationOverride{
						{
							Kind: "DbSystem",
							Async: AsyncConfig{
								Runtime: AsyncRuntimeGeneratedRuntime,
							},
						},
					},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want selected-kind async failure")
	}
	if !strings.Contains(err.Error(), `selected kind "DbSystem" async.strategy is required`) {
		t.Fatalf("Validate() error = %v, want selected-kind strategy failure", err)
	}
}

func TestValidateSelectedAsyncMetadataRequiresContractWithoutAsyncOptIn(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "containerengine",
				SDKPackage:     "example/containerengine",
				Group:          "containerengine",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "Cluster"),
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing selected-kind async failure")
	}
	if !strings.Contains(err.Error(), `selected kind "Cluster" async.strategy is required`) {
		t.Fatalf("Validate() error = %v, want selected-kind strategy failure", err)
	}
}

func TestValidateSelectedAsyncMetadataRequiresContractForSelectedPackageSplitKinds(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "core",
				SDKPackage:     "example/core",
				Group:          "core",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "Instance"),
				PackageSplits: []PackageSplitConfig{
					{
						Name:         "core-network",
						IncludeKinds: []string{"Vcn"},
					},
				},
				Generation: GenerationConfig{
					Resources: []ResourceGenerationOverride{
						{
							Kind: "Instance",
							Async: AsyncConfig{
								Strategy: AsyncStrategyLifecycle,
								Runtime:  AsyncRuntimeGeneratedRuntime,
							},
						},
					},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want package-split selected-kind async failure")
	}
	if !strings.Contains(err.Error(), `selected kind "Vcn" async.strategy is required`) {
		t.Fatalf("Validate() error = %v, want package-split selected-kind failure", err)
	}
}

func TestValidateSelectedAsyncMetadataAllowsResourceOverridesToClearInheritedWorkRequestDefaults(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "queue",
				SDKPackage:     "example/queue",
				Group:          "queue",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "Queue", "Stream"),
				Async: AsyncConfig{
					Strategy: AsyncStrategyWorkRequest,
					Runtime:  AsyncRuntimeHandwritten,
					WorkRequest: AsyncWorkRequestConfig{
						Source: AsyncWorkRequestSourceServiceSDK,
						Phases: []string{AsyncPhaseCreate, AsyncPhaseDelete},
						LegacyFieldBridge: AsyncLegacyFieldBridge{
							Create: "CreateWorkRequestId",
							Delete: "DeleteWorkRequestId",
						},
					},
				},
				Generation: GenerationConfig{
					Resources: []ResourceGenerationOverride{
						{
							Kind: "Queue",
							Async: AsyncConfig{
								Strategy: AsyncStrategyLifecycle,
								Runtime:  AsyncRuntimeGeneratedRuntime,
							},
						},
						{
							Kind: "Stream",
							Async: AsyncConfig{
								Strategy: AsyncStrategyNone,
								Runtime:  AsyncRuntimeGeneratedRuntime,
							},
						},
					},
				},
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestValidateSelectedAsyncMetadataAllowsGeneratedRuntimeWorkRequestContracts(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "queue",
				SDKPackage:     "example/queue",
				Group:          "queue",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "Queue"),
				Generation: GenerationConfig{
					Resources: []ResourceGenerationOverride{
						{
							Kind: "Queue",
							Async: AsyncConfig{
								Strategy: AsyncStrategyWorkRequest,
								Runtime:  AsyncRuntimeGeneratedRuntime,
								WorkRequest: AsyncWorkRequestConfig{
									Source: AsyncWorkRequestSourceServiceSDK,
									Phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete},
									LegacyFieldBridge: AsyncLegacyFieldBridge{
										Create: "CreateWorkRequestId",
										Update: "UpdateWorkRequestId",
										Delete: "DeleteWorkRequestId",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestServiceConfigAsyncConfigForMergesServiceAndResourceOverrides(t *testing.T) {
	t.Parallel()

	service := ServiceConfig{
		Service: "queue",
		Async: AsyncConfig{
			Strategy: AsyncStrategyLifecycle,
			Runtime:  AsyncRuntimeGeneratedRuntime,
		},
		Generation: GenerationConfig{
			Resources: []ResourceGenerationOverride{
				{
					Kind: "Queue",
					Async: AsyncConfig{
						Strategy: AsyncStrategyWorkRequest,
						Runtime:  AsyncRuntimeHandwritten,
						WorkRequest: AsyncWorkRequestConfig{
							Source: AsyncWorkRequestSourceServiceSDK,
							Phases: []string{AsyncPhaseCreate, AsyncPhaseDelete},
						},
					},
				},
			},
		},
	}

	queue := service.AsyncConfigFor("Queue")
	if queue.Strategy != AsyncStrategyWorkRequest {
		t.Fatalf("AsyncConfigFor(Queue).Strategy = %q, want %q", queue.Strategy, AsyncStrategyWorkRequest)
	}
	if queue.Runtime != AsyncRuntimeHandwritten {
		t.Fatalf("AsyncConfigFor(Queue).Runtime = %q, want %q", queue.Runtime, AsyncRuntimeHandwritten)
	}
	if queue.FormalClassification != AsyncStrategyWorkRequest {
		t.Fatalf("AsyncConfigFor(Queue).FormalClassification = %q, want %q", queue.FormalClassification, AsyncStrategyWorkRequest)
	}
	if queue.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("AsyncConfigFor(Queue).WorkRequest.Source = %q, want %q", queue.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(queue.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseDelete}) {
		t.Fatalf("AsyncConfigFor(Queue).WorkRequest.Phases = %v", queue.WorkRequest.Phases)
	}

	fallback := service.AsyncConfigFor("Stream")
	if fallback.Strategy != AsyncStrategyLifecycle {
		t.Fatalf("AsyncConfigFor(Stream).Strategy = %q, want %q", fallback.Strategy, AsyncStrategyLifecycle)
	}
	if fallback.Runtime != AsyncRuntimeGeneratedRuntime {
		t.Fatalf("AsyncConfigFor(Stream).Runtime = %q, want %q", fallback.Runtime, AsyncRuntimeGeneratedRuntime)
	}
	if fallback.FormalClassification != AsyncStrategyLifecycle {
		t.Fatalf("AsyncConfigFor(Stream).FormalClassification = %q, want %q", fallback.FormalClassification, AsyncStrategyLifecycle)
	}
}

func TestServiceConfigAsyncConfigForClearsInheritedWorkRequestDefaultsWhenStrategyChanges(t *testing.T) {
	t.Parallel()

	service := ServiceConfig{
		Service: "queue",
		Async: AsyncConfig{
			Strategy: AsyncStrategyWorkRequest,
			Runtime:  AsyncRuntimeHandwritten,
			WorkRequest: AsyncWorkRequestConfig{
				Source: AsyncWorkRequestSourceServiceSDK,
				Phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete},
				LegacyFieldBridge: AsyncLegacyFieldBridge{
					Create: "CreateWorkRequestId",
					Update: "UpdateWorkRequestId",
					Delete: "DeleteWorkRequestId",
				},
			},
		},
		Generation: GenerationConfig{
			Resources: []ResourceGenerationOverride{
				{
					Kind: "Queue",
					Async: AsyncConfig{
						Strategy: AsyncStrategyLifecycle,
						Runtime:  AsyncRuntimeGeneratedRuntime,
					},
				},
				{
					Kind: "Stream",
					Async: AsyncConfig{
						Strategy: AsyncStrategyNone,
						Runtime:  AsyncRuntimeGeneratedRuntime,
					},
				},
			},
		},
	}

	queue := service.AsyncConfigFor("Queue")
	if queue.Strategy != AsyncStrategyLifecycle {
		t.Fatalf("AsyncConfigFor(Queue).Strategy = %q, want %q", queue.Strategy, AsyncStrategyLifecycle)
	}
	if queue.Runtime != AsyncRuntimeGeneratedRuntime {
		t.Fatalf("AsyncConfigFor(Queue).Runtime = %q, want %q", queue.Runtime, AsyncRuntimeGeneratedRuntime)
	}
	if queue.FormalClassification != AsyncStrategyLifecycle {
		t.Fatalf("AsyncConfigFor(Queue).FormalClassification = %q, want %q", queue.FormalClassification, AsyncStrategyLifecycle)
	}
	if queue.WorkRequest.hasOverride() {
		t.Fatalf("AsyncConfigFor(Queue).WorkRequest = %#v, want empty workRequest metadata", queue.WorkRequest)
	}

	stream := service.AsyncConfigFor("Stream")
	if stream.Strategy != AsyncStrategyNone {
		t.Fatalf("AsyncConfigFor(Stream).Strategy = %q, want %q", stream.Strategy, AsyncStrategyNone)
	}
	if stream.Runtime != AsyncRuntimeGeneratedRuntime {
		t.Fatalf("AsyncConfigFor(Stream).Runtime = %q, want %q", stream.Runtime, AsyncRuntimeGeneratedRuntime)
	}
	if stream.FormalClassification != AsyncStrategyNone {
		t.Fatalf("AsyncConfigFor(Stream).FormalClassification = %q, want %q", stream.FormalClassification, AsyncStrategyNone)
	}
	if stream.WorkRequest.hasOverride() {
		t.Fatalf("AsyncConfigFor(Stream).WorkRequest = %#v, want empty workRequest metadata", stream.WorkRequest)
	}

	fallback := service.AsyncConfigFor("Topic")
	if fallback.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("AsyncConfigFor(Topic).WorkRequest.Source = %q, want %q", fallback.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(fallback.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("AsyncConfigFor(Topic).WorkRequest.Phases = %v", fallback.WorkRequest.Phases)
	}
	if fallback.WorkRequest.LegacyFieldBridge.Create != "CreateWorkRequestId" {
		t.Fatalf("AsyncConfigFor(Topic).WorkRequest.LegacyFieldBridge.Create = %q, want CreateWorkRequestId", fallback.WorkRequest.LegacyFieldBridge.Create)
	}
}

func TestServiceConfigAsyncConfigForWorkRequestOverrideReplacesInheritedMetadata(t *testing.T) {
	t.Parallel()

	service := ServiceConfig{
		Service: "queue",
		Async: AsyncConfig{
			Strategy: AsyncStrategyWorkRequest,
			Runtime:  AsyncRuntimeHandwritten,
			WorkRequest: AsyncWorkRequestConfig{
				Source: AsyncWorkRequestSourceServiceSDK,
				Phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate},
				LegacyFieldBridge: AsyncLegacyFieldBridge{
					Create: "CreateWorkRequestId",
					Update: "UpdateWorkRequestId",
				},
			},
		},
		Generation: GenerationConfig{
			Resources: []ResourceGenerationOverride{
				{
					Kind: "Queue",
					Async: AsyncConfig{
						WorkRequest: AsyncWorkRequestConfig{
							Source: AsyncWorkRequestSourceProviderHelper,
							Phases: []string{AsyncPhaseDelete},
						},
					},
				},
			},
		},
	}

	queue := service.AsyncConfigFor("Queue")
	if queue.Strategy != AsyncStrategyWorkRequest {
		t.Fatalf("AsyncConfigFor(Queue).Strategy = %q, want %q", queue.Strategy, AsyncStrategyWorkRequest)
	}
	if queue.Runtime != AsyncRuntimeHandwritten {
		t.Fatalf("AsyncConfigFor(Queue).Runtime = %q, want %q", queue.Runtime, AsyncRuntimeHandwritten)
	}
	if queue.WorkRequest.Source != AsyncWorkRequestSourceProviderHelper {
		t.Fatalf("AsyncConfigFor(Queue).WorkRequest.Source = %q, want %q", queue.WorkRequest.Source, AsyncWorkRequestSourceProviderHelper)
	}
	if !slices.Equal(queue.WorkRequest.Phases, []string{AsyncPhaseDelete}) {
		t.Fatalf("AsyncConfigFor(Queue).WorkRequest.Phases = %v, want [%s]", queue.WorkRequest.Phases, AsyncPhaseDelete)
	}
	if queue.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("AsyncConfigFor(Queue).WorkRequest.LegacyFieldBridge = %#v, want empty legacy bridge", queue.WorkRequest.LegacyFieldBridge)
	}
}

func TestCheckedInConfigSelectedKindsHaveExplicitAsyncContracts(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	services := make(map[string]*ServiceConfig, len(cfg.Services))
	for i := range cfg.Services {
		service := &cfg.Services[i]
		services[service.Service] = service
	}

	expectedByService := map[string]struct {
		strategy string
		runtime  string
	}{
		"accessgovernancecp": {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"adm":                {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"aidocument":         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"ailanguage":         {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"aispeech":           {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"aivision":           {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"analytics":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"apiaccesscontrol":   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"apiplatform":        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"apmcontrolplane":    {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"bds":                {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"budget":             {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"clusterplacementgroups": {
			strategy: AsyncStrategyWorkRequest,
			runtime:  AsyncRuntimeGeneratedRuntime,
		},
		"containerengine":     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"containerinstances":  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeHandwritten},
		"core":                {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"dataflow":            {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"database":            {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"databasemigration":   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"databasetools":       {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"datalabelingservice": {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"datascience":         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"email":               {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"functions":           {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeHandwritten},
		"generativeai":        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"identity":            {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"keymanagement":       {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"loadbalancer":        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"logging":             {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"marketplace":         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"monitoring":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"mysql":               {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"nosql":               {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"ocvp":                {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"objectstorage":       {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"oda":                 {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"opensearch":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"psql":                {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeHandwritten},
		"queue":               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"redis":               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"streaming":           {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"usageapi":            {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
	}

	targets := defaultActiveExplicitSelectedKindTargets(cfg)
	if len(targets) == 0 {
		t.Fatal("defaultActiveExplicitSelectedKindTargets() returned no targets")
	}

	for _, target := range targets {
		service := services[target.Service]
		expected, ok := expectedByService[target.Service]
		if !ok {
			t.Fatalf("missing async expectation for default-active service %q", target.Service)
		}
		assertAsyncContract(t, service, target.Kind, expected.strategy, expected.runtime)
	}

	ailanguage := assertAsyncContract(t, services["ailanguage"], "Project", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if ailanguage.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("ailanguage Project workRequest.source = %q, want %q", ailanguage.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(ailanguage.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("ailanguage Project workRequest.phases = %v", ailanguage.WorkRequest.Phases)
	}
	if ailanguage.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("ailanguage Project workRequest.legacyFieldBridge = %#v, want empty legacy bridge", ailanguage.WorkRequest.LegacyFieldBridge)
	}

	queue := assertAsyncContract(t, services["queue"], "Queue", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if queue.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("queue Queue workRequest.source = %q, want %q", queue.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(queue.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("queue Queue workRequest.phases = %v", queue.WorkRequest.Phases)
	}
	if queue.WorkRequest.LegacyFieldBridge.Create != "CreateWorkRequestId" {
		t.Fatalf("queue Queue create bridge = %q, want CreateWorkRequestId", queue.WorkRequest.LegacyFieldBridge.Create)
	}
	if queue.WorkRequest.LegacyFieldBridge.Update != "UpdateWorkRequestId" {
		t.Fatalf("queue Queue update bridge = %q, want UpdateWorkRequestId", queue.WorkRequest.LegacyFieldBridge.Update)
	}
	if queue.WorkRequest.LegacyFieldBridge.Delete != "DeleteWorkRequestId" {
		t.Fatalf("queue Queue delete bridge = %q, want DeleteWorkRequestId", queue.WorkRequest.LegacyFieldBridge.Delete)
	}

	clusterPlacementGroups := assertAsyncContract(
		t,
		services["clusterplacementgroups"],
		"ClusterPlacementGroup",
		AsyncStrategyWorkRequest,
		AsyncRuntimeGeneratedRuntime,
	)
	if clusterPlacementGroups.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf(
			"clusterplacementgroups ClusterPlacementGroup workRequest.source = %q, want %q",
			clusterPlacementGroups.WorkRequest.Source,
			AsyncWorkRequestSourceServiceSDK,
		)
	}
	if !slices.Equal(clusterPlacementGroups.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf(
			"clusterplacementgroups ClusterPlacementGroup workRequest.phases = %v",
			clusterPlacementGroups.WorkRequest.Phases,
		)
	}
	if clusterPlacementGroups.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf(
			"clusterplacementgroups ClusterPlacementGroup workRequest.legacyFieldBridge = %#v, want empty legacy bridge",
			clusterPlacementGroups.WorkRequest.LegacyFieldBridge,
		)
	}

	redis := assertAsyncContract(t, services["redis"], "RedisCluster", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if redis.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("redis RedisCluster workRequest.source = %q, want %q", redis.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(redis.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("redis RedisCluster workRequest.phases = %v", redis.WorkRequest.Phases)
	}
	if redis.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("redis RedisCluster workRequest.legacyFieldBridge = %#v, want empty legacy bridge", redis.WorkRequest.LegacyFieldBridge)
	}
}

func TestCheckedInAnalyticsConfigPromotesControllerBackedRollout(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	service := serviceConfigsByName(t, cfg, "analytics")["analytics"]

	if service.PackageProfile != PackageProfileControllerBacked {
		t.Fatalf("analytics packageProfile = %q, want %q", service.PackageProfile, PackageProfileControllerBacked)
	}
	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 6)
	assertPrimaryPortOverride(t, service, "AnalyticsInstance", "analyticsinstance", "analytics/analyticsinstance")
	if got := service.AsyncConfigFor("AnalyticsInstance"); got.Strategy != AsyncStrategyLifecycle || got.Runtime != AsyncRuntimeGeneratedRuntime || got.FormalClassification != AsyncStrategyLifecycle {
		t.Fatalf("analytics AnalyticsInstance async = %#v, want lifecycle/generatedruntime", got)
	}
	overrides := overridesByKind(service)
	for _, kind := range []string{"PrivateAccessChannel", "VanityUrl", "WorkRequest", "WorkRequestError", "WorkRequestLog"} {
		assertDisabledResourceOverride(t, service.Service, kind, overrides[kind])
	}
}

func TestCheckedInMutabilityValidationConfigSelectedKindsHaveExplicitAsyncContracts(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "mutability_validation_services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	services := serviceConfigsByName(t, cfg, "analytics", "containerengine", "core", "dataflow", "nosql", "objectstorage")

	assertServiceSelection(t, services["analytics"], true, SelectionModeExplicit, []string{"AnalyticsInstance"})
	assertServiceSelection(t, services["containerengine"], true, SelectionModeExplicit, []string{"NodePool"})
	assertServiceSelection(t, services["core"], true, SelectionModeExplicit, []string{"Instance"})
	assertServiceSelection(t, services["dataflow"], true, SelectionModeExplicit, []string{"Application"})
	assertServiceSelection(t, services["nosql"], true, SelectionModeExplicit, []string{"Table"})
	assertServiceSelection(t, services["objectstorage"], true, SelectionModeExplicit, []string{"Bucket"})

	expectedByService := map[string]struct {
		strategy string
		runtime  string
	}{
		"analytics":       {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"containerengine": {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"core":            {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"dataflow":        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"nosql":           {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"objectstorage":   {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
	}

	targets := defaultActiveExplicitSelectedKindTargets(cfg)
	if len(targets) == 0 {
		t.Fatal("defaultActiveExplicitSelectedKindTargets() returned no targets for mutability validation config")
	}

	coveredServices := make(map[string]struct{}, len(expectedByService))
	for _, target := range targets {
		service := services[target.Service]
		expected, ok := expectedByService[target.Service]
		if !ok {
			t.Fatalf("missing async expectation for mutability validation service %q", target.Service)
		}
		assertAsyncContract(t, service, target.Kind, expected.strategy, expected.runtime)
		coveredServices[target.Service] = struct{}{}
	}

	missingServices := make([]string, 0, len(expectedByService))
	for serviceName := range expectedByService {
		if _, ok := coveredServices[serviceName]; !ok {
			missingServices = append(missingServices, serviceName)
		}
	}
	slices.Sort(missingServices)
	if len(missingServices) != 0 {
		t.Fatalf("mutability validation config services missing selected-kind async coverage: %v", missingServices)
	}
}

func selectionExplicit(enabled bool, includeKinds ...string) SelectionConfig {
	return SelectionConfig{
		Enabled:      boolPtr(enabled),
		Mode:         SelectionModeExplicit,
		IncludeKinds: includeKinds,
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func serviceNames(services []ServiceConfig) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, service.Service)
	}
	return names
}

func requireServices(t *testing.T, cfg *Config, names ...string) map[string]*ServiceConfig {
	t.Helper()

	services := make(map[string]*ServiceConfig, len(names))
	for _, name := range names {
		services[name] = requireService(t, cfg, name)
	}

	return services
}

func requireService(t *testing.T, cfg *Config, name string) *ServiceConfig {
	t.Helper()

	for i := range cfg.Services {
		if cfg.Services[i].Service == name {
			return &cfg.Services[i]
		}
	}

	t.Fatalf("service %q was not found in services.yaml", name)
	return nil
}

func assertServiceGenerationStrategies(t *testing.T, service *ServiceConfig, want generationStrategyExpectations) {
	t.Helper()

	assertGenerationStrategy(t, service.Service, "controller", service.ControllerGenerationStrategy(), want.controller)
	assertGenerationStrategy(t, service.Service, "service-manager", service.ServiceManagerGenerationStrategy(), want.serviceManager)
	assertGenerationStrategy(t, service.Service, "registration", service.RegistrationGenerationStrategy(), want.registration)
	assertGenerationStrategy(t, service.Service, "webhook", service.WebhookGenerationStrategy(), want.webhook)
}

func assertGenerationStrategy(t *testing.T, serviceName string, surface string, got string, want string) {
	t.Helper()

	if got != want {
		t.Fatalf("%s %s strategy = %q, want %q", serviceName, surface, got, want)
	}
}

func assertResourceOverrideCount(t *testing.T, service *ServiceConfig, want int) {
	t.Helper()

	if len(service.Generation.Resources) != want {
		t.Fatalf("%s generation overrides = %d, want %d", service.Service, len(service.Generation.Resources), want)
	}
}

func assertMySQLGenerationOverride(t *testing.T, override ResourceGenerationOverride, wantExtraRBAC []string) {
	t.Helper()

	if override.Kind != "DbSystem" {
		t.Fatalf("mysql override kind = %q, want %q", override.Kind, "DbSystem")
	}
	if override.Controller.MaxConcurrentReconciles != 3 {
		t.Fatalf("mysql maxConcurrentReconciles = %d, want 3", override.Controller.MaxConcurrentReconciles)
	}
	if !slices.Equal(override.Controller.ExtraRBACMarkers, wantExtraRBAC) {
		t.Fatalf("mysql extra RBAC markers = %v, want %v", override.Controller.ExtraRBACMarkers, wantExtraRBAC)
	}
	if override.ServiceManager.PackagePath != "mysql/dbsystem" {
		t.Fatalf("mysql packagePath = %q, want %q", override.ServiceManager.PackagePath, "mysql/dbsystem")
	}
	if !override.ServiceManager.NeedsCredentialClient {
		t.Fatal("mysql needsCredentialClient = false, want true")
	}
}

func mysqlSecretRBACMarkers() []string {
	return []string{
		`groups="",resources=secrets,verbs=get;list;watch;create;update;delete`,
	}
}

func assertDatabaseRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertResourceOverrideCount(t, service, 1)
	override := service.Generation.Resources[0]
	if override.Kind != "AutonomousDatabase" {
		t.Fatalf("database override kind = %q, want %q", override.Kind, "AutonomousDatabase")
	}
	if !slices.Equal(
		override.Controller.ExtraRBACMarkers,
		[]string{
			`groups="",resources=secrets,verbs=get;list;watch`,
		},
	) {
		t.Fatalf("database extra RBAC markers = %v", override.Controller.ExtraRBACMarkers)
	}
	if override.FormalSpec != "databaseautonomousdatabase" {
		t.Fatalf("database formalSpec = %q, want %q", override.FormalSpec, "databaseautonomousdatabase")
	}
	if len(override.SpecFields) != 1 || override.SpecFields[0].Name != "AdminPassword" || override.SpecFields[0].Type != "shared.PasswordSource" {
		t.Fatalf("database specFields = %#v, want secret-backed adminPassword override", override.SpecFields)
	}
	if !slices.Equal(
		service.Package.ExtraResources,
		[]string{
			"../../../config/rbac/autonomousdatabases_editor_role.yaml",
			"../../../config/rbac/autonomousdatabases_viewer_role.yaml",
		},
	) {
		t.Fatalf("database package extraResources = %v", service.Package.ExtraResources)
	}
	if override.ServiceManager.PackagePath != "database/autonomousdatabase" {
		t.Fatalf("database packagePath = %q, want %q", override.ServiceManager.PackagePath, "database/autonomousdatabase")
	}
	if override.Webhooks.Strategy != "" {
		t.Fatalf("database resource webhook strategy = %q, want empty", override.Webhooks.Strategy)
	}
}

func assertFunctionsRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyManual,
		registration:   GenerationStrategyManual,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 2)

	overrides := overridesByKind(service)
	application, ok := overrides["Application"]
	if !ok {
		t.Fatal("functions does not define a generation override for Application")
	}
	if len(application.Controller.ExtraRBACMarkers) != 0 {
		t.Fatalf("functions Application extra RBAC markers = %v, want no non-default markers", application.Controller.ExtraRBACMarkers)
	}
	if application.ServiceManager.PackagePath != "functions" {
		t.Fatalf("functions Application packagePath = %q, want %q", application.ServiceManager.PackagePath, "functions")
	}

	function, ok := overrides["Function"]
	if !ok {
		t.Fatal("functions does not define a generation override for Function")
	}
	if !slices.Equal(function.Controller.ExtraRBACMarkers, []string{`groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`}) {
		t.Fatalf("functions Function extra RBAC markers = %v, want secret-side-effect permissions only", function.Controller.ExtraRBACMarkers)
	}
	if function.ServiceManager.PackagePath != "functions" {
		t.Fatalf("functions Function packagePath = %q, want %q", function.ServiceManager.PackagePath, "functions")
	}
}

func assertMySQLRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertResourceOverrideCount(t, service, 1)
	override := service.Generation.Resources[0]
	assertMySQLGenerationOverride(t, override, mysqlSecretRBACMarkers())
	if len(override.SpecFields) != 2 {
		t.Fatalf("mysql specFields = %#v, want 2 secret-backed overrides", override.SpecFields)
	}
	if len(override.StatusFields) != 2 {
		t.Fatalf("mysql statusFields = %#v, want 2 secret-backed overrides", override.StatusFields)
	}
	if !strings.Contains(override.Sample.Body, "adminPassword:") || !strings.Contains(override.Sample.Body, "secretName: admin-secret") {
		t.Fatalf("mysql sample override = %q, want secret-backed sample body", override.Sample.Body)
	}
	assertFormalSpecFor(t, service, "DbSystem", "dbsystem")
}

func assertNoSQLRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 1)

	override, ok := overridesByKind(service)["Table"]
	if !ok {
		t.Fatal("nosql does not define a generation override for Table")
	}
	if len(override.Controller.ExtraRBACMarkers) != 0 {
		t.Fatalf("nosql Table extra RBAC markers = %v, want no non-default markers", override.Controller.ExtraRBACMarkers)
	}
	assertFormalSpecFor(t, service, "Table", "table")
}

func assertPSQLRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertResourceOverrideCount(t, service, 1)
	override := service.Generation.Resources[0]
	if override.Kind != "DbSystem" {
		t.Fatalf("psql override kind = %q, want %q", override.Kind, "DbSystem")
	}
	if !slices.Equal(
		override.Controller.ExtraRBACMarkers,
		[]string{
			`groups="",resources=secrets,verbs=get;list;watch`,
		},
	) {
		t.Fatalf("psql extra RBAC markers = %v, want secret read markers only", override.Controller.ExtraRBACMarkers)
	}
	if len(override.SpecFields) != 2 {
		t.Fatalf("psql specFields = %#v, want 2 secret-backed overrides", override.SpecFields)
	}
	if len(override.StatusFields) != 2 {
		t.Fatalf("psql statusFields = %#v, want 2 secret-source tracking overrides", override.StatusFields)
	}
	if !strings.Contains(override.Sample.Body, "adminPassword:") || !strings.Contains(override.Sample.Body, "secretName: admin-secret") {
		t.Fatalf("psql sample override = %q, want secret-backed sample body", override.Sample.Body)
	}
	assertFormalSpecFor(t, service, "DbSystem", "dbsystem")
	if override.ServiceManager.PackagePath != "psql/dbsystem" {
		t.Fatalf("psql packagePath = %q, want %q", override.ServiceManager.PackagePath, "psql/dbsystem")
	}
	if !override.ServiceManager.NeedsCredentialClient {
		t.Fatal("psql needsCredentialClient = false, want true")
	}
}

func assertPrimaryPortOverride(t *testing.T, service *ServiceConfig, kind string, formalSpec string, packagePath string) {
	t.Helper()

	overrides := overridesByKind(service)
	override, ok := overrides[kind]
	if !ok {
		t.Fatalf("%s does not define a generation override for %q", service.Service, kind)
	}
	if override.Kind != kind {
		t.Fatalf("%s override kind = %q, want %q", service.Service, override.Kind, kind)
	}
	if override.FormalSpec != formalSpec {
		t.Fatalf("%s %s formalSpec = %q, want %q", service.Service, kind, override.FormalSpec, formalSpec)
	}
	if override.ServiceManager.PackagePath != packagePath {
		t.Fatalf("%s %s packagePath = %q, want %q", service.Service, kind, override.ServiceManager.PackagePath, packagePath)
	}
}

func assertSampleOverrideContains(t *testing.T, service *ServiceConfig, kind string, want ...string) {
	t.Helper()

	override, ok := overridesByKind(service)[kind]
	if !ok {
		t.Fatalf("%s does not define a generation override for %q", service.Service, kind)
	}
	for _, snippet := range want {
		if !strings.Contains(override.Sample.Body, snippet) {
			t.Fatalf("%s %s sample override = %q, want %q", service.Service, kind, override.Sample.Body, snippet)
		}
	}
}

func assertStreamingRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertResourceOverrideCount(t, service, 1)
	overrides := overridesByKind(service)
	assertFormalSpecFor(t, service, "Stream", "stream")
	if overrides["Stream"].ServiceManager.PackagePath != "streaming/stream" {
		t.Fatalf("streaming packagePath = %q, want %q", overrides["Stream"].ServiceManager.PackagePath, "streaming/stream")
	}
}

func assertDataflowRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if service.PackageProfile != PackageProfileControllerBacked {
		t.Fatalf("dataflow packageProfile = %q, want %q", service.PackageProfile, PackageProfileControllerBacked)
	}
	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 1)
	assertPrimaryPortOverride(t, service, "Application", "application", "dataflow/application")
	assertSampleOverrideContains(
		t,
		service,
		"Application",
		`displayName: "application-sample"`,
		`driverShape: "VM.Standard.E4.Flex"`,
		`fileUri: "oci://bucket@namespace/app/main.py"`,
	)
	assertAsyncContract(t, service, "Application", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)
}

func assertAIDocumentRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 6)
	assertAsyncContract(t, service, "Project", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)

	overrides := overridesByKind(service)
	for _, kind := range []string{"Model", "ProcessorJob", "WorkRequest", "WorkRequestError", "WorkRequestLog"} {
		assertDisabledResourceOverride(t, service.Service, kind, overrides[kind])
	}
}

func assertAILanguageRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 8)
	project := assertAsyncContract(t, service, "Project", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if project.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("ailanguage Project workRequest.source = %q, want %q", project.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(project.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("ailanguage Project workRequest.phases = %v", project.WorkRequest.Phases)
	}
	if project.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("ailanguage Project workRequest.legacyFieldBridge = %#v, want empty legacy bridge", project.WorkRequest.LegacyFieldBridge)
	}

	overrides := overridesByKind(service)
	for _, kind := range []string{"Endpoint", "EvaluationResult", "Model", "ModelType", "WorkRequest", "WorkRequestError", "WorkRequestLog"} {
		assertDisabledResourceOverride(t, service.Service, kind, overrides[kind])
	}
}

func assertAISpeechRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 2)
	assertAsyncContract(t, service, "TranscriptionJob", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)

	overrides := overridesByKind(service)
	assertDisabledResourceOverride(t, service.Service, "TranscriptionTask", overrides["TranscriptionTask"])
}

func assertAIVisionRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 7)
	assertAsyncContract(t, service, "Project", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)

	overrides := overridesByKind(service)
	for _, kind := range []string{"DocumentJob", "ImageJob", "Model", "WorkRequest", "WorkRequestError", "WorkRequestLog"} {
		assertDisabledResourceOverride(t, service.Service, kind, overrides[kind])
	}
}

func assertBDSRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 11)
	assertAsyncContract(t, service, "BdsInstance", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)

	overrides := overridesByKind(service)
	for _, kind := range []string{
		"AutoScalingConfiguration",
		"BdsApiKey",
		"BdsMetastoreConfiguration",
		"OsPatch",
		"OsPatchDetail",
		"Patch",
		"PatchHistory",
		"WorkRequest",
		"WorkRequestError",
		"WorkRequestLog",
	} {
		assertDisabledResourceOverride(t, service.Service, kind, overrides[kind])
	}
}

func assertDatabaseToolsRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 6)
	assertAsyncContract(t, service, "DatabaseToolsConnection", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)

	overrides := overridesByKind(service)
	for _, kind := range []string{
		"DatabaseToolsEndpointService",
		"DatabaseToolsPrivateEndpoint",
		"WorkRequest",
		"WorkRequestError",
		"WorkRequestLog",
	} {
		assertDisabledResourceOverride(t, service.Service, kind, overrides[kind])
	}
}

func assertDatabaseMigrationRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 1)

	async := assertAsyncContract(t, service, "Connection", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if async.FormalClassification != AsyncStrategyWorkRequest {
		t.Fatalf("databasemigration Connection formalClassification = %q, want %q", async.FormalClassification, AsyncStrategyWorkRequest)
	}
	if async.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("databasemigration Connection workRequest.source = %q, want %q", async.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(async.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("databasemigration Connection workRequest.phases = %v, want %v", async.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete})
	}
}

func assertDataScienceRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 24)
	assertAsyncContract(t, service, "Project", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)

	overrides := overridesByKind(service)
	for _, kind := range []string{
		"DataSciencePrivateEndpoint",
		"FastLaunchJobConfig",
		"Job",
		"JobArtifact",
		"JobArtifactContent",
		"JobRun",
		"JobShape",
		"Model",
		"ModelArtifact",
		"ModelArtifactContent",
		"ModelDeployment",
		"ModelDeploymentShape",
		"ModelProvenance",
		"ModelVersionSet",
		"NotebookSession",
		"NotebookSessionShape",
		"Pipeline",
		"PipelineRun",
		"StepArtifact",
		"StepArtifactContent",
		"WorkRequest",
		"WorkRequestError",
		"WorkRequestLog",
	} {
		assertDisabledResourceOverride(t, service.Service, kind, overrides[kind])
	}
}

func assertCoreRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if service.PackageProfile != PackageProfileControllerBacked {
		t.Fatalf("core packageProfile = %q, want %q", service.PackageProfile, PackageProfileControllerBacked)
	}
	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 10)
	overrides := overridesByKind(service)
	assertPrimaryPortOverride(t, service, "Instance", "instance", "core/instance")
	assertSampleOverrideContains(t, service, "Drg", `displayName: "drg-sample"`)
	if overrides["Drg"].FormalSpec != "" {
		t.Fatalf("core Drg formalSpec = %q, want empty", overrides["Drg"].FormalSpec)
	}
	if overrides["Drg"].ServiceManager.PackagePath != "" {
		t.Fatalf("core Drg packagePath = %q, want empty", overrides["Drg"].ServiceManager.PackagePath)
	}
	assertSampleOverrideContains(t, service, "InternetGateway", "isEnabled: true", `displayName: "internetgateway-sample"`)
	assertSampleOverrideContains(t, service, "NatGateway", `displayName: "natgateway-sample"`)
	assertSampleOverrideContains(t, service, "NetworkSecurityGroup", `displayName: "networksecuritygroup-sample"`)
	assertSampleOverrideContains(t, service, "RouteTable", "routeRules:", "destinationType: CIDR_BLOCK")
	assertSampleOverrideContains(t, service, "SecurityList", "egressSecurityRules:", "ingressSecurityRules:")
	assertSampleOverrideContains(t, service, "ServiceGateway", "services:", "serviceId: ocid1.service.oc1..exampleuniqueID")
	assertSampleOverrideContains(t, service, "Subnet", "cidrBlock: 10.0.1.0/24", "securityListIds:")
	assertSampleOverrideContains(t, service, "Vcn", "cidrBlocks:", `dnsLabel: "vcnsample"`)
	for _, formal := range []struct {
		kind string
		slug string
	}{
		{kind: "InternetGateway", slug: "internetgateway"},
		{kind: "NatGateway", slug: "natgateway"},
		{kind: "NetworkSecurityGroup", slug: "networksecuritygroup"},
		{kind: "RouteTable", slug: "routetable"},
		{kind: "SecurityList", slug: "securitylist"},
		{kind: "ServiceGateway", slug: "servicegateway"},
		{kind: "Subnet", slug: "subnet"},
		{kind: "Vcn", slug: "vcn"},
	} {
		assertFormalSpecFor(t, service, formal.kind, formal.slug)
	}
	assertPackageSplitContainsKind(t, service, "core-network", "Drg")
}

func assertContainerInstancesRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if service.PackageProfile != PackageProfileControllerBacked {
		t.Fatalf("containerinstances packageProfile = %q, want %q", service.PackageProfile, PackageProfileControllerBacked)
	}
	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyManual,
		registration:   GenerationStrategyManual,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 1)
	assertPrimaryPortOverride(t, service, "ContainerInstance", "", "containerinstance")
}

func assertContainerengineRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if service.PackageProfile != PackageProfileControllerBacked {
		t.Fatalf("containerengine packageProfile = %q, want %q", service.PackageProfile, PackageProfileControllerBacked)
	}
	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 2)
	assertPrimaryPortOverride(t, service, "Cluster", "cluster", "containerengine/cluster")
	assertPrimaryPortOverride(t, service, "NodePool", "nodepool", "containerengine/nodepool")
	assertSampleOverrideContains(t, service, "Cluster", "kubernetesVersion:", "endpointConfig:", "serviceLbSubnetIds:")
}

func assertOCVPRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if service.PackageProfile != PackageProfileControllerBacked {
		t.Fatalf("ocvp packageProfile = %q, want %q", service.PackageProfile, PackageProfileControllerBacked)
	}
	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 3)
	assertPrimaryPortOverride(t, service, "Cluster", "cluster", "ocvp/cluster")
	assertSampleOverrideContains(t, service, "Cluster", "displayName:", "sddcId:", "networkConfiguration:")
	assertFormalSpecFor(t, service, "EsxiHost", "")
	assertPrimaryPortOverride(t, service, "Sddc", "sddc", "ocvp/sddc")
	assertSampleOverrideContains(t, service, "Sddc", "displayName:", "compartmentId:", "initialConfiguration:")
}

func assertPackageSplitContainsKind(t *testing.T, service *ServiceConfig, splitName string, wantKind string) {
	t.Helper()

	for _, split := range service.PackageSplits {
		if split.Name != splitName {
			continue
		}
		if !slices.Contains(split.IncludeKinds, wantKind) {
			t.Fatalf("%s package split %q includeKinds = %v, want %q to be present", service.Service, splitName, split.IncludeKinds, wantKind)
		}
		return
	}

	t.Fatalf("%s does not define package split %q", service.Service, splitName)
}

func assertIdentityRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if service.PackageProfile != PackageProfileControllerBacked {
		t.Fatalf("identity packageProfile = %q, want %q", service.PackageProfile, PackageProfileControllerBacked)
	}
	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertResourceOverrideCount(t, service, 1)
	assertFormalSpecFor(t, service, "Compartment", "compartment")
	assertPrimaryPortOverride(t, service, "Compartment", "compartment", "identity/compartment")
}

func assertRedisRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertServiceGenerationStrategies(t, service, generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	assertPrimaryPortOverride(t, service, "RedisCluster", "rediscluster", "redis/rediscluster")
	if override := overridesByKind(service)["RedisCluster"]; len(override.Controller.ExtraRBACMarkers) != 0 {
		t.Fatalf("redis RedisCluster extra RBAC markers = %v, want no non-default markers", override.Controller.ExtraRBACMarkers)
	}
}

func assertOpensearchRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	assertResourceOverrideCount(t, service, 7)
	overrides := overridesByKind(service)
	assertFormalSpecFor(t, service, "OpensearchCluster", "opensearchopensearchcluster")
	if overrides["OpensearchCluster"].ServiceManager.PackagePath != "opensearch/opensearchcluster" {
		t.Fatalf("opensearch packagePath = %q, want %q", overrides["OpensearchCluster"].ServiceManager.PackagePath, "opensearch/opensearchcluster")
	}
	for _, kind := range []string{"Manifest", "OpensearchClusterBackup", "OpensearchOpensearchVersion", "WorkRequest", "WorkRequestError", "WorkRequestLog"} {
		assertDisabledResourceOverride(t, service.Service, kind, overrides[kind])
	}
}

func assertFormalSpecFor(t *testing.T, service *ServiceConfig, kind string, want string) {
	t.Helper()

	if got := service.FormalSpecFor(kind); got != want {
		t.Fatalf("%s %s formalSpec = %q, want %q", service.Service, kind, got, want)
	}
}

func assertGeneratedRuntimeOptOutKinds(t *testing.T, cfg *Config, serviceName string, kinds []string) {
	t.Helper()

	service := requireService(t, cfg, serviceName)
	overrides := overridesByKind(service)
	for _, kind := range kinds {
		assertDisabledResourceOverride(t, serviceName, kind, overrides[kind])
	}
}

func overridesByKind(service *ServiceConfig) map[string]ResourceGenerationOverride {
	overrides := make(map[string]ResourceGenerationOverride, len(service.Generation.Resources))
	for _, override := range service.Generation.Resources {
		overrides[override.Kind] = override
	}

	return overrides
}

func assertDisabledResourceOverride(t *testing.T, serviceName string, kind string, override ResourceGenerationOverride) {
	t.Helper()

	if override.Kind == "" {
		t.Fatalf("%s override for %s was not found", serviceName, kind)
	}
	if override.Controller.Strategy != GenerationStrategyNone {
		t.Fatalf("%s %s controller strategy = %q, want %q", serviceName, kind, override.Controller.Strategy, GenerationStrategyNone)
	}
	if override.ServiceManager.Strategy != GenerationStrategyNone {
		t.Fatalf("%s %s service-manager strategy = %q, want %q", serviceName, kind, override.ServiceManager.Strategy, GenerationStrategyNone)
	}
}

func TestCheckedInGeneratedServicesWithoutManualWebhooksUseSharedManagerRollout(t *testing.T) {
	t.Parallel()

	servicesPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	servicesCfg, err := LoadConfig(servicesPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", servicesPath, err)
	}

	manualWebhookServices := map[string]struct{}{
		"database":  {},
		"mysql":     {},
		"streaming": {},
	}
	manualRuntimeServices := map[string]struct{}{
		"containerinstances": {},
		"functions":          {},
	}
	promotedNames := make([]string, 0)
	for _, service := range servicesCfg.Services {
		if _, ok := manualWebhookServices[service.Service]; ok {
			continue
		}
		if service.PackageProfile != PackageProfileControllerBacked {
			continue
		}

		promotedNames = append(promotedNames, service.Service)
		if got := service.ControllerGenerationStrategy(); got != GenerationStrategyGenerated {
			t.Fatalf("%s controller strategy = %q, want %q", service.Service, got, GenerationStrategyGenerated)
		}
		if _, ok := manualRuntimeServices[service.Service]; ok {
			if got := service.ServiceManagerGenerationStrategy(); got != GenerationStrategyManual {
				t.Fatalf("%s service-manager strategy = %q, want %q", service.Service, got, GenerationStrategyManual)
			}
			if got := service.RegistrationGenerationStrategy(); got != GenerationStrategyManual {
				t.Fatalf("%s registration strategy = %q, want %q", service.Service, got, GenerationStrategyManual)
			}
			continue
		}
		if got := service.ServiceManagerGenerationStrategy(); got != GenerationStrategyGenerated {
			t.Fatalf("%s service-manager strategy = %q, want %q", service.Service, got, GenerationStrategyGenerated)
		}
		if got := service.RegistrationGenerationStrategy(); got != GenerationStrategyGenerated {
			t.Fatalf("%s registration strategy = %q, want %q", service.Service, got, GenerationStrategyGenerated)
		}
	}
	if len(promotedNames) == 0 {
		t.Fatal("expected at least one promoted service without manual webhooks in services.yaml")
	}
}

func TestCheckedInConfigPromotesStreamingStreamFormalSpec(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	service := serviceConfigsByName(t, cfg, "streaming")["streaming"]

	assertFormalSpecFor(t, service, "Stream", "stream")
}

func TestCheckedInStreamingRuntimeRolloutIncludesSecretRBAC(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	service := serviceConfigsByName(t, cfg, "streaming")["streaming"]

	overrides := overridesByKind(service)
	if !slices.Equal(
		overrides["Stream"].Controller.ExtraRBACMarkers,
		[]string{
			`groups="",resources=secrets,verbs=get;list;watch;create;update;delete`,
		},
	) {
		t.Fatalf("streaming Stream extra RBAC markers = %v", overrides["Stream"].Controller.ExtraRBACMarkers)
	}
}

func TestCheckedInStreamingPackageInstallRoleNarrowsSecretVerbs(t *testing.T) {
	t.Parallel()

	rolePath := filepath.Join(repoRoot(t), "packages", "streaming", "install", "generated", "rbac", "role.yaml")
	content, err := os.ReadFile(rolePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", rolePath, err)
	}

	rendered := string(content)
	assertCoreResourceVerbs(t, rolePath, map[string][]string{
		"events":  {"create", "patch"},
		"secrets": {"create", "delete", "get", "list", "update", "watch"},
	})
	if strings.Contains(rendered, "  - secrets\n  verbs:\n  - create\n  - delete\n  - get\n  - list\n  - patch\n  - update\n  - watch\n") {
		t.Fatalf("streaming package install role still grants patch on secrets:\n%s", rendered)
	}
}

func TestObservedStateStructCandidates(t *testing.T) {
	t.Parallel()

	service := ServiceConfig{
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"ZoneFromZoneFile": {"Zone", "ZoneSummary", "Zone"},
			},
		},
	}

	got := service.ObservedStateStructCandidates("ZoneFromZoneFile")
	want := []string{"ZoneFromZoneFile", "ZoneFromZoneFileSummary", "Zone", "ZoneSummary"}
	if !slices.Equal(got, want) {
		t.Fatalf("ObservedStateStructCandidates() = %v, want %v", got, want)
	}
}

func TestObservedStateStructCandidatesReplacesNormalizedAliasMatch(t *testing.T) {
	t.Parallel()

	service := ServiceConfig{
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"DhcpOption": {"DhcpOptions"},
			},
		},
	}

	dhcpGot := service.ObservedStateStructCandidates("DhcpOption")
	dhcpWant := []string{"DhcpOptions", "DhcpOptionSummary"}
	if !slices.Equal(dhcpGot, dhcpWant) {
		t.Fatalf("ObservedStateStructCandidates(DhcpOption) = %v, want %v", dhcpGot, dhcpWant)
	}
}

func TestObservedStateExcludedFieldPaths(t *testing.T) {
	t.Parallel()

	service := ServiceConfig{
		ObservedState: ObservedStateConfig{
			ExcludedFieldPaths: map[string][]string{
				"DbSystem": {"Source.SourceUrl", " source.sourceURL "},
			},
		},
	}

	got := service.ObservedStateExcludedFieldPaths("DbSystem")
	wantKey, err := normalizeObservedStateFieldPath("Source.SourceUrl")
	if err != nil {
		t.Fatalf("normalizeObservedStateFieldPath() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ObservedStateExcludedFieldPaths() returned %d entries, want 1", len(got))
	}
	if _, ok := got[wantKey]; !ok {
		t.Fatalf("ObservedStateExcludedFieldPaths() = %v, want %q", got, wantKey)
	}
}

func TestCheckedInConfigExcludesMySQLDbSystemSourceURLFromObservedState(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var mysqlService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "mysql" {
			mysqlService = &cfg.Services[i]
			break
		}
	}
	if mysqlService == nil {
		t.Fatal("mysql service was not found in services.yaml")
	}

	excluded := mysqlService.ObservedStateExcludedFieldPaths("DbSystem")
	wantKey, err := normalizeObservedStateFieldPath("Source.SourceUrl")
	if err != nil {
		t.Fatalf("normalizeObservedStateFieldPath() error = %v", err)
	}
	if len(excluded) != 1 {
		t.Fatalf("mysql DbSystem excluded observed-state paths = %v, want exactly %q", excluded, wantKey)
	}
	if _, ok := excluded[wantKey]; !ok {
		t.Fatalf("mysql DbSystem excluded observed-state paths = %v, want %q", excluded, wantKey)
	}
}

func TestCheckedInConfigAddsODAChannelOdaInstanceIDSpecField(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	odaService := requireService(t, cfg, "oda")
	override, ok := odaService.resourceGenerationOverride("Channel")
	if !ok {
		t.Fatal("oda Channel override was not found in services.yaml")
	}

	var odaInstanceID *FieldOverride
	for i := range override.SpecFields {
		if override.SpecFields[i].Name == "OdaInstanceId" {
			odaInstanceID = &override.SpecFields[i]
			break
		}
	}
	if odaInstanceID == nil {
		t.Fatalf("oda Channel specFields = %#v, want OdaInstanceId", override.SpecFields)
	}
	if odaInstanceID.Type != "string" || odaInstanceID.Tag != `json:"odaInstanceId"` {
		t.Fatalf("oda Channel OdaInstanceId override = %#v, want required string json odaInstanceId", *odaInstanceID)
	}
	if !slices.Contains(odaInstanceID.Markers, "+kubebuilder:validation:Required") {
		t.Fatalf("oda Channel OdaInstanceId markers = %v, want required marker", odaInstanceID.Markers)
	}
}

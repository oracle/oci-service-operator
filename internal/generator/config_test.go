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

func TestLoadConfigPreservesStableAPIKindAliases(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "services.yaml")
	content := `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: generated controllers
services:
  - service: apigateway
    sdkPackage: github.com/oracle/oci-go-sdk/v65/apigateway
    group: apigateway
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds: [Deployment, Gateway]
    kindAliases:
      Deployment: ApiGatewayDeployment
      Gateway: ApiGateway
    async:
      strategy: lifecycle
      runtime: generatedruntime
      formalClassification: lifecycle
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	service := cfg.Services[0]
	if got := service.APIKindFor("Gateway"); got != "ApiGateway" {
		t.Fatalf("APIKindFor(Gateway) = %q, want ApiGateway", got)
	}
	if got := service.SDKKindFor("ApiGatewayDeployment"); got != "Deployment" {
		t.Fatalf("SDKKindFor(ApiGatewayDeployment) = %q, want Deployment", got)
	}
	selectedServices, err := cfg.SelectDefaultActiveOrExplicitServices("", false)
	if err != nil {
		t.Fatalf("SelectDefaultActiveOrExplicitServices() error = %v", err)
	}
	selected := selectedServices[0]
	if got := selected.SelectedAPIKinds(); !slices.Equal(got, []string{"ApiGatewayDeployment", "ApiGateway"}) {
		t.Fatalf("SelectedAPIKinds() = %v, want stable API aliases", got)
	}
}

func TestLoadConfigRejectsDuplicateAPIKindAliases(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "services.yaml")
	content := `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: generated controllers
services:
  - service: apigateway
    sdkPackage: github.com/oracle/oci-go-sdk/v65/apigateway
    group: apigateway
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds: [Deployment, Gateway]
    kindAliases:
      Deployment: ApiGateway
      Gateway: ApiGateway
    async:
      strategy: lifecycle
      runtime: generatedruntime
      formalClassification: lifecycle
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil || !strings.Contains(err.Error(), `both "Deployment" and "Gateway" to API kind "ApiGateway"`) {
		t.Fatalf("LoadConfig() error = %v, want duplicate API kind alias failure", err)
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
		"aidataplatform",
		"aidocument",
		"ailanguage",
		"aispeech",
		"aivision",
		"analytics",
		"announcementsservice",
		"apiaccesscontrol",
		"apigateway",
		"apiplatform",
		"apmconfig",
		"apmcontrolplane",
		"apmsynthetics",
		"apmtraces",
		"appmgmtcontrol",
		"artifacts",
		"autoscaling",
		"bastion",
		"batch",
		"bds",
		"blockchain",
		"budget",
		"capacitymanagement",
		"certificatesmanagement",
		"cloudbridge",
		"cloudguard",
		"cloudmigrations",
		"clusterplacementgroups",
		"computecloudatcustomer",
		"computeinstanceagent",
		"containerengine",
		"containerinstances",
		"core",
		"dashboardservice",
		"database",
		"databasemigration",
		"databasetools",
		"datacatalog",
		"dataflow",
		"dataintegration",
		"datalabelingservice",
		"datalabelingservicedataplane",
		"datasafe",
		"datascience",
		"dbmulticloud",
		"delegateaccesscontrol",
		"demandsignal",
		"desktops",
		"devops",
		"dif",
		"disasterrecovery",
		"distributeddatabase",
		"dns",
		"email",
		"emwarehouse",
		"events",
		"filestorage",
		"fleetappsmanagement",
		"fleetsoftwareupdate",
		"functions",
		"fusionapps",
		"gdp",
		"generativeai",
		"generativeaiagent",
		"generativeaiagentruntime",
		"generativeaidata",
		"genericartifactscontent",
		"goldengate",
		"governancerulescontrolplane",
		"healthchecks",
		"identity",
		"integration",
		"iot",
		"jms",
		"jmsjavadownloads",
		"jmsutils",
		"keymanagement",
		"licensemanager",
		"limits",
		"limitsincrease",
		"loadbalancer",
		"lockbox",
		"loganalytics",
		"logging",
		"lustrefilestorage",
		"managedkafka",
		"managementagent",
		"managementdashboard",
		"marketplace",
		"marketplaceprivateoffer",
		"marketplacepublisher",
		"mediaservices",
		"mngdmac",
		"monitoring",
		"multicloud",
		"mysql",
		"networkfirewall",
		"networkloadbalancer",
		"nosql",
		"objectstorage",
		"oce",
		"ocicontrolcenter",
		"ocvp",
		"oda",
		"onesubscription",
		"ons",
		"opa",
		"opensearch",
		"operatoraccesscontrol",
		"opsi",
		"optimizer",
		"osmanagementhub",
		"osubbillingschedule",
		"osuborganizationsubscription",
		"osubsubscription",
		"psa",
		"psql",
		"queue",
		"recovery",
		"redis",
		"resourceanalytics",
		"resourcemanager",
		"resourcescheduler",
		"rover",
		"sch",
		"securityattribute",
		"self",
		"servicecatalog",
		"servicemanagerproxy",
		"stackmonitoring",
		"streaming",
		"tenantmanagercontrolplane",
		"usageapi",
		"vbsinst",
		"visualbuilder",
		"vnmonitoring",
		"vulnerabilityscanning",
		"waa",
		"waas",
		"waf",
		"wlms",
		"zpr",
	}
	if !slices.Equal(activeServices, wantActiveServices) {
		t.Fatalf("DefaultActiveServices() = %v, want %v", activeServices, wantActiveServices)
	}

	services := requireServices(
		t,
		cfg,
		"accessgovernancecp",
		"adm",
		"aidataplatform",
		"aidocument",
		"ailanguage",
		"aispeech",
		"aivision",
		"analytics",
		"announcementsservice",
		"apiaccesscontrol",
		"apigateway",
		"apiplatform",
		"apmconfig",
		"apmcontrolplane",
		"apmsynthetics",
		"apmtraces",
		"appmgmtcontrol",
		"artifacts",
		"autoscaling",
		"bastion",
		"batch",
		"bds",
		"blockchain",
		"budget",
		"capacitymanagement",
		"certificatesmanagement",
		"cloudbridge",
		"cloudguard",
		"cloudmigrations",
		"clusterplacementgroups",
		"computecloudatcustomer",
		"computeinstanceagent",
		"containerengine",
		"containerinstances",
		"core",
		"dashboardservice",
		"database",
		"databasemigration",
		"databasetools",
		"datacatalog",
		"dataflow",
		"dataintegration",
		"datalabelingservice",
		"datalabelingservicedataplane",
		"datasafe",
		"datascience",
		"dbmulticloud",
		"delegateaccesscontrol",
		"demandsignal",
		"desktops",
		"devops",
		"dif",
		"disasterrecovery",
		"distributeddatabase",
		"dns",
		"email",
		"emwarehouse",
		"events",
		"filestorage",
		"fleetappsmanagement",
		"fleetsoftwareupdate",
		"functions",
		"fusionapps",
		"gdp",
		"generativeai",
		"generativeaiagent",
		"generativeaiagentruntime",
		"generativeaidata",
		"genericartifactscontent",
		"goldengate",
		"governancerulescontrolplane",
		"healthchecks",
		"identity",
		"integration",
		"iot",
		"jms",
		"jmsjavadownloads",
		"jmsutils",
		"keymanagement",
		"licensemanager",
		"limits",
		"limitsincrease",
		"loadbalancer",
		"lockbox",
		"loganalytics",
		"logging",
		"lustrefilestorage",
		"managedkafka",
		"managementagent",
		"managementdashboard",
		"marketplace",
		"marketplaceprivateoffer",
		"marketplacepublisher",
		"mediaservices",
		"mngdmac",
		"monitoring",
		"multicloud",
		"mysql",
		"networkfirewall",
		"networkloadbalancer",
		"nosql",
		"objectstorage",
		"oce",
		"ocicontrolcenter",
		"ocvp",
		"oda",
		"onesubscription",
		"ons",
		"opa",
		"opensearch",
		"operatoraccesscontrol",
		"opsi",
		"optimizer",
		"osmanagementhub",
		"osubbillingschedule",
		"osuborganizationsubscription",
		"osubsubscription",
		"psa",
		"psql",
		"queue",
		"recovery",
		"redis",
		"resourceanalytics",
		"resourcemanager",
		"resourcescheduler",
		"rover",
		"sch",
		"securityattribute",
		"self",
		"servicecatalog",
		"servicemanagerproxy",
		"stackmonitoring",
		"streaming",
		"tenantmanagercontrolplane",
		"usageapi",
		"vault",
		"vbsinst",
		"visualbuilder",
		"vnmonitoring",
		"vulnerabilityscanning",
		"waa",
		"waas",
		"waf",
		"wlms",
		"zpr",
	)
	assertServiceSelection(t, services["accessgovernancecp"], true, SelectionModeExplicit, []string{"GovernanceInstance"})
	assertServiceSelection(t, services["adm"], true, SelectionModeExplicit, []string{"KnowledgeBase"})
	assertServiceSelection(t, services["aidataplatform"], true, SelectionModeExplicit, []string{"AiDataPlatform"})
	assertServiceSelection(t, services["aidocument"], true, SelectionModeExplicit, []string{"Project"})
	assertServiceSelection(t, services["ailanguage"], true, SelectionModeExplicit, []string{"Project"})
	assertServiceSelection(t, services["aispeech"], true, SelectionModeExplicit, []string{"TranscriptionJob"})
	assertServiceSelection(t, services["aivision"], true, SelectionModeExplicit, []string{"Project"})
	assertServiceSelection(t, services["analytics"], true, SelectionModeExplicit, []string{"AnalyticsInstance"})
	assertServiceSelection(t, services["announcementsservice"], true, SelectionModeExplicit, []string{"AnnouncementSubscription"})
	assertServiceSelection(t, services["apiaccesscontrol"], true, SelectionModeExplicit, []string{"PrivilegedApiControl"})
	assertServiceSelection(t, services["apiplatform"], true, SelectionModeExplicit, []string{"ApiPlatformInstance"})
	assertServiceSelection(t, services["apmconfig"], true, SelectionModeExplicit, []string{"Config"})
	assertServiceSelection(t, services["apmcontrolplane"], true, SelectionModeExplicit, []string{"ApmDomain"})
	assertServiceSelection(t, services["apmsynthetics"], true, SelectionModeExplicit, []string{"Script"})
	assertServiceSelection(t, services["apmtraces"], true, SelectionModeExplicit, []string{"ScheduledQuery"})
	assertServiceSelection(t, services["appmgmtcontrol"], true, SelectionModeExplicit, []string{"MonitoredInstance"})
	assertServiceSelection(t, services["artifacts"], true, SelectionModeExplicit, []string{"ContainerImageSignature", "ContainerRepository", "Repository"})
	assertServiceSelection(t, services["autoscaling"], true, SelectionModeExplicit, []string{"AutoScalingConfiguration", "AutoScalingPolicy"})
	assertServiceSelection(t, services["bastion"], true, SelectionModeExplicit, []string{"Bastion", "Session"})
	assertServiceSelection(t, services["batch"], true, SelectionModeExplicit, []string{"BatchContext", "BatchJobPool", "BatchTaskEnvironment", "BatchTaskProfile"})
	assertServiceSelection(t, services["bds"], true, SelectionModeExplicit, []string{"BdsInstance"})
	assertServiceSelection(t, services["blockchain"], true, SelectionModeExplicit, []string{"BlockchainPlatform", "Osn", "Peer"})
	assertServiceSelection(t, services["budget"], true, SelectionModeExplicit, []string{"Budget"})
	assertServiceSelection(t, services["capacitymanagement"], true, SelectionModeExplicit, []string{"OccCapacityRequest"})
	assertServiceSelection(t, services["certificatesmanagement"], true, SelectionModeExplicit, []string{"CaBundle"})
	assertServiceSelection(t, services["cloudbridge"], true, SelectionModeExplicit, []string{"Agent", "AgentDependency", "Asset", "AssetSource", "DiscoverySchedule", "Environment", "Inventory"})
	assertServiceSelection(t, services["cloudguard"], true, SelectionModeExplicit, []string{"AdhocQuery", "DataMaskRule", "DataSource", "DetectorRecipe", "DetectorRecipeDetectorRule", "ManagedList", "ResponderRecipe", "SavedQuery", "SecurityRecipe", "SecurityZone", "Target", "TargetDetectorRecipe", "TargetResponderRecipe", "WlpAgent"})
	assertServiceSelection(t, services["cloudmigrations"], true, SelectionModeExplicit, []string{"Migration", "MigrationAsset", "MigrationPlan", "ReplicationSchedule", "TargetAsset"})
	assertServiceSelection(t, services["clusterplacementgroups"], true, SelectionModeExplicit, []string{"ClusterPlacementGroup"})
	assertServiceSelection(t, services["computecloudatcustomer"], true, SelectionModeExplicit, []string{"CccInfrastructure", "CccUpgradeSchedule"})
	assertServiceSelection(t, services["computeinstanceagent"], true, SelectionModeExplicit, []string{"InstanceAgentPlugin"})
	assertServiceSelection(t, services["containerengine"], true, SelectionModeExplicit, []string{"Cluster", "NodePool"})
	assertServiceSelection(t, services["containerinstances"], true, SelectionModeExplicit, []string{"ContainerInstance"})
	assertServiceSelection(t, services["core"], true, SelectionModeExplicit, []string{"Instance"})
	assertServiceSelection(t, services["dashboardservice"], true, SelectionModeExplicit, []string{"DashboardGroup", "Dashboard"})
	assertServiceSelection(t, services["database"], true, SelectionModeExplicit, []string{"AutonomousDatabase"})
	assertServiceSelection(t, services["databasemigration"], true, SelectionModeExplicit, []string{"Connection", "Assessment", "Migration"})
	assertServiceSelection(t, services["databasetools"], true, SelectionModeExplicit, []string{"DatabaseToolsConnection"})
	assertServiceSelection(t, services["datacatalog"], true, SelectionModeExplicit, []string{"Attribute", "AttributeTag", "Catalog", "CatalogPrivateEndpoint", "Connection", "CustomProperty", "DataAsset", "DataAssetTag", "Entity", "EntityTag", "Folder", "FolderTag", "Glossary", "Job", "JobDefinition", "Metastore", "Namespace", "Pattern", "Term", "TermRelationship"})
	assertServiceSelection(t, services["dataflow"], true, SelectionModeExplicit, []string{"Application"})
	assertServiceSelection(t, services["dataintegration"], true, SelectionModeExplicit, []string{"Application", "ApplicationDetailedDescription", "Connection", "ConnectionValidation", "CopyObjectRequest", "DataAsset", "DataFlow", "DataFlowValidation", "DisApplication", "DisApplicationDetailedDescription", "ExportRequest", "ExternalPublication", "ExternalPublicationValidation", "Folder", "FunctionLibrary", "ImportRequest", "Patch", "Pipeline", "PipelineValidation", "Project", "Schedule", "Task", "TaskRun", "TaskSchedule", "TaskValidation", "UserDefinedFunction", "UserDefinedFunctionValidation", "Workspace"})
	assertServiceSelection(t, services["datalabelingservice"], true, SelectionModeExplicit, []string{"Dataset"})
	assertServiceSelection(t, services["datalabelingservicedataplane"], true, SelectionModeExplicit, []string{"Annotation", "Record"})
	assertServiceSelection(t, services["datasafe"], true, SelectionModeExplicit, []string{"AlertPolicy", "AlertPolicyRule", "AttributeSet", "AuditArchiveRetrieval", "AuditProfile", "DataSafePrivateEndpoint", "DiscoveryJob", "LibraryMaskingFormat", "MaskingColumn", "MaskingPolicy", "OnPremConnector", "PeerTargetDatabase", "ReferentialRelation", "ReportDefinition", "SdmMaskingPolicyDifference", "SecurityAssessment", "SecurityPolicy", "SecurityPolicyConfig", "SecurityPolicyDeployment", "SensitiveColumn", "SensitiveDataModel", "SensitiveType", "SensitiveTypeGroup", "SensitiveTypesExport", "SqlCollection", "TargetAlertPolicyAssociation", "TargetDatabase", "TargetDatabaseGroup", "UnifiedAuditPolicy", "UserAssessment"})
	assertServiceSelection(t, services["datascience"], true, SelectionModeExplicit, []string{"Project"})
	assertServiceSelection(t, services["dbmulticloud"], true, SelectionModeExplicit, []string{"MultiCloudResourceDiscovery", "OracleDbAwsIdentityConnector", "OracleDbAwsKey", "OracleDbAzureBlobContainer", "OracleDbAzureBlobMount", "OracleDbAzureConnector", "OracleDbAzureVault", "OracleDbAzureVaultAssociation", "OracleDbGcpIdentityConnector", "OracleDbGcpKeyRing"})
	assertServiceSelection(t, services["delegateaccesscontrol"], true, SelectionModeExplicit, []string{"DelegationControl"})
	assertServiceSelection(t, services["demandsignal"], true, SelectionModeExplicit, []string{"OccDemandSignal"})
	assertServiceSelection(t, services["desktops"], true, SelectionModeExplicit, []string{"DesktopPool"})
	assertServiceSelection(t, services["devops"], true, SelectionModeExplicit, []string{"Project", "Repository", "BuildPipeline", "DeployPipeline", "DeployArtifact", "Trigger"})
	assertServiceSelection(t, services["dif"], true, SelectionModeExplicit, []string{"Stack"})
	assertServiceSelection(t, services["disasterrecovery"], true, SelectionModeExplicit, []string{"DrProtectionGroup", "DrPlan"})
	assertServiceSelection(t, services["distributeddatabase"], true, SelectionModeExplicit, []string{"DistributedDatabasePrivateEndpoint", "DistributedDatabase"})
	assertServiceSelection(t, services["dns"], true, SelectionModeExplicit, []string{"Zone", "View", "TsigKey", "SteeringPolicy", "SteeringPolicyAttachment"})
	assertServiceSelection(t, services["email"], true, SelectionModeExplicit, []string{"Dkim", "EmailDomain", "Sender", "Suppression"})
	assertServiceSelection(t, services["emwarehouse"], true, SelectionModeExplicit, []string{"EmWarehouse"})
	assertServiceSelection(t, services["events"], true, SelectionModeExplicit, []string{"Rule"})
	assertServiceSelection(t, services["filestorage"], true, SelectionModeExplicit, []string{"Export", "FileSystem", "FilesystemSnapshotPolicy", "MountTarget", "OutboundConnector", "QuotaRule", "Replication", "Snapshot"})
	assertServiceSelection(t, services["fleetappsmanagement"], true, SelectionModeExplicit, []string{"CatalogItem", "CompliancePolicyRule", "Fleet", "FleetCredential", "FleetProperty", "FleetResource", "MaintenanceWindow", "Onboarding", "Patch", "PlatformConfiguration", "Property", "Provision", "Runbook", "RunbookVersion", "SchedulerDefinition", "TaskRecord"})
	assertServiceSelection(t, services["fleetsoftwareupdate"], true, SelectionModeExplicit, []string{"FsuAction", "FsuCollection", "FsuCycle", "FsuDiscovery", "FsuReadinessCheck"})
	assertServiceSelection(t, services["functions"], true, SelectionModeExplicit, []string{"Application", "Function"})
	assertServiceSelection(t, services["fusionapps"], true, SelectionModeExplicit, []string{"FusionEnvironment", "FusionEnvironmentFamily", "RefreshActivity", "ServiceAttachment"})
	assertServiceSelection(t, services["gdp"], true, SelectionModeExplicit, []string{"GdpPipeline"})
	assertServiceSelection(t, services["generativeai"], true, SelectionModeExplicit, []string{"DedicatedAiCluster", "Endpoint", "Model"})
	assertServiceSelection(t, services["generativeaiagent"], true, SelectionModeExplicit, []string{"KnowledgeBase", "Agent", "AgentEndpoint", "DataSource"})
	assertServiceSelection(t, services["generativeaiagentruntime"], true, SelectionModeExplicit, []string{"Session"})
	assertServiceSelection(t, services["generativeaidata"], true, SelectionModeExplicit, []string{"EnrichmentJob"})
	assertServiceSelection(t, services["genericartifactscontent"], true, SelectionModeExplicit, []string{"GenericArtifactContent", "GenericArtifactContentByPath"})
	assertServiceSelection(t, services["goldengate"], true, SelectionModeExplicit, []string{"Certificate", "Connection", "ConnectionAssignment", "DatabaseRegistration", "Deployment", "DeploymentBackup", "Pipeline"})
	assertServiceSelection(t, services["governancerulescontrolplane"], true, SelectionModeExplicit, []string{"GovernanceRule", "InclusionCriterion"})
	assertServiceSelection(t, services["healthchecks"], true, SelectionModeExplicit, []string{"HttpMonitor", "PingMonitor"})
	assertServiceSelection(t, services["identity"], true, SelectionModeExplicit, []string{"Compartment"})
	assertServiceSelection(t, services["integration"], true, SelectionModeExplicit, []string{"IntegrationInstance"})
	assertServiceSelection(t, services["iot"], true, SelectionModeExplicit, []string{"DigitalTwinAdapter", "DigitalTwinInstance", "DigitalTwinModel", "DigitalTwinRelationship", "IotDomain", "IotDomainGroup"})
	assertServiceSelection(t, services["jms"], true, SelectionModeExplicit, []string{"Fleet", "JmsPlugin"})
	assertServiceSelection(t, services["jmsjavadownloads"], true, SelectionModeExplicit, []string{"JavaDownloadToken"})
	assertServiceSelection(t, services["jmsutils"], true, SelectionModeExplicit, []string{"AnalyzeApplicationsConfiguration", "JavaMigrationAnalysis", "PerformanceTuningAnalysis", "SubscriptionAcknowledgmentConfiguration", "WorkItem"})
	assertServiceSelection(t, services["keymanagement"], true, SelectionModeExplicit, []string{"EkmsPrivateEndpoint", "Vault"})
	assertServiceSelection(t, services["licensemanager"], true, SelectionModeExplicit, []string{"LicenseRecord", "ProductLicense"})
	assertServiceSelection(t, services["limits"], true, SelectionModeExplicit, []string{"Quota"})
	assertServiceSelection(t, services["limitsincrease"], true, SelectionModeExplicit, []string{"LimitsIncreaseRequest"})
	assertServiceSelection(t, services["loadbalancer"], true, SelectionModeExplicit, []string{"Backend", "BackendSet", "Certificate", "Hostname", "Listener", "LoadBalancer", "PathRouteSet", "RoutingPolicy", "RuleSet", "SSLCipherSuite"})
	assertServiceSelection(t, services["lockbox"], true, SelectionModeExplicit, []string{"ApprovalTemplate", "Lockbox"})
	assertServiceSelection(t, services["loganalytics"], true, SelectionModeExplicit, []string{"IngestTimeRule", "LogAnalyticsEmBridge", "LogAnalyticsEntity", "LogAnalyticsEntityType", "LogAnalyticsLogGroup", "LogAnalyticsObjectCollectionRule", "ScheduledTask"})
	assertServiceSelection(t, services["logging"], true, SelectionModeExplicit, []string{"Log", "LogGroup", "LogSavedSearch", "UnifiedAgentConfiguration"})
	assertServiceSelection(t, services["lustrefilestorage"], true, SelectionModeExplicit, []string{"LustreFileSystem", "ObjectStorageLink"})
	assertServiceSelection(t, services["managedkafka"], true, SelectionModeExplicit, []string{"KafkaCluster", "KafkaClusterConfig"})
	assertServiceSelection(t, services["managementagent"], true, SelectionModeExplicit, []string{"DataSource", "ManagementAgentInstallKey", "NamedCredential"})
	assertServiceSelection(t, services["managementdashboard"], true, SelectionModeExplicit, []string{"ManagementDashboard", "ManagementSavedSearch"})
	assertServiceSelection(t, services["marketplace"], true, SelectionModeExplicit, []string{"AcceptedAgreement", "Publication"})
	assertServiceSelection(t, services["marketplaceprivateoffer"], true, SelectionModeExplicit, []string{"Attachment", "Offer"})
	assertServiceSelection(t, services["marketplacepublisher"], true, SelectionModeExplicit, []string{"Artifact", "Listing", "ListingRevision", "ListingRevisionAttachment", "ListingRevisionNote", "ListingRevisionPackage", "Term", "TermVersion"})
	assertServiceSelection(t, services["mediaservices"], true, SelectionModeExplicit, []string{"MediaAsset", "MediaWorkflow", "MediaWorkflowConfiguration"})
	assertServiceSelection(t, services["mngdmac"], true, SelectionModeExplicit, []string{"MacOrder", "MacDevice"})
	assertServiceSelection(t, services["monitoring"], true, SelectionModeExplicit, []string{"Alarm", "AlarmSuppression"})
	assertServiceSelection(t, services["multicloud"], true, SelectionModeExplicit, []string{"ExternalLocationDetailsMetadata", "ExternalLocationMappingMetadata", "ExternalLocationSummariesMetadata", "MultiCloudMetadata", "MulticloudResource", "MulticloudSubscription", "NetworkAnchor", "ResourceAnchor"})
	assertServiceSelection(t, services["mysql"], true, SelectionModeExplicit, []string{"DbSystem"})
	assertServiceSelection(t, services["networkfirewall"], true, SelectionModeExplicit, []string{"AddressList", "Application", "ApplicationGroup", "DecryptionProfile", "DecryptionRule", "MappedSecret", "NatRule", "NetworkFirewall", "NetworkFirewallPolicy", "SecurityRule", "Service", "ServiceList", "TunnelInspectionRule", "UrlList"})
	assertServiceSelection(t, services["networkloadbalancer"], true, SelectionModeExplicit, []string{"Backend", "BackendSet", "Listener", "NetworkLoadBalancer"})
	assertServiceSelection(t, services["nosql"], true, SelectionModeExplicit, []string{"Table"})
	assertServiceSelection(t, services["objectstorage"], true, SelectionModeExplicit, []string{"Bucket"})
	assertServiceSelection(t, services["oce"], true, SelectionModeExplicit, []string{"OceInstance"})
	assertServiceSelection(t, services["ocicontrolcenter"], true, SelectionModeExplicit, []string{"MetricProperty", "Namespace"})
	assertServiceSelection(t, services["ocvp"], true, SelectionModeExplicit, []string{"Cluster", "EsxiHost", "Sddc"})
	assertServiceSelection(t, services["oda"], true, SelectionModeExplicit, []string{"AuthenticationProvider", "Channel", "DigitalAssistant", "ImportedPackage", "OdaInstance", "OdaInstanceAttachment", "OdaPrivateEndpoint", "OdaPrivateEndpointAttachment", "OdaPrivateEndpointScanProxy", "Skill", "SkillParameter", "Translator"})
	assertServiceSelection(t, services["onesubscription"], true, SelectionModeExplicit, []string{"Subscription"})
	assertServiceSelection(t, services["ons"], true, SelectionModeExplicit, []string{"Subscription", "Topic"})
	assertServiceSelection(t, services["opa"], true, SelectionModeExplicit, []string{"OpaInstance"})
	assertServiceSelection(t, services["opensearch"], true, SelectionModeExplicit, []string{"OpensearchCluster"})
	assertServiceSelection(t, services["operatoraccesscontrol"], true, SelectionModeExplicit, []string{"OperatorControl", "OperatorControlAssignment"})
	assertServiceSelection(t, services["opsi"], true, SelectionModeExplicit, []string{"AwrHub", "AwrHubSource", "ChargebackPlan", "ChargebackPlanReport", "DatabaseInsight", "EnterpriseManagerBridge", "ExadataInsight", "HostInsight", "NewsReport", "OperationsInsightsPrivateEndpoint", "OperationsInsightsWarehouse", "OperationsInsightsWarehouseUser", "OpsiConfiguration"})
	assertServiceSelection(t, services["optimizer"], true, SelectionModeExplicit, []string{"Profile"})
	assertServiceSelection(t, services["osmanagementhub"], true, SelectionModeExplicit, []string{"LifecycleEnvironment", "ManagedInstanceGroup", "ManagementStation", "Profile", "ScheduledJob", "SoftwareSource"})
	assertServiceSelection(t, services["osubsubscription"], true, SelectionModeExplicit, []string{"Subscription"})
	assertServiceSelection(t, services["psa"], true, SelectionModeExplicit, []string{"PrivateServiceAccess"})
	assertServiceSelection(t, services["psql"], true, SelectionModeExplicit, []string{"DbSystem"})
	assertServiceSelection(t, services["queue"], true, SelectionModeExplicit, []string{"Queue"})
	assertServiceSelection(t, services["recovery"], true, SelectionModeExplicit, []string{"ProtectedDatabase", "ProtectionPolicy", "RecoveryServiceSubnet"})
	assertServiceSelection(t, services["redis"], true, SelectionModeExplicit, []string{"RedisCluster"})
	assertServiceSelection(t, services["resourceanalytics"], true, SelectionModeExplicit, []string{"MonitoredRegion", "ResourceAnalyticsInstance", "TenancyAttachment"})
	assertServiceSelection(t, services["resourcemanager"], true, SelectionModeExplicit, []string{"ConfigurationSourceProvider", "PrivateEndpoint", "Stack", "Template"})
	assertServiceSelection(t, services["resourcescheduler"], true, SelectionModeExplicit, []string{"Schedule"})
	assertServiceSelection(t, services["rover"], true, SelectionModeExplicit, []string{"RoverCluster", "RoverNode"})
	assertServiceSelection(t, services["sch"], true, SelectionModeExplicit, []string{"ServiceConnector"})
	assertServiceSelection(t, services["securityattribute"], true, SelectionModeExplicit, []string{"SecurityAttribute", "SecurityAttributeNamespace"})
	assertServiceSelection(t, services["self"], true, SelectionModeExplicit, []string{"Subscription"})
	assertServiceSelection(t, services["servicecatalog"], true, SelectionModeExplicit, []string{"PrivateApplication", "ServiceCatalog"})
	assertServiceSelection(t, services["servicemanagerproxy"], true, SelectionModeExplicit, []string{"ServiceEnvironment"})
	assertServiceSelection(t, services["stackmonitoring"], true, SelectionModeExplicit, []string{"AlarmCondition", "BaselineableMetric", "Config", "DiscoveryJob", "MaintenanceWindow", "MetricExtension", "MonitoredResource", "MonitoredResourceType", "MonitoringTemplate", "ProcessSet"})
	assertServiceSelection(t, services["streaming"], true, SelectionModeExplicit, []string{"Stream"})
	assertServiceSelection(t, services["tenantmanagercontrolplane"], true, SelectionModeExplicit, []string{"Domain", "Organization", "DomainGovernance"})
	assertServiceSelection(t, services["usageapi"], true, SelectionModeExplicit, []string{"CustomTable", "Query", "Schedule", "UsageCarbonEmissionsQuery"})
	assertServiceSelection(t, services["vault"], false, SelectionModeAll, nil)
	assertServiceSelection(t, services["vbsinst"], true, SelectionModeExplicit, []string{"VbsInstance"})
	assertServiceSelection(t, services["visualbuilder"], true, SelectionModeExplicit, []string{"VbInstance"})
	assertServiceSelection(t, services["vnmonitoring"], true, SelectionModeExplicit, []string{"PathAnalyzerTest"})
	assertServiceSelection(t, services["vulnerabilityscanning"], true, SelectionModeExplicit, []string{"ContainerScanRecipe", "ContainerScanTarget", "HostScanRecipe", "HostScanTarget"})
	assertServiceSelection(t, services["waa"], true, SelectionModeExplicit, []string{"WebAppAcceleration", "WebAppAccelerationPolicy"})
	assertServiceSelection(t, services["waas"], true, SelectionModeExplicit, []string{"AddressList", "Certificate", "CustomProtectionRule", "HttpRedirect", "WaasPolicy"})
	assertServiceSelection(t, services["waf"], true, SelectionModeExplicit, []string{"NetworkAddressList", "WebAppFirewall", "WebAppFirewallPolicy"})
	assertServiceSelection(t, services["wlms"], true, SelectionModeExplicit, []string{"WlsDomain", "ManagedInstance"})
	assertServiceSelection(t, services["zpr"], true, SelectionModeExplicit, []string{"Configuration", "ZprPolicy"})
}

func TestCheckedInVulnerabilityScanningKeepsOptionalApplicationSettingsNullable(t *testing.T) {
	t.Parallel()

	service := requireService(t, loadCheckedInConfig(t), "vulnerabilityscanning")
	override := overridesByKind(service)["HostScanRecipe"]
	if override.Kind == "" {
		t.Fatal("vulnerabilityscanning HostScanRecipe override was not found")
	}

	assertNullable := func(surface string, fields []FieldOverride) {
		t.Helper()
		for _, field := range fields {
			if field.Name != "ApplicationSettings" {
				continue
			}
			if field.Type != "*HostScanRecipeApplicationSettings" {
				t.Fatalf("%s ApplicationSettings type = %q, want nullable helper pointer", surface, field.Type)
			}
			if field.Tag != `json:"applicationSettings,omitempty"` {
				t.Fatalf("%s ApplicationSettings tag = %q, want omitempty", surface, field.Tag)
			}
			return
		}
		t.Fatalf("%s ApplicationSettings override was not found", surface)
	}

	assertNullable("spec", override.SpecFields)
	assertNullable("status", override.StatusFields)
}

func TestCheckedInAPMTracesRequiresScheduledQueryRetentionAndPublishesValidSample(t *testing.T) {
	t.Parallel()

	service := requireService(t, loadCheckedInConfig(t), "apmtraces")
	override := overridesByKind(service)["ScheduledQuery"]
	var retention *FieldOverride
	for index := range override.SpecFields {
		if override.SpecFields[index].Name == "ScheduledQueryRetentionCriteria" {
			retention = &override.SpecFields[index]
			break
		}
	}
	if retention == nil || retention.Tag != `json:"scheduledQueryRetentionCriteria"` || !slices.Contains(retention.Markers, "+kubebuilder:validation:Required") {
		t.Fatalf("apmtraces ScheduledQuery retention override = %#v, want required field", retention)
	}
	assertSampleOverrideContains(t, service, "ScheduledQuery", "scheduledQueryProcessingSubType: NONE", "KEEP_DATA_UNTIL_RETENTION_PERIOD", "EVERY 720 MINUTES")
}

func TestCheckedInLogAnalyticsMakesIngestTimeRuleIDOptional(t *testing.T) {
	t.Parallel()

	service := requireService(t, loadCheckedInConfig(t), "loganalytics")
	override := overridesByKind(service)["IngestTimeRule"]
	for _, field := range override.SpecFields {
		if field.Name != "Id" {
			continue
		}
		if field.Tag != `json:"id,omitempty"` || !slices.Contains(field.Markers, "+kubebuilder:validation:Optional") {
			t.Fatalf("loganalytics IngestTimeRule Id override = %#v, want optional omitempty field", field)
		}
		return
	}
	t.Fatalf("loganalytics IngestTimeRule specFields = %#v, want Id override", override.SpecFields)
}

func TestCheckedInNetworkFirewallPublishesChildIdentityAndNullableNatConfiguration(t *testing.T) {
	t.Parallel()

	service := requireService(t, loadCheckedInConfig(t), "networkfirewall")
	overrides := overridesByKind(service)
	findField := func(kind, name string, fields []FieldOverride) FieldOverride {
		t.Helper()
		for _, field := range fields {
			if field.Name == name {
				return field
			}
		}
		t.Fatalf("networkfirewall %s %s override was not found", kind, name)
		return FieldOverride{}
	}
	for _, kind := range []string{"AddressList", "Application", "ApplicationGroup", "DecryptionProfile", "DecryptionRule", "MappedSecret", "NatRule", "SecurityRule", "Service", "ServiceList", "TunnelInspectionRule", "UrlList"} {
		field := findField(kind, "NetworkFirewallPolicyId", overrides[kind].SpecFields)
		if field.Type != "string" || field.Tag != `json:"networkFirewallPolicyId"` {
			t.Fatalf("networkfirewall %s parent field = %#v", kind, field)
		}
	}
	networkFirewall := overrides["NetworkFirewall"]
	for surface, fields := range map[string][]FieldOverride{"spec": networkFirewall.SpecFields, "status": networkFirewall.StatusFields} {
		field := findField("NetworkFirewall", "NatConfiguration", fields)
		if field.Type != "*NetworkFirewallNatConfiguration" || field.Tag != `json:"natConfiguration,omitempty"` {
			t.Fatalf("networkfirewall NetworkFirewall %s NatConfiguration = %#v", surface, field)
		}
	}
}

func TestCheckedInConfigIncludesRuntimeRolloutMetadata(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	services := serviceConfigsByName(t, cfg, "aidocument", "ailanguage", "aispeech", "aivision", "autoscaling", "bds", "cloudguard", "containerengine", "containerinstances", "core", "dataflow", "database", "databasemigration", "databasetools", "datalabelingservice", "datascience", "disasterrecovery", "distributeddatabase", "functions", "generativeaiagent", "healthchecks", "identity", "jms", "keymanagement", "mediaservices", "mysql", "nosql", "oce", "ocvp", "psql", "redis", "streaming", "tenantmanagercontrolplane")
	assertAIDocumentRuntimeRolloutMetadata(t, services["aidocument"])
	assertAILanguageRuntimeRolloutMetadata(t, services["ailanguage"])
	assertAISpeechRuntimeRolloutMetadata(t, services["aispeech"])
	assertAIVisionRuntimeRolloutMetadata(t, services["aivision"])
	assertAsyncContract(t, services["autoscaling"], "AutoScalingConfiguration", AsyncStrategyNone, AsyncRuntimeGeneratedRuntime)
	assertAsyncContract(t, services["cloudguard"], "WlpAgent", AsyncStrategyNone, AsyncRuntimeGeneratedRuntime)
	assertBDSRuntimeRolloutMetadata(t, services["bds"])
	assertDatabaseMigrationRuntimeRolloutMetadata(t, services["databasemigration"])
	assertDatabaseToolsRuntimeRolloutMetadata(t, services["databasetools"])
	assertDataScienceRuntimeRolloutMetadata(t, services["datascience"])
	assertAsyncContract(t, services["disasterrecovery"], "DrPlan", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	assertAsyncContract(t, services["distributeddatabase"], "DistributedDatabase", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)
	assertAsyncContract(t, services["generativeaiagent"], "AgentEndpoint", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	assertAsyncContract(t, services["generativeaiagent"], "DataSource", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	assertAsyncContract(t, services["healthchecks"], "HttpMonitor", AsyncStrategyNone, AsyncRuntimeGeneratedRuntime)
	assertAsyncContract(t, services["jms"], "JmsPlugin", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)
	assertAsyncContract(t, services["mediaservices"], "MediaWorkflowConfiguration", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)
	assertAsyncContract(t, services["tenantmanagercontrolplane"], "DomainGovernance", AsyncStrategyLifecycle, AsyncRuntimeGeneratedRuntime)

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
	assertServiceGenerationStrategies(t, services["oce"], generationStrategyExpectations{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
	oceAsync := assertAsyncContract(t, services["oce"], "OceInstance", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if oceAsync.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("oce OceInstance workRequest.source = %q, want %q", oceAsync.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(oceAsync.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("oce OceInstance workRequest.phases = %v", oceAsync.WorkRequest.Phases)
	}
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

func TestCheckedInAutoscalingPolicyEnabledPreservesPresence(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	service := serviceConfigsByName(t, cfg, "autoscaling")["autoscaling"]
	assertPointerBool := func(kind, name string) {
		t.Helper()
		override := overridesByKind(service)[kind]
		for _, field := range override.SpecFields {
			if field.Name != name {
				continue
			}
			if field.Type != "*bool" || field.Tag != `json:"isEnabled,omitempty"` {
				t.Fatalf("%s %s override = %#v, want pointer bool with optional JSON tag", kind, name, field)
			}
			return
		}
		t.Fatalf("autoscaling %s is missing %s presence override", kind, name)
	}
	assertPointerBool("AutoScalingConfiguration", "IsEnabled")
	assertPointerBool("AutoScalingConfiguration", "Policies.IsEnabled")
	assertPointerBool("AutoScalingPolicy", "IsEnabled")
}

func TestCheckedInConfigPromotesFormalSpecReferences(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	services := serviceConfigsByName(t, cfg, "aidocument", "ailanguage", "aispeech", "aivision", "analytics", "apiaccesscontrol", "apigateway", "autoscaling", "bds", "cloudguard", "containerengine", "containerinstances", "core", "database", "databasemigration", "databasetools", "datalabelingservice", "datascience", "dataflow", "disasterrecovery", "distributeddatabase", "generativeaiagent", "healthchecks", "identity", "jms", "mediaservices", "mysql", "objectstorage", "oce", "ocvp", "opa", "opensearch", "psql", "redis", "streaming", "tenantmanagercontrolplane")
	assertFormalSpecFor(t, services["aidocument"], "Project", "project")
	assertFormalSpecFor(t, services["ailanguage"], "Project", "project")
	assertFormalSpecFor(t, services["aispeech"], "TranscriptionJob", "transcriptionjob")
	assertFormalSpecFor(t, services["aivision"], "Project", "project")
	assertFormalSpecFor(t, services["analytics"], "AnalyticsInstance", "analyticsinstance")
	assertFormalSpecFor(t, services["apiaccesscontrol"], "PrivilegedApiControl", "privilegedapicontrol")
	assertFormalSpecFor(t, services["apigateway"], "ApiGateway", "gateway")
	assertFormalSpecFor(t, services["apigateway"], "ApiGatewayDeployment", "deployment")
	assertFormalSpecFor(t, services["autoscaling"], "AutoScalingConfiguration", "autoscalingconfiguration")
	assertFormalSpecFor(t, services["bds"], "BdsInstance", "bdsinstance")
	assertFormalSpecFor(t, services["cloudguard"], "SavedQuery", "savedquery")
	assertFormalSpecFor(t, services["cloudguard"], "WlpAgent", "wlpagent")
	assertFormalSpecFor(t, services["containerengine"], "Cluster", "cluster")
	assertFormalSpecFor(t, services["containerengine"], "NodePool", "nodepool")
	assertFormalSpecFor(t, services["containerinstances"], "ContainerInstance", "")
	assertFormalSpecFor(t, services["databasemigration"], "Assessment", "assessment")
	assertFormalSpecFor(t, services["databasemigration"], "Connection", "connection")
	assertFormalSpecFor(t, services["databasemigration"], "Migration", "migration")
	assertFormalSpecFor(t, services["databasetools"], "DatabaseToolsConnection", "databasetoolsconnection")
	assertFormalSpecFor(t, services["datalabelingservice"], "Dataset", "dataset")
	assertFormalSpecFor(t, services["disasterrecovery"], "DrPlan", "drplan")
	assertFormalSpecFor(t, services["distributeddatabase"], "DistributedDatabase", "distributeddatabase")
	assertFormalSpecFor(t, services["generativeaiagent"], "AgentEndpoint", "agentendpoint")
	assertFormalSpecFor(t, services["generativeaiagent"], "DataSource", "datasource")
	assertFormalSpecFor(t, services["healthchecks"], "HttpMonitor", "httpmonitor")
	assertFormalSpecFor(t, services["identity"], "Compartment", "compartment")
	assertFormalSpecFor(t, services["jms"], "JmsPlugin", "jmsplugin")
	assertFormalSpecFor(t, services["mediaservices"], "MediaWorkflowConfiguration", "mediaworkflowconfiguration")
	assertFormalSpecFor(t, services["tenantmanagercontrolplane"], "DomainGovernance", "domaingovernance")
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
	assertFormalSpecFor(t, services["oce"], "OceInstance", "oceinstance")
	assertFormalSpecFor(t, services["ocvp"], "Cluster", "cluster")
	assertFormalSpecFor(t, services["ocvp"], "Sddc", "sddc")
	assertFormalSpecFor(t, services["opa"], "OpaInstance", "opainstance")
	assertFormalSpecFor(t, services["opensearch"], "OpensearchCluster", "opensearchopensearchcluster")
	assertFormalSpecFor(t, services["psql"], "DbSystem", "dbsystem")
	assertFormalSpecFor(t, services["redis"], "RedisCluster", "rediscluster")
	assertFormalSpecFor(t, services["streaming"], "Stream", "stream")
}

func TestCheckedInConfigCoordinatesPrimaryPortPackagePaths(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	services := serviceConfigsByName(t, cfg, "analytics", "containerengine", "containerinstances", "core", "dataflow", "database", "identity", "keymanagement", "mysql", "objectstorage", "oce", "ocvp", "opa", "opensearch", "psql", "redis")

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
	assertPrimaryPortOverride(t, services["oce"], "OceInstance", "oceinstance", "oce/oceinstance")
	assertOCVPRuntimeRolloutMetadata(t, services["ocvp"])
	assertPrimaryPortOverride(t, services["opa"], "OpaInstance", "opainstance", "opa/opainstance")
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
		"accessgovernancecp":           {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"adm":                          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"aidataplatform":               {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"aidocument":                   {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"ailanguage":                   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"aispeech":                     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"aivision":                     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"analytics":                    {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"announcementsservice":         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"apiaccesscontrol":             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"apigateway":                   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"apiplatform":                  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"apmconfig":                    {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"apmcontrolplane":              {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"apmsynthetics":                {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"apmtraces":                    {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"appmgmtcontrol":               {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"artifacts":                    {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"autoscaling":                  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"bastion":                      {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"batch":                        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"bds":                          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"blockchain":                   {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"budget":                       {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"capacitymanagement":           {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"certificatesmanagement":       {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudbridge":                  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudguard":                   {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudmigrations":              {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"clusterplacementgroups":       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"computecloudatcustomer":       {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"computeinstanceagent":         {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"containerengine":              {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"containerinstances":           {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeHandwritten},
		"core":                         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"dashboardservice":             {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"database":                     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"databasemigration":            {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"databasetools":                {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"datacatalog":                  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"dataflow":                     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"dataintegration":              {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"datalabelingservice":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"datalabelingservicedataplane": {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe":                     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"datascience":                  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud":                 {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"delegateaccesscontrol":        {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"demandsignal":                 {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"desktops":                     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"devops":                       {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"dif":                          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"disasterrecovery":             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"distributeddatabase":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"dns":                          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"email":                        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"emwarehouse":                  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"events":                       {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"filestorage":                  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetsoftwareupdate":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"functions":                    {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeHandwritten},
		"fusionapps":                   {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"gdp":                          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"genericartifactscontent":      {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"generativeai":                 {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"generativeaiagent":            {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"generativeaiagentruntime":     {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"generativeaidata":             {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"goldengate":                   {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"governancerulescontrolplane":  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"healthchecks":                 {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"identity":                     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"integration":                  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"iot":                          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"jms":                          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"jmsjavadownloads":             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"jmsutils":                     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"keymanagement":                {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"licensemanager":               {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"limits":                       {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"limitsincrease":               {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"loadbalancer":                 {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"lockbox":                      {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"loganalytics":                 {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"logging":                      {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"lustrefilestorage":            {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"managedkafka":                 {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"managementagent":              {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"managementdashboard":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"marketplace":                  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"marketplaceprivateoffer":      {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"marketplacepublisher":         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"mediaservices":                {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"mngdmac":                      {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"monitoring":                   {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"multicloud":                   {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"mysql":                        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"networkfirewall":              {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"networkloadbalancer":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"nosql":                        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"objectstorage":                {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"ocicontrolcenter":             {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"oce":                          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"ocvp":                         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"oda":                          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"onesubscription":              {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"ons":                          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"opa":                          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opensearch":                   {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"operatoraccesscontrol":        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi":                         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"optimizer":                    {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"osmanagementhub":              {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"osubbillingschedule":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"osuborganizationsubscription": {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"osubsubscription":             {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"psa":                          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"psql":                         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeHandwritten},
		"queue":                        {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"recovery":                     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"redis":                        {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"resourceanalytics":            {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"resourcemanager":              {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"resourcescheduler":            {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"rover":                        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"sch":                          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"securityattribute":            {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"self":                         {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"servicecatalog":               {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"servicemanagerproxy":          {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"stackmonitoring":              {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"streaming":                    {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"tenantmanagercontrolplane":    {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"usageapi":                     {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"vbsinst":                      {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"visualbuilder":                {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"vnmonitoring":                 {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"vulnerabilityscanning":        {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"waa":                          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"waas":                         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"waf":                          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"wlms":                         {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"zpr":                          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
	}

	expectedByTarget := map[string]struct {
		strategy string
		runtime  string
	}{
		"autoscaling/AutoScalingConfiguration":        {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"aidataplatform/AiDataPlatform":               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudguard/DataSource":                       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/DataSafePrivateEndpoint":            {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/MaskingColumn":                      {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/SensitiveColumn":                    {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/SensitiveTypeGroup":                 {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/TargetAlertPolicyAssociation":       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"governancerulescontrolplane/GovernanceRule":  {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"managedkafka/KafkaCluster":                   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"managementagent/DataSource":                  {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeHandwritten},
		"managementagent/NamedCredential":             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"marketplacepublisher/Artifact":               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/AwrHub":                                 {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/AwrHubSource":                           {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/ChargebackPlanReport":                   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/EnterpriseManagerBridge":                {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/ExadataInsight":                         {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/HostInsight":                            {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/NewsReport":                             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/OperationsInsightsPrivateEndpoint":      {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/OperationsInsightsWarehouse":            {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/OperationsInsightsWarehouseUser":        {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"opsi/OpsiConfiguration":                      {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"recovery/ProtectedDatabase":                  {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"resourceanalytics/ResourceAnalyticsInstance": {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"bastion/Bastion":                             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"bastion/Session":                             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"blockchain/Osn":                              {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"blockchain/Peer":                             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudguard/WlpAgent":                         {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/AlertPolicy":                        {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/AttributeSet":                       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/SecurityPolicyConfig":               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/SensitiveTypesExport":               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/TargetDatabaseGroup":                {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"devops/BuildPipeline":                        {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"devops/DeployArtifact":                       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"devops/DeployPipeline":                       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"devops/Project":                              {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"devops/Repository":                           {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"devops/Trigger":                              {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/FleetCredential":         {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/FleetResource":           {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fusionapps/RefreshActivity":                  {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"healthchecks/HttpMonitor":                    {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"iot/IotDomain":                               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"jms/JmsPlugin":                               {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"loadbalancer/Listener":                       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"networkloadbalancer/Backend":                 {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"networkloadbalancer/BackendSet":              {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"networkloadbalancer/Listener":                {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"networkloadbalancer/NetworkLoadBalancer":     {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"recovery/ProtectionPolicy":                   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"recovery/RecoveryServiceSubnet":              {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"sch/ServiceConnector":                        {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"tenantmanagercontrolplane/DomainGovernance":  {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"waas/HttpRedirect":                           {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"waas/WaasPolicy":                             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"waf/WebAppFirewallPolicy":                    {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"wlms/ManagedInstance":                        {strategy: AsyncStrategyNone, runtime: AsyncRuntimeGeneratedRuntime},
		"batch/BatchContext":                          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"batch/BatchJobPool":                          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"blockchain/BlockchainPlatform":               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudbridge/AgentDependency":                 {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudbridge/AssetSource":                     {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudbridge/Inventory":                       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudmigrations/Migration":                   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudmigrations/MigrationAsset":              {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudmigrations/MigrationPlan":               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudmigrations/ReplicationSchedule":         {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"cloudmigrations/TargetAsset":                 {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datacatalog/CatalogPrivateEndpoint":          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datacatalog/Metastore":                       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dataintegration/Workspace":                   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/SecurityAssessment":                 {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/SecurityPolicyDeployment":           {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/TargetDatabase":                     {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"datasafe/UserAssessment":                     {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud/MultiCloudResourceDiscovery":    {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud/OracleDbAwsIdentityConnector":   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud/OracleDbAwsKey":                 {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud/OracleDbAzureBlobContainer":     {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud/OracleDbAzureBlobMount":         {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud/OracleDbAzureConnector":         {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud/OracleDbAzureVault":             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud/OracleDbAzureVaultAssociation":  {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud/OracleDbGcpIdentityConnector":   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dbmulticloud/OracleDbGcpKeyRing":             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"desktops/DesktopPool":                        {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"dif/Stack":                                   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"emwarehouse/EmWarehouse":                     {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/CatalogItem":             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/CompliancePolicyRule":    {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/Fleet":                   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/MaintenanceWindow":       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/Onboarding":              {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/Patch":                   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/PlatformConfiguration":   {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/Provision":               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/Runbook":                 {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/RunbookVersion":          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/SchedulerDefinition":     {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetappsmanagement/TaskRecord":              {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetsoftwareupdate/FsuAction":               {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetsoftwareupdate/FsuCollection":           {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetsoftwareupdate/FsuCycle":                {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetsoftwareupdate/FsuDiscovery":            {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fleetsoftwareupdate/FsuReadinessCheck":       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fusionapps/FusionEnvironment":                {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"fusionapps/FusionEnvironmentFamily":          {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"goldengate/Connection":                       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"goldengate/DatabaseRegistration":             {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"goldengate/Deployment":                       {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"goldengate/DeploymentBackup":                 {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"goldengate/Pipeline":                         {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"stackmonitoring/MaintenanceWindow":           {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
		"stackmonitoring/MonitoredResource":           {strategy: AsyncStrategyWorkRequest, runtime: AsyncRuntimeGeneratedRuntime},
	}

	targets := defaultActiveExplicitSelectedKindTargets(cfg)
	if len(targets) == 0 {
		t.Fatal("defaultActiveExplicitSelectedKindTargets() returned no targets")
	}

	for _, target := range targets {
		service := services[target.Service]
		expected, ok := expectedByTarget[target.Service+"/"+target.Kind]
		if !ok {
			expected, ok = expectedByService[target.Service]
		}
		if !ok {
			t.Fatalf("missing async expectation for default-active service %q", target.Service)
		}
		assertAsyncContract(t, service, target.Kind, expected.strategy, expected.runtime)
	}

	for _, testCase := range []struct {
		service string
		kind    string
		phases  []string
	}{
		{service: "bastion", kind: "Bastion", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "apigateway", kind: "Deployment", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "apigateway", kind: "Gateway", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "aidataplatform", kind: "AiDataPlatform", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "cloudguard", kind: "DataSource", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate}},
		{service: "datasafe", kind: "DataSafePrivateEndpoint", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datasafe", kind: "MaskingColumn", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate}},
		{service: "datasafe", kind: "SensitiveColumn", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate}},
		{service: "datasafe", kind: "SensitiveTypeGroup", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datasafe", kind: "TargetAlertPolicyAssociation", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "governancerulescontrolplane", kind: "GovernanceRule", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "managedkafka", kind: "KafkaCluster", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "managementagent", kind: "NamedCredential", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "marketplacepublisher", kind: "Artifact", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "AwrHub", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "AwrHubSource", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "ChargebackPlanReport", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "EnterpriseManagerBridge", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "ExadataInsight", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "HostInsight", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "NewsReport", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "OperationsInsightsPrivateEndpoint", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "OperationsInsightsWarehouse", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "OperationsInsightsWarehouseUser", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "opsi", kind: "OpsiConfiguration", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "recovery", kind: "ProtectedDatabase", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "resourceanalytics", kind: "ResourceAnalyticsInstance", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "bastion", kind: "Session", phases: []string{AsyncPhaseCreate, AsyncPhaseDelete}},
		{service: "blockchain", kind: "Osn", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "blockchain", kind: "Peer", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datasafe", kind: "AlertPolicy", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datasafe", kind: "AttributeSet", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datasafe", kind: "SecurityPolicyConfig", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datasafe", kind: "SensitiveTypesExport", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate}},
		{service: "datasafe", kind: "TargetDatabaseGroup", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "devops", kind: "BuildPipeline", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "devops", kind: "DeployArtifact", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "devops", kind: "DeployPipeline", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "devops", kind: "Project", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "devops", kind: "Repository", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "devops", kind: "Trigger", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "FleetCredential", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "FleetResource", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fusionapps", kind: "RefreshActivity", phases: []string{AsyncPhaseCreate, AsyncPhaseDelete}},
		{service: "iot", kind: "IotDomain", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "networkloadbalancer", kind: "Backend", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "networkloadbalancer", kind: "BackendSet", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "networkloadbalancer", kind: "Listener", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "networkloadbalancer", kind: "NetworkLoadBalancer", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "recovery", kind: "ProtectionPolicy", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "recovery", kind: "RecoveryServiceSubnet", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "sch", kind: "ServiceConnector", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "waas", kind: "HttpRedirect", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "waas", kind: "WaasPolicy", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "waf", kind: "WebAppFirewallPolicy", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "batch", kind: "BatchContext", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "batch", kind: "BatchJobPool", phases: []string{AsyncPhaseUpdate}},
		{service: "blockchain", kind: "BlockchainPlatform", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "cloudbridge", kind: "AgentDependency", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate}},
		{service: "cloudbridge", kind: "AssetSource", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "cloudbridge", kind: "Inventory", phases: []string{AsyncPhaseCreate, AsyncPhaseDelete}},
		{service: "cloudmigrations", kind: "Migration", phases: []string{AsyncPhaseDelete}},
		{service: "cloudmigrations", kind: "MigrationAsset", phases: []string{AsyncPhaseCreate, AsyncPhaseDelete}},
		{service: "cloudmigrations", kind: "MigrationPlan", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "cloudmigrations", kind: "ReplicationSchedule", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "cloudmigrations", kind: "TargetAsset", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datacatalog", kind: "CatalogPrivateEndpoint", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datacatalog", kind: "Metastore", phases: []string{AsyncPhaseCreate, AsyncPhaseDelete}},
		{service: "dataintegration", kind: "Workspace", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datasafe", kind: "SecurityAssessment", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datasafe", kind: "SecurityPolicyDeployment", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datasafe", kind: "TargetDatabase", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "datasafe", kind: "UserAssessment", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dbmulticloud", kind: "MultiCloudResourceDiscovery", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dbmulticloud", kind: "OracleDbAwsIdentityConnector", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dbmulticloud", kind: "OracleDbAwsKey", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dbmulticloud", kind: "OracleDbAzureBlobContainer", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dbmulticloud", kind: "OracleDbAzureBlobMount", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dbmulticloud", kind: "OracleDbAzureConnector", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dbmulticloud", kind: "OracleDbAzureVault", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dbmulticloud", kind: "OracleDbAzureVaultAssociation", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dbmulticloud", kind: "OracleDbGcpIdentityConnector", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dbmulticloud", kind: "OracleDbGcpKeyRing", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "desktops", kind: "DesktopPool", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "dif", kind: "Stack", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "emwarehouse", kind: "EmWarehouse", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "CatalogItem", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "CompliancePolicyRule", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "Fleet", phases: []string{AsyncPhaseCreate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "MaintenanceWindow", phases: []string{AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "Onboarding", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "Patch", phases: []string{AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "PlatformConfiguration", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "Provision", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "Runbook", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "RunbookVersion", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetappsmanagement", kind: "SchedulerDefinition", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate}},
		{service: "fleetappsmanagement", kind: "TaskRecord", phases: []string{AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetsoftwareupdate", kind: "FsuAction", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetsoftwareupdate", kind: "FsuCollection", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetsoftwareupdate", kind: "FsuCycle", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fleetsoftwareupdate", kind: "FsuDiscovery", phases: []string{AsyncPhaseCreate, AsyncPhaseDelete}},
		{service: "fleetsoftwareupdate", kind: "FsuReadinessCheck", phases: []string{AsyncPhaseCreate, AsyncPhaseDelete}},
		{service: "fusionapps", kind: "FusionEnvironment", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "fusionapps", kind: "FusionEnvironmentFamily", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "goldengate", kind: "Connection", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "goldengate", kind: "DatabaseRegistration", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "goldengate", kind: "Deployment", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "goldengate", kind: "DeploymentBackup", phases: []string{AsyncPhaseCreate, AsyncPhaseDelete}},
		{service: "goldengate", kind: "Pipeline", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "stackmonitoring", kind: "MaintenanceWindow", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
		{service: "stackmonitoring", kind: "MonitoredResource", phases: []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}},
	} {
		async := assertAsyncContract(t, services[testCase.service], testCase.kind, AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
		if async.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
			t.Fatalf("%s %s workRequest.source = %q, want %q", testCase.service, testCase.kind, async.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
		}
		if !slices.Equal(async.WorkRequest.Phases, testCase.phases) {
			t.Fatalf("%s %s workRequest.phases = %v, want %v", testCase.service, testCase.kind, async.WorkRequest.Phases, testCase.phases)
		}
	}

	managementAgentDataSource := assertAsyncContract(t, services["managementagent"], "DataSource", AsyncStrategyWorkRequest, AsyncRuntimeHandwritten)
	if managementAgentDataSource.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("managementagent DataSource workRequest.source = %q, want %q", managementAgentDataSource.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(managementAgentDataSource.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("managementagent DataSource workRequest.phases = %v", managementAgentDataSource.WorkRequest.Phases)
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

	mngdmac := assertAsyncContract(t, services["mngdmac"], "MacOrder", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if mngdmac.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("mngdmac MacOrder workRequest.source = %q, want %q", mngdmac.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(mngdmac.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("mngdmac MacOrder workRequest.phases = %v", mngdmac.WorkRequest.Phases)
	}
	if mngdmac.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("mngdmac MacOrder workRequest.legacyFieldBridge = %#v, want empty legacy bridge", mngdmac.WorkRequest.LegacyFieldBridge)
	}

	vbsinst := assertAsyncContract(t, services["vbsinst"], "VbsInstance", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if vbsinst.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("vbsinst VbsInstance workRequest.source = %q, want %q", vbsinst.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(vbsinst.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("vbsinst VbsInstance workRequest.phases = %v", vbsinst.WorkRequest.Phases)
	}

	zpr := assertAsyncContract(t, services["zpr"], "ZprPolicy", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if zpr.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("zpr ZprPolicy workRequest.source = %q, want %q", zpr.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(zpr.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("zpr ZprPolicy workRequest.phases = %v", zpr.WorkRequest.Phases)
	}
	if zpr.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("zpr ZprPolicy workRequest.legacyFieldBridge = %#v, want empty legacy bridge", zpr.WorkRequest.LegacyFieldBridge)
	}
	configuration := assertAsyncContract(t, services["zpr"], "Configuration", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if configuration.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("zpr Configuration workRequest.source = %q, want %q", configuration.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(configuration.WorkRequest.Phases, []string{AsyncPhaseCreate}) {
		t.Fatalf("zpr Configuration workRequest.phases = %v", configuration.WorkRequest.Phases)
	}
	if configuration.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("zpr Configuration workRequest.legacyFieldBridge = %#v, want empty legacy bridge", configuration.WorkRequest.LegacyFieldBridge)
	}
	if vbsinst.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("vbsinst VbsInstance workRequest.legacyFieldBridge = %#v, want empty legacy bridge", vbsinst.WorkRequest.LegacyFieldBridge)
	}

	psa := assertAsyncContract(t, services["psa"], "PrivateServiceAccess", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if psa.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("psa PrivateServiceAccess workRequest.source = %q, want %q", psa.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(psa.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("psa PrivateServiceAccess workRequest.phases = %v", psa.WorkRequest.Phases)
	}
	if psa.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("psa PrivateServiceAccess workRequest.legacyFieldBridge = %#v, want empty legacy bridge", psa.WorkRequest.LegacyFieldBridge)
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

	lustre := assertAsyncContract(t, services["lustrefilestorage"], "LustreFileSystem", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if lustre.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("lustrefilestorage LustreFileSystem workRequest.source = %q, want %q", lustre.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(lustre.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("lustrefilestorage LustreFileSystem workRequest.phases = %v", lustre.WorkRequest.Phases)
	}
	if lustre.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("lustrefilestorage LustreFileSystem workRequest.legacyFieldBridge = %#v, want empty legacy bridge", lustre.WorkRequest.LegacyFieldBridge)
	}

	objectStorageLink := assertAsyncContract(
		t,
		services["lustrefilestorage"],
		"ObjectStorageLink",
		AsyncStrategyWorkRequest,
		AsyncRuntimeGeneratedRuntime,
	)
	if objectStorageLink.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf(
			"lustrefilestorage ObjectStorageLink workRequest.source = %q, want %q",
			objectStorageLink.WorkRequest.Source,
			AsyncWorkRequestSourceServiceSDK,
		)
	}
	if !slices.Equal(objectStorageLink.WorkRequest.Phases, []string{AsyncPhaseDelete}) {
		t.Fatalf("lustrefilestorage ObjectStorageLink workRequest.phases = %v", objectStorageLink.WorkRequest.Phases)
	}
	if objectStorageLink.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf(
			"lustrefilestorage ObjectStorageLink workRequest.legacyFieldBridge = %#v, want empty legacy bridge",
			objectStorageLink.WorkRequest.LegacyFieldBridge,
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

	generativeAIAgent := assertAsyncContract(
		t,
		services["generativeaiagent"],
		"KnowledgeBase",
		AsyncStrategyWorkRequest,
		AsyncRuntimeGeneratedRuntime,
	)
	if generativeAIAgent.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf(
			"generativeaiagent KnowledgeBase workRequest.source = %q, want %q",
			generativeAIAgent.WorkRequest.Source,
			AsyncWorkRequestSourceServiceSDK,
		)
	}
	if !slices.Equal(generativeAIAgent.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf(
			"generativeaiagent KnowledgeBase workRequest.phases = %v",
			generativeAIAgent.WorkRequest.Phases,
		)
	}
	if generativeAIAgent.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf(
			"generativeaiagent KnowledgeBase workRequest.legacyFieldBridge = %#v, want empty legacy bridge",
			generativeAIAgent.WorkRequest.LegacyFieldBridge,
		)
	}

	visualBuilder := assertAsyncContract(
		t,
		services["visualbuilder"],
		"VbInstance",
		AsyncStrategyWorkRequest,
		AsyncRuntimeGeneratedRuntime,
	)
	if visualBuilder.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("visualbuilder VbInstance workRequest.source = %q, want %q", visualBuilder.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(visualBuilder.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("visualbuilder VbInstance workRequest.phases = %v", visualBuilder.WorkRequest.Phases)
	}
	if visualBuilder.WorkRequest.LegacyFieldBridge.hasOverride() {
		t.Fatalf("visualbuilder VbInstance workRequest.legacyFieldBridge = %#v, want empty legacy bridge", visualBuilder.WorkRequest.LegacyFieldBridge)
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

	services := serviceConfigsByName(t, cfg, "analytics", "capacitymanagement", "containerengine", "core", "dataflow", "nosql", "objectstorage")

	assertServiceSelection(t, services["analytics"], true, SelectionModeExplicit, []string{"AnalyticsInstance"})
	assertServiceSelection(t, services["capacitymanagement"], true, SelectionModeExplicit, []string{"OccCapacityRequest"})
	assertServiceSelection(t, services["containerengine"], true, SelectionModeExplicit, []string{"NodePool"})
	assertServiceSelection(t, services["core"], true, SelectionModeExplicit, []string{"Instance"})
	assertServiceSelection(t, services["dataflow"], true, SelectionModeExplicit, []string{"Application"})
	assertServiceSelection(t, services["nosql"], true, SelectionModeExplicit, []string{"Table"})
	assertServiceSelection(t, services["objectstorage"], true, SelectionModeExplicit, []string{"Bucket"})

	expectedByService := map[string]struct {
		strategy string
		runtime  string
	}{
		"analytics":          {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"capacitymanagement": {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"containerengine":    {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"core":               {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"dataflow":           {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"nosql":              {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
		"objectstorage":      {strategy: AsyncStrategyLifecycle, runtime: AsyncRuntimeGeneratedRuntime},
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
	if !service.Package.DedicatedServiceAccount {
		t.Fatal("psql dedicatedServiceAccount = false, want true")
	}
	if !slices.Equal(
		override.Controller.ExtraRBACMarkers,
		[]string{
			`groups="",resources=secrets,verbs=get`,
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
	assertResourceOverrideCount(t, service, 3)

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

	async = assertAsyncContract(t, service, "Assessment", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if async.FormalClassification != AsyncStrategyWorkRequest {
		t.Fatalf("databasemigration Assessment formalClassification = %q, want %q", async.FormalClassification, AsyncStrategyWorkRequest)
	}
	if async.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("databasemigration Assessment workRequest.source = %q, want %q", async.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(async.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("databasemigration Assessment workRequest.phases = %v, want %v", async.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete})
	}

	async = assertAsyncContract(t, service, "Migration", AsyncStrategyWorkRequest, AsyncRuntimeGeneratedRuntime)
	if async.FormalClassification != AsyncStrategyWorkRequest {
		t.Fatalf("databasemigration Migration formalClassification = %q, want %q", async.FormalClassification, AsyncStrategyWorkRequest)
	}
	if async.WorkRequest.Source != AsyncWorkRequestSourceServiceSDK {
		t.Fatalf("databasemigration Migration workRequest.source = %q, want %q", async.WorkRequest.Source, AsyncWorkRequestSourceServiceSDK)
	}
	if !slices.Equal(async.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete}) {
		t.Fatalf("databasemigration Migration workRequest.phases = %v, want %v", async.WorkRequest.Phases, []string{AsyncPhaseCreate, AsyncPhaseUpdate, AsyncPhaseDelete})
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

func TestObservedStateRequiredPointerFieldPaths(t *testing.T) {
	t.Parallel()

	service := ServiceConfig{
		ObservedState: ObservedStateConfig{
			RequiredPointerFieldPaths: map[string][]string{
				"DbSystem": {"StorageDetails.IsRegionallyDurable", " storageDetails.isRegionallyDurable "},
			},
		},
	}

	got := service.ObservedStateRequiredPointerFieldPaths("DbSystem")
	wantKey, err := normalizeObservedStateFieldPath("StorageDetails.IsRegionallyDurable")
	if err != nil {
		t.Fatalf("normalizeObservedStateFieldPath() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ObservedStateRequiredPointerFieldPaths() returned %d entries, want 1", len(got))
	}
	if _, ok := got[wantKey]; !ok {
		t.Fatalf("ObservedStateRequiredPointerFieldPaths() = %v, want %q", got, wantKey)
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

func TestCheckedInConfigPublishesNestedResourcePathIdentityFields(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	expected := map[string]map[string][]string{
		"cloudguard": {
			"DetectorRecipeDetectorRule": {"DetectorRecipeId", "CompartmentId"},
			"TargetDetectorRecipe":       {"TargetId", "CompartmentId"},
			"TargetResponderRecipe":      {"TargetId", "CompartmentId"},
		},
		"datacatalog": {
			"AttributeTag":     {"CatalogId", "DataAssetKey", "EntityKey", "AttributeKey"},
			"CustomProperty":   {"CatalogId", "NamespaceId"},
			"DataAssetTag":     {"CatalogId", "DataAssetKey"},
			"EntityTag":        {"CatalogId", "DataAssetKey", "EntityKey"},
			"FolderTag":        {"CatalogId", "DataAssetKey", "FolderKey"},
			"Glossary":         {"CatalogId"},
			"Namespace":        {"CatalogId"},
			"Pattern":          {"CatalogId"},
			"Term":             {"CatalogId", "GlossaryKey"},
			"TermRelationship": {"CatalogId", "GlossaryKey", "TermKey"},
		},
	}

	for serviceName, kinds := range expected {
		service := requireService(t, cfg, serviceName)
		overrides := overridesByKind(service)
		for kind, names := range kinds {
			override, ok := overrides[kind]
			if !ok {
				t.Fatalf("%s/%s override was not found", serviceName, kind)
			}
			fields := make(map[string]FieldOverride, len(override.SpecFields))
			for _, field := range override.SpecFields {
				fields[field.Name] = field
			}
			for _, name := range names {
				field, ok := fields[name]
				if !ok {
					t.Fatalf("%s/%s specFields = %#v, want %s", serviceName, kind, override.SpecFields, name)
				}
				if field.Type != "string" || !slices.Contains(field.Markers, "+kubebuilder:validation:Required") {
					t.Fatalf("%s/%s %s override = %#v, want required string", serviceName, kind, name, field)
				}
			}
		}
	}
}

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

func TestLoadConfigRejectsLegacySDKNameOverrideField(t *testing.T) {
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
    generation:
      resources:
        - kind: DbSystem
          sdkName: MySqlDbSystem
          controller:
            strategy: manual
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "sdkName") {
		t.Fatalf("LoadConfig() error = %v, want sdkName rejection", err)
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
			{Service: "database", SDKPackage: "example/database", Group: "database", PackageProfile: "controller-backed"},
			{Service: "mysql", SDKPackage: "example/mysql", Group: "mysql", PackageProfile: "controller-backed"},
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
    observedState:
      sdkAliases:
        WorkRequestLog:
          - WorkRequestLogEntry
  - service: psql
    sdkPackage: github.com/oracle/oci-go-sdk/v65/psql
    group: psql
    packageProfile: crd-only
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
    generation:
      controller:
        strategy: manual
      serviceManager:
        strategy: generated
      registration:
        strategy: generated
      webhooks:
        strategy: manual
      resources:
        - kind: DbSystem
          controller:
            maxConcurrentReconciles: 3
            extraRBACMarkers:
              - groups="",resources=secrets,verbs=get;list;watch
              - groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
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
  - service: core
    sdkPackage: github.com/oracle/oci-go-sdk/v65/core
    group: core
    packageProfile: crd-only
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

	mysqlService := cfg.Services[0]
	assertServiceGenerationStrategies(t, &mysqlService, generationStrategyExpectation{
		controller:     GenerationStrategyManual,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyManual,
	})
	if len(mysqlService.Generation.Resources) != 1 {
		t.Fatalf("len(mysql generation resources) = %d, want 1", len(mysqlService.Generation.Resources))
	}

	override := mysqlService.Generation.Resources[0]
	if override.Kind != "DbSystem" {
		t.Fatalf("mysql override kind = %q, want %q", override.Kind, "DbSystem")
	}
	if override.Controller.MaxConcurrentReconciles != 3 {
		t.Fatalf("mysql maxConcurrentReconciles = %d, want 3", override.Controller.MaxConcurrentReconciles)
	}
	if !slices.Equal(override.Controller.ExtraRBACMarkers, []string{
		`groups="",resources=secrets,verbs=get;list;watch`,
		`groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
	}) {
		t.Fatalf("mysql extra RBAC markers = %v, want secret and event markers", override.Controller.ExtraRBACMarkers)
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

	coreService := cfg.Services[1]
	assertServiceGenerationStrategies(t, &coreService, generationStrategyExpectation{
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

func TestCheckedInConfigIncludesRuntimeRolloutMetadata(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	services := serviceConfigsByName(t, cfg, "database", "mysql", "streaming", "core")

	for _, name := range []string{"database", "mysql", "streaming"} {
		assertServiceGenerationStrategies(t, services[name], generationStrategyExpectation{
			controller:     GenerationStrategyGenerated,
			serviceManager: GenerationStrategyGenerated,
			registration:   GenerationStrategyGenerated,
			webhook:        GenerationStrategyManual,
		})
	}

	assertCheckedInDatabaseRuntimeRolloutMetadata(t, services["database"])
	assertCheckedInMySQLRuntimeRolloutMetadata(t, services["mysql"])
	assertCheckedInStreamingRuntimeRolloutMetadata(t, services["streaming"])

	if services["core"].PackageProfile != PackageProfileControllerBacked {
		t.Fatalf("core packageProfile = %q, want %q", services["core"].PackageProfile, PackageProfileControllerBacked)
	}
	assertServiceGenerationStrategies(t, services["core"], generationStrategyExpectation{
		controller:     GenerationStrategyGenerated,
		serviceManager: GenerationStrategyGenerated,
		registration:   GenerationStrategyGenerated,
		webhook:        GenerationStrategyNone,
	})
}

func TestCheckedInConfigSetsFormalSpecsForPromotedKinds(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	services := serviceConfigsByName(t, cfg, "identity", "database", "mysql", "streaming")

	assertServiceFormalSpec(t, services["identity"], "User", "user")
	if got := services["identity"].FormalSpecFor("Compartment"); got != "" {
		t.Fatalf("identity Compartment formalSpec = %q, want empty", got)
	}

	for _, test := range []struct {
		service *ServiceConfig
		kind    string
		want    string
	}{
		{service: services["database"], kind: "AutonomousDatabase", want: "autonomousdatabase"},
		{service: services["mysql"], kind: "DbSystem", want: "dbsystem"},
		{service: services["streaming"], kind: "Stream", want: "stream"},
	} {
		assertServiceFormalSpec(t, test.service, test.kind, test.want)
	}
}

func TestCheckedInConfigOptsOutEndpointBasedGeneratedRuntimeResources(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)

	wantKinds := map[string][]string{
		"keymanagement": {"Key", "KeyVersion", "ReplicationStatus", "WrappingKey"},
		"streaming":     {"Cursor", "Group", "GroupCursor", "Message"},
	}

	for serviceName, kinds := range wantKinds {
		service := serviceConfigsByName(t, cfg, serviceName)[serviceName]
		assertResourceGenerationDisabled(t, service, kinds...)
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
	promotedNames := make([]string, 0)
	for _, service := range servicesCfg.Services {
		if _, ok := manualWebhookServices[service.Service]; ok {
			continue
		}

		promotedNames = append(promotedNames, service.Service)
		if service.PackageProfile != PackageProfileControllerBacked {
			t.Fatalf("%s packageProfile = %q, want %q", service.Service, service.PackageProfile, PackageProfileControllerBacked)
		}
		if got := service.ControllerGenerationStrategy(); got != GenerationStrategyGenerated {
			t.Fatalf("%s controller strategy = %q, want %q", service.Service, got, GenerationStrategyGenerated)
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

func assertCheckedInDatabaseRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if len(service.Generation.Resources) != 1 {
		t.Fatalf("database generation overrides = %d, want 1", len(service.Generation.Resources))
	}

	override := service.Generation.Resources[0]
	assertServiceFormalSpec(t, service, "AutonomousDatabase", "autonomousdatabase")
	if !slices.Equal(
		override.Controller.ExtraRBACMarkers,
		[]string{
			`groups="",resources=secrets,verbs=get;list;watch`,
			`groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
		},
	) {
		t.Fatalf("database extra RBAC markers = %v", override.Controller.ExtraRBACMarkers)
	}
	if len(override.SpecFields) != 1 || override.SpecFields[0].Name != "AdminPassword" || override.SpecFields[0].Type != "shared.PasswordSource" {
		t.Fatalf("database specFields = %#v, want secret-backed adminPassword override", override.SpecFields)
	}
	if got := service.ServiceManagerPackagePathFor("AutonomousDatabase", "autonomousdatabase"); got != "database/autonomousdatabase" {
		t.Fatalf("database packagePath = %q, want %q", got, "database/autonomousdatabase")
	}
	if got := generationStrategyOrDefault(override.Webhooks.Strategy, GenerationStrategyManual); got != GenerationStrategyManual {
		t.Fatalf("database resource webhook strategy = %q, want %q", got, GenerationStrategyManual)
	}
}

func assertCheckedInMySQLRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if len(service.Generation.Resources) != 1 {
		t.Fatalf("mysql generation overrides = %d, want 1", len(service.Generation.Resources))
	}

	override := service.Generation.Resources[0]
	assertServiceFormalSpec(t, service, "DbSystem", "dbsystem")
	if override.Controller.MaxConcurrentReconciles != 3 {
		t.Fatalf("mysql maxConcurrentReconciles = %d, want 3", override.Controller.MaxConcurrentReconciles)
	}
	if !slices.Equal(
		override.Controller.ExtraRBACMarkers,
		[]string{
			`groups="",resources=secrets,verbs=get;list;watch`,
			`groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
		},
	) {
		t.Fatalf("mysql extra RBAC markers = %v", override.Controller.ExtraRBACMarkers)
	}
	if len(override.SpecFields) != 2 {
		t.Fatalf("mysql specFields = %#v, want 2 secret-backed overrides", override.SpecFields)
	}
	if len(override.StatusFields) != 2 {
		t.Fatalf("mysql statusFields = %#v, want 2 secret-backed overrides", override.StatusFields)
	}
	if !strings.Contains(override.Sample.Body, "adminPassword:") || !strings.Contains(override.Sample.Body, "secretName: admin-secret") {
		t.Fatalf("mysql sample override = %q, want secret-backed sample body", override.Sample.Body)
	}
	if got := service.ServiceManagerPackagePathFor("DbSystem", "dbsystem"); got != "mysql/dbsystem" {
		t.Fatalf("mysql packagePath = %q, want %q", got, "mysql/dbsystem")
	}
}

func assertCheckedInStreamingRuntimeRolloutMetadata(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if len(service.Generation.Resources) != 5 {
		t.Fatalf("streaming generation overrides = %d, want 5", len(service.Generation.Resources))
	}

	overrides := resourceGenerationOverridesByKind(service.Generation.Resources)
	streamOverride, ok := overrides["Stream"]
	if !ok {
		t.Fatal("streaming override for Stream was not found")
	}
	assertServiceFormalSpec(t, service, "Stream", "stream")
	if !slices.Equal(
		streamOverride.Controller.ExtraRBACMarkers,
		[]string{`groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`},
	) {
		t.Fatalf("streaming Stream extra RBAC markers = %v", streamOverride.Controller.ExtraRBACMarkers)
	}
	if got := service.ServiceManagerPackagePathFor("Stream", "stream"); got != "streaming/stream" {
		t.Fatalf("streaming packagePath = %q, want %q", got, "streaming/stream")
	}
	assertResourceGenerationDisabled(t, service, "Cursor", "Group", "GroupCursor", "Message")
}

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
        - kind: MySqlDbSystem
          sdkName: DbSystem
          controller:
            maxConcurrentReconciles: 3
            extraRBACMarkers:
              - groups="",resources=secrets,verbs=get;list;watch
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

	cfg := mustLoadGeneratorConfig(t, configPath)
	if len(cfg.Services) != 2 {
		t.Fatalf("len(cfg.Services) = %d, want 2", len(cfg.Services))
	}

	mysqlService := mustFindGeneratorService(t, cfg, "mysql")
	assertGeneratorStrategies(t, mysqlService, GenerationStrategyManual, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyManual)
	if len(mysqlService.Generation.Resources) != 1 {
		t.Fatalf("len(mysql generation resources) = %d, want 1", len(mysqlService.Generation.Resources))
	}

	override := mysqlService.Generation.Resources[0]
	if override.Kind != "MySqlDbSystem" {
		t.Fatalf("mysql override kind = %q, want %q", override.Kind, "MySqlDbSystem")
	}
	if override.SDKName != "DbSystem" {
		t.Fatalf("mysql override sdkName = %q, want %q", override.SDKName, "DbSystem")
	}
	if override.Controller.MaxConcurrentReconciles != 3 {
		t.Fatalf("mysql maxConcurrentReconciles = %d, want 3", override.Controller.MaxConcurrentReconciles)
	}
	if !slices.Equal(override.Controller.ExtraRBACMarkers, []string{`groups="",resources=secrets,verbs=get;list;watch`}) {
		t.Fatalf("mysql extra RBAC markers = %v, want secrets marker", override.Controller.ExtraRBACMarkers)
	}
	if override.ServiceManager.PackagePath != "mysql/dbsystem" {
		t.Fatalf("mysql packagePath = %q, want %q", override.ServiceManager.PackagePath, "mysql/dbsystem")
	}

	coreService := mustFindGeneratorService(t, cfg, "core")
	assertGeneratorStrategies(t, coreService, GenerationStrategyNone, GenerationStrategyNone, GenerationStrategyNone, GenerationStrategyManual)
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
	if got := service.FormalSpecFor("MySqlDbSystem"); got != "dbsystem" {
		t.Fatalf("FormalSpecFor(MySqlDbSystem) = %q, want %q", got, "dbsystem")
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
					Kind: "MySqlDbSystem",
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

	if got := service.ControllerGenerationStrategyFor("MySqlDbSystem"); got != GenerationStrategyGenerated {
		t.Fatalf("ControllerGenerationStrategyFor(MySqlDbSystem) = %q, want %q", got, GenerationStrategyGenerated)
	}

	config := service.ControllerGenerationConfigFor("MySqlDbSystem")
	if config.Strategy != GenerationStrategyGenerated {
		t.Fatalf("ControllerGenerationConfigFor(MySqlDbSystem).Strategy = %q, want %q", config.Strategy, GenerationStrategyGenerated)
	}
	if config.MaxConcurrentReconciles != 3 {
		t.Fatalf("ControllerGenerationConfigFor(MySqlDbSystem).MaxConcurrentReconciles = %d, want 3", config.MaxConcurrentReconciles)
	}
	if !slices.Equal(config.ExtraRBACMarkers, []string{`groups="",resources=secrets,verbs=get;list;watch`}) {
		t.Fatalf("ControllerGenerationConfigFor(MySqlDbSystem).ExtraRBACMarkers = %v", config.ExtraRBACMarkers)
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
					{Kind: "MySqlDbSystem", Controller: ControllerGenerationOverride{Strategy: GenerationStrategyManual}},
					{Kind: "MySqlDbSystem", ServiceManager: ServiceManagerGenerationOverride{PackagePath: "mysql/dbsystem"}},
				}
			},
			wantErr: `duplicate kind "MySqlDbSystem"`,
		},
		{
			name: "duplicate sdk name override",
			mutate: func(cfg *Config) {
				cfg.Services[0].Generation.Resources = []ResourceGenerationOverride{
					{Kind: "MySqlDbSystem", SDKName: "DbSystem", Controller: ControllerGenerationOverride{Strategy: GenerationStrategyManual}},
					{Kind: "Widget", SDKName: "DbSystem", ServiceManager: ServiceManagerGenerationOverride{PackagePath: "mysql/widget"}},
				}
			},
			wantErr: `duplicate sdkName "DbSystem"`,
		},
		{
			name: "blank extra rbac marker",
			mutate: func(cfg *Config) {
				cfg.Services[0].Generation.Resources = []ResourceGenerationOverride{
					{
						Kind: "MySqlDbSystem",
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
						Kind: "MySqlDbSystem",
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
					{Kind: "MySqlDbSystem"},
				}
			},
			wantErr: `generation.resources["MySqlDbSystem"] does not override any runtime output`,
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
						Kind:       "MySqlDbSystem",
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

	cfg := loadCheckedInGeneratorConfig(t)
	databaseService := mustFindGeneratorService(t, cfg, "database")
	mysqlService := mustFindGeneratorService(t, cfg, "mysql")
	streamingService := mustFindGeneratorService(t, cfg, "streaming")
	coreService := mustFindGeneratorService(t, cfg, "core")

	for _, service := range []*ServiceConfig{databaseService, mysqlService, streamingService} {
		assertGeneratorStrategies(t, service, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyGenerated, service.WebhookGenerationStrategy())
	}
	assertGeneratorStrategies(t, databaseService, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyNone)
	for _, service := range []*ServiceConfig{mysqlService, streamingService} {
		assertGeneratorStrategies(t, service, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyManual)
	}

	assertRuntimeRolloutMetadataForDatabase(t, databaseService)
	assertRuntimeRolloutMetadataForMySQL(t, mysqlService)
	assertRuntimeRolloutMetadataForStreaming(t, streamingService)

	coreService := services["core"]
	if coreService.PackageProfile != PackageProfileControllerBacked {
		t.Fatalf("core packageProfile = %q, want %q", coreService.PackageProfile, PackageProfileControllerBacked)
	}
	assertGeneratorStrategies(t, coreService, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyNone)
}

func TestCheckedInConfigPromotesIdentityUserAndDatabaseAutonomousDatabaseFormalSpec(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInGeneratorConfig(t)
	identityService := mustFindGeneratorService(t, cfg, "identity")
	databaseService := mustFindGeneratorService(t, cfg, "database")
	mysqlService := mustFindGeneratorService(t, cfg, "mysql")
	streamingService := mustFindGeneratorService(t, cfg, "streaming")

	assertFormalSpecFor(t, identityService, "User", "user")
	assertFormalSpecFor(t, identityService, "Compartment", "")
	assertFormalSpecFor(t, databaseService, "AutonomousDatabase", "databaseautonomousdatabase")

	for _, test := range []struct {
		service *ServiceConfig
		kind    string
	}{
		{service: mysqlService, kind: "MySqlDbSystem"},
		{service: streamingService, kind: "Stream"},
	} {
		assertFormalSpecFor(t, test.service, test.kind, "")
	}
}

func TestCheckedInConfigOptsOutEndpointBasedGeneratedRuntimeResources(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInGeneratorConfig(t)

	wantKinds := map[string][]string{
		"keymanagement": {"Key", "KeyVersion", "ReplicationStatus", "WrappingKey"},
		"streaming":     {"Cursor", "Group", "GroupCursor", "Message"},
	}

	for serviceName, kinds := range wantKinds {
		service := mustFindGeneratorService(t, cfg, serviceName)
		for _, kind := range kinds {
			assertResourceOverrideDisabled(t, service, kind)
		}
	}
}

func TestCheckedInGeneratedServicesUseSharedManagerRollout(t *testing.T) {
	t.Parallel()

	servicesPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	servicesCfg, err := LoadConfig(servicesPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", servicesPath, err)
	}

	serviceNames := make([]string, 0, len(servicesCfg.Services))
	for _, service := range servicesCfg.Services {
		serviceNames = append(serviceNames, service.Service)
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
	slices.Sort(serviceNames)

	if len(serviceNames) == 0 {
		t.Fatal("expected at least one configured generated service in services.yaml")
	}
}

func TestCheckedInServicesConfigDoesNotUseParityOrCompatibilityInputs(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile(filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(services.yaml) error = %v", err)
	}

	rendered := string(content)
	if strings.Contains(rendered, "phase: parity") {
		t.Fatalf("services.yaml still contains a parity phase:\n%s", rendered)
	}
	if strings.Contains(rendered, "parityFile:") {
		t.Fatalf("services.yaml still contains parityFile inputs:\n%s", rendered)
	}
	if strings.Contains(rendered, "compatibility:") {
		t.Fatalf("services.yaml still contains compatibility inputs:\n%s", rendered)
	}
	if strings.Contains(rendered, "existingKinds:") {
		t.Fatalf("services.yaml still contains existingKinds inputs:\n%s", rendered)
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

func assertSelectServicesResult(t *testing.T, cfg *Config, serviceName string, all bool, wantCount int, wantErr string) {
	t.Helper()

	services, err := cfg.SelectServices(serviceName, all)
	if wantErr != "" {
		if err == nil {
			t.Fatalf("SelectServices() error = nil, want %q", wantErr)
		}
		if !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("SelectServices() error = %v, want substring %q", err, wantErr)
		}
		return
	}
	if err != nil {
		t.Fatalf("SelectServices() error = %v", err)
	}
	if len(services) != wantCount {
		t.Fatalf("SelectServices() returned %d services, want %d", len(services), wantCount)
	}
}

func mustLoadGeneratorConfig(t *testing.T, path string) *Config {
	t.Helper()

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", path, err)
	}

	return cfg
}

func loadCheckedInGeneratorConfig(t *testing.T) *Config {
	t.Helper()

	return mustLoadGeneratorConfig(t, filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml"))
}

func mustFindGeneratorService(t *testing.T, cfg *Config, name string) *ServiceConfig {
	t.Helper()

	for i := range cfg.Services {
		if cfg.Services[i].Service == name {
			return &cfg.Services[i]
		}
	}

	t.Fatalf("service %q was not found in services.yaml", name)
	return nil
}

func assertGeneratorStrategies(t *testing.T, service *ServiceConfig, controller string, serviceManager string, registration string, webhook string) {
	t.Helper()

	if got := service.ControllerGenerationStrategy(); got != controller {
		t.Fatalf("%s controller strategy = %q, want %q", service.Service, got, controller)
	}
	if got := service.ServiceManagerGenerationStrategy(); got != serviceManager {
		t.Fatalf("%s service-manager strategy = %q, want %q", service.Service, got, serviceManager)
	}
	if got := service.RegistrationGenerationStrategy(); got != registration {
		t.Fatalf("%s registration strategy = %q, want %q", service.Service, got, registration)
	}
	if got := service.WebhookGenerationStrategy(); got != webhook {
		t.Fatalf("%s webhook strategy = %q, want %q", service.Service, got, webhook)
	}
}

func assertRuntimeRolloutMetadataForDatabase(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if len(service.Generation.Resources) != 1 {
		t.Fatalf("database generation overrides = %d, want 1", len(service.Generation.Resources))
	}
	override := service.Generation.Resources[0]
	if !slices.Equal(
		override.Controller.ExtraRBACMarkers,
		[]string{
			`groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`,
			`groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
		},
	) {
		t.Fatalf("database extra RBAC markers = %v", override.Controller.ExtraRBACMarkers)
	}
	if override.FormalSpec != "databaseautonomousdatabase" {
		t.Fatalf("database formalSpec = %q, want %q", override.FormalSpec, "databaseautonomousdatabase")
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
	if override.ServiceManager.PackagePath != "" {
		t.Fatalf("database packagePath = %q, want empty to use default package layout", override.ServiceManager.PackagePath)
	}
	if override.Webhooks.Strategy != "" {
		t.Fatalf("database resource webhook strategy = %q, want empty", override.Webhooks.Strategy)
	}
}

func assertRuntimeRolloutMetadataForMySQL(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if len(service.Generation.Resources) != 1 {
		t.Fatalf("mysql generation overrides = %d, want 1", len(service.Generation.Resources))
	}
	override := service.Generation.Resources[0]
	if override.Controller.MaxConcurrentReconciles != 3 {
		t.Fatalf("mysql maxConcurrentReconciles = %d, want 3", override.Controller.MaxConcurrentReconciles)
	}
	if !slices.Equal(
		override.Controller.ExtraRBACMarkers,
		[]string{
			`groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`,
			`groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
		},
	) {
		t.Fatalf("mysql extra RBAC markers = %v", override.Controller.ExtraRBACMarkers)
	}
	if override.ServiceManager.PackagePath != "mysql/dbsystem" {
		t.Fatalf("mysql packagePath = %q, want %q", override.ServiceManager.PackagePath, "mysql/dbsystem")
	}
}

func assertRuntimeRolloutMetadataForStreaming(t *testing.T, service *ServiceConfig) {
	t.Helper()

	if len(service.Generation.Resources) != 5 {
		t.Fatalf("streaming generation overrides = %d, want 5", len(service.Generation.Resources))
	}

	overrides := generatorOverridesByKind(service)
	if overrides["Stream"].ServiceManager.PackagePath != "streaming/stream" {
		t.Fatalf("streaming packagePath = %q, want %q", overrides["Stream"].ServiceManager.PackagePath, "streaming/stream")
	}
	for _, kind := range []string{"Cursor", "Group", "GroupCursor", "Message"} {
		assertResourceOverrideDisabled(t, service, kind)
	}
}

func generatorOverridesByKind(service *ServiceConfig) map[string]ResourceGenerationOverride {
	overrides := make(map[string]ResourceGenerationOverride, len(service.Generation.Resources))
	for _, override := range service.Generation.Resources {
		overrides[override.Kind] = override
	}

	return overrides
}

func assertResourceOverrideDisabled(t *testing.T, service *ServiceConfig, kind string) {
	t.Helper()

	override, ok := generatorOverridesByKind(service)[kind]
	if !ok {
		t.Fatalf("%s override for %s was not found", service.Service, kind)
	}
	if override.Controller.Strategy != GenerationStrategyNone {
		t.Fatalf("%s %s controller strategy = %q, want %q", service.Service, kind, override.Controller.Strategy, GenerationStrategyNone)
	}
	if override.ServiceManager.Strategy != GenerationStrategyNone {
		t.Fatalf("%s %s service-manager strategy = %q, want %q", service.Service, kind, override.ServiceManager.Strategy, GenerationStrategyNone)
	}
}

func assertFormalSpecFor(t *testing.T, service *ServiceConfig, kind string, want string) {
	t.Helper()

	if got := service.FormalSpecFor(kind); got != want {
		if want == "" {
			t.Fatalf("%s %s formalSpec = %q, want empty while legacy adapter remains active", service.Service, kind, got)
		}
		t.Fatalf("%s %s formalSpec = %q, want %q", service.Service, kind, got, want)
	}
}

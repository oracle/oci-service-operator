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

type selectServicesCase struct {
	name        string
	serviceName string
	all         bool
	wantCount   int
	wantErr     string
}

type resourceOverrideExpectation struct {
	kind                    string
	controllerStrategy      string
	serviceManagerStrategy  string
	webhookStrategy         string
	maxConcurrentReconciles int
	extraRBACMarkers        []string
	packagePath             string
}

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
    description: runtime-integrated groups
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

func TestSelectServices(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{Service: "database", SDKPackage: "example/database", Group: "database", PackageProfile: "controller-backed"},
			{Service: "mysql", SDKPackage: "example/mysql", Group: "mysql", PackageProfile: "controller-backed"},
		},
	}

	tests := []selectServicesCase{
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
			assertSelectServicesResult(t, cfg, test)
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
        strategy: generated
      serviceManager:
        strategy: generated
      registration:
        strategy: generated
      webhooks:
        strategy: manual
      resources:
        - kind: MySqlDbSystem
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

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.Services) != 2 {
		t.Fatalf("len(cfg.Services) = %d, want 2", len(cfg.Services))
	}

	mysqlService := cfg.Services[0]
	assertGenerationStrategies(t, "mysql", &mysqlService, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyManual)
	assertResourceCount(t, "mysql", &mysqlService, 1)
	assertResourceOverride(t, "mysql", mysqlService.Generation.Resources[0], resourceOverrideExpectation{
		kind:                    "MySqlDbSystem",
		maxConcurrentReconciles: 3,
		extraRBACMarkers:        []string{`groups="",resources=secrets,verbs=get;list;watch`},
		packagePath:             "mysql/dbsystem",
	})

	coreService := cfg.Services[1]
	assertGenerationStrategies(t, "core", &coreService, GenerationStrategyNone, GenerationStrategyNone, GenerationStrategyNone, GenerationStrategyManual)
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

	cfg := mustLoadCheckedInConfig(t)
	services := requireServices(t, cfg, "database", "mysql", "streaming", "core")

	assertGenerationStrategies(t, "database", services["database"], GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyManual)
	assertGenerationStrategies(t, "mysql", services["mysql"], GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyManual)
	assertGenerationStrategies(t, "streaming", services["streaming"], GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyManual)
	assertGenerationStrategies(t, "core", services["core"], GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyGenerated, GenerationStrategyNone)
	assertResourceCount(t, "database", services["database"], 1)
	assertResourceCount(t, "mysql", services["mysql"], 1)
	assertResourceCount(t, "streaming", services["streaming"], 5)
	assertPackageProfile(t, services["core"], PackageProfileControllerBacked)

	assertResourceOverride(t, "database", services["database"].Generation.Resources[0], resourceOverrideExpectation{
		extraRBACMarkers: []string{
			`groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`,
			`groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
		},
		packagePath:     "autonomousdatabases/adb",
		webhookStrategy: GenerationStrategyManual,
	})
	assertResourceOverride(t, "mysql", services["mysql"].Generation.Resources[0], resourceOverrideExpectation{
		maxConcurrentReconciles: 3,
		extraRBACMarkers: []string{
			`groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`,
			`groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
		},
		packagePath: "mysql/dbsystem",
	})

	streamingOverrides := resourceOverridesByKind(services["streaming"])
	assertResourceOverride(t, "streaming Stream", streamingOverrides["Stream"], resourceOverrideExpectation{
		packagePath: "streaming/stream",
	})
	assertRuntimeDisabledOverride(t, "streaming", streamingOverrides, "Cursor", "Group", "GroupCursor", "Message")
}

func TestCheckedInConfigPromotesOnlyIdentityUserFormalSpec(t *testing.T) {
	t.Parallel()

	cfg := mustLoadCheckedInConfig(t)
	services := requireServices(t, cfg, "identity", "database", "mysql", "streaming")

	assertFormalSpec(t, services["identity"], "User", "user")
	assertFormalSpec(t, services["identity"], "Compartment", "")
	assertFormalSpec(t, services["database"], "AutonomousDatabases", "")
	assertFormalSpec(t, services["mysql"], "MySqlDbSystem", "")
	assertFormalSpec(t, services["streaming"], "Stream", "")
}

func TestCheckedInConfigOptsOutEndpointBasedGeneratedRuntimeResources(t *testing.T) {
	t.Parallel()

	cfg := mustLoadCheckedInConfig(t)

	wantKinds := map[string][]string{
		"keymanagement": {"Key", "KeyVersion", "ReplicationStatus", "WrappingKey"},
		"streaming":     {"Cursor", "Group", "GroupCursor", "Message"},
	}

	for serviceName, kinds := range wantKinds {
		service := requireServices(t, cfg, serviceName)[serviceName]
		assertRuntimeDisabledOverride(t, serviceName, resourceOverridesByKind(service), kinds...)
	}
}

func TestCheckedInGeneratedNonParityServicesUseSharedManagerRollout(t *testing.T) {
	t.Parallel()

	servicesPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	servicesCfg, err := LoadConfig(servicesPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", servicesPath, err)
	}

	parityServices := map[string]struct{}{
		"database":  {},
		"mysql":     {},
		"streaming": {},
	}
	promotedNames := make([]string, 0)
	for _, service := range servicesCfg.Services {
		if _, ok := parityServices[service.Service]; ok {
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
		if got := service.WebhookGenerationStrategy(); got != GenerationStrategyNone {
			t.Fatalf("%s webhook strategy = %q, want %q", service.Service, got, GenerationStrategyNone)
		}
	}
	slices.Sort(promotedNames)

	if len(promotedNames) == 0 {
		t.Fatal("expected at least one promoted non-parity service in services.yaml")
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

func assertSelectServicesResult(t *testing.T, cfg *Config, test selectServicesCase) {
	t.Helper()

	services, err := cfg.SelectServices(test.serviceName, test.all)
	if test.wantErr != "" {
		if err == nil {
			t.Fatalf("SelectServices() error = nil, want %q", test.wantErr)
		}
		if !strings.Contains(err.Error(), test.wantErr) {
			t.Fatalf("SelectServices() error = %v, want substring %q", err, test.wantErr)
		}
		return
	}
	if err != nil {
		t.Fatalf("SelectServices() error = %v", err)
	}
	if len(services) != test.wantCount {
		t.Fatalf("SelectServices() returned %d services, want %d", len(services), test.wantCount)
	}
}

func mustLoadCheckedInConfig(t *testing.T) *Config {
	t.Helper()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}
	return cfg
}

func requireServices(t *testing.T, cfg *Config, names ...string) map[string]*ServiceConfig {
	t.Helper()

	services := make(map[string]*ServiceConfig, len(names))
	for i := range cfg.Services {
		service := &cfg.Services[i]
		services[service.Service] = service
	}

	selected := make(map[string]*ServiceConfig, len(names))
	for _, name := range names {
		service, ok := services[name]
		if !ok {
			t.Fatalf("service %q was not found in services.yaml", name)
		}
		selected[name] = service
	}
	return selected
}

func assertPackageProfile(t *testing.T, service *ServiceConfig, want string) {
	t.Helper()
	if service.PackageProfile != want {
		t.Fatalf("%s packageProfile = %q, want %q", service.Service, service.PackageProfile, want)
	}
}

func assertGenerationStrategies(t *testing.T, label string, service *ServiceConfig, controller, serviceManager, registration, webhook string) {
	t.Helper()

	if got := service.ControllerGenerationStrategy(); got != controller {
		t.Fatalf("%s controller strategy = %q, want %q", label, got, controller)
	}
	if got := service.ServiceManagerGenerationStrategy(); got != serviceManager {
		t.Fatalf("%s service-manager strategy = %q, want %q", label, got, serviceManager)
	}
	if got := service.RegistrationGenerationStrategy(); got != registration {
		t.Fatalf("%s registration strategy = %q, want %q", label, got, registration)
	}
	if got := service.WebhookGenerationStrategy(); got != webhook {
		t.Fatalf("%s webhook strategy = %q, want %q", label, got, webhook)
	}
}

func assertResourceCount(t *testing.T, label string, service *ServiceConfig, want int) {
	t.Helper()
	if got := len(service.Generation.Resources); got != want {
		t.Fatalf("%s generation overrides = %d, want %d", label, got, want)
	}
}

func assertResourceOverride(t *testing.T, label string, override ResourceGenerationOverride, want resourceOverrideExpectation) {
	t.Helper()

	assertOverrideString(t, label, "kind", override.Kind, want.kind)
	assertOverrideString(t, label, "controller strategy", override.Controller.Strategy, want.controllerStrategy)
	assertOverrideString(t, label, "service-manager strategy", override.ServiceManager.Strategy, want.serviceManagerStrategy)
	assertOverrideWebhookStrategy(t, label, override.Webhooks.Strategy, want.webhookStrategy)
	assertOverrideInt(t, label, "maxConcurrentReconciles", override.Controller.MaxConcurrentReconciles, want.maxConcurrentReconciles)
	assertOverrideStrings(t, label, "extra RBAC markers", override.Controller.ExtraRBACMarkers, want.extraRBACMarkers)
	assertOverrideString(t, label, "packagePath", override.ServiceManager.PackagePath, want.packagePath)
}

func resourceOverridesByKind(service *ServiceConfig) map[string]ResourceGenerationOverride {
	overrides := make(map[string]ResourceGenerationOverride, len(service.Generation.Resources))
	for _, override := range service.Generation.Resources {
		overrides[override.Kind] = override
	}
	return overrides
}

func assertRuntimeDisabledOverride(t *testing.T, label string, overrides map[string]ResourceGenerationOverride, kinds ...string) {
	t.Helper()

	for _, kind := range kinds {
		override, ok := overrides[kind]
		if !ok {
			t.Fatalf("%s override for %s was not found", label, kind)
		}
		assertResourceOverride(t, label+" "+kind, override, resourceOverrideExpectation{
			controllerStrategy:     GenerationStrategyNone,
			serviceManagerStrategy: GenerationStrategyNone,
		})
	}
}

func assertFormalSpec(t *testing.T, service *ServiceConfig, kind string, want string) {
	t.Helper()
	if got := service.FormalSpecFor(kind); got != want {
		t.Fatalf("%s %s formalSpec = %q, want %q", service.Service, kind, got, want)
	}
}

func assertOverrideString(t *testing.T, label string, field string, got string, want string) {
	t.Helper()
	if want == "" {
		return
	}
	if got != want {
		t.Fatalf("%s %s = %q, want %q", label, field, got, want)
	}
}

func assertOverrideWebhookStrategy(t *testing.T, label string, got string, want string) {
	t.Helper()
	if want == "" {
		return
	}
	resolved := generationStrategyOrDefault(got, GenerationStrategyManual)
	if resolved != want {
		t.Fatalf("%s webhook strategy = %q, want %q", label, resolved, want)
	}
}

func assertOverrideInt(t *testing.T, label string, field string, got int, want int) {
	t.Helper()
	if want == 0 {
		return
	}
	if got != want {
		t.Fatalf("%s %s = %d, want %d", label, field, got, want)
	}
}

func assertOverrideStrings(t *testing.T, label string, field string, got []string, want []string) {
	t.Helper()
	if want == nil {
		return
	}
	if !slices.Equal(got, want) {
		t.Fatalf("%s %s = %v, want %v", label, field, got, want)
	}
}

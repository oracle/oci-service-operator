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
	if got := mysqlService.ControllerGenerationStrategy(); got != GenerationStrategyManual {
		t.Fatalf("mysql controller strategy = %q, want %q", got, GenerationStrategyManual)
	}
	if got := mysqlService.ServiceManagerGenerationStrategy(); got != GenerationStrategyGenerated {
		t.Fatalf("mysql service-manager strategy = %q, want %q", got, GenerationStrategyGenerated)
	}
	if got := mysqlService.RegistrationGenerationStrategy(); got != GenerationStrategyGenerated {
		t.Fatalf("mysql registration strategy = %q, want %q", got, GenerationStrategyGenerated)
	}
	if got := mysqlService.WebhookGenerationStrategy(); got != GenerationStrategyManual {
		t.Fatalf("mysql webhook strategy = %q, want %q", got, GenerationStrategyManual)
	}
	if len(mysqlService.Generation.Resources) != 1 {
		t.Fatalf("len(mysql generation resources) = %d, want 1", len(mysqlService.Generation.Resources))
	}

	override := mysqlService.Generation.Resources[0]
	if override.Kind != "MySqlDbSystem" {
		t.Fatalf("mysql override kind = %q, want %q", override.Kind, "MySqlDbSystem")
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

	coreService := cfg.Services[1]
	if got := coreService.ControllerGenerationStrategy(); got != GenerationStrategyNone {
		t.Fatalf("core controller strategy = %q, want %q", got, GenerationStrategyNone)
	}
	if got := coreService.ServiceManagerGenerationStrategy(); got != GenerationStrategyNone {
		t.Fatalf("core service-manager strategy = %q, want %q", got, GenerationStrategyNone)
	}
	if got := coreService.RegistrationGenerationStrategy(); got != GenerationStrategyNone {
		t.Fatalf("core registration strategy = %q, want %q", got, GenerationStrategyNone)
	}
	if got := coreService.WebhookGenerationStrategy(); got != GenerationStrategyManual {
		t.Fatalf("core webhook strategy = %q, want %q", got, GenerationStrategyManual)
	}
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

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var databaseService *ServiceConfig
	var mysqlService *ServiceConfig
	var streamingService *ServiceConfig
	var coreService *ServiceConfig
	for i := range cfg.Services {
		switch cfg.Services[i].Service {
		case "database":
			databaseService = &cfg.Services[i]
		case "mysql":
			mysqlService = &cfg.Services[i]
		case "streaming":
			streamingService = &cfg.Services[i]
		case "core":
			coreService = &cfg.Services[i]
		}
	}
	if databaseService == nil || mysqlService == nil || streamingService == nil || coreService == nil {
		t.Fatal("expected database, mysql, streaming, and core services in services.yaml")
	}

	for _, service := range []*ServiceConfig{databaseService, mysqlService, streamingService} {
		if got := service.ControllerGenerationStrategy(); got != GenerationStrategyGenerated {
			t.Fatalf("%s controller strategy = %q, want %q", service.Service, got, GenerationStrategyGenerated)
		}
		if got := service.ServiceManagerGenerationStrategy(); got != GenerationStrategyGenerated {
			t.Fatalf("%s service-manager strategy = %q, want %q", service.Service, got, GenerationStrategyGenerated)
		}
		if got := service.RegistrationGenerationStrategy(); got != GenerationStrategyGenerated {
			t.Fatalf("%s registration strategy = %q, want %q", service.Service, got, GenerationStrategyGenerated)
		}
		if got := service.WebhookGenerationStrategy(); got != GenerationStrategyManual {
			t.Fatalf("%s webhook strategy = %q, want %q", service.Service, got, GenerationStrategyManual)
		}
	}

	if len(databaseService.Generation.Resources) != 1 {
		t.Fatalf("database generation overrides = %d, want 1", len(databaseService.Generation.Resources))
	}
	if len(mysqlService.Generation.Resources) != 1 {
		t.Fatalf("mysql generation overrides = %d, want 1", len(mysqlService.Generation.Resources))
	}
	if len(streamingService.Generation.Resources) != 5 {
		t.Fatalf("streaming generation overrides = %d, want 5", len(streamingService.Generation.Resources))
	}

	if !slices.Equal(
		databaseService.Generation.Resources[0].Controller.ExtraRBACMarkers,
		[]string{
			`groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`,
			`groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
		},
	) {
		t.Fatalf("database extra RBAC markers = %v", databaseService.Generation.Resources[0].Controller.ExtraRBACMarkers)
	}
	if databaseService.Generation.Resources[0].ServiceManager.PackagePath != "autonomousdatabases/adb" {
		t.Fatalf("database packagePath = %q, want %q", databaseService.Generation.Resources[0].ServiceManager.PackagePath, "autonomousdatabases/adb")
	}
	if got := generationStrategyOrDefault(databaseService.Generation.Resources[0].Webhooks.Strategy, GenerationStrategyManual); got != GenerationStrategyManual {
		t.Fatalf("database resource webhook strategy = %q, want %q", got, GenerationStrategyManual)
	}

	if mysqlService.Generation.Resources[0].Controller.MaxConcurrentReconciles != 3 {
		t.Fatalf("mysql maxConcurrentReconciles = %d, want 3", mysqlService.Generation.Resources[0].Controller.MaxConcurrentReconciles)
	}
	if !slices.Equal(
		mysqlService.Generation.Resources[0].Controller.ExtraRBACMarkers,
		[]string{
			`groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`,
			`groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
		},
	) {
		t.Fatalf("mysql extra RBAC markers = %v", mysqlService.Generation.Resources[0].Controller.ExtraRBACMarkers)
	}
	if mysqlService.Generation.Resources[0].ServiceManager.PackagePath != "mysql/dbsystem" {
		t.Fatalf("mysql packagePath = %q, want %q", mysqlService.Generation.Resources[0].ServiceManager.PackagePath, "mysql/dbsystem")
	}

	streamingOverrides := make(map[string]ResourceGenerationOverride, len(streamingService.Generation.Resources))
	for _, override := range streamingService.Generation.Resources {
		streamingOverrides[override.Kind] = override
	}
	if streamingOverrides["Stream"].ServiceManager.PackagePath != "streaming/stream" {
		t.Fatalf("streaming packagePath = %q, want %q", streamingOverrides["Stream"].ServiceManager.PackagePath, "streaming/stream")
	}
	for _, kind := range []string{"Cursor", "Group", "GroupCursor", "Message"} {
		override, ok := streamingOverrides[kind]
		if !ok {
			t.Fatalf("streaming override for %s was not found", kind)
		}
		if override.Controller.Strategy != GenerationStrategyNone {
			t.Fatalf("streaming %s controller strategy = %q, want %q", kind, override.Controller.Strategy, GenerationStrategyNone)
		}
		if override.ServiceManager.Strategy != GenerationStrategyNone {
			t.Fatalf("streaming %s service-manager strategy = %q, want %q", kind, override.ServiceManager.Strategy, GenerationStrategyNone)
		}
	}

	if coreService.PackageProfile != PackageProfileControllerBacked {
		t.Fatalf("core packageProfile = %q, want %q", coreService.PackageProfile, PackageProfileControllerBacked)
	}
	if got := coreService.ControllerGenerationStrategy(); got != GenerationStrategyGenerated {
		t.Fatalf("core controller strategy = %q, want %q", got, GenerationStrategyGenerated)
	}
	if got := coreService.ServiceManagerGenerationStrategy(); got != GenerationStrategyGenerated {
		t.Fatalf("core service-manager strategy = %q, want %q", got, GenerationStrategyGenerated)
	}
	if got := coreService.RegistrationGenerationStrategy(); got != GenerationStrategyGenerated {
		t.Fatalf("core registration strategy = %q, want %q", got, GenerationStrategyGenerated)
	}
	if got := coreService.WebhookGenerationStrategy(); got != GenerationStrategyNone {
		t.Fatalf("core webhook strategy = %q, want %q", got, GenerationStrategyNone)
	}
}

func TestCheckedInConfigPromotesIdentityUserAndStreamingStreamFormalSpec(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var identityService *ServiceConfig
	var databaseService *ServiceConfig
	var mysqlService *ServiceConfig
	var streamingService *ServiceConfig
	for i := range cfg.Services {
		switch cfg.Services[i].Service {
		case "identity":
			identityService = &cfg.Services[i]
		case "database":
			databaseService = &cfg.Services[i]
		case "mysql":
			mysqlService = &cfg.Services[i]
		case "streaming":
			streamingService = &cfg.Services[i]
		}
	}
	if identityService == nil || databaseService == nil || mysqlService == nil || streamingService == nil {
		t.Fatal("expected identity, database, mysql, and streaming services in services.yaml")
	}

	if got := identityService.FormalSpecFor("User"); got != "user" {
		t.Fatalf("identity User formalSpec = %q, want %q", got, "user")
	}
	if got := identityService.FormalSpecFor("Compartment"); got != "" {
		t.Fatalf("identity Compartment formalSpec = %q, want empty", got)
	}
	if got := streamingService.FormalSpecFor("Stream"); got != "stream" {
		t.Fatalf("streaming Stream formalSpec = %q, want %q", got, "stream")
	}

	for _, test := range []struct {
		service *ServiceConfig
		kind    string
	}{
		{service: databaseService, kind: "AutonomousDatabases"},
		{service: mysqlService, kind: "MySqlDbSystem"},
	} {
		if got := test.service.FormalSpecFor(test.kind); got != "" {
			t.Fatalf("%s %s formalSpec = %q, want empty", test.service.Service, test.kind, got)
		}
	}
}

func TestCheckedInConfigOptsOutEndpointBasedGeneratedRuntimeResources(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	wantKinds := map[string][]string{
		"keymanagement": {"Key", "KeyVersion", "ReplicationStatus", "WrappingKey"},
		"streaming":     {"Cursor", "Group", "GroupCursor", "Message"},
	}

	for serviceName, kinds := range wantKinds {
		var service *ServiceConfig
		for i := range cfg.Services {
			if cfg.Services[i].Service == serviceName {
				service = &cfg.Services[i]
				break
			}
		}
		if service == nil {
			t.Fatalf("service %q was not found in services.yaml", serviceName)
		}

		overrides := make(map[string]ResourceGenerationOverride, len(service.Generation.Resources))
		for _, override := range service.Generation.Resources {
			overrides[override.Kind] = override
		}

		for _, kind := range kinds {
			override, ok := overrides[kind]
			if !ok {
				t.Fatalf("%s override for %s was not found", serviceName, kind)
			}
			if override.Controller.Strategy != GenerationStrategyNone {
				t.Fatalf("%s %s controller strategy = %q, want %q", serviceName, kind, override.Controller.Strategy, GenerationStrategyNone)
			}
			if override.ServiceManager.Strategy != GenerationStrategyNone {
				t.Fatalf("%s %s service-manager strategy = %q, want %q", serviceName, kind, override.ServiceManager.Strategy, GenerationStrategyNone)
			}
		}
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

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"os"
	"path/filepath"
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
generatorEntrypoint: ./cmd/osok-api-generator
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

/*
 Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
 Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package formalscaffold

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/internal/formal"
)

const testGeneratorConfig = `schemaVersion: v1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: Shared manager install
services:
  - service: identity
    sdkPackage: github.com/oracle/oci-go-sdk/v65/identity
    group: identity
    version: v1beta1
    phase: security-and-identity
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: all
`

const testSourcesLock = `{
  "schemaVersion": 1,
  "sources": [
    {
      "name": "terraform-provider-oci",
      "surface": "provider-facts",
      "status": "scaffold",
      "notes": [
        "formal-import will pin a provider revision here."
      ]
    }
  ]
}
`

const testBaseReconcilerContract = `------------------------------ MODULE BaseReconcilerContract ------------------------------
=============================================================================
`

const testControllerLifecycleContract = `------------------------------ MODULE ControllerLifecycleSpec ------------------------------
=============================================================================
`

const testServiceManagerContract = `------------------------------ MODULE OSOKServiceManagerContract ------------------------------
=============================================================================
`

const testSecretSideEffectsContract = `------------------------------ MODULE SecretSideEffectsContract ------------------------------
=============================================================================
`

const testTemplateSpec = `# formal controller binding schema v1
schema_version = 1
surface = repo-authored-semantics
service = template
slug = template
kind = Template
stage = scaffold
import = imports/template/template.json
shared_contracts = shared/BaseReconcilerContract.tla,shared/ControllerLifecycleSpec.tla,shared/OSOKServiceManagerContract.tla,shared/SecretSideEffectsContract.tla
status_projection = required
success_condition = active
requeue_conditions = provisioning,updating,terminating
delete_confirmation = required
finalizer_policy = retain-until-confirmed-delete
secret_side_effects = none
`

const testTemplateLogic = `---
schemaVersion: 1
surface: repo-authored-semantics
service: template
slug: template
gaps: []
---

# Logic Gaps

Template scaffold row.
`

const testTemplateDiagram = `schemaVersion: 1
surface: repo-authored-semantics
service: template
slug: template
kind: Template
archetype: generated-service-manager
states:
  - provisioning
  - active
  - updating
  - terminating
notes:
  - Template scaffold row.
`

const testTemplateImport = `{
  "schemaVersion": 1,
  "surface": "provider-facts",
  "service": "template",
  "slug": "template",
  "kind": "Template",
  "sourceRef": "terraform-provider-oci",
  "providerResource": "template_resource",
  "operations": {
    "create": [
      {
        "operation": "CreateTemplate",
        "requestType": "CreateTemplateRequest",
        "responseType": "CreateTemplateResponse"
      }
    ],
    "get": [
      {
        "operation": "GetTemplate",
        "requestType": "GetTemplateRequest",
        "responseType": "GetTemplateResponse"
      }
    ],
    "list": [
      {
        "operation": "ListTemplate",
        "requestType": "ListTemplateRequest",
        "responseType": "ListTemplateResponse"
      }
    ],
    "update": [
      {
        "operation": "UpdateTemplate",
        "requestType": "UpdateTemplateRequest",
        "responseType": "UpdateTemplateResponse"
      }
    ],
    "delete": [
      {
        "operation": "DeleteTemplate",
        "requestType": "DeleteTemplateRequest",
        "responseType": "DeleteTemplateResponse"
      }
    ]
  },
  "lifecycle": {
    "create": {
      "pending": [
        "PROVISIONING"
      ],
      "target": [
        "ACTIVE"
      ]
    },
    "update": {
      "pending": [
        "UPDATING"
      ],
      "target": [
        "ACTIVE"
      ]
    }
  },
  "mutation": {
    "mutable": [
      "display_name"
    ],
    "forceNew": [
      "compartment_id"
    ],
    "conflictsWith": {}
  },
  "hooks": {
    "create": [
      {
        "helper": "tfresource.CreateResource"
      }
    ],
    "update": [
      {
        "helper": "tfresource.UpdateResource"
      }
    ],
    "delete": [
      {
        "helper": "tfresource.DeleteResource"
      }
    ]
  },
  "deleteConfirmation": {
    "pending": [
      "DELETING"
    ],
    "target": [
      "DELETED"
    ]
  },
  "listLookup": {
    "datasource": "oci_template_templates",
    "collectionField": "templates",
    "responseItemsField": "Items",
    "filterFields": [
      "compartment_id",
      "state"
    ]
  },
  "boundary": {
    "providerFactsOnly": true,
    "repoAuthoredSpecPath": "controllers/template/spec.cfg",
    "repoAuthoredLogicGapsPath": "controllers/template/logic-gaps.md",
    "excludedSemantics": [
      "bind-versus-create",
      "secret-output",
      "delete-confirmation"
    ]
  }
}
`

const testTemplateManifestRow = "template\ttemplate\tTemplate\tscaffold\trepo-authored-semantics\timports/template/template.json\tcontrollers/template/spec.cfg\tcontrollers/template/logic-gaps.md\tcontrollers/template/diagrams\n"

const testSeededManifestRow = "identity\tuser\tUser\tseeded\trepo-authored-semantics\timports/identity/user.json\tcontrollers/identity/user/spec.cfg\tcontrollers/identity/user/logic-gaps.md\tcontrollers/identity/user/diagrams\n"

const testSeededSpec = `# formal controller binding schema v1
schema_version = 1
surface = repo-authored-semantics
service = identity
slug = user
kind = User
stage = seeded
import = imports/identity/user.json
shared_contracts = shared/BaseReconcilerContract.tla,shared/ControllerLifecycleSpec.tla,shared/OSOKServiceManagerContract.tla,shared/SecretSideEffectsContract.tla
status_projection = required
success_condition = active
requeue_conditions = provisioning,updating,terminating
delete_confirmation = required
finalizer_policy = retain-until-confirmed-delete
secret_side_effects = none
`

const testSeededLogic = `---
schemaVersion: 1
surface: repo-authored-semantics
service: identity
slug: user
gaps: []
---

# Logic Gaps

Seeded row should be preserved.
`

const testSeededDiagram = `schemaVersion: 1
surface: repo-authored-semantics
service: identity
slug: user
kind: User
archetype: generated-service-manager
states:
  - provisioning
  - active
  - updating
  - terminating
notes:
  - Seeded row should be preserved.
`

const testSeededImport = `{
  "schemaVersion": 1,
  "surface": "provider-facts",
  "service": "identity",
  "slug": "user",
  "kind": "User",
  "sourceRef": "terraform-provider-oci",
  "providerResource": "oci_identity_user",
  "operations": {
    "create": [
      {
        "operation": "CreateUser",
        "requestType": "identity.CreateUserRequest",
        "responseType": "identity.CreateUserResponse"
      }
    ],
    "get": [
      {
        "operation": "GetUser",
        "requestType": "identity.GetUserRequest",
        "responseType": "identity.GetUserResponse"
      }
    ],
    "list": [
      {
        "operation": "ListUsers",
        "requestType": "identity.ListUsersRequest",
        "responseType": "identity.ListUsersResponse"
      }
    ],
    "update": [
      {
        "operation": "UpdateUser",
        "requestType": "identity.UpdateUserRequest",
        "responseType": "identity.UpdateUserResponse"
      }
    ],
    "delete": [
      {
        "operation": "DeleteUser",
        "requestType": "identity.DeleteUserRequest",
        "responseType": "identity.DeleteUserResponse"
      }
    ]
  },
  "lifecycle": {
    "create": {
      "pending": [
        "CREATING"
      ],
      "target": [
        "ACTIVE"
      ]
    },
    "update": {
      "pending": [
        "UPDATING"
      ],
      "target": [
        "ACTIVE"
      ]
    }
  },
  "mutation": {
    "mutable": [
      "description"
    ],
    "forceNew": [
      "name"
    ],
    "conflictsWith": {}
  },
  "hooks": {
    "create": [
      {
        "helper": "identity.CreateUser"
      }
    ],
    "update": [
      {
        "helper": "identity.UpdateUser"
      }
    ],
    "delete": [
      {
        "helper": "identity.DeleteUser"
      }
    ]
  },
  "deleteConfirmation": {
    "pending": [
      "DELETING"
    ],
    "target": [
      "DELETED"
    ]
  },
  "listLookup": {
    "datasource": "oci_identity_users",
    "collectionField": "users",
    "responseItemsField": "Items",
    "filterFields": [
      "compartment_id",
      "state"
    ]
  },
  "boundary": {
    "providerFactsOnly": true,
    "repoAuthoredSpecPath": "controllers/identity/user/spec.cfg",
    "repoAuthoredLogicGapsPath": "controllers/identity/user/logic-gaps.md",
    "excludedSemantics": [
      "secret-output"
    ]
  },
  "notes": [
    "seeded identity user"
  ]
}
`

const testUserAPI = `package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
type User struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `
}

// +kubebuilder:object:root=true
type UserList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items []User ` + "`json:\"items\"`" + `
}
`

const testNetworkSourceAPI = `package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
type NetworkSource struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `
}

// +kubebuilder:object:root=true
type NetworkSourceList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items []NetworkSource ` + "`json:\"items\"`" + `
}
`

const testDBSystemDBInstanceAPI = `package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
type DbSystemDbInstance struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `
}

// +kubebuilder:object:root=true
type DbSystemDbInstanceList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items []DbSystemDbInstance ` + "`json:\"items\"`" + `
}
`

const testAutonomousDatabaseAPI = `package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
type AutonomousDatabase struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `
}

// +kubebuilder:object:root=true
type AutonomousDatabaseList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items []AutonomousDatabase ` + "`json:\"items\"`" + `
}
`

const testBackupAPI = `package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
type Backup struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `
}

// +kubebuilder:object:root=true
type BackupList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items []Backup ` + "`json:\"items\"`" + `
}
`

const testSecurityListAPI = `package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
type SecurityList struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `
}

// +kubebuilder:object:root=true
type SecurityListList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items []SecurityList ` + "`json:\"items\"`" + `
}
`

func TestGenerateAddsScaffoldsForPublishedKindsAndPreservesSeededRows(t *testing.T) {
	requirePlantUML(t)
	repoRoot := writeTestRepo(t)

	report, err := Generate(Options{
		Root:       filepath.Join(repoRoot, "formal"),
		ConfigPath: filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	assertScaffoldReport(t, report, 1, 2, 1, 3)

	catalog, err := formal.LoadCatalog(filepath.Join(repoRoot, "formal"))
	if err != nil {
		t.Fatalf("formal.LoadCatalog() error = %v", err)
	}
	assertSeededCatalogRowPreserved(t, catalog)
	assertSharedDiagramStrategyArtifacts(t, filepath.Join(repoRoot, "formal", "shared", "diagrams"))
	assertRenderedDiagramFamily(t, filepath.Join(repoRoot, "formal", "controllers", "identity", "user", "diagrams"))
	assertGeneratedCatalogRowAdded(t, catalog)
	assertRenderedDiagramFamily(t, filepath.Join(repoRoot, "formal", "controllers", "identity", "networksource", "diagrams"))
}

func TestGenerateUsesFileStemAsFormalSlug(t *testing.T) {
	requirePlantUML(t)
	repoRoot := t.TempDir()
	writeScaffoldBase(t, repoRoot)
	writeTestFile(t, filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"), strings.ReplaceAll(testGeneratorConfig, "identity", "psql"))
	writeTestFile(t, filepath.Join(repoRoot, "api", "psql", "v1beta1", "dbsystemdbinstance_types.go"), testDBSystemDBInstanceAPI)

	report, err := Generate(Options{
		Root:       filepath.Join(repoRoot, "formal"),
		ConfigPath: filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if report.NewRows != 1 {
		t.Fatalf("report.NewRows = %d, want 1", report.NewRows)
	}

	catalog, err := formal.LoadCatalog(filepath.Join(repoRoot, "formal"))
	if err != nil {
		t.Fatalf("formal.LoadCatalog() error = %v", err)
	}
	binding, ok := catalog.Lookup("psql", "dbsystemdbinstance")
	if !ok {
		t.Fatal("catalog.Lookup(psql, dbsystemdbinstance) unexpectedly missed")
	}
	if binding.Manifest.Kind != "DbSystemDbInstance" {
		t.Fatalf("binding kind = %q, want %q", binding.Manifest.Kind, "DbSystemDbInstance")
	}
}

func TestGenerateUsesConfiguredFormalSpecSlug(t *testing.T) {
	requirePlantUML(t)
	repoRoot := t.TempDir()
	writeScaffoldBase(t, repoRoot)
	writeTestFile(t, filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"), `schemaVersion: v1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: Shared manager install
services:
  - service: database
    sdkPackage: github.com/oracle/oci-go-sdk/v65/database
    group: database
    version: v1beta1
    phase: data-and-storage
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - AutonomousDatabase
    generation:
      resources:
        - kind: AutonomousDatabase
          formalSpec: databaseautonomousdatabase
`)
	writeTestFile(t, filepath.Join(repoRoot, "api", "database", "v1beta1", "autonomousdatabase_types.go"), testAutonomousDatabaseAPI)
	writeTestFile(t, filepath.Join(repoRoot, "api", "database", "v1beta1", "backup_types.go"), testBackupAPI)

	report, err := Generate(Options{
		Root:       filepath.Join(repoRoot, "formal"),
		ConfigPath: filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if report.NewRows != 1 {
		t.Fatalf("report.NewRows = %d, want 1", report.NewRows)
	}

	catalog, err := formal.LoadCatalog(filepath.Join(repoRoot, "formal"))
	if err != nil {
		t.Fatalf("formal.LoadCatalog() error = %v", err)
	}
	if _, ok := catalog.Lookup("database", "autonomousdatabase"); ok {
		t.Fatal("catalog.Lookup(database, autonomousdatabase) unexpectedly found file-stem row")
	}
	if _, ok := catalog.Lookup("database", "backup"); ok {
		t.Fatal("catalog.Lookup(database, backup) unexpectedly found excluded kind row")
	}
	binding, ok := catalog.Lookup("database", "databaseautonomousdatabase")
	if !ok {
		t.Fatal("catalog.Lookup(database, databaseautonomousdatabase) unexpectedly missed")
	}
	if binding.Manifest.Kind != "AutonomousDatabase" {
		t.Fatalf("binding kind = %q, want %q", binding.Manifest.Kind, "AutonomousDatabase")
	}
}

func TestPublishedKindFromFileRejectsFilesWithoutRootKinds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad_types.go")
	writeTestFile(t, path, `package v1beta1
type helper struct{}
`)

	_, err := publishedKindFromFile(path)
	if err == nil || !strings.Contains(err.Error(), "does not define a non-list kubebuilder root kind") {
		t.Fatalf("publishedKindFromFile() error = %v, want missing root-kind failure", err)
	}
}

func TestPublishedKindFromFileAllowsKindsThatEndWithList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "securitylist_types.go")
	writeTestFile(t, path, testSecurityListAPI)

	kind, err := publishedKindFromFile(path)
	if err != nil {
		t.Fatalf("publishedKindFromFile() error = %v", err)
	}
	if kind != "SecurityList" {
		t.Fatalf("kind = %q, want %q", kind, "SecurityList")
	}
}

func TestFilterProviderInventoryEntriesAppliesPublishedSelection(t *testing.T) {
	t.Parallel()

	got := filterProviderInventoryEntries(
		[]inventoryEntry{
			{
				Service: "database",
				Group:   "database",
				Slug:    "databaseautonomousdatabase",
				Kind:    "AutonomousDatabase",
			},
			{
				Service: "identity",
				Group:   "identity",
				Slug:    "user",
				Kind:    "User",
			},
		},
		[]inventoryEntry{
			{
				Service:          "database",
				Group:            "database",
				Slug:             "autonomousdatabase",
				Kind:             "AutonomousDatabase",
				ProviderResource: "oci_database_autonomous_database",
			},
			{
				Service:          "database",
				Group:            "database",
				Slug:             "backup",
				Kind:             "Backup",
				ProviderResource: "oci_database_backup",
			},
			{
				Service:          "identity",
				Group:            "identity",
				Slug:             "user",
				Kind:             "User",
				ProviderResource: "oci_identity_user",
			},
			{
				Service:          "widget",
				Group:            "widget",
				Slug:             "widget",
				Kind:             "Widget",
				ProviderResource: "oci_widget_widget",
			},
		},
	)

	want := []inventoryEntry{
		{
			Service:          "database",
			Group:            "database",
			Slug:             "databaseautonomousdatabase",
			Kind:             "AutonomousDatabase",
			ProviderResource: "oci_database_autonomous_database",
		},
		{
			Service:          "identity",
			Group:            "identity",
			Slug:             "user",
			Kind:             "User",
			ProviderResource: "oci_identity_user",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterProviderInventoryEntries() = %#v, want %#v", got, want)
	}
}

func TestGenerateIgnoresProviderInventoryOutsidePublishedActiveSurface(t *testing.T) {
	requirePlantUML(t)
	repoRoot := writeTestRepo(t)
	providerRoot := writeScaffoldProviderFixture(t)

	report, err := Generate(Options{
		Root:         filepath.Join(repoRoot, "formal"),
		ConfigPath:   filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
		ProviderPath: providerRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if report.ProviderKinds != 0 {
		t.Fatalf("report.ProviderKinds = %d, want 0", report.ProviderKinds)
	}

	catalog, err := formal.LoadCatalog(filepath.Join(repoRoot, "formal"))
	if err != nil {
		t.Fatalf("formal.LoadCatalog() error = %v", err)
	}
	if _, ok := catalog.Lookup("widget", "widget"); ok {
		t.Fatal("catalog.Lookup(widget, widget) unexpectedly found filtered provider-only row")
	}
	assertSharedDiagramStrategyArtifacts(t, filepath.Join(repoRoot, "formal", "shared", "diagrams"))
	assertGeneratedCatalogRowAdded(t, catalog)
	assertRenderedDiagramFamily(t, filepath.Join(repoRoot, "formal", "controllers", "identity", "networksource", "diagrams"))
}

func TestGeneratePrunesRowsOutsideDefaultActiveSurface(t *testing.T) {
	requirePlantUML(t)
	repoRoot := writeTestRepo(t)
	writeTestFile(
		t,
		filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
		strings.Replace(testGeneratorConfig, "enabled: true", "enabled: false", 1),
	)

	report, err := Generate(Options{
		Root:       filepath.Join(repoRoot, "formal"),
		ConfigPath: filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if report.ServicesScanned != 0 {
		t.Fatalf("report.ServicesScanned = %d, want 0", report.ServicesScanned)
	}
	if report.PublishedKinds != 0 {
		t.Fatalf("report.PublishedKinds = %d, want 0", report.PublishedKinds)
	}
	if report.ManifestRows != 1 {
		t.Fatalf("report.ManifestRows = %d, want 1", report.ManifestRows)
	}

	catalog, err := formal.LoadCatalog(filepath.Join(repoRoot, "formal"))
	if err != nil {
		t.Fatalf("formal.LoadCatalog() error = %v", err)
	}
	if _, ok := catalog.Lookup("identity", "user"); ok {
		t.Fatal("catalog.Lookup(identity, user) unexpectedly found pruned row")
	}
	assertPathNotExists(t, filepath.Join(repoRoot, "formal", "controllers", "identity", "user"))
	assertPathNotExists(t, filepath.Join(repoRoot, "formal", "imports", "identity", "user.json"))
}

func TestGeneratePrunesOrphanedArtifactsWhenManifestAlreadyMatchesActiveSurface(t *testing.T) {
	requirePlantUML(t)
	repoRoot := writeTestRepo(t)
	writeTestFile(
		t,
		filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
		strings.Replace(testGeneratorConfig, "enabled: true", "enabled: false", 1),
	)
	writeTestFile(
		t,
		filepath.Join(repoRoot, "formal", "controller_manifest.tsv"),
		manifestHeader+testTemplateManifestRow,
	)

	report, err := Generate(Options{
		Root:       filepath.Join(repoRoot, "formal"),
		ConfigPath: filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if report.ManifestRows != 1 {
		t.Fatalf("report.ManifestRows = %d, want 1", report.ManifestRows)
	}

	assertPathNotExists(t, filepath.Join(repoRoot, "formal", "controllers", "identity", "user"))
	assertPathNotExists(t, filepath.Join(repoRoot, "formal", "imports", "identity", "user.json"))
}

func TestGeneratePreservesConfiguredFormalSpecRowsOutsideDefaultActiveSurface(t *testing.T) {
	requirePlantUML(t)
	repoRoot := writeTestRepo(t)
	config := strings.Replace(
		testGeneratorConfig,
		"selection:\n      enabled: true\n      mode: all\n",
		"selection:\n      enabled: false\n      mode: all\n    generation:\n      resources:\n        - kind: User\n          formalSpec: user\n",
		1,
	)
	writeTestFile(t, filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"), config)
	writeTestFile(
		t,
		filepath.Join(repoRoot, "formal", "controller_manifest.tsv"),
		manifestHeader+"template\ttemplate\tTemplate\tscaffold\trepo-authored-semantics\timports/template/template.json\tcontrollers/template/spec.cfg\tcontrollers/template/logic-gaps.md\tcontrollers/template/diagrams\n",
	)

	report, err := Generate(Options{
		Root:       filepath.Join(repoRoot, "formal"),
		ConfigPath: filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if report.ServicesScanned != 0 {
		t.Fatalf("report.ServicesScanned = %d, want 0", report.ServicesScanned)
	}
	if report.PublishedKinds != 0 {
		t.Fatalf("report.PublishedKinds = %d, want 0", report.PublishedKinds)
	}
	if report.NewRows != 1 {
		t.Fatalf("report.NewRows = %d, want 1", report.NewRows)
	}
	if report.ManifestRows != 2 {
		t.Fatalf("report.ManifestRows = %d, want 2", report.ManifestRows)
	}

	catalog, err := formal.LoadCatalog(filepath.Join(repoRoot, "formal"))
	if err != nil {
		t.Fatalf("formal.LoadCatalog() error = %v", err)
	}
	if _, ok := catalog.Lookup("identity", "user"); !ok {
		t.Fatal("catalog.Lookup(identity, user) unexpectedly missed preserved formal-spec row")
	}
}

func TestVerifyCoverageIgnoresProviderInventoryOutsidePublishedActiveSurface(t *testing.T) {
	requirePlantUML(t)
	repoRoot := writeTestRepo(t)
	providerRoot := writeScaffoldProviderFixture(t)
	if _, err := Generate(Options{
		Root:       filepath.Join(repoRoot, "formal"),
		ConfigPath: filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
	}); err != nil {
		t.Fatalf("Generate() preflight error = %v", err)
	}

	report, err := VerifyCoverage(Options{
		Root:         filepath.Join(repoRoot, "formal"),
		ConfigPath:   filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"),
		ProviderPath: providerRoot,
	})
	if err != nil {
		t.Fatalf("VerifyCoverage() error = %v", err)
	}
	if report.ProviderKinds != 0 {
		t.Fatalf("report.ProviderKinds = %d, want 0", report.ProviderKinds)
	}
}

func assertScaffoldReport(t *testing.T, report Report, servicesScanned, publishedKinds, newRows, manifestRows int) {
	t.Helper()

	if report.ServicesScanned != servicesScanned {
		t.Fatalf("report.ServicesScanned = %d, want %d", report.ServicesScanned, servicesScanned)
	}
	if report.PublishedKinds != publishedKinds {
		t.Fatalf("report.PublishedKinds = %d, want %d", report.PublishedKinds, publishedKinds)
	}
	if report.NewRows != newRows {
		t.Fatalf("report.NewRows = %d, want %d", report.NewRows, newRows)
	}
	if report.ManifestRows != manifestRows {
		t.Fatalf("report.ManifestRows = %d, want %d", report.ManifestRows, manifestRows)
	}
}

func assertPathNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected %q to be removed", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
}

func assertSeededCatalogRowPreserved(t *testing.T, catalog *formal.Catalog) {
	t.Helper()

	user, ok := catalog.Lookup("identity", "user")
	if !ok {
		t.Fatal("catalog.Lookup(identity, user) unexpectedly missed")
	}
	if user.Spec.Stage != "seeded" {
		t.Fatalf("user stage = %q, want %q", user.Spec.Stage, "seeded")
	}
	if got := strings.Join(user.Import.Notes, ","); !strings.Contains(got, "seeded identity user") {
		t.Fatalf("user import notes = %q, want seeded note preserved", got)
	}
}

func assertGeneratedCatalogRowAdded(t *testing.T, catalog *formal.Catalog) {
	t.Helper()

	networkSource, ok := catalog.Lookup("identity", "networksource")
	if !ok {
		t.Fatal("catalog.Lookup(identity, networksource) unexpectedly missed")
	}
	if networkSource.Spec.Stage != "scaffold" {
		t.Fatalf("networksource stage = %q, want %q", networkSource.Spec.Stage, "scaffold")
	}
	if networkSource.Manifest.Kind != "NetworkSource" {
		t.Fatalf("networksource kind = %q, want %q", networkSource.Manifest.Kind, "NetworkSource")
	}
	if networkSource.Import.ProviderResource != "scaffold_identity_networksource" {
		t.Fatalf("networksource providerResource = %q, want %q", networkSource.Import.ProviderResource, "scaffold_identity_networksource")
	}
	if networkSource.Import.Boundary.RepoAuthoredSpecPath != "controllers/identity/networksource/spec.cfg" {
		t.Fatalf("networksource repoAuthoredSpecPath = %q", networkSource.Import.Boundary.RepoAuthoredSpecPath)
	}
}

func writeTestRepo(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	writeScaffoldBase(t, repoRoot)
	writeTestFile(t, filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml"), testGeneratorConfig)
	writeTestFile(t, filepath.Join(repoRoot, "api", "identity", "v1beta1", "user_types.go"), testUserAPI)
	writeTestFile(t, filepath.Join(repoRoot, "api", "identity", "v1beta1", "networksource_types.go"), testNetworkSourceAPI)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "controller_manifest.tsv"), manifestHeader+testTemplateManifestRow+testSeededManifestRow)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "controllers", "identity", "user", "spec.cfg"), testSeededSpec)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "controllers", "identity", "user", "logic-gaps.md"), testSeededLogic)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "controllers", "identity", "user", "diagrams", "runtime-lifecycle.yaml"), testSeededDiagram)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "imports", "identity", "user.json"), testSeededImport)
	return repoRoot
}

func writeScaffoldBase(t *testing.T, repoRoot string) {
	t.Helper()

	writeTestFile(t, filepath.Join(repoRoot, "formal", "controller_manifest.tsv"), manifestHeader+testTemplateManifestRow)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "sources.lock"), testSourcesLock)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "shared", "BaseReconcilerContract.tla"), testBaseReconcilerContract)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "shared", "ControllerLifecycleSpec.tla"), testControllerLifecycleContract)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "shared", "OSOKServiceManagerContract.tla"), testServiceManagerContract)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "shared", "SecretSideEffectsContract.tla"), testSecretSideEffectsContract)
	writeDiagramStrategyFixtures(t, repoRoot)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "controllers", "template", "spec.cfg"), testTemplateSpec)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "controllers", "template", "logic-gaps.md"), testTemplateLogic)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "controllers", "template", "diagrams", "runtime-lifecycle.yaml"), testTemplateDiagram)
	writeTestFile(t, filepath.Join(repoRoot, "formal", "imports", "template", "template.json"), testTemplateImport)
}

func writeScaffoldProviderFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/provider\n\ngo 1.22\n")
	writeTestFile(t, filepath.Join(root, "internal", "schema", "schema.go"), `package schema

type ValueType int

const (
	TypeString ValueType = iota
	TypeList
)

type Resource struct {
	Read   func(*ResourceData, interface{}) error
	Schema map[string]*Schema
}

type Schema struct {
	Type     ValueType
	Optional bool
	Required bool
	Computed bool
	Elem     interface{}
}

type ResourceData struct{}
`)
	writeTestFile(t, filepath.Join(root, "internal", "tfresource", "tfresource.go"), `package tfresource

import "example.com/provider/internal/schema"

func RegisterResource(string, *schema.Resource) {}
func RegisterDatasource(string, *schema.Resource) {}
func ReadResource(interface{}) error { return nil }
func GetDataSourceItemSchema(*schema.Resource) *schema.Resource { return nil }
`)
	writeTestFile(t, filepath.Join(root, "internal", "service", "widget", "register_resource.go"), `package widget

import "example.com/provider/internal/tfresource"

func RegisterResource() {
	tfresource.RegisterResource("oci_widget_widget", WidgetResource())
}
`)
	writeTestFile(t, filepath.Join(root, "internal", "service", "widget", "register_datasource.go"), `package widget

import "example.com/provider/internal/tfresource"

func RegisterDatasource() {
	tfresource.RegisterDatasource("oci_widget_widgets", WidgetsDataSource())
}
`)
	writeTestFile(t, filepath.Join(root, "internal", "service", "widget", "widget_resource.go"), `package widget

import "example.com/provider/internal/schema"

func WidgetResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"display_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}
`)
	writeTestFile(t, filepath.Join(root, "internal", "service", "widget", "widgets_data_source.go"), `package widget

import (
	"example.com/provider/internal/schema"
	"example.com/provider/internal/tfresource"
)

func WidgetsDataSource() *schema.Resource {
	return &schema.Resource{
		Read: readWidgets,
		Schema: map[string]*schema.Schema{
			"widgets": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     tfresource.GetDataSourceItemSchema(WidgetResource()),
			},
		},
	}
}

func readWidgets(*schema.ResourceData, interface{}) error {
	return nil
}
`)
	return root
}

func assertRenderedDiagramFamily(t *testing.T, dir string) {
	t.Helper()
	for _, name := range []string{
		"runtime-lifecycle.yaml",
		"activity.puml",
		"activity.svg",
		"sequence.puml",
		"sequence.svg",
		"state-machine.puml",
		"state-machine.svg",
	} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}
		if info.IsDir() {
			t.Fatalf("%q is a directory, want file", path)
		}
	}
}

func assertSharedDiagramStrategyArtifacts(t *testing.T, dir string) {
	t.Helper()
	for _, name := range []string{
		"shared-reconcile-activity.puml",
		"shared-reconcile-activity.svg",
		"shared-resolution-sequence.puml",
		"shared-resolution-sequence.svg",
		"shared-delete-sequence.puml",
		"shared-delete-sequence.svg",
		"shared-controller-state-machine.puml",
		"shared-controller-state-machine.svg",
		"shared-legend.puml",
		"shared-legend.svg",
	} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}
		if info.IsDir() {
			t.Fatalf("%q is a directory, want file", path)
		}
	}
}

func writeTestFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

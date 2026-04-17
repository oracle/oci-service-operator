package generator

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/internal/formal"
)

func TestGenerateWritesMutabilityOverlayArtifactFromFixture(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	configPath := filepath.Join(repo, "internal", "generator", "config", "services.yaml")
	writeGeneratorTestFile(t, configPath, `schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  crd-only:
    description: CRD-only groups
services:
  - service: mysql
    sdkPackage: example.com/test/sdk
    group: mysql
    packageProfile: crd-only
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - Widget
    async:
      strategy: lifecycle
      runtime: generatedruntime
    generation:
      resources:
        - kind: Widget
          formalSpec: widget
`)
	writeMutabilityOverlayWidgetFormalScaffold(t, repo)
	target := mutabilityOverlayWidgetTarget()
	writeMutabilityOverlayFixture(t, filepath.Join(repo, filepath.FromSlash(mutabilityOverlayDocsFixtureRootRelative)), target, `
<!doctype html>
<html>
  <body>
    <h2 id="argument-reference">Argument Reference</h2>
    <ul>
      <li><code>display_name</code> - The display name of the widget. (Updatable)</li>
      <li><code>name</code> - Updating this value after creation is not supported.</li>
    </ul>
  </body>
</html>
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", configPath, err)
	}
	services, err := cfg.SelectServices("mysql", false)
	if err != nil {
		t.Fatalf("SelectServices(mysql) error = %v", err)
	}

	pipeline := New()
	pipeline.discoverer = &Discoverer{
		resolveDir: func(context.Context, string) (string, error) {
			return sampleSDKDir(t), nil
		},
	}
	pipeline.mutabilityOverlayDocsFetcher = stubMutabilityOverlayDocsFetcher{}

	outputRoot := t.TempDir()
	result, err := pipeline.Generate(context.Background(), cfg, services, Options{
		OutputRoot:              outputRoot,
		EnableMutabilityOverlay: true,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated = %d services, want 1", len(result.Generated))
	}

	artifactPath := filepath.Join(outputRoot, filepath.FromSlash(mutabilityOverlayGeneratedRootRelativePath), "mysql", "widget.json")
	content := readFile(t, artifactPath)

	var doc mutabilityOverlayDocument
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", artifactPath, err)
	}

	if doc.Metadata.ProviderSourceRef != "terraform-provider-oci" {
		t.Fatalf("ProviderSourceRef = %q, want terraform-provider-oci", doc.Metadata.ProviderSourceRef)
	}
	if doc.Metadata.ProviderRevision != "test-provider-revision" {
		t.Fatalf("ProviderRevision = %q, want test-provider-revision", doc.Metadata.ProviderRevision)
	}
	if doc.Metadata.TerraformDocsVersion != defaultMutabilityOverlayTerraformDocsVersion {
		t.Fatalf("TerraformDocsVersion = %q, want %q", doc.Metadata.TerraformDocsVersion, defaultMutabilityOverlayTerraformDocsVersion)
	}
	if doc.Resource.Kind != "Widget" || doc.Resource.FormalSlug != "widget" || doc.Resource.ProviderResource != "oci_mysql_widget" {
		t.Fatalf("resource identity = %+v, want Widget/widget/oci_mysql_widget", doc.Resource)
	}
	if len(doc.Fields) != 2 {
		t.Fatalf("fields = %d, want 2", len(doc.Fields))
	}

	displayName := findMutabilityOverlayField(t, doc.Fields, "displayName")
	if displayName.Docs.EvidenceState != mutabilityOverlayDocsStateConfirmedUpdatable {
		t.Fatalf("displayName docs evidence = %q, want %q", displayName.Docs.EvidenceState, mutabilityOverlayDocsStateConfirmedUpdatable)
	}
	if displayName.Merge.FinalPolicy != mutabilityOverlayPolicyAllowInPlaceUpdate {
		t.Fatalf("displayName final policy = %q, want %q", displayName.Merge.FinalPolicy, mutabilityOverlayPolicyAllowInPlaceUpdate)
	}
	if got := displayName.AST.ConflictsWith; len(got) != 1 || got[0] != "compartmentId" {
		t.Fatalf("displayName conflictsWith = %v, want [compartmentId]", got)
	}

	compartmentID := findMutabilityOverlayField(t, doc.Fields, "compartmentId")
	if !compartmentID.AST.ForceNew {
		t.Fatal("compartmentId forceNew = false, want true")
	}
	if compartmentID.Merge.FinalPolicy != mutabilityOverlayPolicyReplacementRequired {
		t.Fatalf("compartmentId final policy = %q, want %q", compartmentID.Merge.FinalPolicy, mutabilityOverlayPolicyReplacementRequired)
	}

	for _, field := range doc.Fields {
		if field.ASTFieldPath == "name" {
			t.Fatalf("docs-only field %q unexpectedly appeared in artifact", field.ASTFieldPath)
		}
	}
}

func TestGenerateMutabilityOverlayPrefersRepoAuthoredMutationSurface(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	configPath := filepath.Join(repo, "internal", "generator", "config", "services.yaml")
	writeGeneratorTestFile(t, configPath, `schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  crd-only:
    description: CRD-only groups
services:
  - service: mysql
    sdkPackage: example.com/test/sdk
    group: mysql
    packageProfile: crd-only
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - Widget
    async:
      strategy: lifecycle
      runtime: generatedruntime
    generation:
      resources:
        - kind: Widget
          formalSpec: widget
`)
	writeMutabilityOverlayWidgetFormalScaffold(t, repo)
	formalRoot := filepath.Join(repo, "formal")
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "controllers", "mysql", "widget", "diagrams", "runtime-lifecycle.yaml"), `schemaVersion: 1
surface: repo-authored-semantics
service: mysql
slug: widget
kind: Widget
archetype: generated-service-manager
states:
  - provisioning
  - active
  - updating
  - terminating
repoAuthored:
  mutation:
    mutable:
      - display_name
    forceNew:
      - compartment_id
notes:
  - Repo-authored mutation override for mutability overlay tests.
`)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "imports", "mysql", "widget.json"), `{
  "schemaVersion": 1,
  "surface": "provider-facts",
  "service": "mysql",
  "slug": "widget",
  "kind": "Widget",
  "sourceRef": "terraform-provider-oci",
  "providerResource": "oci_mysql_widget",
  "operations": {
    "create": [
      {
        "operation": "CreateWidget",
        "requestType": "CreateWidgetRequest",
        "responseType": "CreateWidgetResponse"
      }
    ],
    "get": [
      {
        "operation": "GetWidget",
        "requestType": "GetWidgetRequest",
        "responseType": "GetWidgetResponse"
      }
    ],
    "list": [
      {
        "operation": "ListWidgets",
        "requestType": "ListWidgetsRequest",
        "responseType": "ListWidgetsResponse"
      }
    ],
    "update": [
      {
        "operation": "UpdateWidget",
        "requestType": "UpdateWidgetRequest",
        "responseType": "UpdateWidgetResponse"
      }
    ],
    "delete": [
      {
        "operation": "DeleteWidget",
        "requestType": "DeleteWidgetRequest",
        "responseType": "DeleteWidgetResponse"
      }
    ]
  },
  "lifecycle": {
    "create": {
      "pending": ["PROVISIONING"],
      "target": ["ACTIVE"]
    },
    "update": {
      "pending": ["UPDATING"],
      "target": ["ACTIVE"]
    }
  },
  "mutation": {
    "mutable": ["unsupported_repo_only"],
    "forceNew": ["unsupported_force_new"],
    "conflictsWith": {}
  },
  "hooks": {
    "create": [
      {
        "helper": "tfresource.CreateResource"
      }
    ]
  },
  "deleteConfirmation": {
    "pending": ["DELETING"],
    "target": ["DELETED"]
  },
  "listLookup": {
    "datasource": "oci_widget_widgets",
    "collectionField": "widgets",
    "responseItemsField": "Items",
    "filterFields": ["compartment_id", "state"]
  },
  "boundary": {
    "providerFactsOnly": true,
    "repoAuthoredSpecPath": "controllers/mysql/widget/spec.cfg",
    "repoAuthoredLogicGapsPath": "controllers/mysql/widget/logic-gaps.md",
    "excludedSemantics": [
      "mutation-policy"
    ]
  }
}
`)
	if _, err := formal.RenderDiagrams(formal.RenderOptions{Root: formalRoot}); err != nil {
		t.Fatalf("RenderDiagrams(%q) error = %v", formalRoot, err)
	}

	target := mutabilityOverlayWidgetTarget()
	writeMutabilityOverlayFixture(t, filepath.Join(repo, filepath.FromSlash(mutabilityOverlayDocsFixtureRootRelative)), target, `
<!doctype html>
<html>
  <body>
    <h2 id="argument-reference">Argument Reference</h2>
    <ul>
      <li><code>display_name</code> - The display name of the widget. (Updatable)</li>
      <li><code>name</code> - Updating this value after creation is not supported.</li>
    </ul>
  </body>
</html>
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", configPath, err)
	}
	services, err := cfg.SelectServices("mysql", false)
	if err != nil {
		t.Fatalf("SelectServices(mysql) error = %v", err)
	}

	pipeline := New()
	pipeline.discoverer = &Discoverer{
		resolveDir: func(context.Context, string) (string, error) {
			return sampleSDKDir(t), nil
		},
	}
	pipeline.mutabilityOverlayDocsFetcher = stubMutabilityOverlayDocsFetcher{}

	outputRoot := t.TempDir()
	if _, err := pipeline.Generate(context.Background(), cfg, services, Options{
		OutputRoot:              outputRoot,
		EnableMutabilityOverlay: true,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	artifactPath := filepath.Join(outputRoot, filepath.FromSlash(mutabilityOverlayGeneratedRootRelativePath), "mysql", "widget.json")
	content := readFile(t, artifactPath)

	var doc mutabilityOverlayDocument
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", artifactPath, err)
	}

	if len(doc.Fields) != 2 {
		t.Fatalf("fields = %d, want 2", len(doc.Fields))
	}
	findMutabilityOverlayField(t, doc.Fields, "displayName")
	findMutabilityOverlayField(t, doc.Fields, "compartmentId")
	for _, field := range doc.Fields {
		if field.ASTFieldPath == "unsupportedRepoOnly" || field.ASTFieldPath == "unsupportedForceNew" {
			t.Fatalf("unexpected import-only field %q appeared in artifact", field.ASTFieldPath)
		}
	}
}

func TestGenerateMutabilityOverlayReportsFetchFailureWithContext(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	configPath := filepath.Join(repo, "internal", "generator", "config", "services.yaml")
	writeGeneratorTestFile(t, configPath, `schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  crd-only:
    description: CRD-only groups
services:
  - service: mysql
    sdkPackage: example.com/test/sdk
    group: mysql
    packageProfile: crd-only
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - Widget
    async:
      strategy: lifecycle
      runtime: generatedruntime
    generation:
      resources:
        - kind: Widget
          formalSpec: widget
`)
	writeMutabilityOverlayWidgetFormalScaffold(t, repo)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", configPath, err)
	}
	services, err := cfg.SelectServices("mysql", false)
	if err != nil {
		t.Fatalf("SelectServices(mysql) error = %v", err)
	}

	target := mutabilityOverlayWidgetTarget()
	pipeline := New()
	pipeline.discoverer = &Discoverer{
		resolveDir: func(context.Context, string) (string, error) {
			return sampleSDKDir(t), nil
		},
	}
	pipeline.mutabilityOverlayDocsFetcher = stubMutabilityOverlayDocsFetcher{
		errors: map[string]error{
			target.RegistryURL: errors.New("boom"),
		},
	}

	_, err = pipeline.Generate(context.Background(), cfg, services, Options{
		OutputRoot:              t.TempDir(),
		EnableMutabilityOverlay: true,
	})
	if err == nil {
		t.Fatal("Generate() unexpectedly succeeded")
	}
	message := err.Error()
	for _, want := range []string{
		`service "mysql"`,
		`kind "Widget"`,
		`providerResource="oci_mysql_widget"`,
		`availabilityFailure`,
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("Generate() error = %v, want substring %q", err, want)
		}
	}
}

func writeMutabilityOverlayWidgetFormalScaffold(t *testing.T, repoRoot string) {
	t.Helper()

	writeGeneratorFormalScaffold(t, repoRoot, "mysql", "widget", "Widget")
	formalRoot := filepath.Join(repoRoot, "formal")
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "sources.lock"), `{
  "schemaVersion": 1,
  "sources": [
    {
      "name": "terraform-provider-oci",
      "surface": "provider-facts",
      "status": "pinned",
      "path": "github.com/oracle/terraform-provider-oci",
      "revision": "test-provider-revision",
      "notes": [
        "Pinned for mutability overlay integration tests."
      ]
    }
  ]
}
`)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "imports", "mysql", "widget.json"), `{
  "schemaVersion": 1,
  "surface": "provider-facts",
  "service": "mysql",
  "slug": "widget",
  "kind": "Widget",
  "sourceRef": "terraform-provider-oci",
  "providerResource": "oci_mysql_widget",
  "operations": {
    "create": [
      {
        "operation": "CreateWidget",
        "requestType": "CreateWidgetRequest",
        "responseType": "CreateWidgetResponse"
      }
    ],
    "get": [
      {
        "operation": "GetWidget",
        "requestType": "GetWidgetRequest",
        "responseType": "GetWidgetResponse"
      }
    ],
    "list": [
      {
        "operation": "ListWidgets",
        "requestType": "ListWidgetsRequest",
        "responseType": "ListWidgetsResponse"
      }
    ],
    "update": [
      {
        "operation": "UpdateWidget",
        "requestType": "UpdateWidgetRequest",
        "responseType": "UpdateWidgetResponse"
      }
    ],
    "delete": [
      {
        "operation": "DeleteWidget",
        "requestType": "DeleteWidgetRequest",
        "responseType": "DeleteWidgetResponse"
      }
    ]
  },
  "lifecycle": {
    "create": {
      "pending": ["PROVISIONING"],
      "target": ["ACTIVE"]
    },
    "update": {
      "pending": ["UPDATING"],
      "target": ["ACTIVE"]
    }
  },
  "mutation": {
    "mutable": ["display_name"],
    "forceNew": ["compartment_id"],
    "conflictsWith": {
      "display_name": ["compartment_id"]
    }
  },
  "hooks": {
    "create": [
      {
        "helper": "tfresource.CreateResource"
      }
    ]
  },
  "deleteConfirmation": {
    "pending": ["DELETING"],
    "target": ["DELETED"]
  },
  "listLookup": {
    "datasource": "oci_widget_widgets",
    "collectionField": "widgets",
    "responseItemsField": "Items",
    "filterFields": ["compartment_id", "state"]
  },
  "boundary": {
    "providerFactsOnly": true,
    "repoAuthoredSpecPath": "controllers/mysql/widget/spec.cfg",
    "repoAuthoredLogicGapsPath": "controllers/mysql/widget/logic-gaps.md",
    "excludedSemantics": [
      "bind-versus-create",
      "secret-output",
      "delete-confirmation"
    ]
  }
}
`)
	if _, err := formal.RenderDiagrams(formal.RenderOptions{Root: formalRoot}); err != nil {
		t.Fatalf("RenderDiagrams(%q) error = %v", formalRoot, err)
	}
}

func writeMutabilityOverlayFixture(
	t *testing.T,
	root string,
	target mutabilityOverlayRegistryPageTarget,
	body string,
) {
	t.Helper()

	input, err := newMutabilityOverlayDocsInput(
		target,
		mutabilityOverlayDocsInputSourceLive,
		target.RegistryURL,
		"",
		"text/html; charset=utf-8",
		body,
	)
	if err != nil {
		t.Fatalf("newMutabilityOverlayDocsInput() error = %v", err)
	}
	if _, err := writeMutabilityOverlayDocsFixtures(root, []mutabilityOverlayDocsInput{input}); err != nil {
		t.Fatalf("writeMutabilityOverlayDocsFixtures(%q) error = %v", root, err)
	}
}

func mutabilityOverlayWidgetTarget() mutabilityOverlayRegistryPageTarget {
	return mutabilityOverlayRegistryPageTarget{
		Service:              "mysql",
		Kind:                 "Widget",
		FormalSlug:           "widget",
		ProviderResource:     "oci_mysql_widget",
		TerraformDocsVersion: defaultMutabilityOverlayTerraformDocsVersion,
		RegistryPath:         "providers/oracle/oci/7.22.0/docs/resources/mysql_widget",
		RegistryURL:          "https://registry.terraform.io/providers/oracle/oci/7.22.0/docs/resources/mysql_widget",
	}
}

func findMutabilityOverlayField(t *testing.T, fields []mutabilityOverlayField, astFieldPath string) mutabilityOverlayField {
	t.Helper()
	for _, field := range fields {
		if field.ASTFieldPath == astFieldPath {
			return field
		}
	}
	t.Fatalf("mutability overlay field %q not found", astFieldPath)
	return mutabilityOverlayField{}
}

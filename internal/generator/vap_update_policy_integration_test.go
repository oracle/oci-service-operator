package generator

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"

	"github.com/oracle/oci-service-operator/internal/formal"
)

func TestGenerateWritesVAPUpdatePolicyArtifactFromMergedOverlay(t *testing.T) {
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
    generation:
      resources:
        - kind: Widget
          formalSpec: widget
`)
	writeMutabilityOverlayWidgetFormalScaffold(t, repo)
	writeGeneratorTestFile(t, filepath.Join(repo, "formal", "imports", "mysql", "widget.json"), `{
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
    "mutable": ["display_name", "name"],
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
	if _, err := formal.RenderDiagrams(formal.RenderOptions{Root: filepath.Join(repo, "formal")}); err != nil {
		t.Fatalf("RenderDiagrams(%q) error = %v", filepath.Join(repo, "formal"), err)
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

	artifactPath := filepath.Join(outputRoot, filepath.FromSlash(vapUpdatePolicyGeneratedRootRelativePath), "mysql", "widget.json")
	content := readFile(t, artifactPath)

	var doc vapUpdatePolicyDocument
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", artifactPath, err)
	}

	if doc.Target.APIVersion != "mysql.oracle.com/v1beta1" {
		t.Fatalf("apiVersion = %q, want mysql.oracle.com/v1beta1", doc.Target.APIVersion)
	}
	if doc.Target.Kind != "Widget" {
		t.Fatalf("kind = %q, want Widget", doc.Target.Kind)
	}
	if !slices.Equal(doc.Update.AllowInPlacePaths, []string{"displayName"}) {
		t.Fatalf("allowInPlacePaths = %v, want [displayName]", doc.Update.AllowInPlacePaths)
	}
	if slices.Contains(doc.Update.AllowInPlacePaths, "name") {
		t.Fatalf("allowInPlacePaths = %v, name must be excluded after docs deny", doc.Update.AllowInPlacePaths)
	}

	nameRule := findVAPUpdatePolicyRule(t, doc.Update.DenyRules, "name")
	if nameRule.Decision != mutabilityOverlayPolicyDenyInPlaceUpdate {
		t.Fatalf("name decision = %q, want %q", nameRule.Decision, mutabilityOverlayPolicyDenyInPlaceUpdate)
	}
	if nameRule.MergeCase != mutabilityOverlayMergeCaseDocsDeniedCandidate {
		t.Fatalf("name mergeCase = %q, want %q", nameRule.MergeCase, mutabilityOverlayMergeCaseDocsDeniedCandidate)
	}
	if nameRule.DocsEvidenceState != mutabilityOverlayDocsStateDeniedUpdatable {
		t.Fatalf("name docsEvidenceState = %q, want %q", nameRule.DocsEvidenceState, mutabilityOverlayDocsStateDeniedUpdatable)
	}

	compartmentIDRule := findVAPUpdatePolicyRule(t, doc.Update.DenyRules, "compartmentId")
	if compartmentIDRule.Decision != mutabilityOverlayPolicyReplacementRequired {
		t.Fatalf("compartmentId decision = %q, want %q", compartmentIDRule.Decision, mutabilityOverlayPolicyReplacementRequired)
	}
}

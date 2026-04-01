package formal

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const testControllerManifest = "service\tslug\tkind\tstage\tsurface\timport\tspec\tlogic_gaps\tdiagram_dir\n" +
	"template\ttemplate\tTemplate\tscaffold\trepo-authored-semantics\timports/template/template.json\tcontrollers/template/spec.cfg\tcontrollers/template/logic-gaps.md\tcontrollers/template/diagrams\n"

const testSpec = `# formal controller binding schema v1
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

const testLogicGaps = `---
schemaVersion: 1
surface: repo-authored-semantics
service: template
slug: template
gaps: []
---

# Logic Gaps
`

const testDiagram = `schemaVersion: 1
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
  - Scaffold metadata only.
`

const testImport = `{
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
        "operation": "ListTemplates",
        "requestType": "ListTemplatesRequest",
        "responseType": "ListTemplatesResponse"
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
    "conflictsWith": {}
  },
  "hooks": {
    "create": [
      {
        "helper": "tfresource.CreateResource"
      },
      {
        "helper": "tfresource.WaitForWorkRequestWithErrorHandling",
        "entityType": "template",
        "action": "CREATED"
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
    "pending": ["DELETING"],
    "target": ["DELETED"]
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

const baseReconcilerContract = `------------------------------ MODULE BaseReconcilerContract ------------------------------
EXTENDS ControllerLifecycleSpec

VARIABLES deletionRequested, deleteConfirmed, finalizerPresent, lifecycleCondition, shouldRequeue, requestedAtStamped

FinalizerRetention == deletionRequested /\ ~deleteConfirmed => finalizerPresent
RetryableConditionsRequeue == lifecycleCondition \in {"Provisioning", "Updating", "Terminating"} => shouldRequeue
StatusProjectionStampsRequestedAt == requestedAtStamped

=============================================================================
`

const controllerLifecycleContract = `------------------------------ MODULE ControllerLifecycleSpec ------------------------------

RetryableConditions == {"Provisioning", "Updating", "Terminating"}
ShouldRequeue(condition) == condition \in RetryableConditions

=============================================================================
`

const serviceManagerContract = `------------------------------ MODULE OSOKServiceManagerContract ------------------------------
EXTENDS Naturals

ResponseShape(response) == response \in [IsSuccessful : BOOLEAN, ShouldRequeue : BOOLEAN, RequeueDuration : Nat]

=============================================================================
`

const secretSideEffectsContract = `------------------------------ MODULE SecretSideEffectsContract ------------------------------

SecretWritePolicies == {"none", "ready-only"}
SecretWritesAllowed(policy, condition) == IF policy = "none" THEN FALSE ELSE condition = "Active"

=============================================================================
`

func TestVerifyCheckedInScaffold(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "formal"))
	if _, err := Verify(root); err != nil {
		t.Fatalf("Verify(%q) returned error: %v", root, err)
	}
}

func TestVerifyRejectsUnsupportedLogicGapCategory(t *testing.T) {
	root := writeScaffold(t)
	writeFile(t, filepath.Join(root, "controllers", "template", "logic-gaps.md"), `---
schemaVersion: 1
surface: repo-authored-semantics
service: template
slug: template
gaps:
  - category: unknown-gap
    status: open
    stopCondition: Replace with a supported category.
---

# Logic Gaps
`)

	_, err := Verify(root)
	if err == nil || !strings.Contains(err.Error(), `unsupported logic gap category "unknown-gap"`) {
		t.Fatalf("Verify(%q) error = %v, want unsupported logic gap category failure", root, err)
	}
}

func TestVerifyRejectsBindingMismatch(t *testing.T) {
	root := writeScaffold(t)
	writeFile(t, filepath.Join(root, "controllers", "template", "spec.cfg"), strings.Replace(testSpec, "imports/template/template.json", "imports/template/other.json", 1))

	_, err := Verify(root)
	if err == nil || !strings.Contains(err.Error(), `import="imports/template/other.json" does not match manifest import "imports/template/template.json"`) {
		t.Fatalf("Verify(%q) error = %v, want binding mismatch failure", root, err)
	}
}

func TestVerifyRejectsImportSurfaceMismatch(t *testing.T) {
	root := writeScaffold(t)
	writeFile(t, filepath.Join(root, "imports", "template", "template.json"), strings.Replace(testImport, `"provider-facts"`, `"repo-authored-semantics"`, 1))

	_, err := Verify(root)
	if err == nil || !strings.Contains(err.Error(), `surface="repo-authored-semantics" is not allowed`) {
		t.Fatalf("Verify(%q) error = %v, want import surface failure", root, err)
	}
}

func TestVerifyRejectsOperationBindingTypeMismatch(t *testing.T) {
	root := writeScaffold(t)
	mismatched := strings.NewReplacer(
		`"requestType": "UpdateTemplateRequest"`,
		`"requestType": "DeleteTemplateRequest"`,
		`"responseType": "UpdateTemplateResponse"`,
		`"responseType": "DeleteTemplateResponse"`,
	).Replace(testImport)
	writeFile(t, filepath.Join(root, "imports", "template", "template.json"), mismatched)

	_, err := Verify(root)
	if err == nil || !strings.Contains(err.Error(), `UpdateTemplate binding requestType="DeleteTemplateRequest", want "UpdateTemplateRequest"`) || !strings.Contains(err.Error(), `UpdateTemplate binding responseType="DeleteTemplateResponse", want "UpdateTemplateResponse"`) {
		t.Fatalf("Verify(%q) error = %v, want operation binding type mismatch failure", root, err)
	}
}

func TestVerifyRejectsMissingRenderedDiagramArtifacts(t *testing.T) {
	root := writeScaffold(t)
	if err := os.Remove(filepath.Join(root, "controllers", "template", "diagrams", "sequence.svg")); err != nil {
		t.Fatalf("Remove(sequence.svg) failed: %v", err)
	}

	_, err := Verify(root)
	if err == nil || !strings.Contains(err.Error(), `missing required diagram artifact "sequence.svg"`) {
		t.Fatalf("Verify(%q) error = %v, want missing sequence.svg failure", root, err)
	}
}

func TestVerifyRejectsStaleRenderedDiagramArtifacts(t *testing.T) {
	root := writeScaffold(t)
	writeFile(t, filepath.Join(root, "controllers", "template", "diagrams", "activity.puml"), "@startuml\nstale\n@enduml\n")

	_, err := Verify(root)
	if err == nil || !strings.Contains(err.Error(), "stale rendered artifact") {
		t.Fatalf("Verify(%q) error = %v, want stale rendered artifact failure", root, err)
	}
}

func TestVerifyRejectsMissingSharedDiagramArtifacts(t *testing.T) {
	root := writeScaffold(t)
	if err := os.Remove(filepath.Join(root, "shared", "diagrams", "shared-reconcile-activity.svg")); err != nil {
		t.Fatalf("Remove(shared-reconcile-activity.svg) failed: %v", err)
	}

	_, err := Verify(root)
	if err == nil || !strings.Contains(err.Error(), "shared/diagrams/shared-reconcile-activity.svg") {
		t.Fatalf("Verify(%q) error = %v, want missing shared diagram failure", root, err)
	}
}

func TestVerifyRejectsStaleManifestOwnedArtifacts(t *testing.T) {
	root := writeScaffold(t)
	writeFile(t, filepath.Join(root, "controllers", "identity", "user", "spec.cfg"), strings.NewReplacer(
		"service = template", "service = identity",
		"slug = template", "slug = user",
		"kind = Template", "kind = User",
		"import = imports/template/template.json", "import = imports/identity/user.json",
	).Replace(testSpec))
	writeFile(t, filepath.Join(root, "controllers", "identity", "user", "logic-gaps.md"), strings.NewReplacer(
		"service: template", "service: identity",
		"slug: template", "slug: user",
	).Replace(testLogicGaps))
	writeFile(t, filepath.Join(root, "controllers", "identity", "user", "diagrams", "runtime-lifecycle.yaml"), strings.NewReplacer(
		"service: template", "service: identity",
		"slug: template", "slug: user",
		"kind: Template", "kind: User",
	).Replace(testDiagram))
	writeFile(t, filepath.Join(root, "imports", "identity", "user.json"), strings.NewReplacer(
		`"service": "template"`, `"service": "identity"`,
		`"slug": "template"`, `"slug": "user"`,
		`"kind": "Template"`, `"kind": "User"`,
		`"repoAuthoredSpecPath": "controllers/template/spec.cfg"`, `"repoAuthoredSpecPath": "controllers/identity/user/spec.cfg"`,
		`"repoAuthoredLogicGapsPath": "controllers/template/logic-gaps.md"`, `"repoAuthoredLogicGapsPath": "controllers/identity/user/logic-gaps.md"`,
	).Replace(testImport))

	_, err := Verify(root)
	if err == nil ||
		!strings.Contains(err.Error(), "controllers/identity/user: stale controller artifacts are not referenced by controller_manifest.tsv") ||
		!strings.Contains(err.Error(), "imports/identity/user.json: stale import artifact is not referenced by controller_manifest.tsv") {
		t.Fatalf("Verify(%q) error = %v, want stale manifest-owned artifact failure", root, err)
	}
}

func TestRenderDiagramsIncludesProviderDrivenDetail(t *testing.T) {
	root := writeScaffold(t)

	activity, err := os.ReadFile(filepath.Join(root, "controllers", "template", "diagrams", "activity.puml"))
	if err != nil {
		t.Fatalf("ReadFile(activity.puml) error = %v", err)
	}
	text := string(activity)
	for _, needle := range []string{
		"shared/diagrams/shared-reconcile-activity.svg",
		"CreateTemplate",
		"UpdateTemplate",
		"retain-until-confirmed-delete",
		"tfresource.WaitForWorkRequestWithErrorHandling",
		"action=CREATED",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("activity.puml missing %q:\n%s", needle, text)
		}
	}

	shared, err := os.ReadFile(filepath.Join(root, "shared", "diagrams", "shared-legend.svg"))
	if err != nil {
		t.Fatalf("ReadFile(shared-legend.svg) error = %v", err)
	}
	sharedText := string(shared)
	for _, needle := range []string{
		"Shared Diagram Legend",
		"Generated Service",
		"Manager Batch",
		"Legacy Adapter Batch",
	} {
		if !strings.Contains(sharedText, needle) {
			t.Fatalf("shared-legend.svg missing %q:\n%s", needle, sharedText)
		}
	}
}

func writeScaffold(t *testing.T) string {
	t.Helper()
	requirePlantUML(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "controller_manifest.tsv"), testControllerManifest)
	writeFile(t, filepath.Join(root, "sources.lock"), testSourcesLock)
	writeFile(t, filepath.Join(root, "shared", "BaseReconcilerContract.tla"), baseReconcilerContract)
	writeFile(t, filepath.Join(root, "shared", "ControllerLifecycleSpec.tla"), controllerLifecycleContract)
	writeFile(t, filepath.Join(root, "shared", "OSOKServiceManagerContract.tla"), serviceManagerContract)
	writeFile(t, filepath.Join(root, "shared", "SecretSideEffectsContract.tla"), secretSideEffectsContract)
	writeControllerDiagramStrategyFixtures(t, root)
	writeFile(t, filepath.Join(root, "controllers", "template", "spec.cfg"), testSpec)
	writeFile(t, filepath.Join(root, "controllers", "template", "logic-gaps.md"), testLogicGaps)
	writeFile(t, filepath.Join(root, "controllers", "template", "diagrams", "runtime-lifecycle.yaml"), testDiagram)
	writeFile(t, filepath.Join(root, "imports", "template", "template.json"), testImport)
	if _, err := RenderDiagrams(RenderOptions{Root: root}); err != nil {
		t.Fatalf("RenderDiagrams(%q) error = %v", root, err)
	}
	return root
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) failed: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", path, err)
	}
}

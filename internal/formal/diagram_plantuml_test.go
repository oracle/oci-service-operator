package formal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderDiagramsEmbedsPlantUMLMetadataInActivitySVG(t *testing.T) {
	root := writeScaffold(t)

	data, err := os.ReadFile(filepath.Join(root, "controllers", "template", "diagrams", "activity.svg"))
	if err != nil {
		t.Fatalf("ReadFile(activity.svg) error = %v", err)
	}

	text := string(data)
	for _, needle := range []string{
		"<?plantuml ",
		"<?plantuml-src ",
		"Tracked or explicit OCI identity present?",
		"Force-new or conflicting drift detected?",
		"CreateTemplate",
		"display_name",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("activity.svg missing %q:\n%s", needle, text)
		}
	}
}

func TestRenderDiagramsEmbedsPlantUMLMetadataInSequenceSVG(t *testing.T) {
	root := writeScaffold(t)

	data, err := os.ReadFile(filepath.Join(root, "controllers", "template", "diagrams", "sequence.svg"))
	if err != nil {
		t.Fatalf("ReadFile(sequence.svg) error = %v", err)
	}

	text := string(data)
	for _, needle := range []string{
		`data-diagram-type="SEQUENCE"`,
		"Kubernetes",
		"BaseReconciler",
		"OSOKServiceManager",
		"CreateTemplate",
		"display_name",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("sequence.svg missing %q:\n%s", needle, text)
		}
	}
}

func TestRenderDiagramsEmbedsPlantUMLMetadataInStateMachineSVG(t *testing.T) {
	root := writeScaffold(t)

	data, err := os.ReadFile(filepath.Join(root, "controllers", "template", "diagrams", "state-machine.svg"))
	if err != nil {
		t.Fatalf("ReadFile(state-machine.svg) error = %v", err)
	}

	text := string(data)
	for _, needle := range []string{
		`data-diagram-type="STATE"`,
		"Observe",
		"Ready",
		"DeletePending",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("state-machine.svg missing %q:\n%s", needle, text)
		}
	}
}

func TestRenderDiagramsWriteFieldAwarePlantUMLSources(t *testing.T) {
	root := writeScaffold(t)

	type check struct {
		path    string
		needles []string
	}
	for _, tc := range []check{
		{
			path: filepath.Join(root, "controllers", "template", "diagrams", "activity.puml"),
			needles: []string{
				`"Force-new or conflicting drift detected?"`,
				`Reject drift before OCI mutation for force-new`,
				`Apply UpdateTemplate only for mutable fields`,
			},
		},
		{
			path: filepath.Join(root, "controllers", "template", "diagrams", "sequence.puml"),
			needles: []string{
				`group Lookup and bind`,
				`Supported update surface: display_name`,
				`Reject before mutate: force-new fields`,
			},
		},
		{
			path: filepath.Join(root, "controllers", "template", "diagrams", "state-machine.puml"),
			needles: []string{
				`state "Observe" as observe`,
				`state "ApplyUpdate" as apply_update`,
				`state "DeletePending" as delete_pending`,
			},
		},
	} {
		data, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", tc.path, err)
		}
		text := string(data)
		for _, needle := range tc.needles {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s missing %q:\n%s", tc.path, needle, text)
			}
		}
	}
}

func TestRenderDiagramsUseRepoAuthoredOverridesWhenPresent(t *testing.T) {
	root := writeScaffold(t)

	writeFile(t, filepath.Join(root, "controllers", "template", "diagrams", "runtime-lifecycle.yaml"), `schemaVersion: 1
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
repoAuthored:
  providerLifecycle:
    createPending:
      - BOOTING
    updatePending:
      - PATCHING
    deletePending:
      - REMOVING
  listLookup:
    filters:
      - compartment_id
      - display_name
      - state=ALL
    matchRule: exact-name unique match only
  mutation:
    mutable:
      - repo_display_name
    forceNew:
      - repo_compartment_id
    createOnly:
      - seed_payload
notes:
  - Repo-authored overrides should replace imported mutation and pending-state summaries.
`)
	if _, err := RenderDiagrams(RenderOptions{Root: root}); err != nil {
		t.Fatalf("RenderDiagrams(%q) error = %v", root, err)
	}

	for _, tc := range []struct {
		path    string
		needles []string
	}{
		{
			path: filepath.Join(root, "controllers", "template", "diagrams", "activity.puml"),
			needles: []string{
				"repo-authored mutation surface",
				"Request requeue while OCI remains in BOOTING",
				"PATCHING, REMOVING",
				"Apply UpdateTemplate only for mutable fields",
				"repo_display_name",
			},
		},
		{
			path: filepath.Join(root, "controllers", "template", "diagrams", "sequence.puml"),
			needles: []string{
				"Supported update surface:",
				"repo_display_name",
				"Reject before mutate: force-new fields",
				"repo_compartment_id",
				"Create-only fields: seed_payload",
				"Lookup matching: exact-name unique match",
				"only",
			},
		},
		{
			path: filepath.Join(root, "controllers", "template", "diagrams", "state-machine.puml"),
			needles: []string{
				"provider states: create BOOTING; update",
				"PATCHING; delete REMOVING",
				"create-only fields: seed_payload",
				"list lookup: oci_template_templates filters",
				"display_name, state=ALL;",
				"exact-name unique match",
				"only",
			},
		},
	} {
		data, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", tc.path, err)
		}
		text := string(data)
		for _, needle := range tc.needles {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s missing %q:\n%s", tc.path, needle, text)
			}
		}
	}
}

func TestRenderDiagramsWritesSharedPlantUMLArtifacts(t *testing.T) {
	root := writeScaffold(t)

	for _, path := range []string{
		filepath.Join(root, "shared", "diagrams", "shared-reconcile-activity.puml"),
		filepath.Join(root, "shared", "diagrams", "shared-resolution-sequence.puml"),
		filepath.Join(root, "shared", "diagrams", "shared-delete-sequence.puml"),
		filepath.Join(root, "shared", "diagrams", "shared-controller-state-machine.puml"),
		filepath.Join(root, "shared", "diagrams", "shared-legend.puml"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		if !strings.Contains(string(data), "@startuml") {
			t.Fatalf("%q missing @startuml:\n%s", path, string(data))
		}
	}

	sharedSVG, err := os.ReadFile(filepath.Join(root, "shared", "diagrams", "shared-reconcile-activity.svg"))
	if err != nil {
		t.Fatalf("ReadFile(shared-reconcile-activity.svg) error = %v", err)
	}
	sharedText := string(sharedSVG)
	for _, needle := range []string{
		"<?plantuml ",
		"Shared Reconcile Activity",
		"Step 1.",
		"Shared Contract",
	} {
		if !strings.Contains(sharedText, needle) {
			t.Fatalf("shared-reconcile-activity.svg missing %q:\n%s", needle, sharedText)
		}
	}
}

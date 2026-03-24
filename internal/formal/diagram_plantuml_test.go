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
		"Resource exists?",
		"CreateTemplate",
		"UpdateTemplate",
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
		"provisioning",
		"active",
		"terminating",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("state-machine.svg missing %q:\n%s", needle, text)
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

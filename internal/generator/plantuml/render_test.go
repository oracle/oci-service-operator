package plantuml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderArtifactsAndValidateRenderedArtifacts(t *testing.T) {
	requirePlantUML(t)

	root := t.TempDir()
	sourcePath := filepath.Join(root, "diagram.puml")
	if err := os.WriteFile(sourcePath, []byte("@startuml\nAlice -> Bob: hi\n@enduml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", sourcePath, err)
	}

	if err := RenderArtifacts(root, []Artifact{{
		SourcePath:   "diagram.puml",
		RenderedPath: "diagram.svg",
	}}); err != nil {
		t.Fatalf("RenderArtifacts() error = %v", err)
	}

	svgPath := filepath.Join(root, "diagram.svg")
	data, err := os.ReadFile(svgPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", svgPath, err)
	}
	if !strings.Contains(string(data), "<?plantuml-src ") {
		t.Fatalf("diagram.svg missing embedded plantuml metadata:\n%s", string(data))
	}

	problems := ValidateRenderedArtifacts(root, []Pair{{
		SourcePath:   "diagram.puml",
		RenderedPath: "diagram.svg",
	}})
	if len(problems) > 0 {
		t.Fatalf("ValidateRenderedArtifacts() problems = %v, want none", problems)
	}
}

func TestValidateRenderedArtifactsDetectsStaleSVG(t *testing.T) {
	requirePlantUML(t)

	root := t.TempDir()
	sourcePath := filepath.Join(root, "diagram.puml")
	if err := os.WriteFile(sourcePath, []byte("@startuml\nAlice -> Bob: hi\n@enduml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", sourcePath, err)
	}

	if err := RenderArtifacts(root, []Artifact{{
		SourcePath:   "diagram.puml",
		RenderedPath: "diagram.svg",
	}}); err != nil {
		t.Fatalf("RenderArtifacts() error = %v", err)
	}

	if err := os.WriteFile(sourcePath, []byte("@startuml\nAlice -> Bob: bye\n@enduml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", sourcePath, err)
	}

	problems := ValidateRenderedArtifacts(root, []Pair{{
		SourcePath:   "diagram.puml",
		RenderedPath: "diagram.svg",
	}})
	if len(problems) != 1 || !strings.Contains(problems[0], "stale rendered artifact") {
		t.Fatalf("ValidateRenderedArtifacts() problems = %v, want stale rendered artifact failure", problems)
	}
}

func requirePlantUML(t *testing.T) {
	t.Helper()
	if _, err := Binary(); err != nil {
		t.Skip(err.Error())
	}
}

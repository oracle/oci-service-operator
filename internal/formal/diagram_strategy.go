package formal

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	controllerDiagramTemplateDir = "controller_diagrams"
	sharedDiagramDir             = "shared/diagrams"
	sharedReconcileActivityPath  = "shared/diagrams/shared-reconcile-activity.svg"
	sharedResolutionSequencePath = "shared/diagrams/shared-resolution-sequence.svg"
	sharedDeleteSequencePath     = "shared/diagrams/shared-delete-sequence.svg"
	sharedControllerStatePath    = "shared/diagrams/shared-controller-state-machine.svg"
	sharedLegendPath             = "shared/diagrams/shared-legend.svg"
)

var requiredControllerDiagramTemplateFiles = []string{
	"activity.yaml",
	"sequence.yaml",
	"state-machine.yaml",
	"shared.yaml",
	"legend.yaml",
}

var requiredSharedDiagramFiles = []string{
	sharedReconcileActivityPath,
	sharedResolutionSequencePath,
	sharedDeleteSequencePath,
	sharedControllerStatePath,
	sharedLegendPath,
}

type diagramStrategy struct {
	Activity     controllerDiagramTemplate
	Sequence     controllerDiagramTemplate
	StateMachine controllerDiagramTemplate
	Shared       sharedDiagramSpecFile
	Legend       legendDiagramSpec

	sharedByPath map[string]sharedDiagramSpec
}

type controllerDiagramTemplate struct {
	SchemaVersion       int               `yaml:"schemaVersion"`
	Family              string            `yaml:"family"`
	Title               string            `yaml:"title"`
	SharedDiagram       string            `yaml:"sharedDiagram"`
	SharedDeleteDiagram string            `yaml:"sharedDeleteDiagram,omitempty"`
	Summary             []string          `yaml:"summary"`
	Participants        []string          `yaml:"participants,omitempty"`
	BaseStates          []string          `yaml:"baseStates,omitempty"`
	ArchetypeBatches    map[string]string `yaml:"archetypeBatches"`
}

type sharedDiagramSpecFile struct {
	SchemaVersion int                 `yaml:"schemaVersion"`
	Diagrams      []sharedDiagramSpec `yaml:"diagrams"`
}

type sharedDiagramSpec struct {
	File     string   `yaml:"file"`
	Title    string   `yaml:"title"`
	Subtitle string   `yaml:"subtitle"`
	Lines    []string `yaml:"lines"`
}

type legendDiagramSpec struct {
	SchemaVersion    int                             `yaml:"schemaVersion"`
	Title            string                          `yaml:"title"`
	Subtitle         string                          `yaml:"subtitle"`
	Palette          map[string]legendPaletteEntry   `yaml:"palette"`
	ArchetypeBatches map[string]legendArchetypeEntry `yaml:"archetypeBatches"`
}

type legendPaletteEntry struct {
	Color       string `yaml:"color"`
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
}

type legendArchetypeEntry struct {
	Color       string `yaml:"color"`
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
}

type sharedDiagramArtifact struct {
	SourcePath   string
	RenderedPath string
	SourceData   []byte
}

func loadDiagramStrategy(root string) (diagramStrategy, error) {
	strategy := diagramStrategy{}

	type familyFile struct {
		name string
		dest *controllerDiagramTemplate
	}
	for _, item := range []familyFile{
		{name: "activity.yaml", dest: &strategy.Activity},
		{name: "sequence.yaml", dest: &strategy.Sequence},
		{name: "state-machine.yaml", dest: &strategy.StateMachine},
	} {
		loaded, err := decodeYAMLFile[controllerDiagramTemplate](filepath.Join(root, controllerDiagramTemplateDir, item.name))
		if err != nil {
			return strategy, fmt.Errorf("%s: %w", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, item.name)), err)
		}
		*item.dest = loaded
	}

	sharedSpec, err := decodeYAMLFile[sharedDiagramSpecFile](filepath.Join(root, controllerDiagramTemplateDir, "shared.yaml"))
	if err != nil {
		return strategy, fmt.Errorf("%s: %w", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "shared.yaml")), err)
	}
	strategy.Shared = sharedSpec

	legendSpec, err := decodeYAMLFile[legendDiagramSpec](filepath.Join(root, controllerDiagramTemplateDir, "legend.yaml"))
	if err != nil {
		return strategy, fmt.Errorf("%s: %w", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "legend.yaml")), err)
	}
	strategy.Legend = legendSpec

	strategy.sharedByPath = make(map[string]sharedDiagramSpec, len(strategy.Shared.Diagrams))
	for _, spec := range strategy.Shared.Diagrams {
		strategy.sharedByPath[strings.TrimSpace(spec.File)] = spec
	}

	if err := strategy.validate(); err != nil {
		return diagramStrategy{}, err
	}
	return strategy, nil
}

func (s diagramStrategy) validate() error {
	var problems []string

	validateFamily := func(label string, template controllerDiagramTemplate, expectedFile string, expectedShared string) {
		if template.SchemaVersion != currentSchemaVersion {
			problems = append(problems, fmt.Sprintf("%s: schemaVersion=%d, want %d", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, expectedFile)), template.SchemaVersion, currentSchemaVersion))
		}
		if strings.TrimSpace(template.Family) != label {
			problems = append(problems, fmt.Sprintf("%s: family=%q, want %q", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, expectedFile)), template.Family, label))
		}
		if strings.TrimSpace(template.Title) == "" {
			problems = append(problems, fmt.Sprintf("%s: title must not be empty", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, expectedFile))))
		}
		if strings.TrimSpace(template.SharedDiagram) != expectedShared {
			problems = append(problems, fmt.Sprintf("%s: sharedDiagram=%q, want %q", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, expectedFile)), template.SharedDiagram, expectedShared))
		}
		if len(template.Summary) == 0 {
			problems = append(problems, fmt.Sprintf("%s: summary must contain at least one line", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, expectedFile))))
		}
		for archetype := range allowedDiagramArchetypes {
			if strings.TrimSpace(template.ArchetypeBatches[archetype]) == "" {
				problems = append(problems, fmt.Sprintf("%s: archetypeBatches must define %q", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, expectedFile)), archetype))
			}
		}
	}

	validateFamily("activity", s.Activity, "activity.yaml", sharedReconcileActivityPath)
	validateFamily("sequence", s.Sequence, "sequence.yaml", sharedResolutionSequencePath)
	validateFamily("state-machine", s.StateMachine, "state-machine.yaml", sharedControllerStatePath)

	if strings.TrimSpace(s.Sequence.SharedDeleteDiagram) != sharedDeleteSequencePath {
		problems = append(problems, fmt.Sprintf("%s: sharedDeleteDiagram=%q, want %q", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "sequence.yaml")), s.Sequence.SharedDeleteDiagram, sharedDeleteSequencePath))
	}
	if len(s.Sequence.Participants) == 0 {
		problems = append(problems, fmt.Sprintf("%s: participants must contain at least one entry", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "sequence.yaml"))))
	}
	if len(s.StateMachine.BaseStates) == 0 {
		problems = append(problems, fmt.Sprintf("%s: baseStates must contain at least one controller phase", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "state-machine.yaml"))))
	}

	if s.Shared.SchemaVersion != currentSchemaVersion {
		problems = append(problems, fmt.Sprintf("%s: schemaVersion=%d, want %d", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "shared.yaml")), s.Shared.SchemaVersion, currentSchemaVersion))
	}
	for _, required := range requiredSharedDiagramFiles[:4] {
		spec, ok := s.sharedByPath[required]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: missing shared diagram spec %q", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "shared.yaml")), required))
			continue
		}
		if strings.TrimSpace(spec.Title) == "" || len(spec.Lines) == 0 {
			problems = append(problems, fmt.Sprintf("%s: spec %q requires a title and at least one line", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "shared.yaml")), required))
		}
	}

	if s.Legend.SchemaVersion != currentSchemaVersion {
		problems = append(problems, fmt.Sprintf("%s: schemaVersion=%d, want %d", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "legend.yaml")), s.Legend.SchemaVersion, currentSchemaVersion))
	}
	if strings.TrimSpace(s.Legend.Title) == "" {
		problems = append(problems, fmt.Sprintf("%s: title must not be empty", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "legend.yaml"))))
	}
	if len(s.Legend.Palette) == 0 {
		problems = append(problems, fmt.Sprintf("%s: palette must contain at least one entry", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "legend.yaml"))))
	}
	for archetype := range allowedDiagramArchetypes {
		entry, ok := s.Legend.ArchetypeBatches[archetype]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: archetypeBatches must define %q", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "legend.yaml")), archetype))
			continue
		}
		if strings.TrimSpace(entry.Color) == "" || strings.TrimSpace(entry.Label) == "" || strings.TrimSpace(entry.Description) == "" {
			problems = append(problems, fmt.Sprintf("%s: archetype %q requires color, label, and description", filepath.ToSlash(path.Join(controllerDiagramTemplateDir, "legend.yaml")), archetype))
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("diagram strategy validation failed:\n- %s", strings.Join(problems, "\n- "))
	}
	return nil
}

func (s diagramStrategy) templateForFamily(family string) controllerDiagramTemplate {
	switch family {
	case "activity":
		return s.Activity
	case "sequence":
		return s.Sequence
	case "state-machine":
		return s.StateMachine
	default:
		return controllerDiagramTemplate{}
	}
}

func (s diagramStrategy) archetypeSummary(family string, archetype string) string {
	template := s.templateForFamily(family)
	return strings.TrimSpace(template.ArchetypeBatches[strings.TrimSpace(archetype)])
}

func (s diagramStrategy) renderSharedDiagramArtifacts() []sharedDiagramArtifact {
	out := make([]sharedDiagramArtifact, 0, len(requiredSharedDiagramFiles))
	for _, required := range requiredSharedDiagramFiles[:4] {
		spec := s.sharedByPath[required]
		out = append(out, sharedDiagramArtifact{
			SourcePath:   sharedDiagramSourcePath(required),
			RenderedPath: required,
			SourceData:   renderSharedDiagramPUML(spec, s.Legend),
		})
	}
	out = append(out, sharedDiagramArtifact{
		SourcePath:   sharedDiagramSourcePath(sharedLegendPath),
		RenderedPath: sharedLegendPath,
		SourceData:   renderLegendPUML(s.Legend),
	})
	return out
}

func sharedDiagramSourcePath(renderedPath string) string {
	renderedPath = strings.TrimSpace(renderedPath)
	if renderedPath == "" {
		return ""
	}
	ext := path.Ext(renderedPath)
	if ext == "" {
		return renderedPath + ".puml"
	}
	return strings.TrimSuffix(renderedPath, ext) + ".puml"
}

func decodeYAMLFile[T any](path string) (T, error) {
	var value T
	contents, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	return value, nil
}

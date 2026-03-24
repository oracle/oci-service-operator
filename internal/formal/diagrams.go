package formal

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

const runtimeLifecycleDiagramName = "runtime-lifecycle.yaml"

// DiagramFiles resolves the repo-local diagram artifacts for one controller row.
type DiagramFiles struct {
	Dir                      string
	RuntimeLifecyclePath     string
	ActivitySourcePath       string
	ActivityRenderedPath     string
	SequenceSourcePath       string
	SequenceRenderedPath     string
	StateMachineSourcePath   string
	StateMachineRenderedPath string
}

// RenderOptions controls deterministic formal diagram regeneration.
type RenderOptions struct {
	Root    string
	Service string
}

// RenderReport summarizes one formal diagram regeneration run.
type RenderReport struct {
	Root           string
	Service        string
	Controllers    int
	SharedDiagrams int
	FilesWritten   int
}

func (r RenderReport) String() string {
	var b strings.Builder
	b.WriteString("formal diagram render completed\n")
	fmt.Fprintf(&b, "- root: %s\n", filepath.ToSlash(r.Root))
	if strings.TrimSpace(r.Service) != "" {
		fmt.Fprintf(&b, "- service: %s\n", r.Service)
	}
	fmt.Fprintf(&b, "- shared diagrams: %d\n", r.SharedDiagrams)
	fmt.Fprintf(&b, "- controllers rendered: %d\n", r.Controllers)
	fmt.Fprintf(&b, "- files written: %d\n", r.FilesWritten)
	return b.String()
}

type diagramSpec struct {
	SchemaVersion int      `yaml:"schemaVersion"`
	Surface       string   `yaml:"surface"`
	Service       string   `yaml:"service"`
	Slug          string   `yaml:"slug"`
	Kind          string   `yaml:"kind"`
	Archetype     string   `yaml:"archetype"`
	States        []string `yaml:"states"`
	Notes         []string `yaml:"notes"`
}

type renderedDiagramArtifacts struct {
	Files              DiagramFiles
	ActivitySource     []byte
	SequenceSource     []byte
	StateMachineSource []byte
}

type diagramContext struct {
	Binding      ControllerBinding
	Diagram      diagramSpec
	Strategy     diagramStrategy
	OpenGaps     []string
	CreateHooks  []string
	UpdateHooks  []string
	DeleteHooks  []string
	ConflictSets []string
}

// DiagramFilesForRow returns the upstream-aligned artifact layout for one row.
func DiagramFilesForRow(row ManifestRow) DiagramFiles {
	return DiagramFiles{
		Dir:                      row.DiagramDir,
		RuntimeLifecyclePath:     path.Join(row.DiagramDir, runtimeLifecycleDiagramName),
		ActivitySourcePath:       path.Join(row.DiagramDir, "activity.puml"),
		ActivityRenderedPath:     path.Join(row.DiagramDir, "activity.svg"),
		SequenceSourcePath:       path.Join(row.DiagramDir, "sequence.puml"),
		SequenceRenderedPath:     path.Join(row.DiagramDir, "sequence.svg"),
		StateMachineSourcePath:   path.Join(row.DiagramDir, "state-machine.puml"),
		StateMachineRenderedPath: path.Join(row.DiagramDir, "state-machine.svg"),
	}
}

// RenderDiagrams rewrites the rendered `.puml` and `.svg` diagram artifacts for
// every controller row under the formal root.
func RenderDiagrams(opts RenderOptions) (RenderReport, error) {
	report := RenderReport{
		Root:    filepath.Clean(strings.TrimSpace(opts.Root)),
		Service: strings.TrimSpace(opts.Service),
	}
	if report.Root == "" {
		return report, fmt.Errorf("formal root must not be empty")
	}

	formalRoot, err := filepath.Abs(report.Root)
	if err != nil {
		return report, err
	}
	report.Root = formalRoot
	if err := requireDirectory(formalRoot); err != nil {
		return report, err
	}
	strategy, err := loadDiagramStrategy(formalRoot)
	if err != nil {
		return report, err
	}

	catalog, err := LoadCatalogUnchecked(formalRoot)
	if err != nil {
		return report, fmt.Errorf("load formal catalog %q: %w", formalRoot, err)
	}

	sharedArtifacts := strategy.renderSharedDiagramArtifacts()
	report.SharedDiagrams = len(sharedArtifacts)
	plantUMLArtifacts := make([]plantUMLArtifact, 0, len(sharedArtifacts)+len(catalog.Controllers)*3)

	for _, target := range sharedArtifacts {
		changed, err := writeDiagramFileIfChanged(filepath.Join(report.Root, filepath.FromSlash(target.SourcePath)), target.SourceData)
		if err != nil {
			return report, err
		}
		if changed {
			report.FilesWritten++
		}
		renderedPath := filepath.Join(report.Root, filepath.FromSlash(target.RenderedPath))
		if _, err := os.Stat(renderedPath); os.IsNotExist(err) {
			report.FilesWritten++
		} else if err != nil {
			return report, fmt.Errorf("stat %q: %w", filepath.ToSlash(target.RenderedPath), err)
		}
		plantUMLArtifacts = append(plantUMLArtifacts, plantUMLArtifact{
			SourcePath:   target.SourcePath,
			SourceData:   target.SourceData,
			RenderedPath: target.RenderedPath,
		})
	}

	for _, binding := range catalog.Controllers {
		if report.Service != "" && binding.Manifest.Service != report.Service {
			continue
		}

		artifacts, err := renderDiagramArtifacts(formalRoot, binding, strategy)
		if err != nil {
			return report, err
		}
		report.Controllers++

		for _, target := range artifacts.plantUMLArtifacts() {
			changed, err := writeDiagramFileIfChanged(filepath.Join(report.Root, filepath.FromSlash(target.SourcePath)), target.SourceData)
			if err != nil {
				return report, err
			}
			if changed {
				report.FilesWritten++
			}
			renderedPath := filepath.Join(report.Root, filepath.FromSlash(target.RenderedPath))
			if _, err := os.Stat(renderedPath); os.IsNotExist(err) {
				report.FilesWritten++
			} else if err != nil {
				return report, fmt.Errorf("stat %q: %w", filepath.ToSlash(target.RenderedPath), err)
			}
			plantUMLArtifacts = append(plantUMLArtifacts, target)
		}
	}

	if err := renderPlantUMLArtifacts(report.Root, plantUMLArtifacts); err != nil {
		return report, err
	}

	return report, nil
}

func renderDiagramArtifacts(root string, binding ControllerBinding, strategy diagramStrategy) (renderedDiagramArtifacts, error) {
	files := DiagramFilesForRow(binding.Manifest)
	metadataPath := filepath.Join(root, filepath.FromSlash(files.RuntimeLifecyclePath))
	spec, err := loadDiagram(metadataPath)
	if err != nil {
		return renderedDiagramArtifacts{}, fmt.Errorf("%s: %w", filepath.ToSlash(files.RuntimeLifecyclePath), err)
	}

	ctx := buildDiagramContext(binding, spec, strategy)
	return renderedDiagramArtifacts{
		Files:              files,
		ActivitySource:     renderActivityPUML(ctx),
		SequenceSource:     renderSequencePUML(ctx),
		StateMachineSource: renderStateMachinePUML(ctx),
	}, nil
}

func (a renderedDiagramArtifacts) plantUMLArtifacts() []plantUMLArtifact {
	return []plantUMLArtifact{
		{
			SourcePath:   a.Files.ActivitySourcePath,
			SourceData:   a.ActivitySource,
			RenderedPath: a.Files.ActivityRenderedPath,
		},
		{
			SourcePath:   a.Files.SequenceSourcePath,
			SourceData:   a.SequenceSource,
			RenderedPath: a.Files.SequenceRenderedPath,
		},
		{
			SourcePath:   a.Files.StateMachineSourcePath,
			SourceData:   a.StateMachineSource,
			RenderedPath: a.Files.StateMachineRenderedPath,
		},
	}
}

func buildDiagramContext(binding ControllerBinding, diagram diagramSpec, strategy diagramStrategy) diagramContext {
	ctx := diagramContext{
		Binding:  binding,
		Diagram:  diagram,
		Strategy: strategy,
	}

	for _, gap := range binding.LogicGaps {
		if strings.TrimSpace(gap.Status) == "resolved" {
			continue
		}
		ctx.OpenGaps = append(ctx.OpenGaps, strings.TrimSpace(gap.Category))
	}
	sort.Strings(ctx.OpenGaps)
	ctx.CreateHooks = summarizeHooks(binding.Import.Hooks.Create)
	ctx.UpdateHooks = summarizeHooks(binding.Import.Hooks.Update)
	ctx.DeleteHooks = summarizeHooks(binding.Import.Hooks.Delete)
	ctx.ConflictSets = summarizeConflicts(binding.Import.Mutation.ConflictsWith)
	return ctx
}

func loadDiagram(path string) (diagramSpec, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return diagramSpec{}, err
	}
	var diagram diagramSpec
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)
	if err := decoder.Decode(&diagram); err != nil {
		return diagramSpec{}, err
	}
	return diagram, nil
}

func renderActivityPUML(ctx diagramContext) []byte {
	binding := ctx.Binding
	template := ctx.Strategy.Activity
	lines := append(activityPlantUMLHeader(fmt.Sprintf("%s - %s/%s", template.Title, binding.Manifest.Service, binding.Manifest.Kind)),
		"start",
		plantUMLAction(fmt.Sprintf("Start from shared flow %s", template.SharedDiagram)),
		plantUMLAction(fmt.Sprintf("Load desired %s spec and status from Kubernetes", binding.Manifest.Kind)),
		plantUMLAction(fmt.Sprintf("Resolve OCI state via %s", observePathSummary(binding))),
	)

	if binding.Import.ListLookup != nil {
		lines = append(lines, plantUMLAction(fmt.Sprintf(
			"Use %s filters %s when OCID lookup or bind-versus-create needs list resolution",
			binding.Import.ListLookup.Datasource,
			summarizeValues(binding.Import.ListLookup.FilterFields, 4),
		)))
	}

	switch {
	case len(binding.Import.Operations.Create) > 0 && len(binding.Import.Operations.Update) > 0:
		lines = append(lines,
			"if (Resource exists?) then (yes)",
			plantUMLAction(fmt.Sprintf("Update path calls %s", summarizeOperations(binding.Import.Operations.Update, 3))),
		)
		if len(binding.Import.Mutation.Mutable) > 0 || len(binding.Import.Mutation.ForceNew) > 0 {
			lines = append(lines, plantUMLAction(fmt.Sprintf(
				"Guard mutable fields %s and force-new fields %s",
				summarizeValues(binding.Import.Mutation.Mutable, 4),
				summarizeValues(binding.Import.Mutation.ForceNew, 4),
			)))
		}
		lines = append(lines,
			"else (no)",
			plantUMLAction(fmt.Sprintf("Create path calls %s", summarizeOperations(binding.Import.Operations.Create, 3))),
			"endif",
		)
	case len(binding.Import.Operations.Update) > 0:
		lines = append(lines, plantUMLAction(fmt.Sprintf("Update path calls %s", summarizeOperations(binding.Import.Operations.Update, 3))))
		if len(binding.Import.Mutation.Mutable) > 0 || len(binding.Import.Mutation.ForceNew) > 0 {
			lines = append(lines, plantUMLAction(fmt.Sprintf(
				"Guard mutable fields %s and force-new fields %s",
				summarizeValues(binding.Import.Mutation.Mutable, 4),
				summarizeValues(binding.Import.Mutation.ForceNew, 4),
			)))
		}
	case len(binding.Import.Operations.Create) > 0:
		lines = append(lines, plantUMLAction(fmt.Sprintf("Create path calls %s", summarizeOperations(binding.Import.Operations.Create, 3))))
	}

	if hooks := hookPhaseSummary(ctx); hooks != "none" {
		lines = append(lines, plantUMLAction(fmt.Sprintf("Run provider helper hooks %s", hooks)))
	}
	lines = append(lines,
		plantUMLAction(fmt.Sprintf("Apply repo-authored secret policy %s", secretPolicySummary(ctx))),
		deleteActivityLine(ctx),
		plantUMLAction(fmt.Sprintf("Project %s status and finalizer policy %s", binding.Spec.StatusProjection, binding.Spec.FinalizerPolicy)),
		plantUMLAction(fmt.Sprintf("Requeue on %s", summarizeValues(binding.Spec.RequeueConditions, 4))),
	)
	if len(ctx.OpenGaps) > 0 {
		lines = append(lines, plantUMLAction(fmt.Sprintf("Keep promotion blocked on %s", summarizeValues(ctx.OpenGaps, 5))))
	}
	lines = append(lines, "stop")
	lines = append(lines, renderActivityNote(ctx)...)
	lines = append(lines, "@enduml")
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderSequencePUML(ctx diagramContext) []byte {
	binding := ctx.Binding
	template := ctx.Strategy.Sequence
	lines := sequencePlantUMLHeader(fmt.Sprintf("%s - %s/%s", template.Title, binding.Manifest.Service, binding.Manifest.Kind))
	lines = append(lines, sequenceParticipantDeclarations(template, binding, includeSecretsParticipant(ctx))...)
	lines = append(lines, "note over BaseReconciler,ServiceManager")
	lines = append(lines, wrapPlantUMLNoteLines(48, "shared resolution: "+template.SharedDiagram)...)
	if strings.TrimSpace(template.SharedDeleteDiagram) != "" {
		lines = append(lines, wrapPlantUMLNoteLines(48, "shared delete: "+template.SharedDeleteDiagram)...)
	}
	lines = append(lines, "end note")

	lines = append(lines,
		fmt.Sprintf("Kubernetes -> Controller: %s", wrapPlantUMLText(fmt.Sprintf("enqueue %s reconcile", binding.Manifest.Kind), 36)),
		"Controller -> BaseReconciler: Reconcile(request)",
		fmt.Sprintf("BaseReconciler -> ServiceManager: %s", wrapPlantUMLText(fmt.Sprintf("reconcile desired %s", binding.Manifest.Kind), 36)),
	)
	if len(binding.Import.Operations.Get) > 0 {
		lines = append(lines, fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Get, 2), 36)))
	}
	if binding.Import.ListLookup != nil {
		lines = append(lines,
			"opt list lookup fallback",
			fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(fmt.Sprintf("%s(filters: %s)", summarizeOperations(binding.Import.Operations.List, 2), summarizeValues(binding.Import.ListLookup.FilterFields, 4)), 36)),
			"OCI --> ServiceManager: matching collection items",
			"end",
		)
	}
	if len(binding.Import.Operations.Create) > 0 || len(binding.Import.Operations.Update) > 0 {
		lines = append(lines, "alt create or bind path")
		if len(binding.Import.Operations.Create) > 0 {
			lines = append(lines, fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Create, 3), 36)))
			if len(binding.Import.Lifecycle.Create.Pending) > 0 && len(binding.Import.Operations.Get) > 0 {
				lines = append(lines,
					fmt.Sprintf("loop %s", wrapPlantUMLText(fmt.Sprintf("create pending %s", summarizeValues(binding.Import.Lifecycle.Create.Pending, 4)), 36)),
					fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Get, 2), 36)),
					"end",
				)
			}
		}
		lines = append(lines, "else update or drift path")
		if len(binding.Import.Operations.Update) > 0 {
			lines = append(lines, fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Update, 3), 36)))
			if len(binding.Import.Lifecycle.Update.Pending) > 0 && len(binding.Import.Operations.Get) > 0 {
				lines = append(lines,
					fmt.Sprintf("loop %s", wrapPlantUMLText(fmt.Sprintf("update pending %s", summarizeValues(binding.Import.Lifecycle.Update.Pending, 4)), 36)),
					fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Get, 2), 36)),
					"end",
				)
			}
		}
		lines = append(lines, "end")
	}
	if includeSecretsParticipant(ctx) {
		lines = append(lines,
			"opt repo-authored secret side effects",
			fmt.Sprintf("ServiceManager -> SecretStore: %s", wrapPlantUMLText(secretSequenceAction(ctx), 36)),
			"end",
		)
	}
	if len(binding.Import.Operations.Delete) > 0 {
		lines = append(lines, "opt delete requested")
		if binding.Spec.DeleteConfirmation == "not-supported" {
			lines = append(lines, "note right of ServiceManager")
			lines = append(lines, wrapPlantUMLNoteLines(46, fmt.Sprintf("delete remains repo-authored unsupported despite %s", summarizeOperations(binding.Import.Operations.Delete, 2)))...)
			lines = append(lines, "end note")
		} else {
			lines = append(lines, fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Delete, 2), 36)))
			if len(binding.Import.DeleteConfirmation.Pending) > 0 && (len(binding.Import.Operations.Get) > 0 || len(binding.Import.Operations.List) > 0) {
				lines = append(lines,
					fmt.Sprintf("loop %s", wrapPlantUMLText(fmt.Sprintf("delete confirmation %s", summarizeValues(binding.Import.DeleteConfirmation.Pending, 3)), 36)),
					fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(deleteObserveSummary(binding), 36)),
					"end",
				)
			}
		}
		lines = append(lines, "end")
	}
	lines = append(lines,
		fmt.Sprintf("ServiceManager --> BaseReconciler: %s", wrapPlantUMLText(fmt.Sprintf("status=%s, finalizer=%s", binding.Spec.StatusProjection, binding.Spec.FinalizerPolicy), 36)),
		fmt.Sprintf("BaseReconciler --> Controller: %s", wrapPlantUMLText(fmt.Sprintf("requeue on %s", summarizeValues(binding.Spec.RequeueConditions, 4)), 36)),
	)
	lines = append(lines, renderSequenceNote(ctx)...)
	lines = append(lines, "@enduml")
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderStateMachinePUML(ctx diagramContext) []byte {
	binding := ctx.Binding
	template := ctx.Strategy.StateMachine
	stateAliases := make(map[string]string, len(ctx.Diagram.States))
	lines := statePlantUMLHeader(fmt.Sprintf("%s - %s/%s", template.Title, binding.Manifest.Service, binding.Manifest.Kind))

	for _, state := range ctx.Diagram.States {
		alias := stateAlias(state)
		stateAliases[state] = alias
		lines = append(lines, fmt.Sprintf("state %q as %s", stateLabel(state, ctx), alias))
	}

	startState := preferredState(ctx.Diagram.States, "provisioning", "active", "updating", "terminating", "failed")
	if startState != "" {
		lines = append(lines, fmt.Sprintf("[*] --> %s : %s", stateAliases[startState], wrapPlantUMLText(createEntryLabel(binding), 26)))
	}
	if hasState(ctx.Diagram.States, "provisioning") && hasState(ctx.Diagram.States, "active") {
		lines = append(lines, fmt.Sprintf("%s --> %s : %s", stateAliases["provisioning"], stateAliases["active"], wrapPlantUMLText(transitionSummary(binding.Import.Lifecycle.Create.Target, "create targets"), 26)))
	}
	if hasState(ctx.Diagram.States, "active") && hasState(ctx.Diagram.States, "updating") && len(binding.Import.Operations.Update) > 0 {
		lines = append(lines, fmt.Sprintf("%s --> %s : %s", stateAliases["active"], stateAliases["updating"], wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Update, 2), 26)))
	}
	if hasState(ctx.Diagram.States, "updating") && hasState(ctx.Diagram.States, "active") {
		lines = append(lines, fmt.Sprintf("%s --> %s : %s", stateAliases["updating"], stateAliases["active"], wrapPlantUMLText(transitionSummary(binding.Import.Lifecycle.Update.Target, "update targets"), 26)))
	}
	if hasState(ctx.Diagram.States, "active") && hasState(ctx.Diagram.States, "terminating") && len(binding.Import.Operations.Delete) > 0 && binding.Spec.DeleteConfirmation != "not-supported" {
		lines = append(lines, fmt.Sprintf("%s --> %s : %s", stateAliases["active"], stateAliases["terminating"], wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Delete, 2), 26)))
		lines = append(lines, fmt.Sprintf("%s --> [*] : %s", stateAliases["terminating"], wrapPlantUMLText(transitionSummary(binding.Import.DeleteConfirmation.Target, "delete targets"), 26)))
	}
	if hasState(ctx.Diagram.States, "failed") {
		for _, source := range []string{"provisioning", "updating", "terminating"} {
			if hasState(ctx.Diagram.States, source) {
				lines = append(lines, fmt.Sprintf("%s --> %s : %s", stateAliases[source], stateAliases["failed"], wrapPlantUMLText("unresolved OCI error or open gap", 26)))
			}
		}
		lines = append(lines, fmt.Sprintf("%s --> [*] : %s", stateAliases["failed"], wrapPlantUMLText("operator intervention", 26)))
	}
	lines = append(lines, renderStateMachineNote(ctx)...)
	lines = append(lines, "@enduml")
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderActivityNote(ctx diagramContext) []string {
	lines := []string{"note right"}
	lines = append(lines, wrapPlantUMLNoteLines(48, templateNoteLines(ctx.Strategy.Activity, ctx.Diagram.Archetype)...)...)
	lines = append(lines, wrapPlantUMLNoteLines(48, commonNoteLines(ctx)...)...)
	lines = append(lines, "end note")
	return lines
}

func renderStateMachineNote(ctx diagramContext) []string {
	anchor := preferredState(ctx.Diagram.States, "active", "provisioning", "updating", "terminating", "failed")
	if anchor == "" {
		return nil
	}
	lines := []string{fmt.Sprintf("note right of %s", stateAlias(anchor))}
	lines = append(lines, wrapPlantUMLNoteLines(48, templateNoteLines(ctx.Strategy.StateMachine, ctx.Diagram.Archetype)...)...)
	lines = append(lines, wrapPlantUMLNoteLines(48, commonNoteLines(ctx)...)...)
	lines = append(lines, "end note")
	return lines
}

func renderSequenceNote(ctx diagramContext) []string {
	lines := []string{"note right of ServiceManager"}
	lines = append(lines, wrapPlantUMLNoteLines(48, templateNoteLines(ctx.Strategy.Sequence, ctx.Diagram.Archetype)...)...)
	lines = append(lines, wrapPlantUMLNoteLines(48, commonNoteLines(ctx)...)...)
	lines = append(lines, "end note")
	return lines
}

func commonNoteLines(ctx diagramContext) []string {
	binding := ctx.Binding
	lines := []string{
		fmt.Sprintf("service: %s", binding.Manifest.Service),
		fmt.Sprintf("slug: %s", binding.Manifest.Slug),
		fmt.Sprintf("kind: %s", binding.Manifest.Kind),
		fmt.Sprintf("archetype: %s", ctx.Diagram.Archetype),
		fmt.Sprintf("status/finalizer: %s / %s", binding.Spec.StatusProjection, binding.Spec.FinalizerPolicy),
		fmt.Sprintf("requeue: %s", summarizeValues(binding.Spec.RequeueConditions, 4)),
		fmt.Sprintf("provider states: create %s; update %s; delete %s",
			transitionSummary(binding.Import.Lifecycle.Create.Pending, "none"),
			transitionSummary(binding.Import.Lifecycle.Update.Pending, "none"),
			transitionSummary(binding.Import.DeleteConfirmation.Pending, "none"),
		),
		fmt.Sprintf("mutable/forceNew: %s / %s",
			summarizeValues(binding.Import.Mutation.Mutable, 4),
			summarizeValues(binding.Import.Mutation.ForceNew, 4),
		),
		fmt.Sprintf("open gaps: %s", summarizeValues(ctx.OpenGaps, 5)),
	}
	for _, note := range ctx.Diagram.Notes {
		lines = append(lines, note)
	}
	return lines
}

func templateNoteLines(template controllerDiagramTemplate, archetype string) []string {
	lines := []string{fmt.Sprintf("shared diagram: %s", template.SharedDiagram)}
	if strings.TrimSpace(template.SharedDeleteDiagram) != "" {
		lines = append(lines, fmt.Sprintf("shared delete diagram: %s", template.SharedDeleteDiagram))
	}
	lines = append(lines, template.Summary...)
	if batch := strings.TrimSpace(template.ArchetypeBatches[strings.TrimSpace(archetype)]); batch != "" {
		lines = append(lines, "archetype batch: "+batch)
	}
	return lines
}

func sequenceParticipantDeclarations(template controllerDiagramTemplate, binding ControllerBinding, includeSecrets bool) []string {
	if len(template.Participants) == 0 {
		return []string{
			"actor Kubernetes",
			"participant Controller",
			"participant BaseReconciler",
			serviceManagerParticipant(binding),
			"participant OCI",
		}
	}

	lines := make([]string, 0, len(template.Participants))
	for _, participant := range template.Participants {
		switch strings.TrimSpace(participant) {
		case "Kubernetes":
			lines = append(lines, "actor Kubernetes")
		case "Controller":
			lines = append(lines, "participant Controller")
		case "BaseReconciler":
			lines = append(lines, "participant BaseReconciler")
		case "ServiceManager":
			lines = append(lines, serviceManagerParticipant(binding))
		case "OCI":
			lines = append(lines, "participant OCI")
		case "SecretStore":
			if includeSecrets {
				lines = append(lines, `participant "Kubernetes Secret" as SecretStore`)
			}
		}
	}
	return lines
}

func serviceManagerParticipant(binding ControllerBinding) string {
	if binding.Manifest.Stage == "seeded" && hasUnresolvedGap(binding.LogicGaps, "legacy-adapter") {
		return `participant "LegacyAdapter" as ServiceManager`
	}
	return `participant "OSOKServiceManager" as ServiceManager`
}

func includeSecretsParticipant(ctx diagramContext) bool {
	return ctx.Binding.Spec.SecretSideEffects != "none" ||
		hasUnresolvedGap(ctx.Binding.LogicGaps, "secret-input") ||
		hasUnresolvedGap(ctx.Binding.LogicGaps, "secret-output") ||
		hasUnresolvedGap(ctx.Binding.LogicGaps, "endpoint-materialization")
}

func secretSequenceAction(ctx diagramContext) string {
	parts := []string{}
	if hasUnresolvedGap(ctx.Binding.LogicGaps, "secret-input") {
		parts = append(parts, "read required input secret")
	}
	if ctx.Binding.Spec.SecretSideEffects == "ready-only" || hasUnresolvedGap(ctx.Binding.LogicGaps, "secret-output") || hasUnresolvedGap(ctx.Binding.LogicGaps, "endpoint-materialization") {
		parts = append(parts, "write ready-only connection secret")
	}
	if len(parts) == 0 {
		return "no secret writes"
	}
	return strings.Join(parts, " and ")
}

func secretPolicySummary(ctx diagramContext) string {
	switch ctx.Binding.Spec.SecretSideEffects {
	case "ready-only":
		if hasUnresolvedGap(ctx.Binding.LogicGaps, "secret-input") {
			return "ready-only writes after ACTIVE plus secret-input reads"
		}
		if hasUnresolvedGap(ctx.Binding.LogicGaps, "endpoint-materialization") || hasUnresolvedGap(ctx.Binding.LogicGaps, "secret-output") {
			return "ready-only writes after ACTIVE"
		}
		return "ready-only"
	default:
		if hasUnresolvedGap(ctx.Binding.LogicGaps, "secret-input") {
			return "no writes, but repo-authored secret reads remain in scope"
		}
		return "none"
	}
}

func deleteActivityLine(ctx diagramContext) string {
	binding := ctx.Binding
	if len(binding.Import.Operations.Delete) == 0 {
		return plantUMLAction("Delete path is not represented by imported provider facts")
	}
	if binding.Spec.DeleteConfirmation == "not-supported" {
		return plantUMLAction(fmt.Sprintf("Delete stays repo-authored not-supported despite provider op %s", summarizeOperations(binding.Import.Operations.Delete, 2)))
	}
	return plantUMLAction(fmt.Sprintf(
		"Delete path uses %s with %s confirmation",
		summarizeOperations(binding.Import.Operations.Delete, 2),
		deleteConfirmationSummary(binding),
	))
}

func deleteConfirmationSummary(binding ControllerBinding) string {
	switch binding.Spec.DeleteConfirmation {
	case "required":
		return fmt.Sprintf("required via %s to %s", deleteObserveSummary(binding), transitionSummary(binding.Import.DeleteConfirmation.Target, "terminal states"))
	case "best-effort":
		return fmt.Sprintf("best-effort via %s to %s", deleteObserveSummary(binding), transitionSummary(binding.Import.DeleteConfirmation.Target, "terminal states"))
	default:
		return "not-supported"
	}
}

func observePathSummary(binding ControllerBinding) string {
	switch {
	case len(binding.Import.Operations.Get) > 0 && len(binding.Import.Operations.List) > 0:
		return fmt.Sprintf("%s with %s fallback", summarizeOperations(binding.Import.Operations.Get, 2), summarizeOperations(binding.Import.Operations.List, 2))
	case len(binding.Import.Operations.Get) > 0:
		return summarizeOperations(binding.Import.Operations.Get, 2)
	case len(binding.Import.Operations.List) > 0:
		return summarizeOperations(binding.Import.Operations.List, 2)
	default:
		return "repo-authored state resolution"
	}
}

func deleteObserveSummary(binding ControllerBinding) string {
	switch {
	case len(binding.Import.Operations.Get) > 0 && len(binding.Import.Operations.List) > 0:
		return fmt.Sprintf("%s or %s", summarizeOperations(binding.Import.Operations.Get, 2), summarizeOperations(binding.Import.Operations.List, 2))
	case len(binding.Import.Operations.Get) > 0:
		return summarizeOperations(binding.Import.Operations.Get, 2)
	case len(binding.Import.Operations.List) > 0:
		return summarizeOperations(binding.Import.Operations.List, 2)
	default:
		return "repo-authored delete observe path"
	}
}

func listLookupSummary(binding ControllerBinding) string {
	if binding.Import.ListLookup == nil {
		return "none"
	}
	return fmt.Sprintf("%s filters %s", binding.Import.ListLookup.Datasource, summarizeValues(binding.Import.ListLookup.FilterFields, 4))
}

func hookPhaseSummary(ctx diagramContext) string {
	parts := []string{}
	if len(ctx.CreateHooks) > 0 {
		parts = append(parts, "create="+summarizeValues(ctx.CreateHooks, 3))
	}
	if len(ctx.UpdateHooks) > 0 {
		parts = append(parts, "update="+summarizeValues(ctx.UpdateHooks, 3))
	}
	if len(ctx.DeleteHooks) > 0 {
		parts = append(parts, "delete="+summarizeValues(ctx.DeleteHooks, 3))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "; ")
}

func summarizeOperations(bindings []operationBinding, limit int) string {
	names := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		name := strings.TrimSpace(binding.Operation)
		if name != "" {
			names = append(names, name)
		}
	}
	return summarizeValues(names, limit)
}

func summarizeHooks(hooks []hook) []string {
	seen := map[string]struct{}{}
	summaries := make([]string, 0, len(hooks))
	for _, hook := range hooks {
		parts := []string{strings.TrimSpace(hook.Helper)}
		if strings.TrimSpace(hook.EntityType) != "" {
			parts = append(parts, "entity="+strings.TrimSpace(hook.EntityType))
		}
		if strings.TrimSpace(hook.Action) != "" {
			parts = append(parts, "action="+strings.TrimSpace(hook.Action))
		}
		value := strings.Join(parts, " ")
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		summaries = append(summaries, value)
	}
	return summaries
}

func summarizeConflicts(conflicts map[string][]string) []string {
	if len(conflicts) == 0 {
		return nil
	}
	keys := make([]string, 0, len(conflicts))
	for key := range conflicts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%s -> %s", key, summarizeValues(conflicts[key], 3)))
	}
	return out
}

func summarizeValues(values []string, limit int) string {
	seen := map[string]struct{}{}
	clean := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		clean = append(clean, value)
	}
	if len(clean) == 0 {
		return "none"
	}
	if limit <= 0 || len(clean) <= limit {
		return strings.Join(clean, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(clean[:limit], ", "), len(clean)-limit)
}

func transitionSummary(values []string, fallback string) string {
	summary := summarizeValues(values, 4)
	if summary == "none" && strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return summary
}

func createEntryLabel(binding ControllerBinding) string {
	if len(binding.Import.Operations.Create) > 0 {
		return summarizeOperations(binding.Import.Operations.Create, 2)
	}
	return "reconcile"
}

func stateLabel(bucket string, ctx diagramContext) string {
	var providerStates []string
	switch bucket {
	case "provisioning":
		providerStates = ctx.Binding.Import.Lifecycle.Create.Pending
	case "active":
		providerStates = append(append([]string(nil), ctx.Binding.Import.Lifecycle.Create.Target...), ctx.Binding.Import.Lifecycle.Update.Target...)
	case "updating":
		providerStates = ctx.Binding.Import.Lifecycle.Update.Pending
	case "terminating":
		providerStates = ctx.Binding.Import.DeleteConfirmation.Pending
	case "failed":
		providerStates = []string{"repo-authored failure bucket"}
	}

	summary := summarizeValues(providerStates, 4)
	if summary == "none" {
		return bucket
	}
	return bucket + "\nprovider: " + summary
}

func preferredState(states []string, candidates ...string) string {
	for _, candidate := range candidates {
		if hasState(states, candidate) {
			return candidate
		}
	}
	if len(states) == 0 {
		return ""
	}
	return states[0]
}

func hasState(states []string, candidate string) bool {
	for _, state := range states {
		if strings.EqualFold(strings.TrimSpace(state), candidate) {
			return true
		}
	}
	return false
}

func stateAlias(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "state"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "state"
	}
	return out
}

func hasUnresolvedGap(gaps []LogicGap, category string) bool {
	for _, gap := range gaps {
		if strings.TrimSpace(gap.Category) != category {
			continue
		}
		if strings.TrimSpace(gap.Status) == "resolved" {
			return false
		}
		return true
	}
	return false
}

func writeDiagramFileIfChanged(fullPath string, contents []byte) (bool, error) {
	fullPath = filepath.Clean(fullPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return false, fmt.Errorf("create %q: %w", filepath.Dir(fullPath), err)
	}

	existing, err := os.ReadFile(fullPath)
	if err == nil && bytes.Equal(existing, contents) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("read %q: %w", fullPath, err)
	}

	if err := os.WriteFile(fullPath, contents, 0o644); err != nil {
		return false, fmt.Errorf("write %q: %w", fullPath, err)
	}
	return true, nil
}

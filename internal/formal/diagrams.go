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
		`partition "Observe and Bind" {`,
		plantUMLAction(fmt.Sprintf("Start from shared flow %s", template.SharedDiagram)),
		plantUMLAction(fmt.Sprintf("Read desired %s spec, tracked status, and delete intent", binding.Manifest.Kind)),
	)
	lines = append(lines, renderObserveAndBindActivity(ctx)...)
	lines = append(lines, "}")
	lines = append(lines, `if ("Delete requested?") then (yes)`)
	lines = append(lines, renderDeleteActivity(ctx)...)
	lines = append(lines, "stop")
	lines = append(lines, "else (no)")
	lines = append(lines,
		`partition "Lifecycle Classification" {`,
	)
	if retryable := retryableStateSummary(binding); retryable != "none" {
		lines = append(lines,
			`if ("OCI state in retryable set?") then (yes)`,
			plantUMLAction(fmt.Sprintf("Request requeue while OCI remains in %s", retryable)),
			"stop",
			"endif",
		)
	}
	lines = append(lines,
		`if ("OCI state is outside success targets?") then (yes)`,
		plantUMLAction(fmt.Sprintf("Return an unsuccessful reconcile result until OCI reaches %s", successTargetSummary(binding))),
		"stop",
		"endif",
		"}",
		`partition "Ready and Drift Handling" {`,
		plantUMLAction("Compare live OCI state against the imported field surface"),
	)
	if hasRejectableDrift(ctx) {
		lines = append(lines,
			`if ("Force-new or conflicting drift detected?") then (yes)`,
			plantUMLAction(rejectDriftActivitySummary(ctx)),
			"stop",
			"endif",
		)
	}
	if len(ctx.ConflictSets) > 0 {
		lines = append(lines, plantUMLAction(fmt.Sprintf(
			"Enforce conflictsWith rules %s before any OCI mutation",
			summarizeValues(ctx.ConflictSets, 3),
		)))
	}
	if len(binding.Import.Mutation.Mutable) > 0 && len(binding.Import.Operations.Update) > 0 {
		lines = append(lines,
			`if ("Supported mutable drift detected?") then (yes)`,
			plantUMLAction(updateActivitySummary(binding)),
			"else (no)",
			plantUMLAction("Skip the no-op mutation path"),
			"endif",
		)
	} else if len(binding.Import.Operations.Update) > 0 {
		lines = append(lines, plantUMLAction(fmt.Sprintf(
			"No imported mutable field surface opens %s",
			summarizeOperations(binding.Import.Operations.Update, 3),
		)))
	}
	if hooks := hookPhaseSummary(ctx); hooks != "none" {
		lines = append(lines, plantUMLAction(fmt.Sprintf(
			"Run provider helper hooks %s when the matching field-driven path executes",
			hooks,
		)))
	}
	lines = append(lines, renderSecretActivity(ctx)...)
	lines = append(lines,
		plantUMLAction(fmt.Sprintf("Project %s status and finalizer policy %s", binding.Spec.StatusProjection, binding.Spec.FinalizerPolicy)),
		plantUMLAction(fmt.Sprintf("Requeue on %s", summarizeValues(binding.Spec.RequeueConditions, 4))),
	)
	if len(ctx.OpenGaps) > 0 {
		lines = append(lines, plantUMLAction(fmt.Sprintf("Keep promotion blocked on %s", summarizeValues(ctx.OpenGaps, 5))))
	}
	lines = append(lines,
		"}",
		"endif",
		"stop",
	)
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
	lines = append(lines, renderObserveAndBindSequence(ctx)...)
	lines = append(lines, "alt delete requested")
	lines = append(lines, renderDeleteSequence(ctx)...)
	if retryable := retryableStateSummary(binding); retryable != "none" {
		lines = append(lines,
			fmt.Sprintf("else %s", wrapPlantUMLText("OCI state is retryable", 36)),
			fmt.Sprintf("ServiceManager --> BaseReconciler: %s", wrapPlantUMLText(fmt.Sprintf("requeue while OCI remains in %s", retryable), 36)),
		)
	}
	lines = append(lines,
		fmt.Sprintf("else %s", wrapPlantUMLText("live state requires field-aware drift evaluation", 36)),
		"group Drift handling",
	)
	lines = append(lines, renderDriftHandlingSequence(ctx)...)
	lines = append(lines,
		"end",
		fmt.Sprintf("ServiceManager --> BaseReconciler: %s", wrapPlantUMLText(fmt.Sprintf("status=%s, finalizer=%s", binding.Spec.StatusProjection, binding.Spec.FinalizerPolicy), 36)),
		"end",
		fmt.Sprintf("BaseReconciler --> Controller: %s", wrapPlantUMLText(fmt.Sprintf("requeue on %s", summarizeValues(binding.Spec.RequeueConditions, 4)), 36)),
	)
	lines = append(lines, renderSequenceNote(ctx)...)
	lines = append(lines, "@enduml")
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderStateMachinePUML(ctx diagramContext) []byte {
	binding := ctx.Binding
	template := ctx.Strategy.StateMachine
	lines := statePlantUMLHeader(fmt.Sprintf("%s - %s/%s", template.Title, binding.Manifest.Service, binding.Manifest.Kind))
	lines = append(lines,
		"left to right direction",
		"hide empty description",
		`state "Observe" as observe`,
		`state "EvaluateReady" as evaluate_ready`,
		`state "Ready" as ready`,
		`state "Retryable" as retryable`,
		`state "Failed" as failed`,
		"observe : read spec, tracked status, delete intent,\\nand current OCI lifecycle",
	)
	if binding.Import.ListLookup != nil {
		lines = append(lines, `state "ResolveByLookup" as resolve_by_lookup`)
	}
	if hasRejectableDrift(ctx) {
		lines = append(lines, `state "RejectUnsupportedDrift" as reject_unsupported_drift`)
	}
	if len(binding.Import.Mutation.Mutable) > 0 && len(binding.Import.Operations.Update) > 0 {
		lines = append(lines, `state "ApplyUpdate" as apply_update`)
	}
	if includeSecretsParticipant(ctx) {
		lines = append(lines,
			`state "SyncSecret" as sync_secret`,
			`state "SecretBlocked" as secret_blocked`,
		)
	}
	if len(binding.Import.Operations.Delete) > 0 {
		lines = append(lines,
			`state "DeletePending" as delete_pending`,
			`state "Deleted" as deleted`,
		)
		if includeSecretsParticipant(ctx) || binding.Spec.FinalizerPolicy == "retain-until-confirmed-delete" {
			lines = append(lines, `state "DeleteCleanupBlocked" as delete_cleanup_blocked`)
		}
	}
	lines = append(lines, fmt.Sprintf("[*] --> observe : %s", wrapPlantUMLText(createEntryLabel(binding), 28)))
	if binding.Import.ListLookup != nil {
		lines = append(lines, "observe --> resolve_by_lookup : tracked identity missing")
		lines = append(lines, fmt.Sprintf("resolve_by_lookup --> evaluate_ready : %s", wrapPlantUMLText(fmt.Sprintf("lookup or create reaches %s", successTargetSummary(binding)), 28)))
		if retryable := retryableStateSummary(binding); retryable != "none" {
			lines = append(lines, fmt.Sprintf("resolve_by_lookup --> retryable : %s", wrapPlantUMLText(fmt.Sprintf("OCI state in %s", retryable), 28)))
		}
		lines = append(lines, "resolve_by_lookup --> failed : unresolved OCI error or non-success state")
	}
	lines = append(lines, fmt.Sprintf("observe --> evaluate_ready : %s", wrapPlantUMLText(fmt.Sprintf("OCI state in %s", successTargetSummary(binding)), 28)))
	if retryable := retryableStateSummary(binding); retryable != "none" {
		lines = append(lines, fmt.Sprintf("observe --> retryable : %s", wrapPlantUMLText(fmt.Sprintf("OCI state in %s", retryable), 28)))
	}
	lines = append(lines, "observe --> failed : unresolved OCI error or non-success state")
	if hasRejectableDrift(ctx) {
		lines = append(lines,
			fmt.Sprintf("evaluate_ready --> reject_unsupported_drift : %s", wrapPlantUMLText("force-new or conflicting drift detected", 28)),
			"reject_unsupported_drift --> ready : wait for spec or live state change",
		)
	}
	if len(binding.Import.Mutation.Mutable) > 0 && len(binding.Import.Operations.Update) > 0 {
		lines = append(lines, fmt.Sprintf(
			"evaluate_ready --> apply_update : %s",
			wrapPlantUMLText(fmt.Sprintf("supported mutable drift for %s", summarizeValues(binding.Import.Mutation.Mutable, 3)), 28),
		))
		if includeSecretsParticipant(ctx) {
			lines = append(lines, "apply_update --> sync_secret : update path completes")
		} else {
			lines = append(lines, "apply_update --> ready : update path completes")
		}
	}
	if includeSecretsParticipant(ctx) {
		lines = append(lines,
			fmt.Sprintf("evaluate_ready --> sync_secret : %s", wrapPlantUMLText(fmt.Sprintf("apply secret policy %s", secretPolicySummary(ctx)), 28)),
			"sync_secret --> secret_blocked : Secret sync fails",
			"secret_blocked --> sync_secret : retry Secret sync",
			"sync_secret --> ready : Secret side effects succeed",
		)
	} else {
		lines = append(lines, "evaluate_ready --> ready : no supported drift remains")
	}
	lines = append(lines,
		"ready --> ready : no supported drift remains",
		"retryable --> retryable : OCI remains retryable",
		"failed --> failed : OCI remains terminal",
	)
	if len(binding.Import.Operations.Delete) > 0 {
		for _, from := range []string{"ready", "retryable", "failed"} {
			lines = append(lines, fmt.Sprintf("%s --> delete_pending : delete requested", from))
		}
		if includeSecretsParticipant(ctx) || binding.Spec.FinalizerPolicy == "retain-until-confirmed-delete" {
			lines = append(lines,
				"delete_pending --> delete_cleanup_blocked : cleanup or finalizer release remains blocked",
				"delete_cleanup_blocked --> deleted : retry delete completion until allowed",
			)
		} else {
			lines = append(lines, fmt.Sprintf(
				"delete_pending --> deleted : %s",
				wrapPlantUMLText(fmt.Sprintf("delete completes at %s", transitionSummary(binding.Import.DeleteConfirmation.Target, "resource absent")), 28),
			))
		}
		lines = append(lines, "deleted --> deleted : terminal stutter")
	}
	lines = append(lines, renderStateMachineNote(ctx)...)
	if len(binding.Import.Operations.Delete) > 0 {
		lines = append(lines, renderDeleteStateMachineNote(ctx)...)
	}
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
	lines := []string{"note right of ready"}
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
		fmt.Sprintf("supported drift: %s", summarizeValues(binding.Import.Mutation.Mutable, 4)),
		fmt.Sprintf("reject before mutate: %s", rejectSurfaceSummary(ctx)),
		fmt.Sprintf("list lookup: %s", listLookupSummary(binding)),
		fmt.Sprintf("conflicts: %s", summarizeValues(ctx.ConflictSets, 3)),
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

func renderObserveAndBindActivity(ctx diagramContext) []string {
	binding := ctx.Binding
	if len(binding.Import.Operations.Get) == 0 && binding.Import.ListLookup == nil && len(binding.Import.Operations.Create) == 0 {
		return []string{plantUMLAction(fmt.Sprintf("Resolve OCI state via %s", observePathSummary(binding)))}
	}

	lines := []string{`if ("Tracked or explicit OCI identity present?") then (yes)`}
	if len(binding.Import.Operations.Get) > 0 {
		lines = append(lines, plantUMLAction(fmt.Sprintf("Get the current OCI resource via %s", summarizeOperations(binding.Import.Operations.Get, 2))))
	} else {
		lines = append(lines, plantUMLAction("Use the tracked OCI identity for follow-up reconciliation"))
	}
	lines = append(lines, "else (no)")
	if binding.Import.ListLookup != nil {
		lines = append(lines, plantUMLAction(fmt.Sprintf(
			"Resolve an existing OCI resource through %s using filters %s",
			summarizeOperations(binding.Import.Operations.List, 2),
			summarizeValues(binding.Import.ListLookup.FilterFields, 4),
		)))
	}
	if len(binding.Import.Operations.Create) > 0 {
		createAction := fmt.Sprintf("Create the OCI resource via %s when no reusable match is found", summarizeOperations(binding.Import.Operations.Create, 3))
		if binding.Import.ListLookup == nil {
			createAction = fmt.Sprintf("Create the OCI resource via %s when no tracked identity is present", summarizeOperations(binding.Import.Operations.Create, 3))
		}
		lines = append(lines, plantUMLAction(createAction))
	}
	if binding.Import.ListLookup != nil || len(binding.Import.Operations.Create) > 0 {
		lines = append(lines, plantUMLAction("Persist the resolved or created OCI identity into status"))
	}
	lines = append(lines, "endif")
	return lines
}

func renderDeleteActivity(ctx diagramContext) []string {
	binding := ctx.Binding
	lines := []string{`partition "Delete" {`}
	lines = append(lines, deleteActivityLine(ctx))
	if binding.Spec.DeleteConfirmation != "not-supported" {
		lines = append(lines, plantUMLAction(fmt.Sprintf(
			"Confirm deletion via %s until OCI reaches %s",
			deleteObserveSummary(binding),
			transitionSummary(binding.Import.DeleteConfirmation.Target, "resource absent"),
		)))
	}
	if includeSecretsParticipant(ctx) {
		lines = append(lines, plantUMLAction("Delete owned Secret side effects only after OCI delete confirmation"))
	}
	switch binding.Spec.FinalizerPolicy {
	case "retain-until-confirmed-delete":
		lines = append(lines,
			`if ("Delete cleanup and finalizer release succeed?") then (yes)`,
			plantUMLAction("Remove the finalizer after delete completion"),
			"else (no)",
			plantUMLAction("Stay blocked until cleanup or finalizer release succeeds"),
			"endif",
		)
	case "none":
		lines = append(lines, plantUMLAction("Allow best-effort completion without finalizer retention"))
	}
	lines = append(lines, "}")
	return lines
}

func renderSecretActivity(ctx diagramContext) []string {
	if !includeSecretsParticipant(ctx) {
		return []string{plantUMLAction(fmt.Sprintf("Return success for the %s success state", successTargetSummary(ctx.Binding)))}
	}
	return []string{
		plantUMLAction(fmt.Sprintf("Apply repo-authored secret policy %s", secretPolicySummary(ctx))),
		`if ("Secret sync succeeds?") then (yes)`,
		plantUMLAction(fmt.Sprintf("Return success for the %s success state", successTargetSummary(ctx.Binding))),
		"else (no)",
		plantUMLAction("Block successful completion until Secret sync succeeds"),
		"endif",
	}
}

func renderObserveAndBindSequence(ctx diagramContext) []string {
	binding := ctx.Binding
	lines := []string{"group Lookup and bind"}
	if len(binding.Import.Operations.Get) > 0 {
		lines = append(lines,
			"alt tracked or explicit OCI identity already exists",
			fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Get, 2), 36)),
		)
	}
	if binding.Import.ListLookup != nil || len(binding.Import.Operations.Create) > 0 {
		lines = append(lines, "else tracked identity is missing")
		if binding.Import.ListLookup != nil {
			lines = append(lines,
				fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(
					fmt.Sprintf("%s(filters: %s)", summarizeOperations(binding.Import.Operations.List, 2), summarizeValues(binding.Import.ListLookup.FilterFields, 4)),
					36,
				)),
			)
		}
		if len(binding.Import.Operations.Create) > 0 {
			lines = append(lines,
				"alt reusable OCI resource is not found",
				fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Create, 3), 36)),
			)
			if len(binding.Import.Lifecycle.Create.Pending) > 0 && len(binding.Import.Operations.Get) > 0 {
				lines = append(lines,
					fmt.Sprintf("loop %s", wrapPlantUMLText(fmt.Sprintf("create pending %s", summarizeValues(binding.Import.Lifecycle.Create.Pending, 4)), 36)),
					fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Get, 2), 36)),
					"end",
				)
			}
			lines = append(lines, "else reusable OCI resource already exists")
			if len(binding.Import.Operations.Get) > 0 {
				lines = append(lines, fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Get, 2), 36)))
			}
			lines = append(lines, "end")
		}
		lines = append(lines, "ServiceManager --> BaseReconciler: persist resolved OCI identity in status")
	}
	lines = append(lines, "end")
	return lines
}

func renderDeleteSequence(ctx diagramContext) []string {
	binding := ctx.Binding
	lines := []string{"group Delete"}
	if binding.Spec.DeleteConfirmation == "not-supported" {
		lines = append(lines, "note right of ServiceManager")
		lines = append(lines, wrapPlantUMLNoteLines(46, fmt.Sprintf("delete remains repo-authored unsupported despite %s", summarizeOperations(binding.Import.Operations.Delete, 2)))...)
		lines = append(lines, "end note", "end")
		return lines
	}
	lines = append(lines,
		fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Delete, 2), 36)),
		fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(fmt.Sprintf("confirm delete via %s", deleteObserveSummary(binding)), 36)),
	)
	if len(binding.Import.DeleteConfirmation.Pending) > 0 && (len(binding.Import.Operations.Get) > 0 || len(binding.Import.Operations.List) > 0) {
		lines = append(lines,
			fmt.Sprintf("loop %s", wrapPlantUMLText(fmt.Sprintf("delete pending %s", summarizeValues(binding.Import.DeleteConfirmation.Pending, 3)), 36)),
			fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(deleteObserveSummary(binding), 36)),
			"end",
		)
	}
	if includeSecretsParticipant(ctx) {
		lines = append(lines, fmt.Sprintf("ServiceManager -> SecretStore: %s", wrapPlantUMLText("delete owned Secret after OCI delete confirmation", 36)))
	}
	switch binding.Spec.FinalizerPolicy {
	case "retain-until-confirmed-delete":
		lines = append(lines,
			"alt cleanup or finalizer release is blocked",
			"ServiceManager --> BaseReconciler: retain delete state and retry",
			"else delete completion observed",
			"ServiceManager --> BaseReconciler: release finalizer after delete completion",
			"end",
		)
	default:
		lines = append(lines, "ServiceManager --> BaseReconciler: best-effort delete completion observed")
	}
	lines = append(lines, "end")
	return lines
}

func renderDriftHandlingSequence(ctx diagramContext) []string {
	binding := ctx.Binding
	lines := []string{"note over ServiceManager,OCI"}
	lines = append(lines, wrapPlantUMLNoteLines(40, driftHandlingNoteLines(ctx)...)...)
	lines = append(lines, "end note")
	if hasRejectableDrift(ctx) {
		lines = append(lines,
			"opt force-new or conflicting drift is detected",
			"ServiceManager --> BaseReconciler: reject before OCI mutation",
			"end",
		)
	}
	if len(binding.Import.Mutation.Mutable) > 0 && len(binding.Import.Operations.Update) > 0 {
		lines = append(lines,
			"opt supported mutable drift is detected",
			fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(updateActivitySummary(binding), 36)),
		)
		if len(binding.Import.Lifecycle.Update.Pending) > 0 && len(binding.Import.Operations.Get) > 0 {
			lines = append(lines,
				fmt.Sprintf("loop %s", wrapPlantUMLText(fmt.Sprintf("update pending %s", summarizeValues(binding.Import.Lifecycle.Update.Pending, 4)), 36)),
				fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(summarizeOperations(binding.Import.Operations.Get, 2), 36)),
				"end",
			)
		}
		lines = append(lines, "end")
	}
	if hooks := hookPhaseSummary(ctx); hooks != "none" {
		lines = append(lines, fmt.Sprintf("ServiceManager -> OCI: %s", wrapPlantUMLText(fmt.Sprintf("run helper hooks %s", hooks), 36)))
	}
	if includeSecretsParticipant(ctx) {
		lines = append(lines,
			fmt.Sprintf("ServiceManager -> SecretStore: %s", wrapPlantUMLText(secretSequenceAction(ctx), 36)),
			"alt Secret sync fails",
			"ServiceManager --> BaseReconciler: block success and retry",
			"end",
		)
	}
	return lines
}

func renderDeleteStateMachineNote(ctx diagramContext) []string {
	lines := []string{"note right of delete_pending"}
	lines = append(lines, wrapPlantUMLNoteLines(48,
		fmt.Sprintf("delete policy: %s", ctx.Binding.Spec.DeleteConfirmation),
		fmt.Sprintf("delete observe: %s", deleteObserveSummary(ctx.Binding)),
		fmt.Sprintf("delete targets: %s", transitionSummary(ctx.Binding.Import.DeleteConfirmation.Target, "resource absent")),
	)...)
	if includeSecretsParticipant(ctx) {
		lines = append(lines, wrapPlantUMLNoteLines(48, "Secret cleanup waits for OCI delete confirmation.")...)
	}
	lines = append(lines, "end note")
	return lines
}

func hasRejectableDrift(ctx diagramContext) bool {
	return len(ctx.Binding.Import.Mutation.ForceNew) > 0 || len(ctx.ConflictSets) > 0
}

func updateActivitySummary(binding ControllerBinding) string {
	return fmt.Sprintf(
		"Apply %s only for mutable fields %s",
		summarizeOperations(binding.Import.Operations.Update, 3),
		summarizeValues(binding.Import.Mutation.Mutable, 4),
	)
}

func rejectDriftActivitySummary(ctx diagramContext) string {
	return fmt.Sprintf("Reject drift before OCI mutation for %s", rejectSurfaceSummary(ctx))
}

func rejectSurfaceSummary(ctx diagramContext) string {
	parts := make([]string, 0, 2)
	if len(ctx.Binding.Import.Mutation.ForceNew) > 0 {
		parts = append(parts, fmt.Sprintf("force-new fields %s", summarizeValues(ctx.Binding.Import.Mutation.ForceNew, 4)))
	}
	if len(ctx.ConflictSets) > 0 {
		parts = append(parts, fmt.Sprintf("conflicts %s", summarizeValues(ctx.ConflictSets, 3)))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " and ")
}

func successTargetSummary(binding ControllerBinding) string {
	switch binding.Spec.SuccessCondition {
	case "terminal":
		return transitionSummary(binding.Import.DeleteConfirmation.Target, "terminal OCI state")
	default:
		targets := append(append([]string(nil), binding.Import.Lifecycle.Create.Target...), binding.Import.Lifecycle.Update.Target...)
		return transitionSummary(targets, "usable OCI state")
	}
}

func retryableStateSummary(binding ControllerBinding) string {
	values := append([]string(nil), binding.Import.Lifecycle.Create.Pending...)
	values = append(values, binding.Import.Lifecycle.Update.Pending...)
	values = append(values, binding.Import.DeleteConfirmation.Pending...)
	return summarizeValues(values, 4)
}

func driftHandlingNoteLines(ctx diagramContext) []string {
	lines := []string{
		fmt.Sprintf("Supported update surface: %s", summarizeValues(ctx.Binding.Import.Mutation.Mutable, 4)),
		fmt.Sprintf("Reject before mutate: %s", rejectSurfaceSummary(ctx)),
	}
	if len(ctx.ConflictSets) > 0 {
		lines = append(lines, fmt.Sprintf("Conflicts: %s", summarizeValues(ctx.ConflictSets, 3)))
	}
	if ctx.Binding.Import.ListLookup != nil {
		lines = append(lines, fmt.Sprintf("Lookup filters: %s", summarizeValues(ctx.Binding.Import.ListLookup.FilterFields, 4)))
	}
	return lines
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

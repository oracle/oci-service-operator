package formal

import (
	"fmt"
	"sort"
	"strings"
)

func renderSharedDiagramPUML(spec sharedDiagramSpec, legend legendDiagramSpec) []byte {
	switch strings.TrimSpace(spec.File) {
	case sharedReconcileActivityPath:
		return renderSharedReconcileActivityPUML(spec, legend)
	case sharedResolutionSequencePath:
		return renderSharedResolutionSequencePUML(spec)
	case sharedDeleteSequencePath:
		return renderSharedDeleteSequencePUML(spec)
	case sharedControllerStatePath:
		return renderSharedControllerStatePUML(spec)
	default:
		return renderSharedFallbackPUML(spec)
	}
}

func renderLegendPUML(spec legendDiagramSpec) []byte {
	paletteKeys := make([]string, 0, len(spec.Palette))
	for key := range spec.Palette {
		paletteKeys = append(paletteKeys, key)
	}
	sort.Strings(paletteKeys)

	archetypeKeys := make([]string, 0, len(spec.ArchetypeBatches))
	for key := range spec.ArchetypeBatches {
		archetypeKeys = append(archetypeKeys, key)
	}
	sort.Strings(archetypeKeys)

	lines := append(basePlantUMLHeader(spec.Title),
		"skinparam packageStyle rectangle",
		"skinparam package {",
		"  backgroundColor #ffffff",
		"  borderColor #0f172a",
		"}",
		"skinparam rectangle {",
		"  backgroundColor #f8fafc",
		"  borderColor #cbd5e1",
		"  fontColor #0f172a",
		"}",
	)

	if strings.TrimSpace(spec.Subtitle) != "" {
		lines = append(lines, "note as legend_subtitle")
		lines = append(lines, wrapPlantUMLNoteLines(52, spec.Subtitle)...)
		lines = append(lines, "end note")
	}

	lines = append(lines, `package "Palette" {`)
	for _, key := range paletteKeys {
		entry := spec.Palette[key]
		lines = append(lines, fmt.Sprintf(`rectangle "%s" as %s`,
			renderLegendBoxLabel(entry.Label, entry.Color, entry.Description),
			stateAlias("palette_"+key),
		))
	}
	lines = append(lines, "}")
	if len(paletteKeys) > 1 {
		for i := 1; i < len(paletteKeys); i++ {
			lines = append(lines, fmt.Sprintf("%s -[hidden]-> %s",
				stateAlias("palette_"+paletteKeys[i-1]),
				stateAlias("palette_"+paletteKeys[i]),
			))
		}
	}

	lines = append(lines, `package "Controller Archetype Batches" {`)
	for _, key := range archetypeKeys {
		entry := spec.ArchetypeBatches[key]
		lines = append(lines, fmt.Sprintf(`rectangle "%s" as %s`,
			renderLegendBoxLabel(entry.Label, entry.Color, entry.Description),
			stateAlias("archetype_"+key),
		))
	}
	lines = append(lines, "}")
	if len(archetypeKeys) > 1 {
		for i := 1; i < len(archetypeKeys); i++ {
			lines = append(lines, fmt.Sprintf("%s -[hidden]-> %s",
				stateAlias("archetype_"+archetypeKeys[i-1]),
				stateAlias("archetype_"+archetypeKeys[i]),
			))
		}
	}

	if len(paletteKeys) > 0 && len(archetypeKeys) > 0 {
		lines = append(lines, fmt.Sprintf("%s -[hidden]down-> %s",
			stateAlias("palette_"+paletteKeys[0]),
			stateAlias("archetype_"+archetypeKeys[0]),
		))
	}
	if strings.TrimSpace(spec.Subtitle) != "" && len(paletteKeys) > 0 {
		lines = append(lines, fmt.Sprintf("legend_subtitle -[hidden]-> %s", stateAlias("palette_"+paletteKeys[0])))
	}

	lines = append(lines, "@enduml")
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderSharedReconcileActivityPUML(spec sharedDiagramSpec, legend legendDiagramSpec) []byte {
	lines := activityPlantUMLHeader(spec.Title)
	lines = append(lines, "start")
	if strings.TrimSpace(spec.Subtitle) != "" {
		lines = append(lines, plantUMLAction(spec.Subtitle))
	}
	for i, step := range spec.Lines {
		lines = append(lines, plantUMLAction(fmt.Sprintf("Step %d. %s", i+1, step)))
	}
	lines = append(lines, "stop")
	lines = append(lines, renderSharedLegendBlock(spec, legend)...)
	lines = append(lines, "@enduml")
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderSharedResolutionSequencePUML(spec sharedDiagramSpec) []byte {
	lines := append(sequencePlantUMLHeader(spec.Title),
		"actor Controller",
		"participant BaseReconciler",
		`participant "OSOKServiceManager" as ServiceManager`,
		"participant OCI",
	)
	if strings.TrimSpace(spec.Subtitle) != "" {
		lines = append(lines, "note over BaseReconciler,ServiceManager")
		lines = append(lines, wrapPlantUMLNoteLines(52, spec.Subtitle)...)
		lines = append(lines, "end note")
	}

	notes := append([]string(nil), spec.Lines...)
	step := func(name string, body []string) {
		lines = append(lines, "group "+name)
		lines = append(lines, body...)
		lines = append(lines, "end")
	}

	step("Step 1", []string{
		"Controller -> BaseReconciler: Start reconcile",
		"BaseReconciler -> ServiceManager: Resolve current OCI identity",
		renderSequenceNoteFor("ServiceManager", notes, 0),
	})
	step("Step 2", []string{
		"ServiceManager -> OCI: GET current resource state",
		"alt datasource fallback",
		"ServiceManager -> OCI: List resources with controller filters",
		"OCI --> ServiceManager: matching collection items",
		"end",
		renderSequenceNoteFor("OCI", notes, 1),
	})
	step("Step 3", []string{
		"ServiceManager --> BaseReconciler: Dispatch hints and retryable states",
		renderSequenceNoteFor("ServiceManager", notes, 2),
	})

	lines = append(lines, "@enduml")
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderSharedDeleteSequencePUML(spec sharedDiagramSpec) []byte {
	lines := append(sequencePlantUMLHeader(spec.Title),
		"participant BaseReconciler",
		`participant "OSOKServiceManager" as ServiceManager`,
		"participant OCI",
		`participant "Kubernetes Secret" as SecretStore`,
	)
	if strings.TrimSpace(spec.Subtitle) != "" {
		lines = append(lines, "note over BaseReconciler,ServiceManager")
		lines = append(lines, wrapPlantUMLNoteLines(52, spec.Subtitle)...)
		lines = append(lines, "end note")
	}

	notes := append([]string(nil), spec.Lines...)
	lines = append(lines,
		"group Step 1",
		"BaseReconciler -> ServiceManager: Handle delete request with finalizer discipline",
		renderSequenceNoteFor("ServiceManager", notes, 0),
		"end",
		"group Step 2",
		"ServiceManager -> OCI: Delete target resource",
		"loop confirmation until terminal delete state",
		"ServiceManager -> OCI: GET or list fallback",
		"end",
		renderSequenceNoteFor("OCI", notes, 1),
		"end",
		"group Step 3",
		"opt repo-authored secret cleanup",
		"ServiceManager -> SecretStore: Delete ready-only secret when policy allows",
		"end",
		"ServiceManager --> BaseReconciler: Remove finalizer when delete is confirmed",
		renderSequenceNoteFor("SecretStore", notes, 2),
		"end",
		"@enduml",
	)

	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderSharedControllerStatePUML(spec sharedDiagramSpec) []byte {
	lines := append(statePlantUMLHeader(spec.Title),
		`state "provisioning" as provisioning`,
		`state "active" as active`,
		`state "updating" as updating`,
		`state "terminating" as terminating`,
		`state "failed" as failed`,
		"[*] --> provisioning : create or bind",
		"provisioning --> active : target state reached",
		"active --> updating : mutable drift or spec change",
		"updating --> active : update target reached",
		"active --> terminating : delete requested",
		"terminating --> [*] : delete confirmed",
		"provisioning --> failed : unresolved OCI error",
		"updating --> failed : unresolved OCI error",
		"terminating --> failed : unresolved OCI error",
		"failed --> [*] : operator intervention",
	)
	if strings.TrimSpace(spec.Subtitle) != "" || len(spec.Lines) > 0 {
		lines = append(lines, "note right of active")
		if strings.TrimSpace(spec.Subtitle) != "" {
			lines = append(lines, wrapPlantUMLNoteLines(48, spec.Subtitle)...)
		}
		for i, step := range spec.Lines {
			lines = append(lines, wrapPlantUMLNoteLines(48, fmt.Sprintf("Step %d. %s", i+1, step))...)
		}
		lines = append(lines, "end note")
	}
	lines = append(lines, "@enduml")
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderSharedFallbackPUML(spec sharedDiagramSpec) []byte {
	lines := basePlantUMLHeader(spec.Title)
	lines = append(lines, `rectangle "Shared Diagram Summary" as summary`)
	lines = append(lines, "note right of summary")
	if strings.TrimSpace(spec.Subtitle) != "" {
		lines = append(lines, wrapPlantUMLNoteLines(52, spec.Subtitle)...)
	}
	lines = append(lines, wrapPlantUMLNoteLines(52, spec.Lines...)...)
	lines = append(lines, "end note", "@enduml")
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderLegendBoxLabel(label, color, description string) string {
	parts := []string{wrapPlantUMLText(label, 20)}
	if strings.TrimSpace(color) != "" {
		parts = append(parts, strings.ToUpper(strings.TrimSpace(color)))
	}
	parts = append(parts, wrapPlantUMLText(description, 30))
	return strings.Join(parts, `\n`)
}

func renderSharedLegendBlock(spec sharedDiagramSpec, legend legendDiagramSpec) []string {
	keys := sharedLegendKeys(spec.Lines)
	if len(keys) == 0 && strings.TrimSpace(spec.Subtitle) == "" {
		return nil
	}

	lines := []string{"legend right"}
	if strings.TrimSpace(spec.Subtitle) != "" {
		lines = append(lines, wrapPlantUMLNoteLines(46, spec.Subtitle)...)
		lines = append(lines, "--")
	}
	for _, key := range keys {
		label := legendPaletteLabel(legend, key, strings.ReplaceAll(key, "-", " "))
		color := legendPaletteColor(legend, key, fallbackPaletteColor(key))
		lines = append(lines, fmt.Sprintf("%s (%s)", label, strings.ToUpper(color)))
		if entry, ok := legend.Palette[strings.TrimSpace(key)]; ok && strings.TrimSpace(entry.Description) != "" {
			lines = append(lines, wrapPlantUMLNoteLines(46, entry.Description)...)
		}
		lines = append(lines, "--")
	}
	if lines[len(lines)-1] == "--" {
		lines = lines[:len(lines)-1]
	}
	lines = append(lines, "endlegend")
	return lines
}

func sharedLegendKeys(lines []string) []string {
	seen := map[string]struct{}{}
	keys := make([]string, 0, len(lines))
	for _, line := range lines {
		key := classifySharedLine(line)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func renderSequenceNoteFor(target string, notes []string, index int) string {
	if index >= len(notes) {
		return "note right of " + target + "\n" + strings.Join(wrapPlantUMLNoteLines(46, "Controller-local specialization keeps this shared step explicit."), "\n") + "\nend note"
	}
	return "note right of " + target + "\n" + strings.Join(wrapPlantUMLNoteLines(46, notes[index]), "\n") + "\nend note"
}

func legendPaletteColor(legend legendDiagramSpec, key, fallback string) string {
	if entry, ok := legend.Palette[strings.TrimSpace(key)]; ok && strings.TrimSpace(entry.Color) != "" {
		return strings.TrimSpace(entry.Color)
	}
	return fallback
}

func legendPaletteLabel(legend legendDiagramSpec, key, fallback string) string {
	if entry, ok := legend.Palette[strings.TrimSpace(key)]; ok && strings.TrimSpace(entry.Label) != "" {
		return strings.TrimSpace(entry.Label)
	}
	return fallback
}

func fallbackPaletteColor(key string) string {
	switch strings.TrimSpace(key) {
	case "shared-contract":
		return "#0f172a"
	case "controller-local":
		return "#1d4ed8"
	case "provider-facts":
		return "#0f766e"
	case "repo-authored":
		return "#b45309"
	case "open-gap":
		return "#b91c1c"
	default:
		return "#64748b"
	}
}

func classifySharedLine(line string) string {
	text := strings.ToLower(strings.TrimSpace(line))
	switch {
	case strings.Contains(text, "open gap"), strings.Contains(text, "legacy"):
		return "open-gap"
	case strings.Contains(text, "controller-local"), strings.Contains(text, "specializ"):
		return "controller-local"
	case strings.Contains(text, "provider"), strings.Contains(text, "wait"), strings.Contains(text, "datasource"), strings.Contains(text, "pagination"), strings.Contains(text, "imported"), strings.Contains(text, "oci identity"):
		return "provider-facts"
	case strings.Contains(text, "secret"), strings.Contains(text, "finalizer"), strings.Contains(text, "osok"), strings.Contains(text, "repo-authored"):
		return "repo-authored"
	default:
		return "shared-contract"
	}
}

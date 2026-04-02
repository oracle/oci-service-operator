package formal

import generatorplantuml "github.com/oracle/oci-service-operator/internal/generator/plantuml"

type plantUMLArtifact = generatorplantuml.Artifact

type plantUMLPair = generatorplantuml.Pair

func renderPlantUMLArtifacts(root string, artifacts []plantUMLArtifact) error {
	return generatorplantuml.RenderArtifacts(root, artifacts)
}

func validateRenderedPlantUMLArtifacts(root string, pairs []plantUMLPair) []string {
	return generatorplantuml.ValidateRenderedArtifacts(root, pairs)
}

func plantUMLBinary() (string, error) {
	return generatorplantuml.Binary()
}

func basePlantUMLHeader(title string) []string {
	return generatorplantuml.BaseHeader(title)
}

func activityPlantUMLHeader(title string) []string {
	return generatorplantuml.ActivityHeader(title)
}

func sequencePlantUMLHeader(title string) []string {
	return generatorplantuml.SequenceHeader(title)
}

func statePlantUMLHeader(title string) []string {
	return generatorplantuml.StateHeader(title)
}

func plantUMLAction(text string) string {
	return generatorplantuml.Action(text)
}

func wrapPlantUMLText(text string, limit int) string {
	return generatorplantuml.WrapText(text, limit)
}

func wrapPlantUMLNoteLines(limit int, values ...string) []string {
	return generatorplantuml.WrapNoteLines(limit, values...)
}

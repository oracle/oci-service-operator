package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func (r *Renderer) RenderMutabilityOverlayArtifacts(root string, artifacts []mutabilityOverlayGeneratedArtifact) error {
	if len(artifacts) == 0 {
		return nil
	}

	sorted := append([]mutabilityOverlayGeneratedArtifact(nil), artifacts...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RelativePath < sorted[j].RelativePath
	})
	for _, artifact := range sorted {
		content, err := renderMutabilityOverlayArtifact(artifact.Document)
		if err != nil {
			return fmt.Errorf("render %s: %w", artifact.RelativePath, err)
		}

		outputPath := filepath.Join(root, filepath.FromSlash(artifact.RelativePath))
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return fmt.Errorf("create mutability overlay dir %q: %w", filepath.Dir(outputPath), err)
		}
		if err := os.WriteFile(outputPath, content, 0o644); err != nil {
			return fmt.Errorf("write mutability overlay artifact %q: %w", outputPath, err)
		}
	}
	return nil
}

func renderMutabilityOverlayArtifact(doc mutabilityOverlayDocument) ([]byte, error) {
	if err := validateMutabilityOverlayDocument(doc); err != nil {
		return nil, err
	}
	content, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal mutability overlay document: %w", err)
	}
	return append(content, '\n'), nil
}

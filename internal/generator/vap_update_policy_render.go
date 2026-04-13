package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func (r *Renderer) RenderVAPUpdatePolicyArtifacts(root string, artifacts []vapUpdatePolicyGeneratedArtifact) error {
	if len(artifacts) == 0 {
		return nil
	}

	sorted := append([]vapUpdatePolicyGeneratedArtifact(nil), artifacts...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RelativePath < sorted[j].RelativePath
	})
	for _, artifact := range sorted {
		content, err := renderVAPUpdatePolicyArtifact(artifact.Document)
		if err != nil {
			return fmt.Errorf("render %s: %w", artifact.RelativePath, err)
		}

		outputPath := filepath.Join(root, filepath.FromSlash(artifact.RelativePath))
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return fmt.Errorf("create vap update policy dir %q: %w", filepath.Dir(outputPath), err)
		}
		if err := os.WriteFile(outputPath, content, 0o644); err != nil {
			return fmt.Errorf("write vap update policy artifact %q: %w", outputPath, err)
		}
	}
	return nil
}

func renderVAPUpdatePolicyArtifact(doc vapUpdatePolicyDocument) ([]byte, error) {
	if err := validateVAPUpdatePolicyDocument(doc); err != nil {
		return nil, err
	}
	content, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal vap update policy document: %w", err)
	}
	return append(content, '\n'), nil
}

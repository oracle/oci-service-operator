package crdsync

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const resourceScaffoldMarker = "# +kubebuilder:scaffold:crdkustomizeresource"

// SyncFile refreshes the shared CRD resource list from the generated base manifests.
func SyncFile(kustomizationPath, basesDir string) error {
	resources, err := resourcePaths(kustomizationPath, basesDir)
	if err != nil {
		return err
	}

	current, err := os.ReadFile(kustomizationPath)
	if err != nil {
		return fmt.Errorf("read kustomization %q: %w", kustomizationPath, err)
	}

	updated, err := syncResourcesBlock(current, resources)
	if err != nil {
		return fmt.Errorf("sync kustomization %q: %w", kustomizationPath, err)
	}
	if bytes.Equal(current, updated) {
		return nil
	}

	if err := os.WriteFile(kustomizationPath, updated, 0o644); err != nil {
		return fmt.Errorf("write kustomization %q: %w", kustomizationPath, err)
	}

	return nil
}

func resourcePaths(kustomizationPath, basesDir string) ([]string, error) {
	entries, err := os.ReadDir(basesDir)
	if err != nil {
		return nil, fmt.Errorf("read bases dir %q: %w", basesDir, err)
	}

	kustomizationDir := filepath.Dir(kustomizationPath)
	resources := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".yaml" || entry.Name() == "kustomization.yaml" {
			continue
		}

		relPath, err := filepath.Rel(kustomizationDir, filepath.Join(basesDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("build relative path for %q: %w", entry.Name(), err)
		}

		resources = append(resources, filepath.ToSlash(relPath))
	}

	sort.Strings(resources)
	return resources, nil
}

func syncResourcesBlock(content []byte, resources []string) ([]byte, error) {
	text := string(content)

	const resourcesSection = "resources:\n"
	start := strings.Index(text, resourcesSection)
	if start == -1 {
		return nil, fmt.Errorf("resources section not found")
	}

	blockStart := start + len(resourcesSection)
	blockEnd := strings.Index(text, resourceScaffoldMarker)
	if blockEnd == -1 {
		return nil, fmt.Errorf("%s marker not found", resourceScaffoldMarker)
	}
	if blockEnd < blockStart {
		return nil, fmt.Errorf("%s marker appears before resources section", resourceScaffoldMarker)
	}

	var builder strings.Builder
	builder.Grow(len(text) + len(resources)*32)
	builder.WriteString(text[:blockStart])
	for _, resource := range resources {
		builder.WriteString("- ")
		builder.WriteString(resource)
		builder.WriteByte('\n')
	}
	builder.WriteString(text[blockEnd:])

	return []byte(builder.String()), nil
}

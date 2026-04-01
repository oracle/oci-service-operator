/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package formal

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
	"testing/quick"
)

type quickFormalArtifactCase struct {
	service string
	slug    string
	kind    string
}

func TestValidateManifestOwnedArtifactsQuickRejectsOnlyUnreferencedArtifacts(t *testing.T) {
	t.Parallel()

	candidates := []quickFormalArtifactCase{
		{service: "alpha", slug: "widget", kind: "Widget"},
		{service: "beta", slug: "report", kind: "Report"},
		{service: "gamma", slug: "item", kind: "Item"},
		{service: "delta", slug: "network", kind: "Network"},
	}

	property := func(existingControllerMask uint8, existingImportMask uint8, desiredMask uint8) bool {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "controllers"), 0o755); err != nil {
			return false
		}
		if err := os.MkdirAll(filepath.Join(root, "imports"), 0o755); err != nil {
			return false
		}

		if err := writeQuickFormalFile(filepath.Join(root, "controllers", "README.md")); err != nil {
			return false
		}
		if err := writeQuickFormalFile(filepath.Join(root, "imports", "README.txt")); err != nil {
			return false
		}

		rows := make([]manifestRow, 0, len(candidates))
		wantProblems := make([]string, 0, len(candidates)*2)

		for i, candidate := range candidates {
			controllerRoot := filepath.Join(root, "controllers", candidate.service, candidate.slug)
			specPath := filepath.Join(controllerRoot, "spec.cfg")
			logicPath := filepath.Join(controllerRoot, "logic-gaps.md")
			diagramPath := filepath.Join(controllerRoot, "diagrams", "runtime-lifecycle.yaml")
			importPath := filepath.Join(root, "imports", candidate.service, candidate.slug+".json")

			if quickFormalMaskBit(existingControllerMask, i) {
				if err := writeQuickFormalFile(specPath); err != nil {
					return false
				}
				if err := writeQuickFormalFile(logicPath); err != nil {
					return false
				}
				if err := writeQuickFormalFile(diagramPath); err != nil {
					return false
				}
			}
			if quickFormalMaskBit(existingImportMask, i) {
				if err := writeQuickFormalFile(importPath); err != nil {
					return false
				}
			}
			if quickFormalMaskBit(desiredMask, i) {
				rows = append(rows, manifestRow{
					Line:       i + 1,
					Service:    candidate.service,
					Slug:       candidate.slug,
					Kind:       candidate.kind,
					ImportPath: filepath.ToSlash(filepath.Join("imports", candidate.service, candidate.slug+".json")),
					SpecPath:   filepath.ToSlash(filepath.Join("controllers", candidate.service, candidate.slug, "spec.cfg")),
					LogicPath:  filepath.ToSlash(filepath.Join("controllers", candidate.service, candidate.slug, "logic-gaps.md")),
					DiagramDir: filepath.ToSlash(filepath.Join("controllers", candidate.service, candidate.slug, "diagrams")),
				})
				continue
			}

			if quickFormalMaskBit(existingControllerMask, i) {
				wantProblems = append(wantProblems, filepath.ToSlash(filepath.Join("controllers", candidate.service, candidate.slug))+": stale controller artifacts are not referenced by controller_manifest.tsv")
			}
			if quickFormalMaskBit(existingImportMask, i) {
				wantProblems = append(wantProblems, filepath.ToSlash(filepath.Join("imports", candidate.service, candidate.slug+".json"))+": stale import artifact is not referenced by controller_manifest.tsv")
			}
		}

		gotProblems := validateManifestOwnedArtifacts(root, rows)
		sort.Strings(gotProblems)
		sort.Strings(wantProblems)
		return slices.Equal(gotProblems, wantProblems)
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 96}); err != nil {
		t.Fatal(err)
	}
}

func quickFormalMaskBit(mask uint8, index int) bool {
	return mask&(1<<index) != 0
}

func writeQuickFormalFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("seed\n"), 0o644)
}

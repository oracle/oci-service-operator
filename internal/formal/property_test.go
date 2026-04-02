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

type quickFormalArtifacts struct {
	rows         []manifestRow
	wantProblems []string
}

type quickFormalPaths struct {
	specPath    string
	logicPath   string
	diagramPath string
	importPath  string
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
		artifacts, err := quickFormalArtifactsForMasks(root, candidates, existingControllerMask, existingImportMask, desiredMask)
		if err != nil {
			return false
		}

		gotProblems := validateManifestOwnedArtifacts(root, artifacts.rows)
		sort.Strings(gotProblems)
		sort.Strings(artifacts.wantProblems)
		return slices.Equal(gotProblems, artifacts.wantProblems)
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 96}); err != nil {
		t.Fatal(err)
	}
}

func quickFormalMaskBit(mask uint8, index int) bool {
	return mask&(1<<index) != 0
}

func quickFormalArtifactsForMasks(
	root string,
	candidates []quickFormalArtifactCase,
	existingControllerMask uint8,
	existingImportMask uint8,
	desiredMask uint8,
) (quickFormalArtifacts, error) {
	if err := quickInitFormalArtifactRoot(root); err != nil {
		return quickFormalArtifacts{}, err
	}

	artifacts := quickFormalArtifacts{
		rows:         make([]manifestRow, 0, len(candidates)),
		wantProblems: make([]string, 0, len(candidates)*2),
	}

	for i, candidate := range candidates {
		wantController := quickFormalMaskBit(existingControllerMask, i)
		wantImport := quickFormalMaskBit(existingImportMask, i)
		desired := quickFormalMaskBit(desiredMask, i)

		row, problems, err := quickFormalCandidateOutcome(root, candidate, i, wantController, wantImport, desired)
		if err != nil {
			return quickFormalArtifacts{}, err
		}
		if row != nil {
			artifacts.rows = append(artifacts.rows, *row)
		}
		artifacts.wantProblems = append(artifacts.wantProblems, problems...)
	}

	return artifacts, nil
}

func quickInitFormalArtifactRoot(root string) error {
	for _, dir := range []string{
		filepath.Join(root, "controllers"),
		filepath.Join(root, "imports"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	for _, path := range []string{
		filepath.Join(root, "controllers", "README.md"),
		filepath.Join(root, "imports", "README.txt"),
	} {
		if err := writeQuickFormalFile(path); err != nil {
			return err
		}
	}
	return nil
}

func quickFormalCandidateOutcome(
	root string,
	candidate quickFormalArtifactCase,
	index int,
	existingController bool,
	existingImport bool,
	desired bool,
) (*manifestRow, []string, error) {
	paths := quickFormalArtifactPaths(root, candidate)
	if err := quickWriteFormalArtifacts(paths, existingController, existingImport); err != nil {
		return nil, nil, err
	}
	if desired {
		row := quickFormalManifestRow(candidate, index)
		return &row, nil, nil
	}
	return nil, quickUnexpectedFormalProblems(candidate, existingController, existingImport), nil
}

func quickFormalArtifactPaths(root string, candidate quickFormalArtifactCase) quickFormalPaths {
	controllerRoot := filepath.Join(root, "controllers", candidate.service, candidate.slug)
	return quickFormalPaths{
		specPath:    filepath.Join(controllerRoot, "spec.cfg"),
		logicPath:   filepath.Join(controllerRoot, "logic-gaps.md"),
		diagramPath: filepath.Join(controllerRoot, "diagrams", "runtime-lifecycle.yaml"),
		importPath:  filepath.Join(root, "imports", candidate.service, candidate.slug+".json"),
	}
}

func quickWriteFormalArtifacts(paths quickFormalPaths, existingController bool, existingImport bool) error {
	if existingController {
		for _, path := range []string{paths.specPath, paths.logicPath, paths.diagramPath} {
			if err := writeQuickFormalFile(path); err != nil {
				return err
			}
		}
	}
	if existingImport {
		if err := writeQuickFormalFile(paths.importPath); err != nil {
			return err
		}
	}
	return nil
}

func quickFormalManifestRow(candidate quickFormalArtifactCase, index int) manifestRow {
	return manifestRow{
		Line:       index + 1,
		Service:    candidate.service,
		Slug:       candidate.slug,
		Kind:       candidate.kind,
		ImportPath: filepath.ToSlash(filepath.Join("imports", candidate.service, candidate.slug+".json")),
		SpecPath:   filepath.ToSlash(filepath.Join("controllers", candidate.service, candidate.slug, "spec.cfg")),
		LogicPath:  filepath.ToSlash(filepath.Join("controllers", candidate.service, candidate.slug, "logic-gaps.md")),
		DiagramDir: filepath.ToSlash(filepath.Join("controllers", candidate.service, candidate.slug, "diagrams")),
	}
}

func quickUnexpectedFormalProblems(candidate quickFormalArtifactCase, existingController bool, existingImport bool) []string {
	problems := make([]string, 0, 2)
	if existingController {
		problems = append(problems, filepath.ToSlash(filepath.Join("controllers", candidate.service, candidate.slug))+": stale controller artifacts are not referenced by controller_manifest.tsv")
	}
	if existingImport {
		problems = append(problems, filepath.ToSlash(filepath.Join("imports", candidate.service, candidate.slug+".json"))+": stale import artifact is not referenced by controller_manifest.tsv")
	}
	return problems
}

func writeQuickFormalFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("seed\n"), 0o644)
}

/*
 Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
 Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sitegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyGeneratedDocsMatchRepo(t *testing.T) {
	t.Parallel()

	if err := VerifyGeneratedDocsMatchRepo(repoRootFromCwd(t)); err != nil {
		t.Fatalf("VerifyGeneratedDocsMatchRepo() error = %v", err)
	}
}

func TestVerifyBuiltSiteLinksDetectsMissingAnchor(t *testing.T) {
	t.Parallel()

	siteDir := t.TempDir()
	writeFile(t, filepath.Join(siteDir, "index.html"), `<html><body><a href="guide/#missing">Guide</a></body></html>`)
	writeFile(t, filepath.Join(siteDir, "guide", "index.html"), `<html><body><h1 id="present">Guide</h1></body></html>`)

	err := VerifyBuiltSiteLinks(siteDir)
	if err == nil {
		t.Fatalf("VerifyBuiltSiteLinks() error = nil, want missing anchor failure")
	}
	if !strings.Contains(err.Error(), "anchor #missing not found") {
		t.Fatalf("VerifyBuiltSiteLinks() error = %v, want missing anchor message", err)
	}
}

func TestVerifyMarkdownSourceLinksDetectsBrokenREADMEAnchor(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "# Root\n\nSee [Installation](docs/README.md#missing-anchor).\n")
	writeFile(t, filepath.Join(root, "docs", "README.md"), "# Docs Home\n")

	err := verifyMarkdownSourceLinks(root, []string{"README.md", "docs/README.md"})
	if err == nil {
		t.Fatalf("verifyMarkdownSourceLinks() error = nil, want broken anchor failure")
	}
	if !strings.Contains(err.Error(), "anchor #missing-anchor not found") {
		t.Fatalf("verifyMarkdownSourceLinks() error = %v, want missing anchor message", err)
	}
}

func TestDescriptionCoverageWarningsOnlyGatePublicSpecFields(t *testing.T) {
	t.Parallel()

	pages := []*apiReferencePage{
		{
			FullGroup: "mysql.oracle.com",
			Version:   "v1beta1",
			Resources: []apiReferenceResource{
				{
					Kind:    "DbSystem",
					Summary: "",
					Packages: []packageExposure{
						{Package: "mysql"},
					},
					SpecSection: &schemaSection{
						Title: "Spec",
						Fields: []schemaField{
							{Name: "displayName", Description: ""},
							{Name: "shapeName", Description: "Rendered shape name."},
						},
						Nested: []*schemaSection{
							{
								Title: "Spec.network",
								Fields: []schemaField{
									{Name: "subnetId", Description: ""},
								},
							},
						},
					},
				},
				{
					Kind:        "HiddenThing",
					Summary:     "hidden",
					SpecSection: &schemaSection{Title: "Spec", Fields: []schemaField{{Name: "ignored", Description: ""}}},
				},
			},
		},
	}

	findings := descriptionCoverageWarnings(pages)
	if len(findings) != 3 {
		t.Fatalf("descriptionCoverageWarnings() len = %d, want 3 (%v)", len(findings), findings)
	}
	assertContainsFinding(t, findings, "mysql.oracle.com/v1beta1 DbSystem: missing top-level kind description")
	assertContainsFinding(t, findings, "mysql.oracle.com/v1beta1 DbSystem spec.displayName: missing public spec-field description")
	assertContainsFinding(t, findings, "mysql.oracle.com/v1beta1 DbSystem spec.network.subnetId: missing public spec-field description")
}

func assertContainsFinding(t *testing.T, findings []string, want string) {
	t.Helper()

	for _, finding := range findings {
		if finding == want {
			return
		}
	}
	t.Fatalf("findings %v do not contain %q", findings, want)
}

func repoRootFromCwd(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %q", dir)
		}
		dir = parent
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

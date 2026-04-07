/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sitegen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/internal/sitegen"
)

func TestBuildAPIReferenceSiteQueuePage(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	site, err := sitegen.BuildAPIReferenceSite(sitegen.APIReferenceBuildOptions{
		RepoRoot:      root,
		SampleRepoURL: "https://github.com/oracle/oci-service-operator",
		SampleRepoRef: "main",
	})
	if err != nil {
		t.Fatalf("BuildAPIReferenceSite() error = %v", err)
	}

	assertContains(t, site.LandingPage.Content, "[queue.oracle.com/v1beta1](queue/v1beta1/index.md)")
	assertContains(t, site.LandingPage.Content, "Core Compute (`Not yet released`), Core Networking (`v2.0.0-alpha`)")

	page := generatedPage(t, site, "queue/v1beta1/index.md")
	assertContains(t, page.Content, "# queue.oracle.com/v1beta1")
	assertContains(t, page.Content, "`APIVersion`: `queue.oracle.com/v1beta1`")
	assertContains(t, page.Content, "Manage OCI Queue service queues.")
	assertContains(t, page.Content, "| Package | Support | Latest release | Resources |")
	assertContains(t, page.Content, "[Queue](#kind-queue)")
	assertContains(t, page.Content, "queue_v1beta1_queue.yaml")
	assertContains(t, page.Content, "<a id=\"kind-queue-spec\"></a>")
	assertContains(t, page.Content, "This content is generated from the checked-in CRD schemas")

	corePage := generatedPage(t, site, "core/v1beta1/index.md")
	assertContains(t, corePage.Content, "| Core Compute | preview | `Not yet released` | [Instance](#kind-instance) |")
	assertContains(t, corePage.Content, "| [Instance](#kind-instance) | Namespaced | [Sample](../../../samples/core/v1beta1/instance.md) | Core Compute (`Not yet released`) |")
}

func TestGenerateAPIReferenceSiteWritesRenderedPages(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	outputDir := t.TempDir()
	if err := sitegen.GenerateAPIReferenceSite(sitegen.APIReferenceBuildOptions{
		RepoRoot:      root,
		OutputDir:     outputDir,
		SampleRepoURL: "https://github.com/oracle/oci-service-operator",
		SampleRepoRef: "main",
	}); err != nil {
		t.Fatalf("GenerateAPIReferenceSite() error = %v", err)
	}

	landing := readGeneratedFile(t, filepath.Join(outputDir, "index.md"))
	assertContains(t, landing, "[apigateway.oracle.com/v1beta1](apigateway/v1beta1/index.md)")

	apigatewayPage := readGeneratedFile(t, filepath.Join(outputDir, "apigateway", "v1beta1", "index.md"))
	assertContains(t, apigatewayPage, "`PRIVATE`")
	assertContains(t, apigatewayPage, "`PUBLIC`")
	assertContains(t, apigatewayPage, "Validation: endpointType is immutable.")

	corePage := readGeneratedFile(t, filepath.Join(outputDir, "core", "v1beta1", "index.md"))
	assertContains(t, corePage, "object (preserves unknown fields)")
	assertContains(t, corePage, "Spec.createVnicDetails")
}

func generatedPage(t *testing.T, site *sitegen.APIReferenceSite, relativePath string) sitegen.GeneratedFile {
	t.Helper()

	for _, page := range site.Pages {
		if page.RelativePath == relativePath {
			return page
		}
	}

	t.Fatalf("generated page %q not found", relativePath)
	return sitegen.GeneratedFile{}
}

func readGeneratedFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()

	if !strings.Contains(got, want) {
		t.Fatalf("expected output to contain %q", want)
	}
}

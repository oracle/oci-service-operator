package generator

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcquireMutabilityOverlayDocsInputsReturnsDeterministicInputs(t *testing.T) {
	t.Parallel()

	targetInstance := newMutabilityOverlayDocsTargetForTest(
		"core",
		"Instance",
		"instance",
		"oci_core_instance",
		"7.22.0",
		"https://registry.example.test/providers/oracle/oci/7.22.0/docs/resources/core_instance",
	)
	targetVcn := newMutabilityOverlayDocsTargetForTest(
		"core",
		"Vcn",
		"vcn",
		"oci_core_vcn",
		"7.22.0",
		"https://registry.example.test/providers/oracle/oci/7.22.0/docs/resources/core_vcn",
	)
	fetcher := stubMutabilityOverlayDocsFetcher{
		responses: map[string]mutabilityOverlayDocsHTTPResponse{
			targetInstance.RegistryURL: {
				StatusCode:  http.StatusOK,
				ContentType: "text/html; charset=utf-8",
				Body:        []byte("<html>\r\n<body><h2>Argument Reference</h2></body>\r\n</html>\r\n"),
			},
			targetVcn.RegistryURL: {
				StatusCode:  http.StatusOK,
				ContentType: "text/html; charset=utf-8",
				Body:        []byte("<html>\n<body><h2>Argument Reference</h2></body>\n</html>\n"),
			},
		},
	}

	inputs, err := acquireMutabilityOverlayDocsInputs(context.Background(), []mutabilityOverlayRegistryPageTarget{targetVcn, targetInstance}, fetcher)
	if err != nil {
		t.Fatalf("acquireMutabilityOverlayDocsInputs() error = %v", err)
	}
	if len(inputs) != 2 {
		t.Fatalf("acquireMutabilityOverlayDocsInputs() returned %d inputs, want 2", len(inputs))
	}

	if got := inputs[0].Metadata.Kind; got != "Instance" {
		t.Fatalf("inputs[0].Metadata.Kind = %q, want %q", got, "Instance")
	}
	if got := inputs[0].Metadata.InputSource; got != mutabilityOverlayDocsInputSourceLive {
		t.Fatalf("inputs[0].Metadata.InputSource = %q, want %q", got, mutabilityOverlayDocsInputSourceLive)
	}
	if got := inputs[0].Metadata.InputIdentity; got != "fetch:"+targetInstance.RegistryURL {
		t.Fatalf("inputs[0].Metadata.InputIdentity = %q, want %q", got, "fetch:"+targetInstance.RegistryURL)
	}
	if got := inputs[0].Metadata.ContentType; got != "text/html; charset=utf-8" {
		t.Fatalf("inputs[0].Metadata.ContentType = %q, want %q", got, "text/html; charset=utf-8")
	}
	if got := inputs[0].Body; got != "<html>\n<body><h2>Argument Reference</h2></body>\n</html>\n" {
		t.Fatalf("inputs[0].Body = %q, want canonical newline form", got)
	}
	if inputs[0].Metadata.BodyBytes != len(inputs[0].Body) {
		t.Fatalf("inputs[0].Metadata.BodyBytes = %d, want %d", inputs[0].Metadata.BodyBytes, len(inputs[0].Body))
	}
	if got := inputs[1].Metadata.Kind; got != "Vcn" {
		t.Fatalf("inputs[1].Metadata.Kind = %q, want %q", got, "Vcn")
	}
}

func TestAcquireMutabilityOverlayDocsInputsClassifiesFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		response   mutabilityOverlayDocsHTTPResponse
		fetchErr   error
		wantReason string
		wantDetail string
	}{
		{
			name: "missing page",
			response: mutabilityOverlayDocsHTTPResponse{
				StatusCode: http.StatusNotFound,
			},
			wantReason: mutabilityOverlayDocsErrorMissingPage,
			wantDetail: "does not exist",
		},
		{
			name: "redirected page",
			response: mutabilityOverlayDocsHTTPResponse{
				StatusCode: http.StatusMovedPermanently,
				Location:   "/providers/oracle/oci/7.22.0/docs/resources/core_compute_instance",
			},
			wantReason: mutabilityOverlayDocsErrorRedirectedPage,
			wantDetail: "renamed or moved",
		},
		{
			name: "rate limited",
			response: mutabilityOverlayDocsHTTPResponse{
				StatusCode: http.StatusTooManyRequests,
			},
			wantReason: mutabilityOverlayDocsErrorRateLimited,
			wantDetail: "rate limited",
		},
		{
			name:       "availability failure from transport",
			fetchErr:   errors.New("dial tcp: no route to host"),
			wantReason: mutabilityOverlayDocsErrorAvailabilityFailure,
			wantDetail: "no route to host",
		},
		{
			name: "unexpected content type",
			response: mutabilityOverlayDocsHTTPResponse{
				StatusCode:  http.StatusOK,
				ContentType: "application/json",
				Body:        []byte(`{"message":"not html"}`),
			},
			wantReason: mutabilityOverlayDocsErrorUnexpectedContentType,
			wantDetail: "expected HTML",
		},
		{
			name: "javascript only page",
			response: mutabilityOverlayDocsHTTPResponse{
				StatusCode:  http.StatusOK,
				ContentType: "text/html; charset=utf-8",
				Body:        []byte("<html><body>Please enable JavaScript to continue.</body></html>"),
			},
			wantReason: mutabilityOverlayDocsErrorJavaScriptOnlyPage,
			wantDetail: "requires JavaScript",
		},
		{
			name: "structurally missing page",
			response: mutabilityOverlayDocsHTTPResponse{
				StatusCode:  http.StatusOK,
				ContentType: "text/html; charset=utf-8",
				Body:        []byte("<html><body><h1>Overview</h1></body></html>"),
			},
			wantReason: mutabilityOverlayDocsErrorStructurallyMissingPage,
			wantDetail: "Argument Reference",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			target := newMutabilityOverlayDocsTargetForTest(
				"core",
				"Instance",
				"instance",
				"oci_core_instance",
				"7.22.0",
				"https://registry.example.test/providers/oracle/oci/7.22.0/docs/resources/core_instance",
			)
			fetcher := stubMutabilityOverlayDocsFetcher{
				responses: map[string]mutabilityOverlayDocsHTTPResponse{
					target.RegistryURL: test.response,
				},
				errors: map[string]error{
					target.RegistryURL: test.fetchErr,
				},
			}

			inputs, err := acquireMutabilityOverlayDocsInputs(context.Background(), []mutabilityOverlayRegistryPageTarget{target}, fetcher)
			if err == nil {
				t.Fatal("acquireMutabilityOverlayDocsInputs() unexpectedly succeeded")
			}
			if len(inputs) != 0 {
				t.Fatalf("acquireMutabilityOverlayDocsInputs() returned %d inputs, want none", len(inputs))
			}

			var acquisitionErr *mutabilityOverlayDocsAcquisitionError
			if !errors.As(err, &acquisitionErr) {
				t.Fatalf("acquireMutabilityOverlayDocsInputs() error = %v, want mutabilityOverlayDocsAcquisitionError", err)
			}
			if acquisitionErr.Reason != test.wantReason {
				t.Fatalf("acquisition error reason = %q, want %q", acquisitionErr.Reason, test.wantReason)
			}
			if !strings.Contains(acquisitionErr.Error(), test.wantDetail) {
				t.Fatalf("acquisition error detail = %q, want substring %q", acquisitionErr.Error(), test.wantDetail)
			}
		})
	}
}

func TestRefreshMutabilityOverlayDocsFixturesWritesAndLoadsDeterministicLayout(t *testing.T) {
	t.Parallel()

	target := newMutabilityOverlayDocsTargetForTest(
		"nosql",
		"Table",
		"table",
		"oci_nosql_table",
		"contract-example-v1",
		"https://registry.example.test/providers/oracle/oci/contract-example-v1/docs/resources/nosql_table",
	)
	fetcher := stubMutabilityOverlayDocsFetcher{
		responses: map[string]mutabilityOverlayDocsHTTPResponse{
			target.RegistryURL: {
				StatusCode:  http.StatusOK,
				ContentType: "text/html; charset=utf-8",
				Body:        []byte("<!DOCTYPE html>\r\n<html>\r\n  <body>\r\n    <h2>Argument Reference</h2>\r\n  </body>\r\n</html>\r\n"),
			},
		},
	}

	root := t.TempDir()
	refreshed, err := refreshMutabilityOverlayDocsFixtures(context.Background(), root, []mutabilityOverlayRegistryPageTarget{target}, fetcher)
	if err != nil {
		t.Fatalf("refreshMutabilityOverlayDocsFixtures() error = %v", err)
	}
	if len(refreshed) != 1 {
		t.Fatalf("refreshMutabilityOverlayDocsFixtures() returned %d inputs, want 1", len(refreshed))
	}
	if got := refreshed[0].Metadata.InputSource; got != mutabilityOverlayDocsInputSourceFixture {
		t.Fatalf("refreshed[0].Metadata.InputSource = %q, want %q", got, mutabilityOverlayDocsInputSourceFixture)
	}

	metadataPath := filepath.Join(root, filepath.FromSlash(mutabilityOverlayDocsFixtureMetadataRelativePath(target)))
	bodyPath := filepath.Join(root, filepath.FromSlash(mutabilityOverlayDocsFixtureBodyRelativePath(target)))
	metadataBefore := readMutabilityOverlayDocsTestFile(t, metadataPath)
	bodyBefore := readMutabilityOverlayDocsTestFile(t, bodyPath)

	loaded, err := loadMutabilityOverlayDocsFixtures(root, []mutabilityOverlayRegistryPageTarget{target})
	if err != nil {
		t.Fatalf("loadMutabilityOverlayDocsFixtures() error = %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("loadMutabilityOverlayDocsFixtures() returned %d inputs, want 1", len(loaded))
	}
	if canonicalJSON(t, loaded[0].Metadata) != canonicalJSON(t, refreshed[0].Metadata) {
		t.Fatalf("loaded metadata mismatch\nwant:\n%s\n\ngot:\n%s", canonicalJSON(t, refreshed[0].Metadata), canonicalJSON(t, loaded[0].Metadata))
	}
	if loaded[0].Body != refreshed[0].Body {
		t.Fatalf("loaded body mismatch\nwant:\n%s\n\ngot:\n%s", refreshed[0].Body, loaded[0].Body)
	}

	refreshedAgain, err := refreshMutabilityOverlayDocsFixtures(context.Background(), root, []mutabilityOverlayRegistryPageTarget{target}, fetcher)
	if err != nil {
		t.Fatalf("second refreshMutabilityOverlayDocsFixtures() error = %v", err)
	}
	if len(refreshedAgain) != 1 {
		t.Fatalf("second refreshMutabilityOverlayDocsFixtures() returned %d inputs, want 1", len(refreshedAgain))
	}
	metadataAfter := readMutabilityOverlayDocsTestFile(t, metadataPath)
	bodyAfter := readMutabilityOverlayDocsTestFile(t, bodyPath)
	if metadataAfter != metadataBefore {
		t.Fatalf("metadata fixture changed across identical refreshes\nbefore:\n%s\n\nafter:\n%s", metadataBefore, metadataAfter)
	}
	if bodyAfter != bodyBefore {
		t.Fatalf("body fixture changed across identical refreshes\nbefore:\n%s\n\nafter:\n%s", bodyBefore, bodyAfter)
	}
}

func TestLoadCheckedInMutabilityOverlayDocsFixture(t *testing.T) {
	t.Parallel()

	target := newMutabilityOverlayDocsTargetForTest(
		"nosql",
		"Table",
		"table",
		"oci_nosql_table",
		"contract-example-v1",
		"https://registry.terraform.io/providers/oracle/oci/contract-example-v1/docs/resources/nosql_table",
	)
	root := filepath.Join(generatorTestDir(t), "testdata", "mutability_overlay", "docs")

	loaded, err := loadMutabilityOverlayDocsFixtures(root, []mutabilityOverlayRegistryPageTarget{target})
	if err != nil {
		t.Fatalf("loadMutabilityOverlayDocsFixtures(%q) error = %v", root, err)
	}
	if len(loaded) != 1 {
		t.Fatalf("loadMutabilityOverlayDocsFixtures(%q) returned %d inputs, want 1", root, len(loaded))
	}
	if got := loaded[0].Metadata.InputIdentity; got != "fixture:contract-example-v1/nosql/table/page.html" {
		t.Fatalf("loaded[0].Metadata.InputIdentity = %q, want %q", got, "fixture:contract-example-v1/nosql/table/page.html")
	}
	if !strings.Contains(loaded[0].Body, "Argument Reference") {
		t.Fatalf("loaded[0].Body = %q, want Argument Reference fixture content", loaded[0].Body)
	}
}

func TestMutabilityOverlayHTTPDocsFetcherDoesNotFollowRedirects(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/renamed", http.StatusMovedPermanently)
	}))
	t.Cleanup(server.Close)

	client := server.Client()
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	fetcher := newMutabilityOverlayHTTPDocsFetcher(client)

	resp, err := fetcher.Fetch(context.Background(), server.URL+"/original")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("Fetch() status = %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
	}
	if got := resp.Location; got != "/renamed" {
		t.Fatalf("Fetch() location = %q, want %q", got, "/renamed")
	}
}

type stubMutabilityOverlayDocsFetcher struct {
	responses map[string]mutabilityOverlayDocsHTTPResponse
	errors    map[string]error
}

func (f stubMutabilityOverlayDocsFetcher) Fetch(_ context.Context, url string) (mutabilityOverlayDocsHTTPResponse, error) {
	if err, ok := f.errors[url]; ok && err != nil {
		return mutabilityOverlayDocsHTTPResponse{}, err
	}
	if resp, ok := f.responses[url]; ok {
		return resp, nil
	}
	return mutabilityOverlayDocsHTTPResponse{}, errors.New("unexpected docs fetch URL: " + url)
}

func newMutabilityOverlayDocsTargetForTest(
	service string,
	kind string,
	formalSlug string,
	providerResource string,
	version string,
	registryURL string,
) mutabilityOverlayRegistryPageTarget {
	return mutabilityOverlayRegistryPageTarget{
		Service:              service,
		Kind:                 kind,
		FormalSlug:           formalSlug,
		ProviderResource:     providerResource,
		TerraformDocsVersion: version,
		RegistryPath:         "providers/oracle/oci/" + version + "/docs/resources/" + strings.TrimPrefix(providerResource, "oci_"),
		RegistryURL:          registryURL,
	}
}

func readMutabilityOverlayDocsTestFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

package generator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"
)

const (
	mutabilityOverlayDocsInputSchemaVersion = 1
	mutabilityOverlayDocsInputSurface       = "generator-mutability-overlay-docs-input"

	mutabilityOverlayDocsInputSourceLive    = "liveFetch"
	mutabilityOverlayDocsInputSourceFixture = "fixture"

	mutabilityOverlayDocsPageShapeArgumentReferenceHTML = "argumentReferenceHTML"

	mutabilityOverlayDocsFixtureMetadataFile = "metadata.json"
	mutabilityOverlayDocsFixtureBodyFile     = "page.html"

	mutabilityOverlayDocsErrorMissingPage             = "missingPage"
	mutabilityOverlayDocsErrorRedirectedPage          = "redirectedPage"
	mutabilityOverlayDocsErrorRateLimited             = "rateLimited"
	mutabilityOverlayDocsErrorAvailabilityFailure     = "availabilityFailure"
	mutabilityOverlayDocsErrorUnexpectedContentType   = "unexpectedContentType"
	mutabilityOverlayDocsErrorJavaScriptOnlyPage      = "javascriptOnlyPage"
	mutabilityOverlayDocsErrorStructurallyMissingPage = "structurallyMissingPage"
	mutabilityOverlayDocsErrorMissingFixture          = "missingFixture"
	mutabilityOverlayDocsErrorInvalidFixture          = "invalidFixture"
)

var (
	mutabilityOverlayDocsSupportedSources = []string{
		mutabilityOverlayDocsInputSourceLive,
		mutabilityOverlayDocsInputSourceFixture,
	}
)

// mutabilityOverlayDocsInputMetadata captures one deterministic raw page input
// for the Terraform docs mutability overlay. The same metadata shape is used for
// live fetch results and checked-in fixtures so parser tests can consume the
// exact on-disk format without touching the network.
type mutabilityOverlayDocsInputMetadata struct {
	SchemaVersion        int    `json:"schemaVersion"`
	Surface              string `json:"surface"`
	Service              string `json:"service"`
	Kind                 string `json:"kind"`
	FormalSlug           string `json:"formalSlug,omitempty"`
	ProviderResource     string `json:"providerResource"`
	TerraformDocsVersion string `json:"terraformDocsVersion"`
	RegistryPath         string `json:"registryPath"`
	RegistryURL          string `json:"registryURL"`
	ContentType          string `json:"contentType"`
	PageShape            string `json:"pageShape"`
	InputSource          string `json:"inputSource"`
	InputIdentity        string `json:"inputIdentity"`
	BodySHA256           string `json:"bodySHA256"`
	BodyBytes            int    `json:"bodyBytes"`
	FixtureBodyPath      string `json:"fixtureBodyPath,omitempty"`
}

// mutabilityOverlayDocsInput carries raw HTML plus its deterministic metadata.
type mutabilityOverlayDocsInput struct {
	Metadata mutabilityOverlayDocsInputMetadata
	Body     string
}

// mutabilityOverlayDocsFetcher abstracts live HTTP retrieval so unit tests can
// drive deterministic inputs without real network access.
type mutabilityOverlayDocsFetcher interface {
	Fetch(ctx context.Context, url string) (mutabilityOverlayDocsHTTPResponse, error)
}

// mutabilityOverlayDocsHTTPResponse is the transport-neutral live fetch result
// consumed by the acquisition layer.
type mutabilityOverlayDocsHTTPResponse struct {
	StatusCode  int
	ContentType string
	Location    string
	Body        []byte
}

type mutabilityOverlayHTTPDocsFetcher struct {
	client *http.Client
}

// mutabilityOverlayDocsAcquisitionError reports one typed live-fetch or
// fixture-load failure with enough resource context to debug the broken input.
type mutabilityOverlayDocsAcquisitionError struct {
	Reason           string
	Service          string
	Kind             string
	FormalSlug       string
	ProviderResource string
	RegistryURL      string
	StatusCode       int
	ContentType      string
	Location         string
	Detail           string
}

func (e *mutabilityOverlayDocsAcquisitionError) Error() string {
	if e == nil {
		return "<nil>"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "mutability overlay docs acquisition failed for service %q kind %q", e.Service, e.Kind)
	if strings.TrimSpace(e.FormalSlug) != "" {
		fmt.Fprintf(&b, " formalSpec %q", e.FormalSlug)
	}
	if strings.TrimSpace(e.ProviderResource) != "" {
		fmt.Fprintf(&b, " providerResource=%q", e.ProviderResource)
	}
	if strings.TrimSpace(e.RegistryURL) != "" {
		fmt.Fprintf(&b, " url=%q", e.RegistryURL)
	}
	fmt.Fprintf(&b, ": %s", e.Reason)
	if e.StatusCode != 0 {
		fmt.Fprintf(&b, " status=%d", e.StatusCode)
	}
	if strings.TrimSpace(e.ContentType) != "" {
		fmt.Fprintf(&b, " contentType=%q", e.ContentType)
	}
	if strings.TrimSpace(e.Location) != "" {
		fmt.Fprintf(&b, " location=%q", e.Location)
	}
	if strings.TrimSpace(e.Detail) != "" {
		fmt.Fprintf(&b, " (%s)", e.Detail)
	}
	return b.String()
}

func newMutabilityOverlayHTTPDocsFetcher(client *http.Client) mutabilityOverlayDocsFetcher {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	if client.CheckRedirect == nil {
		client = cloneMutabilityOverlayHTTPClient(client)
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return &mutabilityOverlayHTTPDocsFetcher{client: client}
}

func cloneMutabilityOverlayHTTPClient(client *http.Client) *http.Client {
	cloned := *client
	return &cloned
}

func (f *mutabilityOverlayHTTPDocsFetcher) Fetch(ctx context.Context, targetURL string) (mutabilityOverlayDocsHTTPResponse, error) {
	if f == nil || f.client == nil {
		return mutabilityOverlayDocsHTTPResponse{}, errors.New("mutability overlay docs fetcher is nil")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return mutabilityOverlayDocsHTTPResponse{}, fmt.Errorf("build docs request %q: %w", targetURL, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return mutabilityOverlayDocsHTTPResponse{}, fmt.Errorf("execute docs request %q: %w", targetURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mutabilityOverlayDocsHTTPResponse{}, fmt.Errorf("read docs response body %q: %w", targetURL, err)
	}

	return mutabilityOverlayDocsHTTPResponse{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Location:    resp.Header.Get("Location"),
		Body:        body,
	}, nil
}

// acquireMutabilityOverlayDocsInputs fetches the live Terraform Registry pages
// selected by yo1.2 and turns them into deterministic raw inputs. The caller is
// expected to persist successful results via refreshMutabilityOverlayDocsFixtures
// when checked-in fixtures should be updated.
func acquireMutabilityOverlayDocsInputs(
	ctx context.Context,
	targets []mutabilityOverlayRegistryPageTarget,
	fetcher mutabilityOverlayDocsFetcher,
) ([]mutabilityOverlayDocsInput, error) {
	if fetcher == nil {
		return nil, errors.New("mutability overlay docs fetcher is required")
	}

	sortedTargets := sortMutabilityOverlayRegistryPageTargets(targets)
	inputs := make([]mutabilityOverlayDocsInput, 0, len(sortedTargets))
	errs := make([]error, 0)
	for _, target := range sortedTargets {
		if err := validateMutabilityOverlayRegistryPageTargetForDocs(target); err != nil {
			errs = append(errs, err)
			continue
		}

		resp, err := fetcher.Fetch(ctx, target.RegistryURL)
		if err != nil {
			errs = append(errs, &mutabilityOverlayDocsAcquisitionError{
				Reason:           mutabilityOverlayDocsErrorAvailabilityFailure,
				Service:          target.Service,
				Kind:             target.Kind,
				FormalSlug:       target.FormalSlug,
				ProviderResource: target.ProviderResource,
				RegistryURL:      target.RegistryURL,
				Detail:           err.Error(),
			})
			continue
		}

		input, err := classifyMutabilityOverlayDocsResponse(target, resp)
		if err != nil {
			fallbackInput, ok, fallbackErr := tryAcquireMutabilityOverlayDocsInputFromProviderMarkdown(ctx, target, fetcher, err)
			switch {
			case ok && fallbackErr == nil:
				inputs = append(inputs, fallbackInput)
				continue
			case ok && fallbackErr != nil:
				errs = append(errs, errors.Join(err, fallbackErr))
				continue
			}
			errs = append(errs, err)
			continue
		}
		inputs = append(inputs, input)
	}

	if len(errs) != 0 {
		return inputs, errors.Join(errs...)
	}
	return inputs, nil
}

// refreshMutabilityOverlayDocsFixtures fetches live pages, refuses partial
// writes on typed fetch failures, and persists the successful deterministic
// fixture format to the provided root for later parser tests.
func refreshMutabilityOverlayDocsFixtures(
	ctx context.Context,
	root string,
	targets []mutabilityOverlayRegistryPageTarget,
	fetcher mutabilityOverlayDocsFetcher,
) ([]mutabilityOverlayDocsInput, error) {
	inputs, err := acquireMutabilityOverlayDocsInputs(ctx, targets, fetcher)
	if err != nil {
		return nil, err
	}
	return writeMutabilityOverlayDocsFixtures(root, inputs)
}

// loadMutabilityOverlayDocsFixtures loads checked-in raw docs fixtures without
// performing any network access. Parser tests should use this path instead of
// live HTTP so a fixed docs version and fixed fixture set stay reproducible.
func loadMutabilityOverlayDocsFixtures(
	root string,
	targets []mutabilityOverlayRegistryPageTarget,
) ([]mutabilityOverlayDocsInput, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("mutability overlay docs fixture root is required")
	}

	sortedTargets := sortMutabilityOverlayRegistryPageTargets(targets)
	inputs := make([]mutabilityOverlayDocsInput, 0, len(sortedTargets))
	errs := make([]error, 0)
	for _, target := range sortedTargets {
		if err := validateMutabilityOverlayRegistryPageTargetForDocs(target); err != nil {
			errs = append(errs, err)
			continue
		}

		bodyRel := mutabilityOverlayDocsFixtureBodyRelativePath(target)
		metadataPath := filepath.Join(root, filepath.FromSlash(mutabilityOverlayDocsFixtureMetadataRelativePath(target)))
		bodyPath := filepath.Join(root, filepath.FromSlash(bodyRel))

		metadataContent, err := os.ReadFile(metadataPath)
		if err != nil {
			errs = append(errs, newMutabilityOverlayDocsFixtureError(target, mutabilityOverlayDocsErrorMissingFixture, err))
			continue
		}

		var metadata mutabilityOverlayDocsInputMetadata
		if err := json.Unmarshal(metadataContent, &metadata); err != nil {
			errs = append(errs, &mutabilityOverlayDocsAcquisitionError{
				Reason:           mutabilityOverlayDocsErrorInvalidFixture,
				Service:          target.Service,
				Kind:             target.Kind,
				FormalSlug:       target.FormalSlug,
				ProviderResource: target.ProviderResource,
				RegistryURL:      target.RegistryURL,
				Detail:           fmt.Sprintf("decode fixture metadata %q: %v", metadataPath, err),
			})
			continue
		}

		bodyContent, err := os.ReadFile(bodyPath)
		if err != nil {
			errs = append(errs, newMutabilityOverlayDocsFixtureError(target, mutabilityOverlayDocsErrorMissingFixture, err))
			continue
		}

		input := mutabilityOverlayDocsInput{
			Metadata: metadata,
			Body:     string(bodyContent),
		}
		if err := validateMutabilityOverlayDocsFixtureInput(target, input); err != nil {
			errs = append(errs, err)
			continue
		}
		inputs = append(inputs, input)
	}

	if len(errs) != 0 {
		return inputs, errors.Join(errs...)
	}
	return inputs, nil
}

func writeMutabilityOverlayDocsFixtures(root string, inputs []mutabilityOverlayDocsInput) ([]mutabilityOverlayDocsInput, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("mutability overlay docs fixture root is required")
	}

	sortedInputs := sortMutabilityOverlayDocsInputs(inputs)
	written := make([]mutabilityOverlayDocsInput, 0, len(sortedInputs))
	seenBodyPaths := make(map[string]struct{}, len(sortedInputs))
	for _, input := range sortedInputs {
		target := mutabilityOverlayRegistryPageTarget{
			Service:              input.Metadata.Service,
			Kind:                 input.Metadata.Kind,
			FormalSlug:           input.Metadata.FormalSlug,
			ProviderResource:     input.Metadata.ProviderResource,
			TerraformDocsVersion: input.Metadata.TerraformDocsVersion,
			RegistryPath:         input.Metadata.RegistryPath,
			RegistryURL:          input.Metadata.RegistryURL,
		}
		if err := validateMutabilityOverlayRegistryPageTargetForDocs(target); err != nil {
			return nil, err
		}

		bodyRel := mutabilityOverlayDocsFixtureBodyRelativePath(target)
		if _, exists := seenBodyPaths[bodyRel]; exists {
			return nil, fmt.Errorf("duplicate mutability overlay docs fixture body path %q", bodyRel)
		}
		seenBodyPaths[bodyRel] = struct{}{}

		fixtureInput, err := newMutabilityOverlayDocsInput(target, mutabilityOverlayDocsInputSourceFixture, "fixture:"+bodyRel, bodyRel, input.Metadata.ContentType, input.Body)
		if err != nil {
			return nil, err
		}

		metadataPath := filepath.Join(root, filepath.FromSlash(mutabilityOverlayDocsFixtureMetadataRelativePath(target)))
		bodyPath := filepath.Join(root, filepath.FromSlash(bodyRel))
		if err := os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
			return nil, fmt.Errorf("create mutability overlay docs fixture dir %q: %w", filepath.Dir(metadataPath), err)
		}
		if err := os.WriteFile(bodyPath, []byte(fixtureInput.Body), 0o644); err != nil {
			return nil, fmt.Errorf("write mutability overlay docs fixture body %q: %w", bodyPath, err)
		}

		metadataContent, err := json.MarshalIndent(fixtureInput.Metadata, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal mutability overlay docs fixture metadata %q: %w", metadataPath, err)
		}
		metadataContent = append(metadataContent, '\n')
		if err := os.WriteFile(metadataPath, metadataContent, 0o644); err != nil {
			return nil, fmt.Errorf("write mutability overlay docs fixture metadata %q: %w", metadataPath, err)
		}

		written = append(written, fixtureInput)
	}

	return written, nil
}

func classifyMutabilityOverlayDocsResponse(
	target mutabilityOverlayRegistryPageTarget,
	resp mutabilityOverlayDocsHTTPResponse,
) (mutabilityOverlayDocsInput, error) {
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return mutabilityOverlayDocsInput{}, &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorMissingPage,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      target.RegistryURL,
			StatusCode:       resp.StatusCode,
			Detail:           "registry page does not exist for the pinned docs version",
		}
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		return mutabilityOverlayDocsInput{}, &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorRedirectedPage,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      target.RegistryURL,
			StatusCode:       resp.StatusCode,
			Location:         strings.TrimSpace(resp.Location),
			Detail:           "registry redirected the resource page; the provider resource may have been renamed or moved",
		}
	case resp.StatusCode == http.StatusTooManyRequests:
		return mutabilityOverlayDocsInput{}, &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorRateLimited,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      target.RegistryURL,
			StatusCode:       resp.StatusCode,
			Detail:           "registry rate limited the docs request",
		}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return mutabilityOverlayDocsInput{}, &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorAvailabilityFailure,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      target.RegistryURL,
			StatusCode:       resp.StatusCode,
			Detail:           "registry returned an unavailable or unexpected status",
		}
	}

	mediaType, canonicalContentType, err := canonicalizeMutabilityOverlayDocsContentType(resp.ContentType)
	if err != nil || !isMutabilityOverlayHTMLMediaType(mediaType) {
		detail := "expected HTML content from the Terraform Registry page"
		if err != nil {
			detail = err.Error()
		}
		return mutabilityOverlayDocsInput{}, &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorUnexpectedContentType,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      target.RegistryURL,
			StatusCode:       resp.StatusCode,
			ContentType:      strings.TrimSpace(resp.ContentType),
			Detail:           detail,
		}
	}

	body := normalizeMutabilityOverlayDocsBody(resp.Body)
	if looksLikeMutabilityOverlayJavaScriptOnlyPage(body) {
		return mutabilityOverlayDocsInput{}, &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorJavaScriptOnlyPage,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      target.RegistryURL,
			StatusCode:       resp.StatusCode,
			ContentType:      canonicalContentType,
			Detail:           "page requires JavaScript and does not expose a deterministic raw HTML argument-reference surface",
		}
	}

	if err := validateMutabilityOverlayDocsHTMLShape(body); err != nil {
		return mutabilityOverlayDocsInput{}, &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorStructurallyMissingPage,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      target.RegistryURL,
			StatusCode:       resp.StatusCode,
			ContentType:      canonicalContentType,
			Detail:           err.Error(),
		}
	}

	return newMutabilityOverlayDocsInput(
		target,
		mutabilityOverlayDocsInputSourceLive,
		"fetch:"+target.RegistryURL,
		"",
		canonicalContentType,
		body,
	)
}

func newMutabilityOverlayDocsInput(
	target mutabilityOverlayRegistryPageTarget,
	inputSource string,
	inputIdentity string,
	fixtureBodyPath string,
	contentType string,
	body string,
) (mutabilityOverlayDocsInput, error) {
	if err := validateMutabilityOverlayRegistryPageTargetForDocs(target); err != nil {
		return mutabilityOverlayDocsInput{}, err
	}
	if !slicesContainString(mutabilityOverlayDocsSupportedSources, inputSource) {
		return mutabilityOverlayDocsInput{}, fmt.Errorf("mutability overlay docs inputSource %q is not supported", inputSource)
	}

	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return mutabilityOverlayDocsInput{}, errors.New("mutability overlay docs contentType is required")
	}
	body = normalizeMutabilityOverlayDocsBody([]byte(body))
	if body == "" {
		return mutabilityOverlayDocsInput{}, errors.New("mutability overlay docs body is required")
	}
	if strings.TrimSpace(inputIdentity) == "" {
		return mutabilityOverlayDocsInput{}, errors.New("mutability overlay docs inputIdentity is required")
	}
	if inputSource == mutabilityOverlayDocsInputSourceFixture && strings.TrimSpace(fixtureBodyPath) == "" {
		return mutabilityOverlayDocsInput{}, errors.New("mutability overlay docs fixtureBodyPath is required for fixture inputs")
	}

	sum := sha256.Sum256([]byte(body))
	metadata := mutabilityOverlayDocsInputMetadata{
		SchemaVersion:        mutabilityOverlayDocsInputSchemaVersion,
		Surface:              mutabilityOverlayDocsInputSurface,
		Service:              target.Service,
		Kind:                 target.Kind,
		FormalSlug:           target.FormalSlug,
		ProviderResource:     target.ProviderResource,
		TerraformDocsVersion: target.TerraformDocsVersion,
		RegistryPath:         target.RegistryPath,
		RegistryURL:          target.RegistryURL,
		ContentType:          contentType,
		PageShape:            mutabilityOverlayDocsPageShapeArgumentReferenceHTML,
		InputSource:          inputSource,
		InputIdentity:        inputIdentity,
		BodySHA256:           hex.EncodeToString(sum[:]),
		BodyBytes:            len(body),
		FixtureBodyPath:      strings.TrimSpace(fixtureBodyPath),
	}
	if err := validateMutabilityOverlayDocsInputMetadata(metadata); err != nil {
		return mutabilityOverlayDocsInput{}, err
	}

	return mutabilityOverlayDocsInput{
		Metadata: metadata,
		Body:     body,
	}, nil
}

func tryAcquireMutabilityOverlayDocsInputFromProviderMarkdown(
	ctx context.Context,
	target mutabilityOverlayRegistryPageTarget,
	fetcher mutabilityOverlayDocsFetcher,
	cause error,
) (mutabilityOverlayDocsInput, bool, error) {
	var acquisitionErr *mutabilityOverlayDocsAcquisitionError
	if !errors.As(cause, &acquisitionErr) || acquisitionErr.Reason != mutabilityOverlayDocsErrorJavaScriptOnlyPage {
		return mutabilityOverlayDocsInput{}, false, nil
	}

	rawURL, ok := mutabilityOverlayProviderMarkdownRawURL(target)
	if !ok {
		return mutabilityOverlayDocsInput{}, false, nil
	}

	resp, err := fetcher.Fetch(ctx, rawURL)
	if err != nil {
		return mutabilityOverlayDocsInput{}, true, &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorAvailabilityFailure,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      rawURL,
			Detail:           fmt.Sprintf("fetch fallback provider markdown: %v", err),
		}
	}

	markdown, contentType, err := classifyMutabilityOverlayDocsMarkdownResponse(target, rawURL, resp)
	if err != nil {
		return mutabilityOverlayDocsInput{}, true, err
	}
	rendered, err := renderMutabilityOverlayDocsMarkdownArgumentReference(markdown)
	if err != nil {
		return mutabilityOverlayDocsInput{}, true, &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorStructurallyMissingPage,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      rawURL,
			StatusCode:       resp.StatusCode,
			ContentType:      contentType,
			Detail:           err.Error(),
		}
	}

	input, err := newMutabilityOverlayDocsInput(
		target,
		mutabilityOverlayDocsInputSourceLive,
		"fetch:"+rawURL,
		"",
		"text/html; charset=utf-8",
		rendered,
	)
	if err != nil {
		return mutabilityOverlayDocsInput{}, true, err
	}
	return input, true, nil
}

func mutabilityOverlayProviderMarkdownRawURL(target mutabilityOverlayRegistryPageTarget) (string, bool) {
	sourcePath := strings.TrimSpace(target.ProviderSourcePath)
	revision := strings.TrimSpace(target.ProviderSourceRevision)
	providerResource := strings.TrimSpace(target.ProviderResource)
	if sourcePath == "" || revision == "" || providerResource == "" {
		return "", false
	}

	sourcePath = strings.TrimPrefix(sourcePath, "https://")
	sourcePath = strings.TrimPrefix(sourcePath, "http://")
	if strings.HasPrefix(sourcePath, "github.com/") {
		sourcePath = strings.TrimPrefix(sourcePath, "github.com/")
	}
	if strings.Count(sourcePath, "/") != 1 {
		return "", false
	}

	resourceName := strings.TrimPrefix(providerResource, mutabilityOverlayProviderResourcePrefix)
	if resourceName == "" || resourceName == providerResource {
		return "", false
	}

	return fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/website/docs/r/%s.html.markdown",
		sourcePath,
		revision,
		resourceName,
	), true
}

func classifyMutabilityOverlayDocsMarkdownResponse(
	target mutabilityOverlayRegistryPageTarget,
	sourceURL string,
	resp mutabilityOverlayDocsHTTPResponse,
) (string, string, error) {
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return "", "", &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorMissingPage,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      sourceURL,
			StatusCode:       resp.StatusCode,
			Detail:           "fallback provider markdown does not exist for the pinned source revision",
		}
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		return "", "", &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorRedirectedPage,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      sourceURL,
			StatusCode:       resp.StatusCode,
			Location:         strings.TrimSpace(resp.Location),
			Detail:           "fallback provider markdown redirected unexpectedly",
		}
	case resp.StatusCode == http.StatusTooManyRequests:
		return "", "", &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorRateLimited,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      sourceURL,
			StatusCode:       resp.StatusCode,
			Detail:           "fallback provider markdown fetch was rate limited",
		}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return "", "", &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorAvailabilityFailure,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      sourceURL,
			StatusCode:       resp.StatusCode,
			Detail:           "fallback provider markdown returned an unavailable or unexpected status",
		}
	}

	mediaType, canonicalContentType, err := canonicalizeMutabilityOverlayDocsContentType(resp.ContentType)
	if err != nil || !isMutabilityOverlayMarkdownMediaType(mediaType) {
		detail := "expected markdown or plain text from the pinned provider docs source"
		if err != nil {
			detail = err.Error()
		}
		return "", "", &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorUnexpectedContentType,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      sourceURL,
			StatusCode:       resp.StatusCode,
			ContentType:      strings.TrimSpace(resp.ContentType),
			Detail:           detail,
		}
	}

	body := normalizeMutabilityOverlayDocsBody(resp.Body)
	if strings.TrimSpace(body) == "" {
		return "", "", &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorStructurallyMissingPage,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      sourceURL,
			StatusCode:       resp.StatusCode,
			ContentType:      canonicalContentType,
			Detail:           "fallback provider markdown body is empty",
		}
	}

	return body, canonicalContentType, nil
}

func isMutabilityOverlayMarkdownMediaType(mediaType string) bool {
	switch strings.TrimSpace(strings.ToLower(mediaType)) {
	case "text/markdown", "text/plain", "text/x-markdown":
		return true
	default:
		return false
	}
}

func renderMutabilityOverlayDocsMarkdownArgumentReference(markdown string) (string, error) {
	body, err := extractMutabilityOverlayDocsMarkdownSection(markdown, mutabilityOverlayDocsSectionArgumentReference)
	if err != nil {
		return "", err
	}

	introLines, listLines := splitMutabilityOverlayDocsMarkdownIntroAndList(body)
	items := parseMutabilityOverlayDocsMarkdownList(listLines)
	if len(items) == 0 {
		return "", errors.New("provider markdown Argument Reference section did not contain any list items")
	}

	var rendered strings.Builder
	rendered.WriteString("<!DOCTYPE html>\n")
	rendered.WriteString("<html>\n")
	rendered.WriteString("  <body>\n")
	rendered.WriteString(`    <h2 id="argument-reference">Argument Reference</h2>` + "\n")
	for _, paragraph := range renderMutabilityOverlayDocsMarkdownParagraphs(introLines) {
		rendered.WriteString("    ")
		rendered.WriteString(paragraph)
		rendered.WriteByte('\n')
	}
	renderMutabilityOverlayDocsMarkdownList(&rendered, items, "    ")
	rendered.WriteString("  </body>\n")
	rendered.WriteString("</html>\n")
	return rendered.String(), nil
}

func extractMutabilityOverlayDocsMarkdownSection(markdown string, title string) (string, error) {
	lines := strings.Split(normalizeMutabilityOverlayDocsBody([]byte(markdown)), "\n")
	title = strings.TrimSpace(title)
	if title == "" {
		return "", errors.New("markdown section title is required")
	}

	start := -1
	level := 0
	for i, line := range lines {
		headingLevel, headingTitle, ok := mutabilityOverlayDocsMarkdownHeading(line)
		if ok && headingTitle == title {
			start = i + 1
			level = headingLevel
			break
		}
	}
	if start == -1 {
		return "", fmt.Errorf("provider markdown does not contain a %q heading", title)
	}

	end := len(lines)
	for i := start; i < len(lines); i++ {
		headingLevel, _, ok := mutabilityOverlayDocsMarkdownHeading(lines[i])
		if ok && headingLevel <= level {
			end = i
			break
		}
	}

	section := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
	if section == "" {
		return "", fmt.Errorf("provider markdown section %q is empty", title)
	}
	return section, nil
}

func mutabilityOverlayDocsMarkdownHeading(line string) (int, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.HasPrefix(trimmed, "#") {
		return 0, "", false
	}

	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level == len(trimmed) || trimmed[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(trimmed[level:]), true
}

type mutabilityOverlayDocsMarkdownListItem struct {
	Text     string
	Children []*mutabilityOverlayDocsMarkdownListItem
}

func splitMutabilityOverlayDocsMarkdownIntroAndList(section string) ([]string, []string) {
	lines := strings.Split(section, "\n")
	firstBullet := len(lines)
	for i, line := range lines {
		if _, _, ok := parseMutabilityOverlayDocsMarkdownBullet(line); ok {
			firstBullet = i
			break
		}
	}
	return lines[:firstBullet], lines[firstBullet:]
}

func renderMutabilityOverlayDocsMarkdownParagraphs(lines []string) []string {
	var paragraphs []string
	var current []string
	flush := func() {
		if len(current) == 0 {
			return
		}
		paragraphs = append(paragraphs, "<p>"+renderMutabilityOverlayDocsMarkdownInline(strings.Join(current, " "))+"</p>")
		current = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			continue
		}
		current = append(current, trimmed)
	}
	flush()
	return paragraphs
}

func parseMutabilityOverlayDocsMarkdownList(lines []string) []*mutabilityOverlayDocsMarkdownListItem {
	var (
		root    []*mutabilityOverlayDocsMarkdownListItem
		parents []*mutabilityOverlayDocsMarkdownListItem
	)
	for _, line := range lines {
		indent, text, ok := parseMutabilityOverlayDocsMarkdownBullet(line)
		if ok {
			item := &mutabilityOverlayDocsMarkdownListItem{Text: strings.TrimSpace(text)}
			if indent <= 0 || len(parents) == 0 {
				root = append(root, item)
				parents = parents[:0]
			} else {
				if indent > len(parents) {
					indent = len(parents)
				}
				parents = parents[:indent]
				parent := parents[len(parents)-1]
				parent.Children = append(parent.Children, item)
			}
			parents = append(parents, item)
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || len(parents) == 0 {
			continue
		}
		current := parents[len(parents)-1]
		if current.Text != "" {
			current.Text += " "
		}
		current.Text += trimmed
	}
	return root
}

func parseMutabilityOverlayDocsMarkdownBullet(line string) (int, string, bool) {
	indent := 0
	for len(line) != 0 {
		switch {
		case strings.HasPrefix(line, "\t"):
			indent++
			line = line[1:]
		case strings.HasPrefix(line, "    "):
			indent++
			line = line[4:]
		default:
			goto parsed
		}
	}

parsed:
	line = strings.TrimLeft(line, " ")
	if !strings.HasPrefix(line, "* ") {
		return 0, "", false
	}
	return indent, strings.TrimSpace(line[2:]), true
}

func renderMutabilityOverlayDocsMarkdownList(
	b *strings.Builder,
	items []*mutabilityOverlayDocsMarkdownListItem,
	indent string,
) {
	if len(items) == 0 {
		return
	}

	b.WriteString(indent)
	b.WriteString("<ul>\n")
	for _, item := range items {
		b.WriteString(indent)
		b.WriteString("  <li>")
		b.WriteString(renderMutabilityOverlayDocsMarkdownInline(item.Text))
		if len(item.Children) != 0 {
			b.WriteByte('\n')
			renderMutabilityOverlayDocsMarkdownList(b, item.Children, indent+"    ")
			b.WriteString(indent)
			b.WriteString("  ")
		}
		b.WriteString("</li>\n")
	}
	b.WriteString(indent)
	b.WriteString("</ul>\n")
}

func renderMutabilityOverlayDocsMarkdownInline(text string) string {
	var rendered strings.Builder
	for len(text) != 0 {
		start := strings.IndexByte(text, '`')
		if start == -1 {
			rendered.WriteString(html.EscapeString(text))
			break
		}

		rendered.WriteString(html.EscapeString(text[:start]))
		text = text[start+1:]
		end := strings.IndexByte(text, '`')
		if end == -1 {
			rendered.WriteString("`")
			rendered.WriteString(html.EscapeString(text))
			break
		}

		rendered.WriteString("<code>")
		rendered.WriteString(html.EscapeString(text[:end]))
		rendered.WriteString("</code>")
		text = text[end+1:]
	}
	return rendered.String()
}

func validateMutabilityOverlayDocsFixtureInput(
	target mutabilityOverlayRegistryPageTarget,
	input mutabilityOverlayDocsInput,
) error {
	if err := validateMutabilityOverlayDocsInputMetadata(input.Metadata); err != nil {
		return &mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorInvalidFixture,
			Service:          target.Service,
			Kind:             target.Kind,
			FormalSlug:       target.FormalSlug,
			ProviderResource: target.ProviderResource,
			RegistryURL:      target.RegistryURL,
			Detail:           err.Error(),
		}
	}

	if input.Metadata.InputSource != mutabilityOverlayDocsInputSourceFixture {
		return newMutabilityOverlayDocsInvalidFixtureError(target, fmt.Sprintf("metadata inputSource = %q, want %q", input.Metadata.InputSource, mutabilityOverlayDocsInputSourceFixture))
	}
	wantBodyRel := mutabilityOverlayDocsFixtureBodyRelativePath(target)
	if input.Metadata.FixtureBodyPath != wantBodyRel {
		return newMutabilityOverlayDocsInvalidFixtureError(target, fmt.Sprintf("metadata fixtureBodyPath = %q, want %q", input.Metadata.FixtureBodyPath, wantBodyRel))
	}
	if input.Metadata.InputIdentity != "fixture:"+wantBodyRel {
		return newMutabilityOverlayDocsInvalidFixtureError(target, fmt.Sprintf("metadata inputIdentity = %q, want %q", input.Metadata.InputIdentity, "fixture:"+wantBodyRel))
	}
	if input.Metadata.Service != target.Service ||
		input.Metadata.Kind != target.Kind ||
		input.Metadata.FormalSlug != target.FormalSlug ||
		input.Metadata.ProviderResource != target.ProviderResource ||
		input.Metadata.TerraformDocsVersion != target.TerraformDocsVersion ||
		input.Metadata.RegistryPath != target.RegistryPath ||
		input.Metadata.RegistryURL != target.RegistryURL {
		return newMutabilityOverlayDocsInvalidFixtureError(target, "fixture metadata does not match the expected registry page target")
	}
	recomputed := normalizeMutabilityOverlayDocsBody([]byte(input.Body))
	if recomputed != input.Body {
		return newMutabilityOverlayDocsInvalidFixtureError(target, "fixture body is not in canonical newline form")
	}
	sum := sha256.Sum256([]byte(input.Body))
	if input.Metadata.BodySHA256 != hex.EncodeToString(sum[:]) {
		return newMutabilityOverlayDocsInvalidFixtureError(target, fmt.Sprintf("metadata bodySHA256 = %q, want %q", input.Metadata.BodySHA256, hex.EncodeToString(sum[:])))
	}
	if input.Metadata.BodyBytes != len(input.Body) {
		return newMutabilityOverlayDocsInvalidFixtureError(target, fmt.Sprintf("metadata bodyBytes = %d, want %d", input.Metadata.BodyBytes, len(input.Body)))
	}
	return nil
}

func validateMutabilityOverlayDocsInputMetadata(metadata mutabilityOverlayDocsInputMetadata) error {
	var errs []string
	if metadata.SchemaVersion != mutabilityOverlayDocsInputSchemaVersion {
		errs = append(errs, fmt.Sprintf("schemaVersion = %d, want %d", metadata.SchemaVersion, mutabilityOverlayDocsInputSchemaVersion))
	}
	if got := strings.TrimSpace(metadata.Surface); got != mutabilityOverlayDocsInputSurface {
		errs = append(errs, fmt.Sprintf("surface = %q, want %q", got, mutabilityOverlayDocsInputSurface))
	}
	errs = append(errs, validateNonEmptyString("service", metadata.Service)...)
	errs = append(errs, validateNonEmptyString("kind", metadata.Kind)...)
	errs = append(errs, validateNonEmptyString("providerResource", metadata.ProviderResource)...)
	errs = append(errs, validateNonEmptyString("terraformDocsVersion", metadata.TerraformDocsVersion)...)
	errs = append(errs, validateNonEmptyString("registryPath", metadata.RegistryPath)...)
	errs = append(errs, validateNonEmptyString("registryURL", metadata.RegistryURL)...)
	errs = append(errs, validateNonEmptyString("contentType", metadata.ContentType)...)
	errs = append(errs, validateNonEmptyString("pageShape", metadata.PageShape)...)
	errs = append(errs, validateNonEmptyString("inputSource", metadata.InputSource)...)
	errs = append(errs, validateNonEmptyString("inputIdentity", metadata.InputIdentity)...)
	errs = append(errs, validateNonEmptyString("bodySHA256", metadata.BodySHA256)...)
	if metadata.BodyBytes <= 0 {
		errs = append(errs, fmt.Sprintf("bodyBytes = %d, want > 0", metadata.BodyBytes))
	}
	if metadata.PageShape != mutabilityOverlayDocsPageShapeArgumentReferenceHTML {
		errs = append(errs, fmt.Sprintf("pageShape = %q, want %q", metadata.PageShape, mutabilityOverlayDocsPageShapeArgumentReferenceHTML))
	}
	if !slicesContainString(mutabilityOverlayDocsSupportedSources, metadata.InputSource) {
		errs = append(errs, fmt.Sprintf("inputSource = %q, want one of %v", metadata.InputSource, mutabilityOverlayDocsSupportedSources))
	}
	if metadata.InputSource == mutabilityOverlayDocsInputSourceFixture {
		errs = append(errs, validateNonEmptyString("fixtureBodyPath", metadata.FixtureBodyPath)...)
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

func validateMutabilityOverlayRegistryPageTargetForDocs(target mutabilityOverlayRegistryPageTarget) error {
	var errs []string
	errs = append(errs, validateNonEmptyString("service", target.Service)...)
	errs = append(errs, validateNonEmptyString("kind", target.Kind)...)
	errs = append(errs, validateNonEmptyString("providerResource", target.ProviderResource)...)
	errs = append(errs, validateNonEmptyString("terraformDocsVersion", target.TerraformDocsVersion)...)
	errs = append(errs, validateNonEmptyString("registryPath", target.RegistryPath)...)
	errs = append(errs, validateNonEmptyString("registryURL", target.RegistryURL)...)
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

func normalizeMutabilityOverlayDocsBody(body []byte) string {
	normalized := strings.ReplaceAll(string(body), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return normalized
}

func canonicalizeMutabilityOverlayDocsContentType(contentType string) (string, string, error) {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return "", "", errors.New("content type is blank")
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", "", fmt.Errorf("parse content type %q: %w", contentType, err)
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	return mediaType, mime.FormatMediaType(mediaType, params), nil
}

func isMutabilityOverlayHTMLMediaType(mediaType string) bool {
	switch strings.TrimSpace(strings.ToLower(mediaType)) {
	case "text/html", "application/xhtml+xml":
		return true
	default:
		return false
	}
}

func looksLikeMutabilityOverlayJavaScriptOnlyPage(body string) bool {
	lower := strings.ToLower(body)
	if strings.Contains(lower, "argument reference") {
		return false
	}

	switch {
	case strings.Contains(lower, "enable javascript"),
		strings.Contains(lower, "requires javascript"),
		strings.Contains(lower, "javascript is required"),
		strings.Contains(lower, "please turn javascript on"):
		return true
	default:
		return false
	}
}

func validateMutabilityOverlayDocsHTMLShape(body string) error {
	doc, err := xhtml.Parse(strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("parse HTML: %w", err)
	}

	var hasHTML bool
	var hasBody bool
	var hasArgumentReference bool
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node == nil {
			return
		}
		if node.Type == xhtml.ElementNode {
			switch node.Data {
			case "html":
				hasHTML = true
			case "body":
				hasBody = true
			}
		}
		if node.Type == xhtml.TextNode {
			text := strings.TrimSpace(node.Data)
			if text != "" && strings.Contains(strings.ToLower(text), "argument reference") {
				hasArgumentReference = true
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	switch {
	case !hasHTML:
		return errors.New("HTML document is missing the root html element")
	case !hasBody:
		return errors.New("HTML document is missing the body element")
	case !hasArgumentReference:
		return errors.New(`HTML document does not contain an "Argument Reference" section`)
	default:
		return nil
	}
}

func sortMutabilityOverlayRegistryPageTargets(targets []mutabilityOverlayRegistryPageTarget) []mutabilityOverlayRegistryPageTarget {
	sorted := append([]mutabilityOverlayRegistryPageTarget(nil), targets...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].TerraformDocsVersion != sorted[j].TerraformDocsVersion {
			return sorted[i].TerraformDocsVersion < sorted[j].TerraformDocsVersion
		}
		if sorted[i].Service != sorted[j].Service {
			return sorted[i].Service < sorted[j].Service
		}
		if sorted[i].Kind != sorted[j].Kind {
			return sorted[i].Kind < sorted[j].Kind
		}
		if sorted[i].ProviderResource != sorted[j].ProviderResource {
			return sorted[i].ProviderResource < sorted[j].ProviderResource
		}
		return sorted[i].RegistryURL < sorted[j].RegistryURL
	})
	return sorted
}

func sortMutabilityOverlayDocsInputs(inputs []mutabilityOverlayDocsInput) []mutabilityOverlayDocsInput {
	sorted := append([]mutabilityOverlayDocsInput(nil), inputs...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Metadata.TerraformDocsVersion != sorted[j].Metadata.TerraformDocsVersion {
			return sorted[i].Metadata.TerraformDocsVersion < sorted[j].Metadata.TerraformDocsVersion
		}
		if sorted[i].Metadata.Service != sorted[j].Metadata.Service {
			return sorted[i].Metadata.Service < sorted[j].Metadata.Service
		}
		if sorted[i].Metadata.Kind != sorted[j].Metadata.Kind {
			return sorted[i].Metadata.Kind < sorted[j].Metadata.Kind
		}
		if sorted[i].Metadata.ProviderResource != sorted[j].Metadata.ProviderResource {
			return sorted[i].Metadata.ProviderResource < sorted[j].Metadata.ProviderResource
		}
		return sorted[i].Metadata.RegistryURL < sorted[j].Metadata.RegistryURL
	})
	return sorted
}

func mutabilityOverlayDocsFixtureMetadataRelativePath(target mutabilityOverlayRegistryPageTarget) string {
	return path.Join(mutabilityOverlayDocsFixtureRelativeDir(target), mutabilityOverlayDocsFixtureMetadataFile)
}

func mutabilityOverlayDocsFixtureBodyRelativePath(target mutabilityOverlayRegistryPageTarget) string {
	return path.Join(mutabilityOverlayDocsFixtureRelativeDir(target), mutabilityOverlayDocsFixtureBodyFile)
}

func mutabilityOverlayDocsFixtureRelativeDir(target mutabilityOverlayRegistryPageTarget) string {
	slug := strings.TrimSpace(target.FormalSlug)
	if slug == "" {
		slug = normalizeMutabilityOverlayDocsFixtureToken(target.Kind)
	}
	return path.Join(target.TerraformDocsVersion, target.Service, slug)
}

func normalizeMutabilityOverlayDocsFixtureToken(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func newMutabilityOverlayDocsFixtureError(
	target mutabilityOverlayRegistryPageTarget,
	reason string,
	err error,
) error {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	return &mutabilityOverlayDocsAcquisitionError{
		Reason:           reason,
		Service:          target.Service,
		Kind:             target.Kind,
		FormalSlug:       target.FormalSlug,
		ProviderResource: target.ProviderResource,
		RegistryURL:      target.RegistryURL,
		Detail:           detail,
	}
}

func newMutabilityOverlayDocsInvalidFixtureError(target mutabilityOverlayRegistryPageTarget, detail string) error {
	return &mutabilityOverlayDocsAcquisitionError{
		Reason:           mutabilityOverlayDocsErrorInvalidFixture,
		Service:          target.Service,
		Kind:             target.Kind,
		FormalSlug:       target.FormalSlug,
		ProviderResource: target.ProviderResource,
		RegistryURL:      target.RegistryURL,
		Detail:           detail,
	}
}

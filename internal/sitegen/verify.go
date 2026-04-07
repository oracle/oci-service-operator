/*
 Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
 Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sitegen

import (
	"fmt"
	stdhtml "html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"

	xhtml "golang.org/x/net/html"
)

var (
	markdownLinkRE = regexp.MustCompile(`(!?)\[[^\]]+\]\(([^)]+)\)`)
	htmlTagRE      = regexp.MustCompile(`<[^>]+>`)
)

// VerifyOptions configures repo-local docs verification.
type VerifyOptions struct {
	RepoRoot                 string
	SiteDir                  string
	StrictPublicDescriptions bool
}

// VerifyResult reports non-fatal warnings emitted by docs verification.
type VerifyResult struct {
	DescriptionWarnings []string
}

// VerifyDocs checks generated docs drift, source/build link integrity, and description coverage.
func VerifyDocs(opts VerifyOptions) (*VerifyResult, error) {
	repoRoot, err := ResolveRepoRoot(opts.RepoRoot)
	if err != nil {
		return nil, err
	}

	siteDir := strings.TrimSpace(opts.SiteDir)
	if siteDir == "" {
		siteDir = filepath.Join(repoRoot, "site")
	} else if !filepath.IsAbs(siteDir) {
		siteDir = filepath.Join(repoRoot, siteDir)
	}

	if err := VerifyGeneratedDocsMatchRepo(repoRoot); err != nil {
		return nil, err
	}
	if err := verifyMarkdownSourceLinks(repoRoot, []string{"README.md", "docs/README.md"}); err != nil {
		return nil, err
	}
	if err := VerifyBuiltSiteLinks(siteDir); err != nil {
		return nil, err
	}

	descriptionWarnings, err := verifyDescriptionCoverage(repoRoot, opts.StrictPublicDescriptions)
	if err != nil {
		return nil, err
	}

	return &VerifyResult{DescriptionWarnings: descriptionWarnings}, nil
}

// VerifyGeneratedDocsMatchRepo renders docs/reference into a temp tree and checks it against the repo.
func VerifyGeneratedDocsMatchRepo(repoRoot string) error {
	repoRoot, err := ResolveRepoRoot(repoRoot)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "osok-sitegen-verify-*")
	if err != nil {
		return fmt.Errorf("create temp dir for docs verification: %w", err)
	}
	defer os.RemoveAll(tempDir)

	result, err := GenerateReferenceDocs(GenerateOptions{
		Root:       repoRoot,
		OutputRoot: tempDir,
	})
	if err != nil {
		return fmt.Errorf("generate reference docs for verification: %w", err)
	}

	checkedInPaths, err := generatedDocsPaths(repoRoot)
	if err != nil {
		return err
	}
	expectedPaths := append([]string{}, result.Written...)
	sort.Strings(expectedPaths)
	if !slices.Equal(expectedPaths, checkedInPaths) {
		return fmt.Errorf(
			"checked-in docs/reference outputs differ from cmd/sitegen output (expected %d files, found %d); run 'make docs-generate'",
			len(expectedPaths),
			len(checkedInPaths),
		)
	}

	for _, relPath := range expectedPaths {
		wantPath := filepath.Join(tempDir, filepath.FromSlash(relPath))
		gotPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))
		want, err := os.ReadFile(wantPath)
		if err != nil {
			return fmt.Errorf("read generated verification file %q: %w", wantPath, err)
		}
		got, err := os.ReadFile(gotPath)
		if err != nil {
			return fmt.Errorf("read checked-in generated file %q: %w", gotPath, err)
		}
		if string(want) != string(got) {
			return fmt.Errorf("checked-in generated doc %q is stale; run 'make docs-generate'", relPath)
		}
	}

	return nil
}

// VerifyBuiltSiteLinks checks rendered internal links and anchors in the built MkDocs site.
func VerifyBuiltSiteLinks(siteDir string) error {
	siteDir = filepath.Clean(siteDir)
	info, err := os.Stat(siteDir)
	if err != nil {
		return fmt.Errorf("stat built site directory %q: %w", siteDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("built site path %q is not a directory", siteDir)
	}

	pages := make(map[string]builtPage)
	if err := filepath.WalkDir(siteDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read built page %q: %w", path, err)
		}
		hrefs, ids, err := parseBuiltPage(content)
		if err != nil {
			return fmt.Errorf("parse built page %q: %w", path, err)
		}
		pages[path] = builtPage{
			Hrefs: hrefs,
			IDs:   ids,
		}
		return nil
	}); err != nil {
		return err
	}

	var issues []string
	for currentPath, page := range pages {
		currentRel := filepath.ToSlash(mustRel(siteDir, currentPath))
		for _, href := range page.Hrefs {
			targetPath, targetAnchor, external, err := resolveBuiltHref(siteDir, currentPath, href)
			if err != nil {
				issues = append(issues, fmt.Sprintf("%s -> %s: %v", currentRel, href, err))
				continue
			}
			if external {
				continue
			}

			targetPage, ok := pages[targetPath]
			if !ok {
				if _, statErr := os.Stat(targetPath); statErr == nil {
					continue
				}
				targetRel := filepath.ToSlash(mustRel(siteDir, targetPath))
				issues = append(issues, fmt.Sprintf("%s -> %s: target %s does not exist", currentRel, href, targetRel))
				continue
			}
			if targetAnchor != "" {
				if _, found := targetPage.IDs[targetAnchor]; !found {
					targetRel := filepath.ToSlash(mustRel(siteDir, targetPath))
					issues = append(issues, fmt.Sprintf("%s -> %s: anchor #%s not found in %s", currentRel, href, targetAnchor, targetRel))
				}
			}
		}
	}

	if len(issues) > 0 {
		sort.Strings(issues)
		return fmt.Errorf("built docs link validation failed:\n- %s", strings.Join(issues, "\n- "))
	}

	return nil
}

type builtPage struct {
	Hrefs []string
	IDs   map[string]struct{}
}

func generatedDocsPaths(root string) ([]string, error) {
	referenceRoot := filepath.Join(root, "docs", "reference")
	var paths []string
	if err := filepath.WalkDir(referenceRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk checked-in docs/reference outputs: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func parseBuiltPage(content []byte) ([]string, map[string]struct{}, error) {
	doc, err := xhtml.Parse(strings.NewReader(string(content)))
	if err != nil {
		return nil, nil, err
	}

	hrefs := make([]string, 0, 16)
	ids := make(map[string]struct{})
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode {
			for _, attr := range node.Attr {
				if attr.Key == "id" {
					ids[strings.TrimSpace(attr.Val)] = struct{}{}
				}
				if node.Data == "a" && attr.Key == "href" {
					target := stdhtml.UnescapeString(strings.TrimSpace(attr.Val))
					if target != "" {
						hrefs = append(hrefs, target)
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	return hrefs, ids, nil
}

func resolveBuiltHref(siteDir string, currentPath string, href string) (string, string, bool, error) {
	parsed, err := url.Parse(href)
	if err != nil {
		return "", "", false, fmt.Errorf("parse href: %w", err)
	}
	if parsed.Scheme != "" || parsed.Host != "" || strings.HasPrefix(href, "//") {
		return "", "", true, nil
	}

	targetPath := filepath.Clean(currentPath)
	if parsed.Path != "" {
		if strings.HasPrefix(parsed.Path, "/") {
			targetPath = resolveAbsoluteBuiltPath(siteDir, parsed.Path)
		} else {
			targetPath = filepath.Join(filepath.Dir(currentPath), filepath.FromSlash(parsed.Path))
		}
		targetPath = filepath.Clean(targetPath)
	}

	if err := ensureWithinRoot(siteDir, targetPath); err != nil {
		return "", "", false, err
	}

	resolvedPath, err := resolveBuiltTargetPath(targetPath)
	if err != nil {
		return "", "", false, err
	}

	return resolvedPath, parsed.Fragment, false, nil
}

func resolveAbsoluteBuiltPath(siteDir string, rawPath string) string {
	trimmed := strings.TrimPrefix(rawPath, "/")
	candidates := []string{
		filepath.Join(siteDir, filepath.FromSlash(trimmed)),
	}
	if first, rest, found := strings.Cut(trimmed, "/"); found && first != "" {
		candidates = append(candidates, filepath.Join(siteDir, filepath.FromSlash(rest)))
	}

	for _, candidate := range candidates {
		if pathExists(candidate) {
			return candidate
		}
		if filepath.Ext(candidate) == "" {
			if pathExists(filepath.Join(candidate, "index.html")) || pathExists(candidate+".html") {
				return candidate
			}
		}
	}

	return candidates[len(candidates)-1]
}

func resolveBuiltTargetPath(targetPath string) (string, error) {
	if info, err := os.Stat(targetPath); err == nil {
		if info.IsDir() {
			indexPath := filepath.Join(targetPath, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				return indexPath, nil
			}
		}
		return targetPath, nil
	}

	if filepath.Ext(targetPath) == "" {
		indexPath := filepath.Join(targetPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			return indexPath, nil
		}
		htmlPath := targetPath + ".html"
		if _, err := os.Stat(htmlPath); err == nil {
			return htmlPath, nil
		}
	}

	return targetPath, nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func verifyMarkdownSourceLinks(repoRoot string, relPaths []string) error {
	anchorCache := make(map[string]map[string]struct{})
	var issues []string
	for _, relPath := range relPaths {
		relPath = filepath.ToSlash(relPath)
		absPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))
		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("read markdown source %q: %w", absPath, err)
		}

		for _, link := range extractMarkdownLinks(string(content)) {
			targetRel, targetAnchor, external, err := resolveMarkdownTarget(repoRoot, relPath, link)
			if err != nil {
				issues = append(issues, fmt.Sprintf("%s -> %s: %v", relPath, link, err))
				continue
			}
			if external {
				continue
			}

			targetAbs := filepath.Join(repoRoot, filepath.FromSlash(targetRel))
			if targetAnchor == "" {
				if _, err := os.Stat(targetAbs); err != nil {
					issues = append(issues, fmt.Sprintf("%s -> %s: target %s does not exist", relPath, link, targetRel))
				}
				continue
			}

			if filepath.Ext(targetAbs) != ".md" {
				if _, err := os.Stat(targetAbs); err != nil {
					issues = append(issues, fmt.Sprintf("%s -> %s: target %s does not exist", relPath, link, targetRel))
				}
				continue
			}

			anchors, err := markdownAnchors(repoRoot, targetRel, anchorCache)
			if err != nil {
				return err
			}
			if _, found := anchors[targetAnchor]; !found {
				issues = append(issues, fmt.Sprintf("%s -> %s: anchor #%s not found in %s", relPath, link, targetAnchor, targetRel))
			}
		}
	}

	if len(issues) > 0 {
		sort.Strings(issues)
		return fmt.Errorf("markdown source link validation failed:\n- %s", strings.Join(issues, "\n- "))
	}
	return nil
}

func markdownAnchors(repoRoot string, relPath string, cache map[string]map[string]struct{}) (map[string]struct{}, error) {
	if anchors, found := cache[relPath]; found {
		return anchors, nil
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relPath)))
	if err != nil {
		return nil, fmt.Errorf("read markdown anchors from %q: %w", relPath, err)
	}

	anchors := extractMarkdownAnchors(string(content))
	cache[relPath] = anchors
	return anchors, nil
}

func extractMarkdownLinks(content string) []string {
	lines := strings.Split(content, "\n")
	links := make([]string, 0, 16)
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		for _, match := range markdownLinkRE.FindAllStringSubmatch(line, -1) {
			if len(match) < 3 || match[1] == "!" {
				continue
			}
			target := normalizeMarkdownLinkTarget(match[2])
			if target != "" {
				links = append(links, target)
			}
		}
	}
	return links
}

func normalizeMarkdownLinkTarget(raw string) string {
	target := strings.TrimSpace(raw)
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "<") {
		if idx := strings.Index(target, ">"); idx > 0 {
			return strings.TrimSpace(target[1:idx])
		}
	}
	for idx, r := range target {
		if unicode.IsSpace(r) {
			return strings.TrimSpace(target[:idx])
		}
	}
	return target
}

func resolveMarkdownTarget(repoRoot string, currentRel string, target string) (string, string, bool, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return "", "", false, fmt.Errorf("parse markdown link: %w", err)
	}
	if parsed.Scheme != "" || parsed.Host != "" || strings.HasPrefix(target, "//") {
		return "", "", true, nil
	}

	pathPart := parsed.Path
	var absTarget string
	switch {
	case pathPart == "":
		absTarget = filepath.Join(repoRoot, filepath.FromSlash(currentRel))
	case strings.HasPrefix(pathPart, "/"):
		absTarget = filepath.Join(repoRoot, filepath.FromSlash(strings.TrimPrefix(pathPart, "/")))
	default:
		absTarget = filepath.Join(filepath.Dir(filepath.Join(repoRoot, filepath.FromSlash(currentRel))), filepath.FromSlash(pathPart))
	}
	absTarget = filepath.Clean(absTarget)
	if err := ensureWithinRoot(repoRoot, absTarget); err != nil {
		return "", "", false, err
	}

	resolvedTarget := absTarget
	if info, err := os.Stat(resolvedTarget); err == nil {
		if info.IsDir() {
			resolvedTarget = resolveMarkdownDirectoryTarget(resolvedTarget)
		}
	} else if filepath.Ext(resolvedTarget) == "" {
		switch {
		case fileExists(resolvedTarget + ".md"):
			resolvedTarget += ".md"
		case fileExists(filepath.Join(resolvedTarget, "index.md")):
			resolvedTarget = filepath.Join(resolvedTarget, "index.md")
		case fileExists(filepath.Join(resolvedTarget, "README.md")):
			resolvedTarget = filepath.Join(resolvedTarget, "README.md")
		}
	}

	relTarget, err := filepath.Rel(repoRoot, resolvedTarget)
	if err != nil {
		return "", "", false, fmt.Errorf("resolve relative markdown target for %q: %w", target, err)
	}

	return filepath.ToSlash(relTarget), parsed.Fragment, false, nil
}

func resolveMarkdownDirectoryTarget(dir string) string {
	switch {
	case fileExists(filepath.Join(dir, "index.md")):
		return filepath.Join(dir, "index.md")
	case fileExists(filepath.Join(dir, "README.md")):
		return filepath.Join(dir, "README.md")
	default:
		return dir
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func extractMarkdownAnchors(content string) map[string]struct{} {
	anchors := make(map[string]struct{})
	seen := make(map[string]int)
	lines := strings.Split(content, "\n")
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		for _, id := range explicitHTMLIDs(line) {
			anchors[id] = struct{}{}
		}

		if anchor := headingAnchor(trimmed, seen); anchor != "" {
			anchors[anchor] = struct{}{}
		}
	}
	return anchors
}

func explicitHTMLIDs(line string) []string {
	matches := anchorIDRegexp.FindAllStringSubmatch(line, -1)
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		if match[1] != "" {
			ids = append(ids, match[1])
			continue
		}
		if match[2] != "" {
			ids = append(ids, match[2])
		}
	}
	return ids
}

var anchorIDRegexp = regexp.MustCompile(`(?i)<a\b[^>]*\bid=(?:"([^"]+)"|'([^']+)')`)

func headingAnchor(line string, seen map[string]int) string {
	if !strings.HasPrefix(line, "#") {
		return ""
	}

	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level >= len(line) || line[level] != ' ' {
		return ""
	}

	text := strings.TrimSpace(line[level:])
	text = strings.TrimSpace(strings.TrimRight(text, "#"))
	text = htmlTagRE.ReplaceAllString(text, "")
	text = markdownLinkRE.ReplaceAllString(text, "$2")
	text = strings.ReplaceAll(text, "`", "")

	base := slugifyHeading(text)
	if base == "" {
		return ""
	}
	if count := seen[base]; count > 0 {
		seen[base] = count + 1
		return fmt.Sprintf("%s-%d", base, count)
	}
	seen[base] = 1
	return base
}

func slugifyHeading(text string) string {
	text = stdhtml.UnescapeString(strings.ToLower(strings.TrimSpace(text)))
	var b strings.Builder
	lastDash := false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_':
			if b.Len() == 0 || lastDash {
				continue
			}
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func verifyDescriptionCoverage(repoRoot string, strict bool) ([]string, error) {
	pages, err := loadAPIReferencePages(APIReferenceBuildOptions{
		RepoRoot: repoRoot,
	})
	if err != nil {
		return nil, err
	}

	findings := descriptionCoverageWarnings(pages)
	if strict && len(findings) > 0 {
		return findings, fmt.Errorf("description coverage failed:\n- %s", strings.Join(findings, "\n- "))
	}
	return findings, nil
}

func descriptionCoverageWarnings(pages []*apiReferencePage) []string {
	findings := make([]string, 0)
	for _, page := range pages {
		apiVersion := page.FullGroup + "/" + page.Version
		for _, resource := range page.Resources {
			if strings.TrimSpace(resource.Summary) == "" {
				findings = append(findings, fmt.Sprintf("%s %s: missing top-level kind description", apiVersion, resource.Kind))
			}
			if len(resource.Packages) == 0 {
				continue
			}
			findings = append(findings, specSectionWarnings(apiVersion, resource.Kind, resource.SpecSection)...)
		}
	}
	sort.Strings(findings)
	return findings
}

func specSectionWarnings(apiVersion string, kind string, section *schemaSection) []string {
	if section == nil {
		return nil
	}

	prefix := section.Title
	if prefix != "" {
		prefix = strings.ToLower(prefix[:1]) + prefix[1:]
	}

	findings := make([]string, 0, len(section.Fields))
	for _, field := range section.Fields {
		if strings.TrimSpace(field.Description) == "" {
			findings = append(findings, fmt.Sprintf("%s %s %s.%s: missing public spec-field description", apiVersion, kind, prefix, field.Name))
		}
	}
	for _, nested := range section.Nested {
		findings = append(findings, specSectionWarnings(apiVersion, kind, nested)...)
	}
	return findings
}

func ensureWithinRoot(root string, target string) error {
	root = filepath.Clean(root)
	target = filepath.Clean(target)

	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("target %q escapes root %q", target, root)
	}
	return nil
}

func mustRel(root string, target string) string {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return target
	}
	return rel
}

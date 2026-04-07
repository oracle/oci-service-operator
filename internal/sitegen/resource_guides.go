/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sitegen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ResourceGuideCatalog stores optional copy overrides for generated resource guides.
type ResourceGuideCatalog struct {
	SchemaVersion string                 `yaml:"schemaVersion"`
	Resources     []ResourceGuideOverlay `yaml:"resources,omitempty"`
}

// ResourceGuideOverlay stores optional title and section copy for one resource guide.
type ResourceGuideOverlay struct {
	Group        string                 `yaml:"group"`
	Kind         string                 `yaml:"kind"`
	Title        string                 `yaml:"title,omitempty"`
	Introduction string                 `yaml:"introduction,omitempty"`
	PreSections  []ResourceGuideSection `yaml:"preSections,omitempty"`
	PostSections []ResourceGuideSection `yaml:"postSections,omitempty"`
}

// ResourceGuideSection stores one optional rendered Markdown section body.
type ResourceGuideSection struct {
	Title string `yaml:"title"`
	Body  string `yaml:"body"`
}

type resourceGuidePage struct {
	Group                string
	Version              string
	Kind                 string
	Title                string
	Introduction         string
	OutputPath           string
	PackageDisplayName   string
	PackageOutputPath    string
	SetupGuidePath       string
	SupportStatus        string
	LatestReleaseVersion string
	InstallNamespace     string
	APIVersion           string
	APIPagePath          string
	APIAnchor            string
	SpecSection          *schemaSection
	StatusSection        *schemaSection
	Sample               *ReferenceSamplePage
	PreSections          []ResourceGuideSection
	PostSections         []ResourceGuideSection
}

func LoadResourceGuideCatalog(path string) (*ResourceGuideCatalog, error) {
	var catalog ResourceGuideCatalog
	if err := decodeYAMLFile(path, &catalog); err != nil {
		return nil, fmt.Errorf("load resource guide catalog %q: %w", path, err)
	}
	if err := catalog.Validate(); err != nil {
		return nil, fmt.Errorf("validate resource guide catalog %q: %w", path, err)
	}
	return &catalog, nil
}

func (c *ResourceGuideCatalog) Validate() error {
	if c == nil {
		return fmt.Errorf("resource guide catalog is required")
	}
	if c.SchemaVersion != SchemaVersionV1Alpha1 {
		return fmt.Errorf("schemaVersion must be %q", SchemaVersionV1Alpha1)
	}

	seen := make(map[string]struct{}, len(c.Resources))
	var previous string
	for _, resource := range c.Resources {
		if err := resource.Validate(); err != nil {
			return err
		}
		key := resourceGuideOverlayKey(resource.Group, resource.Kind)
		if previous != "" && key < previous {
			return fmt.Errorf("resource guides must be sorted by group/kind")
		}
		previous = key
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate resource guide overlay %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (o *ResourceGuideOverlay) Validate() error {
	if strings.TrimSpace(o.Group) == "" {
		return fmt.Errorf("resource guide overlay group is required")
	}
	if strings.TrimSpace(o.Kind) == "" {
		return fmt.Errorf("resource guide overlay kind is required")
	}
	if strings.TrimSpace(o.Title) == "" &&
		strings.TrimSpace(o.Introduction) == "" &&
		len(o.PreSections) == 0 &&
		len(o.PostSections) == 0 {
		return fmt.Errorf("resource guide overlay %s/%s must include title, introduction, or sections", o.Group, o.Kind)
	}
	for _, section := range o.PreSections {
		if err := section.Validate(o.Group, o.Kind, "preSections"); err != nil {
			return err
		}
	}
	for _, section := range o.PostSections {
		if err := section.Validate(o.Group, o.Kind, "postSections"); err != nil {
			return err
		}
	}
	return nil
}

func (s *ResourceGuideSection) Validate(group string, kind string, field string) error {
	if strings.TrimSpace(s.Title) == "" {
		return fmt.Errorf("resource guide overlay %s/%s %s title is required", group, kind, field)
	}
	if strings.TrimSpace(s.Body) == "" {
		return fmt.Errorf("resource guide overlay %s/%s %s %q body is required", group, kind, field, s.Title)
	}
	return nil
}

func buildResourceGuidePages(repoRoot string, site *ReferenceSite) ([]resourceGuidePage, map[groupVersionKindKey]string, error) {
	overlays, err := LoadResourceGuideCatalog(filepath.Join(repoRoot, "docs", "site-data", "resource-guides.yaml"))
	if err != nil {
		return nil, nil, err
	}
	overlayByKey := make(map[string]ResourceGuideOverlay, len(overlays.Resources))
	for _, overlay := range overlays.Resources {
		overlayByKey[resourceGuideOverlayKey(overlay.Group, overlay.Kind)] = overlay
	}

	apiPages, err := loadAPIReferencePages(APIReferenceBuildOptions{RepoRoot: repoRoot})
	if err != nil {
		return nil, nil, err
	}
	apiByKey := make(map[groupVersionKindKey]apiReferenceResource)
	apiPagePathByKey := make(map[groupVersionKindKey]string)
	for _, page := range apiPages {
		pagePath := filepath.ToSlash(filepath.Join("docs", "reference", "api", page.Group, page.Version, "index.md"))
		for _, resource := range page.Resources {
			key := groupVersionKindKey{
				Group:   page.Group,
				Version: page.Version,
				Kind:    resource.Kind,
			}
			apiByKey[key] = resource
			apiPagePathByKey[key] = pagePath
		}
	}

	sampleByPath := make(map[string]ReferenceSamplePage, len(site.SamplePages))
	for _, sample := range site.SamplePages {
		sampleByPath[sample.OutputPath] = sample
	}

	pagesByKey := make(map[groupVersionKindKey]resourceGuidePage)
	guidePathByKey := make(map[groupVersionKindKey]string)
	for _, pkg := range site.PublicPackages {
		for _, resource := range pkg.Resources {
			key := groupVersionKindKey{
				Group:   resource.Group,
				Version: resource.Version,
				Kind:    resource.Kind,
			}
			outputPath := resolveResourceGuideOutputPath(pkg, resource)
			if existingPath, ok := guidePathByKey[key]; ok && existingPath != outputPath {
				return nil, nil, fmt.Errorf("resource guide output path conflict for %s/%s: %q vs %q", resource.Group, resource.Kind, existingPath, outputPath)
			}
			guidePathByKey[key] = outputPath
			if _, exists := pagesByKey[key]; exists {
				continue
			}

			apiResource, ok := apiByKey[key]
			if !ok {
				return nil, nil, fmt.Errorf("missing api reference model for %s/%s %s", resource.Group, resource.Version, resource.Kind)
			}

			var sample *ReferenceSamplePage
			if resource.SampleOutputPath != "" {
				value, ok := sampleByPath[resource.SampleOutputPath]
				if !ok {
					return nil, nil, fmt.Errorf("missing sample page model for %s/%s %s", resource.Group, resource.Version, resource.Kind)
				}
				sampleCopy := value
				sample = &sampleCopy
			}

			overlay := overlayByKey[resourceGuideOverlayKey(resource.Group, resource.Kind)]
			pagesByKey[key] = resourceGuidePage{
				Group:                resource.Group,
				Version:              resource.Version,
				Kind:                 resource.Kind,
				Title:                resourceGuideTitle(pkg, resource, overlay),
				Introduction:         resourceGuideIntroduction(pkg, resource, overlay),
				OutputPath:           outputPath,
				PackageDisplayName:   pkg.DisplayName,
				PackageOutputPath:    pkg.OutputPath,
				SetupGuidePath:       filepath.ToSlash(strings.TrimSpace(pkg.GuidePath)),
				SupportStatus:        pkg.SupportStatus,
				LatestReleaseVersion: pkg.LatestReleaseVersion,
				InstallNamespace:     pkg.InstallNamespace,
				APIVersion:           resource.APIVersion,
				APIPagePath:          apiPagePathByKey[key],
				APIAnchor:            resource.APIAnchor,
				SpecSection:          apiResource.SpecSection,
				StatusSection:        apiResource.StatusSection,
				Sample:               sample,
				PreSections:          cloneResourceGuideSections(overlay.PreSections),
				PostSections:         cloneResourceGuideSections(overlay.PostSections),
			}
		}
	}

	pages := make([]resourceGuidePage, 0, len(pagesByKey))
	for _, page := range pagesByKey {
		pages = append(pages, page)
	}
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Group != pages[j].Group {
			return pages[i].Group < pages[j].Group
		}
		if pages[i].Kind != pages[j].Kind {
			return pages[i].Kind < pages[j].Kind
		}
		return pages[i].OutputPath < pages[j].OutputPath
	})

	return pages, guidePathByKey, nil
}

func applyResourceGuidePaths(packages []ReferencePackage, guidePathByKey map[groupVersionKindKey]string) []ReferencePackage {
	out := make([]ReferencePackage, 0, len(packages))
	for _, pkg := range packages {
		pkgCopy := pkg
		pkgCopy.Resources = make([]ReferenceResource, 0, len(pkg.Resources))
		for _, resource := range pkg.Resources {
			resourceCopy := resource
			key := groupVersionKindKey{
				Group:   resource.Group,
				Version: resource.Version,
				Kind:    resource.Kind,
			}
			resourceCopy.GuideOutputPath = guidePathByKey[key]
			pkgCopy.Resources = append(pkgCopy.Resources, resourceCopy)
		}
		out = append(out, pkgCopy)
	}
	return out
}

func resourceGuideTitle(pkg ReferencePackage, resource ReferenceResource, overlay ResourceGuideOverlay) string {
	if title := strings.TrimSpace(overlay.Title); title != "" {
		return title
	}
	return pkg.DisplayName + ": " + resource.Kind
}

func resourceGuideIntroduction(pkg ReferencePackage, resource ReferenceResource, overlay ResourceGuideOverlay) string {
	if introduction := strings.TrimSpace(overlay.Introduction); introduction != "" {
		return introduction
	}

	base := strings.TrimSpace(resource.Summary)
	if base == "" {
		base = strings.TrimSpace(pkg.Summary)
	}
	if base == "" {
		base = fmt.Sprintf("This guide covers `%s/%s`.", resource.Group, resource.Kind)
	}

	if strings.HasSuffix(base, ".") {
		return base + " This page is generated from checked-in package metadata, CRD schemas, and sample manifests."
	}
	return base + ". This page is generated from checked-in package metadata, CRD schemas, and sample manifests."
}

func resolveResourceGuideOutputPath(pkg ReferencePackage, resource ReferenceResource) string {
	if len(pkg.Resources) == 1 {
		if guidePath := resourceGuidePath(pkg.GuidePath); guidePath != "" {
			return filepath.ToSlash(filepath.Join("docs", guidePath))
		}
	}
	return filepath.ToSlash(filepath.Join("docs", "guides", resource.Group, slug(resource.Kind)+".md"))
}

func renderResourceGuidePage(page resourceGuidePage) string {
	var b strings.Builder
	b.WriteString(generatedMarkdownNotice)
	b.WriteString("\n\n# ")
	b.WriteString(page.Title)
	b.WriteString("\n\n")
	b.WriteString(page.Introduction)
	b.WriteString("\n\n")

	b.WriteString("## Resource Snapshot\n\n")
	b.WriteString("| Field | Value |\n")
	b.WriteString("| --- | --- |\n")
	b.WriteString("| Service | ")
	b.WriteString(codeSpan(page.Group))
	b.WriteString(" |\n")
	b.WriteString("| Resource | ")
	b.WriteString(codeSpan(page.Kind))
	b.WriteString(" |\n")
	b.WriteString("| API Version | ")
	b.WriteString(codeSpan(page.APIVersion))
	b.WriteString(" |\n")
	b.WriteString("| Package | ")
	b.WriteString(markdownLink(page.PackageDisplayName, docsLink(page.OutputPath, page.PackageOutputPath)))
	b.WriteString(" |\n")
	b.WriteString("| Support Status | ")
	b.WriteString(escapeMarkdownCell(page.SupportStatus))
	b.WriteString(" |\n")
	b.WriteString("| Latest Released Version | ")
	b.WriteString(codeSpan(displayReleaseVersion(page.LatestReleaseVersion)))
	b.WriteString(" |\n")
	b.WriteString("| Install Namespace | ")
	b.WriteString(codeSpan(page.InstallNamespace))
	b.WriteString(" |\n")

	b.WriteString("\n## Quick Links\n\n")
	b.WriteString("- ")
	b.WriteString(markdownLink("Resource Guide Index", docsLink(page.OutputPath, filepath.ToSlash(filepath.Join("docs", "guides", "index.md")))))
	b.WriteString("\n")
	if strings.TrimSpace(page.SetupGuidePath) != "" && page.SetupGuidePath != page.OutputPath {
		b.WriteString("- ")
		b.WriteString(markdownLink("Setup Guide", docsLink(page.OutputPath, page.SetupGuidePath)))
		b.WriteString("\n")
	}
	b.WriteString("- ")
	b.WriteString(markdownLink("Package Page", docsLink(page.OutputPath, page.PackageOutputPath)))
	b.WriteString("\n")
	b.WriteString("- ")
	b.WriteString(markdownLink("API Reference", docsLink(page.OutputPath, page.APIPagePath)+"#"+page.APIAnchor))
	b.WriteString("\n")
	if page.SpecSection != nil {
		b.WriteString("- ")
		b.WriteString(markdownLink("Spec Reference", docsLink(page.OutputPath, page.APIPagePath)+"#"+page.SpecSection.Anchor))
		b.WriteString("\n")
	}
	if page.StatusSection != nil {
		b.WriteString("- ")
		b.WriteString(markdownLink("Status Reference", docsLink(page.OutputPath, page.APIPagePath)+"#"+page.StatusSection.Anchor))
		b.WriteString("\n")
	}
	if page.Sample != nil {
		b.WriteString("- ")
		b.WriteString(markdownLink("Rendered Sample", docsLink(page.OutputPath, page.Sample.OutputPath)))
		b.WriteString(" (")
		b.WriteString(codeSpan(page.Sample.SourcePath))
		b.WriteString(")\n")
	}

	renderResourceGuideCustomSections(&b, page.PreSections)
	renderResourceGuideSchemaSummarySection(&b, page, "Spec Fields", page.SpecSection)
	renderResourceGuideSchemaSummarySection(&b, page, "Status Fields", page.StatusSection)
	renderResourceGuideSample(&b, page)
	renderResourceGuideCustomSections(&b, page.PostSections)

	return b.String()
}

func renderResourceGuideCustomSections(b *strings.Builder, sections []ResourceGuideSection) {
	for _, section := range sections {
		b.WriteString("\n## ")
		b.WriteString(section.Title)
		b.WriteString("\n\n")
		b.WriteString(strings.TrimSpace(section.Body))
		b.WriteString("\n")
	}
}

func renderResourceGuideSchemaSummarySection(b *strings.Builder, page resourceGuidePage, title string, section *schemaSection) {
	b.WriteString("\n## ")
	b.WriteString(title)
	b.WriteString("\n\n")

	if section == nil {
		b.WriteString("No documented fields are currently present in the checked-in CRD schema.\n")
		return
	}

	b.WriteString("This summary shows the top-level ")
	b.WriteString(codeSpan(strings.ToLower(strings.TrimSuffix(title, " Fields"))))
	b.WriteString(" fields. Use ")
	b.WriteString(markdownLink("the full API reference", docsLink(page.OutputPath, page.APIPagePath)+"#"+section.Anchor))
	b.WriteString(" for nested fields, defaults, and enum values.\n\n")

	if len(section.Fields) == 0 {
		b.WriteString("No top-level fields are currently documented in the checked-in CRD schema.\n")
		return
	}

	b.WriteString("| Field | Description | Type | Required |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, field := range section.Fields {
		b.WriteString("| ")
		b.WriteString(renderResourceGuideFieldNameCell(page, field))
		b.WriteString(" | ")
		b.WriteString(escapeTableText(dashIfEmpty(field.Description)))
		b.WriteString(" | ")
		b.WriteString(codeSpan(field.Type))
		b.WriteString(" | ")
		b.WriteString(boolLabel(field.Required))
		b.WriteString(" |\n")
	}
	b.WriteString("\n")
}

func renderResourceGuideFieldNameCell(page resourceGuidePage, field schemaField) string {
	if field.Anchor == "" {
		return codeSpan(field.Name)
	}
	return markdownLink(codeSpan(field.Name), docsLink(page.OutputPath, page.APIPagePath)+"#"+field.Anchor)
}

func renderResourceGuideSample(b *strings.Builder, page resourceGuidePage) {
	b.WriteString("\n## Sample Manifest\n\n")
	if page.Sample == nil {
		b.WriteString("No checked-in sample manifest currently exists for this resource.\n")
		return
	}

	b.WriteString("This example is generated from the checked-in sample manifest at ")
	b.WriteString(codeSpan(page.Sample.SourcePath))
	b.WriteString(". Replace placeholder values before applying it.\n\n")
	b.WriteString(markdownLink("Open the rendered sample page", docsLink(page.OutputPath, page.Sample.OutputPath)))
	b.WriteString("\n\n```yaml\n")
	b.WriteString(page.Sample.Content)
	b.WriteString("\n```\n")
}

func cloneResourceGuideSections(in []ResourceGuideSection) []ResourceGuideSection {
	if len(in) == 0 {
		return nil
	}
	out := make([]ResourceGuideSection, 0, len(in))
	for _, section := range in {
		out = append(out, ResourceGuideSection{
			Title: section.Title,
			Body:  section.Body,
		})
	}
	return out
}

func resourceGuideOverlayKey(group string, kind string) string {
	return strings.TrimSpace(group) + "/" + strings.TrimSpace(kind)
}

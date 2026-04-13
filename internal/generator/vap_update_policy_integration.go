package generator

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type vapUpdatePolicyGeneratedArtifact struct {
	Service      string
	RelativePath string
	Document     vapUpdatePolicyDocument
}

type vapUpdatePolicyResourceContext struct {
	Package  *PackageModel
	Resource ResourceModel
}

func buildVAPUpdatePolicyArtifacts(
	packages []*PackageModel,
	overlayArtifacts []mutabilityOverlayGeneratedArtifact,
) ([]vapUpdatePolicyGeneratedArtifact, error) {
	if len(overlayArtifacts) == 0 {
		return nil, nil
	}

	resourceIndex := make(map[string]vapUpdatePolicyResourceContext)
	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		for _, resource := range pkg.Resources {
			if resource.Formal == nil {
				continue
			}
			key := vapUpdatePolicyArtifactKey(pkg.Service.Service, resource.Formal.Reference.Slug, resource.Kind)
			if _, exists := resourceIndex[key]; exists {
				return nil, fmt.Errorf("duplicate vap update policy resource mapping for key %q", key)
			}
			resourceIndex[key] = vapUpdatePolicyResourceContext{
				Package:  pkg,
				Resource: resource,
			}
		}
	}

	artifacts := make([]vapUpdatePolicyGeneratedArtifact, 0, len(overlayArtifacts))
	for _, overlay := range overlayArtifacts {
		key := vapUpdatePolicyArtifactKey(
			overlay.Document.Resource.Service,
			overlay.Document.Resource.FormalSlug,
			overlay.Document.Resource.Kind,
		)
		context, ok := resourceIndex[key]
		if !ok {
			return nil, fmt.Errorf(
				"missing package resource context for mutability overlay service %q formalSpec %q kind %q",
				overlay.Document.Resource.Service,
				overlay.Document.Resource.FormalSlug,
				overlay.Document.Resource.Kind,
			)
		}
		doc, err := buildVAPUpdatePolicyDocument(context.Package, context.Resource, overlay.Document)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, newVAPUpdatePolicyGeneratedArtifact(overlay.Document.Resource.Service, context.Resource, doc))
	}

	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].RelativePath < artifacts[j].RelativePath
	})
	return artifacts, nil
}

func vapUpdatePolicyArtifactKey(service, formalSlug, kind string) string {
	service = strings.TrimSpace(service)
	formalSlug = strings.TrimSpace(formalSlug)
	if formalSlug == "" {
		formalSlug = fileStem(kind)
	}
	return service + "/" + formalSlug
}

func buildVAPUpdatePolicyDocument(
	pkg *PackageModel,
	resource ResourceModel,
	overlayDoc mutabilityOverlayDocument,
) (vapUpdatePolicyDocument, error) {
	if pkg == nil {
		return vapUpdatePolicyDocument{}, fmt.Errorf("vap update policy document requires a package model")
	}
	if resource.Formal == nil {
		return vapUpdatePolicyDocument{}, fmt.Errorf("vap update policy document requires a formal-backed resource")
	}
	if err := validateMutabilityOverlayDocument(overlayDoc); err != nil {
		return vapUpdatePolicyDocument{}, fmt.Errorf(
			"validate source mutability overlay for service %q kind %q: %w",
			overlayDoc.Resource.Service,
			overlayDoc.Resource.Kind,
			err,
		)
	}

	if got := strings.TrimSpace(overlayDoc.Resource.Service); got != pkg.Service.Service {
		return vapUpdatePolicyDocument{}, fmt.Errorf("mutability overlay service = %q, want %q", got, pkg.Service.Service)
	}
	if got := strings.TrimSpace(overlayDoc.Resource.Kind); got != resource.Kind {
		return vapUpdatePolicyDocument{}, fmt.Errorf("mutability overlay kind = %q, want %q", got, resource.Kind)
	}
	if got := strings.TrimSpace(overlayDoc.Resource.FormalSlug); got != resource.Formal.Reference.Slug {
		return vapUpdatePolicyDocument{}, fmt.Errorf("mutability overlay formalSlug = %q, want %q", got, resource.Formal.Reference.Slug)
	}
	if got := strings.TrimSpace(overlayDoc.Resource.ProviderResource); got != strings.TrimSpace(resource.Formal.Binding.Import.ProviderResource) {
		return vapUpdatePolicyDocument{}, fmt.Errorf(
			"mutability overlay providerResource = %q, want %q",
			got,
			strings.TrimSpace(resource.Formal.Binding.Import.ProviderResource),
		)
	}

	allowPaths := make([]string, 0, len(overlayDoc.Fields))
	denyRules := make([]vapUpdatePolicyRule, 0, len(overlayDoc.Fields))
	for _, field := range overlayDoc.Fields {
		switch field.Merge.FinalPolicy {
		case mutabilityOverlayPolicyAllowInPlaceUpdate:
			allowPaths = append(allowPaths, field.ASTFieldPath)
		case mutabilityOverlayPolicyDenyInPlaceUpdate, mutabilityOverlayPolicyReplacementRequired, mutabilityOverlayPolicyUnknown:
			denyRules = append(denyRules, vapUpdatePolicyRule{
				FieldPath:         field.ASTFieldPath,
				Decision:          field.Merge.FinalPolicy,
				MergeCase:         field.Merge.MergeCase,
				DocsEvidenceState: field.Docs.EvidenceState,
				Detail:            strings.TrimSpace(field.Docs.Detail),
			})
		default:
			return vapUpdatePolicyDocument{}, fmt.Errorf(
				"unsupported mutability overlay finalPolicy %q for service %q kind %q field %q",
				field.Merge.FinalPolicy,
				overlayDoc.Resource.Service,
				overlayDoc.Resource.Kind,
				field.ASTFieldPath,
			)
		}
	}
	allowPaths = uniqueSortedStrings(allowPaths)
	sort.Slice(denyRules, func(i, j int) bool {
		if denyRules[i].FieldPath != denyRules[j].FieldPath {
			return denyRules[i].FieldPath < denyRules[j].FieldPath
		}
		return denyRules[i].Decision < denyRules[j].Decision
	})

	doc := vapUpdatePolicyDocument{
		SchemaVersion:   vapUpdatePolicySchemaVersion,
		Surface:         vapUpdatePolicySurface,
		ContractVersion: vapUpdatePolicyContractVersion,
		Metadata: vapUpdatePolicyMetadata{
			SourceSurface:        mutabilityOverlaySurface,
			ProviderSourceRef:    overlayDoc.Metadata.ProviderSourceRef,
			ProviderRevision:     overlayDoc.Metadata.ProviderRevision,
			TerraformDocsVersion: overlayDoc.Metadata.TerraformDocsVersion,
		},
		Target: vapUpdatePolicyTarget{
			Service:          pkg.Service.Service,
			APIVersion:       pkg.GroupDNSName + "/" + pkg.Version,
			Kind:             resource.Kind,
			FormalSlug:       overlayDoc.Resource.FormalSlug,
			ProviderResource: overlayDoc.Resource.ProviderResource,
			SpecPathPrefix:   vapUpdatePolicySpecPathPrefix,
		},
		Update: vapUpdatePolicyUpdate{
			AllowInPlacePaths: allowPaths,
			DenyRules:         denyRules,
		},
	}
	if err := validateVAPUpdatePolicyDocument(doc); err != nil {
		return vapUpdatePolicyDocument{}, fmt.Errorf(
			"build vap update policy document for service %q kind %q: %w",
			pkg.Service.Service,
			resource.Kind,
			err,
		)
	}
	return doc, nil
}

func newVAPUpdatePolicyGeneratedArtifact(
	service string,
	resource ResourceModel,
	doc vapUpdatePolicyDocument,
) vapUpdatePolicyGeneratedArtifact {
	slug := fileStem(resource.Kind)
	if resource.Formal != nil && strings.TrimSpace(resource.Formal.Reference.Slug) != "" {
		slug = strings.TrimSpace(resource.Formal.Reference.Slug)
	}
	return vapUpdatePolicyGeneratedArtifact{
		Service: service,
		RelativePath: filepath.ToSlash(filepath.Join(
			vapUpdatePolicyGeneratedRootRelativePath,
			service,
			slug+".json",
		)),
		Document: doc,
	}
}

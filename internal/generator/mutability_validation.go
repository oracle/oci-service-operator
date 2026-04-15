package generator

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type MutabilityValidationReport struct {
	Summary MutabilityValidationSummary `json:"summary"`
}

type MutabilityValidationSummary struct {
	Aggregate MutabilityValidationAggregate         `json:"aggregate"`
	Services  []MutabilityValidationServiceSummary  `json:"services"`
	Resources []MutabilityValidationResourceSummary `json:"resources"`
}

type MutabilityValidationAggregate struct {
	Services                 int `json:"services"`
	Resources                int `json:"resources"`
	OverlayArtifacts         int `json:"overlayArtifacts"`
	VAPArtifacts             int `json:"vapArtifacts"`
	Fields                   int `json:"fields"`
	AllowInPlaceCount        int `json:"allowInPlaceCount"`
	DenyInPlaceCount         int `json:"denyInPlaceCount"`
	ReplacementRequiredCount int `json:"replacementRequiredCount"`
	UnknownPolicyCount       int `json:"unknownPolicyCount"`
	DocsDeniedCount          int `json:"docsDeniedCount"`
	UnresolvedJoinCount      int `json:"unresolvedJoinCount"`
	AmbiguousJoinCount       int `json:"ambiguousJoinCount"`
	MergeConflictCount       int `json:"mergeConflictCount"`
}

type MutabilityValidationServiceSummary struct {
	Service                  string `json:"service"`
	Resources                int    `json:"resources"`
	Fields                   int    `json:"fields"`
	AllowInPlaceCount        int    `json:"allowInPlaceCount"`
	DenyInPlaceCount         int    `json:"denyInPlaceCount"`
	ReplacementRequiredCount int    `json:"replacementRequiredCount"`
	UnknownPolicyCount       int    `json:"unknownPolicyCount"`
	DocsDeniedCount          int    `json:"docsDeniedCount"`
	UnresolvedJoinCount      int    `json:"unresolvedJoinCount"`
	AmbiguousJoinCount       int    `json:"ambiguousJoinCount"`
	MergeConflictCount       int    `json:"mergeConflictCount"`
}

type MutabilityValidationResourceSummary struct {
	Service                  string                              `json:"service"`
	Kind                     string                              `json:"kind"`
	FormalSlug               string                              `json:"formalSlug"`
	ProviderResource         string                              `json:"providerResource"`
	TerraformDocsVersion     string                              `json:"terraformDocsVersion"`
	OverlayPath              string                              `json:"overlayPath"`
	VAPPath                  string                              `json:"vapPath"`
	Fields                   int                                 `json:"fields"`
	AllowInPlaceCount        int                                 `json:"allowInPlaceCount"`
	DenyInPlaceCount         int                                 `json:"denyInPlaceCount"`
	ReplacementRequiredCount int                                 `json:"replacementRequiredCount"`
	UnknownPolicyCount       int                                 `json:"unknownPolicyCount"`
	AllowInPlacePaths        []string                            `json:"allowInPlacePaths"`
	Decisions                []MutabilityValidationFieldDecision `json:"decisions"`
	DocsDeniedFields         []string                            `json:"docsDeniedFields,omitempty"`
	UnknownFields            []string                            `json:"unknownFields,omitempty"`
	UnresolvedJoinFields     []string                            `json:"unresolvedJoinFields,omitempty"`
	AmbiguousJoinFields      []string                            `json:"ambiguousJoinFields,omitempty"`
	MergeConflicts           []MutabilityValidationMergeConflict `json:"mergeConflicts,omitempty"`
	DenyRules                []MutabilityValidationVAPDenyRule   `json:"denyRules,omitempty"`
}

type MutabilityValidationFieldDecision struct {
	FieldPath         string `json:"fieldPath"`
	FinalPolicy       string `json:"finalPolicy"`
	JoinStatus        string `json:"joinStatus"`
	MergeCase         string `json:"mergeCase"`
	DocsEvidenceState string `json:"docsEvidenceState"`
}

type MutabilityValidationMergeConflict struct {
	FieldPath string `json:"fieldPath"`
	Reason    string `json:"reason"`
}

type MutabilityValidationVAPDenyRule struct {
	FieldPath         string `json:"fieldPath"`
	Decision          string `json:"decision"`
	MergeCase         string `json:"mergeCase"`
	DocsEvidenceState string `json:"docsEvidenceState"`
}

type MutabilityValidationComparison struct {
	ScopeChanges []string `json:"scopeChanges,omitempty"`
	Regressions  []string `json:"regressions,omitempty"`
}

func (c MutabilityValidationComparison) HasFailures() bool {
	return len(c.ScopeChanges) != 0 || len(c.Regressions) != 0
}

type MutabilityValidationFailureGroup struct {
	Class   string                              `json:"class"`
	Reasons []MutabilityValidationFailureReason `json:"reasons"`
}

type MutabilityValidationFailureReason struct {
	Reason   string   `json:"reason"`
	Count    int      `json:"count"`
	Examples []string `json:"examples,omitempty"`
}

type loadedMutabilityOverlayArtifact struct {
	RelativePath string
	Document     mutabilityOverlayDocument
}

type loadedVAPUpdatePolicyArtifact struct {
	RelativePath string
	Document     vapUpdatePolicyDocument
}

func LoadMutabilityValidationReport(root string) (MutabilityValidationReport, error) {
	overlays, err := loadMutabilityValidationOverlayArtifacts(root)
	if err != nil {
		return MutabilityValidationReport{}, err
	}
	vaps, err := loadMutabilityValidationVAPArtifacts(root)
	if err != nil {
		return MutabilityValidationReport{}, err
	}
	summary, err := buildMutabilityValidationSummary(overlays, vaps)
	if err != nil {
		return MutabilityValidationReport{}, err
	}
	return MutabilityValidationReport{Summary: summary}, nil
}

func WriteMutabilityValidationBaseline(path string, summary MutabilityValidationSummary) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("baseline path must not be empty")
	}
	summary = normalizeMutabilityValidationSummary(summary)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

func LoadMutabilityValidationBaseline(path string) (MutabilityValidationSummary, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return MutabilityValidationSummary{}, err
	}
	var summary MutabilityValidationSummary
	if err := json.Unmarshal(content, &summary); err != nil {
		return MutabilityValidationSummary{}, err
	}
	return normalizeMutabilityValidationSummary(summary), nil
}

func CompareMutabilityValidationSummary(current, baseline MutabilityValidationSummary) MutabilityValidationComparison {
	current = normalizeMutabilityValidationSummary(current)
	baseline = normalizeMutabilityValidationSummary(baseline)

	comparison := MutabilityValidationComparison{}
	if current.Aggregate.Resources != baseline.Aggregate.Resources {
		comparison.ScopeChanges = append(
			comparison.ScopeChanges,
			fmt.Sprintf("aggregate resources changed from %d to %d", baseline.Aggregate.Resources, current.Aggregate.Resources),
		)
	}
	if current.Aggregate.Fields != baseline.Aggregate.Fields {
		comparison.ScopeChanges = append(
			comparison.ScopeChanges,
			fmt.Sprintf("aggregate fields changed from %d to %d", baseline.Aggregate.Fields, current.Aggregate.Fields),
		)
	}
	appendMutabilityValidationRegressions(&comparison.Regressions, "aggregate", baseline.Aggregate, current.Aggregate)

	currentServices := make(map[string]MutabilityValidationServiceSummary, len(current.Services))
	for _, service := range current.Services {
		currentServices[service.Service] = service
	}
	baselineServices := make(map[string]MutabilityValidationServiceSummary, len(baseline.Services))
	for _, service := range baseline.Services {
		baselineServices[service.Service] = service
	}

	currentServiceNames := sortedMutabilityValidationServiceNames(currentServices)
	baselineServiceNames := sortedMutabilityValidationServiceNames(baselineServices)
	for _, name := range difference(baselineServiceNames, currentServiceNames) {
		comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf("service %q is missing from the current report", name))
	}
	for _, name := range difference(currentServiceNames, baselineServiceNames) {
		comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf("service %q is new in the current report", name))
	}
	for _, name := range intersection(currentServiceNames, baselineServiceNames) {
		currentService := currentServices[name]
		baselineService := baselineServices[name]
		if currentService.Resources != baselineService.Resources {
			comparison.ScopeChanges = append(
				comparison.ScopeChanges,
				fmt.Sprintf("service %q resources changed from %d to %d", name, baselineService.Resources, currentService.Resources),
			)
		}
		if currentService.Fields != baselineService.Fields {
			comparison.ScopeChanges = append(
				comparison.ScopeChanges,
				fmt.Sprintf("service %q fields changed from %d to %d", name, baselineService.Fields, currentService.Fields),
			)
		}
		appendMutabilityValidationRegressions(&comparison.Regressions, fmt.Sprintf("service %q", name), baselineService, currentService)
	}

	currentResources := make(map[string]MutabilityValidationResourceSummary, len(current.Resources))
	for _, resource := range current.Resources {
		currentResources[mutabilityValidationResourceKey(resource.Service, resource.FormalSlug, resource.Kind)] = resource
	}
	baselineResources := make(map[string]MutabilityValidationResourceSummary, len(baseline.Resources))
	for _, resource := range baseline.Resources {
		baselineResources[mutabilityValidationResourceKey(resource.Service, resource.FormalSlug, resource.Kind)] = resource
	}

	currentResourceKeys := sortedMutabilityValidationResourceKeys(currentResources)
	baselineResourceKeys := sortedMutabilityValidationResourceKeys(baselineResources)
	for _, key := range difference(baselineResourceKeys, currentResourceKeys) {
		resource := baselineResources[key]
		comparison.ScopeChanges = append(
			comparison.ScopeChanges,
			fmt.Sprintf("resource %s/%s (%s) is missing from the current report", resource.Service, resource.Kind, resource.FormalSlug),
		)
	}
	for _, key := range difference(currentResourceKeys, baselineResourceKeys) {
		resource := currentResources[key]
		comparison.ScopeChanges = append(
			comparison.ScopeChanges,
			fmt.Sprintf("resource %s/%s (%s) is new in the current report", resource.Service, resource.Kind, resource.FormalSlug),
		)
	}
	for _, key := range intersection(currentResourceKeys, baselineResourceKeys) {
		currentResource := currentResources[key]
		baselineResource := baselineResources[key]

		if currentResource.TerraformDocsVersion != baselineResource.TerraformDocsVersion {
			comparison.ScopeChanges = append(
				comparison.ScopeChanges,
				fmt.Sprintf(
					"resource %s/%s (%s) terraformDocsVersion changed from %q to %q",
					currentResource.Service,
					currentResource.Kind,
					currentResource.FormalSlug,
					baselineResource.TerraformDocsVersion,
					currentResource.TerraformDocsVersion,
				),
			)
		}
		if currentResource.Fields != baselineResource.Fields {
			comparison.ScopeChanges = append(
				comparison.ScopeChanges,
				fmt.Sprintf(
					"resource %s/%s (%s) fields changed from %d to %d",
					currentResource.Service,
					currentResource.Kind,
					currentResource.FormalSlug,
					baselineResource.Fields,
					currentResource.Fields,
				),
			)
		}

		addedAllowPaths, removedAllowPaths := diffMutabilityValidationStringSets(currentResource.AllowInPlacePaths, baselineResource.AllowInPlacePaths)
		for _, fieldPath := range addedAllowPaths {
			comparison.Regressions = append(
				comparison.Regressions,
				fmt.Sprintf(
					"resource %s/%s (%s) widened allowInPlace path %q",
					currentResource.Service,
					currentResource.Kind,
					currentResource.FormalSlug,
					fieldPath,
				),
			)
		}
		for _, fieldPath := range removedAllowPaths {
			comparison.ScopeChanges = append(
				comparison.ScopeChanges,
				fmt.Sprintf(
					"resource %s/%s (%s) removed allowInPlace path %q",
					currentResource.Service,
					currentResource.Kind,
					currentResource.FormalSlug,
					fieldPath,
				),
			)
		}

		compareMutabilityValidationFieldDecisions(&comparison, currentResource, baselineResource)
		compareMutabilityValidationMergeConflicts(&comparison, currentResource, baselineResource)
	}

	comparison.ScopeChanges = uniqueSortedStrings(comparison.ScopeChanges)
	comparison.Regressions = uniqueSortedStrings(comparison.Regressions)
	return comparison
}

func ClassifyMutabilityValidationError(err error) []MutabilityValidationFailureGroup {
	leafErrors := flattenMutabilityValidationErrors(err)
	if len(leafErrors) == 0 {
		return nil
	}

	type failureBucket struct {
		Count    int
		Examples []string
	}
	classified := make(map[string]map[string]failureBucket)
	appendExample := func(class string, reason string, example string) {
		reasons := classified[class]
		if reasons == nil {
			reasons = make(map[string]failureBucket)
			classified[class] = reasons
		}
		bucket := reasons[reason]
		bucket.Count++
		bucket.Examples = appendUniqueString(bucket.Examples, example)
		reasons[reason] = bucket
	}

	for _, leaf := range leafErrors {
		switch typed := leaf.(type) {
		case *mutabilityOverlayRegistryPageMappingError:
			appendExample("mappingFailures", typed.Reason, typed.Error())
		case *mutabilityOverlayDocsAcquisitionError:
			appendExample("docsAcquisitionFailures", typed.Reason, typed.Error())
		case *mutabilityOverlayDocsParseError:
			appendExample("parserFailures", typed.Reason, typed.Error())
		case *mutabilityOverlayGenerationError:
			appendExample("generationFailures", typed.Reason, typed.Error())
		default:
			appendExample("otherFailures", "unclassified", leaf.Error())
		}
	}

	classes := make([]string, 0, len(classified))
	for class := range classified {
		classes = append(classes, class)
	}
	sort.Strings(classes)

	groups := make([]MutabilityValidationFailureGroup, 0, len(classes))
	for _, class := range classes {
		reasons := classified[class]
		reasonNames := make([]string, 0, len(reasons))
		for reason := range reasons {
			reasonNames = append(reasonNames, reason)
		}
		sort.Strings(reasonNames)

		group := MutabilityValidationFailureGroup{
			Class:   class,
			Reasons: make([]MutabilityValidationFailureReason, 0, len(reasonNames)),
		}
		for _, reason := range reasonNames {
			examples := append([]string(nil), reasons[reason].Examples...)
			sort.Strings(examples)
			group.Reasons = append(group.Reasons, MutabilityValidationFailureReason{
				Reason:   reason,
				Count:    reasons[reason].Count,
				Examples: examples,
			})
		}
		groups = append(groups, group)
	}

	return groups
}

func loadMutabilityValidationOverlayArtifacts(root string) ([]loadedMutabilityOverlayArtifact, error) {
	overlayRoot := filepath.Join(root, filepath.FromSlash(mutabilityOverlayGeneratedRootRelativePath))
	info, err := os.Stat(overlayRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("mutability overlay output root %q does not exist", overlayRoot)
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("mutability overlay output root %q is not a directory", overlayRoot)
	}

	artifacts := make([]loadedMutabilityOverlayArtifact, 0)
	if err := filepath.WalkDir(overlayRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc mutabilityOverlayDocument
		if err := json.Unmarshal(content, &doc); err != nil {
			return fmt.Errorf("decode mutability overlay artifact %q: %w", path, err)
		}
		if err := validateMutabilityOverlayDocument(doc); err != nil {
			return fmt.Errorf("validate mutability overlay artifact %q: %w", path, err)
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, loadedMutabilityOverlayArtifact{
			RelativePath: filepath.ToSlash(relPath),
			Document:     doc,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("no mutability overlay artifacts were found under %q", overlayRoot)
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].RelativePath < artifacts[j].RelativePath
	})
	return artifacts, nil
}

func loadMutabilityValidationVAPArtifacts(root string) ([]loadedVAPUpdatePolicyArtifact, error) {
	vapRoot := filepath.Join(root, filepath.FromSlash(vapUpdatePolicyGeneratedRootRelativePath))
	info, err := os.Stat(vapRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("vap update policy output root %q does not exist", vapRoot)
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("vap update policy output root %q is not a directory", vapRoot)
	}

	artifacts := make([]loadedVAPUpdatePolicyArtifact, 0)
	if err := filepath.WalkDir(vapRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc vapUpdatePolicyDocument
		if err := json.Unmarshal(content, &doc); err != nil {
			return fmt.Errorf("decode vap update policy artifact %q: %w", path, err)
		}
		if err := validateVAPUpdatePolicyDocument(doc); err != nil {
			return fmt.Errorf("validate vap update policy artifact %q: %w", path, err)
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, loadedVAPUpdatePolicyArtifact{
			RelativePath: filepath.ToSlash(relPath),
			Document:     doc,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("no vap update policy artifacts were found under %q", vapRoot)
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].RelativePath < artifacts[j].RelativePath
	})
	return artifacts, nil
}

func buildMutabilityValidationSummary(
	overlays []loadedMutabilityOverlayArtifact,
	vaps []loadedVAPUpdatePolicyArtifact,
) (MutabilityValidationSummary, error) {
	vapIndex := make(map[string]loadedVAPUpdatePolicyArtifact, len(vaps))
	for _, artifact := range vaps {
		key := mutabilityValidationResourceKey(
			artifact.Document.Target.Service,
			artifact.Document.Target.FormalSlug,
			artifact.Document.Target.Kind,
		)
		if _, exists := vapIndex[key]; exists {
			return MutabilityValidationSummary{}, fmt.Errorf("duplicate vap update policy artifact for key %q", key)
		}
		vapIndex[key] = artifact
	}

	summary := MutabilityValidationSummary{
		Aggregate: MutabilityValidationAggregate{
			OverlayArtifacts: len(overlays),
			VAPArtifacts:     len(vaps),
		},
	}
	serviceIndex := make(map[string]*MutabilityValidationServiceSummary)
	seenResources := make(map[string]struct{}, len(overlays))

	for _, overlay := range overlays {
		key := mutabilityValidationResourceKey(
			overlay.Document.Resource.Service,
			overlay.Document.Resource.FormalSlug,
			overlay.Document.Resource.Kind,
		)
		vapArtifact, ok := vapIndex[key]
		if !ok {
			return MutabilityValidationSummary{}, fmt.Errorf(
				"missing vap update policy artifact for service %q kind %q formalSpec %q",
				overlay.Document.Resource.Service,
				overlay.Document.Resource.Kind,
				overlay.Document.Resource.FormalSlug,
			)
		}
		resource := buildMutabilityValidationResourceSummary(overlay, vapArtifact)
		summary.Resources = append(summary.Resources, resource)
		seenResources[key] = struct{}{}

		serviceSummary := serviceIndex[resource.Service]
		if serviceSummary == nil {
			serviceSummary = &MutabilityValidationServiceSummary{Service: resource.Service}
			serviceIndex[resource.Service] = serviceSummary
		}
		serviceSummary.Resources++
		accumulateMutabilityValidationCounts(&summary.Aggregate, resource)
		accumulateMutabilityValidationCounts(serviceSummary, resource)
	}

	for key, artifact := range vapIndex {
		if _, ok := seenResources[key]; ok {
			continue
		}
		return MutabilityValidationSummary{}, fmt.Errorf(
			"missing mutability overlay artifact for service %q kind %q formalSpec %q",
			artifact.Document.Target.Service,
			artifact.Document.Target.Kind,
			artifact.Document.Target.FormalSlug,
		)
	}

	summary.Aggregate.Services = len(serviceIndex)
	summary.Aggregate.Resources = len(summary.Resources)
	summary.Services = make([]MutabilityValidationServiceSummary, 0, len(serviceIndex))
	for _, service := range serviceIndex {
		summary.Services = append(summary.Services, *service)
	}
	return normalizeMutabilityValidationSummary(summary), nil
}

type mutabilityValidationCounter interface {
	GetFields() *int
	GetAllowInPlaceCount() *int
	GetDenyInPlaceCount() *int
	GetReplacementRequiredCount() *int
	GetUnknownPolicyCount() *int
	GetDocsDeniedCount() *int
	GetUnresolvedJoinCount() *int
	GetAmbiguousJoinCount() *int
	GetMergeConflictCount() *int
}

func accumulateMutabilityValidationCounts(counter mutabilityValidationCounter, resource MutabilityValidationResourceSummary) {
	*counter.GetFields() += resource.Fields
	*counter.GetAllowInPlaceCount() += resource.AllowInPlaceCount
	*counter.GetDenyInPlaceCount() += resource.DenyInPlaceCount
	*counter.GetReplacementRequiredCount() += resource.ReplacementRequiredCount
	*counter.GetUnknownPolicyCount() += resource.UnknownPolicyCount
	*counter.GetDocsDeniedCount() += len(resource.DocsDeniedFields)
	*counter.GetUnresolvedJoinCount() += len(resource.UnresolvedJoinFields)
	*counter.GetAmbiguousJoinCount() += len(resource.AmbiguousJoinFields)
	*counter.GetMergeConflictCount() += len(resource.MergeConflicts)
}

func (a *MutabilityValidationAggregate) GetFields() *int            { return &a.Fields }
func (a *MutabilityValidationAggregate) GetAllowInPlaceCount() *int { return &a.AllowInPlaceCount }
func (a *MutabilityValidationAggregate) GetDenyInPlaceCount() *int  { return &a.DenyInPlaceCount }
func (a *MutabilityValidationAggregate) GetReplacementRequiredCount() *int {
	return &a.ReplacementRequiredCount
}
func (a *MutabilityValidationAggregate) GetUnknownPolicyCount() *int  { return &a.UnknownPolicyCount }
func (a *MutabilityValidationAggregate) GetDocsDeniedCount() *int     { return &a.DocsDeniedCount }
func (a *MutabilityValidationAggregate) GetUnresolvedJoinCount() *int { return &a.UnresolvedJoinCount }
func (a *MutabilityValidationAggregate) GetAmbiguousJoinCount() *int  { return &a.AmbiguousJoinCount }
func (a *MutabilityValidationAggregate) GetMergeConflictCount() *int  { return &a.MergeConflictCount }

func (s *MutabilityValidationServiceSummary) GetFields() *int            { return &s.Fields }
func (s *MutabilityValidationServiceSummary) GetAllowInPlaceCount() *int { return &s.AllowInPlaceCount }
func (s *MutabilityValidationServiceSummary) GetDenyInPlaceCount() *int  { return &s.DenyInPlaceCount }
func (s *MutabilityValidationServiceSummary) GetReplacementRequiredCount() *int {
	return &s.ReplacementRequiredCount
}
func (s *MutabilityValidationServiceSummary) GetUnknownPolicyCount() *int {
	return &s.UnknownPolicyCount
}
func (s *MutabilityValidationServiceSummary) GetDocsDeniedCount() *int { return &s.DocsDeniedCount }
func (s *MutabilityValidationServiceSummary) GetUnresolvedJoinCount() *int {
	return &s.UnresolvedJoinCount
}
func (s *MutabilityValidationServiceSummary) GetAmbiguousJoinCount() *int {
	return &s.AmbiguousJoinCount
}
func (s *MutabilityValidationServiceSummary) GetMergeConflictCount() *int {
	return &s.MergeConflictCount
}

func buildMutabilityValidationResourceSummary(
	overlay loadedMutabilityOverlayArtifact,
	vap loadedVAPUpdatePolicyArtifact,
) MutabilityValidationResourceSummary {
	doc := overlay.Document
	vapDoc := vap.Document
	resource := MutabilityValidationResourceSummary{
		Service:              doc.Resource.Service,
		Kind:                 doc.Resource.Kind,
		FormalSlug:           doc.Resource.FormalSlug,
		ProviderResource:     doc.Resource.ProviderResource,
		TerraformDocsVersion: doc.Metadata.TerraformDocsVersion,
		OverlayPath:          overlay.RelativePath,
		VAPPath:              vap.RelativePath,
		Fields:               len(doc.Fields),
	}

	resource.AllowInPlacePaths = uniqueSortedStrings(append([]string(nil), vapDoc.Update.AllowInPlacePaths...))
	resource.DenyRules = make([]MutabilityValidationVAPDenyRule, 0, len(vapDoc.Update.DenyRules))
	for _, rule := range vapDoc.Update.DenyRules {
		resource.DenyRules = append(resource.DenyRules, MutabilityValidationVAPDenyRule{
			FieldPath:         rule.FieldPath,
			Decision:          rule.Decision,
			MergeCase:         rule.MergeCase,
			DocsEvidenceState: rule.DocsEvidenceState,
		})
	}
	sort.Slice(resource.DenyRules, func(i, j int) bool {
		if resource.DenyRules[i].FieldPath != resource.DenyRules[j].FieldPath {
			return resource.DenyRules[i].FieldPath < resource.DenyRules[j].FieldPath
		}
		return resource.DenyRules[i].Decision < resource.DenyRules[j].Decision
	})

	for _, field := range doc.Fields {
		resource.Decisions = append(resource.Decisions, MutabilityValidationFieldDecision{
			FieldPath:         field.ASTFieldPath,
			FinalPolicy:       field.Merge.FinalPolicy,
			JoinStatus:        field.Merge.JoinStatus,
			MergeCase:         field.Merge.MergeCase,
			DocsEvidenceState: field.Docs.EvidenceState,
		})

		switch field.Merge.FinalPolicy {
		case mutabilityOverlayPolicyAllowInPlaceUpdate:
			resource.AllowInPlaceCount++
		case mutabilityOverlayPolicyDenyInPlaceUpdate:
			resource.DenyInPlaceCount++
		case mutabilityOverlayPolicyReplacementRequired:
			resource.ReplacementRequiredCount++
		case mutabilityOverlayPolicyUnknown:
			resource.UnknownPolicyCount++
			resource.UnknownFields = append(resource.UnknownFields, field.ASTFieldPath)
		}
		if field.AST.UpdateCandidateState == mutabilityOverlayASTStateUpdateCandidate &&
			field.Docs.EvidenceState == mutabilityOverlayDocsStateDeniedUpdatable {
			resource.DocsDeniedFields = append(resource.DocsDeniedFields, field.ASTFieldPath)
		}
		switch field.Merge.JoinStatus {
		case mutabilityOverlayJoinUnresolved:
			resource.UnresolvedJoinFields = append(resource.UnresolvedJoinFields, field.ASTFieldPath)
		case mutabilityOverlayJoinAmbiguous:
			resource.AmbiguousJoinFields = append(resource.AmbiguousJoinFields, field.ASTFieldPath)
		}
		for _, reason := range mutabilityValidationOverlayFieldConflicts(field) {
			resource.MergeConflicts = append(resource.MergeConflicts, MutabilityValidationMergeConflict{
				FieldPath: field.ASTFieldPath,
				Reason:    reason,
			})
		}
	}

	resource.MergeConflicts = append(resource.MergeConflicts, mutabilityValidationProjectionConflicts(doc, vapDoc)...)
	return normalizeMutabilityValidationResourceSummary(resource)
}

func mutabilityValidationOverlayFieldConflicts(field mutabilityOverlayField) []string {
	expected := resolveMutabilityOverlayDecision(
		field.AST.UpdateCandidateState,
		field.AST.ForceNew,
		field.Docs.EvidenceState,
		field.Merge.JoinStatus,
	)

	conflicts := make([]string, 0)
	if field.Merge.MergeCase != expected.MergeCase || field.Merge.FinalPolicy != expected.FinalPolicy {
		conflicts = append(
			conflicts,
			fmt.Sprintf(
				"merge result %s/%s does not match expected %s/%s",
				field.Merge.MergeCase,
				field.Merge.FinalPolicy,
				expected.MergeCase,
				expected.FinalPolicy,
			),
		)
	}
	if field.AST.ForceNew && field.Docs.EvidenceState != mutabilityOverlayDocsStateNotDocumented {
		conflicts = append(conflicts, "forceNew field unexpectedly carried docs overlay evidence")
	}
	if field.AST.UpdateCandidateState != mutabilityOverlayASTStateUpdateCandidate &&
		field.Docs.EvidenceState != mutabilityOverlayDocsStateNotDocumented {
		conflicts = append(conflicts, "non-updateCandidate field unexpectedly carried docs overlay evidence")
	}
	return uniqueSortedStrings(conflicts)
}

func mutabilityValidationProjectionConflicts(
	overlayDoc mutabilityOverlayDocument,
	vapDoc vapUpdatePolicyDocument,
) []MutabilityValidationMergeConflict {
	expectedAllowPaths := make(map[string]struct{})
	expectedDenyRules := make(map[string]MutabilityValidationVAPDenyRule)
	for _, field := range overlayDoc.Fields {
		switch field.Merge.FinalPolicy {
		case mutabilityOverlayPolicyAllowInPlaceUpdate:
			expectedAllowPaths[field.ASTFieldPath] = struct{}{}
		case mutabilityOverlayPolicyDenyInPlaceUpdate, mutabilityOverlayPolicyReplacementRequired, mutabilityOverlayPolicyUnknown:
			expectedDenyRules[field.ASTFieldPath] = MutabilityValidationVAPDenyRule{
				FieldPath:         field.ASTFieldPath,
				Decision:          field.Merge.FinalPolicy,
				MergeCase:         field.Merge.MergeCase,
				DocsEvidenceState: field.Docs.EvidenceState,
			}
		}
	}

	actualAllowPaths := make(map[string]struct{}, len(vapDoc.Update.AllowInPlacePaths))
	for _, fieldPath := range vapDoc.Update.AllowInPlacePaths {
		actualAllowPaths[fieldPath] = struct{}{}
	}

	conflicts := make([]MutabilityValidationMergeConflict, 0)
	for fieldPath := range expectedAllowPaths {
		if _, ok := actualAllowPaths[fieldPath]; ok {
			continue
		}
		conflicts = append(conflicts, MutabilityValidationMergeConflict{
			FieldPath: fieldPath,
			Reason:    "vap update policy is missing an allowInPlace path required by the mutability overlay",
		})
	}
	for fieldPath := range actualAllowPaths {
		if _, ok := expectedAllowPaths[fieldPath]; ok {
			continue
		}
		conflicts = append(conflicts, MutabilityValidationMergeConflict{
			FieldPath: fieldPath,
			Reason:    "vap update policy widened allowInPlace beyond the mutability overlay",
		})
	}

	actualDenyRules := make(map[string]MutabilityValidationVAPDenyRule)
	for _, rule := range vapDoc.Update.DenyRules {
		if existing, exists := actualDenyRules[rule.FieldPath]; exists {
			conflicts = append(conflicts, MutabilityValidationMergeConflict{
				FieldPath: rule.FieldPath,
				Reason: fmt.Sprintf(
					"vap update policy emitted duplicate deny rules for %q (%s and %s)",
					rule.FieldPath,
					existing.Decision,
					rule.Decision,
				),
			})
			continue
		}
		actualDenyRules[rule.FieldPath] = MutabilityValidationVAPDenyRule{
			FieldPath:         rule.FieldPath,
			Decision:          rule.Decision,
			MergeCase:         rule.MergeCase,
			DocsEvidenceState: rule.DocsEvidenceState,
		}
	}

	for fieldPath, expectedRule := range expectedDenyRules {
		actualRule, ok := actualDenyRules[fieldPath]
		if !ok {
			conflicts = append(conflicts, MutabilityValidationMergeConflict{
				FieldPath: fieldPath,
				Reason:    "vap update policy is missing a deny rule required by the mutability overlay",
			})
			continue
		}
		if actualRule.Decision != expectedRule.Decision ||
			actualRule.MergeCase != expectedRule.MergeCase ||
			actualRule.DocsEvidenceState != expectedRule.DocsEvidenceState {
			conflicts = append(conflicts, MutabilityValidationMergeConflict{
				FieldPath: fieldPath,
				Reason: fmt.Sprintf(
					"vap update policy deny rule %s/%s/%s does not match mutability overlay %s/%s/%s",
					actualRule.Decision,
					actualRule.MergeCase,
					actualRule.DocsEvidenceState,
					expectedRule.Decision,
					expectedRule.MergeCase,
					expectedRule.DocsEvidenceState,
				),
			})
		}
	}
	for fieldPath := range actualDenyRules {
		if _, ok := expectedDenyRules[fieldPath]; ok {
			continue
		}
		conflicts = append(conflicts, MutabilityValidationMergeConflict{
			FieldPath: fieldPath,
			Reason:    "vap update policy emitted an unexpected deny rule that is not backed by the mutability overlay",
		})
	}

	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].FieldPath != conflicts[j].FieldPath {
			return conflicts[i].FieldPath < conflicts[j].FieldPath
		}
		return conflicts[i].Reason < conflicts[j].Reason
	})
	return conflicts
}

func normalizeMutabilityValidationSummary(summary MutabilityValidationSummary) MutabilityValidationSummary {
	normalized := summary
	normalized.Services = append([]MutabilityValidationServiceSummary(nil), summary.Services...)
	for i := range normalized.Services {
		normalized.Services[i].Service = strings.TrimSpace(normalized.Services[i].Service)
	}
	sort.Slice(normalized.Services, func(i, j int) bool {
		return normalized.Services[i].Service < normalized.Services[j].Service
	})

	normalized.Resources = append([]MutabilityValidationResourceSummary(nil), summary.Resources...)
	for i := range normalized.Resources {
		normalized.Resources[i] = normalizeMutabilityValidationResourceSummary(normalized.Resources[i])
	}
	sort.Slice(normalized.Resources, func(i, j int) bool {
		leftKey := mutabilityValidationResourceKey(
			normalized.Resources[i].Service,
			normalized.Resources[i].FormalSlug,
			normalized.Resources[i].Kind,
		)
		rightKey := mutabilityValidationResourceKey(
			normalized.Resources[j].Service,
			normalized.Resources[j].FormalSlug,
			normalized.Resources[j].Kind,
		)
		return leftKey < rightKey
	})
	return normalized
}

func normalizeMutabilityValidationResourceSummary(resource MutabilityValidationResourceSummary) MutabilityValidationResourceSummary {
	normalized := resource
	normalized.Service = strings.TrimSpace(normalized.Service)
	normalized.Kind = strings.TrimSpace(normalized.Kind)
	normalized.FormalSlug = strings.TrimSpace(normalized.FormalSlug)
	normalized.ProviderResource = strings.TrimSpace(normalized.ProviderResource)
	normalized.TerraformDocsVersion = strings.TrimSpace(normalized.TerraformDocsVersion)
	normalized.OverlayPath = filepath.ToSlash(strings.TrimSpace(normalized.OverlayPath))
	normalized.VAPPath = filepath.ToSlash(strings.TrimSpace(normalized.VAPPath))
	normalized.AllowInPlacePaths = uniqueSortedStrings(normalized.AllowInPlacePaths)
	normalized.DocsDeniedFields = uniqueSortedStrings(normalized.DocsDeniedFields)
	normalized.UnknownFields = uniqueSortedStrings(normalized.UnknownFields)
	normalized.UnresolvedJoinFields = uniqueSortedStrings(normalized.UnresolvedJoinFields)
	normalized.AmbiguousJoinFields = uniqueSortedStrings(normalized.AmbiguousJoinFields)

	normalized.Decisions = append([]MutabilityValidationFieldDecision(nil), normalized.Decisions...)
	sort.Slice(normalized.Decisions, func(i, j int) bool {
		return normalized.Decisions[i].FieldPath < normalized.Decisions[j].FieldPath
	})
	normalized.DenyRules = append([]MutabilityValidationVAPDenyRule(nil), normalized.DenyRules...)
	sort.Slice(normalized.DenyRules, func(i, j int) bool {
		if normalized.DenyRules[i].FieldPath != normalized.DenyRules[j].FieldPath {
			return normalized.DenyRules[i].FieldPath < normalized.DenyRules[j].FieldPath
		}
		return normalized.DenyRules[i].Decision < normalized.DenyRules[j].Decision
	})
	normalized.MergeConflicts = append([]MutabilityValidationMergeConflict(nil), normalized.MergeConflicts...)
	sort.Slice(normalized.MergeConflicts, func(i, j int) bool {
		if normalized.MergeConflicts[i].FieldPath != normalized.MergeConflicts[j].FieldPath {
			return normalized.MergeConflicts[i].FieldPath < normalized.MergeConflicts[j].FieldPath
		}
		return normalized.MergeConflicts[i].Reason < normalized.MergeConflicts[j].Reason
	})
	return normalized
}

func appendMutabilityValidationRegressions[T mutabilityValidationMetrics](
	messages *[]string,
	scope string,
	baseline T,
	current T,
) {
	if current.GetAllowInPlaceCountValue() > baseline.GetAllowInPlaceCountValue() {
		*messages = append(*messages, fmt.Sprintf(
			"%s allowInPlaceCount increased from %d to %d",
			scope,
			baseline.GetAllowInPlaceCountValue(),
			current.GetAllowInPlaceCountValue(),
		))
	}
	if current.GetUnknownPolicyCountValue() > baseline.GetUnknownPolicyCountValue() {
		*messages = append(*messages, fmt.Sprintf(
			"%s unknownPolicyCount increased from %d to %d",
			scope,
			baseline.GetUnknownPolicyCountValue(),
			current.GetUnknownPolicyCountValue(),
		))
	}
	if current.GetUnresolvedJoinCountValue() > baseline.GetUnresolvedJoinCountValue() {
		*messages = append(*messages, fmt.Sprintf(
			"%s unresolvedJoinCount increased from %d to %d",
			scope,
			baseline.GetUnresolvedJoinCountValue(),
			current.GetUnresolvedJoinCountValue(),
		))
	}
	if current.GetAmbiguousJoinCountValue() > baseline.GetAmbiguousJoinCountValue() {
		*messages = append(*messages, fmt.Sprintf(
			"%s ambiguousJoinCount increased from %d to %d",
			scope,
			baseline.GetAmbiguousJoinCountValue(),
			current.GetAmbiguousJoinCountValue(),
		))
	}
	if current.GetMergeConflictCountValue() > baseline.GetMergeConflictCountValue() {
		*messages = append(*messages, fmt.Sprintf(
			"%s mergeConflictCount increased from %d to %d",
			scope,
			baseline.GetMergeConflictCountValue(),
			current.GetMergeConflictCountValue(),
		))
	}
}

type mutabilityValidationMetrics interface {
	GetAllowInPlaceCountValue() int
	GetUnknownPolicyCountValue() int
	GetUnresolvedJoinCountValue() int
	GetAmbiguousJoinCountValue() int
	GetMergeConflictCountValue() int
}

func (a MutabilityValidationAggregate) GetAllowInPlaceCountValue() int  { return a.AllowInPlaceCount }
func (a MutabilityValidationAggregate) GetUnknownPolicyCountValue() int { return a.UnknownPolicyCount }
func (a MutabilityValidationAggregate) GetUnresolvedJoinCountValue() int {
	return a.UnresolvedJoinCount
}
func (a MutabilityValidationAggregate) GetAmbiguousJoinCountValue() int { return a.AmbiguousJoinCount }
func (a MutabilityValidationAggregate) GetMergeConflictCountValue() int { return a.MergeConflictCount }

func (s MutabilityValidationServiceSummary) GetAllowInPlaceCountValue() int {
	return s.AllowInPlaceCount
}
func (s MutabilityValidationServiceSummary) GetUnknownPolicyCountValue() int {
	return s.UnknownPolicyCount
}
func (s MutabilityValidationServiceSummary) GetUnresolvedJoinCountValue() int {
	return s.UnresolvedJoinCount
}
func (s MutabilityValidationServiceSummary) GetAmbiguousJoinCountValue() int {
	return s.AmbiguousJoinCount
}
func (s MutabilityValidationServiceSummary) GetMergeConflictCountValue() int {
	return s.MergeConflictCount
}

func compareMutabilityValidationFieldDecisions(
	comparison *MutabilityValidationComparison,
	currentResource MutabilityValidationResourceSummary,
	baselineResource MutabilityValidationResourceSummary,
) {
	currentDecisions := make(map[string]MutabilityValidationFieldDecision, len(currentResource.Decisions))
	for _, decision := range currentResource.Decisions {
		currentDecisions[decision.FieldPath] = decision
	}
	baselineDecisions := make(map[string]MutabilityValidationFieldDecision, len(baselineResource.Decisions))
	for _, decision := range baselineResource.Decisions {
		baselineDecisions[decision.FieldPath] = decision
	}

	currentFields := sortedMutabilityValidationDecisionFields(currentDecisions)
	baselineFields := sortedMutabilityValidationDecisionFields(baselineDecisions)
	for _, fieldPath := range difference(baselineFields, currentFields) {
		comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf(
			"resource %s/%s (%s) removed field decision %q",
			currentResource.Service,
			currentResource.Kind,
			currentResource.FormalSlug,
			fieldPath,
		))
	}
	for _, fieldPath := range difference(currentFields, baselineFields) {
		comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf(
			"resource %s/%s (%s) added field decision %q",
			currentResource.Service,
			currentResource.Kind,
			currentResource.FormalSlug,
			fieldPath,
		))
	}
	for _, fieldPath := range intersection(currentFields, baselineFields) {
		currentDecision := currentDecisions[fieldPath]
		baselineDecision := baselineDecisions[fieldPath]
		if currentDecision.FinalPolicy != baselineDecision.FinalPolicy {
			if currentDecision.FinalPolicy == mutabilityOverlayPolicyAllowInPlaceUpdate &&
				baselineDecision.FinalPolicy != mutabilityOverlayPolicyAllowInPlaceUpdate {
				comparison.Regressions = append(comparison.Regressions, fmt.Sprintf(
					"resource %s/%s (%s) field %q widened finalPolicy from %q to %q",
					currentResource.Service,
					currentResource.Kind,
					currentResource.FormalSlug,
					fieldPath,
					baselineDecision.FinalPolicy,
					currentDecision.FinalPolicy,
				))
			} else {
				comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf(
					"resource %s/%s (%s) field %q finalPolicy changed from %q to %q",
					currentResource.Service,
					currentResource.Kind,
					currentResource.FormalSlug,
					fieldPath,
					baselineDecision.FinalPolicy,
					currentDecision.FinalPolicy,
				))
			}
		}
		if joinSeverity(currentDecision.JoinStatus) > joinSeverity(baselineDecision.JoinStatus) {
			comparison.Regressions = append(comparison.Regressions, fmt.Sprintf(
				"resource %s/%s (%s) field %q joinStatus regressed from %q to %q",
				currentResource.Service,
				currentResource.Kind,
				currentResource.FormalSlug,
				fieldPath,
				baselineDecision.JoinStatus,
				currentDecision.JoinStatus,
			))
		} else if currentDecision.JoinStatus != baselineDecision.JoinStatus {
			comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf(
				"resource %s/%s (%s) field %q joinStatus changed from %q to %q",
				currentResource.Service,
				currentResource.Kind,
				currentResource.FormalSlug,
				fieldPath,
				baselineDecision.JoinStatus,
				currentDecision.JoinStatus,
			))
		}
		if currentDecision.MergeCase != baselineDecision.MergeCase {
			comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf(
				"resource %s/%s (%s) field %q mergeCase changed from %q to %q",
				currentResource.Service,
				currentResource.Kind,
				currentResource.FormalSlug,
				fieldPath,
				baselineDecision.MergeCase,
				currentDecision.MergeCase,
			))
		}
		if currentDecision.DocsEvidenceState != baselineDecision.DocsEvidenceState {
			comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf(
				"resource %s/%s (%s) field %q docsEvidenceState changed from %q to %q",
				currentResource.Service,
				currentResource.Kind,
				currentResource.FormalSlug,
				fieldPath,
				baselineDecision.DocsEvidenceState,
				currentDecision.DocsEvidenceState,
			))
		}
	}
}

func compareMutabilityValidationMergeConflicts(
	comparison *MutabilityValidationComparison,
	currentResource MutabilityValidationResourceSummary,
	baselineResource MutabilityValidationResourceSummary,
) {
	currentConflicts := make(map[string]struct{}, len(currentResource.MergeConflicts))
	for _, conflict := range currentResource.MergeConflicts {
		currentConflicts[mutabilityValidationMergeConflictKey(conflict)] = struct{}{}
	}
	baselineConflicts := make(map[string]struct{}, len(baselineResource.MergeConflicts))
	for _, conflict := range baselineResource.MergeConflicts {
		baselineConflicts[mutabilityValidationMergeConflictKey(conflict)] = struct{}{}
	}

	currentKeys := sortedMutabilityValidationMergeConflictKeys(currentConflicts)
	baselineKeys := sortedMutabilityValidationMergeConflictKeys(baselineConflicts)
	for _, key := range difference(currentKeys, baselineKeys) {
		comparison.Regressions = append(comparison.Regressions, fmt.Sprintf(
			"resource %s/%s (%s) added merge conflict %s",
			currentResource.Service,
			currentResource.Kind,
			currentResource.FormalSlug,
			key,
		))
	}
	for _, key := range difference(baselineKeys, currentKeys) {
		comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf(
			"resource %s/%s (%s) removed merge conflict %s",
			currentResource.Service,
			currentResource.Kind,
			currentResource.FormalSlug,
			key,
		))
	}
}

func mutabilityValidationResourceKey(service string, formalSlug string, kind string) string {
	service = strings.TrimSpace(service)
	formalSlug = strings.TrimSpace(formalSlug)
	kind = strings.TrimSpace(kind)
	if formalSlug == "" {
		formalSlug = fileStem(kind)
	}
	return service + "\x00" + formalSlug + "\x00" + kind
}

func mutabilityValidationMergeConflictKey(conflict MutabilityValidationMergeConflict) string {
	return strings.TrimSpace(conflict.FieldPath) + "\x00" + strings.TrimSpace(conflict.Reason)
}

func diffMutabilityValidationStringSets(current []string, baseline []string) ([]string, []string) {
	currentSet := make(map[string]struct{}, len(current))
	for _, value := range current {
		currentSet[value] = struct{}{}
	}
	baselineSet := make(map[string]struct{}, len(baseline))
	for _, value := range baseline {
		baselineSet[value] = struct{}{}
	}

	added := make([]string, 0)
	for _, value := range current {
		if _, ok := baselineSet[value]; ok {
			continue
		}
		added = append(added, value)
	}
	removed := make([]string, 0)
	for _, value := range baseline {
		if _, ok := currentSet[value]; ok {
			continue
		}
		removed = append(removed, value)
	}
	return uniqueSortedStrings(added), uniqueSortedStrings(removed)
}

func sortedMutabilityValidationServiceNames(values map[string]MutabilityValidationServiceSummary) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedMutabilityValidationResourceKeys(values map[string]MutabilityValidationResourceSummary) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMutabilityValidationDecisionFields(values map[string]MutabilityValidationFieldDecision) []string {
	fields := make([]string, 0, len(values))
	for field := range values {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

func sortedMutabilityValidationMergeConflictKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func joinSeverity(status string) int {
	switch strings.TrimSpace(status) {
	case mutabilityOverlayJoinMatched:
		return 0
	case mutabilityOverlayJoinUnresolved:
		return 1
	case mutabilityOverlayJoinAmbiguous:
		return 2
	default:
		return 3
	}
}

func appendUniqueString(existing []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || slicesContainString(existing, value) {
		return existing
	}
	return append(existing, value)
}

func difference(left []string, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range right {
		rightSet[value] = struct{}{}
	}
	out := make([]string, 0)
	for _, value := range left {
		if _, ok := rightSet[value]; ok {
			continue
		}
		out = append(out, value)
	}
	return out
}

func intersection(left []string, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range right {
		rightSet[value] = struct{}{}
	}
	out := make([]string, 0)
	for _, value := range left {
		if _, ok := rightSet[value]; ok {
			out = append(out, value)
		}
	}
	return out
}

func flattenMutabilityValidationErrors(err error) []error {
	if err == nil {
		return nil
	}

	type singleUnwrap interface {
		Unwrap() error
	}
	type multiUnwrap interface {
		Unwrap() []error
	}

	switch typed := err.(type) {
	case multiUnwrap:
		out := make([]error, 0)
		for _, child := range typed.Unwrap() {
			out = append(out, flattenMutabilityValidationErrors(child)...)
		}
		return out
	case singleUnwrap:
		child := typed.Unwrap()
		if child == nil {
			return []error{err}
		}
		return flattenMutabilityValidationErrors(child)
	default:
		return []error{err}
	}
}

package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultMutabilityOverlayTerraformDocsVersion = "7.22.0"

	mutabilityOverlayGeneratedRootRelativePath = "internal/generator/generated/mutability_overlay"
	mutabilityOverlayDocsFixtureRootRelative   = "internal/generator/testdata/mutability_overlay/docs"

	mutabilityOverlayGenerationErrorMissingSourceRevision = "missingSourceRevision"
	mutabilityOverlayGenerationErrorASTJoinFailed         = "astJoinFailed"
	mutabilityOverlayGenerationErrorInvalidDocument       = "invalidDocument"
)

type mutabilityOverlayGeneratedArtifact struct {
	Service      string
	RelativePath string
	Document     mutabilityOverlayDocument
}

type mutabilityOverlayASTFieldInput struct {
	FieldPath            string
	UpdateCandidateState string
	ForceNew             bool
	ConflictsWith        []string
	ASTSourceBucket      string
}

type mutabilityOverlaySourceLockFile struct {
	Sources []mutabilityOverlaySourceLockEntry `json:"sources"`
}

type mutabilityOverlaySourceLockEntry struct {
	Name     string `json:"name"`
	Revision string `json:"revision"`
}

type mutabilityOverlayGenerationError struct {
	Reason           string
	Service          string
	Kind             string
	FormalSlug       string
	ProviderResource string
	Detail           string
}

func (e *mutabilityOverlayGenerationError) Error() string {
	if e == nil {
		return "<nil>"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "mutability overlay generation failed for service %q kind %q", e.Service, e.Kind)
	if strings.TrimSpace(e.FormalSlug) != "" {
		fmt.Fprintf(&b, " formalSpec %q", e.FormalSlug)
	}
	if strings.TrimSpace(e.ProviderResource) != "" {
		fmt.Fprintf(&b, " providerResource=%q", e.ProviderResource)
	}
	fmt.Fprintf(&b, ": %s", e.Reason)
	if strings.TrimSpace(e.Detail) != "" {
		fmt.Fprintf(&b, " (%s)", e.Detail)
	}
	return b.String()
}

func (g *Generator) buildMutabilityOverlayArtifacts(
	ctx context.Context,
	cfg *Config,
	packages []*PackageModel,
) ([]mutabilityOverlayGeneratedArtifact, error) {
	contract, err := newMutabilityOverlayDocsContract(g.mutabilityOverlayTerraformDocsVersion())
	if err != nil {
		return nil, err
	}

	fixtureRoot := g.mutabilityOverlayDocsFixtureRoot(cfg)
	var (
		artifacts       []mutabilityOverlayGeneratedArtifact
		sourceRevisions map[string]string
		errs            []error
	)
	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		for _, resource := range pkg.Resources {
			astFields := mutabilityOverlayASTFields(resource)
			if len(astFields) == 0 || resource.Formal == nil {
				continue
			}

			if sourceRevisions == nil {
				sourceRevisions, err = loadMutabilityOverlaySourceRevisions(cfg.FormalRoot())
				if err != nil {
					return nil, err
				}
			}

			target, err := resolveMutabilityOverlayRegistryPageTarget(pkg.Service.Service, resource, contract, nil)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			var parsedDocs *mutabilityOverlayDocsParseResult
			if mutabilityOverlayNeedsDocs(astFields) {
				input, err := g.loadMutabilityOverlayDocsInput(ctx, fixtureRoot, target)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				parsedResult, err := parseMutabilityOverlayDocsArgumentReference(input)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				parsedDocs = &parsedResult
			}

			providerRevision, err := mutabilityOverlaySourceRevisionForResource(sourceRevisions, resource)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			doc, err := buildMutabilityOverlayDocument(pkg.Service.Service, resource, target, providerRevision, parsedDocs, astFields)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			artifacts = append(artifacts, newMutabilityOverlayGeneratedArtifact(pkg.Service.Service, resource, doc))
		}
	}

	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].RelativePath < artifacts[j].RelativePath
	})
	if len(errs) != 0 {
		return artifacts, errors.Join(errs...)
	}
	return artifacts, nil
}

func (g *Generator) mutabilityOverlayTerraformDocsVersion() string {
	if g == nil || strings.TrimSpace(g.mutabilityOverlayDocsVersion) == "" {
		return defaultMutabilityOverlayTerraformDocsVersion
	}
	return strings.TrimSpace(g.mutabilityOverlayDocsVersion)
}

func (g *Generator) mutabilityOverlayDocsFixtureRoot(cfg *Config) string {
	if g != nil && strings.TrimSpace(g.mutabilityOverlayFixtureRoot) != "" {
		return strings.TrimSpace(g.mutabilityOverlayFixtureRoot)
	}
	if cfg == nil || strings.TrimSpace(cfg.configDir) == "" {
		return ""
	}
	repoRoot := filepath.Clean(filepath.Join(cfg.configDir, "..", "..", ".."))
	return filepath.Join(repoRoot, filepath.FromSlash(mutabilityOverlayDocsFixtureRootRelative))
}

func (g *Generator) loadMutabilityOverlayDocsInput(
	ctx context.Context,
	fixtureRoot string,
	target mutabilityOverlayRegistryPageTarget,
) (mutabilityOverlayDocsInput, error) {
	fixtureRoot = strings.TrimSpace(fixtureRoot)
	if fixtureRoot != "" {
		inputs, err := loadMutabilityOverlayDocsFixtures(fixtureRoot, []mutabilityOverlayRegistryPageTarget{target})
		switch {
		case err == nil && len(inputs) == 1:
			return inputs[0], nil
		case err != nil && !isMutabilityOverlayMissingFixtureError(err):
			return mutabilityOverlayDocsInput{}, err
		}
	}

	fetcher := g.mutabilityOverlayDocsFetcher
	if fetcher == nil {
		fetcher = newMutabilityOverlayHTTPDocsFetcher(nil)
	}

	inputs, err := acquireMutabilityOverlayDocsInputs(ctx, []mutabilityOverlayRegistryPageTarget{target}, fetcher)
	if err != nil {
		return mutabilityOverlayDocsInput{}, err
	}
	if len(inputs) != 1 {
		return mutabilityOverlayDocsInput{}, fmt.Errorf(
			"expected exactly one mutability overlay docs input for service %q kind %q, got %d",
			target.Service,
			target.Kind,
			len(inputs),
		)
	}
	return inputs[0], nil
}

func isMutabilityOverlayMissingFixtureError(err error) bool {
	var acquisitionErr *mutabilityOverlayDocsAcquisitionError
	return errors.As(err, &acquisitionErr) && acquisitionErr.Reason == mutabilityOverlayDocsErrorMissingFixture
}

func mutabilityOverlayASTFields(resource ResourceModel) []mutabilityOverlayASTFieldInput {
	if resource.Formal == nil {
		return nil
	}

	updateCandidates := normalizeFormalPaths(resource.Formal.Binding.Import.Mutation.Mutable)
	forceNew := normalizeFormalPaths(resource.Formal.Binding.Import.Mutation.ForceNew)
	if len(updateCandidates) == 0 && len(forceNew) == 0 {
		return nil
	}

	updateSet := make(map[string]struct{}, len(updateCandidates))
	for _, fieldPath := range updateCandidates {
		updateSet[fieldPath] = struct{}{}
	}
	forceNewSet := make(map[string]struct{}, len(forceNew))
	for _, fieldPath := range forceNew {
		forceNewSet[fieldPath] = struct{}{}
	}
	conflicts := normalizeFormalConflicts(resource.Formal.Binding.Import.Mutation.ConflictsWith)

	allPaths := uniqueSortedStrings(append(append([]string(nil), updateCandidates...), forceNew...))
	fields := make([]mutabilityOverlayASTFieldInput, 0, len(allPaths))
	for _, fieldPath := range allPaths {
		_, isUpdateCandidate := updateSet[fieldPath]
		_, isForceNew := forceNewSet[fieldPath]
		sourceBucket := mutabilityOverlayASTSourceBucketMutable
		if isForceNew {
			sourceBucket = mutabilityOverlayASTSourceBucketForceNew
		}
		state := mutabilityOverlayASTStateNotUpdateCandidate
		if isUpdateCandidate {
			state = mutabilityOverlayASTStateUpdateCandidate
		}
		fields = append(fields, mutabilityOverlayASTFieldInput{
			FieldPath:            fieldPath,
			UpdateCandidateState: state,
			ForceNew:             isForceNew,
			ConflictsWith:        append([]string(nil), conflicts[fieldPath]...),
			ASTSourceBucket:      sourceBucket,
		})
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].FieldPath < fields[j].FieldPath
	})
	return fields
}

func mutabilityOverlayNeedsDocs(fields []mutabilityOverlayASTFieldInput) bool {
	for _, field := range fields {
		if field.UpdateCandidateState == mutabilityOverlayASTStateUpdateCandidate && !field.ForceNew {
			return true
		}
	}
	return false
}

func loadMutabilityOverlaySourceRevisions(formalRoot string) (map[string]string, error) {
	formalRoot = strings.TrimSpace(formalRoot)
	if formalRoot == "" {
		return nil, fmt.Errorf("formal root is required to load mutability overlay source pins")
	}

	content, err := os.ReadFile(filepath.Join(formalRoot, "sources.lock"))
	if err != nil {
		return nil, fmt.Errorf("read formal sources.lock: %w", err)
	}

	var lockFile mutabilityOverlaySourceLockFile
	if err := json.Unmarshal(content, &lockFile); err != nil {
		return nil, fmt.Errorf("decode formal sources.lock: %w", err)
	}

	revisions := make(map[string]string, len(lockFile.Sources))
	for _, source := range lockFile.Sources {
		name := strings.TrimSpace(source.Name)
		revision := strings.TrimSpace(source.Revision)
		if name == "" || revision == "" {
			continue
		}
		revisions[name] = revision
	}
	return revisions, nil
}

func mutabilityOverlaySourceRevisionForResource(
	revisions map[string]string,
	resource ResourceModel,
) (string, error) {
	if resource.Formal == nil {
		return "", errors.New("mutability overlay source revision requires a formal model")
	}

	sourceRef := strings.TrimSpace(resource.Formal.Binding.Import.SourceRef)
	if sourceRef == "" {
		sourceRef = mutabilityOverlayProviderSourceRef
	}
	revision := strings.TrimSpace(revisions[sourceRef])
	if revision != "" {
		return revision, nil
	}

	return "", &mutabilityOverlayGenerationError{
		Reason:           mutabilityOverlayGenerationErrorMissingSourceRevision,
		Service:          resource.Formal.Reference.Service,
		Kind:             resource.Kind,
		FormalSlug:       resource.Formal.Reference.Slug,
		ProviderResource: strings.TrimSpace(resource.Formal.Binding.Import.ProviderResource),
		Detail:           fmt.Sprintf("sources.lock does not pin a revision for sourceRef %q", sourceRef),
	}
}

func buildMutabilityOverlayDocument(
	service string,
	resource ResourceModel,
	target mutabilityOverlayRegistryPageTarget,
	providerRevision string,
	parsedDocs *mutabilityOverlayDocsParseResult,
	astFields []mutabilityOverlayASTFieldInput,
) (mutabilityOverlayDocument, error) {
	if resource.Formal == nil {
		return mutabilityOverlayDocument{}, errors.New("mutability overlay document requires a formal model")
	}

	sourceRef := strings.TrimSpace(resource.Formal.Binding.Import.SourceRef)
	if sourceRef == "" {
		sourceRef = mutabilityOverlayProviderSourceRef
	}
	repoAuthoredSpecPath := strings.TrimSpace(resource.Formal.Binding.Import.Boundary.RepoAuthoredSpecPath)
	if repoAuthoredSpecPath == "" {
		repoAuthoredSpecPath = strings.TrimSpace(resource.Formal.Binding.Manifest.SpecPath)
	}

	resolver := newMutabilityOverlayPathResolver(resource)
	fields := make([]mutabilityOverlayField, 0, len(astFields))
	for _, astField := range astFields {
		field, err := buildMutabilityOverlayField(service, resource, resolver, astField, target, parsedDocs, sourceRef, repoAuthoredSpecPath)
		if err != nil {
			return mutabilityOverlayDocument{}, err
		}
		fields = append(fields, field)
	}
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].CanonicalJoinKey != fields[j].CanonicalJoinKey {
			return fields[i].CanonicalJoinKey < fields[j].CanonicalJoinKey
		}
		if fields[i].ASTFieldPath != fields[j].ASTFieldPath {
			return fields[i].ASTFieldPath < fields[j].ASTFieldPath
		}
		return fields[i].TerraformFieldPath < fields[j].TerraformFieldPath
	})

	doc := mutabilityOverlayDocument{
		SchemaVersion:   mutabilityOverlaySchemaVersion,
		Surface:         mutabilityOverlaySurface,
		ContractVersion: mutabilityOverlayContractVersion,
		Metadata: mutabilityOverlayMetadata{
			ProviderSourceRef:    sourceRef,
			ProviderRevision:     strings.TrimSpace(providerRevision),
			TerraformDocsVersion: target.TerraformDocsVersion,
		},
		SourceContract: mutabilityOverlaySourceContract{
			ASTPrimaryFacts:    append([]string(nil), mutabilityOverlayASTPrimaryFacts...),
			ASTMutableAlias:    mutabilityOverlayUpdateCandidateAlias,
			DocsOverlayScope:   mutabilityOverlayDocsOverlayScope,
			DocsEvidenceSource: mutabilityOverlayDocsEvidenceSource,
		},
		Resource: mutabilityOverlayResource{
			Service:              service,
			Kind:                 resource.Kind,
			FormalSlug:           resource.Formal.Reference.Slug,
			ProviderResource:     target.ProviderResource,
			RepoAuthoredSpecPath: repoAuthoredSpecPath,
			FormalImportPath:     resource.Formal.Binding.Manifest.ImportPath,
		},
		Fields: fields,
	}

	if err := validateMutabilityOverlayDocument(doc); err != nil {
		return mutabilityOverlayDocument{}, &mutabilityOverlayGenerationError{
			Reason:           mutabilityOverlayGenerationErrorInvalidDocument,
			Service:          service,
			Kind:             resource.Kind,
			FormalSlug:       resource.Formal.Reference.Slug,
			ProviderResource: target.ProviderResource,
			Detail:           err.Error(),
		}
	}
	return doc, nil
}

func buildMutabilityOverlayField(
	service string,
	resource ResourceModel,
	resolver mutabilityOverlayPathResolver,
	astField mutabilityOverlayASTFieldInput,
	target mutabilityOverlayRegistryPageTarget,
	parsedDocs *mutabilityOverlayDocsParseResult,
	sourceRef string,
	repoAuthoredSpecPath string,
) (mutabilityOverlayField, error) {
	astResolution := resolver.Resolve(astField.FieldPath)
	if astResolution.Status != mutabilityOverlayJoinMatched || len(astResolution.Candidates) != 1 {
		return mutabilityOverlayField{}, &mutabilityOverlayGenerationError{
			Reason:           mutabilityOverlayGenerationErrorASTJoinFailed,
			Service:          service,
			Kind:             resource.Kind,
			FormalSlug:       resource.Formal.Reference.Slug,
			ProviderResource: target.ProviderResource,
			Detail:           strings.Join(append([]string{fmt.Sprintf("AST field path %q did not resolve to exactly one canonical join key", astField.FieldPath)}, astResolution.Diagnostics...), "; "),
		}
	}

	docsSection := mutabilityOverlayDocsSectionArgumentReference
	docsAnchor := ""
	if parsedDocs != nil {
		if title := strings.TrimSpace(parsedDocs.SectionTitle); title != "" {
			docsSection = title
		}
		docsAnchor = strings.TrimSpace(parsedDocs.SectionAnchor)
	}

	field := mutabilityOverlayField{
		ASTFieldPath:       astField.FieldPath,
		TerraformFieldPath: astField.FieldPath,
		CanonicalJoinKey:   astResolution.Candidates[0].CanonicalJoinKey,
		PathShape:          astResolution.Candidates[0].PathShape,
		AST: mutabilityOverlayASTState{
			UpdateCandidateState: astField.UpdateCandidateState,
			ForceNew:             astField.ForceNew,
			ConflictsWith:        append([]string(nil), astField.ConflictsWith...),
		},
		Docs: defaultMutabilityOverlayDocsEvidence(astField),
		Provenance: mutabilityOverlayProvenance{
			FormalImportPath:     resource.Formal.Binding.Manifest.ImportPath,
			FormalSourceRef:      sourceRef,
			ASTSourceBucket:      astField.ASTSourceBucket,
			TerraformDocsPage:    mutabilityOverlayDocsPageRef(target, docsAnchor),
			TerraformDocsSection: docsSection,
		},
	}

	joinStatus := mutabilityOverlayJoinMatched
	var notes []string
	if astField.UpdateCandidateState == mutabilityOverlayASTStateUpdateCandidate && !astField.ForceNew && parsedDocs != nil {
		comparison := resolver.Compare(mutabilityOverlayFieldComparisonInput{
			ASTFieldPath: astField.FieldPath,
			ASTState:     astField.UpdateCandidateState,
			ForceNew:     astField.ForceNew,
			DocsEvidence: parsedDocs.EvidenceInputs(),
		})
		field.TerraformFieldPath, field.Docs = collapseMutabilityOverlayDocsEvidence(astField.FieldPath, comparison)
		joinStatus = comparison.JoinStatus
		notes = append(notes, comparison.Diagnostics...)
	}
	field.Merge = resolveMutabilityOverlayDecision(field.AST.UpdateCandidateState, field.AST.ForceNew, field.Docs.EvidenceState, joinStatus)
	field.Provenance.Notes = uniqueSortedStrings(notes)
	return field, nil
}

func defaultMutabilityOverlayDocsEvidence(astField mutabilityOverlayASTFieldInput) mutabilityOverlayDocsEvidence {
	if astField.ForceNew {
		return mutabilityOverlayDocsEvidence{
			EvidenceState: mutabilityOverlayDocsStateNotDocumented,
			Detail:        "AST forceNew remains authoritative; docs overlay was not applied to this field.",
		}
	}
	return mutabilityOverlayDocsEvidence{
		EvidenceState: mutabilityOverlayDocsStateNotDocumented,
		Detail:        "No explicit Terraform docs update signal was found for this AST updateCandidate field.",
	}
}

func collapseMutabilityOverlayDocsEvidence(
	astFieldPath string,
	comparison mutabilityOverlayFieldComparison,
) (string, mutabilityOverlayDocsEvidence) {
	if comparison.JoinStatus == mutabilityOverlayJoinAmbiguous {
		paths := make([]string, 0, len(comparison.CandidateDocs))
		for _, candidate := range comparison.CandidateDocs {
			if trimmed := strings.TrimSpace(candidate.FieldPath); trimmed != "" {
				paths = append(paths, trimmed)
			}
		}
		paths = uniqueSortedStrings(paths)
		detail := "multiple Terraform docs field paths matched this AST field"
		rawSignal := strings.Join(paths, ", ")
		if rawSignal != "" {
			detail = detail + ": " + rawSignal
		}
		return astFieldPath, mutabilityOverlayDocsEvidence{
			EvidenceState: mutabilityOverlayDocsStateAmbiguous,
			Detail:        detail,
			RawSignal:     rawSignal,
		}
	}

	if len(comparison.CandidateDocs) == 0 {
		return astFieldPath, mutabilityOverlayDocsEvidence{
			EvidenceState: mutabilityOverlayDocsStateNotDocumented,
			Detail:        "No explicit Terraform docs update signal was found for this AST updateCandidate field.",
		}
	}

	candidates := append([]mutabilityOverlayDocsEvidenceComparison(nil), comparison.CandidateDocs...)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].FieldPath != candidates[j].FieldPath {
			return candidates[i].FieldPath < candidates[j].FieldPath
		}
		return candidates[i].EvidenceState < candidates[j].EvidenceState
	})
	selected := candidates[0]
	terraformFieldPath := astFieldPath
	if strings.TrimSpace(selected.FieldPath) != "" {
		terraformFieldPath = selected.FieldPath
	}
	return terraformFieldPath, mutabilityOverlayDocsEvidence{
		EvidenceState: selected.EvidenceState,
		Detail:        selected.Detail,
		RawSignal:     selected.RawSignal,
	}
}

func newMutabilityOverlayGeneratedArtifact(
	service string,
	resource ResourceModel,
	doc mutabilityOverlayDocument,
) mutabilityOverlayGeneratedArtifact {
	slug := fileStem(resource.Kind)
	if resource.Formal != nil && strings.TrimSpace(resource.Formal.Reference.Slug) != "" {
		slug = strings.TrimSpace(resource.Formal.Reference.Slug)
	}
	return mutabilityOverlayGeneratedArtifact{
		Service: service,
		RelativePath: filepath.ToSlash(filepath.Join(
			mutabilityOverlayGeneratedRootRelativePath,
			service,
			slug+".json",
		)),
		Document: doc,
	}
}

func mutabilityOverlayDocsPageRef(target mutabilityOverlayRegistryPageTarget, sectionAnchor string) string {
	pageName := path.Base(strings.TrimSpace(target.RegistryPath))
	if pageName == "." || pageName == "/" || pageName == "" {
		pageName = strings.TrimSpace(target.ProviderResource)
	}
	sectionAnchor = strings.TrimSpace(sectionAnchor)
	if sectionAnchor == "" {
		return pageName
	}
	return pageName + "#" + sectionAnchor
}

package generator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMutabilityValidationReport(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	overlayDoc, vapDoc := mutabilityValidationBucketFixtureDocuments()
	writeMutabilityValidationArtifacts(t, root, overlayDoc, vapDoc)

	report, err := LoadMutabilityValidationReport(root)
	if err != nil {
		t.Fatalf("LoadMutabilityValidationReport() error = %v", err)
	}

	if report.Summary.Aggregate.Resources != 1 {
		t.Fatalf("aggregate resources = %d, want 1", report.Summary.Aggregate.Resources)
	}
	if report.Summary.Aggregate.AllowInPlaceCount != 1 {
		t.Fatalf("aggregate allowInPlaceCount = %d, want 1", report.Summary.Aggregate.AllowInPlaceCount)
	}
	if report.Summary.Aggregate.DocsDeniedCount != 1 {
		t.Fatalf("aggregate docsDeniedCount = %d, want 1", report.Summary.Aggregate.DocsDeniedCount)
	}
	if report.Summary.Aggregate.ReplacementRequiredCount != 1 {
		t.Fatalf("aggregate replacementRequiredCount = %d, want 1", report.Summary.Aggregate.ReplacementRequiredCount)
	}
	if report.Summary.Aggregate.UnknownPolicyCount != 1 {
		t.Fatalf("aggregate unknownPolicyCount = %d, want 1", report.Summary.Aggregate.UnknownPolicyCount)
	}
	if report.Summary.Aggregate.MergeConflictCount != 0 {
		t.Fatalf("aggregate mergeConflictCount = %d, want 0", report.Summary.Aggregate.MergeConflictCount)
	}

	resource := report.Summary.Resources[0]
	if resource.Service != "objectstorage" || resource.Kind != "Bucket" {
		t.Fatalf("resource identity = %+v, want objectstorage/Bucket", resource)
	}
	if len(resource.AllowInPlacePaths) != 1 || resource.AllowInPlacePaths[0] != "metadata" {
		t.Fatalf("allowInPlacePaths = %v, want [metadata]", resource.AllowInPlacePaths)
	}
	if got := resource.DocsDeniedFields; len(got) != 1 || got[0] != "name" {
		t.Fatalf("docsDeniedFields = %v, want [name]", got)
	}
	if got := resource.UnknownFields; len(got) != 1 || got[0] != "definedTags" {
		t.Fatalf("unknownFields = %v, want [definedTags]", got)
	}
}

func TestCompareMutabilityValidationSummaryDetectsAllowlistWidening(t *testing.T) {
	t.Parallel()

	baseline := mutabilityValidationSummaryFixture()
	current := mutabilityValidationSummaryFixture()
	current.Aggregate.AllowInPlaceCount++
	current.Resources[0].AllowInPlaceCount++
	current.Services[0].AllowInPlaceCount++
	current.Resources[0].AllowInPlacePaths = append(current.Resources[0].AllowInPlacePaths, "name")
	current.Resources[0].Decisions[2].FinalPolicy = mutabilityOverlayPolicyAllowInPlaceUpdate

	comparison := CompareMutabilityValidationSummary(current, baseline)
	if !comparison.HasFailures() {
		t.Fatal("CompareMutabilityValidationSummary() unexpectedly reported no failures")
	}

	found := false
	for _, regression := range comparison.Regressions {
		if strings.Contains(regression, `widened allowInPlace path "name"`) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("regressions = %v, want allowlist widening", comparison.Regressions)
	}
}

func TestClassifyMutabilityValidationErrorGroupsTypedFailures(t *testing.T) {
	t.Parallel()

	err := errors.Join(
		&mutabilityOverlayRegistryPageMappingError{
			Reason:     mutabilityOverlayRegistryPageErrorAmbiguousMatch,
			Service:    "core",
			Kind:       "Instance",
			FormalSlug: "instance",
			Detail:     "multiple provider inventory matches resolved",
		},
		&mutabilityOverlayDocsAcquisitionError{
			Reason:           mutabilityOverlayDocsErrorMissingPage,
			Service:          "objectstorage",
			Kind:             "Bucket",
			FormalSlug:       "objectstoragebucket",
			ProviderResource: "oci_objectstorage_bucket",
			RegistryURL:      "https://registry.terraform.io/providers/oracle/oci/7.22.0/docs/resources/objectstorage_bucket",
		},
		&mutabilityOverlayDocsParseError{
			Reason:           mutabilityOverlayDocsParseReasonPartialParse,
			Service:          "nosql",
			Kind:             "Table",
			FormalSlug:       "table",
			ProviderResource: "oci_nosql_table",
			RegistryURL:      "https://registry.terraform.io/providers/oracle/oci/7.22.0/docs/resources/nosql_table",
			Detail:           "Argument Reference emitted field rows and parser errors",
		},
	)

	groups := ClassifyMutabilityValidationError(err)
	if len(groups) != 3 {
		t.Fatalf("groups = %d, want 3", len(groups))
	}

	assertMutabilityValidationFailureGroup(t, groups, "mappingFailures", mutabilityOverlayRegistryPageErrorAmbiguousMatch)
	assertMutabilityValidationFailureGroup(t, groups, "docsAcquisitionFailures", mutabilityOverlayDocsErrorMissingPage)
	assertMutabilityValidationFailureGroup(t, groups, "parserFailures", mutabilityOverlayDocsParseReasonPartialParse)
}

func writeMutabilityValidationArtifacts(
	t *testing.T,
	root string,
	overlayDoc mutabilityOverlayDocument,
	vapDoc vapUpdatePolicyDocument,
) {
	t.Helper()

	overlayPath := filepath.Join(root, filepath.FromSlash(mutabilityOverlayGeneratedRootRelativePath), "objectstorage", "objectstoragebucket.json")
	content, err := renderMutabilityOverlayArtifact(overlayDoc)
	if err != nil {
		t.Fatalf("renderMutabilityOverlayArtifact() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(overlayPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(overlayPath), err)
	}
	if err := os.WriteFile(overlayPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", overlayPath, err)
	}

	vapPath := filepath.Join(root, filepath.FromSlash(vapUpdatePolicyGeneratedRootRelativePath), "objectstorage", "objectstoragebucket.json")
	vapContent, err := renderVAPUpdatePolicyArtifact(vapDoc)
	if err != nil {
		t.Fatalf("renderVAPUpdatePolicyArtifact() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(vapPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(vapPath), err)
	}
	if err := os.WriteFile(vapPath, vapContent, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", vapPath, err)
	}
}

func mutabilityValidationBucketFixtureDocuments() (mutabilityOverlayDocument, vapUpdatePolicyDocument) {
	overlayDoc := mutabilityOverlayDocument{
		SchemaVersion:   mutabilityOverlaySchemaVersion,
		Surface:         mutabilityOverlaySurface,
		ContractVersion: mutabilityOverlayContractVersion,
		Metadata: mutabilityOverlayMetadata{
			ProviderSourceRef:    mutabilityOverlayProviderSourceRef,
			ProviderRevision:     "test-provider-revision",
			TerraformDocsVersion: "7.22.0",
		},
		SourceContract: mutabilityOverlaySourceContract{
			ASTPrimaryFacts:    append([]string(nil), mutabilityOverlayASTPrimaryFacts...),
			ASTMutableAlias:    mutabilityOverlayUpdateCandidateAlias,
			DocsOverlayScope:   mutabilityOverlayDocsOverlayScope,
			DocsEvidenceSource: mutabilityOverlayDocsEvidenceSource,
		},
		Resource: mutabilityOverlayResource{
			Service:              "objectstorage",
			Kind:                 "Bucket",
			FormalSlug:           "objectstoragebucket",
			ProviderResource:     "oci_objectstorage_bucket",
			RepoAuthoredSpecPath: "controllers/objectstorage/objectstoragebucket/spec.cfg",
			FormalImportPath:     "formal/imports/objectstorage/objectstoragebucket.json",
		},
		Fields: []mutabilityOverlayField{
			mutabilityValidationOverlayField("metadata", false, mutabilityOverlayDocsStateConfirmedUpdatable, mutabilityOverlayJoinMatched),
			mutabilityValidationOverlayField("definedTags", false, mutabilityOverlayDocsStateUnknown, mutabilityOverlayJoinMatched),
			mutabilityValidationOverlayField("name", false, mutabilityOverlayDocsStateDeniedUpdatable, mutabilityOverlayJoinMatched),
			mutabilityValidationOverlayField("storageTier", true, mutabilityOverlayDocsStateNotDocumented, mutabilityOverlayJoinMatched),
		},
	}
	vapDoc := vapUpdatePolicyDocument{
		SchemaVersion:   vapUpdatePolicySchemaVersion,
		Surface:         vapUpdatePolicySurface,
		ContractVersion: vapUpdatePolicyContractVersion,
		Metadata: vapUpdatePolicyMetadata{
			SourceSurface:        mutabilityOverlaySurface,
			ProviderSourceRef:    mutabilityOverlayProviderSourceRef,
			ProviderRevision:     "test-provider-revision",
			TerraformDocsVersion: "7.22.0",
		},
		Target: vapUpdatePolicyTarget{
			Service:          "objectstorage",
			APIVersion:       "objectstorage.oracle.com/v1beta1",
			Kind:             "Bucket",
			FormalSlug:       "objectstoragebucket",
			ProviderResource: "oci_objectstorage_bucket",
			SpecPathPrefix:   vapUpdatePolicySpecPathPrefix,
		},
		Update: vapUpdatePolicyUpdate{
			AllowInPlacePaths: []string{"metadata"},
			DenyRules: []vapUpdatePolicyRule{
				{
					FieldPath:         "definedTags",
					Decision:          mutabilityOverlayPolicyUnknown,
					MergeCase:         mutabilityOverlayMergeCaseUnknown,
					DocsEvidenceState: mutabilityOverlayDocsStateUnknown,
				},
				{
					FieldPath:         "name",
					Decision:          mutabilityOverlayPolicyDenyInPlaceUpdate,
					MergeCase:         mutabilityOverlayMergeCaseDocsDeniedCandidate,
					DocsEvidenceState: mutabilityOverlayDocsStateDeniedUpdatable,
				},
				{
					FieldPath:         "storageTier",
					Decision:          mutabilityOverlayPolicyReplacementRequired,
					MergeCase:         mutabilityOverlayMergeCaseReplacementRequired,
					DocsEvidenceState: mutabilityOverlayDocsStateNotDocumented,
				},
			},
		},
	}
	return overlayDoc, vapDoc
}

func mutabilityValidationOverlayField(
	fieldPath string,
	forceNew bool,
	docsEvidenceState string,
	joinStatus string,
) mutabilityOverlayField {
	astSourceBucket := mutabilityOverlayASTSourceBucketMutable
	if forceNew {
		astSourceBucket = mutabilityOverlayASTSourceBucketForceNew
	}
	return mutabilityOverlayField{
		ASTFieldPath:       fieldPath,
		TerraformFieldPath: fieldPath,
		CanonicalJoinKey:   fieldPath,
		PathShape:          mutabilityOverlayPathShapeScalar,
		AST: mutabilityOverlayASTState{
			UpdateCandidateState: mutabilityOverlayASTStateUpdateCandidate,
			ForceNew:             forceNew,
		},
		Docs: mutabilityOverlayDocsEvidence{
			EvidenceState: docsEvidenceState,
		},
		Merge: resolveMutabilityOverlayDecision(mutabilityOverlayASTStateUpdateCandidate, forceNew, docsEvidenceState, joinStatus),
		Provenance: mutabilityOverlayProvenance{
			FormalImportPath:     "formal/imports/objectstorage/objectstoragebucket.json",
			FormalSourceRef:      mutabilityOverlayProviderSourceRef,
			ASTSourceBucket:      astSourceBucket,
			TerraformDocsPage:    "https://registry.terraform.io/providers/oracle/oci/7.22.0/docs/resources/objectstorage_bucket#argument-reference",
			TerraformDocsSection: mutabilityOverlayDocsSectionArgumentReference,
		},
	}
}

func mutabilityValidationSummaryFixture() MutabilityValidationSummary {
	overlayDoc, vapDoc := mutabilityValidationBucketFixtureDocuments()
	return normalizeMutabilityValidationSummary(MutabilityValidationSummary{
		Aggregate: MutabilityValidationAggregate{
			Services:                 1,
			Resources:                1,
			OverlayArtifacts:         1,
			VAPArtifacts:             1,
			Fields:                   4,
			AllowInPlaceCount:        1,
			DenyInPlaceCount:         1,
			ReplacementRequiredCount: 1,
			UnknownPolicyCount:       1,
			DocsDeniedCount:          1,
		},
		Services: []MutabilityValidationServiceSummary{
			{
				Service:                  "objectstorage",
				Resources:                1,
				Fields:                   4,
				AllowInPlaceCount:        1,
				DenyInPlaceCount:         1,
				ReplacementRequiredCount: 1,
				UnknownPolicyCount:       1,
				DocsDeniedCount:          1,
			},
		},
		Resources: []MutabilityValidationResourceSummary{
			buildMutabilityValidationResourceSummary(
				loadedMutabilityOverlayArtifact{
					RelativePath: "internal/generator/generated/mutability_overlay/objectstorage/objectstoragebucket.json",
					Document:     overlayDoc,
				},
				loadedVAPUpdatePolicyArtifact{
					RelativePath: "internal/generator/generated/vap_update_policy/objectstorage/objectstoragebucket.json",
					Document:     vapDoc,
				},
			),
		},
	})
}

func assertMutabilityValidationFailureGroup(
	t *testing.T,
	groups []MutabilityValidationFailureGroup,
	class string,
	reason string,
) {
	t.Helper()

	for _, group := range groups {
		if group.Class != class {
			continue
		}
		for _, candidate := range group.Reasons {
			if candidate.Reason == reason {
				return
			}
		}
		t.Fatalf("group %q reasons = %+v, want %q", class, group.Reasons, reason)
	}
	t.Fatalf("groups = %+v, want class %q", groups, class)
}

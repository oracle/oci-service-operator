package generator

import (
	"slices"
	"testing"

	"github.com/oracle/oci-service-operator/internal/formal"
)

func TestBuildVAPUpdatePolicyDocumentDeniesBucketNameStyleCandidate(t *testing.T) {
	t.Parallel()

	pkg := &PackageModel{
		Service: ServiceConfig{
			Service: "objectstorage",
		},
		GroupDNSName: "objectstorage.oracle.com",
		Version:      "v1beta1",
	}
	resource := ResourceModel{
		Kind: "Bucket",
		Formal: &FormalModel{
			Reference: FormalReferenceModel{
				Service: "objectstorage",
				Slug:    "objectstoragebucket",
			},
			Binding: formal.ControllerBinding{
				Import: formal.ImportModel{
					ProviderResource: "oci_objectstorage_bucket",
				},
			},
		},
	}
	overlay := mutabilityOverlayDocument{
		SchemaVersion:   mutabilityOverlaySchemaVersion,
		Surface:         mutabilityOverlaySurface,
		ContractVersion: mutabilityOverlayContractVersion,
		Metadata: mutabilityOverlayMetadata{
			ProviderSourceRef:    mutabilityOverlayProviderSourceRef,
			ProviderRevision:     "test-provider-revision",
			TerraformDocsVersion: defaultMutabilityOverlayTerraformDocsVersion,
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
			{
				ASTFieldPath:       "definedTags",
				TerraformFieldPath: "defined_tags",
				CanonicalJoinKey:   "definedTags",
				PathShape:          mutabilityOverlayPathShapeMapEntry,
				AST: mutabilityOverlayASTState{
					UpdateCandidateState: mutabilityOverlayASTStateUpdateCandidate,
					ForceNew:             false,
					ConflictsWith:        []string{},
				},
				Docs: mutabilityOverlayDocsEvidence{
					EvidenceState: mutabilityOverlayDocsStateConfirmedUpdatable,
					Detail:        "Tags are updatable.",
				},
				Merge: mutabilityOverlayMergeResult{
					JoinStatus:  mutabilityOverlayJoinMatched,
					MergeCase:   mutabilityOverlayMergeCaseDocsConfirmedCandidate,
					FinalPolicy: mutabilityOverlayPolicyAllowInPlaceUpdate,
				},
				Provenance: mutabilityOverlayProvenance{
					FormalImportPath:     "formal/imports/objectstorage/objectstoragebucket.json",
					FormalSourceRef:      mutabilityOverlayProviderSourceRef,
					ASTSourceBucket:      mutabilityOverlayASTSourceBucketMutable,
					TerraformDocsPage:    "objectstorage_bucket",
					TerraformDocsSection: "Argument Reference",
				},
			},
			{
				ASTFieldPath:       "name",
				TerraformFieldPath: "name",
				CanonicalJoinKey:   "name",
				PathShape:          mutabilityOverlayPathShapeScalar,
				AST: mutabilityOverlayASTState{
					UpdateCandidateState: mutabilityOverlayASTStateUpdateCandidate,
					ForceNew:             false,
					ConflictsWith:        []string{},
				},
				Docs: mutabilityOverlayDocsEvidence{
					EvidenceState: mutabilityOverlayDocsStateDeniedUpdatable,
					Detail:        "Updating this value after creation is not supported.",
				},
				Merge: mutabilityOverlayMergeResult{
					JoinStatus:  mutabilityOverlayJoinMatched,
					MergeCase:   mutabilityOverlayMergeCaseDocsDeniedCandidate,
					FinalPolicy: mutabilityOverlayPolicyDenyInPlaceUpdate,
				},
				Provenance: mutabilityOverlayProvenance{
					FormalImportPath:     "formal/imports/objectstorage/objectstoragebucket.json",
					FormalSourceRef:      mutabilityOverlayProviderSourceRef,
					ASTSourceBucket:      mutabilityOverlayASTSourceBucketMutable,
					TerraformDocsPage:    "objectstorage_bucket",
					TerraformDocsSection: "Argument Reference",
				},
			},
			{
				ASTFieldPath:       "storageTier",
				TerraformFieldPath: "storage_tier",
				CanonicalJoinKey:   "storageTier",
				PathShape:          mutabilityOverlayPathShapeScalar,
				AST: mutabilityOverlayASTState{
					UpdateCandidateState: mutabilityOverlayASTStateNotUpdateCandidate,
					ForceNew:             true,
					ConflictsWith:        []string{},
				},
				Docs: mutabilityOverlayDocsEvidence{
					EvidenceState: mutabilityOverlayDocsStateNotDocumented,
					Detail:        "No explicit Terraform docs update signal was found for this AST updateCandidate field.",
				},
				Merge: mutabilityOverlayMergeResult{
					JoinStatus:  mutabilityOverlayJoinMatched,
					MergeCase:   mutabilityOverlayMergeCaseReplacementRequired,
					FinalPolicy: mutabilityOverlayPolicyReplacementRequired,
				},
				Provenance: mutabilityOverlayProvenance{
					FormalImportPath:     "formal/imports/objectstorage/objectstoragebucket.json",
					FormalSourceRef:      mutabilityOverlayProviderSourceRef,
					ASTSourceBucket:      mutabilityOverlayASTSourceBucketForceNew,
					TerraformDocsPage:    "objectstorage_bucket",
					TerraformDocsSection: "Argument Reference",
				},
			},
		},
	}

	doc, err := buildVAPUpdatePolicyDocument(pkg, resource, overlay)
	if err != nil {
		t.Fatalf("buildVAPUpdatePolicyDocument() error = %v", err)
	}

	if !slices.Equal(doc.Update.AllowInPlacePaths, []string{"definedTags"}) {
		t.Fatalf("allowInPlacePaths = %v, want [definedTags]", doc.Update.AllowInPlacePaths)
	}
	nameRule := findVAPUpdatePolicyRule(t, doc.Update.DenyRules, "name")
	if nameRule.Decision != mutabilityOverlayPolicyDenyInPlaceUpdate {
		t.Fatalf("name decision = %q, want %q", nameRule.Decision, mutabilityOverlayPolicyDenyInPlaceUpdate)
	}
	if nameRule.MergeCase != mutabilityOverlayMergeCaseDocsDeniedCandidate {
		t.Fatalf("name mergeCase = %q, want %q", nameRule.MergeCase, mutabilityOverlayMergeCaseDocsDeniedCandidate)
	}
	if nameRule.DocsEvidenceState != mutabilityOverlayDocsStateDeniedUpdatable {
		t.Fatalf("name docsEvidenceState = %q, want %q", nameRule.DocsEvidenceState, mutabilityOverlayDocsStateDeniedUpdatable)
	}

	storageTierRule := findVAPUpdatePolicyRule(t, doc.Update.DenyRules, "storageTier")
	if storageTierRule.Decision != mutabilityOverlayPolicyReplacementRequired {
		t.Fatalf("storageTier decision = %q, want %q", storageTierRule.Decision, mutabilityOverlayPolicyReplacementRequired)
	}
}

func findVAPUpdatePolicyRule(t *testing.T, rules []vapUpdatePolicyRule, fieldPath string) vapUpdatePolicyRule {
	t.Helper()
	for _, rule := range rules {
		if rule.FieldPath == fieldPath {
			return rule
		}
	}
	t.Fatalf("vap update policy rule %q not found", fieldPath)
	return vapUpdatePolicyRule{}
}

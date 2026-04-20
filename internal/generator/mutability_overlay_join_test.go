package generator

import (
	"strings"
	"testing"
)

func TestMutabilityOverlayPathResolverResolve(t *testing.T) {
	t.Parallel()

	resolver := newMutabilityOverlayPathResolver(testMutabilityOverlayJoinResource())
	tests := []struct {
		name          string
		fieldPath     string
		wantStatus    string
		wantJoinKey   string
		wantPathShape string
		wantDiag      string
	}{
		{
			name:          "scalar field uses generated json name",
			fieldPath:     "ddl_statement",
			wantStatus:    mutabilityOverlayJoinMatched,
			wantJoinKey:   "ddlStatement",
			wantPathShape: mutabilityOverlayPathShapeScalar,
		},
		{
			name:          "nested object field uses generated json name",
			fieldPath:     "table_limits.max_storage_in_gbs",
			wantStatus:    mutabilityOverlayJoinMatched,
			wantJoinKey:   "tableLimits.maxStorageInGBs",
			wantPathShape: mutabilityOverlayPathShapeObject,
		},
		{
			name:          "singular docs block resolves to list item",
			fieldPath:     "table_limits.read_unit.capacity_mode",
			wantStatus:    mutabilityOverlayJoinMatched,
			wantJoinKey:   "tableLimits.readUnits[].capacityMode",
			wantPathShape: mutabilityOverlayPathShapeListItem,
		},
		{
			name:          "map entry preserves wildcard path",
			fieldPath:     "defined_tags.*",
			wantStatus:    mutabilityOverlayJoinMatched,
			wantJoinKey:   "definedTags.*",
			wantPathShape: mutabilityOverlayPathShapeMapEntry,
		},
		{
			name:       "missing field stays unresolved",
			fieldPath:  "ghost_block.value",
			wantStatus: mutabilityOverlayJoinUnresolved,
			wantDiag:   "did not match any generated field",
		},
		{
			name:          "exact field outranks singular list alias",
			fieldPath:     "policy.name",
			wantStatus:    mutabilityOverlayJoinMatched,
			wantJoinKey:   "policy.name",
			wantPathShape: mutabilityOverlayPathShapeObject,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := resolver.Resolve(test.fieldPath)
			if got.Status != test.wantStatus {
				t.Fatalf("Resolve(%q).Status = %q, want %q", test.fieldPath, got.Status, test.wantStatus)
			}
			if test.wantJoinKey != "" {
				if len(got.Candidates) != 1 {
					t.Fatalf("Resolve(%q).Candidates length = %d, want 1", test.fieldPath, len(got.Candidates))
				}
				if got.Candidates[0].CanonicalJoinKey != test.wantJoinKey {
					t.Fatalf("Resolve(%q).CanonicalJoinKey = %q, want %q", test.fieldPath, got.Candidates[0].CanonicalJoinKey, test.wantJoinKey)
				}
				if got.Candidates[0].PathShape != test.wantPathShape {
					t.Fatalf("Resolve(%q).PathShape = %q, want %q", test.fieldPath, got.Candidates[0].PathShape, test.wantPathShape)
				}
			}
			if test.wantDiag != "" && !containsDiagnostic(got.Diagnostics, test.wantDiag) {
				t.Fatalf("Resolve(%q).Diagnostics = %v, want substring %q", test.fieldPath, got.Diagnostics, test.wantDiag)
			}
		})
	}
}

func TestMutabilityOverlayPathResolverCompare(t *testing.T) {
	t.Parallel()

	resolver := newMutabilityOverlayPathResolver(testMutabilityOverlayJoinResource())
	tests := []struct {
		name                  string
		input                 mutabilityOverlayFieldComparisonInput
		wantJoinStatus        string
		wantTerraformField    string
		wantTerraformJoinKey  string
		wantTerraformShape    string
		wantMerge             mutabilityOverlayMergeResult
		wantCandidateDocCount int
		wantDiag              string
	}{
		{
			name: "ast only candidate",
			input: mutabilityOverlayFieldComparisonInput{
				ASTFieldPath: "freeform_tags",
				ASTState:     mutabilityOverlayASTStateUpdateCandidate,
			},
			wantJoinStatus:       mutabilityOverlayJoinMatched,
			wantTerraformField:   "freeform_tags",
			wantTerraformJoinKey: "freeformTags",
			wantTerraformShape:   mutabilityOverlayPathShapeScalar,
			wantMerge: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinMatched,
				MergeCase:   mutabilityOverlayMergeCaseASTOnlyCandidate,
				FinalPolicy: mutabilityOverlayPolicyUnknown,
			},
		},
		{
			name: "docs confirmed candidate",
			input: mutabilityOverlayFieldComparisonInput{
				ASTFieldPath: "ddl_statement",
				ASTState:     mutabilityOverlayASTStateUpdateCandidate,
				DocsEvidence: []mutabilityOverlayDocsEvidenceInput{
					{
						FieldPath:     "ddl_statement",
						EvidenceState: mutabilityOverlayDocsStateConfirmedUpdatable,
					},
				},
			},
			wantJoinStatus:        mutabilityOverlayJoinMatched,
			wantTerraformField:    "ddl_statement",
			wantTerraformJoinKey:  "ddlStatement",
			wantTerraformShape:    mutabilityOverlayPathShapeScalar,
			wantCandidateDocCount: 1,
			wantMerge: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinMatched,
				MergeCase:   mutabilityOverlayMergeCaseDocsConfirmedCandidate,
				FinalPolicy: mutabilityOverlayPolicyAllowInPlaceUpdate,
			},
		},
		{
			name: "same family map entry stays unresolved",
			input: mutabilityOverlayFieldComparisonInput{
				ASTFieldPath: "defined_tags",
				ASTState:     mutabilityOverlayASTStateUpdateCandidate,
				DocsEvidence: []mutabilityOverlayDocsEvidenceInput{
					{
						FieldPath:     "defined_tags.*",
						EvidenceState: mutabilityOverlayDocsStateConfirmedUpdatable,
					},
				},
			},
			wantJoinStatus:        mutabilityOverlayJoinUnresolved,
			wantTerraformField:    "defined_tags.*",
			wantTerraformJoinKey:  "definedTags.*",
			wantTerraformShape:    mutabilityOverlayPathShapeMapEntry,
			wantCandidateDocCount: 1,
			wantDiag:              "had no exact docs match",
			wantMerge: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinUnresolved,
				MergeCase:   mutabilityOverlayMergeCaseUnresolvedJoin,
				FinalPolicy: mutabilityOverlayPolicyUnknown,
			},
		},
		{
			name: "multiple docs aliases stay ambiguous",
			input: mutabilityOverlayFieldComparisonInput{
				ASTFieldPath: "table_limits.read_units.capacity_mode",
				ASTState:     mutabilityOverlayASTStateUpdateCandidate,
				DocsEvidence: []mutabilityOverlayDocsEvidenceInput{
					{
						FieldPath:     "table_limits.read_units.capacity_mode",
						EvidenceState: mutabilityOverlayDocsStateConfirmedUpdatable,
					},
					{
						FieldPath:     "table_limits.read_unit.capacity_mode",
						EvidenceState: mutabilityOverlayDocsStateConfirmedUpdatable,
					},
				},
			},
			wantJoinStatus:        mutabilityOverlayJoinAmbiguous,
			wantCandidateDocCount: 2,
			wantDiag:              "matched multiple docs paths",
			wantMerge: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinAmbiguous,
				MergeCase:   mutabilityOverlayMergeCaseUnresolvedJoin,
				FinalPolicy: mutabilityOverlayPolicyUnknown,
			},
		},
		{
			name: "force new remains replacement required",
			input: mutabilityOverlayFieldComparisonInput{
				ASTFieldPath: "name",
				ASTState:     mutabilityOverlayASTStateNotUpdateCandidate,
				ForceNew:     true,
			},
			wantJoinStatus:       mutabilityOverlayJoinMatched,
			wantTerraformField:   "name",
			wantTerraformJoinKey: "name",
			wantTerraformShape:   mutabilityOverlayPathShapeScalar,
			wantMerge: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinMatched,
				MergeCase:   mutabilityOverlayMergeCaseReplacementRequired,
				FinalPolicy: mutabilityOverlayPolicyReplacementRequired,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := resolver.Compare(test.input)
			if got.JoinStatus != test.wantJoinStatus {
				t.Fatalf("Compare(%q).JoinStatus = %q, want %q", test.input.ASTFieldPath, got.JoinStatus, test.wantJoinStatus)
			}
			if got.TerraformFieldPath != test.wantTerraformField {
				t.Fatalf("Compare(%q).TerraformFieldPath = %q, want %q", test.input.ASTFieldPath, got.TerraformFieldPath, test.wantTerraformField)
			}
			if test.wantTerraformJoinKey != "" {
				if len(got.TerraformResolution.Candidates) != 1 {
					t.Fatalf("Compare(%q).TerraformResolution.Candidates length = %d, want 1", test.input.ASTFieldPath, len(got.TerraformResolution.Candidates))
				}
				if got.TerraformResolution.Candidates[0].CanonicalJoinKey != test.wantTerraformJoinKey {
					t.Fatalf("Compare(%q).TerraformResolution.CanonicalJoinKey = %q, want %q", test.input.ASTFieldPath, got.TerraformResolution.Candidates[0].CanonicalJoinKey, test.wantTerraformJoinKey)
				}
				if got.TerraformResolution.Candidates[0].PathShape != test.wantTerraformShape {
					t.Fatalf("Compare(%q).TerraformResolution.PathShape = %q, want %q", test.input.ASTFieldPath, got.TerraformResolution.Candidates[0].PathShape, test.wantTerraformShape)
				}
			}
			if len(got.CandidateDocs) != test.wantCandidateDocCount {
				t.Fatalf("Compare(%q).CandidateDocs length = %d, want %d", test.input.ASTFieldPath, len(got.CandidateDocs), test.wantCandidateDocCount)
			}
			if got.Merge != test.wantMerge {
				t.Fatalf("Compare(%q).Merge = %+v, want %+v", test.input.ASTFieldPath, got.Merge, test.wantMerge)
			}
			if test.wantDiag != "" && !containsDiagnostic(got.Diagnostics, test.wantDiag) {
				t.Fatalf("Compare(%q).Diagnostics = %v, want substring %q", test.input.ASTFieldPath, got.Diagnostics, test.wantDiag)
			}
		})
	}
}

func testMutabilityOverlayJoinResource() ResourceModel {
	return ResourceModel{
		Kind: "Table",
		SpecFields: []FieldModel{
			{Name: "Name", Type: "string", Tag: jsonTag("name", true)},
			{Name: "DdlStatement", Type: "string", Tag: jsonTag("ddlStatement", false)},
			{Name: "FreeformTags", Type: "map[string]string", Tag: jsonTag("freeformTags", true)},
			{Name: "DefinedTags", Type: "map[string]shared.MapValue", Tag: jsonTag("definedTags", true)},
			{Name: "TableLimits", Type: "TableLimits", Tag: jsonTag("tableLimits", true)},
			{Name: "Policy", Type: "TablePolicy", Tag: jsonTag("policy", true)},
			{Name: "Policies", Type: "[]TablePolicy", Tag: jsonTag("policies", true)},
		},
		HelperTypes: []TypeModel{
			{
				Name: "TableLimits",
				Fields: []FieldModel{
					{Name: "MaxReadUnits", Type: "int", Tag: jsonTag("maxReadUnits", false)},
					{Name: "MaxStorageInGBs", Type: "int", Tag: jsonTag("maxStorageInGBs", false)},
					{Name: "ReadUnits", Type: "[]TableReadUnit", Tag: jsonTag("readUnits", true)},
				},
			},
			{
				Name: "TableReadUnit",
				Fields: []FieldModel{
					{Name: "CapacityMode", Type: "string", Tag: jsonTag("capacityMode", true)},
				},
			},
			{
				Name: "TablePolicy",
				Fields: []FieldModel{
					{Name: "Name", Type: "string", Tag: jsonTag("name", true)},
				},
			},
		},
	}
}

func containsDiagnostic(diagnostics []string, needle string) bool {
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic, needle) {
			return true
		}
	}
	return false
}

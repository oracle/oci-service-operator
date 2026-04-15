package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMutabilityOverlayDecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		astCandidateState string
		forceNew          bool
		docsEvidenceState string
		joinStatus        string
		want              mutabilityOverlayMergeResult
	}{
		{
			name:              "ast only candidate",
			astCandidateState: mutabilityOverlayASTStateUpdateCandidate,
			docsEvidenceState: mutabilityOverlayDocsStateNotDocumented,
			joinStatus:        mutabilityOverlayJoinMatched,
			want: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinMatched,
				MergeCase:   mutabilityOverlayMergeCaseASTOnlyCandidate,
				FinalPolicy: mutabilityOverlayPolicyUnknown,
			},
		},
		{
			name:              "docs confirmed candidate",
			astCandidateState: mutabilityOverlayASTStateUpdateCandidate,
			docsEvidenceState: mutabilityOverlayDocsStateConfirmedUpdatable,
			joinStatus:        mutabilityOverlayJoinMatched,
			want: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinMatched,
				MergeCase:   mutabilityOverlayMergeCaseDocsConfirmedCandidate,
				FinalPolicy: mutabilityOverlayPolicyAllowInPlaceUpdate,
			},
		},
		{
			name:              "docs denied candidate",
			astCandidateState: mutabilityOverlayASTStateUpdateCandidate,
			docsEvidenceState: mutabilityOverlayDocsStateDeniedUpdatable,
			joinStatus:        mutabilityOverlayJoinMatched,
			want: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinMatched,
				MergeCase:   mutabilityOverlayMergeCaseDocsDeniedCandidate,
				FinalPolicy: mutabilityOverlayPolicyDenyInPlaceUpdate,
			},
		},
		{
			name:              "replacement required",
			astCandidateState: mutabilityOverlayASTStateNotUpdateCandidate,
			forceNew:          true,
			docsEvidenceState: mutabilityOverlayDocsStateConfirmedUpdatable,
			joinStatus:        mutabilityOverlayJoinMatched,
			want: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinMatched,
				MergeCase:   mutabilityOverlayMergeCaseReplacementRequired,
				FinalPolicy: mutabilityOverlayPolicyReplacementRequired,
			},
		},
		{
			name:              "unknown",
			astCandidateState: mutabilityOverlayASTStateUpdateCandidate,
			docsEvidenceState: mutabilityOverlayDocsStateAmbiguous,
			joinStatus:        mutabilityOverlayJoinMatched,
			want: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinMatched,
				MergeCase:   mutabilityOverlayMergeCaseUnknown,
				FinalPolicy: mutabilityOverlayPolicyUnknown,
			},
		},
		{
			name:              "unresolved join",
			astCandidateState: mutabilityOverlayASTStateUpdateCandidate,
			docsEvidenceState: mutabilityOverlayDocsStateConfirmedUpdatable,
			joinStatus:        mutabilityOverlayJoinUnresolved,
			want: mutabilityOverlayMergeResult{
				JoinStatus:  mutabilityOverlayJoinUnresolved,
				MergeCase:   mutabilityOverlayMergeCaseUnresolvedJoin,
				FinalPolicy: mutabilityOverlayPolicyUnknown,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := resolveMutabilityOverlayDecision(test.astCandidateState, test.forceNew, test.docsEvidenceState, test.joinStatus)
			if got != test.want {
				t.Fatalf("resolveMutabilityOverlayDecision() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestMutabilityOverlayExampleDocumentValidates(t *testing.T) {
	t.Parallel()

	path := filepath.Join(generatorTestDir(t), "testdata", "mutability_overlay", "example.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	var doc mutabilityOverlayDocument
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", path, err)
	}
	if err := validateMutabilityOverlayDocument(doc); err != nil {
		t.Fatalf("validateMutabilityOverlayDocument() error = %v", err)
	}

	want := canonicalJSON(t, exampleMutabilityOverlayDocument())
	got := canonicalJSONFromBytes(t, content)
	if got != want {
		t.Fatalf("example.json mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestMutabilityOverlaySchemaMatchesContractConstants(t *testing.T) {
	t.Parallel()

	path := filepath.Join(generatorTestDir(t), "testdata", "mutability_overlay", "schema.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	want := canonicalJSON(t, canonicalMutabilityOverlaySchema())
	got := canonicalJSONFromBytes(t, content)
	if got != want {
		t.Fatalf("schema.json mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func canonicalJSON(t *testing.T, value any) string {
	t.Helper()

	content, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return canonicalJSONFromBytes(t, content)
}

func canonicalJSONFromBytes(t *testing.T, content []byte) string {
	t.Helper()

	var decoded any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	normalized, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	return string(append(normalized, '\n'))
}

package generator

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestParseMutabilityOverlayDocsArgumentReferenceParsesCheckedInFixtures(t *testing.T) {
	t.Parallel()

	root := filepath.Join(generatorTestDir(t), "testdata", "mutability_overlay", "docs")
	tests := []struct {
		name          string
		target        mutabilityOverlayRegistryPageTarget
		wantStates    map[string]string
		wantRawSignal map[string]string
	}{
		{
			name: "minimal table fixture",
			target: newMutabilityOverlayDocsTargetForTest(
				"nosql",
				"Table",
				"table",
				"oci_nosql_table",
				"contract-example-v1",
				"https://registry.terraform.io/providers/oracle/oci/contract-example-v1/docs/resources/nosql_table",
			),
			wantStates: map[string]string{
				"ddl_statement": mutabilityOverlayDocsStateConfirmedUpdatable,
			},
			wantRawSignal: map[string]string{
				"ddl_statement": "(updatable)",
			},
		},
		{
			name: "nested database fixture",
			target: newMutabilityOverlayDocsTargetForTest(
				"database",
				"Database",
				"database",
				"oci_database_database",
				"contract-example-v1",
				"https://registry.terraform.io/providers/oracle/oci/contract-example-v1/docs/resources/database_database",
			),
			wantStates: map[string]string{
				"display_name":                    mutabilityOverlayDocsStateConfirmedUpdatable,
				"db_home":                         mutabilityOverlayDocsStateUnknown,
				"db_home.database":                mutabilityOverlayDocsStateUnknown,
				"db_home.database.admin_password": mutabilityOverlayDocsStateDeniedUpdatable,
				"db_home.database.db_workload":    mutabilityOverlayDocsStateReplacementRequired,
				"db_home.database.license_model":  mutabilityOverlayDocsStateAmbiguous,
				"defined_tags":                    mutabilityOverlayDocsStateUnknown,
			},
			wantRawSignal: map[string]string{
				"display_name":                    "(updatable)",
				"db_home.database.admin_password": "not supported",
				"db_home.database.db_workload":    "forces a new resource",
				"db_home.database.license_model":  "require replacement",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			input := loadMutabilityOverlayDocsFixtureForParserTest(t, root, test.target)
			got, err := parseMutabilityOverlayDocsArgumentReference(input)
			if err != nil {
				t.Fatalf("parseMutabilityOverlayDocsArgumentReference() error = %v", err)
			}
			if got.SectionTitle != mutabilityOverlayDocsSectionArgumentReference {
				t.Fatalf("SectionTitle = %q, want %q", got.SectionTitle, mutabilityOverlayDocsSectionArgumentReference)
			}
			if got.SectionAnchor != "argument-reference" {
				t.Fatalf("SectionAnchor = %q, want %q", got.SectionAnchor, "argument-reference")
			}
			if len(got.Diagnostics) != 0 {
				t.Fatalf("Diagnostics = %+v, want none", got.Diagnostics)
			}

			indexed := indexMutabilityOverlayParsedDocsEvidence(got.Fields)
			if len(indexed) != len(test.wantStates) {
				t.Fatalf("parsed field count = %d, want %d", len(indexed), len(test.wantStates))
			}
			if len(got.EvidenceInputs()) != len(test.wantStates) {
				t.Fatalf("EvidenceInputs() count = %d, want %d", len(got.EvidenceInputs()), len(test.wantStates))
			}
			for fieldPath, wantState := range test.wantStates {
				evidence, ok := indexed[fieldPath]
				if !ok {
					t.Fatalf("parsed fields missing %q; got keys %v", fieldPath, sortedMutabilityOverlayEvidenceKeys(indexed))
				}
				if evidence.EvidenceState != wantState {
					t.Fatalf("field %q EvidenceState = %q, want %q", fieldPath, evidence.EvidenceState, wantState)
				}
				if evidence.Provenance.RegistryURL != test.target.RegistryURL {
					t.Fatalf("field %q RegistryURL = %q, want %q", fieldPath, evidence.Provenance.RegistryURL, test.target.RegistryURL)
				}
				if evidence.Provenance.SectionAnchor != "argument-reference" {
					t.Fatalf("field %q SectionAnchor = %q, want %q", fieldPath, evidence.Provenance.SectionAnchor, "argument-reference")
				}
				if evidence.Provenance.SectionTitle != mutabilityOverlayDocsSectionArgumentReference {
					t.Fatalf("field %q SectionTitle = %q, want %q", fieldPath, evidence.Provenance.SectionTitle, mutabilityOverlayDocsSectionArgumentReference)
				}
				if strings.TrimSpace(evidence.Provenance.EvidenceText) == "" {
					t.Fatalf("field %q EvidenceText is empty", fieldPath)
				}
			}
			for fieldPath, wantSubstring := range test.wantRawSignal {
				evidence := indexed[fieldPath]
				if !strings.Contains(strings.ToLower(evidence.RawSignal), strings.ToLower(wantSubstring)) {
					t.Fatalf("field %q RawSignal = %q, want substring %q", fieldPath, evidence.RawSignal, wantSubstring)
				}
			}
		})
	}
}

func TestParseMutabilityOverlayDocsArgumentReferenceReportsUnsupportedLayout(t *testing.T) {
	t.Parallel()

	root := filepath.Join(generatorTestDir(t), "testdata", "mutability_overlay", "docs")
	target := newMutabilityOverlayDocsTargetForTest(
		"core",
		"Instance",
		"instance",
		"oci_core_instance",
		"contract-example-v1",
		"https://registry.terraform.io/providers/oracle/oci/contract-example-v1/docs/resources/core_instance",
	)
	input := loadMutabilityOverlayDocsFixtureForParserTest(t, root, target)

	got, err := parseMutabilityOverlayDocsArgumentReference(input)
	if err == nil {
		t.Fatal("parseMutabilityOverlayDocsArgumentReference() unexpectedly succeeded")
	}

	var parseErr *mutabilityOverlayDocsParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %v, want mutabilityOverlayDocsParseError", err)
	}
	if parseErr.Reason != mutabilityOverlayDocsParseReasonPartialParse {
		t.Fatalf("parse error reason = %q, want %q", parseErr.Reason, mutabilityOverlayDocsParseReasonPartialParse)
	}
	if len(got.Fields) != 0 {
		t.Fatalf("Fields length = %d, want 0", len(got.Fields))
	}
	if !hasMutabilityOverlayDiagnostic(got.Diagnostics, mutabilityOverlayDocsParseDiagnosticUnsupportedSectionNode, "<table>") {
		t.Fatalf("Diagnostics = %+v, want unsupported table diagnostic", got.Diagnostics)
	}
}

func TestParseMutabilityOverlayDocsArgumentReferenceRejectsMissingArgumentReference(t *testing.T) {
	t.Parallel()

	input := mutabilityOverlayDocsInput{
		Metadata: mutabilityOverlayDocsInputMetadata{
			Service:          "core",
			Kind:             "Vcn",
			ProviderResource: "oci_core_vcn",
			RegistryURL:      "https://registry.terraform.io/providers/oracle/oci/contract-example-v1/docs/resources/core_vcn",
		},
		Body: "<!DOCTYPE html><html><body><main><h2>Overview</h2></main></body></html>",
	}

	_, err := parseMutabilityOverlayDocsArgumentReference(input)
	if err == nil {
		t.Fatal("parseMutabilityOverlayDocsArgumentReference() unexpectedly succeeded")
	}

	var parseErr *mutabilityOverlayDocsParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error = %v, want mutabilityOverlayDocsParseError", err)
	}
	if parseErr.Reason != mutabilityOverlayDocsParseReasonMissingArgumentReference {
		t.Fatalf("parse error reason = %q, want %q", parseErr.Reason, mutabilityOverlayDocsParseReasonMissingArgumentReference)
	}
}

func loadMutabilityOverlayDocsFixtureForParserTest(
	t *testing.T,
	root string,
	target mutabilityOverlayRegistryPageTarget,
) mutabilityOverlayDocsInput {
	t.Helper()

	inputs, err := loadMutabilityOverlayDocsFixtures(root, []mutabilityOverlayRegistryPageTarget{target})
	if err != nil {
		t.Fatalf("loadMutabilityOverlayDocsFixtures() error = %v", err)
	}
	if len(inputs) != 1 {
		t.Fatalf("loadMutabilityOverlayDocsFixtures() returned %d inputs, want 1", len(inputs))
	}
	return inputs[0]
}

func indexMutabilityOverlayParsedDocsEvidence(fields []mutabilityOverlayParsedDocsEvidence) map[string]mutabilityOverlayParsedDocsEvidence {
	indexed := make(map[string]mutabilityOverlayParsedDocsEvidence, len(fields))
	for _, field := range fields {
		indexed[field.FieldPath] = field
	}
	return indexed
}

func sortedMutabilityOverlayEvidenceKeys(indexed map[string]mutabilityOverlayParsedDocsEvidence) []string {
	keys := make([]string, 0, len(indexed))
	for key := range indexed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func hasMutabilityOverlayDiagnostic(
	diagnostics []mutabilityOverlayDocsParserDiagnostic,
	reason string,
	detailSubstring string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Reason != reason {
			continue
		}
		if detailSubstring == "" || strings.Contains(diagnostic.Detail, detailSubstring) {
			return true
		}
	}
	return false
}

package generator

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	mutabilityOverlaySchemaVersion     = 1
	mutabilityOverlaySurface           = "generator-mutability-overlay"
	mutabilityOverlayContractVersion   = "v1alpha1"
	mutabilityOverlayProviderSourceRef = "terraform-provider-oci"

	mutabilityOverlayUpdateCandidateAlias = "updateCandidate"
	mutabilityOverlayDocsOverlayScope     = "terraform-argument-reference-for-ast-update-candidate-fields"
	mutabilityOverlayDocsEvidenceSource   = "terraform-argument-reference"

	mutabilityOverlayPathShapeScalar      = "scalar"
	mutabilityOverlayPathShapeObject      = "object"
	mutabilityOverlayPathShapeListItem    = "listItem"
	mutabilityOverlayPathShapeMapEntry    = "mapEntry"
	mutabilityOverlayPathShapeUnsupported = "unsupported"

	mutabilityOverlayASTStateUpdateCandidate    = "updateCandidate"
	mutabilityOverlayASTStateNotUpdateCandidate = "notUpdateCandidate"

	mutabilityOverlayASTSourceBucketMutable  = "mutation.mutable"
	mutabilityOverlayASTSourceBucketForceNew = "mutation.forceNew"

	mutabilityOverlayDocsStateConfirmedUpdatable  = "confirmedUpdatable"
	mutabilityOverlayDocsStateDeniedUpdatable     = "deniedUpdatable"
	mutabilityOverlayDocsStateReplacementRequired = "replacementRequired"
	mutabilityOverlayDocsStateUnknown             = "unknown"
	mutabilityOverlayDocsStateNotDocumented       = "notDocumented"
	mutabilityOverlayDocsStateAmbiguous           = "ambiguous"
	mutabilityOverlayDocsStateUnsupported         = "unsupported"

	mutabilityOverlayJoinMatched    = "matched"
	mutabilityOverlayJoinUnresolved = "unresolved"
	mutabilityOverlayJoinAmbiguous  = "ambiguous"

	mutabilityOverlayMergeCaseASTOnlyCandidate       = "astOnlyCandidate"
	mutabilityOverlayMergeCaseDocsConfirmedCandidate = "docsConfirmedCandidate"
	mutabilityOverlayMergeCaseDocsDeniedCandidate    = "docsDeniedCandidate"
	mutabilityOverlayMergeCaseReplacementRequired    = "replacementRequired"
	mutabilityOverlayMergeCaseUnknown                = "unknown"
	mutabilityOverlayMergeCaseUnresolvedJoin         = "unresolvedJoin"

	mutabilityOverlayPolicyAllowInPlaceUpdate  = "allowInPlaceUpdate"
	mutabilityOverlayPolicyDenyInPlaceUpdate   = "denyInPlaceUpdate"
	mutabilityOverlayPolicyReplacementRequired = "replacementRequired"
	mutabilityOverlayPolicyUnknown             = "unknown"
)

var (
	mutabilityOverlayASTPrimaryFacts = []string{
		"forceNew",
		"conflictsWith",
		"lifecycle",
		"hooks",
		"crud",
		"listLookup",
	}
	mutabilityOverlayPathShapes = []string{
		mutabilityOverlayPathShapeScalar,
		mutabilityOverlayPathShapeObject,
		mutabilityOverlayPathShapeListItem,
		mutabilityOverlayPathShapeMapEntry,
		mutabilityOverlayPathShapeUnsupported,
	}
	mutabilityOverlayASTCandidateStates = []string{
		mutabilityOverlayASTStateUpdateCandidate,
		mutabilityOverlayASTStateNotUpdateCandidate,
	}
	mutabilityOverlayASTSourceBuckets = []string{
		mutabilityOverlayASTSourceBucketMutable,
		mutabilityOverlayASTSourceBucketForceNew,
	}
	mutabilityOverlayDocsEvidenceStates = []string{
		mutabilityOverlayDocsStateConfirmedUpdatable,
		mutabilityOverlayDocsStateDeniedUpdatable,
		mutabilityOverlayDocsStateReplacementRequired,
		mutabilityOverlayDocsStateUnknown,
		mutabilityOverlayDocsStateNotDocumented,
		mutabilityOverlayDocsStateAmbiguous,
		mutabilityOverlayDocsStateUnsupported,
	}
	mutabilityOverlayJoinStatuses = []string{
		mutabilityOverlayJoinMatched,
		mutabilityOverlayJoinUnresolved,
		mutabilityOverlayJoinAmbiguous,
	}
	mutabilityOverlayMergeCases = []string{
		mutabilityOverlayMergeCaseASTOnlyCandidate,
		mutabilityOverlayMergeCaseDocsConfirmedCandidate,
		mutabilityOverlayMergeCaseDocsDeniedCandidate,
		mutabilityOverlayMergeCaseReplacementRequired,
		mutabilityOverlayMergeCaseUnknown,
		mutabilityOverlayMergeCaseUnresolvedJoin,
	}
	mutabilityOverlayFinalPolicies = []string{
		mutabilityOverlayPolicyAllowInPlaceUpdate,
		mutabilityOverlayPolicyDenyInPlaceUpdate,
		mutabilityOverlayPolicyReplacementRequired,
		mutabilityOverlayPolicyUnknown,
	}
)

// mutabilityOverlayDocument captures the generator-owned AST-primary mutability contract.
type mutabilityOverlayDocument struct {
	SchemaVersion   int                             `json:"schemaVersion"`
	Surface         string                          `json:"surface"`
	ContractVersion string                          `json:"contractVersion"`
	Metadata        mutabilityOverlayMetadata       `json:"metadata"`
	SourceContract  mutabilityOverlaySourceContract `json:"sourceContract"`
	Resource        mutabilityOverlayResource       `json:"resource"`
	Fields          []mutabilityOverlayField        `json:"fields"`
}

type mutabilityOverlayMetadata struct {
	ProviderSourceRef    string `json:"providerSourceRef"`
	ProviderRevision     string `json:"providerRevision"`
	TerraformDocsVersion string `json:"terraformDocsVersion"`
}

type mutabilityOverlaySourceContract struct {
	ASTPrimaryFacts    []string `json:"astPrimaryFacts"`
	ASTMutableAlias    string   `json:"astMutableAlias"`
	DocsOverlayScope   string   `json:"docsOverlayScope"`
	DocsEvidenceSource string   `json:"docsEvidenceSource"`
}

type mutabilityOverlayResource struct {
	Service              string `json:"service"`
	Kind                 string `json:"kind"`
	FormalSlug           string `json:"formalSlug"`
	ProviderResource     string `json:"providerResource"`
	RepoAuthoredSpecPath string `json:"repoAuthoredSpecPath"`
	FormalImportPath     string `json:"formalImportPath"`
}

type mutabilityOverlayField struct {
	ASTFieldPath       string                        `json:"astFieldPath"`
	TerraformFieldPath string                        `json:"terraformFieldPath"`
	CanonicalJoinKey   string                        `json:"canonicalJoinKey"`
	PathShape          string                        `json:"pathShape"`
	AST                mutabilityOverlayASTState     `json:"ast"`
	Docs               mutabilityOverlayDocsEvidence `json:"docs"`
	Merge              mutabilityOverlayMergeResult  `json:"merge"`
	Provenance         mutabilityOverlayProvenance   `json:"provenance"`
}

type mutabilityOverlayASTState struct {
	UpdateCandidateState string   `json:"updateCandidateState"`
	ForceNew             bool     `json:"forceNew"`
	ConflictsWith        []string `json:"conflictsWith"`
}

type mutabilityOverlayDocsEvidence struct {
	EvidenceState string `json:"evidenceState"`
	Detail        string `json:"detail,omitempty"`
	RawSignal     string `json:"rawSignal,omitempty"`
}

type mutabilityOverlayMergeResult struct {
	JoinStatus  string `json:"joinStatus"`
	MergeCase   string `json:"mergeCase"`
	FinalPolicy string `json:"finalPolicy"`
}

type mutabilityOverlayProvenance struct {
	FormalImportPath     string   `json:"formalImportPath"`
	FormalSourceRef      string   `json:"formalSourceRef"`
	ASTSourceBucket      string   `json:"astSourceBucket"`
	TerraformDocsPage    string   `json:"terraformDocsPage"`
	TerraformDocsSection string   `json:"terraformDocsSection"`
	Notes                []string `json:"notes,omitempty"`
}

// resolveMutabilityOverlayDecision applies the conservative merge contract for one field row.
func resolveMutabilityOverlayDecision(astCandidateState string, forceNew bool, docsEvidenceState string, joinStatus string) mutabilityOverlayMergeResult {
	if forceNew || docsEvidenceState == mutabilityOverlayDocsStateReplacementRequired {
		return mutabilityOverlayMergeResult{
			JoinStatus:  joinStatus,
			MergeCase:   mutabilityOverlayMergeCaseReplacementRequired,
			FinalPolicy: mutabilityOverlayPolicyReplacementRequired,
		}
	}
	if astCandidateState != mutabilityOverlayASTStateUpdateCandidate {
		return mutabilityOverlayMergeResult{
			JoinStatus:  joinStatus,
			MergeCase:   mutabilityOverlayMergeCaseUnknown,
			FinalPolicy: mutabilityOverlayPolicyDenyInPlaceUpdate,
		}
	}
	if joinStatus != mutabilityOverlayJoinMatched {
		return mutabilityOverlayMergeResult{
			JoinStatus:  joinStatus,
			MergeCase:   mutabilityOverlayMergeCaseUnresolvedJoin,
			FinalPolicy: mutabilityOverlayPolicyUnknown,
		}
	}
	switch docsEvidenceState {
	case mutabilityOverlayDocsStateConfirmedUpdatable:
		return mutabilityOverlayMergeResult{
			JoinStatus:  joinStatus,
			MergeCase:   mutabilityOverlayMergeCaseDocsConfirmedCandidate,
			FinalPolicy: mutabilityOverlayPolicyAllowInPlaceUpdate,
		}
	case mutabilityOverlayDocsStateDeniedUpdatable:
		return mutabilityOverlayMergeResult{
			JoinStatus:  joinStatus,
			MergeCase:   mutabilityOverlayMergeCaseDocsDeniedCandidate,
			FinalPolicy: mutabilityOverlayPolicyDenyInPlaceUpdate,
		}
	case mutabilityOverlayDocsStateNotDocumented:
		return mutabilityOverlayMergeResult{
			JoinStatus:  joinStatus,
			MergeCase:   mutabilityOverlayMergeCaseASTOnlyCandidate,
			FinalPolicy: mutabilityOverlayPolicyUnknown,
		}
	default:
		return mutabilityOverlayMergeResult{
			JoinStatus:  joinStatus,
			MergeCase:   mutabilityOverlayMergeCaseUnknown,
			FinalPolicy: mutabilityOverlayPolicyUnknown,
		}
	}
}

func validateMutabilityOverlayDocument(doc mutabilityOverlayDocument) error {
	var errs []string
	if doc.SchemaVersion != mutabilityOverlaySchemaVersion {
		errs = append(errs, fmt.Sprintf("schemaVersion = %d, want %d", doc.SchemaVersion, mutabilityOverlaySchemaVersion))
	}
	if got := strings.TrimSpace(doc.Surface); got != mutabilityOverlaySurface {
		errs = append(errs, fmt.Sprintf("surface = %q, want %q", got, mutabilityOverlaySurface))
	}
	if got := strings.TrimSpace(doc.ContractVersion); got != mutabilityOverlayContractVersion {
		errs = append(errs, fmt.Sprintf("contractVersion = %q, want %q", got, mutabilityOverlayContractVersion))
	}

	errs = append(errs, validateNonEmptyString("metadata.providerSourceRef", doc.Metadata.ProviderSourceRef)...)
	errs = append(errs, validateNonEmptyString("metadata.providerRevision", doc.Metadata.ProviderRevision)...)
	errs = append(errs, validateNonEmptyString("metadata.terraformDocsVersion", doc.Metadata.TerraformDocsVersion)...)

	if !slices.Equal(doc.SourceContract.ASTPrimaryFacts, mutabilityOverlayASTPrimaryFacts) {
		errs = append(errs, fmt.Sprintf("sourceContract.astPrimaryFacts = %v, want %v", doc.SourceContract.ASTPrimaryFacts, mutabilityOverlayASTPrimaryFacts))
	}
	if got := strings.TrimSpace(doc.SourceContract.ASTMutableAlias); got != mutabilityOverlayUpdateCandidateAlias {
		errs = append(errs, fmt.Sprintf("sourceContract.astMutableAlias = %q, want %q", got, mutabilityOverlayUpdateCandidateAlias))
	}
	if got := strings.TrimSpace(doc.SourceContract.DocsOverlayScope); got != mutabilityOverlayDocsOverlayScope {
		errs = append(errs, fmt.Sprintf("sourceContract.docsOverlayScope = %q, want %q", got, mutabilityOverlayDocsOverlayScope))
	}
	if got := strings.TrimSpace(doc.SourceContract.DocsEvidenceSource); got != mutabilityOverlayDocsEvidenceSource {
		errs = append(errs, fmt.Sprintf("sourceContract.docsEvidenceSource = %q, want %q", got, mutabilityOverlayDocsEvidenceSource))
	}

	errs = append(errs, validateNonEmptyString("resource.service", doc.Resource.Service)...)
	errs = append(errs, validateNonEmptyString("resource.kind", doc.Resource.Kind)...)
	errs = append(errs, validateNonEmptyString("resource.formalSlug", doc.Resource.FormalSlug)...)
	errs = append(errs, validateNonEmptyString("resource.providerResource", doc.Resource.ProviderResource)...)
	errs = append(errs, validateNonEmptyString("resource.repoAuthoredSpecPath", doc.Resource.RepoAuthoredSpecPath)...)
	errs = append(errs, validateNonEmptyString("resource.formalImportPath", doc.Resource.FormalImportPath)...)

	if len(doc.Fields) == 0 {
		errs = append(errs, "fields must not be empty")
	}
	for index, field := range doc.Fields {
		errs = append(errs, validateMutabilityOverlayField(fmt.Sprintf("fields[%d]", index), field)...)
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func validateMutabilityOverlayField(prefix string, field mutabilityOverlayField) []string {
	var errs []string
	errs = append(errs, validateNonEmptyString(prefix+".astFieldPath", field.ASTFieldPath)...)
	errs = append(errs, validateNonEmptyString(prefix+".terraformFieldPath", field.TerraformFieldPath)...)
	errs = append(errs, validateNonEmptyString(prefix+".canonicalJoinKey", field.CanonicalJoinKey)...)
	errs = append(errs, validateAllowedValue(prefix+".pathShape", field.PathShape, mutabilityOverlayPathShapes)...)
	errs = append(errs, validateAllowedValue(prefix+".ast.updateCandidateState", field.AST.UpdateCandidateState, mutabilityOverlayASTCandidateStates)...)
	errs = append(errs, validateAllowedValue(prefix+".docs.evidenceState", field.Docs.EvidenceState, mutabilityOverlayDocsEvidenceStates)...)
	errs = append(errs, validateAllowedValue(prefix+".merge.joinStatus", field.Merge.JoinStatus, mutabilityOverlayJoinStatuses)...)
	errs = append(errs, validateAllowedValue(prefix+".merge.mergeCase", field.Merge.MergeCase, mutabilityOverlayMergeCases)...)
	errs = append(errs, validateAllowedValue(prefix+".merge.finalPolicy", field.Merge.FinalPolicy, mutabilityOverlayFinalPolicies)...)
	errs = append(errs, validateNonEmptyStringList(prefix+".ast.conflictsWith", field.AST.ConflictsWith)...)

	errs = append(errs, validateNonEmptyString(prefix+".provenance.formalImportPath", field.Provenance.FormalImportPath)...)
	errs = append(errs, validateNonEmptyString(prefix+".provenance.formalSourceRef", field.Provenance.FormalSourceRef)...)
	errs = append(errs, validateAllowedValue(prefix+".provenance.astSourceBucket", field.Provenance.ASTSourceBucket, mutabilityOverlayASTSourceBuckets)...)
	errs = append(errs, validateNonEmptyString(prefix+".provenance.terraformDocsPage", field.Provenance.TerraformDocsPage)...)
	errs = append(errs, validateNonEmptyString(prefix+".provenance.terraformDocsSection", field.Provenance.TerraformDocsSection)...)
	errs = append(errs, validateNonEmptyStringList(prefix+".provenance.notes", field.Provenance.Notes)...)

	if field.AST.UpdateCandidateState != mutabilityOverlayASTStateUpdateCandidate && !field.AST.ForceNew {
		errs = append(errs, fmt.Sprintf("%s.ast must be updateCandidate or forceNew", prefix))
	}
	switch field.Provenance.ASTSourceBucket {
	case mutabilityOverlayASTSourceBucketMutable:
		if field.AST.UpdateCandidateState != mutabilityOverlayASTStateUpdateCandidate {
			errs = append(errs, fmt.Sprintf("%s.provenance.astSourceBucket=%q requires ast.updateCandidateState=%q", prefix, field.Provenance.ASTSourceBucket, mutabilityOverlayASTStateUpdateCandidate))
		}
		if field.AST.ForceNew {
			errs = append(errs, fmt.Sprintf("%s.provenance.astSourceBucket=%q must not carry forceNew=true", prefix, field.Provenance.ASTSourceBucket))
		}
	case mutabilityOverlayASTSourceBucketForceNew:
		if !field.AST.ForceNew {
			errs = append(errs, fmt.Sprintf("%s.provenance.astSourceBucket=%q requires ast.forceNew=true", prefix, field.Provenance.ASTSourceBucket))
		}
	}

	expected := resolveMutabilityOverlayDecision(field.AST.UpdateCandidateState, field.AST.ForceNew, field.Docs.EvidenceState, field.Merge.JoinStatus)
	if field.Merge != expected {
		errs = append(errs, fmt.Sprintf("%s.merge = %+v, want %+v", prefix, field.Merge, expected))
	}
	return errs
}

func validateNonEmptyString(name, value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{fmt.Sprintf("%s must not be empty", name)}
	}
	return nil
}

func validateAllowedValue(name, value string, allowed []string) []string {
	if !slices.Contains(allowed, value) {
		return []string{fmt.Sprintf("%s = %q, want one of %v", name, value, allowed)}
	}
	return nil
}

func validateNonEmptyStringList(name string, values []string) []string {
	var errs []string
	for index, value := range values {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, fmt.Sprintf("%s[%d] must not be empty", name, index))
		}
	}
	return errs
}

func canonicalMutabilityOverlaySchema() map[string]any {
	return map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "https://oracle.com/oci-service-operator/schemas/generator/mutability-overlay-v1.schema.json",
		"title":                "OSOK Generator Mutability Overlay Contract",
		"type":                 "object",
		"additionalProperties": false,
		"required": []string{
			"schemaVersion",
			"surface",
			"contractVersion",
			"metadata",
			"sourceContract",
			"resource",
			"fields",
		},
		"properties": map[string]any{
			"schemaVersion":   map[string]any{"const": mutabilityOverlaySchemaVersion},
			"surface":         map[string]any{"const": mutabilityOverlaySurface},
			"contractVersion": map[string]any{"const": mutabilityOverlayContractVersion},
			"metadata":        schemaRef("#/$defs/metadata"),
			"sourceContract":  schemaRef("#/$defs/sourceContract"),
			"resource":        schemaRef("#/$defs/resource"),
			"fields": map[string]any{
				"type":     "array",
				"minItems": 1,
				"items":    schemaRef("#/$defs/field"),
			},
		},
		"$defs": map[string]any{
			"nonEmptyString": nonEmptyStringSchema(),
			"metadata": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required": []string{
					"providerSourceRef",
					"providerRevision",
					"terraformDocsVersion",
				},
				"properties": map[string]any{
					"providerSourceRef":    schemaRef("#/$defs/nonEmptyString"),
					"providerRevision":     schemaRef("#/$defs/nonEmptyString"),
					"terraformDocsVersion": schemaRef("#/$defs/nonEmptyString"),
				},
			},
			"sourceContract": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required": []string{
					"astPrimaryFacts",
					"astMutableAlias",
					"docsOverlayScope",
					"docsEvidenceSource",
				},
				"properties": map[string]any{
					"astPrimaryFacts":    map[string]any{"type": "array", "const": append([]string(nil), mutabilityOverlayASTPrimaryFacts...)},
					"astMutableAlias":    map[string]any{"const": mutabilityOverlayUpdateCandidateAlias},
					"docsOverlayScope":   map[string]any{"const": mutabilityOverlayDocsOverlayScope},
					"docsEvidenceSource": map[string]any{"const": mutabilityOverlayDocsEvidenceSource},
				},
			},
			"resource": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required": []string{
					"service",
					"kind",
					"formalSlug",
					"providerResource",
					"repoAuthoredSpecPath",
					"formalImportPath",
				},
				"properties": map[string]any{
					"service":              schemaRef("#/$defs/nonEmptyString"),
					"kind":                 schemaRef("#/$defs/nonEmptyString"),
					"formalSlug":           schemaRef("#/$defs/nonEmptyString"),
					"providerResource":     schemaRef("#/$defs/nonEmptyString"),
					"repoAuthoredSpecPath": schemaRef("#/$defs/nonEmptyString"),
					"formalImportPath":     schemaRef("#/$defs/nonEmptyString"),
				},
			},
			"field": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required": []string{
					"astFieldPath",
					"terraformFieldPath",
					"canonicalJoinKey",
					"pathShape",
					"ast",
					"docs",
					"merge",
					"provenance",
				},
				"properties": map[string]any{
					"astFieldPath":       schemaRef("#/$defs/nonEmptyString"),
					"terraformFieldPath": schemaRef("#/$defs/nonEmptyString"),
					"canonicalJoinKey":   schemaRef("#/$defs/nonEmptyString"),
					"pathShape":          enumStringSchema(mutabilityOverlayPathShapes),
					"ast":                schemaRef("#/$defs/ast"),
					"docs":               schemaRef("#/$defs/docs"),
					"merge":              schemaRef("#/$defs/merge"),
					"provenance":         schemaRef("#/$defs/provenance"),
				},
			},
			"ast": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"updateCandidateState", "forceNew", "conflictsWith"},
				"properties": map[string]any{
					"updateCandidateState": enumStringSchema(mutabilityOverlayASTCandidateStates),
					"forceNew":             map[string]any{"type": "boolean"},
					"conflictsWith":        stringArraySchema(),
				},
			},
			"docs": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"evidenceState"},
				"properties": map[string]any{
					"evidenceState": enumStringSchema(mutabilityOverlayDocsEvidenceStates),
					"detail":        schemaRef("#/$defs/nonEmptyString"),
					"rawSignal":     schemaRef("#/$defs/nonEmptyString"),
				},
			},
			"merge": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"joinStatus", "mergeCase", "finalPolicy"},
				"properties": map[string]any{
					"joinStatus":  enumStringSchema(mutabilityOverlayJoinStatuses),
					"mergeCase":   enumStringSchema(mutabilityOverlayMergeCases),
					"finalPolicy": enumStringSchema(mutabilityOverlayFinalPolicies),
				},
			},
			"provenance": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required": []string{
					"formalImportPath",
					"formalSourceRef",
					"astSourceBucket",
					"terraformDocsPage",
					"terraformDocsSection",
				},
				"properties": map[string]any{
					"formalImportPath":     schemaRef("#/$defs/nonEmptyString"),
					"formalSourceRef":      schemaRef("#/$defs/nonEmptyString"),
					"astSourceBucket":      enumStringSchema(mutabilityOverlayASTSourceBuckets),
					"terraformDocsPage":    schemaRef("#/$defs/nonEmptyString"),
					"terraformDocsSection": schemaRef("#/$defs/nonEmptyString"),
					"notes":                stringArraySchema(),
				},
			},
		},
	}
}

func exampleMutabilityOverlayDocument() mutabilityOverlayDocument {
	resource := mutabilityOverlayResource{
		Service:              "nosql",
		Kind:                 "Table",
		FormalSlug:           "table",
		ProviderResource:     "oci_nosql_table",
		RepoAuthoredSpecPath: "controllers/nosql/table/spec.cfg",
		FormalImportPath:     "formal/imports/nosql/table.json",
	}
	return mutabilityOverlayDocument{
		SchemaVersion:   mutabilityOverlaySchemaVersion,
		Surface:         mutabilityOverlaySurface,
		ContractVersion: mutabilityOverlayContractVersion,
		Metadata: mutabilityOverlayMetadata{
			ProviderSourceRef:    mutabilityOverlayProviderSourceRef,
			ProviderRevision:     "eb653febb1bab4cc6650a96d404a8baf36fdf671",
			TerraformDocsVersion: "contract-example-v1",
		},
		SourceContract: mutabilityOverlaySourceContract{
			ASTPrimaryFacts:    append([]string(nil), mutabilityOverlayASTPrimaryFacts...),
			ASTMutableAlias:    mutabilityOverlayUpdateCandidateAlias,
			DocsOverlayScope:   mutabilityOverlayDocsOverlayScope,
			DocsEvidenceSource: mutabilityOverlayDocsEvidenceSource,
		},
		Resource: resource,
		Fields: []mutabilityOverlayField{
			newExampleMutabilityOverlayField(resource, exampleMutabilityOverlayFieldConfig{
				ASTFieldPath:       "freeform_tags",
				TerraformFieldPath: "freeform_tags",
				CanonicalJoinKey:   "freeformTags",
				PathShape:          mutabilityOverlayPathShapeScalar,
				ASTState:           mutabilityOverlayASTStateUpdateCandidate,
				DocsState:          mutabilityOverlayDocsStateNotDocumented,
				JoinStatus:         mutabilityOverlayJoinMatched,
				ASTSourceBucket:    mutabilityOverlayASTSourceBucketMutable,
				Detail:             "No explicit Terraform docs update signal was found for this AST candidate.",
				RawSignal:          "No (Updatable) or replacement wording matched this field.",
				Notes: []string{
					"Documents the AST-only candidate case without widening the update allowlist.",
				},
			}),
			newExampleMutabilityOverlayField(resource, exampleMutabilityOverlayFieldConfig{
				ASTFieldPath:       "ddl_statement",
				TerraformFieldPath: "ddl_statement",
				CanonicalJoinKey:   "ddlStatement",
				PathShape:          mutabilityOverlayPathShapeScalar,
				ASTState:           mutabilityOverlayASTStateUpdateCandidate,
				DocsState:          mutabilityOverlayDocsStateConfirmedUpdatable,
				JoinStatus:         mutabilityOverlayJoinMatched,
				ASTSourceBucket:    mutabilityOverlayASTSourceBucketMutable,
				Detail:             "Argument Reference marks the field as updatable in place.",
				RawSignal:          "(Updatable)",
				Notes: []string{
					"Represents a docs-confirmed AST updateCandidate row.",
				},
			}),
			newExampleMutabilityOverlayField(resource, exampleMutabilityOverlayFieldConfig{
				ASTFieldPath:       "table_limits.max_read_units",
				TerraformFieldPath: "table_limits.max_read_units",
				CanonicalJoinKey:   "tableLimits.maxReadUnits",
				PathShape:          mutabilityOverlayPathShapeObject,
				ASTState:           mutabilityOverlayASTStateUpdateCandidate,
				DocsState:          mutabilityOverlayDocsStateDeniedUpdatable,
				JoinStatus:         mutabilityOverlayJoinMatched,
				ASTSourceBucket:    mutabilityOverlayASTSourceBucketMutable,
				Detail:             "Docs text says changing this field requires recreating throughput settings elsewhere.",
				RawSignal:          "Updating this value after creation is not supported.",
				Notes: []string{
					"Shows a nested block path that docs deny even though AST classified it as an updateCandidate.",
				},
			}),
			newExampleMutabilityOverlayField(resource, exampleMutabilityOverlayFieldConfig{
				ASTFieldPath:       "name",
				TerraformFieldPath: "name",
				CanonicalJoinKey:   "name",
				PathShape:          mutabilityOverlayPathShapeScalar,
				ASTState:           mutabilityOverlayASTStateNotUpdateCandidate,
				ForceNew:           true,
				DocsState:          mutabilityOverlayDocsStateNotDocumented,
				JoinStatus:         mutabilityOverlayJoinMatched,
				ASTSourceBucket:    mutabilityOverlayASTSourceBucketForceNew,
				Detail:             "AST forceNew remains authoritative even without docs overlay evidence.",
				Notes: []string{
					"Encodes the replacement-required case sourced directly from formal import facts.",
				},
			}),
			newExampleMutabilityOverlayField(resource, exampleMutabilityOverlayFieldConfig{
				ASTFieldPath:       "table_limits.read_units[].capacity_mode",
				TerraformFieldPath: "table_limits.read_units[].capacity_mode",
				CanonicalJoinKey:   "tableLimits.readUnits[].capacityMode",
				PathShape:          mutabilityOverlayPathShapeListItem,
				ASTState:           mutabilityOverlayASTStateUpdateCandidate,
				DocsState:          mutabilityOverlayDocsStateAmbiguous,
				JoinStatus:         mutabilityOverlayJoinMatched,
				ASTSourceBucket:    mutabilityOverlayASTSourceBucketMutable,
				Detail:             "Docs wording mixes conditional in-place update and recreate guidance.",
				RawSignal:          "Changing replica capacity may update or may require replacement depending on workload mode.",
				Notes: []string{
					"Shows how collection-item paths and ambiguous docs text stay preserved as evidence instead of becoming allowlisted.",
				},
			}),
			newExampleMutabilityOverlayField(resource, exampleMutabilityOverlayFieldConfig{
				ASTFieldPath:       "defined_tags",
				TerraformFieldPath: "defined_tags.*",
				CanonicalJoinKey:   "definedTags.*",
				PathShape:          mutabilityOverlayPathShapeMapEntry,
				ASTState:           mutabilityOverlayASTStateUpdateCandidate,
				DocsState:          mutabilityOverlayDocsStateConfirmedUpdatable,
				JoinStatus:         mutabilityOverlayJoinUnresolved,
				ASTSourceBucket:    mutabilityOverlayASTSourceBucketMutable,
				Detail:             "Docs expose map-entry evidence but the AST row is only available at the map root.",
				RawSignal:          "Defined tags can be updated key by key.",
				Notes: []string{
					"Represents an unresolved canonical join that later work must normalize without silently attaching docs evidence to the wrong field.",
				},
			}),
			newExampleMutabilityOverlayField(resource, exampleMutabilityOverlayFieldConfig{
				ASTFieldPath:       "table_limits.max_storage_in_gbs",
				TerraformFieldPath: "table_limits.max_storage_in_gbs",
				CanonicalJoinKey:   "tableLimits.maxStorageInGbs",
				PathShape:          mutabilityOverlayPathShapeUnsupported,
				ASTState:           mutabilityOverlayASTStateUpdateCandidate,
				DocsState:          mutabilityOverlayDocsStateUnsupported,
				JoinStatus:         mutabilityOverlayJoinMatched,
				ASTSourceBucket:    mutabilityOverlayASTSourceBucketMutable,
				Detail:             "The parser captured the field mention but does not yet support this docs shape deterministically.",
				RawSignal:          "Storage block limits are described in a table layout that the first parser revision does not consume.",
				Notes: []string{
					"Demonstrates that unsupported field shapes are preserved as typed evidence with an unknown final policy.",
				},
			}),
		},
	}
}

type exampleMutabilityOverlayFieldConfig struct {
	ASTFieldPath       string
	TerraformFieldPath string
	CanonicalJoinKey   string
	PathShape          string
	ASTState           string
	ForceNew           bool
	DocsState          string
	JoinStatus         string
	ASTSourceBucket    string
	Detail             string
	RawSignal          string
	Notes              []string
}

func newExampleMutabilityOverlayField(resource mutabilityOverlayResource, cfg exampleMutabilityOverlayFieldConfig) mutabilityOverlayField {
	return mutabilityOverlayField{
		ASTFieldPath:       cfg.ASTFieldPath,
		TerraformFieldPath: cfg.TerraformFieldPath,
		CanonicalJoinKey:   cfg.CanonicalJoinKey,
		PathShape:          cfg.PathShape,
		AST: mutabilityOverlayASTState{
			UpdateCandidateState: cfg.ASTState,
			ForceNew:             cfg.ForceNew,
			ConflictsWith:        []string{},
		},
		Docs: mutabilityOverlayDocsEvidence{
			EvidenceState: cfg.DocsState,
			Detail:        cfg.Detail,
			RawSignal:     cfg.RawSignal,
		},
		Merge: resolveMutabilityOverlayDecision(cfg.ASTState, cfg.ForceNew, cfg.DocsState, cfg.JoinStatus),
		Provenance: mutabilityOverlayProvenance{
			FormalImportPath:     resource.FormalImportPath,
			FormalSourceRef:      mutabilityOverlayProviderSourceRef,
			ASTSourceBucket:      cfg.ASTSourceBucket,
			TerraformDocsPage:    "nosql_table#argument-reference",
			TerraformDocsSection: "Argument Reference",
			Notes:                append([]string(nil), cfg.Notes...),
		},
	}
}

func schemaRef(ref string) map[string]any {
	return map[string]any{"$ref": ref}
}

func nonEmptyStringSchema() map[string]any {
	return map[string]any{
		"type":      "string",
		"minLength": 1,
	}
}

func enumStringSchema(values []string) map[string]any {
	return map[string]any{
		"type": "string",
		"enum": append([]string(nil), values...),
	}
}

func stringArraySchema() map[string]any {
	return map[string]any{
		"type":        "array",
		"items":       schemaRef("#/$defs/nonEmptyString"),
		"uniqueItems": true,
	}
}

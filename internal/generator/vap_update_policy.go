package generator

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	vapUpdatePolicySchemaVersion             = 1
	vapUpdatePolicySurface                   = "generator-vap-update-policy-input"
	vapUpdatePolicyContractVersion           = "v1alpha1"
	vapUpdatePolicyGeneratedRootRelativePath = "internal/generator/generated/vap_update_policy"
	vapUpdatePolicySpecPathPrefix            = "spec"
)

var vapUpdatePolicyDenyDecisions = []string{
	mutabilityOverlayPolicyDenyInPlaceUpdate,
	mutabilityOverlayPolicyReplacementRequired,
	mutabilityOverlayPolicyUnknown,
}

// vapUpdatePolicyDocument captures the generator-owned ValidatingAdmissionPolicy input surface.
type vapUpdatePolicyDocument struct {
	SchemaVersion   int                     `json:"schemaVersion"`
	Surface         string                  `json:"surface"`
	ContractVersion string                  `json:"contractVersion"`
	Metadata        vapUpdatePolicyMetadata `json:"metadata"`
	Target          vapUpdatePolicyTarget   `json:"target"`
	Update          vapUpdatePolicyUpdate   `json:"update"`
}

type vapUpdatePolicyMetadata struct {
	SourceSurface        string `json:"sourceSurface"`
	ProviderSourceRef    string `json:"providerSourceRef"`
	ProviderRevision     string `json:"providerRevision"`
	TerraformDocsVersion string `json:"terraformDocsVersion"`
}

type vapUpdatePolicyTarget struct {
	Service          string `json:"service"`
	APIVersion       string `json:"apiVersion"`
	Kind             string `json:"kind"`
	FormalSlug       string `json:"formalSlug"`
	ProviderResource string `json:"providerResource"`
	SpecPathPrefix   string `json:"specPathPrefix"`
}

type vapUpdatePolicyUpdate struct {
	AllowInPlacePaths []string              `json:"allowInPlacePaths"`
	DenyRules         []vapUpdatePolicyRule `json:"denyRules"`
}

type vapUpdatePolicyRule struct {
	FieldPath         string `json:"fieldPath"`
	Decision          string `json:"decision"`
	MergeCase         string `json:"mergeCase"`
	DocsEvidenceState string `json:"docsEvidenceState"`
	Detail            string `json:"detail,omitempty"`
}

func validateVAPUpdatePolicyDocument(doc vapUpdatePolicyDocument) error {
	var errs []string
	if doc.SchemaVersion != vapUpdatePolicySchemaVersion {
		errs = append(errs, fmt.Sprintf("schemaVersion = %d, want %d", doc.SchemaVersion, vapUpdatePolicySchemaVersion))
	}
	if got := strings.TrimSpace(doc.Surface); got != vapUpdatePolicySurface {
		errs = append(errs, fmt.Sprintf("surface = %q, want %q", got, vapUpdatePolicySurface))
	}
	if got := strings.TrimSpace(doc.ContractVersion); got != vapUpdatePolicyContractVersion {
		errs = append(errs, fmt.Sprintf("contractVersion = %q, want %q", got, vapUpdatePolicyContractVersion))
	}

	if got := strings.TrimSpace(doc.Metadata.SourceSurface); got != mutabilityOverlaySurface {
		errs = append(errs, fmt.Sprintf("metadata.sourceSurface = %q, want %q", got, mutabilityOverlaySurface))
	}
	errs = append(errs, validateNonEmptyString("metadata.providerSourceRef", doc.Metadata.ProviderSourceRef)...)
	errs = append(errs, validateNonEmptyString("metadata.providerRevision", doc.Metadata.ProviderRevision)...)
	errs = append(errs, validateNonEmptyString("metadata.terraformDocsVersion", doc.Metadata.TerraformDocsVersion)...)

	errs = append(errs, validateNonEmptyString("target.service", doc.Target.Service)...)
	errs = append(errs, validateNonEmptyString("target.apiVersion", doc.Target.APIVersion)...)
	errs = append(errs, validateNonEmptyString("target.kind", doc.Target.Kind)...)
	errs = append(errs, validateNonEmptyString("target.formalSlug", doc.Target.FormalSlug)...)
	errs = append(errs, validateNonEmptyString("target.providerResource", doc.Target.ProviderResource)...)
	if got := strings.TrimSpace(doc.Target.SpecPathPrefix); got != vapUpdatePolicySpecPathPrefix {
		errs = append(errs, fmt.Sprintf("target.specPathPrefix = %q, want %q", got, vapUpdatePolicySpecPathPrefix))
	}

	errs = append(errs, validateNonEmptyStringList("update.allowInPlacePaths", doc.Update.AllowInPlacePaths)...)
	if !slices.Equal(doc.Update.AllowInPlacePaths, uniqueSortedStrings(doc.Update.AllowInPlacePaths)) {
		errs = append(errs, "update.allowInPlacePaths must be unique and sorted")
	}
	allowSet := make(map[string]struct{}, len(doc.Update.AllowInPlacePaths))
	for _, fieldPath := range doc.Update.AllowInPlacePaths {
		allowSet[fieldPath] = struct{}{}
	}
	for index, rule := range doc.Update.DenyRules {
		prefix := fmt.Sprintf("update.denyRules[%d]", index)
		errs = append(errs, validateNonEmptyString(prefix+".fieldPath", rule.FieldPath)...)
		errs = append(errs, validateAllowedValue(prefix+".decision", rule.Decision, vapUpdatePolicyDenyDecisions)...)
		errs = append(errs, validateAllowedValue(prefix+".mergeCase", rule.MergeCase, mutabilityOverlayMergeCases)...)
		errs = append(errs, validateAllowedValue(prefix+".docsEvidenceState", rule.DocsEvidenceState, mutabilityOverlayDocsEvidenceStates)...)
		if _, ok := allowSet[rule.FieldPath]; ok {
			errs = append(errs, fmt.Sprintf("%s.fieldPath %q also appears in update.allowInPlacePaths", prefix, rule.FieldPath))
		}
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

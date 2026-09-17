package apispec

import "strings"

const intentionalUntrackedPrefix = "Intentionally untracked: "

var reviewedUntrackedReasons = map[string]string{
	"DataintegrationConnection":                           intentionalUntrackedPrefix + "selected generated staging kind has no reusable desired-state SDK payload in the current SDK discovery; runtime implementation needs resource-local API coverage review.",
	"DataintegrationDataAsset":                            intentionalUntrackedPrefix + "selected generated staging kind has no reusable desired-state SDK payload in the current SDK discovery; runtime implementation needs resource-local API coverage review.",
	"DataintegrationTask":                                 intentionalUntrackedPrefix + "selected generated staging kind has no reusable desired-state SDK payload in the current SDK discovery; runtime implementation needs resource-local API coverage review.",
	"FleetsoftwareupdateFsuAction":                        intentionalUntrackedPrefix + "selected generated staging kind has no reusable desired-state SDK payload in the current SDK discovery; runtime implementation needs resource-local API coverage review.",
	"GenericartifactscontentGenericArtifactContent":       binaryContentReason("the SDK only returns artifact bytes or response metadata for content reads"),
	"GenericartifactscontentGenericArtifactContentByPath": binaryContentReason("the desired content bytes are sourced from Kubernetes Secrets and the OCI SDK request body is a binary payload instead of a reusable desired-state struct"),
	"ManagementagentDataSource":                           intentionalUntrackedPrefix + "DataSource uses polymorphic create, update, and response body payloads; generated runtime still exposes the concrete CRD fields, but APISpec coverage needs resource-local review before mapping one concrete SDK shape.",
	"NetworkfirewallApplication":                          intentionalUntrackedPrefix + "selected generated staging kind has no reusable desired-state SDK payload in the current SDK discovery; runtime implementation needs resource-local API coverage review.",
	"NetworkfirewallTunnelInspectionRule":                 intentionalUntrackedPrefix + "selected generated staging kind has no reusable desired-state SDK payload in the current SDK discovery; runtime implementation needs resource-local API coverage review.",
}

func reviewedUntrackedReason(targetName string) string {
	return reviewedUntrackedReasons[strings.TrimSpace(targetName)]
}

func isIntentionalUntrackedReason(reason string) bool {
	return strings.HasPrefix(strings.TrimSpace(reason), intentionalUntrackedPrefix)
}

func excludedMappingReason(mapping SDKMapping) string {
	if strings.HasPrefix(strings.TrimSpace(mapping.Reason), intentionalUntrackedPrefix) {
		return strings.TrimSpace(mapping.Reason)
	}
	if strings.TrimSpace(mapping.Reason) != "" {
		return intentionalUntrackedPrefix + strings.TrimSpace(mapping.Reason)
	}
	return intentionalUntrackedPrefix + "mapping is intentionally excluded from desired-state coverage by validator registry metadata."
}

func scalarContentReason(description string) string {
	return intentionalUntrackedPrefix + "spec is empty and " + description + ", not a reusable SDK struct for desired-state validation."
}

func binaryContentReason(description string) string {
	return intentionalUntrackedPrefix + "spec is empty and " + description + ", not a reusable SDK struct for desired-state validation."
}

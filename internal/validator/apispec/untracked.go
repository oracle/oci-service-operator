package apispec

import "strings"

const intentionalUntrackedPrefix = "Intentionally untracked: "

var reviewedUntrackedReasons = map[string]string{
	"ManagementagentDataSource": intentionalUntrackedPrefix + "DataSource uses polymorphic create, update, and response body payloads; generated runtime still exposes the concrete CRD fields, but APISpec coverage needs resource-local review before mapping one concrete SDK shape.",
}

func reviewedUntrackedReason(targetName string) string {
	return reviewedUntrackedReasons[strings.TrimSpace(targetName)]
}

func isIntentionalUntrackedReason(reason string) bool {
	return strings.HasPrefix(strings.TrimSpace(reason), intentionalUntrackedPrefix)
}

func readOnlyResponseReason(sdkType string) string {
	return intentionalUntrackedPrefix + "spec is empty and the SDK only exposes read-only response payloads via " + sdkType + ", which would make every field appear missing from desired state."
}

func responseBodyReason(sdkType string) string {
	return intentionalUntrackedPrefix + "spec is empty and the SDK only returns " + sdkType + " in the response body, not as a desired-state payload."
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

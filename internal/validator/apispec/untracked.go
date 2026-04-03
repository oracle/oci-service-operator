package apispec

import "strings"

const intentionalUntrackedPrefix = "Intentionally untracked: "

var reviewedUntrackedReasons = map[string]string{
	"CoreConsoleHistoryContent":        binaryContentReason("the OCI API only returns console history content as a raw binary payload"),
	"CoreCpeDeviceConfigContent":       scalarContentReason("the OCI API only returns CPE device config content as plain-text content"),
	"CoreIpsecCpeDeviceConfigContent":  scalarContentReason("the OCI API only returns IPSec CPE device config content as plain-text content"),
	"CoreTunnelCpeDeviceConfigContent": scalarContentReason("the OCI API only returns tunnel CPE device config content as plain-text content"),
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

package main

import (
	"slices"
	"testing"
)

func TestDeriveSDKTypesIncludesReadOnlyResponseOverrides(t *testing.T) {
	t.Parallel()

	structs := map[string]bool{
		"DrgAttachmentInfo": true,
	}

	got := deriveSDKTypes("core", "AllDrgAttachment", makeTargetName("core", "AllDrgAttachment"), structs)
	want := []string{"DrgAttachmentInfo"}
	if !slices.Equal(got, want) {
		t.Fatalf("deriveSDKTypes() = %v, want %v", got, want)
	}
}

func TestResolveStatusTypePrefersScannedStatusAndFallsBackToExistingTargets(t *testing.T) {
	t.Parallel()

	if got := resolveStatusType(apiTypeInfo{Spec: "Vcn", Status: "VcnObservedState"}, false, specTarget{}); got != "VcnObservedState" {
		t.Fatalf("resolveStatusType(scanned) = %q, want %q", got, "VcnObservedState")
	}
	if got := resolveStatusType(apiTypeInfo{Spec: "Vcn"}, true, specTarget{Status: "LegacyStatus"}); got != "LegacyStatus" {
		t.Fatalf("resolveStatusType(existing) = %q, want %q", got, "LegacyStatus")
	}
	if got := resolveStatusType(apiTypeInfo{Spec: "Vcn"}, false, specTarget{}); got != "" {
		t.Fatalf("resolveStatusType(empty) = %q, want empty string", got)
	}
	if got := resolveStatusType(apiTypeInfo{Spec: "Vcn", Status: "VcnStatus"}, true, specTarget{Status: "LegacyStatus"}); got != "VcnStatus" {
		t.Fatalf("resolveStatusType(scanned+existing) = %q, want %q", got, "VcnStatus")
	}
}

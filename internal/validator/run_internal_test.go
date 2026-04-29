package validator

import (
	"testing"

	"github.com/oracle/oci-service-operator/internal/validator/apispec"
	"github.com/oracle/oci-service-operator/internal/validator/config"
	"github.com/oracle/oci-service-operator/internal/validator/diff"
)

func TestFilterAPIReportByServiceKeepsUntrackedTargets(t *testing.T) {
	report := apispec.Report{
		Structs: []apispec.StructReport{
			{
				Service:        "queue",
				Spec:           "QueueChannel",
				TrackingStatus: apispec.TrackingStatusUntracked,
				TrackingReason: "Generated API spec has no mapped SDK payloads in the validator target registry.",
			},
			{
				Service:   "core",
				Spec:      "CoreInstance",
				SDKStruct: "core.Instance",
			},
		},
	}

	filtered := filterAPIReportByService(report, "queue")
	if len(filtered.Structs) != 1 {
		t.Fatalf("filterAPIReportByService() report count = %d, want 1", len(filtered.Structs))
	}
	if filtered.Structs[0].Spec != "QueueChannel" {
		t.Fatalf("filterAPIReportByService() kept %q, want %q", filtered.Structs[0].Spec, "QueueChannel")
	}
}

func TestFilterControllerReportByStructs(t *testing.T) {
	report := diff.Report{
		Structs: []diff.StructReport{
			{StructType: "core.Instance"},
			{StructType: "core.Subnet"},
		},
	}

	filtered := filterControllerReportByStructs(report, map[string]struct{}{
		"core.Instance": {},
	})
	if len(filtered.Structs) != 1 {
		t.Fatalf("filterControllerReportByStructs() report count = %d, want 1", len(filtered.Structs))
	}
	if filtered.Structs[0].StructType != "core.Instance" {
		t.Fatalf("filterControllerReportByStructs() kept %q, want %q", filtered.Structs[0].StructType, "core.Instance")
	}
}

func TestFilterAPIReportBySelectedServicesHonorsSelectedKinds(t *testing.T) {
	configPath := writeUpgradeSelectionConfig(t)
	selectedSurface, err := loadSelectedValidatorSurface(config.Options{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("loadSelectedValidatorSurface() error = %v", err)
	}

	report := apispec.Report{
		Structs: []apispec.StructReport{
			{Service: "core", Spec: "Instance", SDKStruct: "core.Instance"},
			{Service: "core", Spec: "Subnet", SDKStruct: "core.Subnet"},
			{Service: "mysql", Spec: "DbSystem", SDKStruct: "mysql.DbSystem"},
			{Service: "identity", Spec: "Compartment", SDKStruct: "identity.Compartment"},
		},
	}

	filtered := filterAPIReportBySelectedServices(report, selectedSurface.services)
	if len(filtered.Structs) != 2 {
		t.Fatalf("filterAPIReportBySelectedServices() report count = %d, want 2", len(filtered.Structs))
	}
	if filtered.Structs[0].Spec != "Instance" {
		t.Fatalf("filterAPIReportBySelectedServices() kept first spec %q, want %q", filtered.Structs[0].Spec, "Instance")
	}
	if filtered.Structs[1].Spec != "DbSystem" {
		t.Fatalf("filterAPIReportBySelectedServices() kept second spec %q, want %q", filtered.Structs[1].Spec, "DbSystem")
	}
}

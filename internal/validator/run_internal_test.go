package validator

import (
	"testing"

	"github.com/oracle/oci-service-operator/internal/validator/apispec"
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

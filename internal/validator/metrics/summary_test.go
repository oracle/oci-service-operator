package metrics

import (
	"testing"

	"github.com/oracle/oci-service-operator/internal/validator/apispec"
	"github.com/oracle/oci-service-operator/internal/validator/diff"
)

func TestSummarizeAPI(t *testing.T) {
	t.Parallel()

	report := apispec.Report{
		Structs: []apispec.StructReport{
			{
				Service:        "functions",
				Spec:           "FunctionsApplication",
				APISurface:     "spec",
				SDKStruct:      "functions.CreateApplicationDetails",
				TrackingStatus: apispec.TrackingStatusTracked,
				PresentFields: []apispec.FieldReport{
					{FieldName: "CompartmentId", Mandatory: true},
					{FieldName: "DisplayName", Mandatory: false},
				},
				MissingFields: []apispec.FieldReport{
					{FieldName: "Config", Mandatory: true, Status: diff.FieldStatusPotentialGap},
					{FieldName: "SubnetIds", Mandatory: false, Status: diff.FieldStatusFutureConsideration},
				},
				ExtraSpecFields: []apispec.FieldReport{
					{FieldName: "status", Status: diff.FieldStatusUnclassified},
				},
			},
			{
				Service:        "functions",
				Spec:           "FunctionsApplication",
				APISurface:     "status",
				SDKStruct:      "functions.Application",
				TrackingStatus: apispec.TrackingStatusTracked,
				PresentFields: []apispec.FieldReport{
					{FieldName: "Id", Mandatory: false},
				},
				MissingFields: []apispec.FieldReport{
					{FieldName: "LifecycleState", Mandatory: false, Status: diff.FieldStatusFutureConsideration},
				},
				ExtraSpecFields: []apispec.FieldReport{
					{FieldName: "displayName", Status: diff.FieldStatusUnclassified},
				},
			},
			{
				Service:        "core",
				Spec:           "CoreVCN",
				TrackingStatus: apispec.TrackingStatusUntracked,
				TrackingReason: "Generated API spec has no mapped SDK payloads in the validator target registry.",
			},
			{
				Service:        "loadbalancer",
				Spec:           "LoadBalancerShape",
				APISurface:     "excluded",
				SDKStruct:      "loadbalancer.UpdateLoadBalancerShapeDetails",
				TrackingStatus: apispec.TrackingStatusUntracked,
				TrackingReason: "Intentionally untracked: duplicate desired-state payload is already tracked on LoadBalancerLoadBalancerShape.",
			},
		},
	}

	summary := SummarizeAPI(report, 1)

	if summary.Aggregate.Specs != 2 {
		t.Fatalf("Aggregate.Specs = %d, want 2", summary.Aggregate.Specs)
	}
	if summary.Aggregate.Mappings != 3 {
		t.Fatalf("Aggregate.Mappings = %d, want 3", summary.Aggregate.Mappings)
	}
	if summary.Aggregate.TrackedMappings != 2 {
		t.Fatalf("Aggregate.TrackedMappings = %d, want 2", summary.Aggregate.TrackedMappings)
	}
	if summary.Aggregate.UntrackedMappings != 1 {
		t.Fatalf("Aggregate.UntrackedMappings = %d, want 1", summary.Aggregate.UntrackedMappings)
	}
	if summary.Aggregate.PresentFields != 3 {
		t.Fatalf("Aggregate.PresentFields = %d, want 3", summary.Aggregate.PresentFields)
	}
	if summary.Aggregate.MissingFields != 3 {
		t.Fatalf("Aggregate.MissingFields = %d, want 3", summary.Aggregate.MissingFields)
	}
	if summary.Aggregate.MandatoryPresentFields != 1 {
		t.Fatalf("Aggregate.MandatoryPresentFields = %d, want 1", summary.Aggregate.MandatoryPresentFields)
	}
	if summary.Aggregate.MandatoryMissingFields != 1 {
		t.Fatalf("Aggregate.MandatoryMissingFields = %d, want 1", summary.Aggregate.MandatoryMissingFields)
	}
	if summary.Aggregate.ExtraSpecFields != 1 {
		t.Fatalf("Aggregate.ExtraSpecFields = %d, want 1", summary.Aggregate.ExtraSpecFields)
	}
	if summary.Aggregate.OverallCoveragePercent != 50 {
		t.Fatalf("Aggregate.OverallCoveragePercent = %.1f, want 50", summary.Aggregate.OverallCoveragePercent)
	}
	if summary.Aggregate.MandatoryCoveragePercent != 50 {
		t.Fatalf("Aggregate.MandatoryCoveragePercent = %.1f, want 50", summary.Aggregate.MandatoryCoveragePercent)
	}

	if len(summary.Services) != 2 {
		t.Fatalf("len(Services) = %d, want 2", len(summary.Services))
	}
	if summary.Services[0].Service != "core" {
		t.Fatalf("Services[0].Service = %q, want core", summary.Services[0].Service)
	}
	if summary.Services[1].Service != "functions" {
		t.Fatalf("Services[1].Service = %q, want functions", summary.Services[1].Service)
	}
	if summary.Services[1].MissingFields != 3 {
		t.Fatalf("functions MissingFields = %d, want 3", summary.Services[1].MissingFields)
	}

	if len(summary.TopOffenders.MissingFields) != 1 {
		t.Fatalf("len(TopOffenders.MissingFields) = %d, want 1", len(summary.TopOffenders.MissingFields))
	}
	if summary.TopOffenders.MissingFields[0].Spec != "FunctionsApplication" {
		t.Fatalf("Top missing offender spec = %q, want FunctionsApplication", summary.TopOffenders.MissingFields[0].Spec)
	}
	if summary.TopOffenders.MissingFields[0].Count != 2 {
		t.Fatalf("Top missing offender count = %d, want 2", summary.TopOffenders.MissingFields[0].Count)
	}

	if len(summary.TopOffenders.MandatoryMissingFields) != 1 {
		t.Fatalf("len(TopOffenders.MandatoryMissingFields) = %d, want 1", len(summary.TopOffenders.MandatoryMissingFields))
	}
	if summary.TopOffenders.MandatoryMissingFields[0].FieldNames[0] != "Config" {
		t.Fatalf("Top mandatory missing offender first field = %q, want Config", summary.TopOffenders.MandatoryMissingFields[0].FieldNames[0])
	}

	if len(summary.TopOffenders.ExtraSpecFields) != 1 {
		t.Fatalf("len(TopOffenders.ExtraSpecFields) = %d, want 1", len(summary.TopOffenders.ExtraSpecFields))
	}
	if summary.TopOffenders.ExtraSpecFields[0].FieldNames[0] != "status" {
		t.Fatalf("Top extra offender first field = %q, want status", summary.TopOffenders.ExtraSpecFields[0].FieldNames[0])
	}
	if summary.TopOffenders.ExtraSpecFields[0].APISurface != "spec" {
		t.Fatalf("Top extra offender APISurface = %q, want spec", summary.TopOffenders.ExtraSpecFields[0].APISurface)
	}
}

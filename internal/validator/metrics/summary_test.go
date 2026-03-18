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

func TestSummarizeAPIScopeBreakdown(t *testing.T) {
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
				APISurface:     "spec",
				SDKStruct:      "functions.ApplicationSummary",
				TrackingStatus: apispec.TrackingStatusTracked,
				PresentFields: []apispec.FieldReport{
					{FieldName: "DisplayName", Mandatory: false},
				},
				MissingFields: []apispec.FieldReport{
					{FieldName: "LifecycleState", Mandatory: false, Status: diff.FieldStatusFutureConsideration},
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
				Spec:           "CoreConsoleHistoryContent",
				APISurface:     "responseBody",
				SDKStruct:      "core.GetConsoleHistoryContentResponse",
				TrackingStatus: apispec.TrackingStatusTracked,
				PresentFields: []apispec.FieldReport{
					{FieldName: "Value", Mandatory: false},
				},
			},
			{
				Service:        "core",
				Spec:           "CoreVCN",
				APISurface:     "spec",
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

	summary := SummarizeAPI(report, 0)
	if len(summary.ScopeBreakdown) != 4 {
		t.Fatalf("len(ScopeBreakdown) = %d, want 4", len(summary.ScopeBreakdown))
	}

	desiredState := findScopeSummary(t, summary.ScopeBreakdown, "desiredState")
	if desiredState.Aggregate.Specs != 2 {
		t.Fatalf("desiredState Specs = %d, want 2", desiredState.Aggregate.Specs)
	}
	if desiredState.Aggregate.Mappings != 2 {
		t.Fatalf("desiredState Mappings = %d, want 2", desiredState.Aggregate.Mappings)
	}
	if desiredState.Aggregate.TrackedMappings != 1 {
		t.Fatalf("desiredState TrackedMappings = %d, want 1", desiredState.Aggregate.TrackedMappings)
	}
	if desiredState.Aggregate.UntrackedMappings != 1 {
		t.Fatalf("desiredState UntrackedMappings = %d, want 1", desiredState.Aggregate.UntrackedMappings)
	}
	if desiredState.Aggregate.MissingFields != 2 {
		t.Fatalf("desiredState MissingFields = %d, want 2", desiredState.Aggregate.MissingFields)
	}
	if desiredState.Aggregate.ExtraSpecFields != 1 {
		t.Fatalf("desiredState ExtraSpecFields = %d, want 1", desiredState.Aggregate.ExtraSpecFields)
	}
	if len(desiredState.Services) != 2 {
		t.Fatalf("len(desiredState.Services) = %d, want 2", len(desiredState.Services))
	}

	statusParity := findScopeSummary(t, summary.ScopeBreakdown, "statusParity")
	if statusParity.Aggregate.Specs != 1 {
		t.Fatalf("statusParity Specs = %d, want 1", statusParity.Aggregate.Specs)
	}
	if statusParity.Aggregate.Mappings != 1 {
		t.Fatalf("statusParity Mappings = %d, want 1", statusParity.Aggregate.Mappings)
	}
	if statusParity.Aggregate.ExtraSpecFields != 0 {
		t.Fatalf("statusParity ExtraSpecFields = %d, want 0", statusParity.Aggregate.ExtraSpecFields)
	}

	broadening := findScopeSummary(t, summary.ScopeBreakdown, "broadening")
	if broadening.Aggregate.Specs != 1 {
		t.Fatalf("broadening Specs = %d, want 1", broadening.Aggregate.Specs)
	}
	if broadening.Aggregate.Mappings != 1 {
		t.Fatalf("broadening Mappings = %d, want 1", broadening.Aggregate.Mappings)
	}
	if broadening.Aggregate.PresentFields != 1 {
		t.Fatalf("broadening PresentFields = %d, want 1", broadening.Aggregate.PresentFields)
	}
	if broadening.Aggregate.MissingFields != 1 {
		t.Fatalf("broadening MissingFields = %d, want 1", broadening.Aggregate.MissingFields)
	}
	if broadening.Aggregate.ExtraSpecFields != 1 {
		t.Fatalf("broadening ExtraSpecFields = %d, want 1", broadening.Aggregate.ExtraSpecFields)
	}

	responseBody := findScopeSummary(t, summary.ScopeBreakdown, "responseBody")
	if responseBody.Aggregate.Specs != 1 {
		t.Fatalf("responseBody Specs = %d, want 1", responseBody.Aggregate.Specs)
	}
	if responseBody.Aggregate.Mappings != 1 {
		t.Fatalf("responseBody Mappings = %d, want 1", responseBody.Aggregate.Mappings)
	}
	if responseBody.Aggregate.PresentFields != 1 {
		t.Fatalf("responseBody PresentFields = %d, want 1", responseBody.Aggregate.PresentFields)
	}
	if responseBody.Aggregate.MissingFields != 0 {
		t.Fatalf("responseBody MissingFields = %d, want 0", responseBody.Aggregate.MissingFields)
	}
}

func findScopeSummary(t *testing.T, scopes []ScopeSummary, want string) ScopeSummary {
	t.Helper()

	for _, scope := range scopes {
		if scope.Scope == want {
			return scope
		}
	}

	t.Fatalf("scope %q not found in %#v", want, scopes)
	return ScopeSummary{}
}

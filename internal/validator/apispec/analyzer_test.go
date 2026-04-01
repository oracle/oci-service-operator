package apispec

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/internal/validator/allowlist"
	"github.com/oracle/oci-service-operator/internal/validator/sdk"
)

type testWidgetSpec struct {
	Name string `json:"name,omitempty"`
}

func testSDKMappings(names ...string) []SDKMapping {
	mappings := make([]SDKMapping, 0, len(names))
	for _, name := range names {
		mappings = append(mappings, SDKMapping{SDKStruct: name})
	}
	return mappings
}

func TestBuildReportIncludesUntrackedTargets(t *testing.T) {
	originalTargets := targets
	t.Cleanup(func() {
		targets = originalTargets
	})

	targets = []Target{
		{
			Name:        "TestWidget",
			SpecType:    reflect.TypeOf(testWidgetSpec{}),
			SDKMappings: nil,
		},
	}

	report, err := BuildReport(nil, allowlist.Allowlist{})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(report.Structs) != 1 {
		t.Fatalf("BuildReport() report count = %d, want 1", len(report.Structs))
	}

	got := report.Structs[0]
	if got.Spec != "TestWidget" {
		t.Fatalf("report.Structs[0].Spec = %q, want %q", got.Spec, "TestWidget")
	}
	if got.TrackingStatus != TrackingStatusUntracked {
		t.Fatalf("report.Structs[0].TrackingStatus = %q, want %q", got.TrackingStatus, TrackingStatusUntracked)
	}
	if !strings.Contains(got.TrackingReason, "no mapped SDK payloads") {
		t.Fatalf("report.Structs[0].TrackingReason = %q, want unmapped reason", got.TrackingReason)
	}
	if !HasActionable(report) {
		t.Fatal("HasActionable() = false, want true for untracked target")
	}
}

func TestBuildReportMarksReviewedUntrackedTargetsAsIntentional(t *testing.T) {
	originalTargets := targets
	originalReasons := reviewedUntrackedReasons
	t.Cleanup(func() {
		targets = originalTargets
		reviewedUntrackedReasons = originalReasons
	})

	reviewedUntrackedReasons = map[string]string{
		"TestWidget": scalarContentReason("the SDK only returns plain-text widget content"),
	}
	targets = []Target{
		{
			Name:        "TestWidget",
			SpecType:    reflect.TypeOf(struct{}{}),
			StatusType:  reflect.TypeOf(struct{}{}),
			SDKMappings: nil,
		},
	}

	report, err := BuildReport(nil, allowlist.Allowlist{})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(report.Structs) != 1 {
		t.Fatalf("BuildReport() report count = %d, want 1", len(report.Structs))
	}

	got := report.Structs[0]
	if got.TrackingStatus != TrackingStatusUntracked {
		t.Fatalf("report.Structs[0].TrackingStatus = %q, want %q", got.TrackingStatus, TrackingStatusUntracked)
	}
	if !isIntentionalUntrackedReason(got.TrackingReason) {
		t.Fatalf("report.Structs[0].TrackingReason = %q, want intentional untracked reason", got.TrackingReason)
	}
	if HasActionable(report) {
		t.Fatal("HasActionable() = true, want false for reviewed intentional untracked target")
	}
}

func TestBuildReportTracksResponseBodyTargets(t *testing.T) {
	originalTargets := targets
	originalResponseBodyCoverageTargets := responseBodyCoverageTargets
	t.Cleanup(func() {
		targets = originalTargets
		responseBodyCoverageTargets = originalResponseBodyCoverageTargets
	})
	responseBodyCoverageTargets = map[string]responseBodyCoverage{
		"NotificationUnsubscription": {
			SDKStruct: "ons.GetUnsubscriptionResponse",
			FieldName: "Value",
			Encoding:  "plain-text",
		},
		"DNSZoneContent": {
			SDKStruct: "dns.GetZoneContentResponse",
			FieldName: "Content",
			Encoding:  "binary",
		},
	}

	tests := []struct {
		name      string
		target    string
		sdkStruct string
		fieldName string
	}{
		{
			name:      "plain text body",
			target:    "NotificationUnsubscription",
			sdkStruct: "ons.GetUnsubscriptionResponse",
			fieldName: "Value",
		},
		{
			name:      "binary body",
			target:    "DNSZoneContent",
			sdkStruct: "dns.GetZoneContentResponse",
			fieldName: "Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets = []Target{
				{
					Name:        tt.target,
					SpecType:    reflect.TypeOf(struct{}{}),
					StatusType:  reflect.TypeOf(struct{}{}),
					SDKMappings: nil,
				},
			}

			report, err := BuildReport(nil, allowlist.Allowlist{})
			if err != nil {
				t.Fatalf("BuildReport() error = %v", err)
			}
			if len(report.Structs) != 1 {
				t.Fatalf("BuildReport() report count = %d, want 1", len(report.Structs))
			}

			got := report.Structs[0]
			if got.TrackingStatus != TrackingStatusTracked {
				t.Fatalf("report.Structs[0].TrackingStatus = %q, want %q", got.TrackingStatus, TrackingStatusTracked)
			}
			if got.APISurface != apiSurfaceResponseBody {
				t.Fatalf("report.Structs[0].APISurface = %q, want %q", got.APISurface, apiSurfaceResponseBody)
			}
			if got.SDKStruct != tt.sdkStruct {
				t.Fatalf("report.Structs[0].SDKStruct = %q, want %q", got.SDKStruct, tt.sdkStruct)
			}
			if len(got.PresentFields) != 1 || got.PresentFields[0].FieldName != tt.fieldName {
				t.Fatalf("report.Structs[0].PresentFields = %#v, want %q present", got.PresentFields, tt.fieldName)
			}
			if len(got.MissingFields) != 0 {
				t.Fatalf("report.Structs[0].MissingFields = %#v, want none", got.MissingFields)
			}
			if len(got.ExtraSpecFields) != 0 {
				t.Fatalf("report.Structs[0].ExtraSpecFields = %#v, want none", got.ExtraSpecFields)
			}
			if HasActionable(report) {
				t.Fatal("HasActionable() = true, want false for response-body-covered target")
			}
		})
	}
}

func TestBuildReportHonorsExplicitMappingSurfaceOverrides(t *testing.T) {
	originalTargets := targets
	t.Cleanup(func() {
		targets = originalTargets
	})

	type explicitSurfaceStatus struct {
		OsokStatus testStatusMarker `json:"status,omitempty"`
		Name       string           `json:"name,omitempty"`
	}

	targets = []Target{
		{
			Name:       "CoreInstance",
			SpecType:   reflect.TypeOf(testWidgetSpec{}),
			StatusType: reflect.TypeOf(explicitSurfaceStatus{}),
			SDKMappings: []SDKMapping{
				{
					SDKStruct:  "example.CreateWidgetDetails",
					APISurface: apiSurfaceStatus,
				},
			},
		},
	}

	report, err := BuildReport([]sdk.SDKStruct{
		{
			QualifiedName: "example.CreateWidgetDetails",
			Fields: []sdk.SDKField{
				{Name: "Name", JSONName: "name"},
			},
		},
	}, allowlist.Allowlist{})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(report.Structs) != 1 {
		t.Fatalf("BuildReport() report count = %d, want 1", len(report.Structs))
	}

	got := report.Structs[0]
	if got.APISurface != apiSurfaceStatus {
		t.Fatalf("report.Structs[0].APISurface = %q, want %q", got.APISurface, apiSurfaceStatus)
	}
	if len(got.PresentFields) != 1 || got.PresentFields[0].FieldName != "Name" {
		t.Fatalf("report.Structs[0].PresentFields = %#v, want Name present on explicit status surface", got.PresentFields)
	}
}

func TestBuildReportMarksExplicitlyExcludedMappingsAsIntentional(t *testing.T) {
	originalTargets := targets
	t.Cleanup(func() {
		targets = originalTargets
	})

	targets = []Target{
		{
			Name:       "LoadBalancerShape",
			SpecType:   reflect.TypeOf(struct{}{}),
			StatusType: reflect.TypeOf(struct{}{}),
			SDKMappings: []SDKMapping{
				{
					SDKStruct: "loadbalancer.UpdateLoadBalancerShapeDetails",
					Exclude:   true,
					Reason:    "Intentionally untracked: duplicate desired-state payload is already tracked on LoadBalancerLoadBalancerShape.",
				},
			},
		},
	}

	report, err := BuildReport(nil, allowlist.Allowlist{})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(report.Structs) != 1 {
		t.Fatalf("BuildReport() report count = %d, want 1", len(report.Structs))
	}

	got := report.Structs[0]
	if got.TrackingStatus != TrackingStatusUntracked {
		t.Fatalf("report.Structs[0].TrackingStatus = %q, want %q", got.TrackingStatus, TrackingStatusUntracked)
	}
	if got.APISurface != apiSurfaceExcluded {
		t.Fatalf("report.Structs[0].APISurface = %q, want %q", got.APISurface, apiSurfaceExcluded)
	}
	if !isIntentionalUntrackedReason(got.TrackingReason) {
		t.Fatalf("report.Structs[0].TrackingReason = %q, want intentional untracked reason", got.TrackingReason)
	}
	if HasActionable(report) {
		t.Fatal("HasActionable() = true, want false for explicitly excluded mapping")
	}
}

type testStatusMarker struct{}

type testWidgetStatus struct {
	OsokStatus  testStatusMarker `json:"status,omitempty"`
	DisplayName string           `json:"displayName,omitempty"`
}

type testEscapedStatusField struct {
	OsokStatus testStatusMarker `json:"status,omitempty"`
	Status     string           `json:"sdkStatus,omitempty"`
}

func TestBuildReportUsesStatusSurfaceForStatusTargets(t *testing.T) {
	originalTargets := targets
	t.Cleanup(func() {
		targets = originalTargets
	})

	targets = []Target{
		{
			Name:        "TestReadOnlyWidget",
			SpecType:    reflect.TypeOf(testWidgetSpec{}),
			StatusType:  reflect.TypeOf(testWidgetStatus{}),
			SDKMappings: testSDKMappings("example.Widget"),
		},
	}

	report, err := BuildReport([]sdk.SDKStruct{
		{
			QualifiedName: "example.Widget",
			Fields: []sdk.SDKField{
				{Name: "DisplayName", JSONName: "displayName"},
			},
		},
	}, allowlist.Allowlist{})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(report.Structs) != 1 {
		t.Fatalf("BuildReport() report count = %d, want 1", len(report.Structs))
	}

	got := report.Structs[0]
	if got.APISurface != apiSurfaceStatus {
		t.Fatalf("report.Structs[0].APISurface = %q, want %q", got.APISurface, apiSurfaceStatus)
	}
	if len(got.PresentFields) != 1 || got.PresentFields[0].FieldName != "DisplayName" {
		t.Fatalf("report.Structs[0].PresentFields = %#v, want DisplayName present", got.PresentFields)
	}
	if len(got.ExtraSpecFields) != 0 {
		t.Fatalf("report.Structs[0].ExtraSpecFields = %#v, want OsokStatus skipped", got.ExtraSpecFields)
	}
	if HasActionable(report) {
		t.Fatal("HasActionable() = true, want false for covered status target")
	}
}

func TestBuildReportMatchesEscapedStatusFieldByGoName(t *testing.T) {
	originalTargets := targets
	t.Cleanup(func() {
		targets = originalTargets
	})

	targets = []Target{
		{
			Name:        "TestWorkRequest",
			SpecType:    reflect.TypeOf(testWidgetSpec{}),
			StatusType:  reflect.TypeOf(testEscapedStatusField{}),
			SDKMappings: testSDKMappings("example.WorkRequest"),
		},
	}

	report, err := BuildReport([]sdk.SDKStruct{
		{
			QualifiedName: "example.WorkRequest",
			Fields: []sdk.SDKField{
				{Name: "Status", JSONName: "status"},
			},
		},
	}, allowlist.Allowlist{})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(report.Structs) != 1 {
		t.Fatalf("BuildReport() report count = %d, want 1", len(report.Structs))
	}

	got := report.Structs[0]
	if got.APISurface != apiSurfaceStatus {
		t.Fatalf("report.Structs[0].APISurface = %q, want %q", got.APISurface, apiSurfaceStatus)
	}
	if len(got.PresentFields) != 1 || got.PresentFields[0].FieldName != "Status" {
		t.Fatalf("report.Structs[0].PresentFields = %#v, want Status present", got.PresentFields)
	}
	if len(got.MissingFields) != 0 {
		t.Fatalf("report.Structs[0].MissingFields = %#v, want none", got.MissingFields)
	}
	if len(got.ExtraSpecFields) != 0 {
		t.Fatalf("report.Structs[0].ExtraSpecFields = %#v, want none", got.ExtraSpecFields)
	}
}

func TestBuildReportRoutesDesiredAndObservedSDKStructsToDifferentSurfaces(t *testing.T) {
	originalTargets := targets
	t.Cleanup(func() {
		targets = originalTargets
	})

	targets = []Target{
		{
			Name:        "TestWidget",
			SpecType:    reflect.TypeOf(testWidgetSpec{}),
			StatusType:  reflect.TypeOf(testWidgetStatus{}),
			SDKMappings: testSDKMappings("example.CreateWidgetDetails", "example.Widget"),
		},
	}

	report, err := BuildReport([]sdk.SDKStruct{
		{
			QualifiedName: "example.CreateWidgetDetails",
			Fields: []sdk.SDKField{
				{Name: "Name", JSONName: "name"},
			},
		},
		{
			QualifiedName: "example.Widget",
			Fields: []sdk.SDKField{
				{Name: "DisplayName", JSONName: "displayName"},
			},
		},
	}, allowlist.Allowlist{})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(report.Structs) != 2 {
		t.Fatalf("BuildReport() report count = %d, want 2", len(report.Structs))
	}

	bySDK := make(map[string]StructReport, len(report.Structs))
	for _, structReport := range report.Structs {
		bySDK[structReport.SDKStruct] = structReport
	}

	create := bySDK["example.CreateWidgetDetails"]
	if create.APISurface != apiSurfaceSpec {
		t.Fatalf("CreateWidgetDetails APISurface = %q, want %q", create.APISurface, apiSurfaceSpec)
	}
	if len(create.PresentFields) != 1 || create.PresentFields[0].FieldName != "Name" {
		t.Fatalf("CreateWidgetDetails PresentFields = %#v, want Name present", create.PresentFields)
	}

	observed := bySDK["example.Widget"]
	if observed.APISurface != apiSurfaceStatus {
		t.Fatalf("Widget APISurface = %q, want %q", observed.APISurface, apiSurfaceStatus)
	}
	if len(observed.PresentFields) != 1 || observed.PresentFields[0].FieldName != "DisplayName" {
		t.Fatalf("Widget PresentFields = %#v, want DisplayName present", observed.PresentFields)
	}
}

func TestEmptyRegistryTargetsHaveSpecialHandling(t *testing.T) {
	t.Parallel()

	var got []string
	for _, target := range Targets() {
		if len(target.SDKMappings) != 0 {
			continue
		}
		reason := reviewedUntrackedReason(target.Name)
		_, hasResponseBodyCoverage := responseBodyCoverageForTarget(target.Name)
		if !isIntentionalUntrackedReason(reason) && !hasResponseBodyCoverage {
			got = append(got, target.Name)
		}
	}

	if len(got) != 0 {
		t.Fatalf("empty registry targets without reviewed handling: %v", got)
	}

	var extra []string
	for targetName := range reviewedUntrackedReasons {
		matched := false
		for _, target := range Targets() {
			if target.Name == targetName && len(target.SDKMappings) == 0 {
				matched = true
				break
			}
		}
		if !matched {
			extra = append(extra, targetName)
		}
	}
	slices.Sort(extra)
	if len(extra) != 0 {
		t.Fatalf("reviewed untracked reasons without matching empty registry targets: %v", extra)
	}

	extra = extra[:0]
	for targetName := range responseBodyCoverageTargets {
		matched := false
		for _, target := range Targets() {
			if target.Name == targetName && len(target.SDKMappings) == 0 {
				matched = true
				break
			}
		}
		if !matched {
			extra = append(extra, targetName)
		}
	}
	slices.Sort(extra)
	if len(extra) != 0 {
		t.Fatalf("response-body coverage entries without matching empty registry targets: %v", extra)
	}
}

package dashboardgroup

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	dashboardservicesdk "github.com/oracle/oci-go-sdk/v65/dashboardservice"
	dashboardservicev1beta1 "github.com/oracle/oci-service-operator/api/dashboardservice/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type stubDashboardGroupOCIClient struct {
	create func(context.Context, dashboardservicesdk.CreateDashboardGroupRequest) (dashboardservicesdk.CreateDashboardGroupResponse, error)
	get    func(context.Context, dashboardservicesdk.GetDashboardGroupRequest) (dashboardservicesdk.GetDashboardGroupResponse, error)
	list   func(context.Context, dashboardservicesdk.ListDashboardGroupsRequest) (dashboardservicesdk.ListDashboardGroupsResponse, error)
	update func(context.Context, dashboardservicesdk.UpdateDashboardGroupRequest) (dashboardservicesdk.UpdateDashboardGroupResponse, error)
	delete func(context.Context, dashboardservicesdk.DeleteDashboardGroupRequest) (dashboardservicesdk.DeleteDashboardGroupResponse, error)
}

func (s stubDashboardGroupOCIClient) CreateDashboardGroup(
	ctx context.Context,
	req dashboardservicesdk.CreateDashboardGroupRequest,
) (dashboardservicesdk.CreateDashboardGroupResponse, error) {
	if s.create == nil {
		return dashboardservicesdk.CreateDashboardGroupResponse{}, nil
	}
	return s.create(ctx, req)
}

func (s stubDashboardGroupOCIClient) GetDashboardGroup(
	ctx context.Context,
	req dashboardservicesdk.GetDashboardGroupRequest,
) (dashboardservicesdk.GetDashboardGroupResponse, error) {
	if s.get == nil {
		return dashboardservicesdk.GetDashboardGroupResponse{}, nil
	}
	return s.get(ctx, req)
}

func (s stubDashboardGroupOCIClient) ListDashboardGroups(
	ctx context.Context,
	req dashboardservicesdk.ListDashboardGroupsRequest,
) (dashboardservicesdk.ListDashboardGroupsResponse, error) {
	if s.list == nil {
		return dashboardservicesdk.ListDashboardGroupsResponse{}, nil
	}
	return s.list(ctx, req)
}

func (s stubDashboardGroupOCIClient) UpdateDashboardGroup(
	ctx context.Context,
	req dashboardservicesdk.UpdateDashboardGroupRequest,
) (dashboardservicesdk.UpdateDashboardGroupResponse, error) {
	if s.update == nil {
		return dashboardservicesdk.UpdateDashboardGroupResponse{}, nil
	}
	return s.update(ctx, req)
}

func (s stubDashboardGroupOCIClient) DeleteDashboardGroup(
	ctx context.Context,
	req dashboardservicesdk.DeleteDashboardGroupRequest,
) (dashboardservicesdk.DeleteDashboardGroupResponse, error) {
	if s.delete == nil {
		return dashboardservicesdk.DeleteDashboardGroupResponse{}, nil
	}
	return s.delete(ctx, req)
}

func TestApplyDashboardGroupRuntimeHooksClearsBootstrapFormalGaps(t *testing.T) {
	t.Parallel()

	hooks := newDashboardGroupDefaultRuntimeHooks(dashboardservicesdk.DashboardGroupClient{})
	applyDashboardGroupRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if len(hooks.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("hooks.Semantics.AuxiliaryOperations = %#v, want none", hooks.Semantics.AuxiliaryOperations)
	}
	if len(hooks.Semantics.Unsupported) != 0 {
		t.Fatalf("hooks.Semantics.Unsupported = %#v, want none", hooks.Semantics.Unsupported)
	}
	if hooks.Semantics.List == nil {
		t.Fatal("hooks.Semantics.List = nil, want reviewed list semantics")
	}
	if !slices.Equal(hooks.Semantics.List.MatchFields, []string{"compartmentId", "displayName"}) {
		t.Fatalf("hooks.Semantics.List.MatchFields = %#v, want [compartmentId displayName]", hooks.Semantics.List.MatchFields)
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want reviewed guard")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}
	if len(hooks.List.Fields) != 2 {
		t.Fatalf("hooks.List.Fields length = %d, want 2", len(hooks.List.Fields))
	}
	if hooks.List.Fields[0].FieldName != "CompartmentId" || hooks.List.Fields[1].FieldName != "DisplayName" {
		t.Fatalf("hooks.List.Fields = %#v, want compartmentId/displayName lookup fields", hooks.List.Fields)
	}
}

func TestGuardDashboardGroupExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource *dashboardservicev1beta1.DashboardGroup
		want     generatedruntime.ExistingBeforeCreateDecision
		wantErr  string
	}{
		{
			name:    "nil resource fails",
			want:    generatedruntime.ExistingBeforeCreateDecisionFail,
			wantErr: "resource is nil",
		},
		{
			name: "missing compartment fails",
			resource: &dashboardservicev1beta1.DashboardGroup{
				Spec: dashboardservicev1beta1.DashboardGroupSpec{
					DisplayName: "dashboards",
				},
			},
			want:    generatedruntime.ExistingBeforeCreateDecisionFail,
			wantErr: "spec.compartmentId is required",
		},
		{
			name: "missing display name skips reuse",
			resource: &dashboardservicev1beta1.DashboardGroup{
				Spec: dashboardservicev1beta1.DashboardGroupSpec{
					CompartmentId: "ocid1.compartment.oc1..example",
				},
			},
			want: generatedruntime.ExistingBeforeCreateDecisionSkip,
		},
		{
			name:     "compartment and display name allow reuse",
			resource: newDashboardGroupTestResource(),
			want:     generatedruntime.ExistingBeforeCreateDecisionAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := guardDashboardGroupExistingBeforeCreate(context.Background(), tt.resource)
			if got != tt.want {
				t.Fatalf("guardDashboardGroupExistingBeforeCreate() = %q, want %q", got, tt.want)
			}
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("guardDashboardGroupExistingBeforeCreate() error = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("guardDashboardGroupExistingBeforeCreate() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestBuildDashboardGroupUpdateBodySupportsExplicitClears(t *testing.T) {
	t.Parallel()

	desired := newDashboardGroupTestResource()
	desired.Spec.DisplayName = "dashboards-updated"
	desired.Spec.Description = ""
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	current := observedDashboardGroupFromSpec(
		"ocid1.dashboardgroup.oc1..existing",
		newDashboardGroupTestResource().Spec,
		"ACTIVE",
	)

	details, updateNeeded, err := buildDashboardGroupUpdateBody(desired, current)
	if err != nil {
		t.Fatalf("buildDashboardGroupUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildDashboardGroupUpdateBody() updateNeeded = false, want true")
	}
	if details.DisplayName == nil || *details.DisplayName != desired.Spec.DisplayName {
		t.Fatalf("details.DisplayName = %#v, want %q", details.DisplayName, desired.Spec.DisplayName)
	}
	if details.Description == nil || *details.Description != "" {
		t.Fatalf("details.Description = %#v, want explicit empty string", details.Description)
	}
	if details.FreeformTags == nil || len(details.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want explicit empty map", details.FreeformTags)
	}
	if details.DefinedTags == nil || len(details.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want explicit empty map", details.DefinedTags)
	}

	body := dashboardGroupSerializedRequestBody(t, dashboardservicesdk.UpdateDashboardGroupRequest{
		DashboardGroupId:            common.String("ocid1.dashboardgroup.oc1..existing"),
		UpdateDashboardGroupDetails: details,
	}, http.MethodPut, "/dashboardGroups/ocid1.dashboardgroup.oc1..existing")
	for _, want := range []string{
		`"displayName":"dashboards-updated"`,
		`"description":""`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestNewDashboardGroupServiceClientWithOCIClientUsesScopedExactNameReuse(t *testing.T) {
	t.Parallel()

	resource := newDashboardGroupTestResource()
	existingID := "ocid1.dashboardgroup.oc1..existing"

	var listRequest dashboardservicesdk.ListDashboardGroupsRequest
	var getRequest dashboardservicesdk.GetDashboardGroupRequest
	createCalled := false

	client := newDashboardGroupServiceClientWithOCIClient(
		loggerutil.OSOKLogger{},
		stubDashboardGroupOCIClient{
			create: func(context.Context, dashboardservicesdk.CreateDashboardGroupRequest) (dashboardservicesdk.CreateDashboardGroupResponse, error) {
				createCalled = true
				return dashboardservicesdk.CreateDashboardGroupResponse{}, nil
			},
			list: func(_ context.Context, req dashboardservicesdk.ListDashboardGroupsRequest) (dashboardservicesdk.ListDashboardGroupsResponse, error) {
				listRequest = req
				return dashboardservicesdk.ListDashboardGroupsResponse{
					DashboardGroupCollection: dashboardservicesdk.DashboardGroupCollection{
						Items: []dashboardservicesdk.DashboardGroupSummary{
							{
								Id:             common.String(existingID),
								DisplayName:    common.String(resource.Spec.DisplayName),
								Description:    common.String(resource.Spec.Description),
								CompartmentId:  common.String(resource.Spec.CompartmentId),
								LifecycleState: dashboardservicesdk.DashboardGroupLifecycleStateActive,
								FreeformTags:   map[string]string{"team": "dash"},
								DefinedTags: map[string]map[string]interface{}{
									"Operations": {"CostCenter": "42"},
								},
							},
						},
					},
				}, nil
			},
			get: func(_ context.Context, req dashboardservicesdk.GetDashboardGroupRequest) (dashboardservicesdk.GetDashboardGroupResponse, error) {
				getRequest = req
				return dashboardservicesdk.GetDashboardGroupResponse{
					DashboardGroup: observedDashboardGroupFromSpec(existingID, resource.Spec, "ACTIVE"),
				}, nil
			},
		},
	)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateOrUpdate() invoked CreateDashboardGroup, want existing-before-create reuse")
	}
	if listRequest.CompartmentId == nil || *listRequest.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("ListDashboardGroupsRequest.CompartmentId = %#v, want %q", listRequest.CompartmentId, resource.Spec.CompartmentId)
	}
	if listRequest.DisplayName == nil || *listRequest.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("ListDashboardGroupsRequest.DisplayName = %#v, want %q", listRequest.DisplayName, resource.Spec.DisplayName)
	}
	if getRequest.DashboardGroupId == nil || *getRequest.DashboardGroupId != existingID {
		t.Fatalf("GetDashboardGroupRequest.DashboardGroupId = %#v, want %q", getRequest.DashboardGroupId, existingID)
	}
}

func newDashboardGroupTestResource() *dashboardservicev1beta1.DashboardGroup {
	return &dashboardservicev1beta1.DashboardGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dashboard-group-sample",
			Namespace: "default",
		},
		Spec: dashboardservicev1beta1.DashboardGroupSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "dashboards",
			Description:   "dashboard group",
			FreeformTags: map[string]string{
				"team": "dash",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func observedDashboardGroupFromSpec(
	id string,
	spec dashboardservicev1beta1.DashboardGroupSpec,
	lifecycle string,
) dashboardservicesdk.DashboardGroup {
	return dashboardservicesdk.DashboardGroup{
		Id:             common.String(id),
		DisplayName:    common.String(spec.DisplayName),
		Description:    common.String(spec.Description),
		CompartmentId:  common.String(spec.CompartmentId),
		LifecycleState: dashboardservicesdk.DashboardGroupLifecycleStateEnum(lifecycle),
		FreeformTags:   spec.FreeformTags,
		DefinedTags:    dashboardGroupDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func dashboardGroupSerializedRequestBody(
	t *testing.T,
	request dashboardservicesdk.UpdateDashboardGroupRequest,
	method string,
	path string,
) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}
	defer httpRequest.Body.Close()

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	return string(body)
}

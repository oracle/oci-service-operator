package dashboard

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

type stubDashboardOCIClient struct {
	create func(context.Context, dashboardservicesdk.CreateDashboardRequest) (dashboardservicesdk.CreateDashboardResponse, error)
	get    func(context.Context, dashboardservicesdk.GetDashboardRequest) (dashboardservicesdk.GetDashboardResponse, error)
	list   func(context.Context, dashboardservicesdk.ListDashboardsRequest) (dashboardservicesdk.ListDashboardsResponse, error)
	update func(context.Context, dashboardservicesdk.UpdateDashboardRequest) (dashboardservicesdk.UpdateDashboardResponse, error)
	delete func(context.Context, dashboardservicesdk.DeleteDashboardRequest) (dashboardservicesdk.DeleteDashboardResponse, error)
}

func (s stubDashboardOCIClient) CreateDashboard(
	ctx context.Context,
	req dashboardservicesdk.CreateDashboardRequest,
) (dashboardservicesdk.CreateDashboardResponse, error) {
	if s.create == nil {
		return dashboardservicesdk.CreateDashboardResponse{}, nil
	}
	return s.create(ctx, req)
}

func (s stubDashboardOCIClient) GetDashboard(
	ctx context.Context,
	req dashboardservicesdk.GetDashboardRequest,
) (dashboardservicesdk.GetDashboardResponse, error) {
	if s.get == nil {
		return dashboardservicesdk.GetDashboardResponse{}, nil
	}
	return s.get(ctx, req)
}

func (s stubDashboardOCIClient) ListDashboards(
	ctx context.Context,
	req dashboardservicesdk.ListDashboardsRequest,
) (dashboardservicesdk.ListDashboardsResponse, error) {
	if s.list == nil {
		return dashboardservicesdk.ListDashboardsResponse{}, nil
	}
	return s.list(ctx, req)
}

func (s stubDashboardOCIClient) UpdateDashboard(
	ctx context.Context,
	req dashboardservicesdk.UpdateDashboardRequest,
) (dashboardservicesdk.UpdateDashboardResponse, error) {
	if s.update == nil {
		return dashboardservicesdk.UpdateDashboardResponse{}, nil
	}
	return s.update(ctx, req)
}

func (s stubDashboardOCIClient) DeleteDashboard(
	ctx context.Context,
	req dashboardservicesdk.DeleteDashboardRequest,
) (dashboardservicesdk.DeleteDashboardResponse, error) {
	if s.delete == nil {
		return dashboardservicesdk.DeleteDashboardResponse{}, nil
	}
	return s.delete(ctx, req)
}

func TestApplyDashboardRuntimeHooksPublishesReviewedSemantics(t *testing.T) {
	t.Parallel()

	hooks := newDashboardDefaultRuntimeHooks(dashboardservicesdk.DashboardClient{})
	applyDashboardRuntimeHooks(&hooks)

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
	if !slices.Equal(hooks.Semantics.List.MatchFields, []string{"dashboardGroupId", "displayName"}) {
		t.Fatalf("hooks.Semantics.List.MatchFields = %#v, want [dashboardGroupId displayName]", hooks.Semantics.List.MatchFields)
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want reviewed guard")
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want reviewed create builder")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}
	if len(hooks.List.Fields) != 2 {
		t.Fatalf("hooks.List.Fields length = %d, want 2", len(hooks.List.Fields))
	}
	if hooks.List.Fields[0].FieldName != "DashboardGroupId" || hooks.List.Fields[1].FieldName != "DisplayName" {
		t.Fatalf("hooks.List.Fields = %#v, want dashboardGroupId/displayName lookup fields", hooks.List.Fields)
	}
}

func TestDashboardDesiredStringUpdateDoesNotTreatEmptyAsClear(t *testing.T) {
	t.Parallel()

	desired, ok := dashboardDesiredStringUpdate("", common.String("existing"))
	if ok {
		t.Fatal("dashboardDesiredStringUpdate() ok = true, want false for ambiguous empty string")
	}
	if desired != nil {
		t.Fatalf("dashboardDesiredStringUpdate() desired = %#v, want nil", desired)
	}
}

func TestGuardDashboardExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource *dashboardservicev1beta1.Dashboard
		want     generatedruntime.ExistingBeforeCreateDecision
		wantErr  string
	}{
		{
			name:    "nil resource fails",
			want:    generatedruntime.ExistingBeforeCreateDecisionFail,
			wantErr: "resource is nil",
		},
		{
			name: "missing dashboard group fails",
			resource: &dashboardservicev1beta1.Dashboard{
				Spec: dashboardservicev1beta1.DashboardSpec{
					DisplayName: "ops-dashboard",
				},
			},
			want:    generatedruntime.ExistingBeforeCreateDecisionFail,
			wantErr: "spec.dashboardGroupId is required",
		},
		{
			name: "missing display name skips reuse",
			resource: &dashboardservicev1beta1.Dashboard{
				Spec: dashboardservicev1beta1.DashboardSpec{
					DashboardGroupId: "ocid1.dashboardgroup.oc1..example",
				},
			},
			want: generatedruntime.ExistingBeforeCreateDecisionSkip,
		},
		{
			name:     "group and display name allow reuse",
			resource: newDashboardTestResource(),
			want:     generatedruntime.ExistingBeforeCreateDecisionAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := guardDashboardExistingBeforeCreate(context.Background(), tt.resource)
			if got != tt.want {
				t.Fatalf("guardDashboardExistingBeforeCreate() = %q, want %q", got, tt.want)
			}
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("guardDashboardExistingBeforeCreate() error = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("guardDashboardExistingBeforeCreate() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestBuildDashboardCreateBodyDefaultsV1(t *testing.T) {
	t.Parallel()

	resource := newDashboardTestResource()
	resource.Spec.SchemaVersion = ""

	body, err := buildDashboardCreateBody(resource)
	if err != nil {
		t.Fatalf("buildDashboardCreateBody() error = %v", err)
	}

	details, ok := body.(dashboardservicesdk.CreateV1DashboardDetails)
	if !ok {
		t.Fatalf("buildDashboardCreateBody() type = %T, want %T", body, dashboardservicesdk.CreateV1DashboardDetails{})
	}
	if details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("details.DisplayName = %#v, want %q", details.DisplayName, resource.Spec.DisplayName)
	}
	if len(details.Widgets) != 1 {
		t.Fatalf("details.Widgets len = %d, want 1", len(details.Widgets))
	}

	requestBody := dashboardSerializedCreateRequestBody(t, dashboardservicesdk.CreateDashboardRequest{
		CreateDashboardDetails: details,
	})
	for _, want := range []string{
		`"schemaVersion":"V1"`,
		`"dashboardGroupId":"ocid1.dashboardgroup.oc1..example"`,
		`"displayName":"ops-dashboard"`,
	} {
		if !strings.Contains(requestBody, want) {
			t.Fatalf("request body %s does not contain %s", requestBody, want)
		}
	}
}

func TestBuildDashboardUpdateBodySkipsAmbiguousStringAndNullClears(t *testing.T) {
	t.Parallel()

	desired := newDashboardTestResource()
	desired.Spec.DisplayName = "ops-dashboard-updated"
	desired.Spec.Description = ""
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}
	desired.Spec.Config = dashboardJSONValue(`{"layout":"single"}`)
	desired.Spec.Widgets = []shared.JSONValue{}

	current := observedDashboardFromSpec(
		"ocid1.dashboard.oc1..existing",
		newDashboardTestResource().Spec,
		"ACTIVE",
	)

	body, updateNeeded, err := buildDashboardUpdateBody(desired, current)
	if err != nil {
		t.Fatalf("buildDashboardUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildDashboardUpdateBody() updateNeeded = false, want true")
	}

	details, ok := body.(dashboardservicesdk.UpdateV1DashboardDetails)
	if !ok {
		t.Fatalf("buildDashboardUpdateBody() type = %T, want %T", body, dashboardservicesdk.UpdateV1DashboardDetails{})
	}
	if details.DisplayName == nil || *details.DisplayName != desired.Spec.DisplayName {
		t.Fatalf("details.DisplayName = %#v, want %q", details.DisplayName, desired.Spec.DisplayName)
	}
	if details.Description != nil {
		t.Fatalf("details.Description = %#v, want nil because empty string is treated as omission", details.Description)
	}
	if details.FreeformTags == nil || len(details.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want explicit empty map", details.FreeformTags)
	}
	if details.DefinedTags == nil || len(details.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want explicit empty map", details.DefinedTags)
	}
	if details.Config == nil {
		t.Fatal("details.Config = nil, want updated config payload")
	}
	if len(details.Widgets) != 0 {
		t.Fatalf("details.Widgets = %#v, want explicit empty slice", details.Widgets)
	}

	requestBody := dashboardSerializedUpdateRequestBody(t, dashboardservicesdk.UpdateDashboardRequest{
		DashboardId:            common.String("ocid1.dashboard.oc1..existing"),
		UpdateDashboardDetails: details,
	})
	for _, want := range []string{
		`"schemaVersion":"V1"`,
		`"displayName":"ops-dashboard-updated"`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
		`"config":{"layout":"single"}`,
		`"widgets":[]`,
	} {
		if !strings.Contains(requestBody, want) {
			t.Fatalf("request body %s does not contain %s", requestBody, want)
		}
	}
	if strings.Contains(requestBody, `"description":`) {
		t.Fatalf("request body %s contains description clear, want description omitted", requestBody)
	}
}

func TestNewDashboardServiceClientWithOCIClientUsesScopedExactNameReuse(t *testing.T) {
	t.Parallel()

	resource := newDashboardTestResource()
	existingID := "ocid1.dashboard.oc1..existing"

	var listRequest dashboardservicesdk.ListDashboardsRequest
	var getRequest dashboardservicesdk.GetDashboardRequest
	createCalled := false

	client := newDashboardServiceClientWithOCIClient(
		loggerutil.OSOKLogger{},
		stubDashboardOCIClient{
			create: func(context.Context, dashboardservicesdk.CreateDashboardRequest) (dashboardservicesdk.CreateDashboardResponse, error) {
				createCalled = true
				return dashboardservicesdk.CreateDashboardResponse{}, nil
			},
			list: func(_ context.Context, req dashboardservicesdk.ListDashboardsRequest) (dashboardservicesdk.ListDashboardsResponse, error) {
				listRequest = req
				return dashboardservicesdk.ListDashboardsResponse{
					DashboardCollection: dashboardservicesdk.DashboardCollection{
						Items: []dashboardservicesdk.DashboardSummary{
							{
								Id:               common.String(existingID),
								DashboardGroupId: common.String(resource.Spec.DashboardGroupId),
								DisplayName:      common.String(resource.Spec.DisplayName),
								Description:      common.String(resource.Spec.Description),
								CompartmentId:    common.String("ocid1.compartment.oc1..example"),
								LifecycleState:   dashboardservicesdk.DashboardLifecycleStateActive,
								FreeformTags:     map[string]string{"team": "dash"},
								DefinedTags: map[string]map[string]interface{}{
									"Operations": {"CostCenter": "42"},
								},
							},
						},
					},
				}, nil
			},
			get: func(_ context.Context, req dashboardservicesdk.GetDashboardRequest) (dashboardservicesdk.GetDashboardResponse, error) {
				getRequest = req
				return dashboardservicesdk.GetDashboardResponse{
					Dashboard: observedDashboardFromSpec(existingID, resource.Spec, "ACTIVE"),
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
		t.Fatal("CreateOrUpdate() invoked CreateDashboard, want existing-before-create reuse")
	}
	if listRequest.DashboardGroupId == nil || *listRequest.DashboardGroupId != resource.Spec.DashboardGroupId {
		t.Fatalf("ListDashboardsRequest.DashboardGroupId = %#v, want %q", listRequest.DashboardGroupId, resource.Spec.DashboardGroupId)
	}
	if listRequest.DisplayName == nil || *listRequest.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("ListDashboardsRequest.DisplayName = %#v, want %q", listRequest.DisplayName, resource.Spec.DisplayName)
	}
	if getRequest.DashboardId == nil || *getRequest.DashboardId != existingID {
		t.Fatalf("GetDashboardRequest.DashboardId = %#v, want %q", getRequest.DashboardId, existingID)
	}
}

func newDashboardTestResource() *dashboardservicev1beta1.Dashboard {
	return &dashboardservicev1beta1.Dashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dashboard-sample",
			Namespace: "default",
		},
		Spec: dashboardservicev1beta1.DashboardSpec{
			DashboardGroupId: "ocid1.dashboardgroup.oc1..example",
			SchemaVersion:    "V1",
			DisplayName:      "ops-dashboard",
			Description:      "operations dashboard",
			FreeformTags: map[string]string{
				"team": "dash",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			Config: dashboardJSONValue(`{"layout":"grid"}`),
			Widgets: []shared.JSONValue{
				dashboardJSONValue(`{"name":"cpu"}`),
			},
		},
	}
}

func observedDashboardFromSpec(
	id string,
	spec dashboardservicev1beta1.DashboardSpec,
	lifecycle string,
) dashboardservicesdk.Dashboard {
	widgets, _, err := dashboardWidgetsFromSpec(spec.Widgets)
	if err != nil {
		panic(err)
	}
	config, _, err := dashboardConfigFromSpec(spec.Config)
	if err != nil {
		panic(err)
	}

	return dashboardservicesdk.V1Dashboard{
		Id:               common.String(id),
		DashboardGroupId: common.String(spec.DashboardGroupId),
		DisplayName:      common.String(spec.DisplayName),
		Description:      common.String(spec.Description),
		CompartmentId:    common.String("ocid1.compartment.oc1..example"),
		LifecycleState:   dashboardservicesdk.DashboardLifecycleStateEnum(lifecycle),
		FreeformTags:     spec.FreeformTags,
		DefinedTags:      dashboardDefinedTagsFromSpec(spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {"free-tier-retained": "true"},
		},
		Widgets: widgets,
		Config:  config,
	}
}

func dashboardJSONValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
}

func dashboardSerializedCreateRequestBody(
	t *testing.T,
	request dashboardservicesdk.CreateDashboardRequest,
) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/dashboards", nil, nil)
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

func dashboardSerializedUpdateRequestBody(
	t *testing.T,
	request dashboardservicesdk.UpdateDashboardRequest,
) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(http.MethodPut, "/dashboards/ocid1.dashboard.oc1..existing", nil, nil)
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

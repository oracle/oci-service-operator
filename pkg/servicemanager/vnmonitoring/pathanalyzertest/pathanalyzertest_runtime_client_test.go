package pathanalyzertest

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	vnmonitoringsdk "github.com/oracle/oci-go-sdk/v65/vnmonitoring"
	vnmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/vnmonitoring/v1beta1"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"maps"
	ctrl "sigs.k8s.io/controller-runtime"
)

type stubPathAnalyzerTestOCIClient struct {
	create func(context.Context, vnmonitoringsdk.CreatePathAnalyzerTestRequest) (vnmonitoringsdk.CreatePathAnalyzerTestResponse, error)
	get    func(context.Context, vnmonitoringsdk.GetPathAnalyzerTestRequest) (vnmonitoringsdk.GetPathAnalyzerTestResponse, error)
	list   func(context.Context, vnmonitoringsdk.ListPathAnalyzerTestsRequest) (vnmonitoringsdk.ListPathAnalyzerTestsResponse, error)
	update func(context.Context, vnmonitoringsdk.UpdatePathAnalyzerTestRequest) (vnmonitoringsdk.UpdatePathAnalyzerTestResponse, error)
	delete func(context.Context, vnmonitoringsdk.DeletePathAnalyzerTestRequest) (vnmonitoringsdk.DeletePathAnalyzerTestResponse, error)
}

func (s stubPathAnalyzerTestOCIClient) CreatePathAnalyzerTest(
	ctx context.Context,
	req vnmonitoringsdk.CreatePathAnalyzerTestRequest,
) (vnmonitoringsdk.CreatePathAnalyzerTestResponse, error) {
	if s.create == nil {
		return vnmonitoringsdk.CreatePathAnalyzerTestResponse{}, nil
	}
	return s.create(ctx, req)
}

func (s stubPathAnalyzerTestOCIClient) GetPathAnalyzerTest(
	ctx context.Context,
	req vnmonitoringsdk.GetPathAnalyzerTestRequest,
) (vnmonitoringsdk.GetPathAnalyzerTestResponse, error) {
	if s.get == nil {
		return vnmonitoringsdk.GetPathAnalyzerTestResponse{}, nil
	}
	return s.get(ctx, req)
}

func (s stubPathAnalyzerTestOCIClient) ListPathAnalyzerTests(
	ctx context.Context,
	req vnmonitoringsdk.ListPathAnalyzerTestsRequest,
) (vnmonitoringsdk.ListPathAnalyzerTestsResponse, error) {
	if s.list == nil {
		return vnmonitoringsdk.ListPathAnalyzerTestsResponse{}, nil
	}
	return s.list(ctx, req)
}

func (s stubPathAnalyzerTestOCIClient) UpdatePathAnalyzerTest(
	ctx context.Context,
	req vnmonitoringsdk.UpdatePathAnalyzerTestRequest,
) (vnmonitoringsdk.UpdatePathAnalyzerTestResponse, error) {
	if s.update == nil {
		return vnmonitoringsdk.UpdatePathAnalyzerTestResponse{}, nil
	}
	return s.update(ctx, req)
}

func (s stubPathAnalyzerTestOCIClient) DeletePathAnalyzerTest(
	ctx context.Context,
	req vnmonitoringsdk.DeletePathAnalyzerTestRequest,
) (vnmonitoringsdk.DeletePathAnalyzerTestResponse, error) {
	if s.delete == nil {
		return vnmonitoringsdk.DeletePathAnalyzerTestResponse{}, nil
	}
	return s.delete(ctx, req)
}

func TestApplyPathAnalyzerTestRuntimeHooksUsesReviewedContract(t *testing.T) {
	t.Parallel()

	hooks := newPathAnalyzerTestDefaultRuntimeHooks(vnmonitoringsdk.VnMonitoringClient{})
	applyPathAnalyzerTestRuntimeHooks(nil, &hooks)

	if hooks.Semantics == nil || hooks.Semantics.List == nil {
		t.Fatal("hooks.Semantics.List = nil, want reviewed list semantics")
	}
	wantMatchFields := []string{
		"compartmentId",
		"displayName",
		"protocol",
		"sourceEndpoint",
		"destinationEndpoint",
		"protocolParameters",
		"queryOptions",
	}
	if got := hooks.Semantics.List.MatchFields; len(got) != len(wantMatchFields) {
		t.Fatalf("hooks.Semantics.List.MatchFields = %#v, want %#v", got, wantMatchFields)
	} else {
		for i := range got {
			if got[i] != wantMatchFields[i] {
				t.Fatalf("hooks.Semantics.List.MatchFields = %#v, want %#v", got, wantMatchFields)
			}
		}
	}
	if len(hooks.Semantics.Delete.TerminalStates) != 2 ||
		hooks.Semantics.Delete.TerminalStates[0] != "DELETED" ||
		hooks.Semantics.Delete.TerminalStates[1] != "NOT_FOUND" {
		t.Fatalf("hooks.Semantics.Delete.TerminalStates = %#v, want [DELETED NOT_FOUND]", hooks.Semantics.Delete.TerminalStates)
	}
	if hooks.Identity.Resolve == nil || hooks.Identity.GuardExistingBeforeCreate == nil || hooks.Identity.LookupExisting == nil {
		t.Fatal("PathAnalyzerTest identity hooks are incomplete")
	}
	if hooks.BuildCreateBody == nil || hooks.BuildUpdateBody == nil {
		t.Fatal("PathAnalyzerTest body builders are incomplete")
	}
	if len(hooks.List.Fields) != 2 {
		t.Fatalf("len(hooks.List.Fields) = %d, want 2", len(hooks.List.Fields))
	}
	if got := hooks.List.Fields[0].LookupPaths; len(got) != 3 || got[0] != "status.compartmentId" || got[1] != "spec.compartmentId" || got[2] != "compartmentId" {
		t.Fatalf("hooks.List.Fields[0].LookupPaths = %#v, want status-first compartment lookup", got)
	}
	if got := hooks.List.Fields[1].LookupPaths; len(got) != 3 || got[0] != "status.displayName" || got[1] != "spec.displayName" || got[2] != "displayName" {
		t.Fatalf("hooks.List.Fields[1].LookupPaths = %#v, want status-first displayName lookup", got)
	}
}

func TestBuildPathAnalyzerTestCreateBodyUsesJSONOverrides(t *testing.T) {
	t.Parallel()

	resource := newPathAnalyzerTestResource()
	resource.Spec.Protocol = 17
	resource.Spec.SourceEndpoint.Type = "SUBNET"
	resource.Spec.SourceEndpoint.Address = "10.0.0.10"
	resource.Spec.SourceEndpoint.JsonData = `{"type":"IP_ADDRESS","address":"10.0.9.9"}`
	resource.Spec.ProtocolParameters.Type = "TCP"
	resource.Spec.ProtocolParameters.DestinationPort = 443
	resource.Spec.ProtocolParameters.JsonData = `{"type":"UDP","destinationPort":53}`

	details, err := buildPathAnalyzerTestCreateBody(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("buildPathAnalyzerTestCreateBody() error = %v", err)
	}

	source, ok := details.SourceEndpoint.(vnmonitoringsdk.IpAddressEndpoint)
	if !ok {
		t.Fatalf("details.SourceEndpoint type = %T, want vnmonitoring.IpAddressEndpoint", details.SourceEndpoint)
	}
	if source.Address == nil || *source.Address != "10.0.9.9" {
		t.Fatalf("details.SourceEndpoint.Address = %#v, want 10.0.9.9", source.Address)
	}
	parameters, ok := details.ProtocolParameters.(vnmonitoringsdk.UdpProtocolParameters)
	if !ok {
		t.Fatalf("details.ProtocolParameters type = %T, want vnmonitoring.UdpProtocolParameters", details.ProtocolParameters)
	}
	if parameters.DestinationPort == nil || *parameters.DestinationPort != 53 {
		t.Fatalf("details.ProtocolParameters.DestinationPort = %#v, want 53", parameters.DestinationPort)
	}
	if details.QueryOptions == nil || details.QueryOptions.IsBiDirectionalAnalysis == nil || !*details.QueryOptions.IsBiDirectionalAnalysis {
		t.Fatalf("details.QueryOptions = %#v, want bidirectional analysis true", details.QueryOptions)
	}
}

func TestBuildPathAnalyzerTestUpdateBodyNoopsOnMatchingState(t *testing.T) {
	t.Parallel()

	resource := newPathAnalyzerTestResource()
	current := pathAnalyzerTestSDK("ocid1.pathanalyzertest.oc1..existing", resource)

	details, updateNeeded, err := buildPathAnalyzerTestUpdateBody(
		context.Background(),
		resource,
		"default",
		vnmonitoringsdk.GetPathAnalyzerTestResponse{PathAnalyzerTest: current},
	)
	if err != nil {
		t.Fatalf("buildPathAnalyzerTestUpdateBody() error = %v", err)
	}
	if updateNeeded {
		t.Fatalf("buildPathAnalyzerTestUpdateBody() updateNeeded = true with matching current state; body = %#v", details)
	}
}

func TestBuildPathAnalyzerTestUpdateBodySupportsClears(t *testing.T) {
	t.Parallel()

	resource := newPathAnalyzerTestResource()
	resource.Spec.DisplayName = ""
	resource.Spec.FreeformTags = nil
	resource.Spec.DefinedTags = nil
	resource.Spec.QueryOptions.IsBiDirectionalAnalysis = false

	current := pathAnalyzerTestSDK("ocid1.pathanalyzertest.oc1..existing", newPathAnalyzerTestResource())
	current.DisplayName = common.String("stale-name")
	current.FreeformTags = map[string]string{"env": "prod"}
	current.DefinedTags = map[string]map[string]interface{}{"oracle-tags": {"owner": "team-a"}}
	current.QueryOptions = &vnmonitoringsdk.QueryOptions{IsBiDirectionalAnalysis: common.Bool(true)}

	details, updateNeeded, err := buildPathAnalyzerTestUpdateBody(
		context.Background(),
		resource,
		"default",
		current,
	)
	if err != nil {
		t.Fatalf("buildPathAnalyzerTestUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildPathAnalyzerTestUpdateBody() updateNeeded = false, want clear intent")
	}
	if details.DisplayName == nil || *details.DisplayName != "" {
		t.Fatalf("details.DisplayName = %#v, want explicit empty string", details.DisplayName)
	}
	if len(details.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", details.FreeformTags)
	}
	if len(details.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", details.DefinedTags)
	}
	if details.QueryOptions == nil || details.QueryOptions.IsBiDirectionalAnalysis == nil || *details.QueryOptions.IsBiDirectionalAnalysis {
		t.Fatalf("details.QueryOptions = %#v, want bidirectional analysis false", details.QueryOptions)
	}
}

func TestNewPathAnalyzerTestServiceClientWithOCIClientReusesPagedExactMatch(t *testing.T) {
	t.Parallel()

	resource := newPathAnalyzerTestResource()
	existingID := "ocid1.pathanalyzertest.oc1..existing"
	createCalled := false
	listCalls := 0
	var getRequest vnmonitoringsdk.GetPathAnalyzerTestRequest

	client := newPathAnalyzerTestServiceClientWithOCIClient(
		stubPathAnalyzerTestOCIClient{
			create: func(context.Context, vnmonitoringsdk.CreatePathAnalyzerTestRequest) (vnmonitoringsdk.CreatePathAnalyzerTestResponse, error) {
				createCalled = true
				return vnmonitoringsdk.CreatePathAnalyzerTestResponse{}, nil
			},
			list: func(_ context.Context, req vnmonitoringsdk.ListPathAnalyzerTestsRequest) (vnmonitoringsdk.ListPathAnalyzerTestsResponse, error) {
				listCalls++
				switch listCalls {
				case 1:
					if req.DisplayName == nil || *req.DisplayName != resource.Spec.DisplayName {
						t.Fatalf("first ListPathAnalyzerTestsRequest.DisplayName = %#v, want %q", req.DisplayName, resource.Spec.DisplayName)
					}
					other := newPathAnalyzerTestResource()
					other.Spec.DestinationEndpoint.Address = "10.0.2.30"
					return vnmonitoringsdk.ListPathAnalyzerTestsResponse{
						PathAnalyzerTestCollection: vnmonitoringsdk.PathAnalyzerTestCollection{
							Items: []vnmonitoringsdk.PathAnalyzerTestSummary{
								pathAnalyzerTestSummary("ocid1.pathanalyzertest.oc1..other", other),
							},
						},
						OpcNextPage: common.String("next-page"),
					}, nil
				case 2:
					if req.Page == nil || *req.Page != "next-page" {
						t.Fatalf("second ListPathAnalyzerTestsRequest.Page = %#v, want next-page", req.Page)
					}
					return vnmonitoringsdk.ListPathAnalyzerTestsResponse{
						PathAnalyzerTestCollection: vnmonitoringsdk.PathAnalyzerTestCollection{
							Items: []vnmonitoringsdk.PathAnalyzerTestSummary{
								pathAnalyzerTestSummary(existingID, resource),
							},
						},
					}, nil
				default:
					t.Fatalf("unexpected ListPathAnalyzerTests call #%d", listCalls)
					return vnmonitoringsdk.ListPathAnalyzerTestsResponse{}, nil
				}
			},
			get: func(_ context.Context, req vnmonitoringsdk.GetPathAnalyzerTestRequest) (vnmonitoringsdk.GetPathAnalyzerTestResponse, error) {
				getRequest = req
				return vnmonitoringsdk.GetPathAnalyzerTestResponse{
					PathAnalyzerTest: pathAnalyzerTestSDK(existingID, resource),
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
		t.Fatal("CreateOrUpdate() invoked CreatePathAnalyzerTest, want existing-before-create reuse")
	}
	if getRequest.PathAnalyzerTestId == nil || *getRequest.PathAnalyzerTestId != existingID {
		t.Fatalf("GetPathAnalyzerTestRequest.PathAnalyzerTestId = %#v, want %q", getRequest.PathAnalyzerTestId, existingID)
	}
	if resource.Status.Id != existingID {
		t.Fatalf("resource.Status.Id = %q, want %q", resource.Status.Id, existingID)
	}
}

func TestNewPathAnalyzerTestServiceClientWithOCIClientConfirmsDeleteOnDeletedState(t *testing.T) {
	t.Parallel()

	resource := newPathAnalyzerTestResource()
	resource.Status.Id = "ocid1.pathanalyzertest.oc1..existing"

	client := newPathAnalyzerTestServiceClientWithOCIClient(
		stubPathAnalyzerTestOCIClient{
			delete: func(_ context.Context, req vnmonitoringsdk.DeletePathAnalyzerTestRequest) (vnmonitoringsdk.DeletePathAnalyzerTestResponse, error) {
				if req.PathAnalyzerTestId == nil || *req.PathAnalyzerTestId != resource.Status.Id {
					t.Fatalf("DeletePathAnalyzerTestRequest.PathAnalyzerTestId = %#v, want %q", req.PathAnalyzerTestId, resource.Status.Id)
				}
				return vnmonitoringsdk.DeletePathAnalyzerTestResponse{}, nil
			},
			get: func(_ context.Context, req vnmonitoringsdk.GetPathAnalyzerTestRequest) (vnmonitoringsdk.GetPathAnalyzerTestResponse, error) {
				if req.PathAnalyzerTestId == nil || *req.PathAnalyzerTestId != resource.Status.Id {
					t.Fatalf("GetPathAnalyzerTestRequest.PathAnalyzerTestId = %#v, want %q", req.PathAnalyzerTestId, resource.Status.Id)
				}
				current := pathAnalyzerTestSDK(resource.Status.Id, resource)
				current.LifecycleState = vnmonitoringsdk.PathAnalyzerTestLifecycleStateDeleted
				return vnmonitoringsdk.GetPathAnalyzerTestResponse{PathAnalyzerTest: current}, nil
			},
		},
	)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want confirmed delete")
	}
}

func newPathAnalyzerTestResource() *vnmonitoringv1beta1.PathAnalyzerTest {
	return &vnmonitoringv1beta1.PathAnalyzerTest{
		Spec: vnmonitoringv1beta1.PathAnalyzerTestSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "path-analyzer-test-a",
			Protocol:      6,
			SourceEndpoint: vnmonitoringv1beta1.PathAnalyzerTestSourceEndpoint{
				Type:    "IP_ADDRESS",
				Address: "10.0.0.10",
			},
			DestinationEndpoint: vnmonitoringv1beta1.PathAnalyzerTestDestinationEndpoint{
				Type:    "IP_ADDRESS",
				Address: "10.0.1.20",
			},
			ProtocolParameters: vnmonitoringv1beta1.PathAnalyzerTestProtocolParameters{
				Type:            "TCP",
				DestinationPort: 443,
			},
			QueryOptions: vnmonitoringv1beta1.PathAnalyzerTestQueryOptions{
				IsBiDirectionalAnalysis: true,
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"oracle-tags": {
					"owner": "team-a",
				},
			},
		},
	}
}

func pathAnalyzerTestSDK(id string, resource *vnmonitoringv1beta1.PathAnalyzerTest) vnmonitoringsdk.PathAnalyzerTest {
	now := common.SDKTime{Time: time.Unix(1700000000, 0)}
	sourceEndpoint, err := pathAnalyzerTestSourceEndpointFromSpec(resource.Spec.SourceEndpoint)
	if err != nil {
		panic(err)
	}
	destinationEndpoint, err := pathAnalyzerTestDestinationEndpointFromSpec(resource.Spec.DestinationEndpoint)
	if err != nil {
		panic(err)
	}
	protocolParameters, err := pathAnalyzerTestProtocolParametersFromSpec(resource.Spec.ProtocolParameters)
	if err != nil {
		panic(err)
	}
	definedTags, err := pathAnalyzerTestDefinedTagsFromSpec(resource.Spec.DefinedTags, false)
	if err != nil {
		panic(err)
	}

	return vnmonitoringsdk.PathAnalyzerTest{
		Id:                  common.String(id),
		DisplayName:         common.String(resource.Spec.DisplayName),
		CompartmentId:       common.String(resource.Spec.CompartmentId),
		Protocol:            common.Int(resource.Spec.Protocol),
		SourceEndpoint:      sourceEndpoint,
		DestinationEndpoint: destinationEndpoint,
		QueryOptions: &vnmonitoringsdk.QueryOptions{
			IsBiDirectionalAnalysis: common.Bool(resource.Spec.QueryOptions.IsBiDirectionalAnalysis),
		},
		TimeCreated:        &now,
		TimeUpdated:        &now,
		LifecycleState:     vnmonitoringsdk.PathAnalyzerTestLifecycleStateActive,
		ProtocolParameters: protocolParameters,
		FreeformTags:       maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:        definedTags,
		SystemTags:         map[string]map[string]interface{}{},
	}
}

func pathAnalyzerTestSummary(id string, resource *vnmonitoringv1beta1.PathAnalyzerTest) vnmonitoringsdk.PathAnalyzerTestSummary {
	now := common.SDKTime{Time: time.Unix(1700000000, 0)}
	sourceEndpoint, err := pathAnalyzerTestSourceEndpointFromSpec(resource.Spec.SourceEndpoint)
	if err != nil {
		panic(err)
	}
	destinationEndpoint, err := pathAnalyzerTestDestinationEndpointFromSpec(resource.Spec.DestinationEndpoint)
	if err != nil {
		panic(err)
	}
	protocolParameters, err := pathAnalyzerTestProtocolParametersFromSpec(resource.Spec.ProtocolParameters)
	if err != nil {
		panic(err)
	}
	definedTags, err := pathAnalyzerTestDefinedTagsFromSpec(resource.Spec.DefinedTags, false)
	if err != nil {
		panic(err)
	}

	return vnmonitoringsdk.PathAnalyzerTestSummary{
		Id:                  common.String(id),
		DisplayName:         common.String(resource.Spec.DisplayName),
		CompartmentId:       common.String(resource.Spec.CompartmentId),
		Protocol:            common.Int(resource.Spec.Protocol),
		SourceEndpoint:      sourceEndpoint,
		DestinationEndpoint: destinationEndpoint,
		QueryOptions: &vnmonitoringsdk.QueryOptions{
			IsBiDirectionalAnalysis: common.Bool(resource.Spec.QueryOptions.IsBiDirectionalAnalysis),
		},
		TimeCreated:        &now,
		TimeUpdated:        &now,
		LifecycleState:     vnmonitoringsdk.PathAnalyzerTestLifecycleStateActive,
		ProtocolParameters: protocolParameters,
		FreeformTags:       maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:        definedTags,
		SystemTags:         map[string]map[string]interface{}{},
	}
}

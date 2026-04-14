/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package internetgateway

import (
	"context"
	stderrors "errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	osokcore "github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeInternetGatewayOCIClient struct {
	createFn func(context.Context, coresdk.CreateInternetGatewayRequest) (coresdk.CreateInternetGatewayResponse, error)
	getFn    func(context.Context, coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error)
	listFn   func(context.Context, coresdk.ListInternetGatewaysRequest) (coresdk.ListInternetGatewaysResponse, error)
	updateFn func(context.Context, coresdk.UpdateInternetGatewayRequest) (coresdk.UpdateInternetGatewayResponse, error)
	deleteFn func(context.Context, coresdk.DeleteInternetGatewayRequest) (coresdk.DeleteInternetGatewayResponse, error)
}

func (f *fakeInternetGatewayOCIClient) CreateInternetGateway(ctx context.Context, req coresdk.CreateInternetGatewayRequest) (coresdk.CreateInternetGatewayResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return coresdk.CreateInternetGatewayResponse{}, nil
}

func (f *fakeInternetGatewayOCIClient) GetInternetGateway(ctx context.Context, req coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return coresdk.GetInternetGatewayResponse{}, fakeInternetGatewayServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "missing",
	}
}

func (f *fakeInternetGatewayOCIClient) ListInternetGateways(ctx context.Context, req coresdk.ListInternetGatewaysRequest) (coresdk.ListInternetGatewaysResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return coresdk.ListInternetGatewaysResponse{}, nil
}

func (f *fakeInternetGatewayOCIClient) UpdateInternetGateway(ctx context.Context, req coresdk.UpdateInternetGatewayRequest) (coresdk.UpdateInternetGatewayResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return coresdk.UpdateInternetGatewayResponse{}, nil
}

func (f *fakeInternetGatewayOCIClient) DeleteInternetGateway(ctx context.Context, req coresdk.DeleteInternetGatewayRequest) (coresdk.DeleteInternetGatewayResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return coresdk.DeleteInternetGatewayResponse{}, nil
}

type fakeInternetGatewayServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeInternetGatewayServiceError) Error() string          { return f.message }
func (f fakeInternetGatewayServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeInternetGatewayServiceError) GetMessage() string     { return f.message }
func (f fakeInternetGatewayServiceError) GetCode() string        { return f.code }
func (f fakeInternetGatewayServiceError) GetOpcRequestID() string {
	return ""
}

func newTestGeneratedInternetGatewayDelegate(manager *InternetGatewayServiceManager, client internetGatewayOCIClient) InternetGatewayServiceClient {
	if client == nil {
		client = &fakeInternetGatewayOCIClient{}
	}

	config := generatedruntime.Config[*corev1beta1.InternetGateway]{
		Kind:    "InternetGateway",
		SDKName: "InternetGateway",
		Log:     manager.Log,
		Semantics: &generatedruntime.Semantics{
			FormalService: "core",
			FormalSlug:    "internetgateway",
			Async: &generatedruntime.AsyncSemantics{
				Strategy:             "lifecycle",
				Runtime:              "generatedruntime",
				FormalClassification: "lifecycle",
			},
			StatusProjection:  "required",
			SecretSideEffects: "none",
			FinalizerPolicy:   "retain-until-confirmed-delete",
			Lifecycle: generatedruntime.LifecycleSemantics{
				ProvisioningStates: []string{"PROVISIONING"},
				UpdatingStates:     []string{},
				ActiveStates:       []string{"AVAILABLE"},
			},
			Delete: generatedruntime.DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"TERMINATED", "TERMINATING"},
				TerminalStates: []string{"NOT_FOUND"},
			},
			List: &generatedruntime.ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"compartmentId", "displayName", "id", "state", "vcnId"},
			},
			Mutation: generatedruntime.MutationSemantics{
				Mutable:       []string{"definedTags", "displayName", "freeformTags", "isEnabled", "routeTableId"},
				ForceNew:      []string{"compartmentId", "vcnId"},
				ConflictsWith: map[string][]string{},
			},
			Hooks: generatedruntime.HookSet{
				Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
				Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
				Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
			},
			CreateFollowUp: generatedruntime.FollowUpSemantics{
				Strategy: "read-after-write",
				Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			},
			UpdateFollowUp: generatedruntime.FollowUpSemantics{
				Strategy: "read-after-write",
				Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			},
			DeleteFollowUp: generatedruntime.FollowUpSemantics{
				Strategy: "confirm-delete",
				Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
			},
		},
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.CreateInternetGatewayRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateInternetGateway(ctx, *request.(*coresdk.CreateInternetGatewayRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "CreateInternetGatewayDetails", RequestName: "CreateInternetGatewayDetails", Contribution: "body"}},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.GetInternetGatewayRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetInternetGateway(ctx, *request.(*coresdk.GetInternetGatewayRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "IgId", RequestName: "igId", Contribution: "path", PreferResourceID: true}},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.ListInternetGatewaysRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListInternetGateways(ctx, *request.(*coresdk.ListInternetGatewaysRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "VcnId", RequestName: "vcnId", Contribution: "query"},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
				{FieldName: "Page", RequestName: "page", Contribution: "query"},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
				{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.UpdateInternetGatewayRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateInternetGateway(ctx, *request.(*coresdk.UpdateInternetGatewayRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "IgId", RequestName: "igId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateInternetGatewayDetails", RequestName: "UpdateInternetGatewayDetails", Contribution: "body"},
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.DeleteInternetGatewayRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteInternetGateway(ctx, *request.(*coresdk.DeleteInternetGatewayRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "IgId", RequestName: "igId", Contribution: "path", PreferResourceID: true}},
		},
	}

	return defaultInternetGatewayServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*corev1beta1.InternetGateway](config),
	}
}

func newInternetGatewayTestManager(client internetGatewayOCIClient) *InternetGatewayServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewInternetGatewayServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&internetGatewayGeneratedParityClient{
			manager:  manager,
			delegate: newTestGeneratedInternetGatewayDelegate(manager, client),
			client:   client,
		})
	}
	return manager
}

func makeSpecInternetGateway() *corev1beta1.InternetGateway {
	return &corev1beta1.InternetGateway{
		Spec: corev1beta1.InternetGatewaySpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			IsEnabled:     true,
			VcnId:         "ocid1.vcn.oc1..example",
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			DisplayName:   "test-internet-gateway",
			FreeformTags:  map[string]string{"env": "dev"},
			RouteTableId:  "ocid1.routetable.oc1..example",
		},
	}
}

func makeSDKInternetGateway(id, displayName string, state coresdk.InternetGatewayLifecycleStateEnum) coresdk.InternetGateway {
	return coresdk.InternetGateway{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		LifecycleState: state,
		VcnId:          common.String("ocid1.vcn.oc1..example"),
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		DisplayName:    common.String(displayName),
		FreeformTags:   map[string]string{"env": "dev"},
		IsEnabled:      common.Bool(true),
		TimeCreated:    &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		RouteTableId:   common.String("ocid1.routetable.oc1..example"),
	}
}

func TestCreateOrUpdate_CreateSuccessAndStatusProjection(t *testing.T) {
	var captured coresdk.CreateInternetGatewayRequest
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		createFn: func(_ context.Context, req coresdk.CreateInternetGatewayRequest) (coresdk.CreateInternetGatewayResponse, error) {
			captured = req
			return coresdk.CreateInternetGatewayResponse{
				InternetGateway: makeSDKInternetGateway("ocid1.internetgateway.oc1..create", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecInternetGateway()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CompartmentId)
	assert.Equal(t, common.Bool(true), captured.IsEnabled)
	assert.Equal(t, common.String("ocid1.vcn.oc1..example"), captured.VcnId)
	assert.Equal(t, common.String("test-internet-gateway"), captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, common.String("ocid1.routetable.oc1..example"), captured.RouteTableId)
	assert.Equal(t, "ocid1.internetgateway.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, "test-internet-gateway", resource.Status.DisplayName)
	assert.True(t, resource.Status.IsEnabled)
	assert.Equal(t, "2026-04-01T00:00:00Z", resource.Status.TimeCreated)
}

func TestCreateOrUpdate_ObserveByStatusOCID_NoOpWhenStateMatches(t *testing.T) {
	getCalls := 0
	updateCalls := 0
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		getFn: func(_ context.Context, req coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.internetgateway.oc1..existing", *req.IgId)
			return coresdk.GetInternetGatewayResponse{
				InternetGateway: makeSDKInternetGateway("ocid1.internetgateway.oc1..existing", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateInternetGatewayRequest) (coresdk.UpdateInternetGatewayResponse, error) {
			updateCalls++
			return coresdk.UpdateInternetGatewayResponse{}, nil
		},
	})

	resource := makeSpecInternetGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 2, getCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
}

func TestCreateOrUpdate_ClearsStaleOptionalStatusFieldsOnProjection(t *testing.T) {
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			current := makeSDKInternetGateway("ocid1.internetgateway.oc1..existing", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateAvailable)
			current.DisplayName = nil
			current.DefinedTags = nil
			current.FreeformTags = nil
			current.IsEnabled = nil
			current.TimeCreated = nil
			current.RouteTableId = nil
			return coresdk.GetInternetGatewayResponse{InternetGateway: current}, nil
		},
	})

	resource := makeSpecInternetGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..existing")
	resource.Spec.DisplayName = ""
	resource.Spec.DefinedTags = nil
	resource.Spec.FreeformTags = nil
	resource.Spec.IsEnabled = false
	resource.Spec.RouteTableId = ""
	resource.Status.DisplayName = "stale-name"
	resource.Status.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Status.FreeformTags = map[string]string{"env": "stale"}
	resource.Status.IsEnabled = true
	resource.Status.TimeCreated = "2026-04-01T00:00:00Z"
	resource.Status.RouteTableId = "ocid1.routetable.oc1..stale"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "", resource.Status.DisplayName)
	assert.Nil(t, resource.Status.DefinedTags)
	assert.Nil(t, resource.Status.FreeformTags)
	assert.False(t, resource.Status.IsEnabled)
	assert.Equal(t, "", resource.Status.TimeCreated)
	assert.Equal(t, "", resource.Status.RouteTableId)
}

func TestCreateOrUpdate_UpdateFailureAfterLiveGetKeepsClearedOptionalStatusFields(t *testing.T) {
	getCalls := 0
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			getCalls++
			current := makeSDKInternetGateway("ocid1.internetgateway.oc1..existing", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateAvailable)
			current.DisplayName = nil
			current.DefinedTags = nil
			current.FreeformTags = nil
			current.IsEnabled = nil
			current.TimeCreated = nil
			current.RouteTableId = nil
			return coresdk.GetInternetGatewayResponse{InternetGateway: current}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateInternetGatewayRequest) (coresdk.UpdateInternetGatewayResponse, error) {
			return coresdk.UpdateInternetGatewayResponse{}, fakeInternetGatewayServiceError{
				statusCode: 409,
				code:       errorutil.IncorrectState,
				message:    "update failed",
			}
		},
	})

	resource := makeSpecInternetGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..existing")
	resource.Spec.DisplayName = "new-name"
	resource.Status.DisplayName = "stale-name"
	resource.Status.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Status.FreeformTags = map[string]string{"env": "stale"}
	resource.Status.IsEnabled = true
	resource.Status.TimeCreated = "2026-04-01T00:00:00Z"
	resource.Status.RouteTableId = "ocid1.routetable.oc1..stale"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Equal(t, 2, getCalls)
	assert.Equal(t, "", resource.Status.DisplayName)
	assert.Nil(t, resource.Status.DefinedTags)
	assert.Nil(t, resource.Status.FreeformTags)
	assert.False(t, resource.Status.IsEnabled)
	assert.Equal(t, "", resource.Status.TimeCreated)
	assert.Equal(t, "", resource.Status.RouteTableId)
	assert.Equal(t, string(shared.Failed), resource.Status.OsokStatus.Reason)
}

func TestCreateOrUpdate_MutableDriftTriggersUpdate(t *testing.T) {
	var captured coresdk.UpdateInternetGatewayRequest
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			current := makeSDKInternetGateway("ocid1.internetgateway.oc1..existing", "old-name", coresdk.InternetGatewayLifecycleStateAvailable)
			current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "41"}}
			current.FreeformTags = map[string]string{"env": "stage"}
			current.IsEnabled = common.Bool(true)
			current.RouteTableId = common.String("ocid1.routetable.oc1..old")
			return coresdk.GetInternetGatewayResponse{InternetGateway: current}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateInternetGatewayRequest) (coresdk.UpdateInternetGatewayResponse, error) {
			captured = req
			updated := makeSDKInternetGateway("ocid1.internetgateway.oc1..existing", "new-name", coresdk.InternetGatewayLifecycleStateAvailable)
			updated.IsEnabled = common.Bool(false)
			return coresdk.UpdateInternetGatewayResponse{InternetGateway: updated}, nil
		},
	})

	resource := makeSpecInternetGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..existing")
	resource.Spec.DisplayName = "new-name"
	resource.Spec.IsEnabled = false

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "ocid1.internetgateway.oc1..existing", *captured.IgId)
	assert.Equal(t, "new-name", *captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, false, *captured.IsEnabled)
	assert.Equal(t, "ocid1.routetable.oc1..example", *captured.RouteTableId)
	assert.Equal(t, "new-name", resource.Status.DisplayName)
	assert.False(t, resource.Status.IsEnabled)
}

func TestCreateOrUpdate_ExplicitEmptyTagMapsTriggerUpdate(t *testing.T) {
	var captured coresdk.UpdateInternetGatewayRequest
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			return coresdk.GetInternetGatewayResponse{
				InternetGateway: makeSDKInternetGateway("ocid1.internetgateway.oc1..existing", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateInternetGatewayRequest) (coresdk.UpdateInternetGatewayResponse, error) {
			captured = req
			updated := makeSDKInternetGateway("ocid1.internetgateway.oc1..existing", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateAvailable)
			updated.DefinedTags = map[string]map[string]interface{}{}
			updated.FreeformTags = map[string]string{}
			return coresdk.UpdateInternetGatewayResponse{InternetGateway: updated}, nil
		},
	})

	resource := makeSpecInternetGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..existing")
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	resource.Spec.FreeformTags = map[string]string{}

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, map[string]map[string]interface{}{}, captured.DefinedTags)
	assert.Equal(t, map[string]string{}, captured.FreeformTags)
	assert.Len(t, resource.Status.DefinedTags, 0)
	assert.Len(t, resource.Status.FreeformTags, 0)
}

func TestCreateOrUpdate_RejectsUnsupportedCreateOnlyDrift(t *testing.T) {
	tests := []struct {
		name        string
		mutateSpec  func(*corev1beta1.InternetGateway)
		expectField string
	}{
		{
			name: "compartmentId",
			mutateSpec: func(resource *corev1beta1.InternetGateway) {
				resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
			},
			expectField: "compartmentId",
		},
		{
			name: "vcnId",
			mutateSpec: func(resource *corev1beta1.InternetGateway) {
				resource.Spec.VcnId = "ocid1.vcn.oc1..different"
			},
			expectField: "vcnId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
					return coresdk.GetInternetGatewayResponse{
						InternetGateway: makeSDKInternetGateway("ocid1.internetgateway.oc1..existing", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateAvailable),
					}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateInternetGatewayRequest) (coresdk.UpdateInternetGatewayResponse, error) {
					updateCalls++
					return coresdk.UpdateInternetGatewayResponse{}, nil
				},
			})

			resource := makeSpecInternetGateway()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..existing")
			tt.mutateSpec(resource)

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.Error(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.Contains(t, err.Error(), tt.expectField)
			assert.Equal(t, 0, updateCalls)
		})
	}
}

func TestCreateOrUpdate_RecreatesOnNotFound(t *testing.T) {
	getCalls := 0
	listCalls := 0
	createCalls := 0
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			getCalls++
			if getCalls == 1 {
				return coresdk.GetInternetGatewayResponse{}, fakeInternetGatewayServiceError{
					statusCode: 404,
					code:       "NotFound",
					message:    "not found",
				}
			}
			return coresdk.GetInternetGatewayResponse{
				InternetGateway: makeSDKInternetGateway("ocid1.internetgateway.oc1..recreated", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateAvailable),
			}, nil
		},
		listFn: func(_ context.Context, _ coresdk.ListInternetGatewaysRequest) (coresdk.ListInternetGatewaysResponse, error) {
			listCalls++
			return coresdk.ListInternetGatewaysResponse{}, nil
		},
		createFn: func(_ context.Context, req coresdk.CreateInternetGatewayRequest) (coresdk.CreateInternetGatewayResponse, error) {
			createCalls++
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			assert.Equal(t, common.Bool(true), req.IsEnabled)
			return coresdk.CreateInternetGatewayResponse{
				InternetGateway: makeSDKInternetGateway("ocid1.internetgateway.oc1..recreated", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecInternetGateway()
	resource.Status.Id = "ocid1.internetgateway.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 2, getCalls)
	assert.Equal(t, 0, listCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, shared.OCID("ocid1.internetgateway.oc1..recreated"), resource.Status.OsokStatus.Ocid)
	assert.Equal(t, "ocid1.internetgateway.oc1..recreated", resource.Status.Id)
}

func TestCreateOrUpdate_DoesNotRecreateOnAuthShapedNotFound(t *testing.T) {
	getCalls := 0
	listCalls := 0
	createCalls := 0
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			getCalls++
			return coresdk.GetInternetGatewayResponse{}, fakeInternetGatewayServiceError{
				statusCode: 404,
				code:       errorutil.NotAuthorizedOrNotFound,
				message:    "not authorized or not found",
			}
		},
		listFn: func(_ context.Context, _ coresdk.ListInternetGatewaysRequest) (coresdk.ListInternetGatewaysResponse, error) {
			listCalls++
			return coresdk.ListInternetGatewaysResponse{}, nil
		},
		createFn: func(_ context.Context, _ coresdk.CreateInternetGatewayRequest) (coresdk.CreateInternetGatewayResponse, error) {
			createCalls++
			return coresdk.CreateInternetGatewayResponse{}, nil
		},
	})

	resource := makeSpecInternetGateway()
	resource.Status.Id = "ocid1.internetgateway.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, listCalls)
	assert.Equal(t, 0, createCalls)
	assert.Equal(t, string(shared.Failed), resource.Status.OsokStatus.Reason)
}

func TestDelete_ConfirmsDeletionOnNotFound(t *testing.T) {
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteInternetGatewayRequest) (coresdk.DeleteInternetGatewayResponse, error) {
			assert.Equal(t, "ocid1.internetgateway.oc1..delete", *req.IgId)
			return coresdk.DeleteInternetGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			return coresdk.GetInternetGatewayResponse{}, fakeInternetGatewayServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
	})

	resource := makeSpecInternetGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_KeepsFinalizerWhileObservedTerminating(t *testing.T) {
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteInternetGatewayRequest) (coresdk.DeleteInternetGatewayResponse, error) {
			return coresdk.DeleteInternetGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			return coresdk.GetInternetGatewayResponse{
				InternetGateway: makeSDKInternetGateway("ocid1.internetgateway.oc1..delete", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateTerminating),
			}, nil
		},
	})

	resource := makeSpecInternetGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, "TERMINATING", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestDelete_KeepsFinalizerWhileObservedTerminated(t *testing.T) {
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteInternetGatewayRequest) (coresdk.DeleteInternetGatewayResponse, error) {
			return coresdk.DeleteInternetGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			return coresdk.GetInternetGatewayResponse{
				InternetGateway: makeSDKInternetGateway("ocid1.internetgateway.oc1..delete", "test-internet-gateway", coresdk.InternetGatewayLifecycleStateTerminated),
			}, nil
		},
	})

	resource := makeSpecInternetGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.internetgateway.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, "TERMINATED", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestIsInternetGatewayReadNotFoundOCI_RejectsAuthAmbiguity(t *testing.T) {
	assert.True(t, isInternetGatewayReadNotFoundOCI(errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "normalized not found",
	}))
	assert.False(t, isInternetGatewayReadNotFoundOCI(errorutil.UnauthorizedAndNotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotAuthorizedOrNotFound,
		Description:    "normalized auth ambiguity",
	}))
	assert.False(t, isInternetGatewayReadNotFoundOCI(fakeInternetGatewayServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isInternetGatewayReadNotFoundOCI(fakeInternetGatewayServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
	assert.False(t, isInternetGatewayReadNotFoundOCI(fakeInternetGatewayServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not found",
	}))
	assert.False(t, isInternetGatewayReadNotFoundOCI(errorutil.ConflictOciError{
		HTTPStatusCode: 409,
		ErrorCode:      errorutil.IncorrectState,
		Description:    "normalized conflict",
	}))
	assert.False(t, isInternetGatewayReadNotFoundOCI(fakeInternetGatewayServiceError{
		statusCode: 409,
		code:       errorutil.IncorrectState,
		message:    "resource conflict",
	}))
}

func TestIsInternetGatewayDeleteNotFoundOCI_AcceptsAuthShaped404(t *testing.T) {
	assert.True(t, isInternetGatewayDeleteNotFoundOCI(errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "normalized not found",
	}))
	assert.True(t, isInternetGatewayDeleteNotFoundOCI(errorutil.UnauthorizedAndNotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotAuthorizedOrNotFound,
		Description:    "normalized auth ambiguity",
	}))
	assert.True(t, isInternetGatewayDeleteNotFoundOCI(fakeInternetGatewayServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isInternetGatewayDeleteNotFoundOCI(fakeInternetGatewayServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
	assert.False(t, isInternetGatewayDeleteNotFoundOCI(fakeInternetGatewayServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not found",
	}))
	assert.False(t, isInternetGatewayDeleteNotFoundOCI(errorutil.ConflictOciError{
		HTTPStatusCode: 409,
		ErrorCode:      errorutil.IncorrectState,
		Description:    "normalized conflict",
	}))
	assert.False(t, isInternetGatewayDeleteNotFoundOCI(fakeInternetGatewayServiceError{
		statusCode: 409,
		code:       errorutil.IncorrectState,
		message:    "resource conflict",
	}))
}

func TestReconcileDelete_ReleasesFinalizerOnAuthShapedNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.InternetGateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "InternetGateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-internet-gateway-auth-shaped-404",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.InternetGatewayStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.internetgateway.oc1..delete"),
			},
		},
	}

	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteInternetGatewayRequest) (coresdk.DeleteInternetGatewayResponse, error) {
			assert.Equal(t, "ocid1.internetgateway.oc1..delete", *req.IgId)
			return coresdk.DeleteInternetGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			return coresdk.GetInternetGatewayResponse{}, fakeInternetGatewayServiceError{
				statusCode: 404,
				code:       errorutil.NotAuthorizedOrNotFound,
				message:    "not authorized or not found",
			}
		},
	})

	kubeClient := newMemoryInternetGatewayClient(scheme, resource)
	recorder := record.NewFakeRecorder(10)
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	reconciler := &osokcore.BaseReconciler{
		Client:             kubeClient,
		OSOKServiceManager: manager,
		Log:                log,
		Metrics:            &metrics.Metrics{Name: "oci", ServiceName: "core", Logger: log},
		Recorder:           recorder,
		Scheme:             scheme,
	}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: ctrlclient.ObjectKey{Name: "test-internet-gateway-auth-shaped-404", Namespace: "default"},
	}, &corev1beta1.InternetGateway{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredInternetGateway(), osokcore.OSOKFinalizerName))

	events := drainInternetGatewayEvents(recorder)
	assertInternetGatewayEventContains(t, events, "Removed finalizer")
	assertNoInternetGatewayEventContains(t, events, "Failed to delete resource")
}

func TestReconcileDelete_ReleasesFinalizerOnUnambiguousNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.InternetGateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "InternetGateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-internet-gateway",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.InternetGatewayStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.internetgateway.oc1..delete"),
			},
		},
	}

	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteInternetGatewayRequest) (coresdk.DeleteInternetGatewayResponse, error) {
			assert.Equal(t, "ocid1.internetgateway.oc1..delete", *req.IgId)
			return coresdk.DeleteInternetGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			return coresdk.GetInternetGatewayResponse{}, fakeInternetGatewayServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "resource not found",
			}
		},
	})

	kubeClient := newMemoryInternetGatewayClient(scheme, resource)
	recorder := record.NewFakeRecorder(10)
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	reconciler := &osokcore.BaseReconciler{
		Client:             kubeClient,
		OSOKServiceManager: manager,
		Log:                log,
		Metrics:            &metrics.Metrics{Name: "oci", ServiceName: "core", Logger: log},
		Recorder:           recorder,
		Scheme:             scheme,
	}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: ctrlclient.ObjectKey{Name: "test-internet-gateway", Namespace: "default"},
	}, &corev1beta1.InternetGateway{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredInternetGateway(), osokcore.OSOKFinalizerName))

	events := drainInternetGatewayEvents(recorder)
	assertInternetGatewayEventContains(t, events, "Removed finalizer")
	assertNoInternetGatewayEventContains(t, events, "Failed to delete resource")
}

func drainInternetGatewayEvents(recorder *record.FakeRecorder) []string {
	events := make([]string, 0, len(recorder.Events))
	for {
		select {
		case event := <-recorder.Events:
			events = append(events, event)
		default:
			return events
		}
	}
}

func assertInternetGatewayEventContains(t *testing.T, events []string, want string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, want) {
			return
		}
	}
	t.Fatalf("events %v do not contain %q", events, want)
}

func assertNoInternetGatewayEventContains(t *testing.T, events []string, unexpected string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, unexpected) {
			t.Fatalf("events %v unexpectedly contain %q", events, unexpected)
		}
	}
}

type memoryInternetGatewayClient struct {
	ctrlclient.Client
	stored ctrlclient.Object
}

func newMemoryInternetGatewayClient(scheme *runtime.Scheme, obj ctrlclient.Object) *memoryInternetGatewayClient {
	return &memoryInternetGatewayClient{
		Client: ctrlclientfake.NewClientBuilder().WithScheme(scheme).Build(),
		stored: obj.DeepCopyObject().(ctrlclient.Object),
	}
}

func (c *memoryInternetGatewayClient) Get(_ context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, _ ...ctrlclient.GetOption) error {
	if c.stored == nil || c.stored.GetName() != key.Name || c.stored.GetNamespace() != key.Namespace {
		return apierrors.NewNotFound(schema.GroupResource{Group: "core.oracle.com", Resource: "internetgateways"}, key.Name)
	}

	value := reflect.ValueOf(obj)
	source := reflect.ValueOf(c.stored.DeepCopyObject())
	if value.Kind() != reflect.Ptr || source.Kind() != reflect.Ptr {
		return stderrors.New("memory client requires pointer objects")
	}
	value.Elem().Set(source.Elem())
	return nil
}

func (c *memoryInternetGatewayClient) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.UpdateOption) error {
	c.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (c *memoryInternetGatewayClient) StoredInternetGateway() *corev1beta1.InternetGateway {
	return c.stored.DeepCopyObject().(*corev1beta1.InternetGateway)
}

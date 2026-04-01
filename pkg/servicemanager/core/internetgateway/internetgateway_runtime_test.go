/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package internetgateway

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeInternetGatewayOCIClient struct {
	createFn func(context.Context, coresdk.CreateInternetGatewayRequest) (coresdk.CreateInternetGatewayResponse, error)
	getFn    func(context.Context, coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error)
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
	return coresdk.GetInternetGatewayResponse{}, nil
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

func newInternetGatewayTestManager(client internetGatewayOCIClient) *InternetGatewayServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewInternetGatewayServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&internetGatewayRuntimeClient{
			manager: manager,
			client:  client,
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
	assert.Equal(t, 1, getCalls)
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
	createCalls := 0
	manager := newInternetGatewayTestManager(&fakeInternetGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetInternetGatewayRequest) (coresdk.GetInternetGatewayResponse, error) {
			getCalls++
			return coresdk.GetInternetGatewayResponse{}, fakeInternetGatewayServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
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
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, shared.OCID("ocid1.internetgateway.oc1..recreated"), resource.Status.OsokStatus.Ocid)
	assert.Equal(t, "ocid1.internetgateway.oc1..recreated", resource.Status.Id)
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

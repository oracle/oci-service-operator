/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package routetable

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

type fakeRouteTableOCIClient struct {
	createFn func(context.Context, coresdk.CreateRouteTableRequest) (coresdk.CreateRouteTableResponse, error)
	getFn    func(context.Context, coresdk.GetRouteTableRequest) (coresdk.GetRouteTableResponse, error)
	updateFn func(context.Context, coresdk.UpdateRouteTableRequest) (coresdk.UpdateRouteTableResponse, error)
	deleteFn func(context.Context, coresdk.DeleteRouteTableRequest) (coresdk.DeleteRouteTableResponse, error)
}

func (f *fakeRouteTableOCIClient) CreateRouteTable(ctx context.Context, req coresdk.CreateRouteTableRequest) (coresdk.CreateRouteTableResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return coresdk.CreateRouteTableResponse{}, nil
}

func (f *fakeRouteTableOCIClient) GetRouteTable(ctx context.Context, req coresdk.GetRouteTableRequest) (coresdk.GetRouteTableResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return coresdk.GetRouteTableResponse{}, nil
}

func (f *fakeRouteTableOCIClient) UpdateRouteTable(ctx context.Context, req coresdk.UpdateRouteTableRequest) (coresdk.UpdateRouteTableResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return coresdk.UpdateRouteTableResponse{}, nil
}

func (f *fakeRouteTableOCIClient) DeleteRouteTable(ctx context.Context, req coresdk.DeleteRouteTableRequest) (coresdk.DeleteRouteTableResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return coresdk.DeleteRouteTableResponse{}, nil
}

type fakeRouteTableServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeRouteTableServiceError) Error() string          { return f.message }
func (f fakeRouteTableServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeRouteTableServiceError) GetMessage() string     { return f.message }
func (f fakeRouteTableServiceError) GetCode() string        { return f.code }
func (f fakeRouteTableServiceError) GetOpcRequestID() string {
	return ""
}

func newRouteTableTestManager(client routeTableOCIClient) *RouteTableServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewRouteTableServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&routeTableRuntimeClient{
			manager: manager,
			client:  client,
		})
	}
	return manager
}

func makeSpecRouteTable() *corev1beta1.RouteTable {
	return &corev1beta1.RouteTable{
		Spec: corev1beta1.RouteTableSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			VcnId:         "ocid1.vcn.oc1..example",
			DisplayName:   "test-route-table",
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			FreeformTags:  map[string]string{"env": "dev"},
			RouteRules: []corev1beta1.RouteTableRouteRule{
				{
					NetworkEntityId: "ocid1.internetgateway.oc1..example",
					Destination:     "0.0.0.0/0",
					DestinationType: string(coresdk.RouteRuleDestinationTypeCidrBlock),
					Description:     "internet access",
				},
				{
					NetworkEntityId: "ocid1.servicegateway.oc1..example",
					Destination:     "oci-phx-objectstorage",
					DestinationType: string(coresdk.RouteRuleDestinationTypeServiceCidrBlock),
					Description:     "service access",
				},
			},
		},
	}
}

func makeSDKRouteTable(id, displayName string, state coresdk.RouteTableLifecycleStateEnum) coresdk.RouteTable {
	return coresdk.RouteTable{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		VcnId:          common.String("ocid1.vcn.oc1..example"),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		FreeformTags:   map[string]string{"env": "dev"},
		RouteRules: []coresdk.RouteRule{
			{
				NetworkEntityId: common.String("ocid1.internetgateway.oc1..example"),
				Destination:     common.String("0.0.0.0/0"),
				DestinationType: coresdk.RouteRuleDestinationTypeCidrBlock,
				Description:     common.String("internet access"),
				RouteType:       coresdk.RouteRuleRouteTypeStatic,
			},
			{
				NetworkEntityId: common.String("ocid1.servicegateway.oc1..example"),
				Destination:     common.String("oci-phx-objectstorage"),
				DestinationType: coresdk.RouteRuleDestinationTypeServiceCidrBlock,
				Description:     common.String("service access"),
				RouteType:       coresdk.RouteRuleRouteTypeStatic,
			},
		},
		TimeCreated: &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
	}
}

func TestCreateOrUpdate_CreateSuccessAndStatusProjection(t *testing.T) {
	var captured coresdk.CreateRouteTableRequest
	manager := newRouteTableTestManager(&fakeRouteTableOCIClient{
		createFn: func(_ context.Context, req coresdk.CreateRouteTableRequest) (coresdk.CreateRouteTableResponse, error) {
			captured = req
			return coresdk.CreateRouteTableResponse{
				RouteTable: makeSDKRouteTable("ocid1.routetable.oc1..create", "test-route-table", coresdk.RouteTableLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecRouteTable()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CompartmentId)
	assert.Equal(t, common.String("ocid1.vcn.oc1..example"), captured.VcnId)
	assert.Equal(t, common.String("test-route-table"), captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Len(t, captured.RouteRules, 2)
	assert.Equal(t, "ocid1.routetable.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, "test-route-table", resource.Status.DisplayName)
	assert.Equal(t, "2026-04-01T00:00:00Z", resource.Status.TimeCreated)
	assert.Len(t, resource.Status.RouteRules, 2)
	assert.Equal(t, "0.0.0.0/0", resource.Status.RouteRules[0].Destination)
	assert.Equal(t, string(coresdk.RouteRuleRouteTypeStatic), resource.Status.RouteRules[0].RouteType)
}

func TestCreateOrUpdate_ObserveByStatusOCID_NoOpWhenStateMatches(t *testing.T) {
	getCalls := 0
	updateCalls := 0
	manager := newRouteTableTestManager(&fakeRouteTableOCIClient{
		getFn: func(_ context.Context, req coresdk.GetRouteTableRequest) (coresdk.GetRouteTableResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.routetable.oc1..existing", *req.RtId)
			return coresdk.GetRouteTableResponse{
				RouteTable: makeSDKRouteTable("ocid1.routetable.oc1..existing", "test-route-table", coresdk.RouteTableLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateRouteTableRequest) (coresdk.UpdateRouteTableResponse, error) {
			updateCalls++
			return coresdk.UpdateRouteTableResponse{}, nil
		},
	})

	resource := makeSpecRouteTable()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.routetable.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
}

func TestCreateOrUpdate_MutableDriftTriggersUpdate(t *testing.T) {
	var captured coresdk.UpdateRouteTableRequest
	manager := newRouteTableTestManager(&fakeRouteTableOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetRouteTableRequest) (coresdk.GetRouteTableResponse, error) {
			current := makeSDKRouteTable("ocid1.routetable.oc1..existing", "old-name", coresdk.RouteTableLifecycleStateAvailable)
			current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "41"}}
			current.FreeformTags = map[string]string{"env": "stage"}
			current.RouteRules = []coresdk.RouteRule{
				{
					NetworkEntityId: common.String("ocid1.natgateway.oc1..example"),
					Destination:     common.String("0.0.0.0/0"),
					DestinationType: coresdk.RouteRuleDestinationTypeCidrBlock,
					Description:     common.String("old internet access"),
					RouteType:       coresdk.RouteRuleRouteTypeStatic,
				},
			}
			return coresdk.GetRouteTableResponse{RouteTable: current}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateRouteTableRequest) (coresdk.UpdateRouteTableResponse, error) {
			captured = req
			updated := makeSDKRouteTable("ocid1.routetable.oc1..existing", "new-name", coresdk.RouteTableLifecycleStateAvailable)
			return coresdk.UpdateRouteTableResponse{RouteTable: updated}, nil
		},
	})

	resource := makeSpecRouteTable()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.routetable.oc1..existing")
	resource.Spec.DisplayName = "new-name"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "ocid1.routetable.oc1..existing", *captured.RtId)
	assert.Equal(t, "new-name", *captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, "ocid1.internetgateway.oc1..example", *captured.RouteRules[0].NetworkEntityId)
	assert.Len(t, captured.RouteRules, 2)
	assert.Equal(t, "new-name", resource.Status.DisplayName)
}

func TestCreateOrUpdate_RejectsImmutableDrift(t *testing.T) {
	tests := []struct {
		name        string
		mutateSpec  func(*corev1beta1.RouteTable)
		expectField string
	}{
		{
			name: "compartmentId",
			mutateSpec: func(resource *corev1beta1.RouteTable) {
				resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
			},
			expectField: "compartmentId",
		},
		{
			name: "vcnId",
			mutateSpec: func(resource *corev1beta1.RouteTable) {
				resource.Spec.VcnId = "ocid1.vcn.oc1..different"
			},
			expectField: "vcnId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newRouteTableTestManager(&fakeRouteTableOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetRouteTableRequest) (coresdk.GetRouteTableResponse, error) {
					return coresdk.GetRouteTableResponse{
						RouteTable: makeSDKRouteTable("ocid1.routetable.oc1..existing", "test-route-table", coresdk.RouteTableLifecycleStateAvailable),
					}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateRouteTableRequest) (coresdk.UpdateRouteTableResponse, error) {
					updateCalls++
					return coresdk.UpdateRouteTableResponse{}, nil
				},
			})

			resource := makeSpecRouteTable()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.routetable.oc1..existing")
			tt.mutateSpec(resource)

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.Error(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.Contains(t, err.Error(), tt.expectField)
			assert.Equal(t, 0, updateCalls)
		})
	}
}

func TestCreateOrUpdate_RouteRuleNormalizationAvoidsSpuriousUpdate(t *testing.T) {
	updateCalls := 0
	manager := newRouteTableTestManager(&fakeRouteTableOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetRouteTableRequest) (coresdk.GetRouteTableResponse, error) {
			current := makeSDKRouteTable("ocid1.routetable.oc1..existing", "test-route-table", coresdk.RouteTableLifecycleStateAvailable)
			current.RouteRules = []coresdk.RouteRule{
				{
					NetworkEntityId: common.String("ocid1.localpeeringgateway.oc1..implicit"),
					Destination:     common.String("10.0.0.0/16"),
					DestinationType: coresdk.RouteRuleDestinationTypeCidrBlock,
					Description:     common.String("implicit local route"),
					RouteType:       coresdk.RouteRuleRouteTypeLocal,
				},
				current.RouteRules[1],
				current.RouteRules[0],
			}
			return coresdk.GetRouteTableResponse{RouteTable: current}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateRouteTableRequest) (coresdk.UpdateRouteTableResponse, error) {
			updateCalls++
			return coresdk.UpdateRouteTableResponse{}, nil
		},
	})

	resource := makeSpecRouteTable()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.routetable.oc1..existing")
	resource.Spec.RouteRules = []corev1beta1.RouteTableRouteRule{
		resource.Spec.RouteRules[1],
		resource.Spec.RouteRules[0],
	}

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 0, updateCalls)
}

func TestDelete_ConfirmsDeletionOnNotFound(t *testing.T) {
	manager := newRouteTableTestManager(&fakeRouteTableOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteRouteTableRequest) (coresdk.DeleteRouteTableResponse, error) {
			assert.Equal(t, "ocid1.routetable.oc1..delete", *req.RtId)
			return coresdk.DeleteRouteTableResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetRouteTableRequest) (coresdk.GetRouteTableResponse, error) {
			return coresdk.GetRouteTableResponse{}, fakeRouteTableServiceError{statusCode: 404, code: "NotFound", message: "not found"}
		},
	})

	resource := makeSpecRouteTable()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.routetable.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, done)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_KeepsFinalizerWhileResourceStillExists(t *testing.T) {
	manager := newRouteTableTestManager(&fakeRouteTableOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteRouteTableRequest) (coresdk.DeleteRouteTableResponse, error) {
			return coresdk.DeleteRouteTableResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetRouteTableRequest) (coresdk.GetRouteTableResponse, error) {
			return coresdk.GetRouteTableResponse{
				RouteTable: makeSDKRouteTable("ocid1.routetable.oc1..delete", "test-route-table", coresdk.RouteTableLifecycleStateTerminating),
			}, nil
		},
	})

	resource := makeSpecRouteTable()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.routetable.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, done)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestWithClient_AllowsInjectedFakeRuntimeClient(t *testing.T) {
	manager := newRouteTableTestManager(&fakeRouteTableOCIClient{})

	_, ok := manager.client.(*routeTableRuntimeClient)
	assert.True(t, ok)
}

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package natgateway

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

type fakeNatGatewayOCIClient struct {
	createFn func(context.Context, coresdk.CreateNatGatewayRequest) (coresdk.CreateNatGatewayResponse, error)
	getFn    func(context.Context, coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error)
	updateFn func(context.Context, coresdk.UpdateNatGatewayRequest) (coresdk.UpdateNatGatewayResponse, error)
	deleteFn func(context.Context, coresdk.DeleteNatGatewayRequest) (coresdk.DeleteNatGatewayResponse, error)
}

func (f *fakeNatGatewayOCIClient) CreateNatGateway(ctx context.Context, req coresdk.CreateNatGatewayRequest) (coresdk.CreateNatGatewayResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return coresdk.CreateNatGatewayResponse{}, nil
}

func (f *fakeNatGatewayOCIClient) GetNatGateway(ctx context.Context, req coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return coresdk.GetNatGatewayResponse{}, nil
}

func (f *fakeNatGatewayOCIClient) UpdateNatGateway(ctx context.Context, req coresdk.UpdateNatGatewayRequest) (coresdk.UpdateNatGatewayResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return coresdk.UpdateNatGatewayResponse{}, nil
}

func (f *fakeNatGatewayOCIClient) DeleteNatGateway(ctx context.Context, req coresdk.DeleteNatGatewayRequest) (coresdk.DeleteNatGatewayResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return coresdk.DeleteNatGatewayResponse{}, nil
}

type fakeNatGatewayServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeNatGatewayServiceError) Error() string          { return f.message }
func (f fakeNatGatewayServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeNatGatewayServiceError) GetMessage() string     { return f.message }
func (f fakeNatGatewayServiceError) GetCode() string        { return f.code }
func (f fakeNatGatewayServiceError) GetOpcRequestID() string {
	return ""
}

func newNatGatewayTestManager(client natGatewayOCIClient) *NatGatewayServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewNatGatewayServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&natGatewayRuntimeClient{
			manager: manager,
			client:  client,
		})
	}
	return manager
}

func makeSpecNatGateway() *corev1beta1.NatGateway {
	return &corev1beta1.NatGateway{
		Spec: corev1beta1.NatGatewaySpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			VcnId:         "ocid1.vcn.oc1..example",
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			DisplayName:   "test-nat-gateway",
			FreeformTags:  map[string]string{"env": "dev"},
			BlockTraffic:  true,
			PublicIpId:    "ocid1.publicip.oc1..example",
			RouteTableId:  "ocid1.routetable.oc1..example",
		},
	}
}

func makeSDKNatGateway(id, displayName string, state coresdk.NatGatewayLifecycleStateEnum) coresdk.NatGateway {
	return coresdk.NatGateway{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		VcnId:          common.String("ocid1.vcn.oc1..example"),
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		DisplayName:    common.String(displayName),
		FreeformTags:   map[string]string{"env": "dev"},
		BlockTraffic:   common.Bool(true),
		PublicIpId:     common.String("ocid1.publicip.oc1..example"),
		RouteTableId:   common.String("ocid1.routetable.oc1..example"),
		NatIp:          common.String("203.0.113.10"),
		TimeCreated:    &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		LifecycleState: state,
	}
}

func TestCreateOrUpdate_CreateSuccessAndStatusProjection(t *testing.T) {
	var captured coresdk.CreateNatGatewayRequest
	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
		createFn: func(_ context.Context, req coresdk.CreateNatGatewayRequest) (coresdk.CreateNatGatewayResponse, error) {
			captured = req
			return coresdk.CreateNatGatewayResponse{
				NatGateway: makeSDKNatGateway("ocid1.natgateway.oc1..create", "test-nat-gateway", coresdk.NatGatewayLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecNatGateway()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CompartmentId)
	assert.Equal(t, common.String("ocid1.vcn.oc1..example"), captured.VcnId)
	assert.Equal(t, common.String("test-nat-gateway"), captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, common.Bool(true), captured.BlockTraffic)
	assert.Equal(t, common.String("ocid1.publicip.oc1..example"), captured.PublicIpId)
	assert.Equal(t, common.String("ocid1.routetable.oc1..example"), captured.RouteTableId)
	assert.Equal(t, "ocid1.natgateway.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ocid1.natgateway.oc1..create", resource.Status.Id)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, "203.0.113.10", resource.Status.NatIp)
	assert.Equal(t, "2026-04-01T00:00:00Z", resource.Status.TimeCreated)
}

func TestCreateOrUpdate_ObserveByStatusOCID_NoOpWhenStateMatches(t *testing.T) {
	getCalls := 0
	updateCalls := 0
	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
		getFn: func(_ context.Context, req coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.natgateway.oc1..existing", *req.NatGatewayId)
			return coresdk.GetNatGatewayResponse{
				NatGateway: makeSDKNatGateway("ocid1.natgateway.oc1..existing", "test-nat-gateway", coresdk.NatGatewayLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateNatGatewayRequest) (coresdk.UpdateNatGatewayResponse, error) {
			updateCalls++
			return coresdk.UpdateNatGatewayResponse{}, nil
		},
	})

	resource := makeSpecNatGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.natgateway.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
}

func TestCreateOrUpdate_ClearsStaleOptionalStatusFieldsOnProjection(t *testing.T) {
	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
			current := makeSDKNatGateway("ocid1.natgateway.oc1..existing", "test-nat-gateway", coresdk.NatGatewayLifecycleStateAvailable)
			current.DisplayName = nil
			current.DefinedTags = nil
			current.FreeformTags = nil
			current.PublicIpId = nil
			current.RouteTableId = nil
			current.NatIp = nil
			current.TimeCreated = nil
			return coresdk.GetNatGatewayResponse{NatGateway: current}, nil
		},
	})

	resource := makeSpecNatGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.natgateway.oc1..existing")
	resource.Spec.DisplayName = ""
	resource.Spec.DefinedTags = nil
	resource.Spec.FreeformTags = nil
	resource.Spec.PublicIpId = ""
	resource.Spec.RouteTableId = ""
	resource.Status.DisplayName = "stale-name"
	resource.Status.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Status.FreeformTags = map[string]string{"env": "stale"}
	resource.Status.PublicIpId = "ocid1.publicip.oc1..stale"
	resource.Status.RouteTableId = "ocid1.routetable.oc1..stale"
	resource.Status.NatIp = "198.51.100.10"
	resource.Status.TimeCreated = "2026-04-01T00:00:00Z"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "", resource.Status.DisplayName)
	assert.Nil(t, resource.Status.DefinedTags)
	assert.Nil(t, resource.Status.FreeformTags)
	assert.Equal(t, "", resource.Status.PublicIpId)
	assert.Equal(t, "", resource.Status.RouteTableId)
	assert.Equal(t, "", resource.Status.NatIp)
	assert.Equal(t, "", resource.Status.TimeCreated)
}

func TestCreateOrUpdate_MutableDriftTriggersUpdate(t *testing.T) {
	var captured coresdk.UpdateNatGatewayRequest
	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
			current := makeSDKNatGateway("ocid1.natgateway.oc1..existing", "old-name", coresdk.NatGatewayLifecycleStateAvailable)
			current.BlockTraffic = common.Bool(false)
			current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "41"}}
			current.FreeformTags = map[string]string{"env": "stage"}
			current.RouteTableId = common.String("ocid1.routetable.oc1..old")
			return coresdk.GetNatGatewayResponse{NatGateway: current}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateNatGatewayRequest) (coresdk.UpdateNatGatewayResponse, error) {
			captured = req
			updated := makeSDKNatGateway("ocid1.natgateway.oc1..existing", "new-name", coresdk.NatGatewayLifecycleStateAvailable)
			updated.BlockTraffic = common.Bool(true)
			return coresdk.UpdateNatGatewayResponse{NatGateway: updated}, nil
		},
	})

	resource := makeSpecNatGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.natgateway.oc1..existing")
	resource.Spec.DisplayName = "new-name"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "ocid1.natgateway.oc1..existing", *captured.NatGatewayId)
	assert.Equal(t, "new-name", *captured.DisplayName)
	assert.Equal(t, true, *captured.BlockTraffic)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, "ocid1.routetable.oc1..example", *captured.RouteTableId)
	assert.Equal(t, "new-name", resource.Status.DisplayName)
	assert.True(t, resource.Status.BlockTraffic)
}

func TestCreateOrUpdate_ClearingMutableFieldsTriggersUpdate(t *testing.T) {
	var captured coresdk.UpdateNatGatewayRequest
	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
			return coresdk.GetNatGatewayResponse{
				NatGateway: makeSDKNatGateway("ocid1.natgateway.oc1..existing", "old-name", coresdk.NatGatewayLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateNatGatewayRequest) (coresdk.UpdateNatGatewayResponse, error) {
			captured = req
			updated := makeSDKNatGateway("ocid1.natgateway.oc1..existing", "", coresdk.NatGatewayLifecycleStateAvailable)
			updated.DisplayName = common.String("")
			updated.DefinedTags = map[string]map[string]interface{}{}
			updated.FreeformTags = map[string]string{}
			updated.RouteTableId = common.String("")
			return coresdk.UpdateNatGatewayResponse{NatGateway: updated}, nil
		},
	})

	resource := makeSpecNatGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.natgateway.oc1..existing")
	resource.Spec.DisplayName = ""
	resource.Spec.DefinedTags = nil
	resource.Spec.FreeformTags = nil
	resource.Spec.RouteTableId = ""

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "", *captured.DisplayName)
	assert.Equal(t, map[string]map[string]interface{}{}, captured.DefinedTags)
	assert.Equal(t, map[string]string{}, captured.FreeformTags)
	assert.Equal(t, "", *captured.RouteTableId)
	assert.Equal(t, "", resource.Status.DisplayName)
	assert.Len(t, resource.Status.DefinedTags, 0)
	assert.Len(t, resource.Status.FreeformTags, 0)
	assert.Equal(t, "", resource.Status.RouteTableId)
}

func TestCreateOrUpdate_RejectsImmutableDrift(t *testing.T) {
	tests := []struct {
		name        string
		mutateSpec  func(*corev1beta1.NatGateway)
		expectField string
	}{
		{
			name: "compartmentId",
			mutateSpec: func(resource *corev1beta1.NatGateway) {
				resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
			},
			expectField: "compartmentId",
		},
		{
			name: "vcnId",
			mutateSpec: func(resource *corev1beta1.NatGateway) {
				resource.Spec.VcnId = "ocid1.vcn.oc1..different"
			},
			expectField: "vcnId",
		},
		{
			name: "publicIpId",
			mutateSpec: func(resource *corev1beta1.NatGateway) {
				resource.Spec.PublicIpId = "ocid1.publicip.oc1..different"
			},
			expectField: "publicIpId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
					return coresdk.GetNatGatewayResponse{
						NatGateway: makeSDKNatGateway("ocid1.natgateway.oc1..existing", "test-nat-gateway", coresdk.NatGatewayLifecycleStateAvailable),
					}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateNatGatewayRequest) (coresdk.UpdateNatGatewayResponse, error) {
					updateCalls++
					return coresdk.UpdateNatGatewayResponse{}, nil
				},
			})

			resource := makeSpecNatGateway()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.natgateway.oc1..existing")
			tt.mutateSpec(resource)

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.Error(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.Contains(t, err.Error(), tt.expectField)
			assert.Equal(t, 0, updateCalls)
		})
	}
}

func TestCreateOrUpdate_RetryableStates(t *testing.T) {
	tests := []struct {
		name   string
		state  coresdk.NatGatewayLifecycleStateEnum
		reason shared.OSOKConditionType
	}{
		{name: "provisioning", state: coresdk.NatGatewayLifecycleStateProvisioning, reason: shared.Provisioning},
		{name: "terminating", state: coresdk.NatGatewayLifecycleStateTerminating, reason: shared.Terminating},
		{name: "terminated", state: coresdk.NatGatewayLifecycleStateTerminated, reason: shared.Terminating},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
					current := makeSDKNatGateway("ocid1.natgateway.oc1..existing", "old-name", tt.state)
					current.RouteTableId = common.String("ocid1.routetable.oc1..old")
					return coresdk.GetNatGatewayResponse{NatGateway: current}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateNatGatewayRequest) (coresdk.UpdateNatGatewayResponse, error) {
					updateCalls++
					return coresdk.UpdateNatGatewayResponse{}, nil
				},
			})

			resource := makeSpecNatGateway()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.natgateway.oc1..existing")
			resource.Spec.DisplayName = "new-name"

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.NoError(t, err)
			assert.True(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
			assert.Equal(t, natGatewayRequeueDuration, resp.RequeueDuration)
			assert.Equal(t, string(tt.reason), resource.Status.OsokStatus.Reason)
			assert.Equal(t, 0, updateCalls)
		})
	}
}

func TestCreateOrUpdate_RecreatesOnNotFound(t *testing.T) {
	getCalls := 0
	createCalls := 0
	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
			getCalls++
			return coresdk.GetNatGatewayResponse{}, fakeNatGatewayServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
		createFn: func(_ context.Context, req coresdk.CreateNatGatewayRequest) (coresdk.CreateNatGatewayResponse, error) {
			createCalls++
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			return coresdk.CreateNatGatewayResponse{
				NatGateway: makeSDKNatGateway("ocid1.natgateway.oc1..recreated", "test-nat-gateway", coresdk.NatGatewayLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecNatGateway()
	resource.Status.Id = "ocid1.natgateway.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.natgateway.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, shared.OCID("ocid1.natgateway.oc1..recreated"), resource.Status.OsokStatus.Ocid)
	assert.Equal(t, "ocid1.natgateway.oc1..recreated", resource.Status.Id)
}

func TestDelete_ConfirmsDeletionOnNotFound(t *testing.T) {
	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteNatGatewayRequest) (coresdk.DeleteNatGatewayResponse, error) {
			assert.Equal(t, "ocid1.natgateway.oc1..delete", *req.NatGatewayId)
			return coresdk.DeleteNatGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
			return coresdk.GetNatGatewayResponse{}, fakeNatGatewayServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
	})

	resource := makeSpecNatGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.natgateway.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_KeepsFinalizerWhileObservedTerminating(t *testing.T) {
	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteNatGatewayRequest) (coresdk.DeleteNatGatewayResponse, error) {
			return coresdk.DeleteNatGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
			return coresdk.GetNatGatewayResponse{
				NatGateway: makeSDKNatGateway("ocid1.natgateway.oc1..delete", "test-nat-gateway", coresdk.NatGatewayLifecycleStateTerminating),
			}, nil
		},
	})

	resource := makeSpecNatGateway()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.natgateway.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, "TERMINATING", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestIsNatGatewayReadNotFoundOCI_RejectsAuthAmbiguity(t *testing.T) {
	assert.True(t, isNatGatewayReadNotFoundOCI(errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "normalized not found",
	}))
	assert.False(t, isNatGatewayReadNotFoundOCI(errorutil.UnauthorizedAndNotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotAuthorizedOrNotFound,
		Description:    "normalized auth ambiguity",
	}))
	assert.False(t, isNatGatewayReadNotFoundOCI(fakeNatGatewayServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isNatGatewayReadNotFoundOCI(fakeNatGatewayServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
	assert.False(t, isNatGatewayReadNotFoundOCI(fakeNatGatewayServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not found",
	}))
	assert.False(t, isNatGatewayReadNotFoundOCI(errorutil.ConflictOciError{
		HTTPStatusCode: 409,
		ErrorCode:      errorutil.IncorrectState,
		Description:    "normalized conflict",
	}))
	assert.False(t, isNatGatewayReadNotFoundOCI(fakeNatGatewayServiceError{
		statusCode: 409,
		code:       errorutil.IncorrectState,
		message:    "resource conflict",
	}))
}

func TestIsNatGatewayDeleteNotFoundOCI_AcceptsAuthShaped404(t *testing.T) {
	assert.True(t, isNatGatewayDeleteNotFoundOCI(errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "normalized not found",
	}))
	assert.True(t, isNatGatewayDeleteNotFoundOCI(errorutil.UnauthorizedAndNotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotAuthorizedOrNotFound,
		Description:    "normalized auth ambiguity",
	}))
	assert.True(t, isNatGatewayDeleteNotFoundOCI(fakeNatGatewayServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isNatGatewayDeleteNotFoundOCI(fakeNatGatewayServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
	assert.False(t, isNatGatewayDeleteNotFoundOCI(fakeNatGatewayServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not found",
	}))
	assert.False(t, isNatGatewayDeleteNotFoundOCI(errorutil.ConflictOciError{
		HTTPStatusCode: 409,
		ErrorCode:      errorutil.IncorrectState,
		Description:    "normalized conflict",
	}))
	assert.False(t, isNatGatewayDeleteNotFoundOCI(fakeNatGatewayServiceError{
		statusCode: 409,
		code:       errorutil.IncorrectState,
		message:    "resource conflict",
	}))
}

func TestReconcileDelete_ReleasesFinalizerOnAuthShapedNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.NatGateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "NatGateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-nat-gateway-auth-shaped-404",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.NatGatewayStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.natgateway.oc1..delete"),
			},
		},
	}

	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteNatGatewayRequest) (coresdk.DeleteNatGatewayResponse, error) {
			assert.Equal(t, "ocid1.natgateway.oc1..delete", *req.NatGatewayId)
			return coresdk.DeleteNatGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
			return coresdk.GetNatGatewayResponse{}, fakeNatGatewayServiceError{
				statusCode: 404,
				code:       errorutil.NotAuthorizedOrNotFound,
				message:    "not authorized or not found",
			}
		},
	})

	kubeClient := newMemoryNatGatewayClient(scheme, resource)
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
		NamespacedName: ctrlclient.ObjectKey{Name: "test-nat-gateway-auth-shaped-404", Namespace: "default"},
	}, &corev1beta1.NatGateway{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredNatGateway(), osokcore.OSOKFinalizerName))

	events := drainNatGatewayEvents(recorder)
	assertNatGatewayEventContains(t, events, "Removed finalizer")
	assertNoNatGatewayEventContains(t, events, "Failed to delete resource")
}

func TestReconcileDelete_ReleasesFinalizerOnUnambiguousNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.NatGateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "NatGateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-nat-gateway",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.NatGatewayStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.natgateway.oc1..delete"),
			},
		},
	}

	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteNatGatewayRequest) (coresdk.DeleteNatGatewayResponse, error) {
			assert.Equal(t, "ocid1.natgateway.oc1..delete", *req.NatGatewayId)
			return coresdk.DeleteNatGatewayResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetNatGatewayRequest) (coresdk.GetNatGatewayResponse, error) {
			return coresdk.GetNatGatewayResponse{}, fakeNatGatewayServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "resource not found",
			}
		},
	})

	kubeClient := newMemoryNatGatewayClient(scheme, resource)
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
		NamespacedName: ctrlclient.ObjectKey{Name: "test-nat-gateway", Namespace: "default"},
	}, &corev1beta1.NatGateway{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredNatGateway(), osokcore.OSOKFinalizerName))

	events := drainNatGatewayEvents(recorder)
	assertNatGatewayEventContains(t, events, "Removed finalizer")
	assertNoNatGatewayEventContains(t, events, "Failed to delete resource")
}

func TestWithClient_AllowsInjectedFakeRuntimeClient(t *testing.T) {
	manager := newNatGatewayTestManager(&fakeNatGatewayOCIClient{})

	_, ok := manager.client.(*natGatewayRuntimeClient)
	assert.True(t, ok)
}

func drainNatGatewayEvents(recorder *record.FakeRecorder) []string {
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

func assertNatGatewayEventContains(t *testing.T, events []string, want string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, want) {
			return
		}
	}
	t.Fatalf("events %v do not contain %q", events, want)
}

func assertNoNatGatewayEventContains(t *testing.T, events []string, unexpected string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, unexpected) {
			t.Fatalf("events %v unexpectedly contain %q", events, unexpected)
		}
	}
}

type memoryNatGatewayClient struct {
	ctrlclient.Client
	stored ctrlclient.Object
}

func newMemoryNatGatewayClient(scheme *runtime.Scheme, obj ctrlclient.Object) *memoryNatGatewayClient {
	return &memoryNatGatewayClient{
		Client: ctrlclientfake.NewClientBuilder().WithScheme(scheme).Build(),
		stored: obj.DeepCopyObject().(ctrlclient.Object),
	}
}

func (c *memoryNatGatewayClient) Get(_ context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, _ ...ctrlclient.GetOption) error {
	if c.stored == nil || c.stored.GetName() != key.Name || c.stored.GetNamespace() != key.Namespace {
		return apierrors.NewNotFound(schema.GroupResource{Group: "core.oracle.com", Resource: "natgateways"}, key.Name)
	}

	value := reflect.ValueOf(obj)
	source := reflect.ValueOf(c.stored.DeepCopyObject())
	if value.Kind() != reflect.Ptr || source.Kind() != reflect.Ptr {
		return stderrors.New("memory client requires pointer objects")
	}
	value.Elem().Set(source.Elem())
	return nil
}

func (c *memoryNatGatewayClient) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.UpdateOption) error {
	c.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (c *memoryNatGatewayClient) StoredNatGateway() *corev1beta1.NatGateway {
	return c.stored.DeepCopyObject().(*corev1beta1.NatGateway)
}

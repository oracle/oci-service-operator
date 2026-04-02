/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networksecuritygroup

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

type fakeNetworkSecurityGroupOCIClient struct {
	createFn func(context.Context, coresdk.CreateNetworkSecurityGroupRequest) (coresdk.CreateNetworkSecurityGroupResponse, error)
	getFn    func(context.Context, coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error)
	updateFn func(context.Context, coresdk.UpdateNetworkSecurityGroupRequest) (coresdk.UpdateNetworkSecurityGroupResponse, error)
	deleteFn func(context.Context, coresdk.DeleteNetworkSecurityGroupRequest) (coresdk.DeleteNetworkSecurityGroupResponse, error)
}

func (f *fakeNetworkSecurityGroupOCIClient) CreateNetworkSecurityGroup(ctx context.Context, req coresdk.CreateNetworkSecurityGroupRequest) (coresdk.CreateNetworkSecurityGroupResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return coresdk.CreateNetworkSecurityGroupResponse{}, nil
}

func (f *fakeNetworkSecurityGroupOCIClient) GetNetworkSecurityGroup(ctx context.Context, req coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return coresdk.GetNetworkSecurityGroupResponse{}, nil
}

func (f *fakeNetworkSecurityGroupOCIClient) UpdateNetworkSecurityGroup(ctx context.Context, req coresdk.UpdateNetworkSecurityGroupRequest) (coresdk.UpdateNetworkSecurityGroupResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return coresdk.UpdateNetworkSecurityGroupResponse{}, nil
}

func (f *fakeNetworkSecurityGroupOCIClient) DeleteNetworkSecurityGroup(ctx context.Context, req coresdk.DeleteNetworkSecurityGroupRequest) (coresdk.DeleteNetworkSecurityGroupResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return coresdk.DeleteNetworkSecurityGroupResponse{}, nil
}

type fakeNetworkSecurityGroupServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeNetworkSecurityGroupServiceError) Error() string          { return f.message }
func (f fakeNetworkSecurityGroupServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeNetworkSecurityGroupServiceError) GetMessage() string     { return f.message }
func (f fakeNetworkSecurityGroupServiceError) GetCode() string        { return f.code }
func (f fakeNetworkSecurityGroupServiceError) GetOpcRequestID() string {
	return ""
}

func newNetworkSecurityGroupTestManager(client networkSecurityGroupOCIClient) *NetworkSecurityGroupServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewNetworkSecurityGroupServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&networkSecurityGroupRuntimeClient{
			manager: manager,
			client:  client,
		})
	}
	return manager
}

func makeSpecNetworkSecurityGroup() *corev1beta1.NetworkSecurityGroup {
	return &corev1beta1.NetworkSecurityGroup{
		Spec: corev1beta1.NetworkSecurityGroupSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			VcnId:         "ocid1.vcn.oc1..example",
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			DisplayName:   "test-network-security-group",
			FreeformTags:  map[string]string{"env": "dev"},
		},
	}
}

func makeSDKNetworkSecurityGroup(id, displayName string, state coresdk.NetworkSecurityGroupLifecycleStateEnum) coresdk.NetworkSecurityGroup {
	return coresdk.NetworkSecurityGroup{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		VcnId:          common.String("ocid1.vcn.oc1..example"),
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		DisplayName:    common.String(displayName),
		FreeformTags:   map[string]string{"env": "dev"},
		TimeCreated:    &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		LifecycleState: state,
	}
}

func TestCreateOrUpdate_CreateSuccessAndStatusProjection(t *testing.T) {
	var captured coresdk.CreateNetworkSecurityGroupRequest
	manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
		createFn: func(_ context.Context, req coresdk.CreateNetworkSecurityGroupRequest) (coresdk.CreateNetworkSecurityGroupResponse, error) {
			captured = req
			return coresdk.CreateNetworkSecurityGroupResponse{
				NetworkSecurityGroup: makeSDKNetworkSecurityGroup("ocid1.networksecuritygroup.oc1..create", "test-network-security-group", coresdk.NetworkSecurityGroupLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecNetworkSecurityGroup()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CompartmentId)
	assert.Equal(t, common.String("ocid1.vcn.oc1..example"), captured.VcnId)
	assert.Equal(t, common.String("test-network-security-group"), captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, "ocid1.networksecuritygroup.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ocid1.networksecuritygroup.oc1..create", resource.Status.Id)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, "test-network-security-group", resource.Status.DisplayName)
	assert.Equal(t, "2026-04-01T00:00:00Z", resource.Status.TimeCreated)
}

func TestCreateOrUpdate_ObserveByStatusOCID_NoOpWhenStateMatches(t *testing.T) {
	getCalls := 0
	updateCalls := 0
	manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
		getFn: func(_ context.Context, req coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.networksecuritygroup.oc1..existing", *req.NetworkSecurityGroupId)
			return coresdk.GetNetworkSecurityGroupResponse{
				NetworkSecurityGroup: makeSDKNetworkSecurityGroup("ocid1.networksecuritygroup.oc1..existing", "test-network-security-group", coresdk.NetworkSecurityGroupLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateNetworkSecurityGroupRequest) (coresdk.UpdateNetworkSecurityGroupResponse, error) {
			updateCalls++
			return coresdk.UpdateNetworkSecurityGroupResponse{}, nil
		},
	})

	resource := makeSpecNetworkSecurityGroup()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.networksecuritygroup.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
}

func TestCreateOrUpdate_ClearingMutableFieldsTriggersUpdate(t *testing.T) {
	var captured coresdk.UpdateNetworkSecurityGroupRequest
	manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
			return coresdk.GetNetworkSecurityGroupResponse{
				NetworkSecurityGroup: makeSDKNetworkSecurityGroup("ocid1.networksecuritygroup.oc1..existing", "old-name", coresdk.NetworkSecurityGroupLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateNetworkSecurityGroupRequest) (coresdk.UpdateNetworkSecurityGroupResponse, error) {
			captured = req
			updated := makeSDKNetworkSecurityGroup("ocid1.networksecuritygroup.oc1..existing", "", coresdk.NetworkSecurityGroupLifecycleStateAvailable)
			updated.DisplayName = common.String("")
			updated.DefinedTags = map[string]map[string]interface{}{}
			updated.FreeformTags = map[string]string{}
			return coresdk.UpdateNetworkSecurityGroupResponse{NetworkSecurityGroup: updated}, nil
		},
	})

	resource := makeSpecNetworkSecurityGroup()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.networksecuritygroup.oc1..existing")
	resource.Spec.DisplayName = ""
	resource.Spec.DefinedTags = nil
	resource.Spec.FreeformTags = nil

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "", *captured.DisplayName)
	assert.Equal(t, map[string]map[string]interface{}{}, captured.DefinedTags)
	assert.Equal(t, map[string]string{}, captured.FreeformTags)
	assert.Equal(t, "", resource.Status.DisplayName)
	assert.Len(t, resource.Status.DefinedTags, 0)
	assert.Len(t, resource.Status.FreeformTags, 0)
}

func TestCreateOrUpdate_RejectsUnsupportedCreateOnlyDrift(t *testing.T) {
	tests := []struct {
		name        string
		mutateSpec  func(*corev1beta1.NetworkSecurityGroup)
		expectField string
	}{
		{
			name: "compartmentId",
			mutateSpec: func(resource *corev1beta1.NetworkSecurityGroup) {
				resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
			},
			expectField: "compartmentId",
		},
		{
			name: "vcnId",
			mutateSpec: func(resource *corev1beta1.NetworkSecurityGroup) {
				resource.Spec.VcnId = "ocid1.vcn.oc1..different"
			},
			expectField: "vcnId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
					return coresdk.GetNetworkSecurityGroupResponse{
						NetworkSecurityGroup: makeSDKNetworkSecurityGroup("ocid1.networksecuritygroup.oc1..existing", "test-network-security-group", coresdk.NetworkSecurityGroupLifecycleStateAvailable),
					}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateNetworkSecurityGroupRequest) (coresdk.UpdateNetworkSecurityGroupResponse, error) {
					updateCalls++
					return coresdk.UpdateNetworkSecurityGroupResponse{}, nil
				},
			})

			resource := makeSpecNetworkSecurityGroup()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.networksecuritygroup.oc1..existing")
			tt.mutateSpec(resource)

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.Error(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.Contains(t, err.Error(), tt.expectField)
			assert.Equal(t, 0, updateCalls)
		})
	}
}

func TestCreateOrUpdate_ClearsStaleOptionalStatusFieldsOnProjection(t *testing.T) {
	manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
			current := makeSDKNetworkSecurityGroup("ocid1.networksecuritygroup.oc1..existing", "test-network-security-group", coresdk.NetworkSecurityGroupLifecycleStateAvailable)
			current.DisplayName = nil
			current.DefinedTags = nil
			current.FreeformTags = nil
			current.TimeCreated = nil
			return coresdk.GetNetworkSecurityGroupResponse{NetworkSecurityGroup: current}, nil
		},
	})

	resource := makeSpecNetworkSecurityGroup()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.networksecuritygroup.oc1..existing")
	resource.Spec.DisplayName = ""
	resource.Spec.DefinedTags = nil
	resource.Spec.FreeformTags = nil
	resource.Status.DisplayName = "stale-name"
	resource.Status.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Status.FreeformTags = map[string]string{"env": "stale"}
	resource.Status.TimeCreated = "2026-04-01T00:00:00Z"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "", resource.Status.DisplayName)
	assert.Nil(t, resource.Status.DefinedTags)
	assert.Nil(t, resource.Status.FreeformTags)
	assert.Equal(t, "", resource.Status.TimeCreated)
}

func TestCreateOrUpdate_RetryableStates(t *testing.T) {
	tests := []struct {
		name   string
		state  coresdk.NetworkSecurityGroupLifecycleStateEnum
		reason shared.OSOKConditionType
	}{
		{name: "provisioning", state: coresdk.NetworkSecurityGroupLifecycleStateProvisioning, reason: shared.Provisioning},
		{name: "updating", state: networkSecurityGroupLifecycleStateUpdate, reason: shared.Provisioning},
		{name: "terminating", state: coresdk.NetworkSecurityGroupLifecycleStateTerminating, reason: shared.Terminating},
		{name: "terminated", state: coresdk.NetworkSecurityGroupLifecycleStateTerminated, reason: shared.Terminating},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
					return coresdk.GetNetworkSecurityGroupResponse{
						NetworkSecurityGroup: makeSDKNetworkSecurityGroup("ocid1.networksecuritygroup.oc1..existing", "test-network-security-group", tt.state),
					}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateNetworkSecurityGroupRequest) (coresdk.UpdateNetworkSecurityGroupResponse, error) {
					updateCalls++
					return coresdk.UpdateNetworkSecurityGroupResponse{}, nil
				},
			})

			resource := makeSpecNetworkSecurityGroup()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.networksecuritygroup.oc1..existing")
			resource.Spec.DisplayName = "new-name"

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.NoError(t, err)
			assert.True(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
			assert.Equal(t, networkSecurityGroupRequeueDuration, resp.RequeueDuration)
			assert.Equal(t, string(tt.reason), resource.Status.OsokStatus.Reason)
			assert.Equal(t, 0, updateCalls)
		})
	}
}

func TestCreateOrUpdate_RecreatesOnNotFound(t *testing.T) {
	getCalls := 0
	createCalls := 0
	manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
			getCalls++
			return coresdk.GetNetworkSecurityGroupResponse{}, fakeNetworkSecurityGroupServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
		createFn: func(_ context.Context, req coresdk.CreateNetworkSecurityGroupRequest) (coresdk.CreateNetworkSecurityGroupResponse, error) {
			createCalls++
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			return coresdk.CreateNetworkSecurityGroupResponse{
				NetworkSecurityGroup: makeSDKNetworkSecurityGroup("ocid1.networksecuritygroup.oc1..recreated", "test-network-security-group", coresdk.NetworkSecurityGroupLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecNetworkSecurityGroup()
	resource.Status.Id = "ocid1.networksecuritygroup.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.networksecuritygroup.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, shared.OCID("ocid1.networksecuritygroup.oc1..recreated"), resource.Status.OsokStatus.Ocid)
	assert.Equal(t, "ocid1.networksecuritygroup.oc1..recreated", resource.Status.Id)
}

func TestDelete_ConfirmsDeletionOnNotFound(t *testing.T) {
	manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteNetworkSecurityGroupRequest) (coresdk.DeleteNetworkSecurityGroupResponse, error) {
			assert.Equal(t, "ocid1.networksecuritygroup.oc1..delete", *req.NetworkSecurityGroupId)
			return coresdk.DeleteNetworkSecurityGroupResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
			return coresdk.GetNetworkSecurityGroupResponse{}, fakeNetworkSecurityGroupServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
	})

	resource := makeSpecNetworkSecurityGroup()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.networksecuritygroup.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_AlreadyMissingOCIResourceIsTreatedAsDeleted(t *testing.T) {
	manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteNetworkSecurityGroupRequest) (coresdk.DeleteNetworkSecurityGroupResponse, error) {
			assert.Equal(t, "ocid1.networksecuritygroup.oc1..delete", *req.NetworkSecurityGroupId)
			return coresdk.DeleteNetworkSecurityGroupResponse{}, fakeNetworkSecurityGroupServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
	})

	resource := makeSpecNetworkSecurityGroup()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.networksecuritygroup.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "OCI resource no longer exists", resource.Status.OsokStatus.Message)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_KeepsFinalizerWhileObservedTerminating(t *testing.T) {
	manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteNetworkSecurityGroupRequest) (coresdk.DeleteNetworkSecurityGroupResponse, error) {
			return coresdk.DeleteNetworkSecurityGroupResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
			return coresdk.GetNetworkSecurityGroupResponse{
				NetworkSecurityGroup: makeSDKNetworkSecurityGroup("ocid1.networksecuritygroup.oc1..delete", "test-network-security-group", coresdk.NetworkSecurityGroupLifecycleStateTerminating),
			}, nil
		},
	})

	resource := makeSpecNetworkSecurityGroup()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.networksecuritygroup.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, "TERMINATING", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestIsNetworkSecurityGroupReadNotFoundOCI_RejectsAuthAmbiguity(t *testing.T) {
	assert.True(t, isNetworkSecurityGroupReadNotFoundOCI(errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "normalized not found",
	}))
	assert.False(t, isNetworkSecurityGroupReadNotFoundOCI(errorutil.UnauthorizedAndNotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotAuthorizedOrNotFound,
		Description:    "normalized auth ambiguity",
	}))
	assert.False(t, isNetworkSecurityGroupReadNotFoundOCI(fakeNetworkSecurityGroupServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isNetworkSecurityGroupReadNotFoundOCI(fakeNetworkSecurityGroupServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
	assert.False(t, isNetworkSecurityGroupReadNotFoundOCI(fakeNetworkSecurityGroupServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not found",
	}))
	assert.False(t, isNetworkSecurityGroupReadNotFoundOCI(errorutil.ConflictOciError{
		HTTPStatusCode: 409,
		ErrorCode:      errorutil.IncorrectState,
		Description:    "normalized conflict",
	}))
	assert.False(t, isNetworkSecurityGroupReadNotFoundOCI(fakeNetworkSecurityGroupServiceError{
		statusCode: 409,
		code:       errorutil.IncorrectState,
		message:    "resource conflict",
	}))
}

func TestIsNetworkSecurityGroupDeleteNotFoundOCI_AcceptsAuthShaped404(t *testing.T) {
	assert.True(t, isNetworkSecurityGroupDeleteNotFoundOCI(errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "normalized not found",
	}))
	assert.True(t, isNetworkSecurityGroupDeleteNotFoundOCI(errorutil.UnauthorizedAndNotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotAuthorizedOrNotFound,
		Description:    "normalized auth ambiguity",
	}))
	assert.True(t, isNetworkSecurityGroupDeleteNotFoundOCI(fakeNetworkSecurityGroupServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isNetworkSecurityGroupDeleteNotFoundOCI(fakeNetworkSecurityGroupServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
	assert.False(t, isNetworkSecurityGroupDeleteNotFoundOCI(fakeNetworkSecurityGroupServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not found",
	}))
	assert.False(t, isNetworkSecurityGroupDeleteNotFoundOCI(errorutil.ConflictOciError{
		HTTPStatusCode: 409,
		ErrorCode:      errorutil.IncorrectState,
		Description:    "normalized conflict",
	}))
	assert.False(t, isNetworkSecurityGroupDeleteNotFoundOCI(fakeNetworkSecurityGroupServiceError{
		statusCode: 409,
		code:       errorutil.IncorrectState,
		message:    "resource conflict",
	}))
}

func TestReconcileDelete_ReleasesFinalizerOnAuthShapedNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.NetworkSecurityGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "NetworkSecurityGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-network-security-group-auth-shaped-404",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.NetworkSecurityGroupStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.networksecuritygroup.oc1..delete"),
			},
		},
	}

	manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteNetworkSecurityGroupRequest) (coresdk.DeleteNetworkSecurityGroupResponse, error) {
			assert.Equal(t, "ocid1.networksecuritygroup.oc1..delete", *req.NetworkSecurityGroupId)
			return coresdk.DeleteNetworkSecurityGroupResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
			return coresdk.GetNetworkSecurityGroupResponse{}, fakeNetworkSecurityGroupServiceError{
				statusCode: 404,
				code:       errorutil.NotAuthorizedOrNotFound,
				message:    "not authorized or not found",
			}
		},
	})

	kubeClient := newMemoryNetworkSecurityGroupClient(scheme, resource)
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
		NamespacedName: ctrlclient.ObjectKey{Name: "test-network-security-group-auth-shaped-404", Namespace: "default"},
	}, &corev1beta1.NetworkSecurityGroup{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredNetworkSecurityGroup(), osokcore.OSOKFinalizerName))

	events := drainNetworkSecurityGroupEvents(recorder)
	assertNetworkSecurityGroupEventContains(t, events, "Removed finalizer")
	assertNoNetworkSecurityGroupEventContains(t, events, "Failed to delete resource")
}

func TestReconcileDelete_ReleasesFinalizerOnUnambiguousNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.NetworkSecurityGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "NetworkSecurityGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-network-security-group",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.NetworkSecurityGroupStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.networksecuritygroup.oc1..delete"),
			},
		},
	}

	manager := newNetworkSecurityGroupTestManager(&fakeNetworkSecurityGroupOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteNetworkSecurityGroupRequest) (coresdk.DeleteNetworkSecurityGroupResponse, error) {
			assert.Equal(t, "ocid1.networksecuritygroup.oc1..delete", *req.NetworkSecurityGroupId)
			return coresdk.DeleteNetworkSecurityGroupResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error) {
			return coresdk.GetNetworkSecurityGroupResponse{}, fakeNetworkSecurityGroupServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "resource not found",
			}
		},
	})

	kubeClient := newMemoryNetworkSecurityGroupClient(scheme, resource)
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
		NamespacedName: ctrlclient.ObjectKey{Name: "test-network-security-group", Namespace: "default"},
	}, &corev1beta1.NetworkSecurityGroup{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredNetworkSecurityGroup(), osokcore.OSOKFinalizerName))

	events := drainNetworkSecurityGroupEvents(recorder)
	assertNetworkSecurityGroupEventContains(t, events, "Removed finalizer")
	assertNoNetworkSecurityGroupEventContains(t, events, "Failed to delete resource")
}

func drainNetworkSecurityGroupEvents(recorder *record.FakeRecorder) []string {
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

func assertNetworkSecurityGroupEventContains(t *testing.T, events []string, want string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, want) {
			return
		}
	}
	t.Fatalf("events %v do not contain %q", events, want)
}

func assertNoNetworkSecurityGroupEventContains(t *testing.T, events []string, unexpected string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, unexpected) {
			t.Fatalf("events %v unexpectedly contain %q", events, unexpected)
		}
	}
}

type memoryNetworkSecurityGroupClient struct {
	ctrlclient.Client
	stored ctrlclient.Object
}

func newMemoryNetworkSecurityGroupClient(scheme *runtime.Scheme, obj ctrlclient.Object) *memoryNetworkSecurityGroupClient {
	return &memoryNetworkSecurityGroupClient{
		Client: ctrlclientfake.NewClientBuilder().WithScheme(scheme).Build(),
		stored: obj.DeepCopyObject().(ctrlclient.Object),
	}
}

func (c *memoryNetworkSecurityGroupClient) Get(_ context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, _ ...ctrlclient.GetOption) error {
	if c.stored == nil || c.stored.GetName() != key.Name || c.stored.GetNamespace() != key.Namespace {
		return apierrors.NewNotFound(schema.GroupResource{Group: "core.oracle.com", Resource: "networksecuritygroups"}, key.Name)
	}

	value := reflect.ValueOf(obj)
	source := reflect.ValueOf(c.stored.DeepCopyObject())
	if value.Kind() != reflect.Ptr || source.Kind() != reflect.Ptr {
		return stderrors.New("memory client requires pointer objects")
	}
	value.Elem().Set(source.Elem())
	return nil
}

func (c *memoryNetworkSecurityGroupClient) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.UpdateOption) error {
	c.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (c *memoryNetworkSecurityGroupClient) StoredNetworkSecurityGroup() *corev1beta1.NetworkSecurityGroup {
	return c.stored.DeepCopyObject().(*corev1beta1.NetworkSecurityGroup)
}

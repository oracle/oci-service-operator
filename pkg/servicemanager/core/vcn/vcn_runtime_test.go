/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vcn

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeVcnOCIClient struct {
	createFn func(context.Context, coresdk.CreateVcnRequest) (coresdk.CreateVcnResponse, error)
	getFn    func(context.Context, coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error)
	updateFn func(context.Context, coresdk.UpdateVcnRequest) (coresdk.UpdateVcnResponse, error)
	deleteFn func(context.Context, coresdk.DeleteVcnRequest) (coresdk.DeleteVcnResponse, error)
}

func (f *fakeVcnOCIClient) CreateVcn(ctx context.Context, req coresdk.CreateVcnRequest) (coresdk.CreateVcnResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return coresdk.CreateVcnResponse{}, nil
}

func (f *fakeVcnOCIClient) GetVcn(ctx context.Context, req coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return coresdk.GetVcnResponse{}, nil
}

func (f *fakeVcnOCIClient) UpdateVcn(ctx context.Context, req coresdk.UpdateVcnRequest) (coresdk.UpdateVcnResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return coresdk.UpdateVcnResponse{}, nil
}

func (f *fakeVcnOCIClient) DeleteVcn(ctx context.Context, req coresdk.DeleteVcnRequest) (coresdk.DeleteVcnResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return coresdk.DeleteVcnResponse{}, nil
}

type fakeServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeServiceError) Error() string          { return f.message }
func (f fakeServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeServiceError) GetMessage() string     { return f.message }
func (f fakeServiceError) GetCode() string        { return f.code }
func (f fakeServiceError) GetOpcRequestID() string {
	return ""
}

func newTestManager(client vcnOCIClient) *VcnServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewVcnServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&vcnRuntimeClient{
			manager: manager,
			client:  client,
		})
	}
	return manager
}

func makeSpecVcn() *corev1beta1.Vcn {
	return &corev1beta1.Vcn{
		Spec: corev1beta1.VcnSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			CidrBlocks:    []string{"10.0.0.0/16"},
			DisplayName:   "test-vcn",
		},
	}
}

func makeSDKVcn(id, displayName string, state coresdk.VcnLifecycleStateEnum) coresdk.Vcn {
	return coresdk.Vcn{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		CidrBlock:      common.String("10.0.0.0/16"),
		CidrBlocks:     []string{"10.0.0.0/16"},
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {"CostCenter": "42"},
		},
		FreeformTags: map[string]string{"env": "dev"},
	}
}

func TestCreateOrUpdate_CreateSuccessAndStatusProjection(t *testing.T) {
	var captured coresdk.CreateVcnRequest
	manager := newTestManager(&fakeVcnOCIClient{
		createFn: func(_ context.Context, req coresdk.CreateVcnRequest) (coresdk.CreateVcnResponse, error) {
			captured = req
			return coresdk.CreateVcnResponse{
				Vcn: makeSDKVcn("ocid1.vcn.oc1..create", "test-vcn", coresdk.VcnLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecVcn()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CompartmentId)
	assert.Equal(t, []string{"10.0.0.0/16"}, captured.CidrBlocks)
	assert.Equal(t, common.String("test-vcn"), captured.DisplayName)
	assert.Nil(t, captured.DnsLabel)
	assert.Nil(t, captured.IsIpv6Enabled)
	assert.Equal(t, "ocid1.vcn.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, "test-vcn", resource.Status.DisplayName)
	assert.Equal(t, []string{"10.0.0.0/16"}, resource.Status.CidrBlocks)
}

func TestCreateOrUpdate_ObserveByStatusOCID(t *testing.T) {
	getCalls := 0
	updateCalls := 0
	manager := newTestManager(&fakeVcnOCIClient{
		getFn: func(_ context.Context, req coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.vcn.oc1..existing", *req.VcnId)
			return coresdk.GetVcnResponse{
				Vcn: makeSDKVcn("ocid1.vcn.oc1..existing", "test-vcn", coresdk.VcnLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateVcnRequest) (coresdk.UpdateVcnResponse, error) {
			updateCalls++
			return coresdk.UpdateVcnResponse{}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..existing")
	resource.Spec.DefinedTags = map[string]shared.MapValue{
		"Operations": {"CostCenter": "42"},
	}
	resource.Spec.FreeformTags = map[string]string{"env": "dev"}

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
}

func TestCreateOrUpdate_MutableDriftTriggersUpdate(t *testing.T) {
	var captured coresdk.UpdateVcnRequest
	manager := newTestManager(&fakeVcnOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			return coresdk.GetVcnResponse{
				Vcn: makeSDKVcn("ocid1.vcn.oc1..existing", "old-name", coresdk.VcnLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateVcnRequest) (coresdk.UpdateVcnResponse, error) {
			captured = req
			updated := makeSDKVcn("ocid1.vcn.oc1..existing", "new-name", coresdk.VcnLifecycleStateAvailable)
			return coresdk.UpdateVcnResponse{Vcn: updated}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..existing")
	resource.Spec.DisplayName = "new-name"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "ocid1.vcn.oc1..existing", *captured.VcnId)
	assert.Equal(t, "new-name", *captured.DisplayName)
	assert.Nil(t, captured.FreeformTags)
	assert.Equal(t, "new-name", resource.Status.DisplayName)
}

func TestCreateOrUpdate_RetryableStates(t *testing.T) {
	tests := []struct {
		name   string
		state  coresdk.VcnLifecycleStateEnum
		reason shared.OSOKConditionType
	}{
		{name: "provisioning", state: coresdk.VcnLifecycleStateProvisioning, reason: shared.Provisioning},
		{name: "updating", state: coresdk.VcnLifecycleStateUpdating, reason: shared.Updating},
		{name: "terminating", state: coresdk.VcnLifecycleStateTerminating, reason: shared.Terminating},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := newTestManager(&fakeVcnOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
					return coresdk.GetVcnResponse{
						Vcn: makeSDKVcn("ocid1.vcn.oc1..existing", "test-vcn", tt.state),
					}, nil
				},
			})

			resource := makeSpecVcn()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..existing")

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.NoError(t, err)
			assert.True(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
			assert.Equal(t, vcnRequeueDuration, resp.RequeueDuration)
			assert.Equal(t, string(tt.reason), resource.Status.OsokStatus.Reason)
		})
	}
}

func TestCreateOrUpdate_RejectsUnsupportedCreateOnlyDrift(t *testing.T) {
	updateCalls := 0
	manager := newTestManager(&fakeVcnOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			current := makeSDKVcn("ocid1.vcn.oc1..existing", "test-vcn", coresdk.VcnLifecycleStateAvailable)
			current.CidrBlocks = []string{"192.168.0.0/16"}
			current.CidrBlock = common.String("192.168.0.0/16")
			return coresdk.GetVcnResponse{Vcn: current}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateVcnRequest) (coresdk.UpdateVcnResponse, error) {
			updateCalls++
			return coresdk.UpdateVcnResponse{}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "create-only field drift")
	assert.Equal(t, 0, updateCalls)
}

func TestCreateOrUpdate_RecreatesOnExplicitNotFound(t *testing.T) {
	getCalls := 0
	createCalls := 0
	manager := newTestManager(&fakeVcnOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			getCalls++
			return coresdk.GetVcnResponse{}, fakeServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "missing",
			}
		},
		createFn: func(_ context.Context, req coresdk.CreateVcnRequest) (coresdk.CreateVcnResponse, error) {
			createCalls++
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			return coresdk.CreateVcnResponse{
				Vcn: makeSDKVcn("ocid1.vcn.oc1..recreated", "test-vcn", coresdk.VcnLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.Id = "ocid1.vcn.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..existing")
	resource.Status.OsokStatus.Message = "stale"
	oldCreatedAt := metav1.Now()
	resource.Status.OsokStatus.CreatedAt = &oldCreatedAt

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, "ocid1.vcn.oc1..recreated", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ocid1.vcn.oc1..recreated", resource.Status.Id)
	assert.NotNil(t, resource.Status.OsokStatus.CreatedAt)
	assert.NotEqual(t, oldCreatedAt, *resource.Status.OsokStatus.CreatedAt)
}

func TestCreateOrUpdate_DoesNotRecreateOnAuthAmbiguity(t *testing.T) {
	createCalls := 0
	manager := newTestManager(&fakeVcnOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			return coresdk.GetVcnResponse{}, fakeServiceError{
				statusCode: 404,
				code:       "NotAuthorizedOrNotFound",
				message:    "auth ambiguity",
			}
		},
		createFn: func(_ context.Context, _ coresdk.CreateVcnRequest) (coresdk.CreateVcnResponse, error) {
			createCalls++
			return coresdk.CreateVcnResponse{}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.Id = "ocid1.vcn.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Equal(t, 0, createCalls)
	assert.Equal(t, "ocid1.vcn.oc1..existing", string(resource.Status.OsokStatus.Ocid))
}

func TestDelete_ConfirmsDeletionOnNotFound(t *testing.T) {
	manager := newTestManager(&fakeVcnOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteVcnRequest) (coresdk.DeleteVcnResponse, error) {
			assert.Equal(t, "ocid1.vcn.oc1..delete", *req.VcnId)
			return coresdk.DeleteVcnResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			return coresdk.GetVcnResponse{}, fakeServiceError{statusCode: 404, code: "NotFound", message: "not found"}
		},
	})

	resource := makeSpecVcn()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, done)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_ConfirmsDeletionOnObservedTerminated(t *testing.T) {
	manager := newTestManager(&fakeVcnOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteVcnRequest) (coresdk.DeleteVcnResponse, error) {
			return coresdk.DeleteVcnResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			return coresdk.GetVcnResponse{
				Vcn: makeSDKVcn("ocid1.vcn.oc1..delete", "test-vcn", coresdk.VcnLifecycleStateTerminated),
			}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, done)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_DoesNotConfirmDeletionOnAuthAmbiguity(t *testing.T) {
	manager := newTestManager(&fakeVcnOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteVcnRequest) (coresdk.DeleteVcnResponse, error) {
			return coresdk.DeleteVcnResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			return coresdk.GetVcnResponse{}, fakeServiceError{
				statusCode: 404,
				code:       "NotAuthorizedOrNotFound",
				message:    "auth ambiguity",
			}
		},
	})

	resource := makeSpecVcn()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.Error(t, err)
	assert.False(t, done)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestIsNotFoundOCI_RejectsAuthAmbiguity(t *testing.T) {
	assert.False(t, isNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
}

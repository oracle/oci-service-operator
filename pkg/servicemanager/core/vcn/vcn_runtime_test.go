/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vcn

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

func TestCreateOrUpdate_RejectsConflictingCreateCIDRInputs(t *testing.T) {
	createCalls := 0
	manager := newTestManager(&fakeVcnOCIClient{
		createFn: func(_ context.Context, _ coresdk.CreateVcnRequest) (coresdk.CreateVcnResponse, error) {
			createCalls++
			return coresdk.CreateVcnResponse{}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Spec.CidrBlock = "10.0.0.0/16"
	resource.Spec.CidrBlocks = []string{"10.0.0.0/16"}

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "both cidrBlock and cidrBlocks")
	assert.Equal(t, 0, createCalls)
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

func TestCreateOrUpdate_ClearsStaleOptionalStatusFieldsOnProjection(t *testing.T) {
	manager := newTestManager(&fakeVcnOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			current := makeSDKVcn("ocid1.vcn.oc1..existing", "test-vcn", coresdk.VcnLifecycleStateAvailable)
			current.DisplayName = nil
			current.DefinedTags = nil
			current.FreeformTags = nil
			current.VcnDomainName = nil
			current.DefaultRouteTableId = nil
			return coresdk.GetVcnResponse{
				Vcn: current,
			}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..existing")
	resource.Spec.DisplayName = ""
	resource.Status.DisplayName = "stale-name"
	resource.Status.DnsLabel = "stale-dns"
	resource.Status.FreeformTags = map[string]string{"env": "stale"}
	resource.Status.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Status.Ipv6CidrBlocks = []string{"fd00::/56"}
	resource.Status.VcnDomainName = "stale.oraclevcn.com"
	resource.Status.DefaultRouteTableId = "ocid1.routetable.oc1..stale"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "", resource.Status.DisplayName)
	assert.Equal(t, "", resource.Status.DnsLabel)
	assert.Nil(t, resource.Status.FreeformTags)
	assert.Nil(t, resource.Status.DefinedTags)
	assert.Nil(t, resource.Status.Ipv6CidrBlocks)
	assert.Equal(t, "", resource.Status.VcnDomainName)
	assert.Equal(t, "", resource.Status.DefaultRouteTableId)
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

func TestCreateOrUpdate_DoesNotUpdateDuringRetryableLiveStates(t *testing.T) {
	tests := []struct {
		name  string
		state coresdk.VcnLifecycleStateEnum
	}{
		{name: "provisioning", state: coresdk.VcnLifecycleStateProvisioning},
		{name: "updating", state: coresdk.VcnLifecycleStateUpdating},
		{name: "terminating", state: coresdk.VcnLifecycleStateTerminating},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newTestManager(&fakeVcnOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
					current := makeSDKVcn("ocid1.vcn.oc1..existing", "old-name", tt.state)
					return coresdk.GetVcnResponse{Vcn: current}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateVcnRequest) (coresdk.UpdateVcnResponse, error) {
					updateCalls++
					return coresdk.UpdateVcnResponse{}, nil
				},
			})

			resource := makeSpecVcn()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..existing")
			resource.Spec.DisplayName = "new-name"

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.NoError(t, err)
			assert.True(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
			assert.Equal(t, 0, updateCalls)
		})
	}
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

func TestCreateOrUpdate_AllowsEquivalentReorderedCreateOnlyLists(t *testing.T) {
	updateCalls := 0
	manager := newTestManager(&fakeVcnOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			current := makeSDKVcn("ocid1.vcn.oc1..existing", "test-vcn", coresdk.VcnLifecycleStateAvailable)
			current.CidrBlocks = []string{"10.1.0.0/16", "10.0.0.0/16"}
			current.Ipv6PrivateCidrBlocks = []string{"fd00:2::/56", "fd00:1::/56"}
			current.Byoipv6CidrBlocks = []string{"2001:db8:2::/56", "2001:db8:1::/56"}
			return coresdk.GetVcnResponse{Vcn: current}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateVcnRequest) (coresdk.UpdateVcnResponse, error) {
			updateCalls++
			return coresdk.UpdateVcnResponse{}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..existing")
	resource.Spec.CidrBlocks = []string{"10.0.0.0/16", "10.1.0.0/16"}
	resource.Spec.Ipv6PrivateCidrBlocks = []string{"fd00:1::/56", "fd00:2::/56"}
	resource.Spec.Byoipv6CidrDetails = []corev1beta1.VcnByoipv6CidrDetail{
		{Byoipv6RangeId: "ocid1.byoipv6.oc1..a", Ipv6CidrBlock: "2001:db8:1::/56"},
		{Byoipv6RangeId: "ocid1.byoipv6.oc1..b", Ipv6CidrBlock: "2001:db8:2::/56"},
	}

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
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

func TestCreateOrUpdate_RecreateClearsStaleNestedOsokStatusMetadata(t *testing.T) {
	manager := newTestManager(&fakeVcnOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			return coresdk.GetVcnResponse{}, fakeServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "missing",
			}
		},
		createFn: func(_ context.Context, _ coresdk.CreateVcnRequest) (coresdk.CreateVcnResponse, error) {
			return coresdk.CreateVcnResponse{
				Vcn: makeSDKVcn("ocid1.vcn.oc1..recreated", "test-vcn", coresdk.VcnLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.Id = "ocid1.vcn.oc1..deleted"
	resource.Status.OsokStatus = shared.OSOKStatus{
		Ocid:      "ocid1.vcn.oc1..deleted",
		Message:   "old delete message",
		Reason:    string(shared.Terminating),
		DeletedAt: &metav1.Time{Time: time.Now()},
		Conditions: []shared.OSOKCondition{
			{Type: shared.Terminating, Status: "True"},
		},
	}

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, shared.OCID("ocid1.vcn.oc1..recreated"), resource.Status.OsokStatus.Ocid)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(shared.Active), resource.Status.OsokStatus.Reason)
	assert.Len(t, resource.Status.OsokStatus.Conditions, 1)
	assert.Equal(t, shared.Active, resource.Status.OsokStatus.Conditions[0].Type)
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

func TestCreateOrUpdate_RecreatesTrackedTerminatedVcn(t *testing.T) {
	createCalls := 0
	updateCalls := 0
	manager := newTestManager(&fakeVcnOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			return coresdk.GetVcnResponse{
				Vcn: makeSDKVcn("ocid1.vcn.oc1..terminated", "old-name", coresdk.VcnLifecycleStateTerminated),
			}, nil
		},
		createFn: func(_ context.Context, _ coresdk.CreateVcnRequest) (coresdk.CreateVcnResponse, error) {
			createCalls++
			return coresdk.CreateVcnResponse{
				Vcn: makeSDKVcn("ocid1.vcn.oc1..recreated", "test-vcn", coresdk.VcnLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateVcnRequest) (coresdk.UpdateVcnResponse, error) {
			updateCalls++
			return coresdk.UpdateVcnResponse{}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.Id = "ocid1.vcn.oc1..terminated"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..terminated")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, shared.OCID("ocid1.vcn.oc1..recreated"), resource.Status.OsokStatus.Ocid)
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

func TestDelete_KeepsFinalizerWhileObservedTerminated(t *testing.T) {
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
	assert.False(t, done)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestDelete_KeepsFinalizerWhileObservedTerminating(t *testing.T) {
	manager := newTestManager(&fakeVcnOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteVcnRequest) (coresdk.DeleteVcnResponse, error) {
			return coresdk.DeleteVcnResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			return coresdk.GetVcnResponse{
				Vcn: makeSDKVcn("ocid1.vcn.oc1..delete", "test-vcn", coresdk.VcnLifecycleStateTerminating),
			}, nil
		},
	})

	resource := makeSpecVcn()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vcn.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, done)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestDelete_ConfirmsDeletionOnAuthShapedNotFound(t *testing.T) {
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

	assert.NoError(t, err)
	assert.True(t, done)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestReconcileDelete_ReleasesFinalizerOnUnambiguousNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.Vcn{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "Vcn",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-vcn",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.VcnStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.vcn.oc1..delete"),
			},
		},
	}

	manager := newTestManager(&fakeVcnOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteVcnRequest) (coresdk.DeleteVcnResponse, error) {
			assert.Equal(t, "ocid1.vcn.oc1..delete", *req.VcnId)
			return coresdk.DeleteVcnResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error) {
			return coresdk.GetVcnResponse{}, fakeServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "resource not found",
			}
		},
	})

	kubeClient := newMemoryVcnClient(scheme, resource)
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
		NamespacedName: ctrlclient.ObjectKey{Name: "test-vcn", Namespace: "default"},
	}, &corev1beta1.Vcn{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredVcn(), osokcore.OSOKFinalizerName))

	events := drainVcnEvents(recorder)
	assertEventContains(t, events, "Removed finalizer")
	assertNoEventContains(t, events, "Failed to delete resource")
}

func TestReconcileDelete_ReleasesFinalizerOnAuthShapedNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.Vcn{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "Vcn",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-vcn-auth-shaped-404",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.VcnStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.vcn.oc1..delete"),
			},
		},
	}

	manager := newTestManager(&fakeVcnOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteVcnRequest) (coresdk.DeleteVcnResponse, error) {
			assert.Equal(t, "ocid1.vcn.oc1..delete", *req.VcnId)
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

	kubeClient := newMemoryVcnClient(scheme, resource)
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
		NamespacedName: ctrlclient.ObjectKey{Name: "test-vcn-auth-shaped-404", Namespace: "default"},
	}, &corev1beta1.Vcn{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredVcn(), osokcore.OSOKFinalizerName))

	events := drainVcnEvents(recorder)
	assertEventContains(t, events, "Removed finalizer")
	assertNoEventContains(t, events, "Failed to delete resource")
}

func TestIsNotFoundOCI_AcceptsAuthShaped404(t *testing.T) {
	assert.True(t, isNotFoundOCI(errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "normalized not found",
	}))
	assert.True(t, isNotFoundOCI(errorutil.UnauthorizedAndNotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotAuthorizedOrNotFound,
		Description:    "normalized auth ambiguity",
	}))
	assert.True(t, isNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
	assert.False(t, isNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not found",
	}))
	assert.False(t, isNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not authorized",
	}))
	assert.False(t, isNotFoundOCI(errorutil.ConflictOciError{
		HTTPStatusCode: 409,
		ErrorCode:      errorutil.IncorrectState,
		Description:    "normalized conflict",
	}))
	assert.False(t, isNotFoundOCI(fakeServiceError{
		statusCode: 409,
		code:       errorutil.IncorrectState,
		message:    "resource conflict",
	}))
}

func drainVcnEvents(recorder *record.FakeRecorder) []string {
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

func assertEventContains(t *testing.T, events []string, want string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, want) {
			return
		}
	}
	t.Fatalf("events %v do not contain %q", events, want)
}

func assertNoEventContains(t *testing.T, events []string, unexpected string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, unexpected) {
			t.Fatalf("events %v unexpectedly contain %q", events, unexpected)
		}
	}
}

type memoryVcnClient struct {
	ctrlclient.Client
	stored ctrlclient.Object
}

func newMemoryVcnClient(scheme *runtime.Scheme, obj ctrlclient.Object) *memoryVcnClient {
	return &memoryVcnClient{
		Client: ctrlclientfake.NewClientBuilder().WithScheme(scheme).Build(),
		stored: obj.DeepCopyObject().(ctrlclient.Object),
	}
}

func (c *memoryVcnClient) Get(_ context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, _ ...ctrlclient.GetOption) error {
	if c.stored == nil || c.stored.GetName() != key.Name || c.stored.GetNamespace() != key.Namespace {
		return apierrors.NewNotFound(schema.GroupResource{Group: "core.oracle.com", Resource: "vcns"}, key.Name)
	}

	value := reflect.ValueOf(obj)
	source := reflect.ValueOf(c.stored.DeepCopyObject())
	if value.Kind() != reflect.Ptr || source.Kind() != reflect.Ptr {
		return stderrors.New("memory client requires pointer objects")
	}
	value.Elem().Set(source.Elem())
	return nil
}

func (c *memoryVcnClient) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.UpdateOption) error {
	c.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (c *memoryVcnClient) StoredVcn() *corev1beta1.Vcn {
	return c.stored.DeepCopyObject().(*corev1beta1.Vcn)
}

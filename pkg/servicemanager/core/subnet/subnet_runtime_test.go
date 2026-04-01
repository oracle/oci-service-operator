/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subnet

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeSubnetOCIClient struct {
	createFn func(context.Context, coresdk.CreateSubnetRequest) (coresdk.CreateSubnetResponse, error)
	getFn    func(context.Context, coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error)
	updateFn func(context.Context, coresdk.UpdateSubnetRequest) (coresdk.UpdateSubnetResponse, error)
	deleteFn func(context.Context, coresdk.DeleteSubnetRequest) (coresdk.DeleteSubnetResponse, error)
}

func (f *fakeSubnetOCIClient) CreateSubnet(ctx context.Context, req coresdk.CreateSubnetRequest) (coresdk.CreateSubnetResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return coresdk.CreateSubnetResponse{}, nil
}

func (f *fakeSubnetOCIClient) GetSubnet(ctx context.Context, req coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return coresdk.GetSubnetResponse{}, nil
}

func (f *fakeSubnetOCIClient) UpdateSubnet(ctx context.Context, req coresdk.UpdateSubnetRequest) (coresdk.UpdateSubnetResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return coresdk.UpdateSubnetResponse{}, nil
}

func (f *fakeSubnetOCIClient) DeleteSubnet(ctx context.Context, req coresdk.DeleteSubnetRequest) (coresdk.DeleteSubnetResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return coresdk.DeleteSubnetResponse{}, nil
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

func newTestManager(client subnetOCIClient) *SubnetServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewSubnetServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&subnetRuntimeClient{
			manager: manager,
			client:  client,
		})
	}
	return manager
}

func makeSpecSubnet() *corev1beta1.Subnet {
	return &corev1beta1.Subnet{
		Spec: corev1beta1.SubnetSpec{
			CidrBlock:               "10.0.1.0/24",
			CompartmentId:           "ocid1.compartment.oc1..example",
			VcnId:                   "ocid1.vcn.oc1..example",
			AvailabilityDomain:      "AD-1",
			DhcpOptionsId:           "ocid1.dhcp.oc1..example",
			DisplayName:             "test-subnet",
			DnsLabel:                "subnet123",
			DefinedTags:             map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			FreeformTags:            map[string]string{"env": "dev"},
			Ipv6CidrBlock:           "2001:db8::/64",
			Ipv6CidrBlocks:          []string{"2001:db8::/64", "2001:db8:1::/64"},
			ProhibitInternetIngress: true,
			ProhibitPublicIpOnVnic:  true,
			RouteTableId:            "ocid1.routetable.oc1..example",
			SecurityListIds:         []string{"ocid1.securitylist.oc1..b", "ocid1.securitylist.oc1..a"},
		},
	}
}

func makeSDKSubnet(id, displayName string, state coresdk.SubnetLifecycleStateEnum) coresdk.Subnet {
	return coresdk.Subnet{
		Id:                      common.String(id),
		CidrBlock:               common.String("10.0.1.0/24"),
		CompartmentId:           common.String("ocid1.compartment.oc1..example"),
		VcnId:                   common.String("ocid1.vcn.oc1..example"),
		RouteTableId:            common.String("ocid1.routetable.oc1..example"),
		VirtualRouterIp:         common.String("10.0.1.1"),
		VirtualRouterMac:        common.String("00:00:00:00:00:01"),
		AvailabilityDomain:      common.String("AD-1"),
		DhcpOptionsId:           common.String("ocid1.dhcp.oc1..example"),
		DisplayName:             common.String(displayName),
		DnsLabel:                common.String("subnet123"),
		DefinedTags:             map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		FreeformTags:            map[string]string{"env": "dev"},
		Ipv6CidrBlock:           common.String("2001:db8::/64"),
		Ipv6CidrBlocks:          []string{"2001:db8::/64", "2001:db8:1::/64"},
		Ipv6VirtualRouterIp:     common.String("2001:db8::1"),
		ProhibitInternetIngress: common.Bool(true),
		ProhibitPublicIpOnVnic:  common.Bool(true),
		SecurityListIds:         []string{"ocid1.securitylist.oc1..a", "ocid1.securitylist.oc1..b"},
		SubnetDomainName:        common.String("subnet123.vcn1.oraclevcn.com"),
		LifecycleState:          state,
	}
}

func TestCreateOrUpdate_CreateSuccessAndStatusProjection(t *testing.T) {
	var captured coresdk.CreateSubnetRequest
	manager := newTestManager(&fakeSubnetOCIClient{
		createFn: func(_ context.Context, req coresdk.CreateSubnetRequest) (coresdk.CreateSubnetResponse, error) {
			captured = req
			return coresdk.CreateSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..create", "test-subnet", coresdk.SubnetLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecSubnet()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, common.String("10.0.1.0/24"), captured.CidrBlock)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CompartmentId)
	assert.Equal(t, common.String("ocid1.vcn.oc1..example"), captured.VcnId)
	assert.Equal(t, common.String("AD-1"), captured.AvailabilityDomain)
	assert.Equal(t, common.String("ocid1.dhcp.oc1..example"), captured.DhcpOptionsId)
	assert.Equal(t, common.String("test-subnet"), captured.DisplayName)
	assert.Equal(t, common.String("subnet123"), captured.DnsLabel)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Equal(t, []string{"2001:db8:1::/64", "2001:db8::/64"}, captured.Ipv6CidrBlocks)
	assert.Equal(t, common.Bool(true), captured.ProhibitInternetIngress)
	assert.Equal(t, common.Bool(true), captured.ProhibitPublicIpOnVnic)
	assert.Equal(t, common.String("ocid1.routetable.oc1..example"), captured.RouteTableId)
	assert.Equal(t, []string{"ocid1.securitylist.oc1..a", "ocid1.securitylist.oc1..b"}, captured.SecurityListIds)
	assert.Equal(t, "ocid1.subnet.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, "test-subnet", resource.Status.DisplayName)
	assert.Equal(t, "subnet123", resource.Status.DnsLabel)
}

func TestCreateOrUpdate_ObserveByStatusOCID(t *testing.T) {
	getCalls := 0
	updateCalls := 0
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, req coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.subnet.oc1..existing", *req.SubnetId)
			return coresdk.GetSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..existing", "test-subnet", coresdk.SubnetLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateSubnetRequest) (coresdk.UpdateSubnetResponse, error) {
			updateCalls++
			return coresdk.UpdateSubnetResponse{}, nil
		},
	})

	resource := makeSpecSubnet()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.subnet.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, "10.0.1.1", resource.Status.VirtualRouterIp)
}

func TestCreateOrUpdate_ClearsStaleOptionalStatusFieldsOnProjection(t *testing.T) {
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			current := makeSDKSubnet("ocid1.subnet.oc1..existing", "test-subnet", coresdk.SubnetLifecycleStateAvailable)
			current.DisplayName = nil
			current.DefinedTags = nil
			current.FreeformTags = nil
			current.Ipv6CidrBlocks = nil
			current.SecurityListIds = nil
			current.SubnetDomainName = nil
			return coresdk.GetSubnetResponse{Subnet: current}, nil
		},
	})

	resource := makeSpecSubnet()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.subnet.oc1..existing")
	resource.Spec.DisplayName = ""
	resource.Spec.DefinedTags = nil
	resource.Spec.FreeformTags = nil
	resource.Spec.Ipv6CidrBlocks = nil
	resource.Spec.SecurityListIds = nil
	resource.Status.DisplayName = "stale-name"
	resource.Status.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Status.FreeformTags = map[string]string{"env": "stale"}
	resource.Status.Ipv6CidrBlocks = []string{"fd00::/64"}
	resource.Status.SecurityListIds = []string{"ocid1.securitylist.oc1..stale"}
	resource.Status.SubnetDomainName = "stale.oraclevcn.com"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "", resource.Status.DisplayName)
	assert.Nil(t, resource.Status.DefinedTags)
	assert.Nil(t, resource.Status.FreeformTags)
	assert.Nil(t, resource.Status.Ipv6CidrBlocks)
	assert.Nil(t, resource.Status.SecurityListIds)
	assert.Equal(t, "", resource.Status.SubnetDomainName)
}

func TestCreateOrUpdate_MutableDriftTriggersUpdate(t *testing.T) {
	var captured coresdk.UpdateSubnetRequest
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			current := makeSDKSubnet("ocid1.subnet.oc1..existing", "old-name", coresdk.SubnetLifecycleStateAvailable)
			current.RouteTableId = common.String("ocid1.routetable.oc1..old")
			current.SecurityListIds = []string{"ocid1.securitylist.oc1..old"}
			return coresdk.GetSubnetResponse{Subnet: current}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateSubnetRequest) (coresdk.UpdateSubnetResponse, error) {
			captured = req
			updated := makeSDKSubnet("ocid1.subnet.oc1..existing", "new-name", coresdk.SubnetLifecycleStateAvailable)
			return coresdk.UpdateSubnetResponse{Subnet: updated}, nil
		},
	})

	resource := makeSpecSubnet()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.subnet.oc1..existing")
	resource.Spec.DisplayName = "new-name"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "ocid1.subnet.oc1..existing", *captured.SubnetId)
	assert.Equal(t, "new-name", *captured.DisplayName)
	assert.Equal(t, "ocid1.routetable.oc1..example", *captured.RouteTableId)
	assert.Equal(t, []string{"ocid1.securitylist.oc1..a", "ocid1.securitylist.oc1..b"}, captured.SecurityListIds)
	assert.Equal(t, "new-name", resource.Status.DisplayName)
}

func TestCreateOrUpdate_RejectsUnsupportedCreateOnlyDrift(t *testing.T) {
	updateCalls := 0
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			return coresdk.GetSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..existing", "test-subnet", coresdk.SubnetLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateSubnetRequest) (coresdk.UpdateSubnetResponse, error) {
			updateCalls++
			return coresdk.UpdateSubnetResponse{}, nil
		},
	})

	resource := makeSpecSubnet()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.subnet.oc1..existing")
	resource.Spec.DnsLabel = ""
	resource.Spec.ProhibitInternetIngress = false

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "create-only field drift")
	assert.Contains(t, err.Error(), "dnsLabel")
	assert.Contains(t, err.Error(), "prohibitInternetIngress")
	assert.Equal(t, 0, updateCalls)
}

func TestCreateOrUpdate_RetryableStates(t *testing.T) {
	tests := []struct {
		name   string
		state  coresdk.SubnetLifecycleStateEnum
		reason shared.OSOKConditionType
	}{
		{name: "provisioning", state: coresdk.SubnetLifecycleStateProvisioning, reason: shared.Provisioning},
		{name: "updating", state: coresdk.SubnetLifecycleStateUpdating, reason: shared.Updating},
		{name: "terminating", state: coresdk.SubnetLifecycleStateTerminating, reason: shared.Terminating},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := newTestManager(&fakeSubnetOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
					return coresdk.GetSubnetResponse{
						Subnet: makeSDKSubnet("ocid1.subnet.oc1..existing", "test-subnet", tt.state),
					}, nil
				},
			})

			resource := makeSpecSubnet()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.subnet.oc1..existing")

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.NoError(t, err)
			assert.True(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
			assert.Equal(t, subnetRequeueDuration, resp.RequeueDuration)
			assert.Equal(t, string(tt.reason), resource.Status.OsokStatus.Reason)
		})
	}
}

func TestCreateOrUpdate_RecreatesOnExplicitNotFound(t *testing.T) {
	getCalls := 0
	createCalls := 0
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			getCalls++
			return coresdk.GetSubnetResponse{}, fakeServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "missing",
			}
		},
		createFn: func(_ context.Context, req coresdk.CreateSubnetRequest) (coresdk.CreateSubnetResponse, error) {
			createCalls++
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			return coresdk.CreateSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..recreated", "test-subnet", coresdk.SubnetLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecSubnet()
	resource.Status.Id = "ocid1.subnet.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.subnet.oc1..existing")
	resource.Status.OsokStatus.Message = "stale"
	oldCreatedAt := metav1.Now()
	resource.Status.OsokStatus.CreatedAt = &oldCreatedAt

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, "ocid1.subnet.oc1..recreated", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ocid1.subnet.oc1..recreated", resource.Status.Id)
	assert.NotNil(t, resource.Status.OsokStatus.CreatedAt)
	assert.NotEqual(t, oldCreatedAt, *resource.Status.OsokStatus.CreatedAt)
}

func TestDelete_ConfirmsDeletionOnNotFound(t *testing.T) {
	manager := newTestManager(&fakeSubnetOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteSubnetRequest) (coresdk.DeleteSubnetResponse, error) {
			assert.Equal(t, "ocid1.subnet.oc1..delete", *req.SubnetId)
			return coresdk.DeleteSubnetResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			return coresdk.GetSubnetResponse{}, fakeServiceError{statusCode: 404, code: "NotFound", message: "not found"}
		},
	})

	resource := makeSpecSubnet()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.subnet.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, done)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_RequeuesWhileTerminating(t *testing.T) {
	manager := newTestManager(&fakeSubnetOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteSubnetRequest) (coresdk.DeleteSubnetResponse, error) {
			return coresdk.DeleteSubnetResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			return coresdk.GetSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..delete", "test-subnet", coresdk.SubnetLifecycleStateTerminating),
			}, nil
		},
	})

	resource := makeSpecSubnet()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.subnet.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, done)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestDelete_RequeuesWhileObservedTerminated(t *testing.T) {
	manager := newTestManager(&fakeSubnetOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteSubnetRequest) (coresdk.DeleteSubnetResponse, error) {
			return coresdk.DeleteSubnetResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			return coresdk.GetSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..delete", "test-subnet", coresdk.SubnetLifecycleStateTerminated),
			}, nil
		},
	})

	resource := makeSpecSubnet()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.subnet.oc1..delete")

	done, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, done)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestIsSubnetNotFoundOCI_RejectsAuthAmbiguity(t *testing.T) {
	assert.False(t, isSubnetNotFoundOCI(errorutil.UnauthorizedAndNotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotAuthorizedOrNotFound,
		Description:    "normalized auth ambiguity",
	}))
	assert.False(t, isSubnetNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isSubnetNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
	assert.True(t, isSubnetNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not found",
	}))
	assert.False(t, isSubnetNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not authorized",
	}))
}

func TestCreateOrUpdate_RecreateClearsStaleNestedOsokStatusMetadata(t *testing.T) {
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			return coresdk.GetSubnetResponse{}, fakeServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "missing",
			}
		},
		createFn: func(_ context.Context, _ coresdk.CreateSubnetRequest) (coresdk.CreateSubnetResponse, error) {
			return coresdk.CreateSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..recreated", "test-subnet", coresdk.SubnetLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecSubnet()
	resource.Status.Id = "ocid1.subnet.oc1..deleted"
	resource.Status.OsokStatus = shared.OSOKStatus{
		Ocid:      "ocid1.subnet.oc1..deleted",
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
	assert.Equal(t, shared.OCID("ocid1.subnet.oc1..recreated"), resource.Status.OsokStatus.Ocid)
	assert.Nil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(shared.Active), resource.Status.OsokStatus.Reason)
	assert.Len(t, resource.Status.OsokStatus.Conditions, 1)
	assert.Equal(t, shared.Active, resource.Status.OsokStatus.Conditions[0].Type)
}

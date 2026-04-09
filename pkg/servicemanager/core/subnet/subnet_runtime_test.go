/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subnet

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

type fakeSubnetOCIClient struct {
	createFn func(context.Context, coresdk.CreateSubnetRequest) (coresdk.CreateSubnetResponse, error)
	getFn    func(context.Context, coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error)
	listFn   func(context.Context, coresdk.ListSubnetsRequest) (coresdk.ListSubnetsResponse, error)
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
	return coresdk.GetSubnetResponse{}, fakeServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "missing",
	}
}

func (f *fakeSubnetOCIClient) ListSubnets(ctx context.Context, req coresdk.ListSubnetsRequest) (coresdk.ListSubnetsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return coresdk.ListSubnetsResponse{}, nil
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

func newTestGeneratedDelegate(manager *SubnetServiceManager, client subnetOCIClient) SubnetServiceClient {
	if client == nil {
		client = &fakeSubnetOCIClient{}
	}

	config := generatedruntime.Config[*corev1beta1.Subnet]{
		Kind:    "Subnet",
		SDKName: "Subnet",
		Log:     manager.Log,
		Semantics: &generatedruntime.Semantics{
			FormalService:     "core",
			FormalSlug:        "subnet",
			StatusProjection:  "required",
			SecretSideEffects: "none",
			FinalizerPolicy:   "retain-until-confirmed-delete",
			Lifecycle: generatedruntime.LifecycleSemantics{
				ProvisioningStates: []string{"PROVISIONING"},
				UpdatingStates:     []string{"UPDATING"},
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
				Mutable:       []string{"cidrBlock", "definedTags", "dhcpOptionsId", "displayName", "freeformTags", "ipv6CidrBlock", "ipv6CidrBlocks", "routeTableId", "securityListIds"},
				ForceNew:      []string{"availabilityDomain", "compartmentId", "dnsLabel", "prohibitInternetIngress", "prohibitPublicIpOnVnic", "vcnId"},
				ConflictsWith: map[string][]string{},
			},
			Hooks: generatedruntime.HookSet{
				Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "template", Action: "CREATED"}},
				Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
				Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
			},
			CreateFollowUp: generatedruntime.FollowUpSemantics{
				Strategy: "read-after-write",
				Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}, {Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "template", Action: "CREATED"}},
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
			NewRequest: func() any { return &coresdk.CreateSubnetRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateSubnet(ctx, *request.(*coresdk.CreateSubnetRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "CreateSubnetDetails", RequestName: "CreateSubnetDetails", Contribution: "body"}},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.GetSubnetRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetSubnet(ctx, *request.(*coresdk.GetSubnetRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "SubnetId", RequestName: "subnetId", Contribution: "path", PreferResourceID: true}},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.ListSubnetsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.ListSubnets(ctx, *request.(*coresdk.ListSubnetsRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
				{FieldName: "Page", RequestName: "page", Contribution: "query"},
				{FieldName: "VcnId", RequestName: "vcnId", Contribution: "query"},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
				{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.UpdateSubnetRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateSubnet(ctx, *request.(*coresdk.UpdateSubnetRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "SubnetId", RequestName: "subnetId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateSubnetDetails", RequestName: "UpdateSubnetDetails", Contribution: "body"},
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &coresdk.DeleteSubnetRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteSubnet(ctx, *request.(*coresdk.DeleteSubnetRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "SubnetId", RequestName: "subnetId", Contribution: "path", PreferResourceID: true}},
		},
	}

	return defaultSubnetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*corev1beta1.Subnet](config),
	}
}

func newTestManager(client subnetOCIClient) *SubnetServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewSubnetServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&subnetRuntimeClient{
			manager:  manager,
			delegate: newTestGeneratedDelegate(manager, client),
			client:   client,
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
		getFn: func(_ context.Context, req coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			assert.Equal(t, "ocid1.subnet.oc1..create", *req.SubnetId)
			return coresdk.GetSubnetResponse{
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
	assert.Equal(t, 2, getCalls)
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
	getCalls := 0
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			getCalls++
			if getCalls >= 3 {
				return coresdk.GetSubnetResponse{
					Subnet: makeSDKSubnet("ocid1.subnet.oc1..existing", "new-name", coresdk.SubnetLifecycleStateAvailable),
				}, nil
			}

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
	assert.Equal(t, 3, getCalls)
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

func TestCreateOrUpdate_DoesNotUpdateWhileProvisioning(t *testing.T) {
	assertNoUpdateWhileLifecycleRetryable(t, coresdk.SubnetLifecycleStateProvisioning)
}

func TestCreateOrUpdate_DoesNotUpdateWhileUpdating(t *testing.T) {
	assertNoUpdateWhileLifecycleRetryable(t, coresdk.SubnetLifecycleStateUpdating)
}

func TestCreateOrUpdate_DoesNotUpdateWhileTerminating(t *testing.T) {
	assertNoUpdateWhileLifecycleRetryable(t, coresdk.SubnetLifecycleStateTerminating)
}

func TestCreateOrUpdate_DefersCreateOnlyDriftWhileProvisioning(t *testing.T) {
	assertCreateOnlyDriftDeferredWhileLifecycleRetryable(t, coresdk.SubnetLifecycleStateProvisioning)
}

func TestCreateOrUpdate_DefersCreateOnlyDriftWhileUpdating(t *testing.T) {
	assertCreateOnlyDriftDeferredWhileLifecycleRetryable(t, coresdk.SubnetLifecycleStateUpdating)
}

func TestCreateOrUpdate_DefersCreateOnlyDriftWhileTerminating(t *testing.T) {
	assertCreateOnlyDriftDeferredWhileLifecycleRetryable(t, coresdk.SubnetLifecycleStateTerminating)
}

func TestCreateOrUpdate_FailsWhenObservedTerminated(t *testing.T) {
	updateCalls := 0
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			return coresdk.GetSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..existing", "test-subnet", coresdk.SubnetLifecycleStateTerminated),
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

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Equal(t, 0, updateCalls)
	assert.Contains(t, err.Error(), "Subnet lifecycle state \"TERMINATED\" is not modeled for create or update")
	assert.Equal(t, "TERMINATED", resource.Status.LifecycleState)
}

func TestCreateOrUpdate_RecreatesOnExplicitNotFound(t *testing.T) {
	getCalls := 0
	createCalls := 0
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, req coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			getCalls++
			switch *req.SubnetId {
			case "ocid1.subnet.oc1..existing":
				return coresdk.GetSubnetResponse{}, fakeServiceError{
					statusCode: 404,
					code:       "NotFound",
					message:    "missing",
				}
			case "ocid1.subnet.oc1..recreated":
				return coresdk.GetSubnetResponse{
					Subnet: makeSDKSubnet("ocid1.subnet.oc1..recreated", "test-subnet", coresdk.SubnetLifecycleStateAvailable),
				}, nil
			default:
				t.Fatalf("unexpected subnet lookup %q", *req.SubnetId)
				return coresdk.GetSubnetResponse{}, nil
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
	assert.Equal(t, 2, getCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, "ocid1.subnet.oc1..recreated", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ocid1.subnet.oc1..recreated", resource.Status.Id)
	assert.NotNil(t, resource.Status.OsokStatus.CreatedAt)
	assert.NotEqual(t, oldCreatedAt, *resource.Status.OsokStatus.CreatedAt)
}

func TestCreateOrUpdate_DoesNotBindByStatusIDWithoutTrackedOCID(t *testing.T) {
	getCalls := 0
	listCalls := 0
	createCalls := 0
	manager := newTestManager(&fakeSubnetOCIClient{
		listFn: func(_ context.Context, _ coresdk.ListSubnetsRequest) (coresdk.ListSubnetsResponse, error) {
			listCalls++
			return coresdk.ListSubnetsResponse{}, nil
		},
		createFn: func(_ context.Context, _ coresdk.CreateSubnetRequest) (coresdk.CreateSubnetResponse, error) {
			createCalls++
			return coresdk.CreateSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..create", "test-subnet", coresdk.SubnetLifecycleStateAvailable),
			}, nil
		},
		getFn: func(_ context.Context, req coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.subnet.oc1..create", *req.SubnetId)
			return coresdk.GetSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..create", "test-subnet", coresdk.SubnetLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecSubnet()
	resource.Status.Id = "ocid1.subnet.oc1..status-only"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, listCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, shared.OCID("ocid1.subnet.oc1..create"), resource.Status.OsokStatus.Ocid)
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

func TestReconcileDelete_ReleasesFinalizerOnUnambiguousNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.Subnet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "Subnet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-subnet",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.SubnetStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.subnet.oc1..delete"),
			},
		},
	}

	manager := newTestManager(&fakeSubnetOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteSubnetRequest) (coresdk.DeleteSubnetResponse, error) {
			assert.Equal(t, "ocid1.subnet.oc1..delete", *req.SubnetId)
			return coresdk.DeleteSubnetResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			return coresdk.GetSubnetResponse{}, fakeServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "resource not found",
			}
		},
	})

	kubeClient := newMemorySubnetClient(scheme, resource)
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
		NamespacedName: ctrlclient.ObjectKey{Name: "test-subnet", Namespace: "default"},
	}, &corev1beta1.Subnet{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredSubnet(), osokcore.OSOKFinalizerName))

	events := drainSubnetEvents(recorder)
	assertSubnetEventContains(t, events, "Removed finalizer")
	assertNoSubnetEventContains(t, events, "Failed to delete resource")
}

func assertNoUpdateWhileLifecycleRetryable(t *testing.T, state coresdk.SubnetLifecycleStateEnum) {
	t.Helper()

	updateCalls := 0
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			current := makeSDKSubnet("ocid1.subnet.oc1..existing", "old-name", state)
			current.RouteTableId = common.String("ocid1.routetable.oc1..old")
			return coresdk.GetSubnetResponse{Subnet: current}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateSubnetRequest) (coresdk.UpdateSubnetResponse, error) {
			updateCalls++
			return coresdk.UpdateSubnetResponse{}, nil
		},
	})

	resource := makeSpecSubnet()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.subnet.oc1..existing")
	resource.Spec.DisplayName = "new-name"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, resp.ShouldRequeue)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, string(state), resource.Status.LifecycleState)
}

func assertCreateOnlyDriftDeferredWhileLifecycleRetryable(t *testing.T, state coresdk.SubnetLifecycleStateEnum) {
	t.Helper()

	expectedReason := shared.Terminating
	switch state {
	case coresdk.SubnetLifecycleStateProvisioning:
		expectedReason = shared.Provisioning
	case coresdk.SubnetLifecycleStateUpdating:
		expectedReason = shared.Updating
	}

	updateCalls := 0
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			return coresdk.GetSubnetResponse{
				Subnet: makeSDKSubnet("ocid1.subnet.oc1..existing", "test-subnet", state),
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

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, resp.ShouldRequeue)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, string(state), resource.Status.LifecycleState)
	assert.Equal(t, string(expectedReason), resource.Status.OsokStatus.Reason)
}

func TestIsSubnetReadNotFoundOCI_RejectsAuthAmbiguity(t *testing.T) {
	assert.True(t, isSubnetReadNotFoundOCI(errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "normalized not found",
	}))
	assert.False(t, isSubnetReadNotFoundOCI(errorutil.UnauthorizedAndNotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotAuthorizedOrNotFound,
		Description:    "normalized auth ambiguity",
	}))
	assert.False(t, isSubnetReadNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isSubnetReadNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
	assert.False(t, isSubnetReadNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not found",
	}))
	assert.False(t, isSubnetReadNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not authorized",
	}))
	assert.False(t, isSubnetReadNotFoundOCI(errorutil.ConflictOciError{
		HTTPStatusCode: 409,
		ErrorCode:      errorutil.IncorrectState,
		Description:    "normalized conflict",
	}))
	assert.False(t, isSubnetReadNotFoundOCI(fakeServiceError{
		statusCode: 409,
		code:       errorutil.IncorrectState,
		message:    "resource conflict",
	}))
}

func TestIsSubnetDeleteNotFoundOCI_AcceptsAuthShaped404(t *testing.T) {
	assert.True(t, isSubnetDeleteNotFoundOCI(errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "normalized not found",
	}))
	assert.True(t, isSubnetDeleteNotFoundOCI(errorutil.UnauthorizedAndNotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotAuthorizedOrNotFound,
		Description:    "normalized auth ambiguity",
	}))
	assert.True(t, isSubnetDeleteNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "NotAuthorizedOrNotFound",
		message:    "auth ambiguity",
	}))
	assert.True(t, isSubnetDeleteNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "NotFound",
		message:    "not found",
	}))
	assert.False(t, isSubnetDeleteNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not found",
	}))
	assert.False(t, isSubnetDeleteNotFoundOCI(fakeServiceError{
		statusCode: 404,
		code:       "UnexpectedCode",
		message:    "resource not authorized",
	}))
	assert.False(t, isSubnetDeleteNotFoundOCI(errorutil.ConflictOciError{
		HTTPStatusCode: 409,
		ErrorCode:      errorutil.IncorrectState,
		Description:    "normalized conflict",
	}))
	assert.False(t, isSubnetDeleteNotFoundOCI(fakeServiceError{
		statusCode: 409,
		code:       errorutil.IncorrectState,
		message:    "resource conflict",
	}))
}

func TestReconcileDelete_ReleasesFinalizerOnAuthShapedNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.Subnet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "Subnet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-subnet-auth-shaped-404",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.SubnetStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.subnet.oc1..delete"),
			},
		},
	}

	manager := newTestManager(&fakeSubnetOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteSubnetRequest) (coresdk.DeleteSubnetResponse, error) {
			assert.Equal(t, "ocid1.subnet.oc1..delete", *req.SubnetId)
			return coresdk.DeleteSubnetResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			return coresdk.GetSubnetResponse{}, fakeServiceError{
				statusCode: 404,
				code:       errorutil.NotAuthorizedOrNotFound,
				message:    "not authorized or not found",
			}
		},
	})

	kubeClient := newMemorySubnetClient(scheme, resource)
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
		NamespacedName: ctrlclient.ObjectKey{Name: "test-subnet-auth-shaped-404", Namespace: "default"},
	}, &corev1beta1.Subnet{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredSubnet(), osokcore.OSOKFinalizerName))

	events := drainSubnetEvents(recorder)
	assertSubnetEventContains(t, events, "Removed finalizer")
	assertNoSubnetEventContains(t, events, "Failed to delete resource")
}

func TestCreateOrUpdate_RecreateClearsStaleNestedOsokStatusMetadata(t *testing.T) {
	manager := newTestManager(&fakeSubnetOCIClient{
		getFn: func(_ context.Context, req coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error) {
			switch *req.SubnetId {
			case "ocid1.subnet.oc1..deleted":
				return coresdk.GetSubnetResponse{}, fakeServiceError{
					statusCode: 404,
					code:       "NotFound",
					message:    "missing",
				}
			case "ocid1.subnet.oc1..recreated":
				return coresdk.GetSubnetResponse{
					Subnet: makeSDKSubnet("ocid1.subnet.oc1..recreated", "test-subnet", coresdk.SubnetLifecycleStateAvailable),
				}, nil
			default:
				t.Fatalf("unexpected subnet lookup %q", *req.SubnetId)
				return coresdk.GetSubnetResponse{}, nil
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

func drainSubnetEvents(recorder *record.FakeRecorder) []string {
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

func assertSubnetEventContains(t *testing.T, events []string, want string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, want) {
			return
		}
	}
	t.Fatalf("events %v do not contain %q", events, want)
}

func assertNoSubnetEventContains(t *testing.T, events []string, unexpected string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, unexpected) {
			t.Fatalf("events %v unexpectedly contain %q", events, unexpected)
		}
	}
}

type memorySubnetClient struct {
	ctrlclient.Client
	stored ctrlclient.Object
}

func newMemorySubnetClient(scheme *runtime.Scheme, obj ctrlclient.Object) *memorySubnetClient {
	return &memorySubnetClient{
		Client: ctrlclientfake.NewClientBuilder().WithScheme(scheme).Build(),
		stored: obj.DeepCopyObject().(ctrlclient.Object),
	}
}

func (c *memorySubnetClient) Get(_ context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, _ ...ctrlclient.GetOption) error {
	if c.stored == nil || c.stored.GetName() != key.Name || c.stored.GetNamespace() != key.Namespace {
		return apierrors.NewNotFound(schema.GroupResource{Group: "core.oracle.com", Resource: "subnets"}, key.Name)
	}

	value := reflect.ValueOf(obj)
	source := reflect.ValueOf(c.stored.DeepCopyObject())
	if value.Kind() != reflect.Ptr || source.Kind() != reflect.Ptr {
		return stderrors.New("memory client requires pointer objects")
	}
	value.Elem().Set(source.Elem())
	return nil
}

func (c *memorySubnetClient) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.UpdateOption) error {
	c.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (c *memorySubnetClient) StoredSubnet() *corev1beta1.Subnet {
	return c.stored.DeepCopyObject().(*corev1beta1.Subnet)
}

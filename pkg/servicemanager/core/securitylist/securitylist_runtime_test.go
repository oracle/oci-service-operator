/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securitylist

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
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
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

type fakeSecurityListOCIClient struct {
	createFn func(context.Context, coresdk.CreateSecurityListRequest) (coresdk.CreateSecurityListResponse, error)
	getFn    func(context.Context, coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error)
	listFn   func(context.Context, coresdk.ListSecurityListsRequest) (coresdk.ListSecurityListsResponse, error)
	updateFn func(context.Context, coresdk.UpdateSecurityListRequest) (coresdk.UpdateSecurityListResponse, error)
	deleteFn func(context.Context, coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error)
}

func (f *fakeSecurityListOCIClient) CreateSecurityList(ctx context.Context, req coresdk.CreateSecurityListRequest) (coresdk.CreateSecurityListResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return coresdk.CreateSecurityListResponse{}, nil
}

func (f *fakeSecurityListOCIClient) GetSecurityList(ctx context.Context, req coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return coresdk.GetSecurityListResponse{}, nil
}

func (f *fakeSecurityListOCIClient) ListSecurityLists(ctx context.Context, req coresdk.ListSecurityListsRequest) (coresdk.ListSecurityListsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return coresdk.ListSecurityListsResponse{}, nil
}

func (f *fakeSecurityListOCIClient) UpdateSecurityList(ctx context.Context, req coresdk.UpdateSecurityListRequest) (coresdk.UpdateSecurityListResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return coresdk.UpdateSecurityListResponse{}, nil
}

func (f *fakeSecurityListOCIClient) DeleteSecurityList(ctx context.Context, req coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return coresdk.DeleteSecurityListResponse{}, nil
}

type fakeSecurityListServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeSecurityListServiceError) Error() string          { return f.message }
func (f fakeSecurityListServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeSecurityListServiceError) GetMessage() string     { return f.message }
func (f fakeSecurityListServiceError) GetCode() string        { return f.code }
func (f fakeSecurityListServiceError) GetOpcRequestID() string {
	return ""
}

func newSecurityListTestManager(client securityListOCIClient) *SecurityListServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewSecurityListServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(newSecurityListServiceClientWithOCIClient(log, client))
	}
	return manager
}

func newSecurityListServiceClientWithOCIClient(log loggerutil.OSOKLogger, client securityListOCIClient) SecurityListServiceClient {
	manager := &SecurityListServiceManager{Log: log}
	hooks := newSecurityListRuntimeHooksWithOCIClient(client)
	applySecurityListRuntimeHooks(manager, &hooks)
	delegate := defaultSecurityListServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*corev1beta1.SecurityList](
			buildSecurityListGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSecurityListGeneratedClient(hooks, delegate)
}

func newSecurityListRuntimeHooksWithOCIClient(client securityListOCIClient) SecurityListRuntimeHooks {
	return SecurityListRuntimeHooks{
		Semantics:       newSecurityListRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*corev1beta1.SecurityList]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*corev1beta1.SecurityList]{},
		StatusHooks:     generatedruntime.StatusHooks[*corev1beta1.SecurityList]{},
		ParityHooks:     generatedruntime.ParityHooks[*corev1beta1.SecurityList]{},
		Async:           generatedruntime.AsyncHooks[*corev1beta1.SecurityList]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*corev1beta1.SecurityList]{},
		Create: runtimeOperationHooks[coresdk.CreateSecurityListRequest, coresdk.CreateSecurityListResponse]{
			Fields: securityListCreateFields(),
			Call: func(ctx context.Context, request coresdk.CreateSecurityListRequest) (coresdk.CreateSecurityListResponse, error) {
				return client.CreateSecurityList(ctx, request)
			},
		},
		Get: runtimeOperationHooks[coresdk.GetSecurityListRequest, coresdk.GetSecurityListResponse]{
			Fields: securityListGetFields(),
			Call: func(ctx context.Context, request coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
				return client.GetSecurityList(ctx, request)
			},
		},
		List: runtimeOperationHooks[coresdk.ListSecurityListsRequest, coresdk.ListSecurityListsResponse]{
			Fields: securityListListFields(),
			Call: func(ctx context.Context, request coresdk.ListSecurityListsRequest) (coresdk.ListSecurityListsResponse, error) {
				return client.ListSecurityLists(ctx, request)
			},
		},
		Update: runtimeOperationHooks[coresdk.UpdateSecurityListRequest, coresdk.UpdateSecurityListResponse]{
			Fields: securityListUpdateFields(),
			Call: func(ctx context.Context, request coresdk.UpdateSecurityListRequest) (coresdk.UpdateSecurityListResponse, error) {
				return client.UpdateSecurityList(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[coresdk.DeleteSecurityListRequest, coresdk.DeleteSecurityListResponse]{
			Fields: securityListDeleteFields(),
			Call: func(ctx context.Context, request coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error) {
				return client.DeleteSecurityList(ctx, request)
			},
		},
	}
}

func securityListCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateSecurityListDetails", RequestName: "CreateSecurityListDetails", Contribution: "body"},
	}
}

func securityListGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SecurityListId", RequestName: "securityListId", Contribution: "path", PreferResourceID: true},
	}
}

func securityListListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "VcnId", RequestName: "vcnId", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
	}
}

func securityListUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SecurityListId", RequestName: "securityListId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateSecurityListDetails", RequestName: "UpdateSecurityListDetails", Contribution: "body"},
	}
}

func securityListDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SecurityListId", RequestName: "securityListId", Contribution: "path", PreferResourceID: true},
	}
}

func TestNewSecurityListServiceManager_UsesGeneratedRuntimeDelegate(t *testing.T) {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewSecurityListServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)

	runtimeClient, isWrappedRuntime := manager.client.(*securityListRuntimeClient)
	assert.True(t, isWrappedRuntime)
	_, isGeneratedRuntimeDelegate := runtimeClient.delegate.(defaultSecurityListServiceClient)
	assert.True(t, isGeneratedRuntimeDelegate)
}

func makeSpecSecurityList() *corev1beta1.SecurityList {
	return &corev1beta1.SecurityList{
		Spec: corev1beta1.SecurityListSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			VcnId:         "ocid1.vcn.oc1..example",
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			DisplayName:   "test-security-list",
			FreeformTags:  map[string]string{"env": "dev"},
			EgressSecurityRules: []corev1beta1.SecurityListEgressSecurityRule{
				{
					Destination:     "0.0.0.0/0",
					Protocol:        "6",
					DestinationType: string(coresdk.EgressSecurityRuleDestinationTypeCidrBlock),
					IsStateless:     true,
					Description:     "allow https outbound",
					TcpOptions: corev1beta1.SecurityListEgressSecurityRuleTcpOptions{
						DestinationPortRange: corev1beta1.SecurityListEgressSecurityRuleTcpOptionsDestinationPortRange{Min: 443, Max: 443},
					},
				},
				{
					Destination: "oci-phx-objectstorage",
					Protocol:    "1",
					IcmpOptions: corev1beta1.SecurityListEgressSecurityRuleIcmpOptions{Type: 3, Code: 4},
					Description: "allow icmp service traffic",
				},
			},
			IngressSecurityRules: []corev1beta1.SecurityListIngressSecurityRule{
				{
					Protocol:    "17",
					Source:      "10.0.0.0/16",
					SourceType:  string(coresdk.IngressSecurityRuleSourceTypeCidrBlock),
					Description: "allow dns ingress",
					UdpOptions: corev1beta1.SecurityListIngressSecurityRuleUdpOptions{
						DestinationPortRange: corev1beta1.SecurityListIngressSecurityRuleUdpOptionsDestinationPortRange{Min: 53, Max: 53},
						SourcePortRange:      corev1beta1.SecurityListIngressSecurityRuleUdpOptionsSourcePortRange{Min: 1024, Max: 65535},
					},
				},
			},
		},
	}
}

func makeSDKSecurityList(id, displayName string, state coresdk.SecurityListLifecycleStateEnum) coresdk.SecurityList {
	return coresdk.SecurityList{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		VcnId:          common.String("ocid1.vcn.oc1..example"),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		FreeformTags:   map[string]string{"env": "dev"},
		TimeCreated:    &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		EgressSecurityRules: []coresdk.EgressSecurityRule{
			{
				Destination:     common.String("0.0.0.0/0"),
				Protocol:        common.String("6"),
				DestinationType: coresdk.EgressSecurityRuleDestinationTypeCidrBlock,
				IsStateless:     common.Bool(true),
				Description:     common.String("allow https outbound"),
				TcpOptions: &coresdk.TcpOptions{
					DestinationPortRange: &coresdk.PortRange{Min: common.Int(443), Max: common.Int(443)},
				},
			},
			{
				Destination: common.String("oci-phx-objectstorage"),
				Protocol:    common.String("1"),
				Description: common.String("allow icmp service traffic"),
				IcmpOptions: &coresdk.IcmpOptions{
					Type: common.Int(3),
					Code: common.Int(4),
				},
			},
		},
		IngressSecurityRules: []coresdk.IngressSecurityRule{
			{
				Protocol:    common.String("17"),
				Source:      common.String("10.0.0.0/16"),
				SourceType:  coresdk.IngressSecurityRuleSourceTypeCidrBlock,
				Description: common.String("allow dns ingress"),
				UdpOptions: &coresdk.UdpOptions{
					DestinationPortRange: &coresdk.PortRange{Min: common.Int(53), Max: common.Int(53)},
					SourcePortRange:      &coresdk.PortRange{Min: common.Int(1024), Max: common.Int(65535)},
				},
			},
		},
	}
}

func TestCreateOrUpdate_CreateSuccessAndStatusProjection(t *testing.T) {
	var captured coresdk.CreateSecurityListRequest
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		createFn: func(_ context.Context, req coresdk.CreateSecurityListRequest) (coresdk.CreateSecurityListResponse, error) {
			captured = req
			return coresdk.CreateSecurityListResponse{
				SecurityList: makeSDKSecurityList("ocid1.securitylist.oc1..create", "test-security-list", coresdk.SecurityListLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecSecurityList()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CompartmentId)
	assert.Equal(t, common.String("ocid1.vcn.oc1..example"), captured.VcnId)
	assert.Equal(t, common.String("test-security-list"), captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Len(t, captured.EgressSecurityRules, 2)
	assert.Len(t, captured.IngressSecurityRules, 1)
	assert.NotNil(t, captured.EgressSecurityRules[0].TcpOptions)
	assert.Nil(t, captured.EgressSecurityRules[0].UdpOptions)
	assert.NotNil(t, captured.EgressSecurityRules[1].IcmpOptions)
	assert.NotNil(t, captured.IngressSecurityRules[0].UdpOptions)
	assert.Equal(t, "ocid1.securitylist.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ocid1.securitylist.oc1..create", resource.Status.Id)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, "test-security-list", resource.Status.DisplayName)
	assert.Equal(t, "2026-04-01T00:00:00Z", resource.Status.TimeCreated)
	assert.Len(t, resource.Status.EgressSecurityRules, 2)
	assert.Len(t, resource.Status.IngressSecurityRules, 1)
	assert.Equal(t, 443, resource.Status.EgressSecurityRules[0].TcpOptions.DestinationPortRange.Min)
	assert.Equal(t, 53, resource.Status.IngressSecurityRules[0].UdpOptions.DestinationPortRange.Min)
}

func TestValidateSecurityListSDKContract(t *testing.T) {
	assert.NoError(t, validateSecurityListSDKContract())
}

func TestCreateOrUpdate_ObserveByStatusOCID_NoOpWhenStateMatches(t *testing.T) {
	getCalls := 0
	updateCalls := 0
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		getFn: func(_ context.Context, req coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.securitylist.oc1..existing", *req.SecurityListId)
			return coresdk.GetSecurityListResponse{
				SecurityList: makeSDKSecurityList("ocid1.securitylist.oc1..existing", "test-security-list", coresdk.SecurityListLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateSecurityListRequest) (coresdk.UpdateSecurityListResponse, error) {
			updateCalls++
			return coresdk.UpdateSecurityListResponse{}, nil
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
}

func TestCreateOrUpdate_RecreatesOnStaleTrackedIdentityWithoutListReuse(t *testing.T) {
	getCalls := 0
	listCalls := 0
	createCalls := 0
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		getFn: func(_ context.Context, req coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.securitylist.oc1..stale", *req.SecurityListId)
			return coresdk.GetSecurityListResponse{}, fakeSecurityListServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "stale tracked security list",
			}
		},
		listFn: func(_ context.Context, _ coresdk.ListSecurityListsRequest) (coresdk.ListSecurityListsResponse, error) {
			listCalls++
			return coresdk.ListSecurityListsResponse{}, nil
		},
		createFn: func(_ context.Context, req coresdk.CreateSecurityListRequest) (coresdk.CreateSecurityListResponse, error) {
			createCalls++
			assert.Equal(t, common.String("ocid1.compartment.oc1..example"), req.CompartmentId)
			return coresdk.CreateSecurityListResponse{
				SecurityList: makeSDKSecurityList("ocid1.securitylist.oc1..recreated", "test-security-list", coresdk.SecurityListLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..stale")
	resource.Status.Id = "ocid1.securitylist.oc1..stale"
	resource.Status.DisplayName = "stale"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, listCalls)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, "ocid1.securitylist.oc1..recreated", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ocid1.securitylist.oc1..recreated", resource.Status.Id)
	assert.Equal(t, "test-security-list", resource.Status.DisplayName)
}

func TestCreateOrUpdate_MutableDriftTriggersUpdateForRulesAndTags(t *testing.T) {
	var captured coresdk.UpdateSecurityListRequest
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			current := makeSDKSecurityList("ocid1.securitylist.oc1..existing", "old-name", coresdk.SecurityListLifecycleStateAvailable)
			current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "41"}}
			current.FreeformTags = map[string]string{"env": "stage"}
			current.EgressSecurityRules = []coresdk.EgressSecurityRule{
				{
					Destination: common.String("0.0.0.0/0"),
					Protocol:    common.String("6"),
					Description: common.String("old outbound"),
				},
			}
			current.IngressSecurityRules = []coresdk.IngressSecurityRule{
				{
					Protocol: common.String("17"),
					Source:   common.String("192.168.0.0/24"),
				},
			}
			return coresdk.GetSecurityListResponse{SecurityList: current}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateSecurityListRequest) (coresdk.UpdateSecurityListResponse, error) {
			captured = req
			updated := makeSDKSecurityList("ocid1.securitylist.oc1..existing", "new-name", coresdk.SecurityListLifecycleStateAvailable)
			return coresdk.UpdateSecurityListResponse{SecurityList: updated}, nil
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..existing")
	resource.Spec.DisplayName = "new-name"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "ocid1.securitylist.oc1..existing", *captured.SecurityListId)
	assert.Equal(t, "new-name", *captured.DisplayName)
	assert.Equal(t, map[string]string{"env": "dev"}, captured.FreeformTags)
	assert.Len(t, captured.EgressSecurityRules, 2)
	assert.Len(t, captured.IngressSecurityRules, 1)
	assert.Equal(t, "0.0.0.0/0", *captured.EgressSecurityRules[0].Destination)
	assert.Equal(t, "new-name", resource.Status.DisplayName)
}

func TestCreateOrUpdate_RuleNormalizationAvoidsSpuriousUpdate(t *testing.T) {
	updateCalls := 0
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			current := makeSDKSecurityList("ocid1.securitylist.oc1..existing", "test-security-list", coresdk.SecurityListLifecycleStateAvailable)
			current.EgressSecurityRules = []coresdk.EgressSecurityRule{
				current.EgressSecurityRules[1],
				current.EgressSecurityRules[0],
			}
			return coresdk.GetSecurityListResponse{SecurityList: current}, nil
		},
		updateFn: func(_ context.Context, _ coresdk.UpdateSecurityListRequest) (coresdk.UpdateSecurityListResponse, error) {
			updateCalls++
			return coresdk.UpdateSecurityListResponse{}, nil
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, 0, updateCalls)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
}

func TestCreateOrUpdate_ClearingMutableFieldsTriggersUpdate(t *testing.T) {
	var captured coresdk.UpdateSecurityListRequest
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			return coresdk.GetSecurityListResponse{
				SecurityList: makeSDKSecurityList("ocid1.securitylist.oc1..existing", "old-name", coresdk.SecurityListLifecycleStateAvailable),
			}, nil
		},
		updateFn: func(_ context.Context, req coresdk.UpdateSecurityListRequest) (coresdk.UpdateSecurityListResponse, error) {
			captured = req
			updated := makeSDKSecurityList("ocid1.securitylist.oc1..existing", "", coresdk.SecurityListLifecycleStateAvailable)
			updated.DisplayName = common.String("")
			updated.DefinedTags = map[string]map[string]interface{}{}
			updated.FreeformTags = map[string]string{}
			return coresdk.UpdateSecurityListResponse{SecurityList: updated}, nil
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..existing")
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
		mutateSpec  func(*corev1beta1.SecurityList)
		expectField string
	}{
		{
			name: "compartmentId",
			mutateSpec: func(resource *corev1beta1.SecurityList) {
				resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
			},
			expectField: "compartmentId",
		},
		{
			name: "vcnId",
			mutateSpec: func(resource *corev1beta1.SecurityList) {
				resource.Spec.VcnId = "ocid1.vcn.oc1..different"
			},
			expectField: "vcnId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
					return coresdk.GetSecurityListResponse{
						SecurityList: makeSDKSecurityList("ocid1.securitylist.oc1..existing", "test-security-list", coresdk.SecurityListLifecycleStateAvailable),
					}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateSecurityListRequest) (coresdk.UpdateSecurityListResponse, error) {
					updateCalls++
					return coresdk.UpdateSecurityListResponse{}, nil
				},
			})

			resource := makeSpecSecurityList()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..existing")
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
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			current := makeSDKSecurityList("ocid1.securitylist.oc1..existing", "test-security-list", coresdk.SecurityListLifecycleStateAvailable)
			current.DisplayName = nil
			current.DefinedTags = nil
			current.FreeformTags = nil
			current.TimeCreated = nil
			current.EgressSecurityRules = []coresdk.EgressSecurityRule{{
				Destination: common.String("0.0.0.0/0"),
				Protocol:    common.String("6"),
			}}
			current.IngressSecurityRules = []coresdk.IngressSecurityRule{{
				Protocol: common.String("17"),
				Source:   common.String("10.0.0.0/16"),
			}}
			return coresdk.GetSecurityListResponse{SecurityList: current}, nil
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..existing")
	resource.Spec.DisplayName = ""
	resource.Spec.DefinedTags = nil
	resource.Spec.FreeformTags = nil
	resource.Spec.EgressSecurityRules = []corev1beta1.SecurityListEgressSecurityRule{{
		Destination: "0.0.0.0/0",
		Protocol:    "6",
	}}
	resource.Spec.IngressSecurityRules = []corev1beta1.SecurityListIngressSecurityRule{{
		Source:   "10.0.0.0/16",
		Protocol: "17",
	}}
	resource.Status.DisplayName = "stale-name"
	resource.Status.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Status.FreeformTags = map[string]string{"env": "stale"}
	resource.Status.TimeCreated = "2026-04-01T00:00:00Z"
	resource.Status.EgressSecurityRules = []corev1beta1.SecurityListEgressSecurityRule{{
		Destination: "stale",
		TcpOptions:  corev1beta1.SecurityListEgressSecurityRuleTcpOptions{DestinationPortRange: corev1beta1.SecurityListEgressSecurityRuleTcpOptionsDestinationPortRange{Min: 1, Max: 1}},
	}}
	resource.Status.IngressSecurityRules = []corev1beta1.SecurityListIngressSecurityRule{{
		Source: "stale",
		UdpOptions: corev1beta1.SecurityListIngressSecurityRuleUdpOptions{
			DestinationPortRange: corev1beta1.SecurityListIngressSecurityRuleUdpOptionsDestinationPortRange{Min: 53, Max: 53},
		},
	}}

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "", resource.Status.DisplayName)
	assert.Nil(t, resource.Status.DefinedTags)
	assert.Nil(t, resource.Status.FreeformTags)
	assert.Equal(t, "", resource.Status.TimeCreated)
	assert.Equal(t, 0, resource.Status.EgressSecurityRules[0].TcpOptions.DestinationPortRange.Min)
	assert.Equal(t, 0, resource.Status.IngressSecurityRules[0].UdpOptions.DestinationPortRange.Min)
}

func TestCreateOrUpdate_RetryableStates(t *testing.T) {
	tests := []struct {
		name   string
		state  coresdk.SecurityListLifecycleStateEnum
		reason shared.OSOKConditionType
	}{
		{name: "provisioning", state: coresdk.SecurityListLifecycleStateProvisioning, reason: shared.Provisioning},
		{name: "updating", state: securityListLifecycleStateUpdate, reason: shared.Updating},
		{name: "terminating", state: coresdk.SecurityListLifecycleStateTerminating, reason: shared.Terminating},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
				getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
					return coresdk.GetSecurityListResponse{
						SecurityList: makeSDKSecurityList("ocid1.securitylist.oc1..existing", "test-security-list", tt.state),
					}, nil
				},
				updateFn: func(_ context.Context, _ coresdk.UpdateSecurityListRequest) (coresdk.UpdateSecurityListResponse, error) {
					updateCalls++
					return coresdk.UpdateSecurityListResponse{}, nil
				},
			})

			resource := makeSpecSecurityList()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..existing")
			resource.Spec.DisplayName = "new-name"

			resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

			assert.NoError(t, err)
			assert.True(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
			assert.Equal(t, securityListRequeueDuration, resp.RequeueDuration)
			assert.Equal(t, string(tt.reason), resource.Status.OsokStatus.Reason)
			assert.Equal(t, 0, updateCalls)
		})
	}
}

func TestDelete_ConfirmsDeletionOnNotFound(t *testing.T) {
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error) {
			assert.Equal(t, "ocid1.securitylist.oc1..delete", *req.SecurityListId)
			return coresdk.DeleteSecurityListResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			return coresdk.GetSecurityListResponse{}, fakeSecurityListServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_AlreadyMissingOCIResourceIsTreatedAsDeleted(t *testing.T) {
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error) {
			assert.Equal(t, "ocid1.securitylist.oc1..delete", *req.SecurityListId)
			return coresdk.DeleteSecurityListResponse{}, fakeSecurityListServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "not found",
			}
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "OCI resource no longer exists", resource.Status.OsokStatus.Message)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_WithoutTrackedOCID_DoesNotListOrDelete(t *testing.T) {
	listCalls := 0
	deleteCalls := 0
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		listFn: func(_ context.Context, _ coresdk.ListSecurityListsRequest) (coresdk.ListSecurityListsResponse, error) {
			listCalls++
			return coresdk.ListSecurityListsResponse{}, nil
		},
		deleteFn: func(_ context.Context, _ coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error) {
			deleteCalls++
			return coresdk.DeleteSecurityListResponse{}, nil
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.Id = "ocid1.securitylist.oc1..status-only"

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, 0, listCalls)
	assert.Equal(t, 0, deleteCalls)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "OCI resource identifier is not recorded", resource.Status.OsokStatus.Message)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestDelete_KeepsFinalizerWhileObservedTerminating(t *testing.T) {
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		deleteFn: func(_ context.Context, _ coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error) {
			return coresdk.DeleteSecurityListResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			return coresdk.GetSecurityListResponse{
				SecurityList: makeSDKSecurityList("ocid1.securitylist.oc1..delete", "test-security-list", coresdk.SecurityListLifecycleStateTerminating),
			}, nil
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, "TERMINATING", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestDelete_PostDeleteConfirmReadMarksTerminatingForNonPendingLifecycle(t *testing.T) {
	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error) {
			assert.Equal(t, "ocid1.securitylist.oc1..delete", *req.SecurityListId)
			return coresdk.DeleteSecurityListResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			return coresdk.GetSecurityListResponse{
				SecurityList: makeSDKSecurityList("ocid1.securitylist.oc1..delete", "test-security-list", coresdk.SecurityListLifecycleStateAvailable),
			}, nil
		},
	})

	resource := makeSpecSecurityList()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitylist.oc1..delete")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, "AVAILABLE", resource.Status.LifecycleState)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "SecurityList test-security-list is AVAILABLE", resource.Status.OsokStatus.Message)
}

func TestSecurityListClassifierCoverageMatchesManualRuntimeContract(t *testing.T) {
	contract, err := errortest.ManualRuntimeClassifierContractFromReviewedRegistration("core", "SecurityList")
	if err != nil {
		t.Fatalf("ManualRuntimeClassifierContractFromReviewedRegistration() error = %v", err)
	}
	errortest.RunManualRuntimeClassifierContract(t, contract, isSecurityListReadNotFoundOCI, isSecurityListDeleteNotFoundOCI)
}

func TestReconcileDelete_ReleasesFinalizerOnAuthShapedNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.SecurityList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "SecurityList",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-security-list-auth-shaped-404",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.SecurityListStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.securitylist.oc1..delete"),
			},
		},
	}

	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error) {
			assert.Equal(t, "ocid1.securitylist.oc1..delete", *req.SecurityListId)
			return coresdk.DeleteSecurityListResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			return coresdk.GetSecurityListResponse{}, fakeSecurityListServiceError{
				statusCode: 404,
				code:       errorutil.NotAuthorizedOrNotFound,
				message:    "not authorized or not found",
			}
		},
	})

	kubeClient := newMemorySecurityListClient(scheme, resource)
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
		NamespacedName: ctrlclient.ObjectKey{Name: "test-security-list-auth-shaped-404", Namespace: "default"},
	}, &corev1beta1.SecurityList{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredSecurityList(), osokcore.OSOKFinalizerName))

	events := drainSecurityListEvents(recorder)
	assertSecurityListEventContains(t, events, "Removed finalizer")
	assertNoSecurityListEventContains(t, events, "Failed to delete resource")
}

func TestReconcileDelete_ReleasesFinalizerOnUnambiguousNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1beta1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	resource := &corev1beta1.SecurityList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.oracle.com/v1beta1",
			Kind:       "SecurityList",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-security-list",
			Namespace:         "default",
			Finalizers:        []string{osokcore.OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
		Status: corev1beta1.SecurityListStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.securitylist.oc1..delete"),
			},
		},
	}

	manager := newSecurityListTestManager(&fakeSecurityListOCIClient{
		deleteFn: func(_ context.Context, req coresdk.DeleteSecurityListRequest) (coresdk.DeleteSecurityListResponse, error) {
			assert.Equal(t, "ocid1.securitylist.oc1..delete", *req.SecurityListId)
			return coresdk.DeleteSecurityListResponse{}, nil
		},
		getFn: func(_ context.Context, _ coresdk.GetSecurityListRequest) (coresdk.GetSecurityListResponse, error) {
			return coresdk.GetSecurityListResponse{}, fakeSecurityListServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    "resource not found",
			}
		},
	})

	kubeClient := newMemorySecurityListClient(scheme, resource)
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
		NamespacedName: ctrlclient.ObjectKey{Name: "test-security-list", Namespace: "default"},
	}, &corev1beta1.SecurityList{})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.False(t, osokcore.HasFinalizer(kubeClient.StoredSecurityList(), osokcore.OSOKFinalizerName))

	events := drainSecurityListEvents(recorder)
	assertSecurityListEventContains(t, events, "Removed finalizer")
	assertNoSecurityListEventContains(t, events, "Failed to delete resource")
}

func TestConvertSpecRules_OmitEmptyNestedOptionalObjects(t *testing.T) {
	egress := convertSpecEgressRulesToOCI([]corev1beta1.SecurityListEgressSecurityRule{{
		Destination: "0.0.0.0/0",
		Protocol:    "6",
	}})
	ingress := convertSpecIngressRulesToOCI([]corev1beta1.SecurityListIngressSecurityRule{{
		Source:   "10.0.0.0/16",
		Protocol: "17",
	}})

	assert.Len(t, egress, 1)
	assert.Nil(t, egress[0].IcmpOptions)
	assert.Nil(t, egress[0].TcpOptions)
	assert.Nil(t, egress[0].UdpOptions)
	assert.Nil(t, egress[0].IsStateless)
	assert.Len(t, ingress, 1)
	assert.Nil(t, ingress[0].IcmpOptions)
	assert.Nil(t, ingress[0].TcpOptions)
	assert.Nil(t, ingress[0].UdpOptions)
	assert.Nil(t, ingress[0].IsStateless)
}

func drainSecurityListEvents(recorder *record.FakeRecorder) []string {
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

func assertSecurityListEventContains(t *testing.T, events []string, want string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, want) {
			return
		}
	}
	t.Fatalf("events %v do not contain %q", events, want)
}

func assertNoSecurityListEventContains(t *testing.T, events []string, unexpected string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, unexpected) {
			t.Fatalf("events %v unexpectedly contain %q", events, unexpected)
		}
	}
}

type memorySecurityListClient struct {
	ctrlclient.Client
	stored ctrlclient.Object
}

func newMemorySecurityListClient(scheme *runtime.Scheme, obj ctrlclient.Object) *memorySecurityListClient {
	return &memorySecurityListClient{
		Client: ctrlclientfake.NewClientBuilder().WithScheme(scheme).Build(),
		stored: obj.DeepCopyObject().(ctrlclient.Object),
	}
}

func (c *memorySecurityListClient) Get(_ context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, _ ...ctrlclient.GetOption) error {
	if c.stored == nil || c.stored.GetName() != key.Name || c.stored.GetNamespace() != key.Namespace {
		return apierrors.NewNotFound(schema.GroupResource{Group: "core.oracle.com", Resource: "securitylists"}, key.Name)
	}

	value := reflect.ValueOf(obj)
	source := reflect.ValueOf(c.stored.DeepCopyObject())
	if value.Kind() != reflect.Ptr || source.Kind() != reflect.Ptr {
		return stderrors.New("memory client requires pointer objects")
	}
	value.Elem().Set(source.Elem())
	return nil
}

func (c *memorySecurityListClient) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.UpdateOption) error {
	c.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (c *memorySecurityListClient) StoredSecurityList() *corev1beta1.SecurityList {
	return c.stored.DeepCopyObject().(*corev1beta1.SecurityList)
}

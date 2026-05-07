/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vbinstance

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	visualbuildersdk "github.com/oracle/oci-go-sdk/v65/visualbuilder"
	visualbuilderv1beta1 "github.com/oracle/oci-service-operator/api/visualbuilder/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeVbInstanceOCIClient struct {
	createFn      func(context.Context, visualbuildersdk.CreateVbInstanceRequest) (visualbuildersdk.CreateVbInstanceResponse, error)
	getFn         func(context.Context, visualbuildersdk.GetVbInstanceRequest) (visualbuildersdk.GetVbInstanceResponse, error)
	listFn        func(context.Context, visualbuildersdk.ListVbInstancesRequest) (visualbuildersdk.ListVbInstancesResponse, error)
	updateFn      func(context.Context, visualbuildersdk.UpdateVbInstanceRequest) (visualbuildersdk.UpdateVbInstanceResponse, error)
	deleteFn      func(context.Context, visualbuildersdk.DeleteVbInstanceRequest) (visualbuildersdk.DeleteVbInstanceResponse, error)
	workRequestFn func(context.Context, visualbuildersdk.GetWorkRequestRequest) (visualbuildersdk.GetWorkRequestResponse, error)

	createRequests      []visualbuildersdk.CreateVbInstanceRequest
	getRequests         []visualbuildersdk.GetVbInstanceRequest
	listRequests        []visualbuildersdk.ListVbInstancesRequest
	updateRequests      []visualbuildersdk.UpdateVbInstanceRequest
	deleteRequests      []visualbuildersdk.DeleteVbInstanceRequest
	workRequestRequests []visualbuildersdk.GetWorkRequestRequest
}

func (f *fakeVbInstanceOCIClient) CreateVbInstance(ctx context.Context, request visualbuildersdk.CreateVbInstanceRequest) (visualbuildersdk.CreateVbInstanceResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return visualbuildersdk.CreateVbInstanceResponse{}, nil
}

func (f *fakeVbInstanceOCIClient) GetVbInstance(ctx context.Context, request visualbuildersdk.GetVbInstanceRequest) (visualbuildersdk.GetVbInstanceResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return visualbuildersdk.GetVbInstanceResponse{}, errortest.NewServiceError(404, "NotFound", "missing vb instance")
}

func (f *fakeVbInstanceOCIClient) ListVbInstances(ctx context.Context, request visualbuildersdk.ListVbInstancesRequest) (visualbuildersdk.ListVbInstancesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return visualbuildersdk.ListVbInstancesResponse{}, nil
}

func (f *fakeVbInstanceOCIClient) UpdateVbInstance(ctx context.Context, request visualbuildersdk.UpdateVbInstanceRequest) (visualbuildersdk.UpdateVbInstanceResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return visualbuildersdk.UpdateVbInstanceResponse{}, nil
}

func (f *fakeVbInstanceOCIClient) DeleteVbInstance(ctx context.Context, request visualbuildersdk.DeleteVbInstanceRequest) (visualbuildersdk.DeleteVbInstanceResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return visualbuildersdk.DeleteVbInstanceResponse{}, nil
}

func (f *fakeVbInstanceOCIClient) GetWorkRequest(ctx context.Context, request visualbuildersdk.GetWorkRequestRequest) (visualbuildersdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, request)
	}
	return visualbuildersdk.GetWorkRequestResponse{}, nil
}

func TestReviewedVbInstanceRuntimeSemanticsEncodesReviewedContract(t *testing.T) {
	t.Parallel()

	semantics := reviewedVbInstanceRuntimeSemantics()
	if semantics == nil {
		t.Fatal("reviewedVbInstanceRuntimeSemantics() = nil")
	}
	if semantics.FormalService != "visualbuilder" {
		t.Fatalf("FormalService = %q, want visualbuilder", semantics.FormalService)
	}
	if semantics.FormalSlug != "vbinstance" {
		t.Fatalf("FormalSlug = %q, want vbinstance", semantics.FormalSlug)
	}
	if semantics.Async == nil || semantics.Async.WorkRequest == nil {
		t.Fatalf("Async.WorkRequest = %#v, want service-sdk workrequest semantics", semantics.Async)
	}
	requireVbInstanceStringSliceEqual(t, "Async.WorkRequest.Phases", semantics.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	requireVbInstanceStringSliceEqual(t, "Lifecycle.ProvisioningStates", semantics.Lifecycle.ProvisioningStates, []string{"CREATING"})
	requireVbInstanceStringSliceEqual(t, "Lifecycle.UpdatingStates", semantics.Lifecycle.UpdatingStates, []string{"UPDATING"})
	requireVbInstanceStringSliceEqual(t, "Lifecycle.ActiveStates", semantics.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"})
	requireVbInstanceStringSliceEqual(t, "List.MatchFields", semantics.List.MatchFields, []string{"compartmentId", "displayName"})
	requireVbInstanceStringSliceEqual(t, "Mutation.Mutable", semantics.Mutation.Mutable, []string{
		"alternateCustomEndpoints.certificateSecretId",
		"alternateCustomEndpoints.hostname",
		"customEndpoint.certificateSecretId",
		"customEndpoint.hostname",
		"definedTags",
		"displayName",
		"freeformTags",
		"isVisualBuilderEnabled",
		"networkEndpointDetails.allowlistedHttpIps",
		"networkEndpointDetails.allowlistedHttpVcns.allowlistedIpCidrs",
		"networkEndpointDetails.allowlistedHttpVcns.id",
		"networkEndpointDetails.networkEndpointType",
		"networkEndpointDetails.networkSecurityGroupIds",
		"networkEndpointDetails.subnetId",
		"nodeCount",
	})
	requireVbInstanceStringSliceEqual(t, "Mutation.ForceNew", semantics.Mutation.ForceNew, []string{
		"compartmentId",
		"consumptionModel",
		"networkEndpointDetails.privateEndpointIp",
	})
	if semantics.CreateFollowUp.Strategy != "GetWorkRequest -> GetVbInstance" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetVbInstance", semantics.CreateFollowUp.Strategy)
	}
	if semantics.UpdateFollowUp.Strategy != "GetWorkRequest -> GetVbInstance" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetVbInstance", semantics.UpdateFollowUp.Strategy)
	}
	if semantics.DeleteFollowUp.Strategy != "GetWorkRequest -> GetVbInstance/ListVbInstances confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want GetWorkRequest -> GetVbInstance/ListVbInstances confirm-delete", semantics.DeleteFollowUp.Strategy)
	}
}

func TestBuildVbInstanceCreateBodyPreservesPolymorphicPublicEndpoint(t *testing.T) {
	t.Parallel()

	resource := newTestVbInstance()
	resource.Spec.IsVisualBuilderEnabled = true
	resource.Spec.NetworkEndpointDetails = visualbuilderv1beta1.VbInstanceNetworkEndpointDetails{
		JsonData:            "ignored-helper-field",
		NetworkEndpointType: "PUBLIC",
		AllowlistedHttpIps:  []string{"10.0.0.0/24"},
		AllowlistedHttpVcns: []visualbuilderv1beta1.VbInstanceNetworkEndpointDetailsAllowlistedHttpVcn{
			{
				Id:                 "ocid1.vcn.oc1..allowed",
				AllowlistedIpCidrs: []string{"10.0.1.0/24"},
			},
		},
	}

	details, err := buildVbInstanceCreateBody(resource)
	if err != nil {
		t.Fatalf("buildVbInstanceCreateBody() error = %v", err)
	}

	publicDetails, ok := details.NetworkEndpointDetails.(visualbuildersdk.PublicEndpointDetails)
	if !ok {
		t.Fatalf("Create.NetworkEndpointDetails = %T, want visualbuilder.PublicEndpointDetails", details.NetworkEndpointDetails)
	}
	if len(publicDetails.AllowlistedHttpIps) != 1 || publicDetails.AllowlistedHttpIps[0] != "10.0.0.0/24" {
		t.Fatalf("AllowlistedHttpIps = %#v, want preserved public allowlist", publicDetails.AllowlistedHttpIps)
	}
	if len(publicDetails.AllowlistedHttpVcns) != 1 || vbInstanceStringValue(publicDetails.AllowlistedHttpVcns[0].Id) != "ocid1.vcn.oc1..allowed" {
		t.Fatalf("AllowlistedHttpVcns = %#v, want preserved VCN allowlist", publicDetails.AllowlistedHttpVcns)
	}
	if details.IsVisualBuilderEnabled == nil || !*details.IsVisualBuilderEnabled {
		t.Fatalf("Create.IsVisualBuilderEnabled = %#v, want true", details.IsVisualBuilderEnabled)
	}

	body := vbInstanceSerializedRequestBody(t, visualbuildersdk.CreateVbInstanceRequest{
		CreateVbInstanceDetails: details,
	}, http.MethodPost, "/vbInstances")
	requireVbInstanceContains(t, body, `"networkEndpointType":"PUBLIC"`)
	requireVbInstanceContains(t, body, `"allowlistedHttpIps":["10.0.0.0/24"]`)
	requireVbInstanceContains(t, body, `"allowlistedHttpVcns":[`)
	requireVbInstanceContains(t, body, `"id":"ocid1.vcn.oc1..allowed"`)
	requireVbInstanceContains(t, body, `"allowlistedIpCidrs":["10.0.1.0/24"]`)
	requireVbInstanceContains(t, body, `"isVisualBuilderEnabled":true`)
	if strings.Contains(body, `"jsonData"`) {
		t.Fatalf("request body unexpectedly exposed jsonData helper field: %s", body)
	}
}

func TestVbInstanceCreateOrUpdateBindsExistingActiveMatchWithoutCreate(t *testing.T) {
	t.Parallel()

	resource := newMinimalTestVbInstance()
	client := &fakeVbInstanceOCIClient{}
	client.listFn = func(_ context.Context, request visualbuildersdk.ListVbInstancesRequest) (visualbuildersdk.ListVbInstancesResponse, error) {
		requireVbInstanceStringPtr(t, "List.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
		requireVbInstanceStringPtr(t, "List.DisplayName", request.DisplayName, resource.Spec.DisplayName)
		return visualbuildersdk.ListVbInstancesResponse{
			VbInstanceSummaryCollection: visualbuildersdk.VbInstanceSummaryCollection{
				Items: []visualbuildersdk.VbInstanceSummary{
					testVbInstanceSummaryFromResource("ocid1.vbinstance.oc1..bound", resource, visualbuildersdk.VbInstanceSummaryLifecycleStateActive),
				},
			},
		}, nil
	}
	client.getFn = func(_ context.Context, request visualbuildersdk.GetVbInstanceRequest) (visualbuildersdk.GetVbInstanceResponse, error) {
		requireVbInstanceStringPtr(t, "Get.VbInstanceId", request.VbInstanceId, "ocid1.vbinstance.oc1..bound")
		return visualbuildersdk.GetVbInstanceResponse{
			VbInstance: observedVbInstanceFromResource("ocid1.vbinstance.oc1..bound", resource, visualbuildersdk.VbInstanceLifecycleStateActive),
		}, nil
	}

	response, err := newVbInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want converged success", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateVbInstance() calls = %d, want 0 after list bind", len(client.createRequests))
	}
	if resource.Status.Id != "ocid1.vbinstance.oc1..bound" {
		t.Fatalf("status.id = %q, want bound OCID", resource.Status.Id)
	}
	if resource.Status.OsokStatus.Ocid != shared.OCID("ocid1.vbinstance.oc1..bound") {
		t.Fatalf("status.status.ocid = %q, want bound OCID", resource.Status.OsokStatus.Ocid)
	}
}

func TestVbInstanceCreateOrUpdateTracksCreateWorkRequestAndRecoversID(t *testing.T) {
	t.Parallel()

	resource := newMinimalTestVbInstance()
	client := &fakeVbInstanceOCIClient{}
	workRequestStatus := visualbuildersdk.WorkRequestStatusInProgress
	workRequestAction := visualbuildersdk.WorkRequestResourceActionTypeInProgress

	client.createFn = func(_ context.Context, request visualbuildersdk.CreateVbInstanceRequest) (visualbuildersdk.CreateVbInstanceResponse, error) {
		requireVbInstanceStringPtr(t, "Create.DisplayName", request.DisplayName, resource.Spec.DisplayName)
		return visualbuildersdk.CreateVbInstanceResponse{
			OpcRequestId:     common.String("opc-create"),
			OpcWorkRequestId: common.String("wr-create"),
		}, nil
	}
	client.workRequestFn = func(_ context.Context, request visualbuildersdk.GetWorkRequestRequest) (visualbuildersdk.GetWorkRequestResponse, error) {
		requireVbInstanceStringPtr(t, "GetWorkRequest.WorkRequestId", request.WorkRequestId, "wr-create")
		return visualbuildersdk.GetWorkRequestResponse{
			WorkRequest: testVbInstanceWorkRequest("wr-create", visualbuildersdk.WorkRequestOperationTypeCreateVbInstance, workRequestStatus, workRequestAction, "ocid1.vbinstance.oc1..created"),
		}, nil
	}
	client.getFn = func(_ context.Context, request visualbuildersdk.GetVbInstanceRequest) (visualbuildersdk.GetVbInstanceResponse, error) {
		requireVbInstanceStringPtr(t, "Get.VbInstanceId", request.VbInstanceId, "ocid1.vbinstance.oc1..created")
		return visualbuildersdk.GetVbInstanceResponse{
			VbInstance: observedVbInstanceFromResource("ocid1.vbinstance.oc1..created", resource, visualbuildersdk.VbInstanceLifecycleStateActive),
		}, nil
	}

	serviceClient := newVbInstanceServiceClientWithOCIClient(client)

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() pending error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() pending response = %#v, want successful requeue", response)
	}
	requireCurrentVbInstanceAsync(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create")
	requireVbInstanceOpcRequestID(t, resource, "opc-create")

	workRequestStatus = visualbuildersdk.WorkRequestStatusSucceeded
	workRequestAction = visualbuildersdk.WorkRequestResourceActionTypeCreated

	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() resume error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() resume response = %#v, want converged success", response)
	}
	if len(client.getRequests) == 0 {
		t.Fatal("GetVbInstance() calls = 0, want reread after work request success")
	}
	if resource.Status.Id != "ocid1.vbinstance.oc1..created" {
		t.Fatalf("status.id = %q, want recovered OCID", resource.Status.Id)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after convergence", resource.Status.OsokStatus.Async.Current)
	}
}

func TestBuildVbInstanceUpdateBodyPreservesExplicitEmptyClears(t *testing.T) {
	t.Parallel()

	resource := newTestVbInstance()
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	resource.Spec.AlternateCustomEndpoints = []visualbuilderv1beta1.VbInstanceAlternateCustomEndpoint{}
	resource.Spec.NetworkEndpointDetails = visualbuilderv1beta1.VbInstanceNetworkEndpointDetails{
		NetworkEndpointType: "PUBLIC",
		AllowlistedHttpIps:  []string{},
		AllowlistedHttpVcns: []visualbuilderv1beta1.VbInstanceNetworkEndpointDetailsAllowlistedHttpVcn{},
	}

	current := observedVbInstanceFromResource("ocid1.vbinstance.oc1..existing", newTestVbInstance(), visualbuildersdk.VbInstanceLifecycleStateActive)
	current.FreeformTags = map[string]string{"remove": "me"}
	current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
	current.AlternateCustomEndpoints = []visualbuildersdk.CustomEndpointDetails{
		{
			Hostname:            common.String("alt.visualbuilder.example.com"),
			CertificateSecretId: common.String("ocid1.vaultsecret.oc1..alt"),
		},
	}
	current.NetworkEndpointDetails = visualbuildersdk.PublicEndpointDetails{
		AllowlistedHttpIps: []string{"10.10.0.0/24"},
		AllowlistedHttpVcns: []visualbuildersdk.VirtualCloudNetwork{
			{
				Id:                 common.String("ocid1.vcn.oc1..existing"),
				AllowlistedIpCidrs: []string{"10.11.0.0/24"},
			},
		},
	}

	details, updateNeeded, err := buildVbInstanceUpdateBody(resource, visualbuildersdk.GetVbInstanceResponse{VbInstance: current})
	if err != nil {
		t.Fatalf("buildVbInstanceUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildVbInstanceUpdateBody() updateNeeded = false, want explicit clear intent")
	}
	if len(details.FreeformTags) != 0 {
		t.Fatalf("FreeformTags = %#v, want explicit empty map clear", details.FreeformTags)
	}
	if len(details.DefinedTags) != 0 {
		t.Fatalf("DefinedTags = %#v, want explicit empty map clear", details.DefinedTags)
	}
	if len(details.AlternateCustomEndpoints) != 0 {
		t.Fatalf("AlternateCustomEndpoints = %#v, want explicit empty slice clear", details.AlternateCustomEndpoints)
	}

	publicDetails, ok := details.NetworkEndpointDetails.(visualbuildersdk.UpdatePublicEndpointDetails)
	if !ok {
		t.Fatalf("NetworkEndpointDetails = %T, want visualbuilder.UpdatePublicEndpointDetails", details.NetworkEndpointDetails)
	}
	if len(publicDetails.AllowlistedHttpIps) != 0 {
		t.Fatalf("AllowlistedHttpIps = %#v, want explicit empty slice clear", publicDetails.AllowlistedHttpIps)
	}
	if len(publicDetails.AllowlistedHttpVcns) != 0 {
		t.Fatalf("AllowlistedHttpVcns = %#v, want explicit empty slice clear", publicDetails.AllowlistedHttpVcns)
	}

	body := vbInstanceSerializedRequestBody(t, visualbuildersdk.UpdateVbInstanceRequest{
		VbInstanceId:            common.String("ocid1.vbinstance.oc1..existing"),
		UpdateVbInstanceDetails: details,
	}, http.MethodPut, "/vbInstances/ocid1.vbinstance.oc1..existing")
	requireVbInstanceContains(t, body, `"freeformTags":{}`)
	requireVbInstanceContains(t, body, `"definedTags":{}`)
	requireVbInstanceContains(t, body, `"alternateCustomEndpoints":[]`)
	requireVbInstanceContains(t, body, `"allowlistedHttpIps":[]`)
	requireVbInstanceContains(t, body, `"allowlistedHttpVcns":[]`)
}

func TestBuildVbInstanceUpdateBodyDoesNotDisableOrReplayIdcsOpenID(t *testing.T) {
	t.Parallel()

	resource := newMinimalTestVbInstance()
	resource.Spec.IsVisualBuilderEnabled = false
	resource.Spec.IdcsOpenId = "opaque-token"

	current := observedVbInstanceFromResource("ocid1.vbinstance.oc1..existing", resource, visualbuildersdk.VbInstanceLifecycleStateActive)
	current.IsVisualBuilderEnabled = common.Bool(true)

	details, updateNeeded, err := buildVbInstanceUpdateBody(resource, visualbuildersdk.GetVbInstanceResponse{VbInstance: current})
	if err != nil {
		t.Fatalf("buildVbInstanceUpdateBody() error = %v", err)
	}
	if updateNeeded {
		t.Fatalf("buildVbInstanceUpdateBody() updateNeeded = true, want no-op when only desired drift is disable/idcsOpenId replay: %#v", details)
	}
	if details.IsVisualBuilderEnabled != nil {
		t.Fatalf("IsVisualBuilderEnabled = %#v, want nil to avoid disable", details.IsVisualBuilderEnabled)
	}
	if details.IdcsOpenId != nil {
		t.Fatalf("IdcsOpenId = %#v, want nil because OCI does not echo it back for parity", details.IdcsOpenId)
	}
}

func TestVbInstanceDeleteTracksWorkRequestUntilConfirmed(t *testing.T) {
	t.Parallel()

	resource := newTestVbInstance()
	resource.Status.Id = "ocid1.vbinstance.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.vbinstance.oc1..existing")

	client := &fakeVbInstanceOCIClient{}
	client.getFn = func(_ context.Context, request visualbuildersdk.GetVbInstanceRequest) (visualbuildersdk.GetVbInstanceResponse, error) {
		requireVbInstanceStringPtr(t, "Get.VbInstanceId", request.VbInstanceId, "ocid1.vbinstance.oc1..existing")
		return visualbuildersdk.GetVbInstanceResponse{
			VbInstance: observedVbInstanceFromResource("ocid1.vbinstance.oc1..existing", resource, visualbuildersdk.VbInstanceLifecycleStateActive),
		}, nil
	}
	client.deleteFn = func(_ context.Context, request visualbuildersdk.DeleteVbInstanceRequest) (visualbuildersdk.DeleteVbInstanceResponse, error) {
		requireVbInstanceStringPtr(t, "Delete.VbInstanceId", request.VbInstanceId, "ocid1.vbinstance.oc1..existing")
		return visualbuildersdk.DeleteVbInstanceResponse{
			OpcRequestId:     common.String("opc-delete"),
			OpcWorkRequestId: common.String("wr-delete"),
		}, nil
	}
	client.workRequestFn = func(_ context.Context, request visualbuildersdk.GetWorkRequestRequest) (visualbuildersdk.GetWorkRequestResponse, error) {
		requireVbInstanceStringPtr(t, "GetWorkRequest.WorkRequestId", request.WorkRequestId, "wr-delete")
		return visualbuildersdk.GetWorkRequestResponse{
			WorkRequest: testVbInstanceWorkRequest("wr-delete", visualbuildersdk.WorkRequestOperationTypeDeleteVbInstance, visualbuildersdk.WorkRequestStatusInProgress, visualbuildersdk.WorkRequestResourceActionTypeDeleted, "ocid1.vbinstance.oc1..existing"),
		}, nil
	}

	deleted, err := newVbInstanceServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want pending while delete work request is in progress")
	}
	requireCurrentVbInstanceAsync(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete")
	requireVbInstanceOpcRequestID(t, resource, "opc-delete")
}

type vbInstanceRequestBodyBuilder interface {
	HTTPRequest(string, string, *common.OCIReadSeekCloser, map[string]string) (http.Request, error)
}

func vbInstanceSerializedRequestBody(t *testing.T, request vbInstanceRequestBodyBuilder, method string, path string) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}
	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request.Body) error = %v", err)
	}
	return string(body)
}

func newTestVbInstance() *visualbuilderv1beta1.VbInstance {
	return &visualbuilderv1beta1.VbInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vb-instance-sample",
			Namespace: "default",
		},
		Spec: visualbuilderv1beta1.VbInstanceSpec{
			DisplayName:   "vb-instance-sample",
			CompartmentId: "ocid1.compartment.oc1..visualbuilder",
			NodeCount:     2,
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			CustomEndpoint: visualbuilderv1beta1.VbInstanceCustomEndpoint{
				Hostname:            "vb.visualbuilder.example.com",
				CertificateSecretId: "ocid1.vaultsecret.oc1..primary",
			},
			AlternateCustomEndpoints: []visualbuilderv1beta1.VbInstanceAlternateCustomEndpoint{
				{
					Hostname:            "alt.visualbuilder.example.com",
					CertificateSecretId: "ocid1.vaultsecret.oc1..alt",
				},
			},
			ConsumptionModel: "UCM",
			NetworkEndpointDetails: visualbuilderv1beta1.VbInstanceNetworkEndpointDetails{
				NetworkEndpointType: "PRIVATE",
				SubnetId:            "ocid1.subnet.oc1..visualbuilder",
				NetworkSecurityGroupIds: []string{
					"ocid1.nsg.oc1..visualbuilder",
				},
				PrivateEndpointIp: "10.0.0.15",
			},
		},
	}
}

func newMinimalTestVbInstance() *visualbuilderv1beta1.VbInstance {
	return &visualbuilderv1beta1.VbInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vb-instance-minimal",
			Namespace: "default",
		},
		Spec: visualbuilderv1beta1.VbInstanceSpec{
			DisplayName:   "vb-instance-minimal",
			CompartmentId: "ocid1.compartment.oc1..visualbuilder",
			NodeCount:     2,
		},
	}
}

func observedVbInstanceFromResource(
	id string,
	resource *visualbuilderv1beta1.VbInstance,
	state visualbuildersdk.VbInstanceLifecycleStateEnum,
) visualbuildersdk.VbInstance {
	current := visualbuildersdk.VbInstance{
		Id:             common.String(id),
		DisplayName:    common.String(resource.Spec.DisplayName),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		LifecycleState: state,
		InstanceUrl:    common.String("https://vb.visualbuilder.example.com"),
		NodeCount:      common.Int(resource.Spec.NodeCount),
		FreeformTags:   mapsClone(resource.Spec.FreeformTags),
		DefinedTags:    vbInstanceDefinedTagsFromSpec(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {"free-tier-retained": "true"},
		},
		ConsumptionModel: visualbuildersdk.VbInstanceConsumptionModelEnum(resource.Spec.ConsumptionModel),
	}
	if resource.Spec.IsVisualBuilderEnabled {
		current.IsVisualBuilderEnabled = common.Bool(true)
	} else {
		current.IsVisualBuilderEnabled = common.Bool(false)
	}
	if endpoint, ok, err := buildVbInstanceCreateCustomEndpoint(resource.Spec.CustomEndpoint); err == nil && ok {
		current.CustomEndpoint = &visualbuildersdk.CustomEndpointDetails{
			Hostname:                 endpoint.Hostname,
			CertificateSecretId:      endpoint.CertificateSecretId,
			CertificateSecretVersion: common.Int(1),
		}
	}
	if resource.Spec.AlternateCustomEndpoints != nil {
		current.AlternateCustomEndpoints = make([]visualbuildersdk.CustomEndpointDetails, 0, len(resource.Spec.AlternateCustomEndpoints))
		for _, item := range resource.Spec.AlternateCustomEndpoints {
			current.AlternateCustomEndpoints = append(current.AlternateCustomEndpoints, visualbuildersdk.CustomEndpointDetails{
				Hostname:                 common.String(item.Hostname),
				CertificateSecretId:      common.String(item.CertificateSecretId),
				CertificateSecretVersion: common.Int(1),
			})
		}
	}
	if endpoint, ok, err := buildVbInstanceCreateNetworkEndpointDetails(resource.Spec.NetworkEndpointDetails); err == nil && ok {
		current.NetworkEndpointDetails = endpoint
	}
	return current
}

func testVbInstanceSummaryFromResource(
	id string,
	resource *visualbuilderv1beta1.VbInstance,
	state visualbuildersdk.VbInstanceSummaryLifecycleStateEnum,
) visualbuildersdk.VbInstanceSummary {
	current := visualbuildersdk.VbInstanceSummary{
		Id:             common.String(id),
		DisplayName:    common.String(resource.Spec.DisplayName),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		LifecycleState: state,
		InstanceUrl:    common.String("https://vb.visualbuilder.example.com"),
		NodeCount:      common.Int(resource.Spec.NodeCount),
		FreeformTags:   mapsClone(resource.Spec.FreeformTags),
		DefinedTags:    vbInstanceDefinedTagsFromSpec(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {"free-tier-retained": "true"},
		},
		ConsumptionModel: visualbuildersdk.VbInstanceSummaryConsumptionModelEnum(resource.Spec.ConsumptionModel),
	}
	if resource.Spec.IsVisualBuilderEnabled {
		current.IsVisualBuilderEnabled = common.Bool(true)
	} else {
		current.IsVisualBuilderEnabled = common.Bool(false)
	}
	if endpoint, ok, err := buildVbInstanceCreateCustomEndpoint(resource.Spec.CustomEndpoint); err == nil && ok {
		current.CustomEndpoint = &visualbuildersdk.CustomEndpointDetails{
			Hostname:                 endpoint.Hostname,
			CertificateSecretId:      endpoint.CertificateSecretId,
			CertificateSecretVersion: common.Int(1),
		}
	}
	if resource.Spec.AlternateCustomEndpoints != nil {
		current.AlternateCustomEndpoints = make([]visualbuildersdk.CustomEndpointDetails, 0, len(resource.Spec.AlternateCustomEndpoints))
		for _, item := range resource.Spec.AlternateCustomEndpoints {
			current.AlternateCustomEndpoints = append(current.AlternateCustomEndpoints, visualbuildersdk.CustomEndpointDetails{
				Hostname:                 common.String(item.Hostname),
				CertificateSecretId:      common.String(item.CertificateSecretId),
				CertificateSecretVersion: common.Int(1),
			})
		}
	}
	if endpoint, ok, err := buildVbInstanceCreateNetworkEndpointDetails(resource.Spec.NetworkEndpointDetails); err == nil && ok {
		current.NetworkEndpointDetails = endpoint
	}
	return current
}

func testVbInstanceWorkRequest(
	id string,
	operationType visualbuildersdk.WorkRequestOperationTypeEnum,
	status visualbuildersdk.WorkRequestStatusEnum,
	action visualbuildersdk.WorkRequestResourceActionTypeEnum,
	resourceID string,
) visualbuildersdk.WorkRequest {
	accepted := common.SDKTime{Time: time.Unix(1700000000, 0)}
	return visualbuildersdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operationType,
		Status:          status,
		CompartmentId:   common.String("ocid1.compartment.oc1..visualbuilder"),
		PercentComplete: common.Float32(25),
		TimeAccepted:    &accepted,
		Resources: []visualbuildersdk.WorkRequestResource{
			{
				EntityType: common.String("VbInstance"),
				ActionType: action,
				Identifier: common.String(resourceID),
				EntityUri:  common.String("/vbInstances/" + resourceID),
			},
		},
	}
}

func requireCurrentVbInstanceAsync(
	t *testing.T,
	resource *visualbuilderv1beta1.VbInstance,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracked work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func requireVbInstanceOpcRequestID(t *testing.T, resource *visualbuilderv1beta1.VbInstance, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestID = %q, want %q", got, want)
	}
}

func requireVbInstanceStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if vbInstanceStringValue(got) != want {
		t.Fatalf("%s = %v, want %q", field, got, want)
	}
}

func requireVbInstanceStringSliceEqual(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q", field, i, got[i], want[i])
		}
	}
}

func requireVbInstanceContains(t *testing.T, body string, want string) {
	t.Helper()
	if !strings.Contains(body, want) {
		t.Fatalf("request body %s does not contain %s", body, want)
	}
}

func mapsClone(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

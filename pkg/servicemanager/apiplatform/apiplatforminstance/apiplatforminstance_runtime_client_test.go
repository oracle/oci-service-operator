/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apiplatforminstance

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	apiplatformsdk "github.com/oracle/oci-go-sdk/v65/apiplatform"
	"github.com/oracle/oci-go-sdk/v65/common"
	apiplatformv1beta1 "github.com/oracle/oci-service-operator/api/apiplatform/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeApiPlatformInstanceOCIClient struct {
	createFn func(context.Context, apiplatformsdk.CreateApiPlatformInstanceRequest) (apiplatformsdk.CreateApiPlatformInstanceResponse, error)
	getFn    func(context.Context, apiplatformsdk.GetApiPlatformInstanceRequest) (apiplatformsdk.GetApiPlatformInstanceResponse, error)
	listFn   func(context.Context, apiplatformsdk.ListApiPlatformInstancesRequest) (apiplatformsdk.ListApiPlatformInstancesResponse, error)
	updateFn func(context.Context, apiplatformsdk.UpdateApiPlatformInstanceRequest) (apiplatformsdk.UpdateApiPlatformInstanceResponse, error)
	deleteFn func(context.Context, apiplatformsdk.DeleteApiPlatformInstanceRequest) (apiplatformsdk.DeleteApiPlatformInstanceResponse, error)
}

func (f *fakeApiPlatformInstanceOCIClient) CreateApiPlatformInstance(
	ctx context.Context,
	req apiplatformsdk.CreateApiPlatformInstanceRequest,
) (apiplatformsdk.CreateApiPlatformInstanceResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return apiplatformsdk.CreateApiPlatformInstanceResponse{}, nil
}

func (f *fakeApiPlatformInstanceOCIClient) GetApiPlatformInstance(
	ctx context.Context,
	req apiplatformsdk.GetApiPlatformInstanceRequest,
) (apiplatformsdk.GetApiPlatformInstanceResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return apiplatformsdk.GetApiPlatformInstanceResponse{}, nil
}

func (f *fakeApiPlatformInstanceOCIClient) ListApiPlatformInstances(
	ctx context.Context,
	req apiplatformsdk.ListApiPlatformInstancesRequest,
) (apiplatformsdk.ListApiPlatformInstancesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return apiplatformsdk.ListApiPlatformInstancesResponse{}, nil
}

func (f *fakeApiPlatformInstanceOCIClient) UpdateApiPlatformInstance(
	ctx context.Context,
	req apiplatformsdk.UpdateApiPlatformInstanceRequest,
) (apiplatformsdk.UpdateApiPlatformInstanceResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return apiplatformsdk.UpdateApiPlatformInstanceResponse{}, nil
}

func (f *fakeApiPlatformInstanceOCIClient) DeleteApiPlatformInstance(
	ctx context.Context,
	req apiplatformsdk.DeleteApiPlatformInstanceRequest,
) (apiplatformsdk.DeleteApiPlatformInstanceResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return apiplatformsdk.DeleteApiPlatformInstanceResponse{}, nil
}

func testApiPlatformInstanceClient(fake *fakeApiPlatformInstanceOCIClient) ApiPlatformInstanceServiceClient {
	return newApiPlatformInstanceServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func makeApiPlatformInstanceResource() *apiplatformv1beta1.ApiPlatformInstance {
	return &apiplatformv1beta1.ApiPlatformInstance{
		Spec: apiplatformv1beta1.ApiPlatformInstanceSpec{
			Name:          "apip-alpha",
			CompartmentId: "ocid1.compartment.oc1..example",
			Description:   "desired description",
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKApiPlatformInstance(
	id string,
	compartmentID string,
	name string,
	description string,
	state apiplatformsdk.ApiPlatformInstanceLifecycleStateEnum,
) apiplatformsdk.ApiPlatformInstance {
	instance := apiplatformsdk.ApiPlatformInstance{
		Id:             common.String(id),
		Name:           common.String(name),
		CompartmentId:  common.String(compartmentID),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
	if description != "" {
		instance.Description = common.String(description)
	}
	return instance
}

func makeSDKApiPlatformInstanceSummary(
	id string,
	compartmentID string,
	name string,
	state apiplatformsdk.ApiPlatformInstanceLifecycleStateEnum,
) apiplatformsdk.ApiPlatformInstanceSummary {
	return apiplatformsdk.ApiPlatformInstanceSummary{
		Id:             common.String(id),
		Name:           common.String(name),
		CompartmentId:  common.String(compartmentID),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func requireTrailingApiPlatformCondition(
	t *testing.T,
	resource *apiplatformv1beta1.ApiPlatformInstance,
	want shared.OSOKConditionType,
) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatal("status.conditions = empty, want trailing condition")
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %q, want %q", got, want)
	}
}

func TestApiPlatformInstanceRuntimeHooksUseReviewedSemantics(t *testing.T) {
	t.Parallel()

	hooks := newApiPlatformInstanceRuntimeHooks(&ApiPlatformInstanceServiceManager{}, apiplatformsdk.ApiPlatformClient{})

	if !reflect.DeepEqual(hooks.List.Fields, apiPlatformInstanceListFields()) {
		t.Fatalf("list fields = %#v, want %#v", hooks.List.Fields, apiPlatformInstanceListFields())
	}
	if hooks.Semantics == nil {
		t.Fatal("semantics = nil, want reviewed semantics")
	}
	if got := hooks.Semantics.List; got == nil {
		t.Fatal("semantics.list = nil, want reviewed list semantics")
	} else if !reflect.DeepEqual(got.MatchFields, []string{"compartmentId", "name"}) {
		t.Fatalf("semantics.list.matchFields = %#v, want %#v", got.MatchFields, []string{"compartmentId", "name"})
	}
	if len(hooks.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("semantics.auxiliaryOperations = %#v, want reviewed omission of ChangeApiPlatformInstanceCompartment", hooks.Semantics.AuxiliaryOperations)
	}
}

func TestApiPlatformInstanceCreateOrUpdateCreatesAndRequeuesWhileCreating(t *testing.T) {
	t.Parallel()

	var createRequest apiplatformsdk.CreateApiPlatformInstanceRequest
	var getRequest apiplatformsdk.GetApiPlatformInstanceRequest

	client := testApiPlatformInstanceClient(&fakeApiPlatformInstanceOCIClient{
		createFn: func(_ context.Context, req apiplatformsdk.CreateApiPlatformInstanceRequest) (apiplatformsdk.CreateApiPlatformInstanceResponse, error) {
			createRequest = req
			return apiplatformsdk.CreateApiPlatformInstanceResponse{
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
				ApiPlatformInstance: makeSDKApiPlatformInstance(
					"ocid1.apiplatforminstance.oc1..created",
					"ocid1.compartment.oc1..example",
					"apip-alpha",
					"desired description",
					apiplatformsdk.ApiPlatformInstanceLifecycleStateCreating,
				),
			}, nil
		},
		getFn: func(_ context.Context, req apiplatformsdk.GetApiPlatformInstanceRequest) (apiplatformsdk.GetApiPlatformInstanceResponse, error) {
			getRequest = req
			return apiplatformsdk.GetApiPlatformInstanceResponse{
				ApiPlatformInstance: makeSDKApiPlatformInstance(
					"ocid1.apiplatforminstance.oc1..created",
					"ocid1.compartment.oc1..example",
					"apip-alpha",
					"desired description",
					apiplatformsdk.ApiPlatformInstanceLifecycleStateCreating,
				),
			}, nil
		},
	})

	resource := makeApiPlatformInstanceResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while create is still CREATING")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while lifecycle remains CREATING")
	}
	if createRequest.Name == nil || *createRequest.Name != resource.Spec.Name {
		t.Fatalf("create request name = %v, want %q", createRequest.Name, resource.Spec.Name)
	}
	if createRequest.CompartmentId == nil || *createRequest.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create request compartmentId = %v, want %q", createRequest.CompartmentId, resource.Spec.CompartmentId)
	}
	if getRequest.ApiPlatformInstanceId == nil || *getRequest.ApiPlatformInstanceId != "ocid1.apiplatforminstance.oc1..created" {
		t.Fatalf("get request apiPlatformInstanceId = %v, want created OCID", getRequest.ApiPlatformInstanceId)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.apiplatforminstance.oc1..created" {
		t.Fatalf("status.ocid = %q, want created OCID", got)
	}
	if got := resource.Status.LifecycleState; got != "CREATING" {
		t.Fatalf("status.lifecycleState = %q, want %q", got, "CREATING")
	}
	requireTrailingApiPlatformCondition(t, resource, shared.Provisioning)
}

func TestApiPlatformInstanceCreateOrUpdateReusesExistingActiveInstanceWithoutLifecycleFilter(t *testing.T) {
	t.Parallel()

	createCalled := false
	var listRequest apiplatformsdk.ListApiPlatformInstancesRequest
	var getRequest apiplatformsdk.GetApiPlatformInstanceRequest

	client := testApiPlatformInstanceClient(&fakeApiPlatformInstanceOCIClient{
		createFn: func(context.Context, apiplatformsdk.CreateApiPlatformInstanceRequest) (apiplatformsdk.CreateApiPlatformInstanceResponse, error) {
			createCalled = true
			t.Fatal("CreateApiPlatformInstance should not be called when an ACTIVE reusable match exists")
			return apiplatformsdk.CreateApiPlatformInstanceResponse{}, nil
		},
		listFn: func(_ context.Context, req apiplatformsdk.ListApiPlatformInstancesRequest) (apiplatformsdk.ListApiPlatformInstancesResponse, error) {
			listRequest = req
			return apiplatformsdk.ListApiPlatformInstancesResponse{
				ApiPlatformInstanceCollection: apiplatformsdk.ApiPlatformInstanceCollection{
					Items: []apiplatformsdk.ApiPlatformInstanceSummary{
						makeSDKApiPlatformInstanceSummary(
							"ocid1.apiplatforminstance.oc1..existing",
							"ocid1.compartment.oc1..example",
							"apip-alpha",
							apiplatformsdk.ApiPlatformInstanceLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req apiplatformsdk.GetApiPlatformInstanceRequest) (apiplatformsdk.GetApiPlatformInstanceResponse, error) {
			getRequest = req
			return apiplatformsdk.GetApiPlatformInstanceResponse{
				ApiPlatformInstance: makeSDKApiPlatformInstance(
					"ocid1.apiplatforminstance.oc1..existing",
					"ocid1.compartment.oc1..example",
					"apip-alpha",
					"desired description",
					apiplatformsdk.ApiPlatformInstanceLifecycleStateActive,
				),
			}, nil
		},
	})

	resource := makeApiPlatformInstanceResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when reusing an ACTIVE ApiPlatformInstance")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once the reusable ApiPlatformInstance is ACTIVE")
	}
	if createCalled {
		t.Fatal("CreateApiPlatformInstance should not be called when list reuse succeeds")
	}
	if listRequest.CompartmentId == nil || *listRequest.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("list request compartmentId = %v, want %q", listRequest.CompartmentId, resource.Spec.CompartmentId)
	}
	if listRequest.Name == nil || *listRequest.Name != resource.Spec.Name {
		t.Fatalf("list request name = %v, want %q", listRequest.Name, resource.Spec.Name)
	}
	if listRequest.LifecycleState != "" {
		t.Fatalf("list request lifecycleState = %q, want empty reviewed lookup filter", listRequest.LifecycleState)
	}
	if listRequest.Id != nil {
		t.Fatalf("list request id = %v, want nil for first bind-or-create", listRequest.Id)
	}
	if getRequest.ApiPlatformInstanceId == nil || *getRequest.ApiPlatformInstanceId != "ocid1.apiplatforminstance.oc1..existing" {
		t.Fatalf("get request apiPlatformInstanceId = %v, want existing OCID", getRequest.ApiPlatformInstanceId)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.apiplatforminstance.oc1..existing" {
		t.Fatalf("status.ocid = %q, want existing OCID", got)
	}
	requireTrailingApiPlatformCondition(t, resource, shared.Active)
}

func TestApiPlatformInstanceCreateOrUpdateReportsFailedLifecycle(t *testing.T) {
	t.Parallel()

	client := testApiPlatformInstanceClient(&fakeApiPlatformInstanceOCIClient{
		createFn: func(_ context.Context, _ apiplatformsdk.CreateApiPlatformInstanceRequest) (apiplatformsdk.CreateApiPlatformInstanceResponse, error) {
			return apiplatformsdk.CreateApiPlatformInstanceResponse{
				ApiPlatformInstance: makeSDKApiPlatformInstance(
					"ocid1.apiplatforminstance.oc1..failed",
					"ocid1.compartment.oc1..example",
					"apip-alpha",
					"desired description",
					apiplatformsdk.ApiPlatformInstanceLifecycleStateFailed,
				),
			}, nil
		},
		getFn: func(_ context.Context, _ apiplatformsdk.GetApiPlatformInstanceRequest) (apiplatformsdk.GetApiPlatformInstanceResponse, error) {
			return apiplatformsdk.GetApiPlatformInstanceResponse{
				ApiPlatformInstance: makeSDKApiPlatformInstance(
					"ocid1.apiplatforminstance.oc1..failed",
					"ocid1.compartment.oc1..example",
					"apip-alpha",
					"desired description",
					apiplatformsdk.ApiPlatformInstanceLifecycleStateFailed,
				),
			}, nil
		},
	})

	resource := makeApiPlatformInstanceResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report unsuccessful for FAILED lifecycle")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for FAILED lifecycle")
	}
	if got := resource.Status.LifecycleState; got != "FAILED" {
		t.Fatalf("status.lifecycleState = %q, want %q", got, "FAILED")
	}
	requireTrailingApiPlatformCondition(t, resource, shared.Failed)
}

func TestApiPlatformInstanceDeleteKeepsFinalizerWhileDeleting(t *testing.T) {
	t.Parallel()

	var deleteRequest apiplatformsdk.DeleteApiPlatformInstanceRequest
	getCalls := 0

	client := testApiPlatformInstanceClient(&fakeApiPlatformInstanceOCIClient{
		deleteFn: func(_ context.Context, req apiplatformsdk.DeleteApiPlatformInstanceRequest) (apiplatformsdk.DeleteApiPlatformInstanceResponse, error) {
			deleteRequest = req
			return apiplatformsdk.DeleteApiPlatformInstanceResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
		getFn: func(_ context.Context, req apiplatformsdk.GetApiPlatformInstanceRequest) (apiplatformsdk.GetApiPlatformInstanceResponse, error) {
			getCalls++
			state := apiplatformsdk.ApiPlatformInstanceLifecycleStateDeleting
			if getCalls == 1 {
				state = apiplatformsdk.ApiPlatformInstanceLifecycleStateActive
			}
			if req.ApiPlatformInstanceId == nil || *req.ApiPlatformInstanceId != "ocid1.apiplatforminstance.oc1..created" {
				t.Fatalf("get request apiPlatformInstanceId = %v, want tracked OCID", req.ApiPlatformInstanceId)
			}
			return apiplatformsdk.GetApiPlatformInstanceResponse{
				ApiPlatformInstance: makeSDKApiPlatformInstance(
					"ocid1.apiplatforminstance.oc1..created",
					"ocid1.compartment.oc1..example",
					"apip-alpha",
					"desired description",
					state,
				),
			}, nil
		},
	})

	resource := makeApiPlatformInstanceResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.apiplatforminstance.oc1..created")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while OCI delete is pending")
	}
	if deleteRequest.ApiPlatformInstanceId == nil || *deleteRequest.ApiPlatformInstanceId != "ocid1.apiplatforminstance.oc1..created" {
		t.Fatalf("delete request apiPlatformInstanceId = %v, want tracked OCID", deleteRequest.ApiPlatformInstanceId)
	}
	if getCalls != 2 {
		t.Fatalf("get calls = %d, want 2 (pre-delete read plus confirm-delete reread)", getCalls)
	}
	if got := resource.Status.LifecycleState; got != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want %q", got, "DELETING")
	}
	requireTrailingApiPlatformCondition(t, resource, shared.Terminating)
}

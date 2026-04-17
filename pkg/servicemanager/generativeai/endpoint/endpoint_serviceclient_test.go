/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package endpoint

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaisdk "github.com/oracle/oci-go-sdk/v65/generativeai"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeEndpointOCIClient struct {
	createEndpointFn func(context.Context, generativeaisdk.CreateEndpointRequest) (generativeaisdk.CreateEndpointResponse, error)
	getEndpointFn    func(context.Context, generativeaisdk.GetEndpointRequest) (generativeaisdk.GetEndpointResponse, error)
	listEndpointsFn  func(context.Context, generativeaisdk.ListEndpointsRequest) (generativeaisdk.ListEndpointsResponse, error)
	updateEndpointFn func(context.Context, generativeaisdk.UpdateEndpointRequest) (generativeaisdk.UpdateEndpointResponse, error)
	deleteEndpointFn func(context.Context, generativeaisdk.DeleteEndpointRequest) (generativeaisdk.DeleteEndpointResponse, error)
}

func (f *fakeEndpointOCIClient) CreateEndpoint(
	ctx context.Context,
	req generativeaisdk.CreateEndpointRequest,
) (generativeaisdk.CreateEndpointResponse, error) {
	if f.createEndpointFn != nil {
		return f.createEndpointFn(ctx, req)
	}
	return generativeaisdk.CreateEndpointResponse{}, nil
}

func (f *fakeEndpointOCIClient) GetEndpoint(
	ctx context.Context,
	req generativeaisdk.GetEndpointRequest,
) (generativeaisdk.GetEndpointResponse, error) {
	if f.getEndpointFn != nil {
		return f.getEndpointFn(ctx, req)
	}
	return generativeaisdk.GetEndpointResponse{}, nil
}

func (f *fakeEndpointOCIClient) ListEndpoints(
	ctx context.Context,
	req generativeaisdk.ListEndpointsRequest,
) (generativeaisdk.ListEndpointsResponse, error) {
	if f.listEndpointsFn != nil {
		return f.listEndpointsFn(ctx, req)
	}
	return generativeaisdk.ListEndpointsResponse{}, nil
}

func (f *fakeEndpointOCIClient) UpdateEndpoint(
	ctx context.Context,
	req generativeaisdk.UpdateEndpointRequest,
) (generativeaisdk.UpdateEndpointResponse, error) {
	if f.updateEndpointFn != nil {
		return f.updateEndpointFn(ctx, req)
	}
	return generativeaisdk.UpdateEndpointResponse{}, nil
}

func (f *fakeEndpointOCIClient) DeleteEndpoint(
	ctx context.Context,
	req generativeaisdk.DeleteEndpointRequest,
) (generativeaisdk.DeleteEndpointResponse, error) {
	if f.deleteEndpointFn != nil {
		return f.deleteEndpointFn(ctx, req)
	}
	return generativeaisdk.DeleteEndpointResponse{}, nil
}

func testEndpointClient(fake *fakeEndpointOCIClient) defaultEndpointServiceClient {
	return defaultEndpointServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*generativeaiv1beta1.Endpoint](generatedruntime.Config[*generativeaiv1beta1.Endpoint]{
			Kind:      "Endpoint",
			SDKName:   "Endpoint",
			Log:       loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
			Semantics: newEndpointRuntimeSemantics(),
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &generativeaisdk.CreateEndpointRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.CreateEndpoint(ctx, *request.(*generativeaisdk.CreateEndpointRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CreateEndpointDetails", RequestName: "CreateEndpointDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &generativeaisdk.GetEndpointRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.GetEndpoint(ctx, *request.(*generativeaisdk.GetEndpointRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "EndpointId", RequestName: "endpointId", Contribution: "path", PreferResourceID: true},
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &generativeaisdk.ListEndpointsRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.ListEndpoints(ctx, *request.(*generativeaisdk.ListEndpointsRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
					{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", PreferResourceID: false},
					{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", PreferResourceID: false},
					{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: false},
					{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
					{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &generativeaisdk.UpdateEndpointRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.UpdateEndpoint(ctx, *request.(*generativeaisdk.UpdateEndpointRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "EndpointId", RequestName: "endpointId", Contribution: "path", PreferResourceID: true},
					{FieldName: "UpdateEndpointDetails", RequestName: "UpdateEndpointDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &generativeaisdk.DeleteEndpointRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return fake.DeleteEndpoint(ctx, *request.(*generativeaisdk.DeleteEndpointRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "EndpointId", RequestName: "endpointId", Contribution: "path", PreferResourceID: true},
				},
			},
		}),
	}
}

func makeEndpointResource() *generativeaiv1beta1.Endpoint {
	return &generativeaiv1beta1.Endpoint{
		Spec: generativeaiv1beta1.EndpointSpec{
			CompartmentId:        "ocid1.compartment.oc1..endpointexample",
			ModelId:              "ocid1.generativeaimodel.oc1..model",
			DedicatedAiClusterId: "ocid1.generativeaidedicatedaicluster.oc1..cluster",
			DisplayName:          "osok-endpoint-sample",
			Description:          "OSOK Endpoint sample",
			ContentModerationConfig: generativeaiv1beta1.EndpointContentModerationConfig{
				IsEnabled: false,
			},
			FreeformTags: map[string]string{"managed-by": "osok"},
		},
	}
}

func makeSDKEndpoint(
	id string,
	spec generativeaiv1beta1.EndpointSpec,
	state generativeaisdk.EndpointLifecycleStateEnum,
) generativeaisdk.Endpoint {
	endpoint := generativeaisdk.Endpoint{
		Id:                   common.String(id),
		ModelId:              common.String(spec.ModelId),
		CompartmentId:        common.String(spec.CompartmentId),
		DedicatedAiClusterId: common.String(spec.DedicatedAiClusterId),
		LifecycleState:       state,
		ContentModerationConfig: &generativeaisdk.ContentModerationConfig{
			IsEnabled: boolPtr(spec.ContentModerationConfig.IsEnabled),
		},
		FreeformTags: map[string]string{},
		DefinedTags:  map[string]map[string]interface{}{},
		SystemTags:   map[string]map[string]interface{}{},
	}
	if spec.DisplayName != "" {
		endpoint.DisplayName = common.String(spec.DisplayName)
	}
	if spec.Description != "" {
		endpoint.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		endpoint.FreeformTags = spec.FreeformTags
	}
	return endpoint
}

func makeSDKEndpointSummary(
	id string,
	spec generativeaiv1beta1.EndpointSpec,
	state generativeaisdk.EndpointLifecycleStateEnum,
) generativeaisdk.EndpointSummary {
	summary := generativeaisdk.EndpointSummary{
		Id:                   common.String(id),
		ModelId:              common.String(spec.ModelId),
		CompartmentId:        common.String(spec.CompartmentId),
		DedicatedAiClusterId: common.String(spec.DedicatedAiClusterId),
		LifecycleState:       state,
		ContentModerationConfig: &generativeaisdk.ContentModerationConfig{
			IsEnabled: boolPtr(spec.ContentModerationConfig.IsEnabled),
		},
	}
	if spec.DisplayName != "" {
		summary.DisplayName = common.String(spec.DisplayName)
	}
	if spec.Description != "" {
		summary.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		summary.FreeformTags = spec.FreeformTags
	}
	return summary
}

func boolPtr(value bool) *bool {
	return &value
}

func TestEndpointServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	var createRequest generativeaisdk.CreateEndpointRequest
	getCalls := 0
	client := testEndpointClient(&fakeEndpointOCIClient{
		listEndpointsFn: func(_ context.Context, req generativeaisdk.ListEndpointsRequest) (generativeaisdk.ListEndpointsResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..endpointexample" {
				t.Fatalf("list compartmentId = %v, want endpoint compartment", req.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != "osok-endpoint-sample" {
				t.Fatalf("list displayName = %v, want endpoint displayName", req.DisplayName)
			}
			return generativeaisdk.ListEndpointsResponse{}, nil
		},
		createEndpointFn: func(_ context.Context, req generativeaisdk.CreateEndpointRequest) (generativeaisdk.CreateEndpointResponse, error) {
			createRequest = req
			return generativeaisdk.CreateEndpointResponse{
				Endpoint:     makeSDKEndpoint("ocid1.endpoint.oc1..created", makeEndpointResource().Spec, generativeaisdk.EndpointLifecycleStateCreating),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getEndpointFn: func(_ context.Context, req generativeaisdk.GetEndpointRequest) (generativeaisdk.GetEndpointResponse, error) {
			getCalls++
			if req.EndpointId == nil || *req.EndpointId != "ocid1.endpoint.oc1..created" {
				t.Fatalf("get endpointId = %v, want created endpoint OCID", req.EndpointId)
			}
			return generativeaisdk.GetEndpointResponse{
				Endpoint: makeSDKEndpoint("ocid1.endpoint.oc1..created", makeEndpointResource().Spec, generativeaisdk.EndpointLifecycleStateActive),
			}, nil
		},
	})

	resource := makeEndpointResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once follow-up GetEndpoint reports ACTIVE")
	}
	if createRequest.CreateEndpointDetails.CompartmentId == nil || *createRequest.CreateEndpointDetails.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", createRequest.CreateEndpointDetails.CompartmentId, resource.Spec.CompartmentId)
	}
	if createRequest.CreateEndpointDetails.ModelId == nil || *createRequest.CreateEndpointDetails.ModelId != resource.Spec.ModelId {
		t.Fatalf("create modelId = %v, want %q", createRequest.CreateEndpointDetails.ModelId, resource.Spec.ModelId)
	}
	if createRequest.CreateEndpointDetails.DedicatedAiClusterId == nil || *createRequest.CreateEndpointDetails.DedicatedAiClusterId != resource.Spec.DedicatedAiClusterId {
		t.Fatalf("create dedicatedAiClusterId = %v, want %q", createRequest.CreateEndpointDetails.DedicatedAiClusterId, resource.Spec.DedicatedAiClusterId)
	}
	if getCalls != 1 {
		t.Fatalf("GetEndpoint() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.endpoint.oc1..created" {
		t.Fatalf("status.ocid = %q, want created endpoint OCID", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-create-1")
	}
	if got := resource.Status.Id; got != "ocid1.endpoint.oc1..created" {
		t.Fatalf("status.id = %q, want created endpoint OCID", got)
	}
	if got := resource.Status.ModelId; got != resource.Spec.ModelId {
		t.Fatalf("status.modelId = %q, want %q", got, resource.Spec.ModelId)
	}
	if got := resource.Status.DedicatedAiClusterId; got != resource.Spec.DedicatedAiClusterId {
		t.Fatalf("status.dedicatedAiClusterId = %q, want %q", got, resource.Spec.DedicatedAiClusterId)
	}
	if got := resource.Status.LifecycleState; got != string(generativeaisdk.EndpointLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestEndpointServiceClientBindsExistingEndpointWithoutCreateWhenDisplayNameEmpty(t *testing.T) {
	t.Parallel()

	createCalled := false
	getCalls := 0
	resource := makeEndpointResource()
	resource.Spec.DisplayName = ""

	mismatchSpec := resource.Spec
	mismatchSpec.ModelId = "ocid1.generativeaimodel.oc1..different-model"
	mismatchSpec.DisplayName = "different-endpoint"

	matchedSpec := resource.Spec
	matchedSpec.DisplayName = "server-generated-name"

	client := testEndpointClient(&fakeEndpointOCIClient{
		listEndpointsFn: func(_ context.Context, req generativeaisdk.ListEndpointsRequest) (generativeaisdk.ListEndpointsResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("list compartmentId = %v, want endpoint compartment", req.CompartmentId)
			}
			if req.DisplayName != nil {
				t.Fatalf("list displayName = %v, want nil when spec.displayName is empty", req.DisplayName)
			}
			return generativeaisdk.ListEndpointsResponse{
				EndpointCollection: generativeaisdk.EndpointCollection{
					Items: []generativeaisdk.EndpointSummary{
						makeSDKEndpointSummary("ocid1.endpoint.oc1..mismatch", mismatchSpec, generativeaisdk.EndpointLifecycleStateActive),
						makeSDKEndpointSummary("ocid1.endpoint.oc1..existing", matchedSpec, generativeaisdk.EndpointLifecycleStateActive),
					},
				},
			}, nil
		},
		getEndpointFn: func(_ context.Context, req generativeaisdk.GetEndpointRequest) (generativeaisdk.GetEndpointResponse, error) {
			getCalls++
			if req.EndpointId == nil || *req.EndpointId != "ocid1.endpoint.oc1..existing" {
				t.Fatalf("get endpointId = %v, want existing endpoint OCID", req.EndpointId)
			}
			return generativeaisdk.GetEndpointResponse{
				Endpoint: makeSDKEndpoint("ocid1.endpoint.oc1..existing", matchedSpec, generativeaisdk.EndpointLifecycleStateActive),
			}, nil
		},
		createEndpointFn: func(_ context.Context, _ generativeaisdk.CreateEndpointRequest) (generativeaisdk.CreateEndpointResponse, error) {
			createCalled = true
			return generativeaisdk.CreateEndpointResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when list lookup reuses an existing endpoint")
	}
	if createCalled {
		t.Fatal("CreateEndpoint() should not be called when ListEndpoints finds a matching endpoint")
	}
	if getCalls != 1 {
		t.Fatalf("GetEndpoint() calls = %d, want 1 read of the bound endpoint", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.endpoint.oc1..existing" {
		t.Fatalf("status.ocid = %q, want existing endpoint OCID", got)
	}
}

func TestEndpointServiceClientUpdatesContentModerationConfig(t *testing.T) {
	t.Parallel()

	var updateRequest generativeaisdk.UpdateEndpointRequest
	getCalls := 0
	updateCalls := 0
	client := testEndpointClient(&fakeEndpointOCIClient{
		getEndpointFn: func(_ context.Context, req generativeaisdk.GetEndpointRequest) (generativeaisdk.GetEndpointResponse, error) {
			getCalls++
			if req.EndpointId == nil || *req.EndpointId != "ocid1.endpoint.oc1..existing" {
				t.Fatalf("get endpointId = %v, want existing endpoint OCID", req.EndpointId)
			}

			spec := makeEndpointResource().Spec
			if getCalls > 1 {
				spec.ContentModerationConfig.IsEnabled = true
			}
			return generativeaisdk.GetEndpointResponse{
				Endpoint: makeSDKEndpoint("ocid1.endpoint.oc1..existing", spec, generativeaisdk.EndpointLifecycleStateActive),
			}, nil
		},
		updateEndpointFn: func(_ context.Context, req generativeaisdk.UpdateEndpointRequest) (generativeaisdk.UpdateEndpointResponse, error) {
			updateCalls++
			updateRequest = req

			spec := makeEndpointResource().Spec
			spec.ContentModerationConfig.IsEnabled = true
			return generativeaisdk.UpdateEndpointResponse{
				Endpoint:     makeSDKEndpoint("ocid1.endpoint.oc1..existing", spec, generativeaisdk.EndpointLifecycleStateActive),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	resource := makeEndpointResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.endpoint.oc1..existing")
	resource.Spec.ContentModerationConfig.IsEnabled = true

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after updating mutable contentModerationConfig")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once update follow-up GetEndpoint reports ACTIVE")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateEndpoint() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetEndpoint() calls = %d, want 2 (observe + follow-up)", getCalls)
	}
	if updateRequest.EndpointId == nil || *updateRequest.EndpointId != "ocid1.endpoint.oc1..existing" {
		t.Fatalf("update endpointId = %v, want existing endpoint OCID", updateRequest.EndpointId)
	}
	if updateRequest.UpdateEndpointDetails.ContentModerationConfig == nil ||
		updateRequest.UpdateEndpointDetails.ContentModerationConfig.IsEnabled == nil ||
		!*updateRequest.UpdateEndpointDetails.ContentModerationConfig.IsEnabled {
		t.Fatalf("update contentModerationConfig = %#v, want isEnabled=true", updateRequest.UpdateEndpointDetails.ContentModerationConfig)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-update-1")
	}
	if !resource.Status.ContentModerationConfig.IsEnabled {
		t.Fatal("status.contentModerationConfig.isEnabled = false, want true")
	}
}

func TestEndpointServiceClientRejectsReplacementOnlyModelDrift(t *testing.T) {
	t.Parallel()

	updateCalled := false
	client := testEndpointClient(&fakeEndpointOCIClient{
		getEndpointFn: func(_ context.Context, _ generativeaisdk.GetEndpointRequest) (generativeaisdk.GetEndpointResponse, error) {
			return generativeaisdk.GetEndpointResponse{
				Endpoint: makeSDKEndpoint("ocid1.endpoint.oc1..existing", makeEndpointResource().Spec, generativeaisdk.EndpointLifecycleStateActive),
			}, nil
		},
		updateEndpointFn: func(_ context.Context, _ generativeaisdk.UpdateEndpointRequest) (generativeaisdk.UpdateEndpointResponse, error) {
			updateCalled = true
			return generativeaisdk.UpdateEndpointResponse{}, nil
		},
	})

	resource := makeEndpointResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.endpoint.oc1..existing")
	resource.Spec.ModelId = "ocid1.generativeaimodel.oc1..replacement"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when modelId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateEndpoint() should not be called when replacement-only drift is detected")
	}
}

func TestEndpointServiceClientDeleteConfirmsDeletion(t *testing.T) {
	t.Parallel()

	var deleteRequest generativeaisdk.DeleteEndpointRequest
	getCalls := 0
	client := testEndpointClient(&fakeEndpointOCIClient{
		getEndpointFn: func(_ context.Context, req generativeaisdk.GetEndpointRequest) (generativeaisdk.GetEndpointResponse, error) {
			getCalls++
			if req.EndpointId == nil || *req.EndpointId != "ocid1.endpoint.oc1..existing" {
				t.Fatalf("get endpointId = %v, want existing endpoint OCID", req.EndpointId)
			}

			state := generativeaisdk.EndpointLifecycleStateActive
			if getCalls > 1 {
				state = generativeaisdk.EndpointLifecycleStateDeleted
			}
			return generativeaisdk.GetEndpointResponse{
				Endpoint: makeSDKEndpoint("ocid1.endpoint.oc1..existing", makeEndpointResource().Spec, state),
			}, nil
		},
		deleteEndpointFn: func(_ context.Context, req generativeaisdk.DeleteEndpointRequest) (generativeaisdk.DeleteEndpointResponse, error) {
			deleteRequest = req
			return generativeaisdk.DeleteEndpointResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	resource := makeEndpointResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.endpoint.oc1..existing")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success once follow-up GetEndpoint confirms DELETED")
	}
	if getCalls != 2 {
		t.Fatalf("GetEndpoint() calls = %d, want 2 (preflight + confirmation)", getCalls)
	}
	if deleteRequest.EndpointId == nil || *deleteRequest.EndpointId != "ocid1.endpoint.oc1..existing" {
		t.Fatalf("delete endpointId = %v, want existing endpoint OCID", deleteRequest.EndpointId)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.LifecycleState; got != string(generativeaisdk.EndpointLifecycleStateDeleted) {
		t.Fatalf("status.lifecycleState = %q, want DELETED", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-delete-1")
	}
}

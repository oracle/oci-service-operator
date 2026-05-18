/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivedatamodel

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testSensitiveDataModelID            = "ocid1.datasafesensitivedatamodel.oc1..model"
	testOtherSensitiveDataModelID       = "ocid1.datasafesensitivedatamodel.oc1..other"
	testSensitiveDataModelCompartmentID = "ocid1.compartment.oc1..datasafe"
	testSensitiveDataModelTargetID      = "ocid1.datasafetargetdatabase.oc1..target"
	testSensitiveDataModelName          = "customer-model"
	testSensitiveDataModelUpdatedName   = "customer-model-renamed"
	testSensitiveDataModelUID           = "sensitive-data-model-uid"
)

type fakeSensitiveDataModelOCIClient struct {
	createFn func(context.Context, datasafesdk.CreateSensitiveDataModelRequest) (datasafesdk.CreateSensitiveDataModelResponse, error)
	getFn    func(context.Context, datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error)
	listFn   func(context.Context, datasafesdk.ListSensitiveDataModelsRequest) (datasafesdk.ListSensitiveDataModelsResponse, error)
	updateFn func(context.Context, datasafesdk.UpdateSensitiveDataModelRequest) (datasafesdk.UpdateSensitiveDataModelResponse, error)
	deleteFn func(context.Context, datasafesdk.DeleteSensitiveDataModelRequest) (datasafesdk.DeleteSensitiveDataModelResponse, error)

	createRequests []datasafesdk.CreateSensitiveDataModelRequest
	getRequests    []datasafesdk.GetSensitiveDataModelRequest
	listRequests   []datasafesdk.ListSensitiveDataModelsRequest
	updateRequests []datasafesdk.UpdateSensitiveDataModelRequest
	deleteRequests []datasafesdk.DeleteSensitiveDataModelRequest
}

func (f *fakeSensitiveDataModelOCIClient) CreateSensitiveDataModel(
	ctx context.Context,
	request datasafesdk.CreateSensitiveDataModelRequest,
) (datasafesdk.CreateSensitiveDataModelResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return datasafesdk.CreateSensitiveDataModelResponse{}, nil
}

func (f *fakeSensitiveDataModelOCIClient) GetSensitiveDataModel(
	ctx context.Context,
	request datasafesdk.GetSensitiveDataModelRequest,
) (datasafesdk.GetSensitiveDataModelResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return datasafesdk.GetSensitiveDataModelResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
}

func (f *fakeSensitiveDataModelOCIClient) ListSensitiveDataModels(
	ctx context.Context,
	request datasafesdk.ListSensitiveDataModelsRequest,
) (datasafesdk.ListSensitiveDataModelsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return datasafesdk.ListSensitiveDataModelsResponse{}, nil
}

func (f *fakeSensitiveDataModelOCIClient) UpdateSensitiveDataModel(
	ctx context.Context,
	request datasafesdk.UpdateSensitiveDataModelRequest,
) (datasafesdk.UpdateSensitiveDataModelResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return datasafesdk.UpdateSensitiveDataModelResponse{}, nil
}

func (f *fakeSensitiveDataModelOCIClient) DeleteSensitiveDataModel(
	ctx context.Context,
	request datasafesdk.DeleteSensitiveDataModelRequest,
) (datasafesdk.DeleteSensitiveDataModelResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return datasafesdk.DeleteSensitiveDataModelResponse{}, nil
}

func TestSensitiveDataModelRuntimeHooksConfigured(t *testing.T) {
	hooks := newSensitiveDataModelDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applySensitiveDataModelRuntimeHooks(&hooks)

	requireSensitiveDataModelRuntimeSemantics(t, hooks)
	requireSensitiveDataModelRuntimeHookFunctions(t, hooks)
	requireSensitiveDataModelCreateBody(t, hooks)
}

func requireSensitiveDataModelRuntimeSemantics(t *testing.T, hooks SensitiveDataModelRuntimeHooks) {
	t.Helper()
	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed runtime semantics")
	}
	if got := hooks.Semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	if got := hooks.Semantics.Async.Strategy; got != "lifecycle" {
		t.Fatalf("Async.Strategy = %q, want lifecycle", got)
	}
	assertContainsAll(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "displayName", "targetId", "description", "freeformTags", "definedTags")
	assertContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "compartmentId", "isIncludeAllSchemas", "isIncludeAllSensitiveTypes")
	assertContainsAll(t, "List.MatchFields", hooks.Semantics.List.MatchFields, "compartmentId", "targetId", "displayName", "id")
}

func requireSensitiveDataModelRuntimeHookFunctions(t *testing.T, hooks SensitiveDataModelRuntimeHooks) {
	t.Helper()
	if hooks.BuildCreateBody == nil || hooks.BuildUpdateBody == nil {
		t.Fatal("BuildCreateBody/BuildUpdateBody = nil, want resource-specific body builders")
	}
	if hooks.DeleteHooks.HandleError == nil || hooks.DeleteHooks.ApplyOutcome == nil {
		t.Fatal("delete hooks are not configured")
	}
	if len(hooks.WrapGeneratedClient) == 0 {
		t.Fatal("WrapGeneratedClient is empty, want conservative delete wrapper")
	}
}

func requireSensitiveDataModelCreateBody(t *testing.T, hooks SensitiveDataModelRuntimeHooks) {
	t.Helper()
	body, err := hooks.BuildCreateBody(context.Background(), makeSensitiveDataModelResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(datasafesdk.CreateSensitiveDataModelDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateSensitiveDataModelDetails", body)
	}
	requireStringPtr(t, "CreateSensitiveDataModelDetails.CompartmentId", details.CompartmentId, testSensitiveDataModelCompartmentID)
	requireStringPtr(t, "CreateSensitiveDataModelDetails.TargetId", details.TargetId, testSensitiveDataModelTargetID)
	requireBoolPtr(t, "CreateSensitiveDataModelDetails.IsSampleDataCollectionEnabled", details.IsSampleDataCollectionEnabled, false)
	requireBoolPtr(t, "CreateSensitiveDataModelDetails.IsAppDefinedRelationDiscoveryEnabled", details.IsAppDefinedRelationDiscoveryEnabled, true)
	requireTablesForDiscovery(t, details.TablesForDiscovery)
}

func TestSensitiveDataModelCreateRecordsIdentityRequestIDAndLifecycleWorkRequest(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	created := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateCreating)
	client := &fakeSensitiveDataModelOCIClient{
		createFn: func(_ context.Context, request datasafesdk.CreateSensitiveDataModelRequest) (datasafesdk.CreateSensitiveDataModelResponse, error) {
			requireSensitiveDataModelCreateRequest(t, request, resource)
			return datasafesdk.CreateSensitiveDataModelResponse{
				SensitiveDataModel: created,
				OpcWorkRequestId:   common.String("wr-create"),
				OpcRequestId:       common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			return datasafesdk.GetSensitiveDataModelResponse{SensitiveDataModel: created}, nil
		},
	}

	response, err := newTestSensitiveDataModelClient(client).CreateOrUpdate(
		context.Background(),
		resource,
		sensitiveDataModelRequest(resource),
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want requeue while CREATING")
	}
	requireCallCount(t, "ListSensitiveDataModels()", len(client.listRequests), 1)
	requireCallCount(t, "CreateSensitiveDataModel()", len(client.createRequests), 1)
	requireCallCount(t, "GetSensitiveDataModel()", len(client.getRequests), 1)
	requireRecordedSensitiveDataModelID(t, resource, testSensitiveDataModelID)
	requireOpcRequestID(t, resource, "opc-create")
	requireAsync(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create")
}

func TestSensitiveDataModelCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	existing := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	var pages []string
	client := &fakeSensitiveDataModelOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListSensitiveDataModelsRequest) (datasafesdk.ListSensitiveDataModelsResponse, error) {
			pages = append(pages, stringValue(request.Page))
			requireStringPtr(t, "ListSensitiveDataModelsRequest.CompartmentId", request.CompartmentId, testSensitiveDataModelCompartmentID)
			requireStringPtr(t, "ListSensitiveDataModelsRequest.DisplayName", request.DisplayName, testSensitiveDataModelName)
			requireStringPtr(t, "ListSensitiveDataModelsRequest.TargetId", request.TargetId, testSensitiveDataModelTargetID)
			if request.Page == nil {
				return datasafesdk.ListSensitiveDataModelsResponse{
					SensitiveDataModelCollection: datasafesdk.SensitiveDataModelCollection{
						Items: []datasafesdk.SensitiveDataModelSummary{
							sdkSensitiveDataModelSummary(resource, testOtherSensitiveDataModelID, "other-model", datasafesdk.DiscoveryLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return datasafesdk.ListSensitiveDataModelsResponse{
				SensitiveDataModelCollection: datasafesdk.SensitiveDataModelCollection{
					Items: []datasafesdk.SensitiveDataModelSummary{
						sdkSensitiveDataModelSummary(resource, testSensitiveDataModelID, testSensitiveDataModelName, datasafesdk.DiscoveryLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			return datasafesdk.GetSensitiveDataModelResponse{SensitiveDataModel: existing}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateSensitiveDataModelRequest) (datasafesdk.CreateSensitiveDataModelResponse, error) {
			t.Fatal("CreateSensitiveDataModel() called despite existing paginated list match")
			return datasafesdk.CreateSensitiveDataModelResponse{}, nil
		},
	}

	response, err := newTestSensitiveDataModelClient(client).CreateOrUpdate(
		context.Background(),
		resource,
		sensitiveDataModelRequest(resource),
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListSensitiveDataModels() pages = %q, want \",page-2\"", got)
	}
	requireRecordedSensitiveDataModelID(t, resource, testSensitiveDataModelID)
	requireCallCount(t, "CreateSensitiveDataModel()", len(client.createRequests), 0)
	requireLastCondition(t, resource, shared.Active)
}

func TestSensitiveDataModelCreateSkipsPreCreateBindWhenDisplayNameEmpty(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Spec.DisplayName = ""
	created := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	created.DisplayName = nil

	client := &fakeSensitiveDataModelOCIClient{
		listFn: func(context.Context, datasafesdk.ListSensitiveDataModelsRequest) (datasafesdk.ListSensitiveDataModelsResponse, error) {
			t.Fatal("ListSensitiveDataModels() called; empty displayName must not bind by compartmentId and targetId alone")
			return datasafesdk.ListSensitiveDataModelsResponse{}, nil
		},
		createFn: func(_ context.Context, request datasafesdk.CreateSensitiveDataModelRequest) (datasafesdk.CreateSensitiveDataModelResponse, error) {
			requireStringPtr(t, "CreateSensitiveDataModelRequest.OpcRetryToken", request.OpcRetryToken, testSensitiveDataModelUID)
			details := request.CreateSensitiveDataModelDetails
			requireStringPtr(t, "CreateSensitiveDataModelDetails.CompartmentId", details.CompartmentId, testSensitiveDataModelCompartmentID)
			requireStringPtr(t, "CreateSensitiveDataModelDetails.TargetId", details.TargetId, testSensitiveDataModelTargetID)
			if details.DisplayName != nil {
				t.Fatalf("CreateSensitiveDataModelDetails.DisplayName = %q, want nil for omitted displayName", *details.DisplayName)
			}
			return datasafesdk.CreateSensitiveDataModelResponse{
				SensitiveDataModel: created,
				OpcRequestId:       common.String("opc-create-empty-name"),
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			return datasafesdk.GetSensitiveDataModelResponse{SensitiveDataModel: created}, nil
		},
	}

	response, err := newTestSensitiveDataModelClient(client).CreateOrUpdate(
		context.Background(),
		resource,
		sensitiveDataModelRequest(resource),
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	requireCallCount(t, "ListSensitiveDataModels()", len(client.listRequests), 0)
	requireCallCount(t, "CreateSensitiveDataModel()", len(client.createRequests), 1)
	requireRecordedSensitiveDataModelID(t, resource, testSensitiveDataModelID)
	requireOpcRequestID(t, resource, "opc-create-empty-name")
	requireLastCondition(t, resource, shared.Active)
}

func TestSensitiveDataModelCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	current := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			return datasafesdk.GetSensitiveDataModelResponse{SensitiveDataModel: current}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateSensitiveDataModelRequest) (datasafesdk.CreateSensitiveDataModelResponse, error) {
			t.Fatal("CreateSensitiveDataModel() called during no-op reconcile")
			return datasafesdk.CreateSensitiveDataModelResponse{}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateSensitiveDataModelRequest) (datasafesdk.UpdateSensitiveDataModelResponse, error) {
			t.Fatal("UpdateSensitiveDataModel() called during no-op reconcile")
			return datasafesdk.UpdateSensitiveDataModelResponse{}, nil
		},
	}

	response, err := newTestSensitiveDataModelClient(client).CreateOrUpdate(
		context.Background(),
		resource,
		sensitiveDataModelRequest(resource),
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false for active no-op")
	}
	requireCallCount(t, "GetSensitiveDataModel()", len(client.getRequests), 1)
	requireCallCount(t, "CreateSensitiveDataModel()", len(client.createRequests), 0)
	requireCallCount(t, "UpdateSensitiveDataModel()", len(client.updateRequests), 0)
	requireLastCondition(t, resource, shared.Active)
}

func TestSensitiveDataModelNoopsWhenAppSuiteNameOmittedAndObservedDefaultedGeneric(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Spec.AppSuiteName = ""
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	current := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	current.AppSuiteName = common.String("GENERIC")
	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			return datasafesdk.GetSensitiveDataModelResponse{SensitiveDataModel: current}, nil
		},
		updateFn: func(_ context.Context, request datasafesdk.UpdateSensitiveDataModelRequest) (datasafesdk.UpdateSensitiveDataModelResponse, error) {
			t.Fatalf("UpdateSensitiveDataModel() called with omitted appSuiteName: %#v", request.AppSuiteName)
			return datasafesdk.UpdateSensitiveDataModelResponse{}, nil
		},
	}

	response, err := newTestSensitiveDataModelClient(client).CreateOrUpdate(
		context.Background(),
		resource,
		sensitiveDataModelRequest(resource),
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	requireCallCount(t, "GetSensitiveDataModel()", len(client.getRequests), 1)
	requireCallCount(t, "UpdateSensitiveDataModel()", len(client.updateRequests), 0)
	if got := resource.Status.AppSuiteName; got != "GENERIC" {
		t.Fatalf("status.appSuiteName = %q, want GENERIC", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestSensitiveDataModelMutableUpdateShapesRequestAndRefreshesStatus(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	resource.Spec.DisplayName = testSensitiveDataModelUpdatedName
	resource.Spec.Description = "updated model description"
	resource.Spec.IsSampleDataCollectionEnabled = true
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	current := sdkSensitiveDataModel(makeSensitiveDataModelResource(), testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	updated := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	getResponses := []datasafesdk.GetSensitiveDataModelResponse{
		{SensitiveDataModel: current},
		{SensitiveDataModel: updated},
	}
	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			if len(getResponses) == 0 {
				t.Fatal("GetSensitiveDataModel() called more times than expected")
			}
			response := getResponses[0]
			getResponses = getResponses[1:]
			return response, nil
		},
		updateFn: func(_ context.Context, request datasafesdk.UpdateSensitiveDataModelRequest) (datasafesdk.UpdateSensitiveDataModelResponse, error) {
			requireSensitiveDataModelUpdateRequest(t, request, resource)
			return datasafesdk.UpdateSensitiveDataModelResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateSensitiveDataModelRequest) (datasafesdk.CreateSensitiveDataModelResponse, error) {
			t.Fatal("CreateSensitiveDataModel() called during tracked update")
			return datasafesdk.CreateSensitiveDataModelResponse{}, nil
		},
	}

	response, err := newTestSensitiveDataModelClient(client).CreateOrUpdate(
		context.Background(),
		resource,
		sensitiveDataModelRequest(resource),
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	requireCallCount(t, "GetSensitiveDataModel()", len(client.getRequests), 2)
	requireCallCount(t, "UpdateSensitiveDataModel()", len(client.updateRequests), 1)
	requireCallCount(t, "CreateSensitiveDataModel()", len(client.createRequests), 0)
	requireRecordedSensitiveDataModelID(t, resource, testSensitiveDataModelID)
	requireOpcRequestID(t, resource, "opc-update")
	requireLastCondition(t, resource, shared.Active)
	if got := resource.Status.DisplayName; got != testSensitiveDataModelUpdatedName {
		t.Fatalf("status.displayName = %q, want %q", got, testSensitiveDataModelUpdatedName)
	}
	if got := resource.Status.Description; got != resource.Spec.Description {
		t.Fatalf("status.description = %q, want %q", got, resource.Spec.Description)
	}
}

func TestSensitiveDataModelMutableUpdateKeepsRequeueWhenWorkRequestReadbackStaysActive(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	resource.Spec.DisplayName = testSensitiveDataModelUpdatedName

	current := sdkSensitiveDataModel(makeSensitiveDataModelResource(), testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	getResponses := []datasafesdk.GetSensitiveDataModelResponse{
		{SensitiveDataModel: current},
		{SensitiveDataModel: current},
	}
	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			if len(getResponses) == 0 {
				t.Fatal("GetSensitiveDataModel() called more times than expected")
			}
			response := getResponses[0]
			getResponses = getResponses[1:]
			return response, nil
		},
		updateFn: func(_ context.Context, request datasafesdk.UpdateSensitiveDataModelRequest) (datasafesdk.UpdateSensitiveDataModelResponse, error) {
			requireStringPtr(t, "UpdateSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			requireStringPtr(t, "UpdateSensitiveDataModelDetails.DisplayName", request.DisplayName, testSensitiveDataModelUpdatedName)
			return datasafesdk.UpdateSensitiveDataModelResponse{
				OpcWorkRequestId: common.String("wr-update-stale"),
				OpcRequestId:     common.String("opc-update-stale"),
			}, nil
		},
	}

	response, err := newTestSensitiveDataModelClient(client).CreateOrUpdate(
		context.Background(),
		resource,
		sensitiveDataModelRequest(resource),
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want requeue while accepted update is pending")
	}
	requireCallCount(t, "GetSensitiveDataModel()", len(client.getRequests), 2)
	requireCallCount(t, "UpdateSensitiveDataModel()", len(client.updateRequests), 1)
	requireOpcRequestID(t, resource, "opc-update-stale")
	requireWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-stale")
	requireLastCondition(t, resource, shared.Updating)
}

func TestSensitiveDataModelPendingUpdateWorkRequestSuppressesDuplicateMutationOnStaleActiveReadback(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	resource.Spec.DisplayName = testSensitiveDataModelUpdatedName
	seedSensitiveDataModelPendingWorkRequest(resource, shared.OSOKAsyncPhaseUpdate, "wr-existing-update")

	current := sdkSensitiveDataModel(makeSensitiveDataModelResource(), testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			return datasafesdk.GetSensitiveDataModelResponse{SensitiveDataModel: current}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateSensitiveDataModelRequest) (datasafesdk.UpdateSensitiveDataModelResponse, error) {
			t.Fatal("UpdateSensitiveDataModel() reissued while prior update work request was still pending")
			return datasafesdk.UpdateSensitiveDataModelResponse{}, nil
		},
	}

	response, err := newTestSensitiveDataModelClient(client).CreateOrUpdate(
		context.Background(),
		resource,
		sensitiveDataModelRequest(resource),
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want requeue while prior accepted update is pending")
	}
	requireCallCount(t, "GetSensitiveDataModel()", len(client.getRequests), 1)
	requireCallCount(t, "UpdateSensitiveDataModel()", len(client.updateRequests), 0)
	requireWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-existing-update")
	requireLastCondition(t, resource, shared.Updating)
}

func TestSensitiveDataModelCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	current := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	current.CompartmentId = common.String("ocid1.compartment.oc1..different")

	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(context.Context, datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			return datasafesdk.GetSensitiveDataModelResponse{SensitiveDataModel: current}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateSensitiveDataModelRequest) (datasafesdk.UpdateSensitiveDataModelResponse, error) {
			t.Fatal("UpdateSensitiveDataModel() called despite create-only compartment drift")
			return datasafesdk.UpdateSensitiveDataModelResponse{}, nil
		},
	}

	response, err := newTestSensitiveDataModelClient(client).CreateOrUpdate(
		context.Background(),
		resource,
		sensitiveDataModelRequest(resource),
	)
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %q, want compartmentId drift detail", err.Error())
	}
	requireCallCount(t, "UpdateSensitiveDataModel()", len(client.updateRequests), 0)
}

func TestSensitiveDataModelCreateErrorRecordsOpcRequestID(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	client := &fakeSensitiveDataModelOCIClient{
		createFn: func(context.Context, datasafesdk.CreateSensitiveDataModelRequest) (datasafesdk.CreateSensitiveDataModelResponse, error) {
			return datasafesdk.CreateSensitiveDataModelResponse{}, createErr
		},
	}

	response, err := newTestSensitiveDataModelClient(client).CreateOrUpdate(
		context.Background(),
		resource,
		sensitiveDataModelRequest(resource),
	)
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	requireCallCount(t, "CreateSensitiveDataModel()", len(client.createRequests), 1)
	requireOpcRequestID(t, resource, "opc-request-id")
	requireLastCondition(t, resource, shared.Failed)
}

func TestSensitiveDataModelDeleteKeepsFinalizerWhileLifecycleDeleting(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	active := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	deleting := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateDeleting)
	getResponses := []datasafesdk.GetSensitiveDataModelResponse{
		{SensitiveDataModel: active},
		{SensitiveDataModel: active},
		{SensitiveDataModel: deleting},
	}
	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			if len(getResponses) == 0 {
				t.Fatal("GetSensitiveDataModel() called more times than expected")
			}
			response := getResponses[0]
			getResponses = getResponses[1:]
			return response, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteSensitiveDataModelRequest) (datasafesdk.DeleteSensitiveDataModelResponse, error) {
			requireStringPtr(t, "DeleteSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			return datasafesdk.DeleteSensitiveDataModelResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
	}

	deleted, err := newTestSensitiveDataModelClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while lifecycle is DELETING")
	}
	requireCallCount(t, "GetSensitiveDataModel()", len(client.getRequests), 3)
	requireCallCount(t, "DeleteSensitiveDataModel()", len(client.deleteRequests), 1)
	requireOpcRequestID(t, resource, "opc-delete")
	requireAsync(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete")
	requireLastCondition(t, resource, shared.Terminating)
}

func TestSensitiveDataModelDeleteKeepsFinalizerWhenWorkRequestReadbackStaysActive(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	active := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	getResponses := []datasafesdk.GetSensitiveDataModelResponse{
		{SensitiveDataModel: active},
		{SensitiveDataModel: active},
		{SensitiveDataModel: active},
	}
	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			if len(getResponses) == 0 {
				t.Fatal("GetSensitiveDataModel() called more times than expected")
			}
			response := getResponses[0]
			getResponses = getResponses[1:]
			return response, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteSensitiveDataModelRequest) (datasafesdk.DeleteSensitiveDataModelResponse, error) {
			requireStringPtr(t, "DeleteSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			return datasafesdk.DeleteSensitiveDataModelResponse{
				OpcWorkRequestId: common.String("wr-delete-stale"),
				OpcRequestId:     common.String("opc-delete-stale"),
			}, nil
		},
	}

	deleted, err := newTestSensitiveDataModelClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while accepted delete is pending")
	}
	requireCallCount(t, "GetSensitiveDataModel()", len(client.getRequests), 3)
	requireCallCount(t, "DeleteSensitiveDataModel()", len(client.deleteRequests), 1)
	requireOpcRequestID(t, resource, "opc-delete-stale")
	requireWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-stale")
	requireLastCondition(t, resource, shared.Terminating)
}

func TestSensitiveDataModelPendingDeleteWorkRequestSuppressesDuplicateMutationOnStaleActiveReadback(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	seedSensitiveDataModelPendingWorkRequest(resource, shared.OSOKAsyncPhaseDelete, "wr-existing-delete")
	active := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)

	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			return datasafesdk.GetSensitiveDataModelResponse{SensitiveDataModel: active}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteSensitiveDataModelRequest) (datasafesdk.DeleteSensitiveDataModelResponse, error) {
			t.Fatal("DeleteSensitiveDataModel() reissued while prior delete work request was still pending")
			return datasafesdk.DeleteSensitiveDataModelResponse{}, nil
		},
	}

	deleted, err := newTestSensitiveDataModelClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while prior accepted delete is pending")
	}
	requireCallCount(t, "GetSensitiveDataModel()", len(client.getRequests), 1)
	requireCallCount(t, "DeleteSensitiveDataModel()", len(client.deleteRequests), 0)
	requireWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-existing-delete")
	requireLastCondition(t, resource, shared.Terminating)
}

func TestSensitiveDataModelDeleteConfirmsTerminalDeletedAndClearsAsync(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	active := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateActive)
	deletedModel := sdkSensitiveDataModel(resource, testSensitiveDataModelID, datasafesdk.DiscoveryLifecycleStateDeleted)
	getResponses := []datasafesdk.GetSensitiveDataModelResponse{
		{SensitiveDataModel: active},
		{SensitiveDataModel: active},
		{SensitiveDataModel: deletedModel},
	}
	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			requireStringPtr(t, "GetSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			if len(getResponses) == 0 {
				t.Fatal("GetSensitiveDataModel() called more times than expected")
			}
			response := getResponses[0]
			getResponses = getResponses[1:]
			return response, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteSensitiveDataModelRequest) (datasafesdk.DeleteSensitiveDataModelResponse, error) {
			requireStringPtr(t, "DeleteSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
			return datasafesdk.DeleteSensitiveDataModelResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
	}

	deleted, err := newTestSensitiveDataModelClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after terminal DELETED readback")
	}
	requireCallCount(t, "GetSensitiveDataModel()", len(client.getRequests), 3)
	requireCallCount(t, "DeleteSensitiveDataModel()", len(client.deleteRequests), 1)
	requireOpcRequestID(t, resource, "opc-delete")
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after delete confirmation", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.LifecycleState; got != string(datasafesdk.DiscoveryLifecycleStateDeleted) {
		t.Fatalf("status.lifecycleState = %q, want DELETED", got)
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestSensitiveDataModelDeleteRejectsAuthShapedPreDeleteGet(t *testing.T) {
	resource := makeSensitiveDataModelResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveDataModelID)
	resource.Status.Id = testSensitiveDataModelID
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	client := &fakeSensitiveDataModelOCIClient{
		getFn: func(context.Context, datasafesdk.GetSensitiveDataModelRequest) (datasafesdk.GetSensitiveDataModelResponse, error) {
			return datasafesdk.GetSensitiveDataModelResponse{}, authErr
		},
		deleteFn: func(context.Context, datasafesdk.DeleteSensitiveDataModelRequest) (datasafesdk.DeleteSensitiveDataModelResponse, error) {
			t.Fatal("DeleteSensitiveDataModel() called after ambiguous pre-delete get")
			return datasafesdk.DeleteSensitiveDataModelResponse{}, nil
		},
	}

	deleted, err := newTestSensitiveDataModelClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous NotAuthorizedOrNotFound")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want NotAuthorizedOrNotFound detail", err.Error())
	}
	requireCallCount(t, "DeleteSensitiveDataModel()", len(client.deleteRequests), 0)
	requireOpcRequestID(t, resource, "opc-request-id")
}

func newTestSensitiveDataModelClient(client *fakeSensitiveDataModelOCIClient) SensitiveDataModelServiceClient {
	return newSensitiveDataModelServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
}

func makeSensitiveDataModelResource() *datasafev1beta1.SensitiveDataModel {
	return &datasafev1beta1.SensitiveDataModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSensitiveDataModelName,
			Namespace: "default",
			UID:       types.UID(testSensitiveDataModelUID),
		},
		Spec: datasafev1beta1.SensitiveDataModelSpec{
			CompartmentId:                        testSensitiveDataModelCompartmentID,
			TargetId:                             testSensitiveDataModelTargetID,
			DisplayName:                          testSensitiveDataModelName,
			AppSuiteName:                         "GENERIC",
			Description:                          "customer data model",
			SchemasForDiscovery:                  []string{"HR", "OE"},
			TablesForDiscovery:                   []datasafev1beta1.SensitiveDataModelTablesForDiscovery{{SchemaName: "HR", TableNames: []string{"EMPLOYEES", "JOBS"}}},
			SensitiveTypeIdsForDiscovery:         []string{"ocid1.datasafesensitivetype.oc1..type"},
			SensitiveTypeGroupIdsForDiscovery:    []string{"ocid1.datasafesensitivetypegroup.oc1..group"},
			IsSampleDataCollectionEnabled:        false,
			IsAppDefinedRelationDiscoveryEnabled: true,
			IsIncludeAllSchemas:                  false,
			IsIncludeAllSensitiveTypes:           true,
			FreeformTags:                         map[string]string{"env": "test"},
			DefinedTags:                          map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func sensitiveDataModelRequest(resource *datasafev1beta1.SensitiveDataModel) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}}
}

func sdkSensitiveDataModel(
	resource *datasafev1beta1.SensitiveDataModel,
	id string,
	state datasafesdk.DiscoveryLifecycleStateEnum,
) datasafesdk.SensitiveDataModel {
	now := common.SDKTime{Time: time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)}
	return datasafesdk.SensitiveDataModel{
		Id:                                   common.String(id),
		DisplayName:                          common.String(resource.Spec.DisplayName),
		CompartmentId:                        common.String(resource.Spec.CompartmentId),
		TargetId:                             common.String(resource.Spec.TargetId),
		TimeCreated:                          &now,
		TimeUpdated:                          &now,
		LifecycleState:                       state,
		AppSuiteName:                         common.String(resource.Spec.AppSuiteName),
		IsSampleDataCollectionEnabled:        common.Bool(resource.Spec.IsSampleDataCollectionEnabled),
		IsAppDefinedRelationDiscoveryEnabled: common.Bool(resource.Spec.IsAppDefinedRelationDiscoveryEnabled),
		IsIncludeAllSchemas:                  common.Bool(resource.Spec.IsIncludeAllSchemas),
		IsIncludeAllSensitiveTypes:           common.Bool(resource.Spec.IsIncludeAllSensitiveTypes),
		Description:                          common.String(resource.Spec.Description),
		SchemasForDiscovery:                  cloneSensitiveDataModelStringSlice(resource.Spec.SchemasForDiscovery),
		TablesForDiscovery:                   sensitiveDataModelTablesForDiscoveryFromSpec(resource.Spec.TablesForDiscovery),
		SensitiveTypeIdsForDiscovery:         cloneSensitiveDataModelStringSlice(resource.Spec.SensitiveTypeIdsForDiscovery),
		SensitiveTypeGroupIdsForDiscovery:    cloneSensitiveDataModelStringSlice(resource.Spec.SensitiveTypeGroupIdsForDiscovery),
		FreeformTags:                         map[string]string{"env": resource.Spec.FreeformTags["env"]},
		DefinedTags:                          sensitiveDataModelDefinedTagsFromSpec(resource.Spec.DefinedTags),
		SystemTags:                           map[string]map[string]interface{}{"orcl-cloud": {"retained": "true"}},
	}
}

func sdkSensitiveDataModelSummary(
	resource *datasafev1beta1.SensitiveDataModel,
	id string,
	displayName string,
	state datasafesdk.DiscoveryLifecycleStateEnum,
) datasafesdk.SensitiveDataModelSummary {
	now := common.SDKTime{Time: time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)}
	return datasafesdk.SensitiveDataModelSummary{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		TargetId:       common.String(resource.Spec.TargetId),
		TimeCreated:    &now,
		TimeUpdated:    &now,
		LifecycleState: state,
		AppSuiteName:   common.String(resource.Spec.AppSuiteName),
		Description:    common.String(resource.Spec.Description),
		FreeformTags:   map[string]string{"env": resource.Spec.FreeformTags["env"]},
		DefinedTags:    sensitiveDataModelDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
}

func requireSensitiveDataModelCreateRequest(
	t *testing.T,
	request datasafesdk.CreateSensitiveDataModelRequest,
	resource *datasafev1beta1.SensitiveDataModel,
) {
	t.Helper()
	requireStringPtr(t, "CreateSensitiveDataModelRequest.OpcRetryToken", request.OpcRetryToken, testSensitiveDataModelUID)
	details := request.CreateSensitiveDataModelDetails
	requireStringPtr(t, "CreateSensitiveDataModelDetails.CompartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateSensitiveDataModelDetails.TargetId", details.TargetId, resource.Spec.TargetId)
	requireStringPtr(t, "CreateSensitiveDataModelDetails.DisplayName", details.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "CreateSensitiveDataModelDetails.AppSuiteName", details.AppSuiteName, resource.Spec.AppSuiteName)
	requireStringPtr(t, "CreateSensitiveDataModelDetails.Description", details.Description, resource.Spec.Description)
	requireBoolPtr(t, "CreateSensitiveDataModelDetails.IsIncludeAllSensitiveTypes", details.IsIncludeAllSensitiveTypes, true)
	if !reflect.DeepEqual(details.SchemasForDiscovery, resource.Spec.SchemasForDiscovery) {
		t.Fatalf("CreateSensitiveDataModelDetails.SchemasForDiscovery = %#v, want %#v", details.SchemasForDiscovery, resource.Spec.SchemasForDiscovery)
	}
	requireTablesForDiscovery(t, details.TablesForDiscovery)
	if !reflect.DeepEqual(details.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("CreateSensitiveDataModelDetails.FreeformTags = %#v, want %#v", details.FreeformTags, resource.Spec.FreeformTags)
	}
}

func requireSensitiveDataModelUpdateRequest(
	t *testing.T,
	request datasafesdk.UpdateSensitiveDataModelRequest,
	resource *datasafev1beta1.SensitiveDataModel,
) {
	t.Helper()
	requireStringPtr(t, "UpdateSensitiveDataModelRequest.SensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
	details := request.UpdateSensitiveDataModelDetails
	requireStringPtr(t, "UpdateSensitiveDataModelDetails.DisplayName", details.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "UpdateSensitiveDataModelDetails.Description", details.Description, resource.Spec.Description)
	requireBoolPtr(t, "UpdateSensitiveDataModelDetails.IsSampleDataCollectionEnabled", details.IsSampleDataCollectionEnabled, true)
	if !reflect.DeepEqual(details.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("UpdateSensitiveDataModelDetails.FreeformTags = %#v, want %#v", details.FreeformTags, resource.Spec.FreeformTags)
	}
}

func requireStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", field, *got, want)
	}
}

func requireBoolPtr(t *testing.T, field string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %t", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", field, *got, want)
	}
}

func requireTablesForDiscovery(t *testing.T, got []datasafesdk.TablesForDiscovery) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("TablesForDiscovery len = %d, want 1", len(got))
	}
	requireStringPtr(t, "TablesForDiscovery[0].SchemaName", got[0].SchemaName, "HR")
	if !reflect.DeepEqual(got[0].TableNames, []string{"EMPLOYEES", "JOBS"}) {
		t.Fatalf("TablesForDiscovery[0].TableNames = %#v, want %#v", got[0].TableNames, []string{"EMPLOYEES", "JOBS"})
	}
}

func requireRecordedSensitiveDataModelID(t *testing.T, resource *datasafev1beta1.SensitiveDataModel, want string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func requireOpcRequestID(t *testing.T, resource *datasafev1beta1.SensitiveDataModel, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func requireAsync(
	t *testing.T,
	resource *datasafev1beta1.SensitiveDataModel,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want current async operation")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.status.async.current.source = %q, want lifecycle", current.Source)
	}
	if current.Phase != phase {
		t.Fatalf("status.status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func requireWorkRequestAsync(
	t *testing.T,
	resource *datasafev1beta1.SensitiveDataModel,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want current async operation")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.status.async.current.source = %q, want workrequest", current.Source)
	}
	if current.Phase != phase {
		t.Fatalf("status.status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func seedSensitiveDataModelPendingWorkRequest(
	resource *datasafev1beta1.SensitiveDataModel,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) {
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            phase,
		WorkRequestID:    workRequestID,
		RawStatus:        string(datasafesdk.WorkRequestStatusAccepted),
		RawOperationType: sensitiveDataModelWorkRequestOperationType(phase),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          "accepted work request is pending",
		UpdatedAt:        &now,
	}
}

func requireLastCondition(t *testing.T, resource *datasafev1beta1.SensitiveDataModel, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions = empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireCallCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}

func assertContainsAll(t *testing.T, name string, got []string, want ...string) {
	t.Helper()
	values := map[string]struct{}{}
	for _, value := range got {
		values[value] = struct{}{}
	}
	for _, value := range want {
		if _, ok := values[value]; !ok {
			t.Fatalf("%s = %#v, missing %q", name, got, value)
		}
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

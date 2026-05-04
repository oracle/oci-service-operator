/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package resourceanalyticsinstance

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	resourceanalyticssdk "github.com/oracle/oci-go-sdk/v65/resourceanalytics"
	resourceanalyticsv1beta1 "github.com/oracle/oci-service-operator/api/resourceanalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeResourceAnalyticsInstanceOCIClient struct {
	createFn      func(context.Context, resourceanalyticssdk.CreateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse, error)
	getFn         func(context.Context, resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error)
	listFn        func(context.Context, resourceanalyticssdk.ListResourceAnalyticsInstancesRequest) (resourceanalyticssdk.ListResourceAnalyticsInstancesResponse, error)
	updateFn      func(context.Context, resourceanalyticssdk.UpdateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.UpdateResourceAnalyticsInstanceResponse, error)
	deleteFn      func(context.Context, resourceanalyticssdk.DeleteResourceAnalyticsInstanceRequest) (resourceanalyticssdk.DeleteResourceAnalyticsInstanceResponse, error)
	workRequestFn func(context.Context, resourceanalyticssdk.GetWorkRequestRequest) (resourceanalyticssdk.GetWorkRequestResponse, error)

	createRequests      []resourceanalyticssdk.CreateResourceAnalyticsInstanceRequest
	getRequests         []resourceanalyticssdk.GetResourceAnalyticsInstanceRequest
	listRequests        []resourceanalyticssdk.ListResourceAnalyticsInstancesRequest
	updateRequests      []resourceanalyticssdk.UpdateResourceAnalyticsInstanceRequest
	deleteRequests      []resourceanalyticssdk.DeleteResourceAnalyticsInstanceRequest
	workRequestRequests []resourceanalyticssdk.GetWorkRequestRequest
}

func (f *fakeResourceAnalyticsInstanceOCIClient) CreateResourceAnalyticsInstance(ctx context.Context, request resourceanalyticssdk.CreateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse{}, nil
}

func (f *fakeResourceAnalyticsInstanceOCIClient) GetResourceAnalyticsInstance(ctx context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{}, errortest.NewServiceError(404, "NotFound", "missing resource analytics instance")
}

func (f *fakeResourceAnalyticsInstanceOCIClient) ListResourceAnalyticsInstances(ctx context.Context, request resourceanalyticssdk.ListResourceAnalyticsInstancesRequest) (resourceanalyticssdk.ListResourceAnalyticsInstancesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{}, nil
}

func (f *fakeResourceAnalyticsInstanceOCIClient) UpdateResourceAnalyticsInstance(ctx context.Context, request resourceanalyticssdk.UpdateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.UpdateResourceAnalyticsInstanceResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return resourceanalyticssdk.UpdateResourceAnalyticsInstanceResponse{}, nil
}

func (f *fakeResourceAnalyticsInstanceOCIClient) DeleteResourceAnalyticsInstance(ctx context.Context, request resourceanalyticssdk.DeleteResourceAnalyticsInstanceRequest) (resourceanalyticssdk.DeleteResourceAnalyticsInstanceResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return resourceanalyticssdk.DeleteResourceAnalyticsInstanceResponse{}, nil
}

func (f *fakeResourceAnalyticsInstanceOCIClient) GetWorkRequest(ctx context.Context, request resourceanalyticssdk.GetWorkRequestRequest) (resourceanalyticssdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, request)
	}
	return resourceanalyticssdk.GetWorkRequestResponse{}, nil
}

func TestResourceAnalyticsInstanceRuntimeSemanticsEncodesReviewedContract(t *testing.T) {
	semantics := resourceAnalyticsInstanceRuntimeSemantics()
	if semantics == nil {
		t.Fatal("resourceAnalyticsInstanceRuntimeSemantics() = nil")
	}
	if semantics.Async == nil || semantics.Async.WorkRequest == nil {
		t.Fatalf("Async.WorkRequest = %#v, want service-sdk work request contract", semantics.Async)
	}
	requireResourceAnalyticsInstanceStringSliceEqual(t, "Async.WorkRequest.Phases", semantics.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	requireResourceAnalyticsInstanceStringSliceEqual(t, "Mutation.Mutable", semantics.Mutation.Mutable, []string{"displayName", "description", "freeformTags", "definedTags"})
	requireResourceAnalyticsInstanceStringSliceEqual(t, "Mutation.ForceNew", semantics.Mutation.ForceNew, []string{
		"compartmentId",
		"adwAdminPassword",
		"subnetId",
		"isMutualTlsRequired",
		"nsgIds",
		"licenseModel",
	})
}

func TestResourceAnalyticsInstanceCreateBuildsPolymorphicBodyAndTracksWorkRequest(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	requireResourceAnalyticsInstanceNoCreateOnlyFingerprint(t, resource)
	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.createFn = func(_ context.Context, request resourceanalyticssdk.CreateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse, error) {
		if request.CompartmentId == nil || *request.CompartmentId != resource.Spec.CompartmentId {
			t.Fatalf("Create.CompartmentId = %v, want %q", request.CompartmentId, resource.Spec.CompartmentId)
		}
		if _, ok := request.AdwAdminPassword.(resourceanalyticssdk.PlainTextPasswordDetails); !ok {
			t.Fatalf("Create.AdwAdminPassword = %T, want PlainTextPasswordDetails", request.AdwAdminPassword)
		}
		return resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: testResourceAnalyticsInstanceBody("rai-1", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateCreating),
			OpcRequestId:              common.String("opc-create"),
			OpcWorkRequestId:          common.String("wr-create"),
		}, nil
	}
	client.workRequestFn = func(_ context.Context, request resourceanalyticssdk.GetWorkRequestRequest) (resourceanalyticssdk.GetWorkRequestResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "GetWorkRequest.WorkRequestId", request.WorkRequestId, "wr-create")
		return resourceanalyticssdk.GetWorkRequestResponse{
			WorkRequest: testResourceAnalyticsInstanceWorkRequest(
				"wr-create",
				resourceanalyticssdk.OperationTypeCreateResourceAnalyticsInstance,
				resourceanalyticssdk.OperationStatusInProgress,
				resourceanalyticssdk.ActionTypeCreated,
				"rai-1",
			),
		}, nil
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateResourceAnalyticsInstance() calls = %d, want 1", len(client.createRequests))
	}
	requireResourceAnalyticsInstanceCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create")
	requireResourceAnalyticsInstanceOpcRequestID(t, resource, "opc-create")
	if _, ok := resourceAnalyticsInstanceRecordedCreateOnlyFingerprint(resource); !ok {
		t.Fatal("create-only fingerprint was not recorded after create")
	}
}

func TestResourceAnalyticsInstanceBindsFromPaginatedListWithoutCreate(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	requireResourceAnalyticsInstanceNoCreateOnlyFingerprint(t, resource)
	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.listFn = func(_ context.Context, request resourceanalyticssdk.ListResourceAnalyticsInstancesRequest) (resourceanalyticssdk.ListResourceAnalyticsInstancesResponse, error) {
		switch resourceAnalyticsInstanceStringValue(request.Page) {
		case "":
			return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{
				ResourceAnalyticsInstanceCollection: resourceanalyticssdk.ResourceAnalyticsInstanceCollection{
					Items: []resourceanalyticssdk.ResourceAnalyticsInstanceSummary{
						testResourceAnalyticsInstanceSummary("rai-other", "other", resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{
				ResourceAnalyticsInstanceCollection: resourceanalyticssdk.ResourceAnalyticsInstanceCollection{
					Items: []resourceanalyticssdk.ResourceAnalyticsInstanceSummary{
						testResourceAnalyticsInstanceSummary("rai-bound", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected list page %q", resourceAnalyticsInstanceStringValue(request.Page))
			return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{}, nil
		}
	}
	client.getFn = func(_ context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Get.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-bound")
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: testResourceAnalyticsInstanceBody("rai-bound", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive),
		}, nil
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateResourceAnalyticsInstance() calls = %d, want 0", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListResourceAnalyticsInstances() calls = %d, want 2", len(client.listRequests))
	}
	requireResourceAnalyticsInstanceTrackedID(t, resource, "rai-bound")
	if _, ok := resourceAnalyticsInstanceRecordedCreateOnlyFingerprint(resource); !ok {
		t.Fatal("create-only fingerprint was not recorded after bind")
	}
}

func TestResourceAnalyticsInstanceBindIgnoresTerminalLifecycleSummaries(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.listFn = func(_ context.Context, request resourceanalyticssdk.ListResourceAnalyticsInstancesRequest) (resourceanalyticssdk.ListResourceAnalyticsInstancesResponse, error) {
		switch resourceAnalyticsInstanceStringValue(request.Page) {
		case "":
			return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{
				ResourceAnalyticsInstanceCollection: resourceanalyticssdk.ResourceAnalyticsInstanceCollection{
					Items: []resourceanalyticssdk.ResourceAnalyticsInstanceSummary{
						testResourceAnalyticsInstanceSummary("rai-deleting", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateDeleting),
						testResourceAnalyticsInstanceSummary("rai-deleted", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateDeleted),
						testResourceAnalyticsInstanceSummary("rai-failed", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateFailed),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{
				ResourceAnalyticsInstanceCollection: resourceanalyticssdk.ResourceAnalyticsInstanceCollection{
					Items: []resourceanalyticssdk.ResourceAnalyticsInstanceSummary{
						testResourceAnalyticsInstanceSummary("rai-active", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected list page %q", resourceAnalyticsInstanceStringValue(request.Page))
			return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{}, nil
		}
	}
	client.getFn = func(_ context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Get.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-active")
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: testResourceAnalyticsInstanceBody("rai-active", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive),
		}, nil
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateResourceAnalyticsInstance() calls = %d, want 0 after active bind match", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListResourceAnalyticsInstances() calls = %d, want 2", len(client.listRequests))
	}
	requireResourceAnalyticsInstanceTrackedID(t, resource, "rai-active")
}

func TestResourceAnalyticsInstanceCreateIgnoresOnlyTerminalLifecycleMatches(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.listFn = func(_ context.Context, request resourceanalyticssdk.ListResourceAnalyticsInstancesRequest) (resourceanalyticssdk.ListResourceAnalyticsInstancesResponse, error) {
		if got := resourceAnalyticsInstanceStringValue(request.Page); got != "" {
			t.Fatalf("List.Page = %q, want empty", got)
		}
		return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{
			ResourceAnalyticsInstanceCollection: resourceanalyticssdk.ResourceAnalyticsInstanceCollection{
				Items: []resourceanalyticssdk.ResourceAnalyticsInstanceSummary{
					testResourceAnalyticsInstanceSummary("rai-deleting", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateDeleting),
					testResourceAnalyticsInstanceSummary("rai-deleted", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateDeleted),
					testResourceAnalyticsInstanceSummary("rai-failed", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateFailed),
				},
			},
		}, nil
	}
	client.createFn = func(_ context.Context, request resourceanalyticssdk.CreateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Create.DisplayName", request.DisplayName, resource.Spec.DisplayName)
		return resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: testResourceAnalyticsInstanceBody("rai-created", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateCreating),
			OpcWorkRequestId:          common.String("wr-create"),
		}, nil
	}
	client.workRequestFn = func(_ context.Context, request resourceanalyticssdk.GetWorkRequestRequest) (resourceanalyticssdk.GetWorkRequestResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "GetWorkRequest.WorkRequestId", request.WorkRequestId, "wr-create")
		return resourceanalyticssdk.GetWorkRequestResponse{
			WorkRequest: testResourceAnalyticsInstanceWorkRequest(
				"wr-create",
				resourceanalyticssdk.OperationTypeCreateResourceAnalyticsInstance,
				resourceanalyticssdk.OperationStatusInProgress,
				resourceanalyticssdk.ActionTypeCreated,
				"rai-created",
			),
		}, nil
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful create requeue", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateResourceAnalyticsInstance() calls = %d, want 1 after terminal-only list", len(client.createRequests))
	}
	if len(client.listRequests) == 0 {
		t.Fatal("ListResourceAnalyticsInstances() calls = 0, want pre-create lookup before create")
	}
	requireResourceAnalyticsInstanceCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create")
}

func TestResourceAnalyticsInstanceNoopReconcileUsesTrackedGet(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	resource.Status.Id = "rai-1"
	resource.Status.OsokStatus.Ocid = shared.OCID("rai-1")
	recordResourceAnalyticsInstanceCreateOnlyFingerprint(resource)

	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.getFn = func(_ context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Get.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-1")
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: testResourceAnalyticsInstanceBody("rai-1", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive),
			OpcRequestId:              common.String("opc-get"),
		}, nil
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateResourceAnalyticsInstance() calls = %d, want 0", len(client.updateRequests))
	}
}

func TestResourceAnalyticsInstanceOmittedDescriptionDoesNotClearReadback(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	resource.Spec.Description = ""
	resource.Status.Id = "rai-1"
	resource.Status.OsokStatus.Ocid = shared.OCID("rai-1")
	recordResourceAnalyticsInstanceCreateOnlyFingerprint(resource)

	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.getFn = func(_ context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Get.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-1")
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: resourceanalyticssdk.ResourceAnalyticsInstance{
				Id:             common.String("rai-1"),
				DisplayName:    common.String(resource.Spec.DisplayName),
				CompartmentId:  common.String(resource.Spec.CompartmentId),
				LifecycleState: resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive,
				Description:    common.String("existing OCI description"),
				FreeformTags:   map[string]string{"env": "test"},
				DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
			},
		}, nil
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateResourceAnalyticsInstance() calls = %d, want 0 when spec.description is omitted", len(client.updateRequests))
	}
}

func TestResourceAnalyticsInstanceMutableUpdatePreservesExplicitEmptyTags(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	resource.Spec.Description = "new description"
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	resource.Status.Id = "rai-1"
	resource.Status.OsokStatus.Ocid = shared.OCID("rai-1")
	recordResourceAnalyticsInstanceCreateOnlyFingerprint(resource)

	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.getFn = func(_ context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Get.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-1")
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: resourceanalyticssdk.ResourceAnalyticsInstance{
				Id:             common.String("rai-1"),
				DisplayName:    common.String(resource.Spec.DisplayName),
				CompartmentId:  common.String(resource.Spec.CompartmentId),
				LifecycleState: resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive,
				Description:    common.String("old description"),
				FreeformTags:   map[string]string{"remove": "me"},
				DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
			},
		}, nil
	}
	client.updateFn = func(_ context.Context, request resourceanalyticssdk.UpdateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.UpdateResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Update.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-1")
		if request.Description == nil || *request.Description != "new description" {
			t.Fatalf("Update.Description = %v, want new description", request.Description)
		}
		if len(request.FreeformTags) != 0 {
			t.Fatalf("Update.FreeformTags = %#v, want explicit empty map", request.FreeformTags)
		}
		if len(request.DefinedTags) != 0 {
			t.Fatalf("Update.DefinedTags = %#v, want explicit empty map", request.DefinedTags)
		}
		return resourceanalyticssdk.UpdateResourceAnalyticsInstanceResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update"),
		}, nil
	}
	client.workRequestFn = func(_ context.Context, _ resourceanalyticssdk.GetWorkRequestRequest) (resourceanalyticssdk.GetWorkRequestResponse, error) {
		return resourceanalyticssdk.GetWorkRequestResponse{
			WorkRequest: testResourceAnalyticsInstanceWorkRequest(
				"wr-update",
				resourceanalyticssdk.OperationTypeUpdateResourceAnalyticsInstance,
				resourceanalyticssdk.OperationStatusInProgress,
				resourceanalyticssdk.ActionTypeUpdated,
				"rai-1",
			),
		}, nil
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdateResourceAnalyticsInstance() calls = %d, want 1", len(client.updateRequests))
	}
	requireResourceAnalyticsInstanceCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, "wr-update")
	requireResourceAnalyticsInstanceOpcRequestID(t, resource, "opc-update")
}

func TestResourceAnalyticsInstanceRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	resource.Status.Id = "rai-1"
	resource.Status.OsokStatus.Ocid = shared.OCID("rai-1")
	recordResourceAnalyticsInstanceCreateOnlyFingerprint(resource)
	resource.Spec.SubnetId = "subnet-changed"
	resource.Spec.Description = "new description"

	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.getFn = func(_ context.Context, _ resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: testResourceAnalyticsInstanceBody("rai-1", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive),
		}, nil
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "create-only fields change") {
		t.Fatalf("CreateOrUpdate() error = %q, want create-only drift message", err.Error())
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateResourceAnalyticsInstance() calls = %d, want 0", len(client.updateRequests))
	}
}

func TestResourceAnalyticsInstanceRejectsMissingCreateOnlyFingerprintBeforeUpdate(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	markResourceAnalyticsInstanceEstablished(resource, "rai-1")
	resource.Spec.SubnetId = "subnet-changed"
	resource.Spec.Description = "new description"
	requireResourceAnalyticsInstanceNoCreateOnlyFingerprint(t, resource)

	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.getFn = func(_ context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Get.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-1")
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: testResourceAnalyticsInstanceBody("rai-1", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive),
		}, nil
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing create-only fingerprint rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "create-only fingerprint is missing") {
		t.Fatalf("CreateOrUpdate() error = %q, want missing create-only fingerprint message", err.Error())
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateResourceAnalyticsInstance() calls = %d, want 0", len(client.updateRequests))
	}
	requireResourceAnalyticsInstanceNoCreateOnlyFingerprint(t, resource)
}

func TestResourceAnalyticsInstanceRejectsMissingCreateOnlyFingerprintBeforeNoop(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	markResourceAnalyticsInstanceEstablished(resource, "rai-1")
	requireResourceAnalyticsInstanceNoCreateOnlyFingerprint(t, resource)

	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.getFn = func(_ context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Get.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-1")
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: testResourceAnalyticsInstanceBody("rai-1", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive),
		}, nil
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing create-only fingerprint rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "create-only fingerprint is missing") {
		t.Fatalf("CreateOrUpdate() error = %q, want missing create-only fingerprint message", err.Error())
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateResourceAnalyticsInstance() calls = %d, want 0", len(client.updateRequests))
	}
	requireResourceAnalyticsInstanceNoCreateOnlyFingerprint(t, resource)
}

func TestResourceAnalyticsInstanceDeleteTracksWorkRequestUntilConfirmed(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	resource.Status.Id = "rai-1"
	resource.Status.OsokStatus.Ocid = shared.OCID("rai-1")
	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.getFn = func(_ context.Context, _ resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{
			ResourceAnalyticsInstance: testResourceAnalyticsInstanceBody("rai-1", resource.Spec.DisplayName, resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive),
		}, nil
	}
	client.deleteFn = func(_ context.Context, request resourceanalyticssdk.DeleteResourceAnalyticsInstanceRequest) (resourceanalyticssdk.DeleteResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Delete.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-1")
		return resourceanalyticssdk.DeleteResourceAnalyticsInstanceResponse{
			OpcRequestId:     common.String("opc-delete"),
			OpcWorkRequestId: common.String("wr-delete"),
		}, nil
	}
	client.workRequestFn = func(_ context.Context, _ resourceanalyticssdk.GetWorkRequestRequest) (resourceanalyticssdk.GetWorkRequestResponse, error) {
		return resourceanalyticssdk.GetWorkRequestResponse{
			WorkRequest: testResourceAnalyticsInstanceWorkRequest(
				"wr-delete",
				resourceanalyticssdk.OperationTypeDeleteResourceAnalyticsInstance,
				resourceanalyticssdk.OperationStatusInProgress,
				resourceanalyticssdk.ActionTypeDeleted,
				"rai-1",
			),
		}, nil
	}

	deleted, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want pending while work request is in progress")
	}
	requireResourceAnalyticsInstanceCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete")
	requireResourceAnalyticsInstanceOpcRequestID(t, resource, "opc-delete")
}

func TestResourceAnalyticsInstanceDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	resource.Status.Id = "rai-1"
	resource.Status.OsokStatus.Ocid = shared.OCID("rai-1")
	authNotFound := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous missing resource analytics instance")

	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.getFn = func(_ context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Get.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-1")
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{}, authNotFound
	}

	deleted, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped pre-delete confirm-read rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous pre-delete confirm-read")
	}
	if !strings.Contains(err.Error(), "pre-delete confirmation returned authorization-shaped not found") {
		t.Fatalf("Delete() error = %q, want pre-delete confirmation context", err.Error())
	}
	requireResourceAnalyticsInstanceOpcRequestID(t, resource, authNotFound.GetOpcRequestID())
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteResourceAnalyticsInstance() calls = %d, want 0 after ambiguous pre-delete confirm-read", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %#v, want nil for ambiguous pre-delete confirm-read", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestResourceAnalyticsInstanceDeleteRejectsAuthShapedCompletedWorkRequestConfirmRead(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	resource.Status.Id = "rai-1"
	resource.Status.OsokStatus.Ocid = shared.OCID("rai-1")
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &metav1.Time{},
	}
	authNotFound := errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "ambiguous missing resource analytics instance")

	client := &fakeResourceAnalyticsInstanceOCIClient{}
	client.workRequestFn = func(_ context.Context, request resourceanalyticssdk.GetWorkRequestRequest) (resourceanalyticssdk.GetWorkRequestResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "GetWorkRequest.WorkRequestId", request.WorkRequestId, "wr-delete")
		return resourceanalyticssdk.GetWorkRequestResponse{
			WorkRequest: testResourceAnalyticsInstanceWorkRequest(
				"wr-delete",
				resourceanalyticssdk.OperationTypeDeleteResourceAnalyticsInstance,
				resourceanalyticssdk.OperationStatusSucceeded,
				resourceanalyticssdk.ActionTypeDeleted,
				"rai-1",
			),
		}, nil
	}
	client.getFn = func(_ context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
		requireResourceAnalyticsInstanceStringPtr(t, "Get.ResourceAnalyticsInstanceId", request.ResourceAnalyticsInstanceId, "rai-1")
		return resourceanalyticssdk.GetResourceAnalyticsInstanceResponse{}, authNotFound
	}

	deleted, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirm-read rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous confirm-read")
	}
	if !strings.Contains(err.Error(), "authorization-shaped not found") {
		t.Fatalf("Delete() error = %q, want authorization-shaped not found", err.Error())
	}
	requireResourceAnalyticsInstanceOpcRequestID(t, resource, authNotFound.GetOpcRequestID())
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteResourceAnalyticsInstance() calls = %d, want 0 while confirming completed work request", len(client.deleteRequests))
	}
}

func TestResourceAnalyticsInstanceCreateErrorRecordsOpcRequestID(t *testing.T) {
	resource := newTestResourceAnalyticsInstance()
	createErr := errortest.NewServiceError(409, "IncorrectState", "create conflict")
	client := &fakeResourceAnalyticsInstanceOCIClient{
		createFn: func(context.Context, resourceanalyticssdk.CreateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse, error) {
			return resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse{}, createErr
		},
	}

	response, err := newResourceAnalyticsInstanceServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	requireResourceAnalyticsInstanceOpcRequestID(t, resource, createErr.GetOpcRequestID())
}

func newResourceAnalyticsInstanceServiceClientWithOCIClient(client resourceAnalyticsInstanceOCIClient) ResourceAnalyticsInstanceServiceClient {
	manager := &ResourceAnalyticsInstanceServiceManager{}
	hooks := newResourceAnalyticsInstanceRuntimeHooksWithOCIClient(client)
	applyResourceAnalyticsInstanceRuntimeHooks(&hooks, client, nil)
	delegate := defaultResourceAnalyticsInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*resourceanalyticsv1beta1.ResourceAnalyticsInstance](
			buildResourceAnalyticsInstanceGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapResourceAnalyticsInstanceGeneratedClient(hooks, delegate)
}

func newResourceAnalyticsInstanceRuntimeHooksWithOCIClient(client resourceAnalyticsInstanceOCIClient) ResourceAnalyticsInstanceRuntimeHooks {
	return ResourceAnalyticsInstanceRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*resourceanalyticsv1beta1.ResourceAnalyticsInstance]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*resourceanalyticsv1beta1.ResourceAnalyticsInstance]{},
		StatusHooks:     generatedruntime.StatusHooks[*resourceanalyticsv1beta1.ResourceAnalyticsInstance]{},
		ParityHooks:     generatedruntime.ParityHooks[*resourceanalyticsv1beta1.ResourceAnalyticsInstance]{},
		Async:           generatedruntime.AsyncHooks[*resourceanalyticsv1beta1.ResourceAnalyticsInstance]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*resourceanalyticsv1beta1.ResourceAnalyticsInstance]{},
		Create: runtimeOperationHooks[resourceanalyticssdk.CreateResourceAnalyticsInstanceRequest, resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateResourceAnalyticsInstanceDetails", RequestName: "CreateResourceAnalyticsInstanceDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request resourceanalyticssdk.CreateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse, error) {
				return client.CreateResourceAnalyticsInstance(ctx, request)
			},
		},
		Get: runtimeOperationHooks[resourceanalyticssdk.GetResourceAnalyticsInstanceRequest, resourceanalyticssdk.GetResourceAnalyticsInstanceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ResourceAnalyticsInstanceId", RequestName: "resourceAnalyticsInstanceId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error) {
				return client.GetResourceAnalyticsInstance(ctx, request)
			},
		},
		List: runtimeOperationHooks[resourceanalyticssdk.ListResourceAnalyticsInstancesRequest, resourceanalyticssdk.ListResourceAnalyticsInstancesResponse]{
			Fields: resourceAnalyticsInstanceListFields(),
			Call: func(ctx context.Context, request resourceanalyticssdk.ListResourceAnalyticsInstancesRequest) (resourceanalyticssdk.ListResourceAnalyticsInstancesResponse, error) {
				return client.ListResourceAnalyticsInstances(ctx, request)
			},
		},
		Update: runtimeOperationHooks[resourceanalyticssdk.UpdateResourceAnalyticsInstanceRequest, resourceanalyticssdk.UpdateResourceAnalyticsInstanceResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ResourceAnalyticsInstanceId", RequestName: "resourceAnalyticsInstanceId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateResourceAnalyticsInstanceDetails", RequestName: "UpdateResourceAnalyticsInstanceDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request resourceanalyticssdk.UpdateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.UpdateResourceAnalyticsInstanceResponse, error) {
				return client.UpdateResourceAnalyticsInstance(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[resourceanalyticssdk.DeleteResourceAnalyticsInstanceRequest, resourceanalyticssdk.DeleteResourceAnalyticsInstanceResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ResourceAnalyticsInstanceId", RequestName: "resourceAnalyticsInstanceId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request resourceanalyticssdk.DeleteResourceAnalyticsInstanceRequest) (resourceanalyticssdk.DeleteResourceAnalyticsInstanceResponse, error) {
				return client.DeleteResourceAnalyticsInstance(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ResourceAnalyticsInstanceServiceClient) ResourceAnalyticsInstanceServiceClient{},
	}
}

func newTestResourceAnalyticsInstance() *resourceanalyticsv1beta1.ResourceAnalyticsInstance {
	return &resourceanalyticsv1beta1.ResourceAnalyticsInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "rai", Namespace: "default"},
		Spec: resourceanalyticsv1beta1.ResourceAnalyticsInstanceSpec{
			CompartmentId: "compartment-1",
			AdwAdminPassword: resourceanalyticsv1beta1.ResourceAnalyticsInstanceAdwAdminPassword{
				PasswordType: string(resourceanalyticssdk.AdwAdminPasswordDetailsPasswordTypePlainText),
				Password:     "ValidPassword1",
			},
			SubnetId:            "subnet-1",
			DisplayName:         "resource-analytics",
			Description:         "analytics instance",
			IsMutualTlsRequired: true,
			NsgIds:              []string{"nsg-2", "nsg-1"},
			LicenseModel:        string(resourceanalyticssdk.CreateResourceAnalyticsInstanceDetailsLicenseModelLicenseIncluded),
			FreeformTags:        map[string]string{"env": "test"},
			DefinedTags:         map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func testResourceAnalyticsInstanceBody(
	id string,
	displayName string,
	state resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateEnum,
) resourceanalyticssdk.ResourceAnalyticsInstance {
	return resourceanalyticssdk.ResourceAnalyticsInstance{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String("compartment-1"),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "test"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		Description:    common.String("analytics instance"),
	}
}

func testResourceAnalyticsInstanceSummary(
	id string,
	displayName string,
	state resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateEnum,
) resourceanalyticssdk.ResourceAnalyticsInstanceSummary {
	return resourceanalyticssdk.ResourceAnalyticsInstanceSummary{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String("compartment-1"),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "test"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		Description:    common.String("analytics instance"),
	}
}

func testResourceAnalyticsInstanceWorkRequest(
	id string,
	operation resourceanalyticssdk.OperationTypeEnum,
	status resourceanalyticssdk.OperationStatusEnum,
	action resourceanalyticssdk.ActionTypeEnum,
	resourceID string,
) resourceanalyticssdk.WorkRequest {
	percent := float32(25)
	return resourceanalyticssdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operation,
		Status:          status,
		CompartmentId:   common.String("compartment-1"),
		PercentComplete: &percent,
		Resources: []resourceanalyticssdk.WorkRequestResource{
			{
				EntityType: common.String(resourceAnalyticsInstanceKind),
				ActionType: action,
				Identifier: common.String(resourceID),
			},
		},
	}
}

func requireResourceAnalyticsInstanceTrackedID(t *testing.T, resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance, want string) {
	t.Helper()
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func requireResourceAnalyticsInstanceCurrentWorkRequest(
	t *testing.T,
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
	wantPhase shared.OSOKAsyncPhase,
	wantClass shared.OSOKAsyncNormalizedClass,
	wantWorkRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, wantClass)
	}
	if current.WorkRequestID != wantWorkRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, wantWorkRequestID)
	}
}

func requireResourceAnalyticsInstanceStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", field, got, want)
	}
}

func requireResourceAnalyticsInstanceOpcRequestID(t *testing.T, resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func requireResourceAnalyticsInstanceNoCreateOnlyFingerprint(t *testing.T, resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance) {
	t.Helper()
	if got, ok := resourceAnalyticsInstanceRecordedCreateOnlyFingerprint(resource); ok {
		t.Fatalf("create-only fingerprint = %q, want none", got)
	}
}

func markResourceAnalyticsInstanceEstablished(resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance, id string) {
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	resource.Status.OsokStatus.CreatedAt = &metav1.Time{}
}

func requireResourceAnalyticsInstanceStringSliceEqual(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", field, got, want)
		}
	}
}

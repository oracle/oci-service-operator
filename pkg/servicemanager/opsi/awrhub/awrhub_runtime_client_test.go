/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package awrhub

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testAwrHubID          = "ocid1.awrhub.oc1..test"
	testAwrHubOtherID     = "ocid1.awrhub.oc1..other"
	testAwrWarehouseID    = "ocid1.opsiwarehouse.oc1..test"
	testAwrCompartmentID  = "ocid1.compartment.oc1..test"
	testAwrDisplayName    = "awr-hub"
	testAwrUpdatedName    = "awr-hub-updated"
	testAwrObjectBucket   = "awr-bucket"
	testAwrWorkRequestID  = "ocid1.workrequest.oc1..awrhub"
	testAwrOpcRequestID   = "opc-awrhub-request"
	testAwrObjectBucketV2 = "awr-bucket-v2"
)

type fakeAwrHubOCIClient struct {
	t *testing.T

	createFn         func(context.Context, opsisdk.CreateAwrHubRequest) (opsisdk.CreateAwrHubResponse, error)
	getFn            func(context.Context, opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error)
	listFn           func(context.Context, opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error)
	updateFn         func(context.Context, opsisdk.UpdateAwrHubRequest) (opsisdk.UpdateAwrHubResponse, error)
	deleteFn         func(context.Context, opsisdk.DeleteAwrHubRequest) (opsisdk.DeleteAwrHubResponse, error)
	getWorkRequestFn func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

func (f *fakeAwrHubOCIClient) CreateAwrHub(ctx context.Context, request opsisdk.CreateAwrHubRequest) (opsisdk.CreateAwrHubResponse, error) {
	f.t.Helper()
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	f.t.Fatalf("CreateAwrHub() was called unexpectedly with %#v", request)
	return opsisdk.CreateAwrHubResponse{}, nil
}

func (f *fakeAwrHubOCIClient) GetAwrHub(ctx context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
	f.t.Helper()
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	f.t.Fatalf("GetAwrHub() was called unexpectedly with %#v", request)
	return opsisdk.GetAwrHubResponse{}, nil
}

func (f *fakeAwrHubOCIClient) ListAwrHubs(ctx context.Context, request opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error) {
	f.t.Helper()
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	f.t.Fatalf("ListAwrHubs() was called unexpectedly with %#v", request)
	return opsisdk.ListAwrHubsResponse{}, nil
}

func (f *fakeAwrHubOCIClient) UpdateAwrHub(ctx context.Context, request opsisdk.UpdateAwrHubRequest) (opsisdk.UpdateAwrHubResponse, error) {
	f.t.Helper()
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	f.t.Fatalf("UpdateAwrHub() was called unexpectedly with %#v", request)
	return opsisdk.UpdateAwrHubResponse{}, nil
}

func (f *fakeAwrHubOCIClient) DeleteAwrHub(ctx context.Context, request opsisdk.DeleteAwrHubRequest) (opsisdk.DeleteAwrHubResponse, error) {
	f.t.Helper()
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	f.t.Fatalf("DeleteAwrHub() was called unexpectedly with %#v", request)
	return opsisdk.DeleteAwrHubResponse{}, nil
}

func (f *fakeAwrHubOCIClient) GetWorkRequest(ctx context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
	f.t.Helper()
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, request)
	}
	f.t.Fatalf("GetWorkRequest() was called unexpectedly with %#v", request)
	return opsisdk.GetWorkRequestResponse{}, nil
}

func TestAwrHubRuntimeHooksEncodeReviewedContract(t *testing.T) {
	t.Parallel()

	client := &fakeAwrHubOCIClient{t: t}
	hooks := newAwrHubRuntimeHooksWithOCIClient(client)
	applyAwrHubRuntimeHooks(&hooks, client, nil)

	t.Run("runtime hooks", func(t *testing.T) {
		t.Parallel()
		requireReviewedAwrHubRuntimeHooks(t, hooks)
	})
	t.Run("create body", func(t *testing.T) {
		t.Parallel()
		requireReviewedAwrHubCreateBody(t, hooks)
	})
}

func requireReviewedAwrHubRuntimeHooks(t *testing.T, hooks AwrHubRuntimeHooks) {
	t.Helper()

	requireReviewedAwrHubSemantics(t, hooks)
	requireReviewedAwrHubHookSeams(t, hooks)
}

func requireReviewedAwrHubSemantics(t *testing.T, hooks AwrHubRuntimeHooks) {
	t.Helper()

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed AwrHub semantics")
	}
	if got := hooks.Semantics.Async.Strategy; got != "workrequest" {
		t.Fatalf("hooks.Semantics.Async.Strategy = %q, want workrequest", got)
	}
}

func requireReviewedAwrHubHookSeams(t *testing.T, hooks AwrHubRuntimeHooks) {
	t.Helper()

	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want custom create body")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want custom update body")
	}
	if hooks.DeleteHooks.ConfirmRead == nil || hooks.DeleteHooks.ApplyOutcome == nil || hooks.DeleteHooks.HandleError == nil {
		t.Fatal("delete hooks are incomplete, want confirm read, outcome, and conservative error handling")
	}
	if hooks.Async.GetWorkRequest == nil || hooks.Async.ResolvePhase == nil || hooks.Async.RecoverResourceID == nil {
		t.Fatal("async hooks are incomplete, want work-request polling hooks")
	}
}

func requireReviewedAwrHubCreateBody(t *testing.T, hooks AwrHubRuntimeHooks) {
	t.Helper()

	resource := newTestAwrHubResource()
	resource.Spec.ObjectStorageBucketName = ""
	resource.Spec.FreeformTags = map[string]string{"env": "test"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}

	body, err := hooks.BuildCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("hooks.BuildCreateBody() error = %v", err)
	}
	createDetails, ok := body.(opsisdk.CreateAwrHubDetails)
	if !ok {
		t.Fatalf("hooks.BuildCreateBody() body type = %T, want opsisdk.CreateAwrHubDetails", body)
	}
	if createDetails.ObjectStorageBucketName != nil {
		t.Fatalf("ObjectStorageBucketName = %q, want nil when spec omits it", *createDetails.ObjectStorageBucketName)
	}
	if got := createDetails.FreeformTags["env"]; got != "test" {
		t.Fatalf("FreeformTags[env] = %q, want test", got)
	}
	if got := createDetails.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("DefinedTags[Operations][CostCenter] = %#v, want 42", got)
	}
}

func TestAwrHubServiceClientBindsFromSecondListPageWithoutCreating(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	var listRequests []opsisdk.ListAwrHubsRequest
	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		listFn: func(_ context.Context, request opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error) {
			listRequests = append(listRequests, request)
			if request.Page == nil {
				otherSpec := resource.Spec
				otherSpec.DisplayName = "other-awr-hub"
				return opsisdk.ListAwrHubsResponse{
					AwrHubSummaryCollection: opsisdk.AwrHubSummaryCollection{
						Items: []opsisdk.AwrHubSummary{
							makeSDKAwrHubSummary(testAwrHubOtherID, otherSpec, opsisdk.AwrHubLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			if *request.Page != "page-2" {
				t.Fatalf("ListAwrHubs() page = %q, want page-2", *request.Page)
			}
			return opsisdk.ListAwrHubsResponse{
				AwrHubSummaryCollection: opsisdk.AwrHubSummaryCollection{
					Items: []opsisdk.AwrHubSummary{
						makeSDKAwrHubSummary(testAwrHubID, resource.Spec, opsisdk.AwrHubLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
			requireStringPtr(t, "GetAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			return opsisdk.GetAwrHubResponse{
				AwrHub: makeSDKAwrHub(testAwrHubID, resource.Spec, opsisdk.AwrHubLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testAwrHubRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if got := len(listRequests); got != 2 {
		t.Fatalf("ListAwrHubs() calls = %d, want 2", got)
	}
	if resource.Status.Id != testAwrHubID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testAwrHubID)
	}
	if resource.Status.OsokStatus.Ocid != shared.OCID(testAwrHubID) {
		t.Fatalf("status.status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testAwrHubID)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after active bind", resource.Status.OsokStatus.Async.Current)
	}
}

func TestAwrHubServiceClientCompletesCreateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	var createRequest opsisdk.CreateAwrHubRequest
	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		listFn: func(_ context.Context, request opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error) {
			requireStringPtr(t, "ListAwrHubsRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testAwrWarehouseID)
			requireStringPtr(t, "ListAwrHubsRequest.CompartmentId", request.CompartmentId, testAwrCompartmentID)
			requireStringPtr(t, "ListAwrHubsRequest.DisplayName", request.DisplayName, testAwrDisplayName)
			return opsisdk.ListAwrHubsResponse{}, nil
		},
		createFn: func(_ context.Context, request opsisdk.CreateAwrHubRequest) (opsisdk.CreateAwrHubResponse, error) {
			createRequest = request
			return opsisdk.CreateAwrHubResponse{
				OpcWorkRequestId: common.String(testAwrWorkRequestID),
				OpcRequestId:     common.String(testAwrOpcRequestID),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testAwrWorkRequestID)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeAwrHubWorkRequest(
					testAwrWorkRequestID,
					opsisdk.OperationStatusSucceeded,
					opsisdk.OperationTypeCreateAwrhub,
					opsisdk.ActionTypeCreated,
					testAwrHubID,
				),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
			requireStringPtr(t, "GetAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			return opsisdk.GetAwrHubResponse{
				AwrHub: makeSDKAwrHub(testAwrHubID, resource.Spec, opsisdk.AwrHubLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testAwrHubRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	requireStringPtr(t, "CreateAwrHubRequest.OperationsInsightsWarehouseId", createRequest.OperationsInsightsWarehouseId, testAwrWarehouseID)
	requireStringPtr(t, "CreateAwrHubRequest.CompartmentId", createRequest.CompartmentId, testAwrCompartmentID)
	requireStringPtr(t, "CreateAwrHubRequest.DisplayName", createRequest.DisplayName, testAwrDisplayName)
	if resource.Status.Id != testAwrHubID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testAwrHubID)
	}
	if resource.Status.OsokStatus.Ocid != shared.OCID(testAwrHubID) {
		t.Fatalf("status.status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testAwrHubID)
	}
	if resource.Status.OsokStatus.OpcRequestID != testAwrOpcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, testAwrOpcRequestID)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after succeeded create work request", resource.Status.OsokStatus.Async.Current)
	}
}

func TestAwrHubServiceClientUpdatesMutableFieldsThroughWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	resource.Status.Id = testAwrHubID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAwrHubID)
	resource.Spec.DisplayName = testAwrUpdatedName
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	getCalls := 0
	var updateRequest opsisdk.UpdateAwrHubRequest
	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		getFn: func(_ context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
			requireStringPtr(t, "GetAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			getCalls++
			currentSpec := resource.Spec
			if getCalls == 1 {
				currentSpec.DisplayName = testAwrDisplayName
				currentSpec.FreeformTags = map[string]string{"env": "dev"}
				currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
			}
			return opsisdk.GetAwrHubResponse{
				AwrHub: makeSDKAwrHub(testAwrHubID, currentSpec, opsisdk.AwrHubLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, request opsisdk.UpdateAwrHubRequest) (opsisdk.UpdateAwrHubResponse, error) {
			updateRequest = request
			return opsisdk.UpdateAwrHubResponse{
				OpcWorkRequestId: common.String(testAwrWorkRequestID),
				OpcRequestId:     common.String(testAwrOpcRequestID),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testAwrWorkRequestID)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeAwrHubWorkRequest(
					testAwrWorkRequestID,
					opsisdk.OperationStatusSucceeded,
					opsisdk.OperationTypeUpdateAwrhub,
					opsisdk.ActionTypeUpdated,
					testAwrHubID,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testAwrHubRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	requireStringPtr(t, "UpdateAwrHubRequest.AwrHubId", updateRequest.AwrHubId, testAwrHubID)
	requireStringPtr(t, "UpdateAwrHubRequest.DisplayName", updateRequest.DisplayName, testAwrUpdatedName)
	if got := updateRequest.FreeformTags["env"]; got != "prod" {
		t.Fatalf("UpdateAwrHubRequest.FreeformTags[env] = %q, want prod", got)
	}
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("UpdateAwrHubRequest.DefinedTags[Operations][CostCenter] = %#v, want 84", got)
	}
	if resource.Status.DisplayName != testAwrUpdatedName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, testAwrUpdatedName)
	}
	if resource.Status.OsokStatus.OpcRequestID != testAwrOpcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, testAwrOpcRequestID)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after succeeded update work request", resource.Status.OsokStatus.Async.Current)
	}
}

func TestAwrHubServiceClientNoOpReconcileDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	resource.Status.Id = testAwrHubID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAwrHubID)
	getCalls := 0
	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		getFn: func(_ context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
			getCalls++
			requireStringPtr(t, "GetAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			return opsisdk.GetAwrHubResponse{
				AwrHub: makeSDKAwrHub(testAwrHubID, resource.Spec, opsisdk.AwrHubLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testAwrHubRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if getCalls != 1 {
		t.Fatalf("GetAwrHub() calls = %d, want 1", getCalls)
	}
	if resource.Status.DisplayName != testAwrDisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, testAwrDisplayName)
	}
}

func TestAwrHubServiceClientRejectsObjectStorageBucketDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	resource.Status.Id = testAwrHubID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAwrHubID)
	resource.Spec.ObjectStorageBucketName = testAwrObjectBucketV2

	currentSpec := resource.Spec
	currentSpec.ObjectStorageBucketName = testAwrObjectBucket
	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		getFn: func(_ context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
			requireStringPtr(t, "GetAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			return opsisdk.GetAwrHubResponse{
				AwrHub: makeSDKAwrHub(testAwrHubID, currentSpec, opsisdk.AwrHubLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testAwrHubRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "objectStorageBucketName") {
		t.Fatalf("CreateOrUpdate() error = %q, want objectStorageBucketName drift", err.Error())
	}
}

func TestAwrHubDeleteCompletesAfterSucceededWorkRequestAndConfirmedNotFound(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	resource.Status.Id = testAwrHubID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAwrHubID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   testAwrWorkRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testAwrWorkRequestID)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeAwrHubWorkRequest(
					testAwrWorkRequestID,
					opsisdk.OperationStatusSucceeded,
					opsisdk.OperationTypeDeleteAwrhub,
					opsisdk.ActionTypeDeleted,
					testAwrHubID,
				),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
			requireStringPtr(t, "GetAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			return opsisdk.GetAwrHubResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotFound,
				"not found",
			)
		},
		listFn: func(_ context.Context, request opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error) {
			requireStringPtr(t, "ListAwrHubsRequest.Id", request.Id, testAwrHubID)
			return opsisdk.ListAwrHubsResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatalf("Delete() deleted = false, want true after succeeded work request and confirmed not-found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after confirmed delete", resource.Status.OsokStatus.Async.Current)
	}
}

func TestAwrHubDeleteWithoutTrackedIDCompletesWhenListFindsNoMatch(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	listCalls := 0
	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		listFn: func(_ context.Context, request opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error) {
			listCalls++
			requireStringPtr(t, "ListAwrHubsRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testAwrWarehouseID)
			requireStringPtr(t, "ListAwrHubsRequest.CompartmentId", request.CompartmentId, testAwrCompartmentID)
			requireStringPtr(t, "ListAwrHubsRequest.DisplayName", request.DisplayName, testAwrDisplayName)
			if request.Id != nil {
				t.Fatalf("ListAwrHubsRequest.Id = %q, want nil without tracked identity", *request.Id)
			}
			return opsisdk.ListAwrHubsResponse{
				OpcRequestId: common.String(testAwrOpcRequestID),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after list confirmation finds no AwrHub")
	}
	if listCalls != 1 {
		t.Fatalf("ListAwrHubs() calls = %d, want 1", listCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
	if resource.Status.OsokStatus.OpcRequestID != testAwrOpcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, testAwrOpcRequestID)
	}
}

func TestAwrHubDeleteWithoutTrackedIDRejectsAuthShapedListConfirmation(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	serviceErr := errortest.NewServiceError(
		404,
		errorutil.NotAuthorizedOrNotFound,
		"not authorized or not found",
	)
	serviceErr.OpcRequestID = testAwrOpcRequestID

	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		listFn: func(_ context.Context, request opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error) {
			requireStringPtr(t, "ListAwrHubsRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testAwrWarehouseID)
			return opsisdk.ListAwrHubsResponse{}, serviceErr
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous list confirmation rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous NotAuthorizedOrNotFound", err.Error())
	}
	if resource.Status.OsokStatus.OpcRequestID != testAwrOpcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, testAwrOpcRequestID)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %#v, want nil after ambiguous list confirmation", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestAwrHubServiceClientDeleteStartsAndPollsWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	resource.Status.Id = testAwrHubID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAwrHubID)

	deleteCalls := 0
	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		getFn: func(_ context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
			requireStringPtr(t, "GetAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			return opsisdk.GetAwrHubResponse{
				AwrHub: makeSDKAwrHub(testAwrHubID, resource.Spec, opsisdk.AwrHubLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request opsisdk.DeleteAwrHubRequest) (opsisdk.DeleteAwrHubResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			return opsisdk.DeleteAwrHubResponse{
				OpcWorkRequestId: common.String(testAwrWorkRequestID),
				OpcRequestId:     common.String(testAwrOpcRequestID),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testAwrWorkRequestID)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeAwrHubWorkRequest(
					testAwrWorkRequestID,
					opsisdk.OperationStatusInProgress,
					opsisdk.OperationTypeDeleteAwrhub,
					opsisdk.ActionTypeInProgress,
					testAwrHubID,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false while delete work request is pending")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAwrHub() calls = %d, want 1", deleteCalls)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want pending delete work request")
	}
	if current.WorkRequestID != testAwrWorkRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, testAwrWorkRequestID)
	}
	if current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current.phase = %q, want delete", current.Phase)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
}

func TestAwrHubDeleteRetainsFinalizerForPendingCreateWorkRequest(t *testing.T) {
	t.Parallel()

	requireAwrHubDeleteRetainsFinalizerForPendingWriteWorkRequest(
		t,
		shared.OSOKAsyncPhaseCreate,
		opsisdk.OperationTypeCreateAwrhub,
	)
}

func TestAwrHubDeleteRetainsFinalizerForPendingUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	requireAwrHubDeleteRetainsFinalizerForPendingWriteWorkRequest(
		t,
		shared.OSOKAsyncPhaseUpdate,
		opsisdk.OperationTypeUpdateAwrhub,
	)
}

func requireAwrHubDeleteRetainsFinalizerForPendingWriteWorkRequest(
	t *testing.T,
	phase shared.OSOKAsyncPhase,
	operationType opsisdk.OperationTypeEnum,
) {
	t.Helper()

	resource := newTestAwrHubResource()
	resource.Status.Id = testAwrHubID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAwrHubID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   testAwrWorkRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testAwrWorkRequestID)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeAwrHubWorkRequest(
					testAwrWorkRequestID,
					opsisdk.OperationStatusInProgress,
					operationType,
					opsisdk.ActionTypeInProgress,
					testAwrHubID,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false while %s work request is pending", phase)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want pending write work request retained")
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != testAwrWorkRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, testAwrWorkRequestID)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
	if !strings.Contains(current.Message, "waiting before delete") {
		t.Fatalf("status.async.current.message = %q, want waiting-before-delete detail", current.Message)
	}
}

func TestAwrHubDeleteRetainsFinalizerForSucceededCreateWorkRequestWithoutReadback(t *testing.T) {
	t.Parallel()

	requireAwrHubDeleteRetainsFinalizerForSucceededWriteWorkRequestWithoutReadback(
		t,
		shared.OSOKAsyncPhaseCreate,
		opsisdk.OperationTypeCreateAwrhub,
		opsisdk.ActionTypeCreated,
	)
}

func TestAwrHubDeleteRetainsFinalizerForSucceededUpdateWorkRequestWithoutReadback(t *testing.T) {
	t.Parallel()

	requireAwrHubDeleteRetainsFinalizerForSucceededWriteWorkRequestWithoutReadback(
		t,
		shared.OSOKAsyncPhaseUpdate,
		opsisdk.OperationTypeUpdateAwrhub,
		opsisdk.ActionTypeUpdated,
	)
}

func requireAwrHubDeleteRetainsFinalizerForSucceededWriteWorkRequestWithoutReadback(
	t *testing.T,
	phase shared.OSOKAsyncPhase,
	operationType opsisdk.OperationTypeEnum,
	action opsisdk.ActionTypeEnum,
) {
	t.Helper()

	resource := newTestAwrHubResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   testAwrWorkRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testAwrWorkRequestID)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeAwrHubWorkRequest(
					testAwrWorkRequestID,
					opsisdk.OperationStatusSucceeded,
					operationType,
					action,
					testAwrHubID,
				),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
			requireStringPtr(t, "GetAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			return opsisdk.GetAwrHubResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotFound,
				"not found",
			)
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false while %s readback is unresolved", phase)
	}
	if resource.Status.Id != testAwrHubID {
		t.Fatalf("status.id = %q, want recovered AwrHub id %q", resource.Status.Id, testAwrHubID)
	}
	if resource.Status.OsokStatus.Ocid != shared.OCID(testAwrHubID) {
		t.Fatalf("status.status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testAwrHubID)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want unresolved write work request retained")
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
	if !strings.Contains(current.Message, "waiting for AwrHub") {
		t.Fatalf("status.async.current.message = %q, want readback-wait detail", current.Message)
	}
}

func TestAwrHubDeleteRejectsAuthShapedNotFoundAfterSucceededDeleteWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	resource.Status.Id = testAwrHubID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAwrHubID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   testAwrWorkRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	serviceErr := errortest.NewServiceError(
		404,
		errorutil.NotAuthorizedOrNotFound,
		"not authorized or not found",
	)
	serviceErr.OpcRequestID = testAwrOpcRequestID

	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testAwrWorkRequestID)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeAwrHubWorkRequest(
					testAwrWorkRequestID,
					opsisdk.OperationStatusSucceeded,
					opsisdk.OperationTypeDeleteAwrhub,
					opsisdk.ActionTypeDeleted,
					testAwrHubID,
				),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
			requireStringPtr(t, "GetAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			return opsisdk.GetAwrHubResponse{}, serviceErr
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous delete confirmation rejection")
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous NotAuthorizedOrNotFound", err.Error())
	}
	if resource.Status.OsokStatus.OpcRequestID != testAwrOpcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, testAwrOpcRequestID)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %#v, want nil after ambiguous delete confirmation", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestAwrHubDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	t.Parallel()

	resource := newTestAwrHubResource()
	resource.Status.Id = testAwrHubID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAwrHubID)

	client := newTestAwrHubServiceClient(t, &fakeAwrHubOCIClient{
		t: t,
		getFn: func(_ context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
			requireStringPtr(t, "GetAwrHubRequest.AwrHubId", request.AwrHubId, testAwrHubID)
			return opsisdk.GetAwrHubResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"not authorized or not found",
			)
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound rejection")
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous NotAuthorizedOrNotFound", err.Error())
	}
}

func newTestAwrHubServiceClient(t *testing.T, client awrHubOCIClient) AwrHubServiceClient {
	t.Helper()
	hooks := newAwrHubRuntimeHooksWithOCIClient(client)
	applyAwrHubRuntimeHooks(&hooks, client, nil)
	delegate := defaultAwrHubServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.AwrHub](
			buildAwrHubGeneratedRuntimeConfig(&AwrHubServiceManager{}, hooks),
		),
	}
	return wrapAwrHubGeneratedClient(hooks, delegate)
}

func newTestAwrHubResource() *opsiv1beta1.AwrHub {
	return &opsiv1beta1.AwrHub{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "awrhub-sample",
			Namespace: "default",
			UID:       k8stypes.UID("uid-awrhub"),
		},
		Spec: opsiv1beta1.AwrHubSpec{
			OperationsInsightsWarehouseId: testAwrWarehouseID,
			CompartmentId:                 testAwrCompartmentID,
			DisplayName:                   testAwrDisplayName,
			ObjectStorageBucketName:       testAwrObjectBucket,
		},
	}
}

func testAwrHubRequest() ctrl.Request {
	return ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{
			Namespace: "default",
			Name:      "awrhub-sample",
		},
	}
}

func makeSDKAwrHub(
	id string,
	spec opsiv1beta1.AwrHubSpec,
	state opsisdk.AwrHubLifecycleStateEnum,
) opsisdk.AwrHub {
	return opsisdk.AwrHub{
		OperationsInsightsWarehouseId: common.String(spec.OperationsInsightsWarehouseId),
		Id:                            common.String(id),
		CompartmentId:                 common.String(spec.CompartmentId),
		DisplayName:                   common.String(spec.DisplayName),
		ObjectStorageBucketName:       common.String(spec.ObjectStorageBucketName),
		LifecycleState:                state,
		AwrMailboxUrl:                 common.String("https://mailbox.example.invalid"),
		FreeformTags:                  cloneAwrHubStringMap(spec.FreeformTags),
		DefinedTags:                   awrHubDefinedTagsFromSpec(spec.DefinedTags),
		SystemTags:                    map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "false"}},
		LifecycleDetails:              common.String("ready"),
		HubDstTimezoneVersion:         common.String("42"),
	}
}

func makeSDKAwrHubSummary(
	id string,
	spec opsiv1beta1.AwrHubSpec,
	state opsisdk.AwrHubLifecycleStateEnum,
) opsisdk.AwrHubSummary {
	return opsisdk.AwrHubSummary{
		OperationsInsightsWarehouseId: common.String(spec.OperationsInsightsWarehouseId),
		Id:                            common.String(id),
		CompartmentId:                 common.String(spec.CompartmentId),
		DisplayName:                   common.String(spec.DisplayName),
		ObjectStorageBucketName:       common.String(spec.ObjectStorageBucketName),
		LifecycleState:                state,
		FreeformTags:                  cloneAwrHubStringMap(spec.FreeformTags),
		DefinedTags:                   awrHubDefinedTagsFromSpec(spec.DefinedTags),
		LifecycleDetails:              common.String("ready"),
	}
}

func makeAwrHubWorkRequest(
	id string,
	status opsisdk.OperationStatusEnum,
	operationType opsisdk.OperationTypeEnum,
	action opsisdk.ActionTypeEnum,
	resourceID string,
) opsisdk.WorkRequest {
	percent := float32(50)
	return opsisdk.WorkRequest{
		Id:              common.String(id),
		CompartmentId:   common.String(testAwrCompartmentID),
		Status:          status,
		OperationType:   operationType,
		PercentComplete: &percent,
		Resources: []opsisdk.WorkRequestResource{
			{
				EntityType: common.String("AwrHub"),
				ActionType: action,
				Identifier: common.String(resourceID),
			},
		},
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

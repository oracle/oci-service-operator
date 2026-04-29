/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package kafkacluster

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	managedkafkasdk "github.com/oracle/oci-go-sdk/v65/managedkafka"
	managedkafkav1beta1 "github.com/oracle/oci-service-operator/api/managedkafka/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeKafkaClusterOCIClient struct {
	createFn         func(context.Context, managedkafkasdk.CreateKafkaClusterRequest) (managedkafkasdk.CreateKafkaClusterResponse, error)
	getFn            func(context.Context, managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error)
	listFn           func(context.Context, managedkafkasdk.ListKafkaClustersRequest) (managedkafkasdk.ListKafkaClustersResponse, error)
	updateFn         func(context.Context, managedkafkasdk.UpdateKafkaClusterRequest) (managedkafkasdk.UpdateKafkaClusterResponse, error)
	deleteFn         func(context.Context, managedkafkasdk.DeleteKafkaClusterRequest) (managedkafkasdk.DeleteKafkaClusterResponse, error)
	getWorkRequestFn func(context.Context, managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error)

	createRequests         []managedkafkasdk.CreateKafkaClusterRequest
	getRequests            []managedkafkasdk.GetKafkaClusterRequest
	listRequests           []managedkafkasdk.ListKafkaClustersRequest
	updateRequests         []managedkafkasdk.UpdateKafkaClusterRequest
	deleteRequests         []managedkafkasdk.DeleteKafkaClusterRequest
	getWorkRequestRequests []managedkafkasdk.GetWorkRequestRequest
}

func (f *fakeKafkaClusterOCIClient) CreateKafkaCluster(ctx context.Context, req managedkafkasdk.CreateKafkaClusterRequest) (managedkafkasdk.CreateKafkaClusterResponse, error) {
	f.createRequests = append(f.createRequests, req)
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return managedkafkasdk.CreateKafkaClusterResponse{}, nil
}

func (f *fakeKafkaClusterOCIClient) GetKafkaCluster(ctx context.Context, req managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return managedkafkasdk.GetKafkaClusterResponse{}, errortest.NewServiceError(404, "NotFound", "")
}

func (f *fakeKafkaClusterOCIClient) ListKafkaClusters(ctx context.Context, req managedkafkasdk.ListKafkaClustersRequest) (managedkafkasdk.ListKafkaClustersResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return managedkafkasdk.ListKafkaClustersResponse{}, nil
}

func (f *fakeKafkaClusterOCIClient) UpdateKafkaCluster(ctx context.Context, req managedkafkasdk.UpdateKafkaClusterRequest) (managedkafkasdk.UpdateKafkaClusterResponse, error) {
	f.updateRequests = append(f.updateRequests, req)
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return managedkafkasdk.UpdateKafkaClusterResponse{}, nil
}

func (f *fakeKafkaClusterOCIClient) DeleteKafkaCluster(ctx context.Context, req managedkafkasdk.DeleteKafkaClusterRequest) (managedkafkasdk.DeleteKafkaClusterResponse, error) {
	f.deleteRequests = append(f.deleteRequests, req)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return managedkafkasdk.DeleteKafkaClusterResponse{}, nil
}

func (f *fakeKafkaClusterOCIClient) GetWorkRequest(ctx context.Context, req managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
	f.getWorkRequestRequests = append(f.getWorkRequestRequests, req)
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return managedkafkasdk.GetWorkRequestResponse{}, nil
}

func TestKafkaClusterCreateStartsWorkRequest(t *testing.T) {
	resource := newTestKafkaCluster()
	fake := &fakeKafkaClusterOCIClient{}
	fake.createFn = func(_ context.Context, req managedkafkasdk.CreateKafkaClusterRequest) (managedkafkasdk.CreateKafkaClusterResponse, error) {
		assertKafkaClusterCreateRequest(t, req, resource)
		return managedkafkasdk.CreateKafkaClusterResponse{
			KafkaCluster:     makeSDKKafkaCluster(resource, "ocid1.kafkacluster.oc1..created", managedkafkasdk.KafkaClusterLifecycleStateCreating),
			OpcRequestId:     common.String("opc-create-1"),
			OpcWorkRequestId: common.String("wr-create-1"),
		}, nil
	}
	fake.getWorkRequestFn = func(_ context.Context, req managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
		assertKafkaClusterWorkRequestRequest(t, req, "wr-create-1")
		return managedkafkasdk.GetWorkRequestResponse{
			WorkRequest: makeKafkaClusterWorkRequest(
				"wr-create-1",
				managedkafkasdk.OperationTypeCreateKafkaCluster,
				managedkafkasdk.OperationStatusInProgress,
				managedkafkasdk.ActionTypeCreated,
				"ocid1.kafkacluster.oc1..created",
			),
		}, nil
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, testKafkaClusterRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful requeue", response)
	}
	if got, want := len(fake.listRequests), 1; got != want {
		t.Fatalf("list calls = %d, want %d", got, want)
	}
	if got, want := len(fake.createRequests), 1; got != want {
		t.Fatalf("create calls = %d, want %d", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create-1"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	requireCurrentKafkaClusterWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1")
}

func TestKafkaClusterPendingCreateWorkRequestFallsBackToOperationTypeWhenActionIsInProgress(t *testing.T) {
	resource := newTestKafkaCluster()
	fake := &fakeKafkaClusterOCIClient{}
	fake.createFn = func(_ context.Context, _ managedkafkasdk.CreateKafkaClusterRequest) (managedkafkasdk.CreateKafkaClusterResponse, error) {
		return managedkafkasdk.CreateKafkaClusterResponse{
			KafkaCluster:     makeSDKKafkaCluster(resource, "ocid1.kafkacluster.oc1..created", managedkafkasdk.KafkaClusterLifecycleStateCreating),
			OpcRequestId:     common.String("opc-create-pending"),
			OpcWorkRequestId: common.String("wr-create-pending"),
		}, nil
	}
	fake.getWorkRequestFn = func(_ context.Context, _ managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
		workRequest := makeKafkaClusterWorkRequest(
			"wr-create-pending",
			managedkafkasdk.OperationTypeCreateKafkaCluster,
			managedkafkasdk.OperationStatusInProgress,
			managedkafkasdk.ActionTypeInProgress,
			"ocid1.kafkacluster.oc1..created",
		)
		workRequest.Resources = append(workRequest.Resources, managedkafkasdk.WorkRequestResource{
			EntityType: common.String(kafkaClusterWorkRequestEntityType),
			ActionType: managedkafkasdk.ActionTypeRelated,
			Identifier: common.String("ocid1.kafkacluster.oc1..related"),
		})
		return managedkafkasdk.GetWorkRequestResponse{WorkRequest: workRequest}, nil
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, testKafkaClusterRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful requeue", response)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want pending create work request")
	}
	if current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, shared.OSOKAsyncPhaseCreate)
	}
	if current.RawOperationType != string(managedkafkasdk.OperationTypeCreateKafkaCluster) {
		t.Fatalf("status.async.current.rawOperationType = %q, want %q", current.RawOperationType, managedkafkasdk.OperationTypeCreateKafkaCluster)
	}
}

func TestKafkaClusterBindUsesPagedListMatch(t *testing.T) {
	resource := newTestKafkaCluster()
	pages := []string{}
	createCalled := false
	updateCalled := false
	getCalls := 0

	fake := &fakeKafkaClusterOCIClient{}
	fake.listFn = func(_ context.Context, req managedkafkasdk.ListKafkaClustersRequest) (managedkafkasdk.ListKafkaClustersResponse, error) {
		pages = append(pages, stringValue(req.Page))
		assertKafkaClusterListRequest(t, req, resource)
		return kafkaClusterPagedListResponse(resource, req), nil
	}
	fake.getFn = func(_ context.Context, req managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		getCalls++
		assertKafkaClusterGetRequest(t, req, "ocid1.kafkacluster.oc1..existing")
		return managedkafkasdk.GetKafkaClusterResponse{
			KafkaCluster: makeSDKKafkaCluster(resource, "ocid1.kafkacluster.oc1..existing", managedkafkasdk.KafkaClusterLifecycleStateActive),
		}, nil
	}
	fake.createFn = func(context.Context, managedkafkasdk.CreateKafkaClusterRequest) (managedkafkasdk.CreateKafkaClusterResponse, error) {
		createCalled = true
		return managedkafkasdk.CreateKafkaClusterResponse{}, nil
	}
	fake.updateFn = func(context.Context, managedkafkasdk.UpdateKafkaClusterRequest) (managedkafkasdk.UpdateKafkaClusterResponse, error) {
		updateCalled = true
		return managedkafkasdk.UpdateKafkaClusterResponse{}, nil
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, testKafkaClusterRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertKafkaClusterBindResult(t, response, resource, pages, getCalls, createCalled, updateCalled)
}

func TestKafkaClusterDuplicatePagedListMatchesAreRejected(t *testing.T) {
	resource := newTestKafkaCluster()
	createCalled := false

	fake := &fakeKafkaClusterOCIClient{}
	fake.listFn = func(_ context.Context, req managedkafkasdk.ListKafkaClustersRequest) (managedkafkasdk.ListKafkaClustersResponse, error) {
		if req.Page == nil {
			return managedkafkasdk.ListKafkaClustersResponse{
				KafkaClusterCollection: managedkafkasdk.KafkaClusterCollection{Items: []managedkafkasdk.KafkaClusterSummary{
					makeSDKKafkaClusterSummary(resource, "ocid1.kafkacluster.oc1..first", managedkafkasdk.KafkaClusterLifecycleStateActive),
				}},
				OpcNextPage: common.String("page-2"),
			}, nil
		}
		return managedkafkasdk.ListKafkaClustersResponse{
			KafkaClusterCollection: managedkafkasdk.KafkaClusterCollection{Items: []managedkafkasdk.KafkaClusterSummary{
				makeSDKKafkaClusterSummary(resource, "ocid1.kafkacluster.oc1..second", managedkafkasdk.KafkaClusterLifecycleStateActive),
			}},
		}, nil
	}
	fake.createFn = func(context.Context, managedkafkasdk.CreateKafkaClusterRequest) (managedkafkasdk.CreateKafkaClusterResponse, error) {
		createCalled = true
		return managedkafkasdk.CreateKafkaClusterResponse{}, nil
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, testKafkaClusterRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want duplicate match error")
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match detail", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful", response)
	}
	if createCalled {
		t.Fatal("CreateKafkaCluster() was called after duplicate list matches")
	}
}

func TestKafkaClusterNoopObserveDoesNotUpdateOCI(t *testing.T) {
	resource := newTestKafkaCluster()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.kafkacluster.oc1..existing")
	current := makeSDKKafkaCluster(resource, "ocid1.kafkacluster.oc1..existing", managedkafkasdk.KafkaClusterLifecycleStateActive)
	current.BrokerShape.StorageSizeInGbs = common.Int(50)

	fake := &fakeKafkaClusterOCIClient{}
	fake.getFn = func(_ context.Context, _ managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		return managedkafkasdk.GetKafkaClusterResponse{KafkaCluster: current}, nil
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, testKafkaClusterRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful non-requeue", response)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update calls = %d, want 0", got)
	}
	if got, want := resource.Status.LifecycleState, string(managedkafkasdk.KafkaClusterLifecycleStateActive); got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
}

func TestKafkaClusterMutableUpdateStartsWorkRequest(t *testing.T) {
	resource := newTestKafkaCluster()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.kafkacluster.oc1..existing")
	current := makeSDKKafkaCluster(resource, "ocid1.kafkacluster.oc1..existing", managedkafkasdk.KafkaClusterLifecycleStateActive)
	current.DisplayName = common.String("old-kafka-cluster")

	fake := &fakeKafkaClusterOCIClient{}
	fake.getFn = func(_ context.Context, _ managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		return managedkafkasdk.GetKafkaClusterResponse{KafkaCluster: current}, nil
	}
	fake.updateFn = func(_ context.Context, req managedkafkasdk.UpdateKafkaClusterRequest) (managedkafkasdk.UpdateKafkaClusterResponse, error) {
		if got, want := stringValue(req.KafkaClusterId), "ocid1.kafkacluster.oc1..existing"; got != want {
			t.Fatalf("update kafkaClusterId = %q, want %q", got, want)
		}
		if got, want := stringValue(req.DisplayName), resource.Spec.DisplayName; got != want {
			t.Fatalf("update displayName = %q, want %q", got, want)
		}
		updated := makeSDKKafkaCluster(resource, "ocid1.kafkacluster.oc1..existing", managedkafkasdk.KafkaClusterLifecycleStateUpdating)
		return managedkafkasdk.UpdateKafkaClusterResponse{
			KafkaCluster:     updated,
			OpcRequestId:     common.String("opc-update-1"),
			OpcWorkRequestId: common.String("wr-update-1"),
		}, nil
	}
	fake.getWorkRequestFn = func(_ context.Context, _ managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
		return managedkafkasdk.GetWorkRequestResponse{
			WorkRequest: makeKafkaClusterWorkRequest(
				"wr-update-1",
				managedkafkasdk.OperationTypeUpdateKafkaCluster,
				managedkafkasdk.OperationStatusInProgress,
				managedkafkasdk.ActionTypeUpdated,
				"ocid1.kafkacluster.oc1..existing",
			),
		}, nil
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, testKafkaClusterRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful requeue", response)
	}
	if got, want := len(fake.updateRequests), 1; got != want {
		t.Fatalf("update calls = %d, want %d", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-update-1"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	requireCurrentKafkaClusterWorkRequest(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-1")
}

func TestKafkaClusterDeleteWaitsForPendingWriteLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name  string
		phase shared.OSOKAsyncPhase
		state managedkafkasdk.KafkaClusterLifecycleStateEnum
	}{
		{
			name:  "create",
			phase: shared.OSOKAsyncPhaseCreate,
			state: managedkafkasdk.KafkaClusterLifecycleStateCreating,
		},
		{
			name:  "update",
			phase: shared.OSOKAsyncPhaseUpdate,
			state: managedkafkasdk.KafkaClusterLifecycleStateUpdating,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := newTestKafkaCluster()
			resourceID := "ocid1.kafkacluster.oc1.." + tc.name
			resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
			resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
				Source:          shared.OSOKAsyncSourceLifecycle,
				Phase:           tc.phase,
				RawStatus:       string(tc.state),
				NormalizedClass: shared.OSOKAsyncClassPending,
			}

			fake := &fakeKafkaClusterOCIClient{}
			fake.getFn = func(_ context.Context, req managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
				assertKafkaClusterGetRequest(t, req, resourceID)
				return managedkafkasdk.GetKafkaClusterResponse{
					KafkaCluster: makeSDKKafkaCluster(resource, resourceID, tc.state),
				}, nil
			}

			client := newTestKafkaClusterGeneratedClient(fake)
			deleted, err := client.Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want false while lifecycle write is pending")
			}
			if got := len(fake.deleteRequests); got != 0 {
				t.Fatalf("delete calls = %d, want 0 while lifecycle write is pending", got)
			}
			requireCurrentKafkaClusterLifecycle(t, resource, tc.phase, string(tc.state))
		})
	}
}

func TestKafkaClusterDeleteWaitsForPendingCreateWorkRequest(t *testing.T) {
	resource := newTestKafkaCluster()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create-pending",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	fake := &fakeKafkaClusterOCIClient{}
	fake.getWorkRequestFn = func(_ context.Context, req managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
		assertKafkaClusterWorkRequestRequest(t, req, "wr-create-pending")
		return managedkafkasdk.GetWorkRequestResponse{
			WorkRequest: makeKafkaClusterWorkRequest(
				"wr-create-pending",
				managedkafkasdk.OperationTypeCreateKafkaCluster,
				managedkafkasdk.OperationStatusInProgress,
				managedkafkasdk.ActionTypeCreated,
				"ocid1.kafkacluster.oc1..creating",
			),
		}, nil
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while create work request is pending")
	}
	if got := len(fake.deleteRequests); got != 0 {
		t.Fatalf("delete calls = %d, want 0 while create work request is pending", got)
	}
	requireCurrentKafkaClusterWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-pending")
}

func TestKafkaClusterDeleteWaitsWhenSucceededWriteWorkRequestReadbackIsPending(t *testing.T) {
	for _, tc := range []struct {
		name   string
		phase  shared.OSOKAsyncPhase
		op     managedkafkasdk.OperationTypeEnum
		action managedkafkasdk.ActionTypeEnum
		state  managedkafkasdk.KafkaClusterLifecycleStateEnum
	}{
		{
			name:   "create",
			phase:  shared.OSOKAsyncPhaseCreate,
			op:     managedkafkasdk.OperationTypeCreateKafkaCluster,
			action: managedkafkasdk.ActionTypeCreated,
			state:  managedkafkasdk.KafkaClusterLifecycleStateCreating,
		},
		{
			name:   "update",
			phase:  shared.OSOKAsyncPhaseUpdate,
			op:     managedkafkasdk.OperationTypeUpdateKafkaCluster,
			action: managedkafkasdk.ActionTypeUpdated,
			state:  managedkafkasdk.KafkaClusterLifecycleStateUpdating,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := newTestKafkaCluster()
			resourceID := "ocid1.kafkacluster.oc1.." + tc.name
			workRequestID := "wr-" + tc.name + "-succeeded"
			resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
			resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
				Source:          shared.OSOKAsyncSourceWorkRequest,
				Phase:           tc.phase,
				WorkRequestID:   workRequestID,
				NormalizedClass: shared.OSOKAsyncClassPending,
			}

			fake := &fakeKafkaClusterOCIClient{}
			fake.getWorkRequestFn = func(_ context.Context, req managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
				assertKafkaClusterWorkRequestRequest(t, req, workRequestID)
				return managedkafkasdk.GetWorkRequestResponse{
					WorkRequest: makeKafkaClusterWorkRequest(
						workRequestID,
						tc.op,
						managedkafkasdk.OperationStatusSucceeded,
						tc.action,
						resourceID,
					),
				}, nil
			}
			fake.getFn = func(_ context.Context, req managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
				assertKafkaClusterGetRequest(t, req, resourceID)
				return managedkafkasdk.GetKafkaClusterResponse{
					KafkaCluster: makeSDKKafkaCluster(resource, resourceID, tc.state),
				}, nil
			}

			client := newTestKafkaClusterGeneratedClient(fake)
			deleted, err := client.Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want false while readback lifecycle is pending")
			}
			if got := len(fake.deleteRequests); got != 0 {
				t.Fatalf("delete calls = %d, want 0 while write readback is still pending", got)
			}
			requireCurrentKafkaClusterLifecycle(t, resource, tc.phase, string(tc.state))
		})
	}
}

func TestKafkaClusterDeleteStartsAfterSucceededUpdateWorkRequest(t *testing.T) {
	resource := newTestKafkaCluster()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.kafkacluster.oc1..existing")
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:   "wr-update-succeeded",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	fake := &fakeKafkaClusterOCIClient{}
	fake.getFn = func(_ context.Context, req managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		assertKafkaClusterGetRequest(t, req, "ocid1.kafkacluster.oc1..existing")
		return managedkafkasdk.GetKafkaClusterResponse{
			KafkaCluster: makeSDKKafkaCluster(resource, "ocid1.kafkacluster.oc1..existing", managedkafkasdk.KafkaClusterLifecycleStateActive),
		}, nil
	}
	fake.deleteFn = func(_ context.Context, req managedkafkasdk.DeleteKafkaClusterRequest) (managedkafkasdk.DeleteKafkaClusterResponse, error) {
		if got, want := stringValue(req.KafkaClusterId), "ocid1.kafkacluster.oc1..existing"; got != want {
			t.Fatalf("delete kafkaClusterId = %q, want %q", got, want)
		}
		return managedkafkasdk.DeleteKafkaClusterResponse{
			OpcRequestId:     common.String("opc-delete-after-update"),
			OpcWorkRequestId: common.String("wr-delete-after-update"),
		}, nil
	}
	fake.getWorkRequestFn = func(_ context.Context, req managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
		switch workRequestID := stringValue(req.WorkRequestId); workRequestID {
		case "wr-update-succeeded":
			return managedkafkasdk.GetWorkRequestResponse{
				WorkRequest: makeKafkaClusterWorkRequest(
					workRequestID,
					managedkafkasdk.OperationTypeUpdateKafkaCluster,
					managedkafkasdk.OperationStatusSucceeded,
					managedkafkasdk.ActionTypeUpdated,
					"ocid1.kafkacluster.oc1..existing",
				),
			}, nil
		case "wr-delete-after-update":
			return managedkafkasdk.GetWorkRequestResponse{
				WorkRequest: makeKafkaClusterWorkRequest(
					workRequestID,
					managedkafkasdk.OperationTypeDeleteKafkaCluster,
					managedkafkasdk.OperationStatusInProgress,
					managedkafkasdk.ActionTypeDeleted,
					"ocid1.kafkacluster.oc1..existing",
				),
			}, nil
		default:
			t.Fatalf("unexpected work request lookup %q", workRequestID)
			return managedkafkasdk.GetWorkRequestResponse{}, nil
		}
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if got, want := len(fake.deleteRequests), 1; got != want {
		t.Fatalf("delete calls = %d, want %d after update work request succeeds", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete-after-update"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	requireCurrentKafkaClusterWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-after-update")
}

func TestKafkaClusterDeleteAfterLifecyclePendingWriteSkipsUpdateDrift(t *testing.T) {
	resource := newTestKafkaCluster()
	resourceID := "ocid1.kafkacluster.oc1..existing"
	current := makeSDKKafkaCluster(resource, resourceID, managedkafkasdk.KafkaClusterLifecycleStateActive)
	resource.Spec.KafkaVersion = "3.9.0"
	resource.Spec.DisplayName = "delete-requested-with-drift"
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		RawStatus:       string(managedkafkasdk.KafkaClusterLifecycleStateUpdating),
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	fake := &fakeKafkaClusterOCIClient{}
	fake.getFn = func(_ context.Context, req managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		assertKafkaClusterGetRequest(t, req, resourceID)
		return managedkafkasdk.GetKafkaClusterResponse{KafkaCluster: current}, nil
	}
	fake.updateFn = func(context.Context, managedkafkasdk.UpdateKafkaClusterRequest) (managedkafkasdk.UpdateKafkaClusterResponse, error) {
		t.Fatal("UpdateKafkaCluster() should not run during delete after lifecycle-pending write")
		return managedkafkasdk.UpdateKafkaClusterResponse{}, nil
	}
	fake.deleteFn = func(_ context.Context, req managedkafkasdk.DeleteKafkaClusterRequest) (managedkafkasdk.DeleteKafkaClusterResponse, error) {
		if got, want := stringValue(req.KafkaClusterId), resourceID; got != want {
			t.Fatalf("delete kafkaClusterId = %q, want %q", got, want)
		}
		return managedkafkasdk.DeleteKafkaClusterResponse{
			OpcRequestId:     common.String("opc-delete-after-lifecycle"),
			OpcWorkRequestId: common.String("wr-delete-after-lifecycle"),
		}, nil
	}
	fake.getWorkRequestFn = func(_ context.Context, req managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
		assertKafkaClusterWorkRequestRequest(t, req, "wr-delete-after-lifecycle")
		return managedkafkasdk.GetWorkRequestResponse{
			WorkRequest: makeKafkaClusterWorkRequest(
				"wr-delete-after-lifecycle",
				managedkafkasdk.OperationTypeDeleteKafkaCluster,
				managedkafkasdk.OperationStatusInProgress,
				managedkafkasdk.ActionTypeDeleted,
				resourceID,
			),
		}, nil
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update calls = %d, want 0 during delete after lifecycle-pending write", got)
	}
	if got, want := len(fake.deleteRequests), 1; got != want {
		t.Fatalf("delete calls = %d, want %d after lifecycle-pending write clears", got, want)
	}
	requireCurrentKafkaClusterWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-after-lifecycle")
}

func TestKafkaClusterCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	resource := newTestKafkaCluster()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.kafkacluster.oc1..existing")
	current := makeSDKKafkaCluster(resource, "ocid1.kafkacluster.oc1..existing", managedkafkasdk.KafkaClusterLifecycleStateActive)
	current.KafkaVersion = common.String("3.7.0")

	fake := &fakeKafkaClusterOCIClient{}
	fake.getFn = func(_ context.Context, _ managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		return managedkafkasdk.GetKafkaClusterResponse{KafkaCluster: current}, nil
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, testKafkaClusterRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "kafkaVersion") {
		t.Fatalf("CreateOrUpdate() error = %v, want kafkaVersion drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful", response)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update calls = %d, want 0", got)
	}
}

func TestKafkaClusterDeleteStartsAndCompletesWorkRequest(t *testing.T) {
	resource := newTestKafkaCluster()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.kafkacluster.oc1..delete")
	current := makeSDKKafkaCluster(resource, "ocid1.kafkacluster.oc1..delete", managedkafkasdk.KafkaClusterLifecycleStateActive)

	fake := &fakeKafkaClusterOCIClient{}
	fake.getFn = func(_ context.Context, _ managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		return managedkafkasdk.GetKafkaClusterResponse{KafkaCluster: current}, nil
	}
	fake.deleteFn = func(_ context.Context, req managedkafkasdk.DeleteKafkaClusterRequest) (managedkafkasdk.DeleteKafkaClusterResponse, error) {
		if got, want := stringValue(req.KafkaClusterId), "ocid1.kafkacluster.oc1..delete"; got != want {
			t.Fatalf("delete kafkaClusterId = %q, want %q", got, want)
		}
		return managedkafkasdk.DeleteKafkaClusterResponse{
			OpcRequestId:     common.String("opc-delete-1"),
			OpcWorkRequestId: common.String("wr-delete-1"),
		}, nil
	}
	fake.getWorkRequestFn = func(_ context.Context, _ managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
		return managedkafkasdk.GetWorkRequestResponse{
			WorkRequest: makeKafkaClusterWorkRequest(
				"wr-delete-1",
				managedkafkasdk.OperationTypeDeleteKafkaCluster,
				managedkafkasdk.OperationStatusInProgress,
				managedkafkasdk.ActionTypeDeleted,
				"ocid1.kafkacluster.oc1..delete",
			),
		}, nil
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while work request is pending")
	}
	if got, want := len(fake.deleteRequests), 1; got != want {
		t.Fatalf("delete calls = %d, want %d", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete-1"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	requireCurrentKafkaClusterWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-1")

	fake.getFn = func(_ context.Context, _ managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		return managedkafkasdk.GetKafkaClusterResponse{}, errortest.NewServiceError(404, "NotFound", "")
	}
	fake.getWorkRequestFn = func(_ context.Context, _ managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
		return managedkafkasdk.GetWorkRequestResponse{
			WorkRequest: makeKafkaClusterWorkRequest(
				"wr-delete-1",
				managedkafkasdk.OperationTypeDeleteKafkaCluster,
				managedkafkasdk.OperationStatusSucceeded,
				managedkafkasdk.ActionTypeDeleted,
				"ocid1.kafkacluster.oc1..delete",
			),
		}, nil
	}
	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() after succeeded work request error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() after succeeded work request deleted = false, want true")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %+v, want nil after delete confirmation", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
}

func TestKafkaClusterDeleteAuthShapedNotFoundIsFatal(t *testing.T) {
	resource := newTestKafkaCluster()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.kafkacluster.oc1..delete")

	fake := &fakeKafkaClusterOCIClient{}
	fake.getFn = func(_ context.Context, _ managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		return managedkafkasdk.GetKafkaClusterResponse{
			KafkaCluster: makeSDKKafkaCluster(resource, "ocid1.kafkacluster.oc1..delete", managedkafkasdk.KafkaClusterLifecycleStateActive),
		}, nil
	}
	fake.deleteFn = func(context.Context, managedkafkasdk.DeleteKafkaClusterRequest) (managedkafkasdk.DeleteKafkaClusterResponse, error) {
		return managedkafkasdk.DeleteKafkaClusterResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped 404 error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %#v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	if !strings.Contains(err.Error(), "ambiguous 404") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 detail", err)
	}
}

func TestKafkaClusterDeleteConfirmationAuthShapedReadIsFatal(t *testing.T) {
	resource := newTestKafkaCluster()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.kafkacluster.oc1..delete")
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:        shared.OSOKAsyncSourceWorkRequest,
		Phase:         shared.OSOKAsyncPhaseDelete,
		WorkRequestID: "wr-delete-1",
	}

	fake := &fakeKafkaClusterOCIClient{}
	fake.getWorkRequestFn = func(context.Context, managedkafkasdk.GetWorkRequestRequest) (managedkafkasdk.GetWorkRequestResponse, error) {
		return managedkafkasdk.GetWorkRequestResponse{
			WorkRequest: makeKafkaClusterWorkRequest(
				"wr-delete-1",
				managedkafkasdk.OperationTypeDeleteKafkaCluster,
				managedkafkasdk.OperationStatusSucceeded,
				managedkafkasdk.ActionTypeDeleted,
				"ocid1.kafkacluster.oc1..delete",
			),
		}, nil
	}
	fake.getFn = func(context.Context, managedkafkasdk.GetKafkaClusterRequest) (managedkafkasdk.GetKafkaClusterResponse, error) {
		return managedkafkasdk.GetKafkaClusterResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped readback error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped readback")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %#v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	if !strings.Contains(err.Error(), "ambiguous 404") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 detail", err)
	}
}

func TestKafkaClusterCreateErrorRecordsOpcRequestID(t *testing.T) {
	resource := newTestKafkaCluster()
	fake := &fakeKafkaClusterOCIClient{}
	fake.createFn = func(_ context.Context, _ managedkafkasdk.CreateKafkaClusterRequest) (managedkafkasdk.CreateKafkaClusterResponse, error) {
		err := errortest.NewServiceError(429, "TooManyRequests", "rate limited")
		err.OpcRequestID = "opc-create-error"
		return managedkafkasdk.CreateKafkaClusterResponse{}, err
	}

	client := newTestKafkaClusterGeneratedClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, testKafkaClusterRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful", response)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create-error"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
}

func assertKafkaClusterCreateRequest(
	t *testing.T,
	req managedkafkasdk.CreateKafkaClusterRequest,
	resource *managedkafkav1beta1.KafkaCluster,
) {
	t.Helper()
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("create opc retry token is empty")
	}
	if got, want := stringValue(req.CompartmentId), resource.Spec.CompartmentId; got != want {
		t.Fatalf("create compartmentId = %q, want %q", got, want)
	}
	if got, want := stringValue(req.DisplayName), resource.Spec.DisplayName; got != want {
		t.Fatalf("create displayName = %q, want %q", got, want)
	}
	if got, want := len(req.AccessSubnets), len(resource.Spec.AccessSubnets); got != want {
		t.Fatalf("create accessSubnets length = %d, want %d", got, want)
	}
}

func assertKafkaClusterWorkRequestRequest(
	t *testing.T,
	req managedkafkasdk.GetWorkRequestRequest,
	workRequestID string,
) {
	t.Helper()
	if got, want := stringValue(req.WorkRequestId), workRequestID; got != want {
		t.Fatalf("get work request id = %q, want %q", got, want)
	}
}

func assertKafkaClusterListRequest(
	t *testing.T,
	req managedkafkasdk.ListKafkaClustersRequest,
	resource *managedkafkav1beta1.KafkaCluster,
) {
	t.Helper()
	if got, want := stringValue(req.CompartmentId), resource.Spec.CompartmentId; got != want {
		t.Fatalf("list compartmentId = %q, want %q", got, want)
	}
	if got, want := stringValue(req.DisplayName), resource.Spec.DisplayName; got != want {
		t.Fatalf("list displayName = %q, want %q", got, want)
	}
}

func kafkaClusterPagedListResponse(
	resource *managedkafkav1beta1.KafkaCluster,
	req managedkafkasdk.ListKafkaClustersRequest,
) managedkafkasdk.ListKafkaClustersResponse {
	if req.Page != nil {
		return managedkafkasdk.ListKafkaClustersResponse{
			KafkaClusterCollection: managedkafkasdk.KafkaClusterCollection{Items: []managedkafkasdk.KafkaClusterSummary{
				makeSDKKafkaClusterSummary(resource, "ocid1.kafkacluster.oc1..existing", managedkafkasdk.KafkaClusterLifecycleStateActive),
			}},
		}
	}
	other := makeSDKKafkaClusterSummary(resource, "ocid1.kafkacluster.oc1..other", managedkafkasdk.KafkaClusterLifecycleStateActive)
	other.DisplayName = common.String("other-kafka-cluster")
	return managedkafkasdk.ListKafkaClustersResponse{
		KafkaClusterCollection: managedkafkasdk.KafkaClusterCollection{Items: []managedkafkasdk.KafkaClusterSummary{
			makeSDKKafkaClusterSummary(resource, "ocid1.kafkacluster.oc1..deleted", managedkafkasdk.KafkaClusterLifecycleStateDeleted),
			other,
		}},
		OpcNextPage: common.String("page-2"),
	}
}

func assertKafkaClusterGetRequest(t *testing.T, req managedkafkasdk.GetKafkaClusterRequest, id string) {
	t.Helper()
	if got, want := stringValue(req.KafkaClusterId), id; got != want {
		t.Fatalf("get kafkaClusterId = %q, want %q", got, want)
	}
}

func assertKafkaClusterBindResult(
	t *testing.T,
	response servicemanager.OSOKResponse,
	resource *managedkafkav1beta1.KafkaCluster,
	pages []string,
	getCalls int,
	createCalled bool,
	updateCalled bool,
) {
	t.Helper()
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful non-requeue bind", response)
	}
	if createCalled || updateCalled {
		t.Fatalf("create/update called = %v/%v, want neither for bind", createCalled, updateCalled)
	}
	if !reflect.DeepEqual(pages, []string{"", "page-2"}) {
		t.Fatalf("list pages = %#v, want first page then page-2", pages)
	}
	if getCalls != 1 {
		t.Fatalf("get calls = %d, want 1 live read after bind", getCalls)
	}
	if got, want := resource.Status.Id, "ocid1.kafkacluster.oc1..existing"; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), "ocid1.kafkacluster.oc1..existing"; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func newTestKafkaClusterGeneratedClient(fake *fakeKafkaClusterOCIClient) KafkaClusterServiceClient {
	hooks := newKafkaClusterDefaultRuntimeHooks(managedkafkasdk.KafkaClusterClient{})
	hooks.Create.Call = fake.CreateKafkaCluster
	hooks.Get.Call = fake.GetKafkaCluster
	hooks.List.Call = fake.ListKafkaClusters
	hooks.Update.Call = fake.UpdateKafkaCluster
	hooks.Delete.Call = fake.DeleteKafkaCluster
	applyKafkaClusterRuntimeHooks(&hooks, fake, nil)
	delegate := defaultKafkaClusterServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*managedkafkav1beta1.KafkaCluster](
			buildKafkaClusterGeneratedRuntimeConfig(&KafkaClusterServiceManager{}, hooks),
		),
	}
	return wrapKafkaClusterGeneratedClient(hooks, delegate)
}

func newTestKafkaCluster() *managedkafkav1beta1.KafkaCluster {
	return &managedkafkav1beta1.KafkaCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-cluster-sample",
			Namespace: "default",
			UID:       types.UID("uid-kafka-cluster-sample"),
		},
		Spec: managedkafkav1beta1.KafkaClusterSpec{
			CompartmentId:        "ocid1.compartment.oc1..test",
			AccessSubnets:        []managedkafkav1beta1.KafkaClusterAccessSubnet{{Subnets: []string{"ocid1.subnet.oc1..a", "ocid1.subnet.oc1..b"}}},
			KafkaVersion:         "3.8.0",
			ClusterType:          string(managedkafkasdk.KafkaClusterClusterTypeDevelopment),
			BrokerShape:          managedkafkav1beta1.KafkaClusterBrokerShape{NodeCount: 1, OcpuCount: 1, NodeShape: "VM.Standard.E4.Flex"},
			ClusterConfigId:      "ocid1.kafkaclusterconfig.oc1..test",
			ClusterConfigVersion: 1,
			CoordinationType:     string(managedkafkasdk.KafkaClusterCoordinationTypeKraft),
			DisplayName:          "kafka-cluster-sample",
			FreeformTags:         map[string]string{"owner": "osok"},
			DefinedTags:          map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func testKafkaClusterRequest(resource *managedkafkav1beta1.KafkaCluster) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func makeSDKKafkaCluster(
	resource *managedkafkav1beta1.KafkaCluster,
	id string,
	state managedkafkasdk.KafkaClusterLifecycleStateEnum,
) managedkafkasdk.KafkaCluster {
	spec := resource.Spec
	return managedkafkasdk.KafkaCluster{
		Id:                   common.String(id),
		DisplayName:          common.String(spec.DisplayName),
		CompartmentId:        common.String(spec.CompartmentId),
		LifecycleState:       state,
		AccessSubnets:        buildKafkaClusterAccessSubnets(spec.AccessSubnets),
		KafkaVersion:         common.String(spec.KafkaVersion),
		ClusterType:          managedkafkasdk.KafkaClusterClusterTypeEnum(spec.ClusterType),
		BrokerShape:          buildKafkaClusterBrokerShape(spec.BrokerShape),
		ClusterConfigId:      common.String(spec.ClusterConfigId),
		ClusterConfigVersion: common.Int(spec.ClusterConfigVersion),
		FreeformTags:         cloneKafkaClusterStringMap(spec.FreeformTags),
		DefinedTags:          kafkaClusterDefinedTagsFromSpec(spec.DefinedTags),
		CoordinationType:     managedkafkasdk.KafkaClusterCoordinationTypeEnum(spec.CoordinationType),
	}
}

func makeSDKKafkaClusterSummary(
	resource *managedkafkav1beta1.KafkaCluster,
	id string,
	state managedkafkasdk.KafkaClusterLifecycleStateEnum,
) managedkafkasdk.KafkaClusterSummary {
	spec := resource.Spec
	return managedkafkasdk.KafkaClusterSummary{
		Id:                   common.String(id),
		DisplayName:          common.String(spec.DisplayName),
		CompartmentId:        common.String(spec.CompartmentId),
		LifecycleState:       state,
		AccessSubnets:        buildKafkaClusterAccessSubnets(spec.AccessSubnets),
		KafkaVersion:         common.String(spec.KafkaVersion),
		ClusterType:          managedkafkasdk.KafkaClusterClusterTypeEnum(spec.ClusterType),
		BrokerShape:          buildKafkaClusterBrokerShape(spec.BrokerShape),
		ClusterConfigId:      common.String(spec.ClusterConfigId),
		ClusterConfigVersion: common.Int(spec.ClusterConfigVersion),
		FreeformTags:         cloneKafkaClusterStringMap(spec.FreeformTags),
		DefinedTags:          kafkaClusterDefinedTagsFromSpec(spec.DefinedTags),
		CoordinationType:     managedkafkasdk.KafkaClusterCoordinationTypeEnum(spec.CoordinationType),
	}
}

func makeKafkaClusterWorkRequest(
	id string,
	operationType managedkafkasdk.OperationTypeEnum,
	status managedkafkasdk.OperationStatusEnum,
	action managedkafkasdk.ActionTypeEnum,
	resourceID string,
) managedkafkasdk.WorkRequest {
	percent := float32(50)
	return managedkafkasdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operationType,
		Status:          status,
		PercentComplete: &percent,
		Resources: []managedkafkasdk.WorkRequestResource{{
			EntityType: common.String(kafkaClusterWorkRequestEntityType),
			ActionType: action,
			Identifier: common.String(resourceID),
		}},
	}
}

func requireCurrentKafkaClusterWorkRequest(
	t *testing.T,
	resource *managedkafkav1beta1.KafkaCluster,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
}

func requireCurrentKafkaClusterLifecycle(
	t *testing.T,
	resource *managedkafkav1beta1.KafkaCluster,
	phase shared.OSOKAsyncPhase,
	rawStatus string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want lifecycle operation")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.RawStatus != rawStatus {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

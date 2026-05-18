/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package peertargetdatabase

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testPeerTargetDatabaseTargetID = "ocid1.datasafetargetdatabase.oc1..target"
	testPeerTargetDatabaseName     = "peer-target"
)

type fakePeerTargetDatabaseOCIClient struct {
	createFunc func(context.Context, datasafesdk.CreatePeerTargetDatabaseRequest) (datasafesdk.CreatePeerTargetDatabaseResponse, error)
	getFunc    func(context.Context, datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error)
	listFunc   func(context.Context, datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error)
	updateFunc func(context.Context, datasafesdk.UpdatePeerTargetDatabaseRequest) (datasafesdk.UpdatePeerTargetDatabaseResponse, error)
	deleteFunc func(context.Context, datasafesdk.DeletePeerTargetDatabaseRequest) (datasafesdk.DeletePeerTargetDatabaseResponse, error)

	createRequests []datasafesdk.CreatePeerTargetDatabaseRequest
	getRequests    []datasafesdk.GetPeerTargetDatabaseRequest
	listRequests   []datasafesdk.ListPeerTargetDatabasesRequest
	updateRequests []datasafesdk.UpdatePeerTargetDatabaseRequest
	deleteRequests []datasafesdk.DeletePeerTargetDatabaseRequest
}

func (f *fakePeerTargetDatabaseOCIClient) CreatePeerTargetDatabase(ctx context.Context, request datasafesdk.CreatePeerTargetDatabaseRequest) (datasafesdk.CreatePeerTargetDatabaseResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return datasafesdk.CreatePeerTargetDatabaseResponse{}, nil
}

func (f *fakePeerTargetDatabaseOCIClient) GetPeerTargetDatabase(ctx context.Context, request datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return datasafesdk.GetPeerTargetDatabaseResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
}

func (f *fakePeerTargetDatabaseOCIClient) ListPeerTargetDatabases(ctx context.Context, request datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return datasafesdk.ListPeerTargetDatabasesResponse{}, nil
}

func (f *fakePeerTargetDatabaseOCIClient) UpdatePeerTargetDatabase(ctx context.Context, request datasafesdk.UpdatePeerTargetDatabaseRequest) (datasafesdk.UpdatePeerTargetDatabaseResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return datasafesdk.UpdatePeerTargetDatabaseResponse{}, nil
}

func (f *fakePeerTargetDatabaseOCIClient) DeletePeerTargetDatabase(ctx context.Context, request datasafesdk.DeletePeerTargetDatabaseRequest) (datasafesdk.DeletePeerTargetDatabaseResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return datasafesdk.DeletePeerTargetDatabaseResponse{}, nil
}

func TestPeerTargetDatabaseCreateUsesParentAnnotationAndRecordsKey(t *testing.T) {
	resource := newTestPeerTargetDatabase()
	fake := newPeerTargetDatabaseCreateSuccessFake(t, resource)

	response, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreatePeerTargetDatabase() calls = %d, want 1", len(fake.createRequests))
	}
	requirePeerTargetDatabaseCreateStatus(t, resource)
}

func newPeerTargetDatabaseCreateSuccessFake(t *testing.T, resource *datasafev1beta1.PeerTargetDatabase) *fakePeerTargetDatabaseOCIClient {
	t.Helper()
	return &fakePeerTargetDatabaseOCIClient{
		createFunc: func(_ context.Context, request datasafesdk.CreatePeerTargetDatabaseRequest) (datasafesdk.CreatePeerTargetDatabaseResponse, error) {
			requirePeerTargetDatabaseCreateRequest(t, request)
			return datasafesdk.CreatePeerTargetDatabaseResponse{
				PeerTargetDatabase: newSDKPeerTargetDatabase(12, resource.Spec.DisplayName, datasafesdk.TargetDatabaseLifecycleStateCreating),
				OpcRequestId:       common.String("opc-create"),
				OpcWorkRequestId:   common.String("wr-create"),
			}, nil
		},
	}
}

func requirePeerTargetDatabaseCreateRequest(t *testing.T, request datasafesdk.CreatePeerTargetDatabaseRequest) {
	t.Helper()
	requireStringPointer(t, "Create.TargetDatabaseId", request.TargetDatabaseId, testPeerTargetDatabaseTargetID)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("Create.OpcRetryToken is empty")
	}
	details, ok := request.DatabaseDetails.(datasafesdk.InstalledDatabaseDetails)
	if !ok {
		t.Fatalf("Create.DatabaseDetails type = %T, want InstalledDatabaseDetails", request.DatabaseDetails)
	}
	requireStringPointer(t, "Create.DatabaseDetails.ServiceName", details.ServiceName, "salespdb")
	if details.ListenerPort == nil || *details.ListenerPort != 1521 {
		t.Fatalf("Create.DatabaseDetails.ListenerPort = %v, want 1521", details.ListenerPort)
	}
}

func requirePeerTargetDatabaseCreateStatus(t *testing.T, resource *datasafev1beta1.PeerTargetDatabase) {
	t.Helper()
	if got := resource.Status.Key; got != 12 {
		t.Fatalf("status.key = %d, want 12", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testPeerTargetDatabaseTargetID {
		t.Fatalf("status.status.ocid = %q, want target database id", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.WorkRequestID != "wr-create" || current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("status.status.async.current = %+v, want create work request", current)
	}
	if resource.Status.TlsConfig.StorePassword != "" || resource.Status.TlsConfig.TrustStoreContent != "" {
		t.Fatalf("status.tlsConfig leaked sensitive content: %+v", resource.Status.TlsConfig)
	}
}

func TestPeerTargetDatabaseBindUsesPaginatedListAndReadback(t *testing.T) {
	resource := newTestPeerTargetDatabase()
	fake := &fakePeerTargetDatabaseOCIClient{
		listFunc: func(_ context.Context, request datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error) {
			requireStringPointer(t, "List.TargetDatabaseId", request.TargetDatabaseId, testPeerTargetDatabaseTargetID)
			if request.Page == nil {
				return datasafesdk.ListPeerTargetDatabasesResponse{
					PeerTargetDatabaseCollection: datasafesdk.PeerTargetDatabaseCollection{
						Items: []datasafesdk.PeerTargetDatabaseSummary{
							newSDKPeerTargetDatabaseSummary(7, "other", "dg-other"),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPointer(t, "List.Page", request.Page, "page-2")
			return datasafesdk.ListPeerTargetDatabasesResponse{
				PeerTargetDatabaseCollection: datasafesdk.PeerTargetDatabaseCollection{
					Items: []datasafesdk.PeerTargetDatabaseSummary{
						newSDKPeerTargetDatabaseSummary(9, resource.Spec.DisplayName, resource.Spec.DataguardAssociationId),
					},
				},
			}, nil
		},
		getFunc: func(_ context.Context, request datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error) {
			requireStringPointer(t, "Get.TargetDatabaseId", request.TargetDatabaseId, testPeerTargetDatabaseTargetID)
			requireIntPointer(t, "Get.PeerTargetDatabaseId", request.PeerTargetDatabaseId, 9)
			return datasafesdk.GetPeerTargetDatabaseResponse{
				PeerTargetDatabase: newSDKPeerTargetDatabase(9, resource.Spec.DisplayName, datasafesdk.TargetDatabaseLifecycleStateActive),
			}, nil
		},
	}

	response, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful no requeue", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListPeerTargetDatabases() calls = %d, want 2", len(fake.listRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetPeerTargetDatabase() calls = %d, want 1", len(fake.getRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreatePeerTargetDatabase() calls = %d, want 0", len(fake.createRequests))
	}
	if got := resource.Status.Key; got != 9 {
		t.Fatalf("status.key = %d, want 9", got)
	}
}

func TestPeerTargetDatabaseMutableUpdateUsesKeyAndReadback(t *testing.T) {
	resource := newTestPeerTargetDatabase()
	resource.Status.OsokStatus.Ocid = shared.OCID(testPeerTargetDatabaseTargetID)
	resource.Status.Key = 15
	resource.Spec.Description = "updated description"

	getCalls := 0
	fake := &fakePeerTargetDatabaseOCIClient{
		getFunc: func(_ context.Context, request datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error) {
			getCalls++
			requireStringPointer(t, "Get.TargetDatabaseId", request.TargetDatabaseId, testPeerTargetDatabaseTargetID)
			requireIntPointer(t, "Get.PeerTargetDatabaseId", request.PeerTargetDatabaseId, 15)
			description := "old description"
			if getCalls > 1 {
				description = resource.Spec.Description
			}
			return datasafesdk.GetPeerTargetDatabaseResponse{
				PeerTargetDatabase: newSDKPeerTargetDatabaseWithDescription(15, resource.Spec.DisplayName, description),
			}, nil
		},
		updateFunc: func(_ context.Context, request datasafesdk.UpdatePeerTargetDatabaseRequest) (datasafesdk.UpdatePeerTargetDatabaseResponse, error) {
			requireStringPointer(t, "Update.TargetDatabaseId", request.TargetDatabaseId, testPeerTargetDatabaseTargetID)
			requireIntPointer(t, "Update.PeerTargetDatabaseId", request.PeerTargetDatabaseId, 15)
			requireStringPointer(t, "Update.Description", request.Description, resource.Spec.Description)
			return datasafesdk.UpdatePeerTargetDatabaseResponse{
				OpcRequestId:     common.String("opc-update"),
				OpcWorkRequestId: common.String("wr-update"),
			}, nil
		},
	}

	response, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful no requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdatePeerTargetDatabase() calls = %d, want 1", len(fake.updateRequests))
	}
	if len(fake.getRequests) != 2 {
		t.Fatalf("GetPeerTargetDatabase() calls = %d, want 2", len(fake.getRequests))
	}
	if got := resource.Status.Description; got != resource.Spec.Description {
		t.Fatalf("status.description = %q, want updated description", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
}

func TestPeerTargetDatabaseNoOpReconcileSkipsUpdate(t *testing.T) {
	resource := newTestPeerTargetDatabase()
	resource.Status.OsokStatus.Ocid = shared.OCID(testPeerTargetDatabaseTargetID)
	resource.Status.Key = 15
	fake := &fakePeerTargetDatabaseOCIClient{
		getFunc: func(_ context.Context, request datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error) {
			requireStringPointer(t, "Get.TargetDatabaseId", request.TargetDatabaseId, testPeerTargetDatabaseTargetID)
			requireIntPointer(t, "Get.PeerTargetDatabaseId", request.PeerTargetDatabaseId, 15)
			return datasafesdk.GetPeerTargetDatabaseResponse{
				PeerTargetDatabase: newSDKPeerTargetDatabase(15, resource.Spec.DisplayName, datasafesdk.TargetDatabaseLifecycleStateActive),
			}, nil
		},
	}

	response, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful no requeue", response)
	}
	if len(fake.createRequests) != 0 || len(fake.updateRequests) != 0 {
		t.Fatalf("create/update calls = %d/%d, want 0/0", len(fake.createRequests), len(fake.updateRequests))
	}
	if got := resource.Status.Description; got != resource.Spec.Description {
		t.Fatalf("status.description = %q, want %q", got, resource.Spec.Description)
	}
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("status.status.async.current = %+v, want nil", current)
	}
}

func TestPeerTargetDatabasePendingLifecycleSkipsMutableUpdate(t *testing.T) {
	tests := []struct {
		name           string
		state          datasafesdk.TargetDatabaseLifecycleStateEnum
		wantReason     shared.OSOKConditionType
		wantAsyncPhase shared.OSOKAsyncPhase
	}{
		{
			name:           "creating",
			state:          datasafesdk.TargetDatabaseLifecycleStateCreating,
			wantReason:     shared.Provisioning,
			wantAsyncPhase: shared.OSOKAsyncPhaseCreate,
		},
		{
			name:           "updating",
			state:          datasafesdk.TargetDatabaseLifecycleStateUpdating,
			wantReason:     shared.Updating,
			wantAsyncPhase: shared.OSOKAsyncPhaseUpdate,
		},
		{
			name:           "deleting",
			state:          datasafesdk.TargetDatabaseLifecycleStateDeleting,
			wantReason:     shared.Terminating,
			wantAsyncPhase: shared.OSOKAsyncPhaseDelete,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resource := newTestPeerTargetDatabase()
			resource.Status.OsokStatus.Ocid = shared.OCID(testPeerTargetDatabaseTargetID)
			resource.Status.Key = 15
			resource.Spec.Description = "updated description"
			fake := &fakePeerTargetDatabaseOCIClient{
				getFunc: func(_ context.Context, request datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error) {
					requireStringPointer(t, "Get.TargetDatabaseId", request.TargetDatabaseId, testPeerTargetDatabaseTargetID)
					requireIntPointer(t, "Get.PeerTargetDatabaseId", request.PeerTargetDatabaseId, 15)
					return datasafesdk.GetPeerTargetDatabaseResponse{
						PeerTargetDatabase: newSDKPeerTargetDatabaseWithDescription(15, resource.Spec.DisplayName, "old description", test.state),
					}, nil
				},
			}

			response, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			requirePeerTargetDatabasePendingUpdateSkipped(t, resource, fake, response, test.state, test.wantReason, test.wantAsyncPhase)
		})
	}
}

func requirePeerTargetDatabasePendingUpdateSkipped(
	t *testing.T,
	resource *datasafev1beta1.PeerTargetDatabase,
	fake *fakePeerTargetDatabaseOCIClient,
	response servicemanager.OSOKResponse,
	state datasafesdk.TargetDatabaseLifecycleStateEnum,
	wantReason shared.OSOKConditionType,
	wantAsyncPhase shared.OSOKAsyncPhase,
) {
	t.Helper()
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful requeue", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdatePeerTargetDatabase() calls = %d, want 0", len(fake.updateRequests))
	}
	if got := resource.Status.Description; got != "old description" {
		t.Fatalf("status.description = %q, want observed old description", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(wantReason) {
		t.Fatalf("status.status.reason = %q, want %q", got, wantReason)
	}
	requirePeerTargetDatabaseLifecycleAsync(t, resource, state, wantAsyncPhase)
}

func requirePeerTargetDatabaseLifecycleAsync(
	t *testing.T,
	resource *datasafev1beta1.PeerTargetDatabase,
	state datasafesdk.TargetDatabaseLifecycleStateEnum,
	wantAsyncPhase shared.OSOKAsyncPhase,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want lifecycle pending operation")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle || current.Phase != wantAsyncPhase {
		t.Fatalf("status.status.async.current = %+v, want lifecycle phase %q", current, wantAsyncPhase)
	}
	if current.RawStatus != string(state) || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current = %+v, want raw state %q pending", current, state)
	}
}

func TestPeerTargetDatabaseUpdateRecordsOCIError(t *testing.T) {
	resource := newTestPeerTargetDatabase()
	resource.Status.OsokStatus.Ocid = shared.OCID(testPeerTargetDatabaseTargetID)
	resource.Status.Key = 15
	resource.Spec.Description = "updated description"
	updateErr := errortest.NewServiceError(500, "InternalError", "update failed")
	fake := &fakePeerTargetDatabaseOCIClient{
		getFunc: func(context.Context, datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error) {
			return datasafesdk.GetPeerTargetDatabaseResponse{
				PeerTargetDatabase: newSDKPeerTargetDatabaseWithDescription(15, resource.Spec.DisplayName, "old description"),
			}, nil
		},
		updateFunc: func(context.Context, datasafesdk.UpdatePeerTargetDatabaseRequest) (datasafesdk.UpdatePeerTargetDatabaseResponse, error) {
			return datasafesdk.UpdatePeerTargetDatabaseResponse{}, updateErr
		},
	}

	response, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "update failed") {
		t.Fatalf("CreateOrUpdate() error = %v, want update failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdatePeerTargetDatabase() calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want error request id", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.status.reason = %q, want %q", got, shared.Failed)
	}
}

func TestPeerTargetDatabaseRejectsCreateOnlyDataguardDrift(t *testing.T) {
	resource := newTestPeerTargetDatabase()
	resource.Status.OsokStatus.Ocid = shared.OCID(testPeerTargetDatabaseTargetID)
	resource.Status.Key = 15
	resource.Spec.DataguardAssociationId = "ocid1.dataguardassociation.oc1..replacement"
	fake := &fakePeerTargetDatabaseOCIClient{
		getFunc: func(context.Context, datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error) {
			return datasafesdk.GetPeerTargetDatabaseResponse{
				PeerTargetDatabase: newSDKPeerTargetDatabase(15, resource.Spec.DisplayName, datasafesdk.TargetDatabaseLifecycleStateActive),
			}, nil
		},
	}

	_, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "create-only") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only drift", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdatePeerTargetDatabase() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestPeerTargetDatabaseDeleteWithoutRecordedKeyReleasesMissingParentWithoutOCI(t *testing.T) {
	resource := newTestPeerTargetDatabase()
	resource.Annotations = nil
	resource.Status.OsokStatus.Reason = string(shared.Failed)
	fake := &fakePeerTargetDatabaseOCIClient{}

	deleted, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when no peer key was recorded")
	}
	requireNoPeerTargetDatabaseDeleteLookup(t, fake)
	if got := resource.Status.OsokStatus.Message; !strings.Contains(got, "never recorded") {
		t.Fatalf("status.status.message = %q, want never recorded", got)
	}
}

func TestPeerTargetDatabaseDeleteWithoutRecordedKeyDoesNotListByBindCriteria(t *testing.T) {
	resource := newTestPeerTargetDatabase()
	fake := &fakePeerTargetDatabaseOCIClient{
		listFunc: func(context.Context, datasafesdk.ListPeerTargetDatabasesRequest) (datasafesdk.ListPeerTargetDatabasesResponse, error) {
			return datasafesdk.ListPeerTargetDatabasesResponse{
				PeerTargetDatabaseCollection: datasafesdk.PeerTargetDatabaseCollection{
					Items: []datasafesdk.PeerTargetDatabaseSummary{
						newSDKPeerTargetDatabaseSummary(21, resource.Spec.DisplayName, resource.Spec.DataguardAssociationId),
					},
				},
			}, nil
		},
	}

	deleted, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when no peer key was recorded")
	}
	requireNoPeerTargetDatabaseDeleteLookup(t, fake)
}

func TestPeerTargetDatabaseDeleteRetainsFinalizerUntilConfirmed(t *testing.T) {
	resource := newTestPeerTargetDatabase()
	resource.Status.OsokStatus.Ocid = shared.OCID(testPeerTargetDatabaseTargetID)
	resource.Status.Key = 21
	getCalls := 0
	fake := &fakePeerTargetDatabaseOCIClient{
		getFunc: func(_ context.Context, request datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error) {
			getCalls++
			requireStringPointer(t, "Get.TargetDatabaseId", request.TargetDatabaseId, testPeerTargetDatabaseTargetID)
			requireIntPointer(t, "Get.PeerTargetDatabaseId", request.PeerTargetDatabaseId, 21)
			state := datasafesdk.TargetDatabaseLifecycleStateActive
			if getCalls > 1 {
				state = datasafesdk.TargetDatabaseLifecycleStateDeleting
			}
			return datasafesdk.GetPeerTargetDatabaseResponse{
				PeerTargetDatabase: newSDKPeerTargetDatabase(21, resource.Spec.DisplayName, state),
			}, nil
		},
		deleteFunc: func(_ context.Context, request datasafesdk.DeletePeerTargetDatabaseRequest) (datasafesdk.DeletePeerTargetDatabaseResponse, error) {
			requireStringPointer(t, "Delete.TargetDatabaseId", request.TargetDatabaseId, testPeerTargetDatabaseTargetID)
			requireIntPointer(t, "Delete.PeerTargetDatabaseId", request.PeerTargetDatabaseId, 21)
			return datasafesdk.DeletePeerTargetDatabaseResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			}, nil
		},
	}

	deleted, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI is DELETING")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeletePeerTargetDatabase() calls = %d, want 1", len(fake.deleteRequests))
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseDelete || current.WorkRequestID != "wr-delete" {
		t.Fatalf("status.status.async.current = %+v, want delete work request", current)
	}
}

func TestPeerTargetDatabaseDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	resource := newTestPeerTargetDatabase()
	resource.Status.OsokStatus.Ocid = shared.OCID(testPeerTargetDatabaseTargetID)
	resource.Status.Key = 21
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakePeerTargetDatabaseOCIClient{
		getFunc: func(context.Context, datasafesdk.GetPeerTargetDatabaseRequest) (datasafesdk.GetPeerTargetDatabaseResponse, error) {
			return datasafesdk.GetPeerTargetDatabaseResponse{}, authErr
		},
	}

	deleted, err := newPeerTargetDatabaseServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), errorutil.NotAuthorizedOrNotFound) {
		t.Fatalf("Delete() error = %v, want auth-shaped not found", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous auth-shaped read")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeletePeerTargetDatabase() calls = %d, want 0", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want error request id", got)
	}
}

func requireNoPeerTargetDatabaseDeleteLookup(t *testing.T, fake *fakePeerTargetDatabaseOCIClient) {
	t.Helper()
	if len(fake.getRequests) != 0 {
		t.Fatalf("GetPeerTargetDatabase() calls = %d, want 0", len(fake.getRequests))
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("ListPeerTargetDatabases() calls = %d, want 0", len(fake.listRequests))
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeletePeerTargetDatabase() calls = %d, want 0", len(fake.deleteRequests))
	}
}

func newTestPeerTargetDatabase() *datasafev1beta1.PeerTargetDatabase {
	return &datasafev1beta1.PeerTargetDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPeerTargetDatabaseName,
			Namespace: "default",
			Annotations: map[string]string{
				peerTargetDatabaseTargetDatabaseIDAnnotation: testPeerTargetDatabaseTargetID,
			},
		},
		Spec: datasafev1beta1.PeerTargetDatabaseSpec{
			DisplayName:            testPeerTargetDatabaseName,
			Description:            "initial description",
			DataguardAssociationId: "ocid1.dataguardassociation.oc1..example",
			DatabaseDetails: datasafev1beta1.PeerTargetDatabaseDatabaseDetails{
				InfrastructureType: string(datasafesdk.InfrastructureTypeOnPremises),
				DatabaseType:       string(datasafesdk.DatabaseTypeInstalledDatabase),
				ListenerPort:       1521,
				ServiceName:        "salespdb",
				IpAddresses:        []string{"192.0.2.10"},
			},
			TlsConfig: datasafev1beta1.PeerTargetDatabaseTlsConfig{
				Status:               string(datasafesdk.TlsConfigStatusEnabled),
				CertificateStoreType: string(datasafesdk.TlsConfigCertificateStoreTypeJks),
				StorePassword:        "super-secret",
				TrustStoreContent:    "trust-store-content",
				KeyStoreContent:      "key-store-content",
			},
		},
	}
}

func newSDKPeerTargetDatabase(
	key int,
	displayName string,
	state datasafesdk.TargetDatabaseLifecycleStateEnum,
) datasafesdk.PeerTargetDatabase {
	return newSDKPeerTargetDatabaseWithDescription(key, displayName, "initial description", state)
}

func newSDKPeerTargetDatabaseWithDescription(
	key int,
	displayName string,
	description string,
	state ...datasafesdk.TargetDatabaseLifecycleStateEnum,
) datasafesdk.PeerTargetDatabase {
	lifecycleState := datasafesdk.TargetDatabaseLifecycleStateActive
	if len(state) > 0 {
		lifecycleState = state[0]
	}
	return datasafesdk.PeerTargetDatabase{
		DisplayName:            common.String(displayName),
		Key:                    common.Int(key),
		DataguardAssociationId: common.String("ocid1.dataguardassociation.oc1..example"),
		DatabaseDetails: datasafesdk.InstalledDatabaseDetails{
			InfrastructureType: datasafesdk.InfrastructureTypeOnPremises,
			ListenerPort:       common.Int(1521),
			ServiceName:        common.String("salespdb"),
			IpAddresses:        []string{"192.0.2.10"},
		},
		LifecycleState:   lifecycleState,
		Description:      common.String(description),
		LifecycleDetails: common.String(string(lifecycleState)),
		TlsConfig: &datasafesdk.TlsConfig{
			Status:               datasafesdk.TlsConfigStatusEnabled,
			CertificateStoreType: datasafesdk.TlsConfigCertificateStoreTypeJks,
		},
	}
}

func newSDKPeerTargetDatabaseSummary(key int, displayName string, dataguardAssociationID string) datasafesdk.PeerTargetDatabaseSummary {
	return datasafesdk.PeerTargetDatabaseSummary{
		DisplayName:            common.String(displayName),
		Key:                    common.Int(key),
		DataguardAssociationId: common.String(dataguardAssociationID),
		LifecycleState:         datasafesdk.TargetDatabaseLifecycleStateActive,
	}
}

func requireStringPointer(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", label, got, want)
	}
}

func requireIntPointer(t *testing.T, label string, got *int, want int) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %d", label, got, want)
	}
}

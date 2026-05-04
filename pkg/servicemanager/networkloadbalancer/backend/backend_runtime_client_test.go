package backend

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	networkloadbalancersdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	networkloadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeBackendOCIClient struct {
	createFn         func(context.Context, networkloadbalancersdk.CreateBackendRequest) (networkloadbalancersdk.CreateBackendResponse, error)
	getFn            func(context.Context, networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error)
	listFn           func(context.Context, networkloadbalancersdk.ListBackendsRequest) (networkloadbalancersdk.ListBackendsResponse, error)
	updateFn         func(context.Context, networkloadbalancersdk.UpdateBackendRequest) (networkloadbalancersdk.UpdateBackendResponse, error)
	deleteFn         func(context.Context, networkloadbalancersdk.DeleteBackendRequest) (networkloadbalancersdk.DeleteBackendResponse, error)
	getWorkRequestFn func(context.Context, networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error)
}

func (f *fakeBackendOCIClient) CreateBackend(ctx context.Context, req networkloadbalancersdk.CreateBackendRequest) (networkloadbalancersdk.CreateBackendResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return networkloadbalancersdk.CreateBackendResponse{}, nil
}

func (f *fakeBackendOCIClient) GetBackend(ctx context.Context, req networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return networkloadbalancersdk.GetBackendResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing backend")
}

func (f *fakeBackendOCIClient) ListBackends(ctx context.Context, req networkloadbalancersdk.ListBackendsRequest) (networkloadbalancersdk.ListBackendsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return networkloadbalancersdk.ListBackendsResponse{}, nil
}

func (f *fakeBackendOCIClient) UpdateBackend(ctx context.Context, req networkloadbalancersdk.UpdateBackendRequest) (networkloadbalancersdk.UpdateBackendResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return networkloadbalancersdk.UpdateBackendResponse{}, nil
}

func (f *fakeBackendOCIClient) DeleteBackend(ctx context.Context, req networkloadbalancersdk.DeleteBackendRequest) (networkloadbalancersdk.DeleteBackendResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return networkloadbalancersdk.DeleteBackendResponse{}, nil
}

func (f *fakeBackendOCIClient) GetWorkRequest(ctx context.Context, req networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return networkloadbalancersdk.GetWorkRequestResponse{}, nil
}

func newTestBackendRuntimeClient(client *fakeBackendOCIClient) BackendServiceClient {
	hooks := newBackendRuntimeHooksWithOCIClient(client)
	applyBackendRuntimeHooks(&hooks, client, nil)
	config := buildBackendGeneratedRuntimeConfig(&BackendServiceManager{}, hooks)
	delegate := defaultBackendServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkloadbalancerv1beta1.Backend](config),
	}
	return wrapBackendGeneratedClient(hooks, delegate)
}

func TestCreateOrUpdateBindsExistingBackend(t *testing.T) {
	t.Parallel()

	createCalled := false
	updateCalled := false
	var getRequest networkloadbalancersdk.GetBackendRequest
	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(_ context.Context, req networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			getRequest = req
			return networkloadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, false, false, false)}, nil
		},
		createFn: func(context.Context, networkloadbalancersdk.CreateBackendRequest) (networkloadbalancersdk.CreateBackendResponse, error) {
			createCalled = true
			return networkloadbalancersdk.CreateBackendResponse{}, nil
		},
		updateFn: func(context.Context, networkloadbalancersdk.UpdateBackendRequest) (networkloadbalancersdk.UpdateBackendResponse, error) {
			updateCalled = true
			return networkloadbalancersdk.UpdateBackendResponse{}, nil
		},
	})

	resource := makeBackendResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful response", response)
	}
	if createCalled {
		t.Fatal("CreateBackend() called, want bind/observe path")
	}
	if updateCalled {
		t.Fatal("UpdateBackend() called, want observe-only bind path")
	}
	assertBackendPathIdentity(t, getRequest.NetworkLoadBalancerId, getRequest.BackendSetName, getRequest.BackendName)
	assertBackendStatusIdentity(t, resource)
	if got := resource.Status.OsokStatus.Ocid; got != shared.OCID(backendName) {
		t.Fatalf("status.status.ocid = %q, want %q", got, backendName)
	}
}

func TestCreateOrUpdateCreatesBackendAndPollsWorkRequest(t *testing.T) {
	t.Parallel()

	getCalls := 0
	createCalls := 0
	workRequestCalls := 0
	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(_ context.Context, req networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			getCalls++
			assertBackendPathIdentity(t, req.NetworkLoadBalancerId, req.BackendSetName, req.BackendName)
			if getCalls == 1 {
				return networkloadbalancersdk.GetBackendResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing backend")
			}
			return networkloadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, false, false, false)}, nil
		},
		createFn: func(_ context.Context, req networkloadbalancersdk.CreateBackendRequest) (networkloadbalancersdk.CreateBackendResponse, error) {
			createCalls++
			assertCreateBackendRequest(t, req)
			return networkloadbalancersdk.CreateBackendResponse{
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			if got := stringValue(req.WorkRequestId); got != "wr-create" {
				t.Fatalf("GetWorkRequestRequest.workRequestId = %q, want %q", got, "wr-create")
			}
			return backendWorkRequestResponse("wr-create", networkloadbalancersdk.OperationStatusSucceeded, networkloadbalancersdk.OperationTypeCreateBackend), nil
		},
	})

	resource := makeBackendResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful response", response)
	}
	if createCalls != 1 {
		t.Fatalf("CreateBackend() calls = %d, want 1", createCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetBackend() calls = %d, want 2", getCalls)
	}
	assertBackendStatusIdentity(t, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, "opc-create")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after successful work request", resource.Status.OsokStatus.Async.Current)
	}
}

func TestCreateOrUpdateUpdatesMutableBackendFieldsIncludingFalse(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendResource()
	resource.Spec.Weight = 5
	resource.Spec.IsBackup = false
	resource.Spec.IsDrain = false
	resource.Spec.IsOffline = false
	resource.Status.IsBackup = true
	resource.Status.IsDrain = true
	resource.Status.IsOffline = true

	getCalls := 0
	var updateRequest networkloadbalancersdk.UpdateBackendRequest
	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(_ context.Context, req networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			getCalls++
			assertBackendPathIdentity(t, req.NetworkLoadBalancerId, req.BackendSetName, req.BackendName)
			if getCalls == 1 {
				return networkloadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, true, true, true)}, nil
			}
			return networkloadbalancersdk.GetBackendResponse{Backend: sdkBackend(resource.Spec.Weight, false, false, false)}, nil
		},
		updateFn: func(_ context.Context, req networkloadbalancersdk.UpdateBackendRequest) (networkloadbalancersdk.UpdateBackendResponse, error) {
			updateRequest = req
			return networkloadbalancersdk.UpdateBackendResponse{
				OpcRequestId:     common.String("opc-update"),
				OpcWorkRequestId: common.String("wr-update"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error) {
			if got := stringValue(req.WorkRequestId); got != "wr-update" {
				t.Fatalf("GetWorkRequestRequest.workRequestId = %q, want %q", got, "wr-update")
			}
			return backendWorkRequestResponse("wr-update", networkloadbalancersdk.OperationStatusSucceeded, networkloadbalancersdk.OperationTypeUpdateBackend), nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update response", response)
	}
	assertMutableUpdateBackendRequest(t, updateRequest, resource.Spec.Weight)
	assertMutableUpdateBackendStatus(t, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, "opc-update")
	}
}

func TestCreateOrUpdateRejectsForceNewBackendDrift(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		mutate  func(*networkloadbalancerv1beta1.Backend)
		wantErr string
	}{
		{
			name: "networkLoadBalancerId",
			mutate: func(resource *networkloadbalancerv1beta1.Backend) {
				resource.Spec.NetworkLoadBalancerId = "ocid1.networkloadbalancer.oc1..replacement"
			},
			wantErr: "require replacement when networkLoadBalancerId changes",
		},
		{
			name: "backendSetName",
			mutate: func(resource *networkloadbalancerv1beta1.Backend) {
				resource.Spec.BackendSetName = "replacement_backend_set"
			},
			wantErr: "require replacement when backendSetName changes",
		},
		{
			name: "ipAddress",
			mutate: func(resource *networkloadbalancerv1beta1.Backend) {
				resource.Spec.IpAddress = "10.0.0.9"
			},
			wantErr: "require replacement when ipAddress changes",
		},
		{
			name: "name",
			mutate: func(resource *networkloadbalancerv1beta1.Backend) {
				resource.Spec.Name = "replacement_backend"
			},
			wantErr: "require replacement when name changes",
		},
		{
			name: "port",
			mutate: func(resource *networkloadbalancerv1beta1.Backend) {
				resource.Spec.Port = 9090
			},
			wantErr: "require replacement when port changes",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := makeTrackedBackendResource()
			tc.mutate(resource)

			updateCalled := false
			client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
				getFn: func(_ context.Context, req networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
					assertBackendPathIdentity(t, req.NetworkLoadBalancerId, req.BackendSetName, req.BackendName)
					return networkloadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, false, false, false)}, nil
				},
				updateFn: func(context.Context, networkloadbalancersdk.UpdateBackendRequest) (networkloadbalancersdk.UpdateBackendResponse, error) {
					updateCalled = true
					return networkloadbalancersdk.UpdateBackendResponse{}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("CreateOrUpdate() error = %v, want %q", err, tc.wantErr)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
			}
			if updateCalled {
				t.Fatal("UpdateBackend() called, want force-new drift rejection before update")
			}
		})
	}
}

func TestCreateOrUpdateRejectsBackendIpAddressToTargetIDDriftBeforeObserve(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendResource()
	resource.Spec.IpAddress = ""
	resource.Spec.TargetId = backendTargetID

	assertBackendAddressDriftRejectedBeforeObserve(t, resource, "require replacement when targetId changes")
}

func TestCreateOrUpdateRejectsBackendTargetIDToIpAddressDriftBeforeObserve(t *testing.T) {
	t.Parallel()

	resource := makeTrackedTargetBackendResource()
	resource.Spec.TargetId = ""
	resource.Spec.IpAddress = backendIP

	assertBackendAddressDriftRejectedBeforeObserve(t, resource, "require replacement when ipAddress changes")
}

func TestCreateOrUpdateRejectsBackendTargetIDValueDriftBeforeObserve(t *testing.T) {
	t.Parallel()

	resource := makeTrackedTargetBackendResource()
	resource.Spec.TargetId = replacementBackendTargetID

	assertBackendAddressDriftRejectedBeforeObserve(t, resource, "require replacement when targetId changes")
}

func TestDeletePreservesPendingBackendCreateWorkRequest(t *testing.T) {
	t.Parallel()

	runBackendPendingWriteWorkRequestDeleteTest(
		t,
		shared.OSOKAsyncPhaseCreate,
		"wr-create",
		networkloadbalancersdk.OperationTypeCreateBackend,
	)
}

func TestDeletePreservesPendingBackendUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	runBackendPendingWriteWorkRequestDeleteTest(
		t,
		shared.OSOKAsyncPhaseUpdate,
		"wr-update",
		networkloadbalancersdk.OperationTypeUpdateBackend,
	)
}

func TestDeletePollsWorkRequestAndConfirmsBackendRemoval(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendResource()
	resource.Spec.NetworkLoadBalancerId = "ocid1.networkloadbalancer.oc1..replacement"
	resource.Spec.BackendSetName = "replacement_backend_set"
	resource.Spec.IpAddress = "10.0.0.9"
	resource.Spec.Port = 9090

	getCalls := 0
	deleteCalls := 0
	workRequestCalls := 0
	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(_ context.Context, req networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			getCalls++
			assertBackendPathIdentity(t, req.NetworkLoadBalancerId, req.BackendSetName, req.BackendName)
			if getCalls == 1 {
				return networkloadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, false, false, false)}, nil
			}
			return networkloadbalancersdk.GetBackendResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "backend deleted")
		},
		deleteFn: func(_ context.Context, req networkloadbalancersdk.DeleteBackendRequest) (networkloadbalancersdk.DeleteBackendResponse, error) {
			deleteCalls++
			assertBackendPathIdentity(t, req.NetworkLoadBalancerId, req.BackendSetName, req.BackendName)
			return networkloadbalancersdk.DeleteBackendResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			if got := stringValue(req.WorkRequestId); got != "wr-delete" {
				t.Fatalf("GetWorkRequestRequest.workRequestId = %q, want %q", got, "wr-delete")
			}
			if workRequestCalls == 1 {
				return backendWorkRequestResponse("wr-delete", networkloadbalancersdk.OperationStatusInProgress, networkloadbalancersdk.OperationTypeDeleteBackend), nil
			}
			return backendWorkRequestResponse("wr-delete", networkloadbalancersdk.OperationStatusSucceeded, networkloadbalancersdk.OperationTypeDeleteBackend), nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	assertPendingBackendDelete(t, resource, deleted, deleteCalls, getCalls)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	assertConfirmedBackendDelete(t, resource, deleted, deleteCalls, getCalls)
}

func TestDeleteSucceededWorkRequestTreatsAuthShapedConfirmationNotFoundAsFatal(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendResource()
	seedPendingBackendDeleteWorkRequest(resource, "wr-delete")

	getCalls := 0
	deleteCalls := 0
	workRequestCalls := 0
	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(_ context.Context, req networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			getCalls++
			assertBackendPathIdentity(t, req.NetworkLoadBalancerId, req.BackendSetName, req.BackendName)
			return networkloadbalancersdk.GetBackendResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous delete confirmation")
		},
		deleteFn: func(context.Context, networkloadbalancersdk.DeleteBackendRequest) (networkloadbalancersdk.DeleteBackendResponse, error) {
			deleteCalls++
			return networkloadbalancersdk.DeleteBackendResponse{}, nil
		},
		getWorkRequestFn: func(_ context.Context, req networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			if got := stringValue(req.WorkRequestId); got != "wr-delete" {
				t.Fatalf("GetWorkRequestRequest.workRequestId = %q, want %q", got, "wr-delete")
			}
			return backendWorkRequestResponse("wr-delete", networkloadbalancersdk.OperationStatusSucceeded, networkloadbalancersdk.OperationTypeDeleteBackend), nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirmation failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for ambiguous confirmation")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteBackend() calls = %d, want 0 after completed work request confirmation fails", deleteCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetBackend() calls = %d, want 1 confirmation read", getCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1 guard poll", workRequestCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, "opc-request-id")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt should stay empty for ambiguous delete confirmation")
	}
	assertCurrentBackendDeleteWorkRequest(t, resource, "wr-delete")
}

func TestDeleteSucceededWorkRequestAcceptsUnambiguousNotFoundConfirmation(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendResource()
	seedPendingBackendDeleteWorkRequest(resource, "wr-delete")

	getCalls := 0
	deleteCalls := 0
	workRequestCalls := 0
	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(_ context.Context, req networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			getCalls++
			assertBackendPathIdentity(t, req.NetworkLoadBalancerId, req.BackendSetName, req.BackendName)
			return networkloadbalancersdk.GetBackendResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "backend deleted")
		},
		deleteFn: func(context.Context, networkloadbalancersdk.DeleteBackendRequest) (networkloadbalancersdk.DeleteBackendResponse, error) {
			deleteCalls++
			return networkloadbalancersdk.DeleteBackendResponse{}, nil
		},
		getWorkRequestFn: func(_ context.Context, req networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			if got := stringValue(req.WorkRequestId); got != "wr-delete" {
				t.Fatalf("GetWorkRequestRequest.workRequestId = %q, want %q", got, "wr-delete")
			}
			return backendWorkRequestResponse("wr-delete", networkloadbalancersdk.OperationStatusSucceeded, networkloadbalancersdk.OperationTypeDeleteBackend), nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want confirmed delete for unambiguous NotFound")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteBackend() calls = %d, want 0 while resuming completed delete work request", deleteCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetBackend() calls = %d, want guard and generatedruntime confirmation reads", getCalls)
	}
	if workRequestCalls != 2 {
		t.Fatalf("GetWorkRequest() calls = %d, want guard and generatedruntime polls", workRequestCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt should be set after unambiguous delete confirmation")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after confirmed delete", resource.Status.OsokStatus.Async.Current)
	}
}

func TestDeleteReturnsPreDeleteAuthShapedNotFoundWithoutDeleteCall(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	resource := makeTrackedBackendResource()
	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(_ context.Context, req networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			assertBackendPathIdentity(t, req.NetworkLoadBalancerId, req.BackendSetName, req.BackendName)
			return networkloadbalancersdk.GetBackendResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous pre-delete read")
		},
		deleteFn: func(context.Context, networkloadbalancersdk.DeleteBackendRequest) (networkloadbalancersdk.DeleteBackendResponse, error) {
			deleteCalled = true
			return networkloadbalancersdk.DeleteBackendResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want pre-delete NotAuthorizedOrNotFound error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for pre-delete NotAuthorizedOrNotFound")
	}
	if deleteCalled {
		t.Fatal("DeleteBackend() called, want pre-delete NotAuthorizedOrNotFound to stop before delete")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want NotAuthorizedOrNotFound context", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, "opc-request-id")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt should stay empty for pre-delete NotAuthorizedOrNotFound")
	}
}

func TestDeleteTreatsAuthShapedNotFoundAsAmbiguous(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendResource()
	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(_ context.Context, req networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			assertBackendPathIdentity(t, req.NetworkLoadBalancerId, req.BackendSetName, req.BackendName)
			return networkloadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, false, false, false)}, nil
		},
		deleteFn: func(context.Context, networkloadbalancersdk.DeleteBackendRequest) (networkloadbalancersdk.DeleteBackendResponse, error) {
			return networkloadbalancersdk.DeleteBackendResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous delete")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for ambiguous NotAuthorizedOrNotFound")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want NotAuthorizedOrNotFound context", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, "opc-request-id")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt should stay empty for ambiguous NotAuthorizedOrNotFound")
	}
}

func TestCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(context.Context, networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			return networkloadbalancersdk.GetBackendResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing backend")
		},
		createFn: func(context.Context, networkloadbalancersdk.CreateBackendRequest) (networkloadbalancersdk.CreateBackendResponse, error) {
			return networkloadbalancersdk.CreateBackendResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
		},
	})

	resource := makeBackendResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, "opc-request-id")
	}
}

const (
	networkLoadBalancerID      = "ocid1.networkloadbalancer.oc1..exampleuniqueID"
	backendSetName             = "example_backend_set"
	backendIP                  = "10.0.0.3"
	backendPort                = 8080
	backendName                = "10.0.0.3:8080"
	backendTargetID            = "ocid1.privateip.oc1..exampleuniqueID"
	replacementBackendTargetID = "ocid1.privateip.oc1..replacement"
	targetBackendName          = "ocid1.privateip.oc1..exampleuniqueID:8080"
)

func makeBackendResource() *networkloadbalancerv1beta1.Backend {
	return &networkloadbalancerv1beta1.Backend{
		Spec: networkloadbalancerv1beta1.BackendSpec{
			NetworkLoadBalancerId: networkLoadBalancerID,
			BackendSetName:        backendSetName,
			IpAddress:             backendIP,
			Port:                  backendPort,
			Weight:                1,
		},
	}
}

func makeTrackedBackendResource() *networkloadbalancerv1beta1.Backend {
	resource := makeBackendResource()
	resource.Status = networkloadbalancerv1beta1.BackendStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(backendName),
		},
		NetworkLoadBalancerId: networkLoadBalancerID,
		BackendSetName:        backendSetName,
		Name:                  backendName,
		IpAddress:             backendIP,
		Port:                  backendPort,
		Weight:                1,
	}
	return resource
}

func makeTrackedTargetBackendResource() *networkloadbalancerv1beta1.Backend {
	resource := makeBackendResource()
	resource.Spec.IpAddress = ""
	resource.Spec.TargetId = backendTargetID
	resource.Status = networkloadbalancerv1beta1.BackendStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(targetBackendName),
		},
		NetworkLoadBalancerId: networkLoadBalancerID,
		BackendSetName:        backendSetName,
		Name:                  targetBackendName,
		TargetId:              backendTargetID,
		Port:                  backendPort,
		Weight:                1,
	}
	return resource
}

func seedPendingBackendDeleteWorkRequest(resource *networkloadbalancerv1beta1.Backend, workRequestID string) {
	seedPendingBackendWorkRequest(resource, shared.OSOKAsyncPhaseDelete, workRequestID)
}

func seedPendingBackendWorkRequest(resource *networkloadbalancerv1beta1.Backend, phase shared.OSOKAsyncPhase, workRequestID string) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
}

func sdkBackend(weight int, backup, drain, offline bool) networkloadbalancersdk.Backend {
	return networkloadbalancersdk.Backend{
		Name:      common.String(backendName),
		IpAddress: common.String(backendIP),
		Port:      common.Int(backendPort),
		Weight:    common.Int(weight),
		IsBackup:  common.Bool(backup),
		IsDrain:   common.Bool(drain),
		IsOffline: common.Bool(offline),
	}
}

func backendWorkRequestResponse(
	id string,
	status networkloadbalancersdk.OperationStatusEnum,
	operationType networkloadbalancersdk.OperationTypeEnum,
) networkloadbalancersdk.GetWorkRequestResponse {
	percentComplete := float32(100)
	return networkloadbalancersdk.GetWorkRequestResponse{
		WorkRequest: networkloadbalancersdk.WorkRequest{
			Id:              common.String(id),
			Status:          status,
			OperationType:   operationType,
			CompartmentId:   common.String("ocid1.compartment.oc1..exampleuniqueID"),
			PercentComplete: &percentComplete,
		},
	}
}

func assertBackendPathIdentity(t *testing.T, networkLoadBalancer, backendSet, backendNameValue *string) {
	t.Helper()

	if got := stringValue(networkLoadBalancer); got != networkLoadBalancerID {
		t.Fatalf("path networkLoadBalancerId = %q, want %q", got, networkLoadBalancerID)
	}
	if got := stringValue(backendSet); got != backendSetName {
		t.Fatalf("path backendSetName = %q, want %q", got, backendSetName)
	}
	if got := stringValue(backendNameValue); got != backendName {
		t.Fatalf("path backendName = %q, want %q", got, backendName)
	}
}

func assertBackendStatusIdentity(t *testing.T, resource *networkloadbalancerv1beta1.Backend) {
	t.Helper()

	if got := resource.Status.NetworkLoadBalancerId; got != networkLoadBalancerID {
		t.Fatalf("status.networkLoadBalancerId = %q, want %q", got, networkLoadBalancerID)
	}
	if got := resource.Status.BackendSetName; got != backendSetName {
		t.Fatalf("status.backendSetName = %q, want %q", got, backendSetName)
	}
	if got := resource.Status.Name; got != backendName {
		t.Fatalf("status.name = %q, want %q", got, backendName)
	}
}

func assertCreateBackendRequest(t *testing.T, req networkloadbalancersdk.CreateBackendRequest) {
	t.Helper()

	if got := stringValue(req.NetworkLoadBalancerId); got != networkLoadBalancerID {
		t.Fatalf("CreateBackendRequest.networkLoadBalancerId = %q, want %q", got, networkLoadBalancerID)
	}
	if got := stringValue(req.BackendSetName); got != backendSetName {
		t.Fatalf("CreateBackendRequest.backendSetName = %q, want %q", got, backendSetName)
	}
	if got := stringValue(req.IpAddress); got != backendIP {
		t.Fatalf("CreateBackendRequest.ipAddress = %q, want %q", got, backendIP)
	}
	if got := intValue(req.Port); got != backendPort {
		t.Fatalf("CreateBackendRequest.port = %d, want %d", got, backendPort)
	}
}

func assertMutableUpdateBackendRequest(t *testing.T, req networkloadbalancersdk.UpdateBackendRequest, wantWeight int) {
	t.Helper()

	assertBackendPathIdentity(t, req.NetworkLoadBalancerId, req.BackendSetName, req.BackendName)
	if got := intValue(req.Weight); got != wantWeight {
		t.Fatalf("UpdateBackendRequest.weight = %d, want %d", got, wantWeight)
	}
	assertBoolPointer(t, "UpdateBackendRequest.isBackup", req.IsBackup, false)
	assertBoolPointer(t, "UpdateBackendRequest.isDrain", req.IsDrain, false)
	assertBoolPointer(t, "UpdateBackendRequest.isOffline", req.IsOffline, false)
}

func assertMutableUpdateBackendStatus(t *testing.T, resource *networkloadbalancerv1beta1.Backend) {
	t.Helper()

	if got := resource.Status.Weight; got != resource.Spec.Weight {
		t.Fatalf("status.weight = %d, want %d", got, resource.Spec.Weight)
	}
	if resource.Status.IsBackup || resource.Status.IsDrain || resource.Status.IsOffline {
		t.Fatalf("status bool fields = backup:%t drain:%t offline:%t, want all false", resource.Status.IsBackup, resource.Status.IsDrain, resource.Status.IsOffline)
	}
}

func assertPendingBackendDelete(
	t *testing.T,
	resource *networkloadbalancerv1beta1.Backend,
	deleted bool,
	deleteCalls int,
	getCalls int,
) {
	t.Helper()

	if deleted {
		t.Fatal("Delete() first call = true, want pending delete requeue")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteBackend() calls after first delete = %d, want 1", deleteCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetBackend() calls after first delete = %d, want 1 pre-delete guard read", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt should stay empty while delete work request is pending")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want pending delete work request")
	}
	if current.Phase != shared.OSOKAsyncPhaseDelete || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current = %#v, want pending delete work request", current)
	}
}

func assertConfirmedBackendDelete(
	t *testing.T,
	resource *networkloadbalancerv1beta1.Backend,
	deleted bool,
	deleteCalls int,
	getCalls int,
) {
	t.Helper()

	if !deleted {
		t.Fatal("Delete() second call = false, want confirmed delete")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteBackend() calls after second delete = %d, want 1", deleteCalls)
	}
	if getCalls != 3 {
		t.Fatalf("GetBackend() calls after second delete = %d, want 3 including guard and generatedruntime confirmation reads", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt should be set after confirmed delete")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after confirmed delete", resource.Status.OsokStatus.Async.Current)
	}
}

func assertCurrentBackendDeleteWorkRequest(
	t *testing.T,
	resource *networkloadbalancerv1beta1.Backend,
	workRequestID string,
) {
	t.Helper()
	assertCurrentBackendWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, workRequestID)
}

func assertCurrentBackendWorkRequest(
	t *testing.T,
	resource *networkloadbalancerv1beta1.Backend,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want pending delete work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest ||
		current.Phase != phase ||
		current.NormalizedClass != shared.OSOKAsyncClassPending ||
		current.WorkRequestID != workRequestID {
		t.Fatalf("status.status.async.current = %#v, want pending %s work request %q", current, phase, workRequestID)
	}
}

func runBackendPendingWriteWorkRequestDeleteTest(
	t *testing.T,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	operationType networkloadbalancersdk.OperationTypeEnum,
) {
	t.Helper()

	resource := makeTrackedBackendResource()
	seedPendingBackendWorkRequest(resource, phase, workRequestID)

	getCalls := 0
	deleteCalls := 0
	workRequestCalls := 0
	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(context.Context, networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			getCalls++
			return networkloadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, false, false, false)}, nil
		},
		deleteFn: func(context.Context, networkloadbalancersdk.DeleteBackendRequest) (networkloadbalancersdk.DeleteBackendResponse, error) {
			deleteCalls++
			return networkloadbalancersdk.DeleteBackendResponse{}, nil
		},
		getWorkRequestFn: func(_ context.Context, req networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			if got := stringValue(req.WorkRequestId); got != workRequestID {
				t.Fatalf("GetWorkRequestRequest.workRequestId = %q, want %q", got, workRequestID)
			}
			return backendWorkRequestResponse(workRequestID, networkloadbalancersdk.OperationStatusInProgress, operationType), nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatalf("Delete() = true, want pending %s work request to keep finalizer", phase)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteBackend() calls = %d, want 0 while %s work request is pending", deleteCalls, phase)
	}
	if getCalls != 0 {
		t.Fatalf("GetBackend() calls = %d, want 0 while %s work request is pending", getCalls, phase)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1 pending %s poll", workRequestCalls, phase)
	}
	assertCurrentBackendWorkRequest(t, resource, phase, workRequestID)
}

func assertBoolPointer(t *testing.T, name string, value *bool, want bool) {
	t.Helper()

	if value == nil {
		t.Fatalf("%s = nil, want %t pointer", name, want)
	}
	if *value != want {
		t.Fatalf("%s = %t, want %t", name, *value, want)
	}
}

func assertBackendAddressDriftRejectedBeforeObserve(
	t *testing.T,
	resource *networkloadbalancerv1beta1.Backend,
	wantErr string,
) {
	t.Helper()

	getCalled := false
	createCalled := false
	updateCalled := false
	client := newTestBackendRuntimeClient(&fakeBackendOCIClient{
		getFn: func(context.Context, networkloadbalancersdk.GetBackendRequest) (networkloadbalancersdk.GetBackendResponse, error) {
			getCalled = true
			return networkloadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, false, false, false)}, nil
		},
		createFn: func(context.Context, networkloadbalancersdk.CreateBackendRequest) (networkloadbalancersdk.CreateBackendResponse, error) {
			createCalled = true
			return networkloadbalancersdk.CreateBackendResponse{}, nil
		},
		updateFn: func(context.Context, networkloadbalancersdk.UpdateBackendRequest) (networkloadbalancersdk.UpdateBackendResponse, error) {
			updateCalled = true
			return networkloadbalancersdk.UpdateBackendResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("CreateOrUpdate() error = %v, want %q", err, wantErr)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if getCalled {
		t.Fatal("GetBackend() called, want address drift rejected before observe")
	}
	if createCalled {
		t.Fatal("CreateBackend() called, want address drift rejected before create")
	}
	if updateCalled {
		t.Fatal("UpdateBackend() called, want address drift rejected before update")
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

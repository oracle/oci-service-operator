package backend

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeGeneratedBackendOCIClient struct {
	createFn func(context.Context, loadbalancersdk.CreateBackendRequest) (loadbalancersdk.CreateBackendResponse, error)
	getFn    func(context.Context, loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error)
	listFn   func(context.Context, loadbalancersdk.ListBackendsRequest) (loadbalancersdk.ListBackendsResponse, error)
	updateFn func(context.Context, loadbalancersdk.UpdateBackendRequest) (loadbalancersdk.UpdateBackendResponse, error)
	deleteFn func(context.Context, loadbalancersdk.DeleteBackendRequest) (loadbalancersdk.DeleteBackendResponse, error)
}

type backendLookupClient interface {
	GetBackend(context.Context, loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error)
}

func (f *fakeGeneratedBackendOCIClient) CreateBackend(ctx context.Context, req loadbalancersdk.CreateBackendRequest) (loadbalancersdk.CreateBackendResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return loadbalancersdk.CreateBackendResponse{}, nil
}

func (f *fakeGeneratedBackendOCIClient) GetBackend(ctx context.Context, req loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return loadbalancersdk.GetBackendResponse{}, errortest.NewServiceError(404, "NotFound", "missing backend")
}

func (f *fakeGeneratedBackendOCIClient) ListBackends(ctx context.Context, req loadbalancersdk.ListBackendsRequest) (loadbalancersdk.ListBackendsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return loadbalancersdk.ListBackendsResponse{}, nil
}

func (f *fakeGeneratedBackendOCIClient) UpdateBackend(ctx context.Context, req loadbalancersdk.UpdateBackendRequest) (loadbalancersdk.UpdateBackendResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return loadbalancersdk.UpdateBackendResponse{}, nil
}

func (f *fakeGeneratedBackendOCIClient) DeleteBackend(ctx context.Context, req loadbalancersdk.DeleteBackendRequest) (loadbalancersdk.DeleteBackendResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return loadbalancersdk.DeleteBackendResponse{}, nil
}

func newTestGeneratedBackendDelegate(client *fakeGeneratedBackendOCIClient) BackendServiceClient {
	hooks := newBackendRuntimeHooksWithOCIClient(client)
	applyBackendRuntimeHooks(&hooks)
	config := buildBackendGeneratedRuntimeConfig(&BackendServiceManager{}, hooks)

	return defaultBackendServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.Backend](config),
	}
}

func newTestBackendRuntimeClient(client *fakeGeneratedBackendOCIClient) BackendServiceClient {
	return newTestBackendRuntimeClientWithLookup(client, nil)
}

func newTestBackendRuntimeClientWithLookup(client *fakeGeneratedBackendOCIClient, lookup backendLookupClient) BackendServiceClient {
	hooks := newBackendRuntimeHooksWithOCIClient(client)
	applyBackendRuntimeHooks(&hooks)
	if lookup != nil {
		hooks.Identity.LookupExisting = func(ctx context.Context, _ *loadbalancerv1beta1.Backend, identity any) (any, error) {
			resolved := identity.(backendIdentity)
			return lookup.GetBackend(ctx, loadbalancersdk.GetBackendRequest{
				LoadBalancerId: common.String(resolved.loadBalancerID),
				BackendSetName: common.String(resolved.backendSetName),
				BackendName:    common.String(resolved.backendName),
			})
		}
	}

	config := buildBackendGeneratedRuntimeConfig(&BackendServiceManager{}, hooks)
	return defaultBackendServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.Backend](config),
	}
}

func TestCreateOrUpdateExecutesMutableBackendUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendResource()
	resource.Spec.Weight = 5
	resource.Spec.Backup = true
	resource.Spec.Drain = true
	resource.Spec.Offline = true

	getCalls := 0
	var updateRequest loadbalancersdk.UpdateBackendRequest

	client := newTestBackendRuntimeClient(&fakeGeneratedBackendOCIClient{
		getFn: func(_ context.Context, req loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error) {
			getCalls++
			assertBackendPathIdentity(t, req.LoadBalancerId, req.BackendSetName, req.BackendName)
			if getCalls == 1 {
				return loadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, false, false, false)}, nil
			}
			return loadbalancersdk.GetBackendResponse{Backend: sdkBackend(resource.Spec.Weight, resource.Spec.Backup, resource.Spec.Drain, resource.Spec.Offline)}, nil
		},
		updateFn: func(_ context.Context, req loadbalancersdk.UpdateBackendRequest) (loadbalancersdk.UpdateBackendResponse, error) {
			updateRequest = req
			return loadbalancersdk.UpdateBackendResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update response", response)
	}
	assertBackendPathIdentity(t, updateRequest.LoadBalancerId, updateRequest.BackendSetName, updateRequest.BackendName)
	if got := intValue(updateRequest.UpdateBackendDetails.Weight); got != resource.Spec.Weight {
		t.Fatalf("UpdateBackendRequest.Weight = %d, want %d", got, resource.Spec.Weight)
	}
	if got := boolValue(updateRequest.UpdateBackendDetails.Backup); got != resource.Spec.Backup {
		t.Fatalf("UpdateBackendRequest.Backup = %t, want %t", got, resource.Spec.Backup)
	}
	if got := boolValue(updateRequest.UpdateBackendDetails.Drain); got != resource.Spec.Drain {
		t.Fatalf("UpdateBackendRequest.Drain = %t, want %t", got, resource.Spec.Drain)
	}
	if got := boolValue(updateRequest.UpdateBackendDetails.Offline); got != resource.Spec.Offline {
		t.Fatalf("UpdateBackendRequest.Offline = %t, want %t", got, resource.Spec.Offline)
	}
	if got := resource.Status.Weight; got != resource.Spec.Weight {
		t.Fatalf("status.weight = %d, want %d", got, resource.Spec.Weight)
	}
	if got := resource.Status.Backup; got != resource.Spec.Backup {
		t.Fatalf("status.backup = %t, want %t", got, resource.Spec.Backup)
	}
	if got := resource.Status.Drain; got != resource.Spec.Drain {
		t.Fatalf("status.drain = %t, want %t", got, resource.Spec.Drain)
	}
	if got := resource.Status.Offline; got != resource.Spec.Offline {
		t.Fatalf("status.offline = %t, want %t", got, resource.Spec.Offline)
	}
	if got := resource.Status.Name; got != backendName {
		t.Fatalf("status.name = %q, want %q", got, backendName)
	}
	if got := resource.Status.LoadBalancerId; got != loadBalancerID {
		t.Fatalf("status.loadBalancerId = %q, want %q", got, loadBalancerID)
	}
	if got := resource.Status.BackendSetName; got != backendSetName {
		t.Fatalf("status.backendSetName = %q, want %q", got, backendSetName)
	}
}

func TestCreateOrUpdateRejectsForceNewBackendDrift(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		mutate  func(*loadbalancerv1beta1.Backend)
		wantErr string
	}{
		{
			name: "backendSetName",
			mutate: func(resource *loadbalancerv1beta1.Backend) {
				resource.Spec.BackendSetName = "replacement_backend_set"
			},
			wantErr: "require replacement when backendSetName changes",
		},
		{
			name: "ipAddress",
			mutate: func(resource *loadbalancerv1beta1.Backend) {
				resource.Spec.IpAddress = "10.0.0.9"
			},
			wantErr: "require replacement when ipAddress changes",
		},
		{
			name: "loadBalancerId",
			mutate: func(resource *loadbalancerv1beta1.Backend) {
				resource.Spec.LoadBalancerId = "ocid1.loadbalancer.oc1..replacement"
			},
			wantErr: "require replacement when loadBalancerId changes",
		},
		{
			name: "port",
			mutate: func(resource *loadbalancerv1beta1.Backend) {
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
			client := newTestBackendRuntimeClient(&fakeGeneratedBackendOCIClient{
				getFn: func(_ context.Context, req loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error) {
					assertBackendPathIdentity(t, req.LoadBalancerId, req.BackendSetName, req.BackendName)
					return loadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, false, false, false)}, nil
				},
				updateFn: func(context.Context, loadbalancersdk.UpdateBackendRequest) (loadbalancersdk.UpdateBackendResponse, error) {
					updateCalled = true
					return loadbalancersdk.UpdateBackendResponse{}, nil
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

func TestDeleteConfirmsBackendRemovalBeforeReportingSuccess(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendResource()
	resource.Spec.LoadBalancerId = "ocid1.loadbalancer.oc1..replacement"
	resource.Spec.BackendSetName = "replacement_backend_set"
	resource.Spec.IpAddress = "10.0.0.9"
	resource.Spec.Port = 9090

	getCalls := 0
	deleteCalls := 0

	client := newTestBackendRuntimeClient(&fakeGeneratedBackendOCIClient{
		getFn: func(_ context.Context, req loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error) {
			getCalls++
			assertBackendPathIdentity(t, req.LoadBalancerId, req.BackendSetName, req.BackendName)
			if getCalls < 3 {
				return loadbalancersdk.GetBackendResponse{
					Backend: sdkBackend(resource.Status.Weight, resource.Status.Backup, resource.Status.Drain, resource.Status.Offline),
				}, nil
			}
			return loadbalancersdk.GetBackendResponse{}, errortest.NewServiceError(404, "NotFound", "missing backend")
		},
		deleteFn: func(_ context.Context, req loadbalancersdk.DeleteBackendRequest) (loadbalancersdk.DeleteBackendResponse, error) {
			deleteCalls++
			assertBackendPathIdentity(t, req.LoadBalancerId, req.BackendSetName, req.BackendName)
			return loadbalancersdk.DeleteBackendResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call = true, want delete confirmation requeue while Backend still exists")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteBackend() calls after first delete = %d, want 1", deleteCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetBackend() calls after first delete = %d, want 2 (pre-delete read plus confirm read)", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay empty while delete confirmation is pending")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason after first delete = %q, want %q", got, shared.Terminating)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call = false, want terminal delete confirmation after NotFound reread")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteBackend() calls after second delete = %d, want 1", deleteCalls)
	}
	if getCalls != 3 {
		t.Fatalf("GetBackend() calls after second delete = %d, want 3", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed delete")
	}
}

const (
	loadBalancerID = "ocid1.loadbalancer.oc1..exampleuniqueID"
	backendSetName = "example_backend_set"
	backendIP      = "10.0.0.3"
	backendPort    = 8080
	backendName    = "10.0.0.3:8080"
)

func makeTrackedBackendResource() *loadbalancerv1beta1.Backend {
	return &loadbalancerv1beta1.Backend{
		Spec: loadbalancerv1beta1.BackendSpec{
			LoadBalancerId: loadBalancerID,
			BackendSetName: backendSetName,
			IpAddress:      backendIP,
			Port:           backendPort,
			Weight:         1,
		},
		Status: loadbalancerv1beta1.BackendStatus{
			Name:           backendName,
			LoadBalancerId: loadBalancerID,
			BackendSetName: backendSetName,
			IpAddress:      backendIP,
			Port:           backendPort,
			Weight:         1,
		},
	}
}

func sdkBackend(weight int, backup, drain, offline bool) loadbalancersdk.Backend {
	return loadbalancersdk.Backend{
		Name:      common.String(backendName),
		IpAddress: common.String(backendIP),
		Port:      common.Int(backendPort),
		Weight:    common.Int(weight),
		Backup:    common.Bool(backup),
		Drain:     common.Bool(drain),
		Offline:   common.Bool(offline),
	}
}

func assertBackendPathIdentity(t *testing.T, loadBalancer, backendSet, backendNameValue *string) {
	t.Helper()

	if got := stringValue(loadBalancer); got != loadBalancerID {
		t.Fatalf("path loadBalancerId = %q, want %q", got, loadBalancerID)
	}
	if got := stringValue(backendSet); got != backendSetName {
		t.Fatalf("path backendSetName = %q, want %q", got, backendSetName)
	}
	if got := stringValue(backendNameValue); got != backendName {
		t.Fatalf("path backendName = %q, want %q", got, backendName)
	}
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

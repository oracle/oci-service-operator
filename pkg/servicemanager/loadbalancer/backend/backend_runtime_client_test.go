package backend

import (
	"context"
	"testing"

	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeBackendLookupClient struct {
	requests []loadbalancersdk.GetBackendRequest
	response loadbalancersdk.GetBackendResponse
	err      error
}

func (f *fakeBackendLookupClient) GetBackend(_ context.Context, request loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error) {
	f.requests = append(f.requests, request)
	if f.err != nil {
		return loadbalancersdk.GetBackendResponse{}, f.err
	}
	return f.response, nil
}

func TestCreateOrUpdateBindsExistingBackend(t *testing.T) {
	t.Parallel()

	lookup := &fakeBackendLookupClient{
		response: loadbalancersdk.GetBackendResponse{
			Backend: sdkBackend(1, false, false, false),
		},
	}
	createCalled := false
	updateCalled := false
	client := newTestBackendRuntimeClientWithLookup(&fakeGeneratedBackendOCIClient{
		createFn: func(context.Context, loadbalancersdk.CreateBackendRequest) (loadbalancersdk.CreateBackendResponse, error) {
			createCalled = true
			return loadbalancersdk.CreateBackendResponse{}, nil
		},
		updateFn: func(context.Context, loadbalancersdk.UpdateBackendRequest) (loadbalancersdk.UpdateBackendResponse, error) {
			updateCalled = true
			return loadbalancersdk.UpdateBackendResponse{}, nil
		},
	}, lookup)

	resource := &loadbalancerv1beta1.Backend{
		Spec: loadbalancerv1beta1.BackendSpec{
			LoadBalancerId: "ocid1.loadbalancer.oc1..exampleuniqueID",
			BackendSetName: "example_backend_set",
			IpAddress:      "10.0.0.3",
			Port:           8080,
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if createCalled {
		t.Fatal("CreateBackend() called, want bind/observe path")
	}
	if updateCalled {
		t.Fatal("UpdateBackend() called, want observe-only bind path")
	}
	if got := resource.Status.OsokStatus.Ocid; got != "" {
		t.Fatalf("status.status.ocid = %q, want empty after synthetic ID restore", got)
	}
	if got := resource.Status.LoadBalancerId; got != resource.Spec.LoadBalancerId {
		t.Fatalf("status.loadBalancerId = %q, want %q", got, resource.Spec.LoadBalancerId)
	}
	if got := resource.Status.BackendSetName; got != resource.Spec.BackendSetName {
		t.Fatalf("status.backendSetName = %q, want %q", got, resource.Spec.BackendSetName)
	}
	if got := resource.Status.Name; got != "10.0.0.3:8080" {
		t.Fatalf("status.name = %q, want %q", got, "10.0.0.3:8080")
	}
	if len(lookup.requests) != 1 {
		t.Fatalf("lookup requests = %d, want 1", len(lookup.requests))
	}
	if got := stringValue(lookup.requests[0].BackendName); got != "10.0.0.3:8080" {
		t.Fatalf("lookup backendName = %q, want %q", got, "10.0.0.3:8080")
	}
}

func TestCreateOrUpdateCreatesWhenBackendIsMissing(t *testing.T) {
	t.Parallel()

	lookup := &fakeBackendLookupClient{
		err: errortest.NewServiceError(404, "NotFound", "missing backend"),
	}
	createCalled := false
	client := newTestBackendRuntimeClientWithLookup(&fakeGeneratedBackendOCIClient{
		createFn: func(_ context.Context, req loadbalancersdk.CreateBackendRequest) (loadbalancersdk.CreateBackendResponse, error) {
			createCalled = true
			if got := stringValue(req.LoadBalancerId); got != "ocid1.loadbalancer.oc1..exampleuniqueID" {
				t.Fatalf("create request loadBalancerId = %q, want %q", got, "ocid1.loadbalancer.oc1..exampleuniqueID")
			}
			if got := stringValue(req.BackendSetName); got != "example_backend_set" {
				t.Fatalf("create request backendSetName = %q, want %q", got, "example_backend_set")
			}
			return loadbalancersdk.CreateBackendResponse{}, nil
		},
	}, lookup)

	resource := &loadbalancerv1beta1.Backend{
		Spec: loadbalancerv1beta1.BackendSpec{
			LoadBalancerId: "ocid1.loadbalancer.oc1..exampleuniqueID",
			BackendSetName: "example_backend_set",
			IpAddress:      "10.0.0.3",
			Port:           8080,
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if !createCalled {
		t.Fatal("CreateBackend() not called, want create path after missing lookup")
	}
	if got := resource.Status.OsokStatus.Ocid; got != "" {
		t.Fatalf("status.status.ocid = %q, want empty for create path", got)
	}
	if got := resource.Status.LoadBalancerId; got != resource.Spec.LoadBalancerId {
		t.Fatalf("status.loadBalancerId = %q, want %q", got, resource.Spec.LoadBalancerId)
	}
	if got := resource.Status.BackendSetName; got != resource.Spec.BackendSetName {
		t.Fatalf("status.backendSetName = %q, want %q", got, resource.Spec.BackendSetName)
	}
}

func TestDeleteUsesBoundStatusIdentity(t *testing.T) {
	t.Parallel()

	var deleteRequest loadbalancersdk.DeleteBackendRequest
	getCalls := 0
	client := newTestBackendRuntimeClient(&fakeGeneratedBackendOCIClient{
		getFn: func(_ context.Context, req loadbalancersdk.GetBackendRequest) (loadbalancersdk.GetBackendResponse, error) {
			getCalls++
			if got := stringValue(req.LoadBalancerId); got != "ocid1.loadbalancer.oc1..old" {
				t.Fatalf("get request loadBalancerId = %q, want %q", got, "ocid1.loadbalancer.oc1..old")
			}
			if got := stringValue(req.BackendSetName); got != "old_backend_set" {
				t.Fatalf("get request backendSetName = %q, want %q", got, "old_backend_set")
			}
			if got := stringValue(req.BackendName); got != "10.0.0.3:8080" {
				t.Fatalf("get request backendName = %q, want %q", got, "10.0.0.3:8080")
			}
			if getCalls == 1 {
				return loadbalancersdk.GetBackendResponse{Backend: sdkBackend(1, false, false, false)}, nil
			}
			return loadbalancersdk.GetBackendResponse{}, errortest.NewServiceError(404, "NotFound", "missing backend")
		},
		deleteFn: func(_ context.Context, req loadbalancersdk.DeleteBackendRequest) (loadbalancersdk.DeleteBackendResponse, error) {
			deleteRequest = req
			return loadbalancersdk.DeleteBackendResponse{}, nil
		},
	})

	resource := &loadbalancerv1beta1.Backend{
		Spec: loadbalancerv1beta1.BackendSpec{
			LoadBalancerId: "ocid1.loadbalancer.oc1..new",
			BackendSetName: "new_backend_set",
			IpAddress:      "10.0.0.9",
			Port:           9090,
		},
		Status: loadbalancerv1beta1.BackendStatus{
			Name:           "10.0.0.3:8080",
			LoadBalancerId: "ocid1.loadbalancer.oc1..old",
			BackendSetName: "old_backend_set",
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true")
	}
	if got := stringValue(deleteRequest.LoadBalancerId); got != "ocid1.loadbalancer.oc1..old" {
		t.Fatalf("delete request loadBalancerId = %q, want %q", got, "ocid1.loadbalancer.oc1..old")
	}
	if got := stringValue(deleteRequest.BackendSetName); got != "old_backend_set" {
		t.Fatalf("delete request backendSetName = %q, want %q", got, "old_backend_set")
	}
	if got := stringValue(deleteRequest.BackendName); got != "10.0.0.3:8080" {
		t.Fatalf("delete request backendName = %q, want %q", got, "10.0.0.3:8080")
	}
	if got := resource.Status.LoadBalancerId; got != "ocid1.loadbalancer.oc1..old" {
		t.Fatalf("status.loadBalancerId = %q, want %q", got, "ocid1.loadbalancer.oc1..old")
	}
	if got := resource.Status.BackendSetName; got != "old_backend_set" {
		t.Fatalf("status.backendSetName = %q, want %q", got, "old_backend_set")
	}
	if got := resource.Status.OsokStatus.Ocid; got != "" {
		t.Fatalf("status.status.ocid = %q, want empty after synthetic ID restore", got)
	}
}

func TestCreateOrUpdateRejectsMissingPathIdentity(t *testing.T) {
	t.Parallel()

	client := newTestBackendRuntimeClient(&fakeGeneratedBackendOCIClient{})

	resource := &loadbalancerv1beta1.Backend{
		Spec: loadbalancerv1beta1.BackendSpec{
			IpAddress: "10.0.0.3",
			Port:      8080,
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing loadBalancerId error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful response", response)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

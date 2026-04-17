package backend

import (
	"context"
	"testing"

	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
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

type fakeBackendDelegate struct {
	createCalls            int
	deleteCalls            int
	receivedSyntheticOCID  shared.OCID
	receivedLoadBalancerID string
	receivedBackendSetName string
	createResponse         servicemanager.OSOKResponse
	createErr              error
	deleteResponse         bool
	deleteErr              error
}

func (f *fakeBackendDelegate) CreateOrUpdate(_ context.Context, resource *loadbalancerv1beta1.Backend, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	f.createCalls++
	f.receivedSyntheticOCID = resource.Status.OsokStatus.Ocid
	f.receivedLoadBalancerID = resource.Status.LoadBalancerId
	f.receivedBackendSetName = resource.Status.BackendSetName
	return f.createResponse, f.createErr
}

func (f *fakeBackendDelegate) Delete(_ context.Context, resource *loadbalancerv1beta1.Backend) (bool, error) {
	f.deleteCalls++
	f.receivedSyntheticOCID = resource.Status.OsokStatus.Ocid
	f.receivedLoadBalancerID = resource.Status.LoadBalancerId
	f.receivedBackendSetName = resource.Status.BackendSetName
	return f.deleteResponse, f.deleteErr
}

func TestCreateOrUpdateBindsExistingBackend(t *testing.T) {
	t.Parallel()

	lookup := &fakeBackendLookupClient{}
	delegate := &fakeBackendDelegate{
		createResponse: servicemanager.OSOKResponse{IsSuccessful: true},
	}
	client := &backendRuntimeServiceClient{
		delegate: delegate,
		lookup:   lookup,
	}

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
	if delegate.createCalls != 1 {
		t.Fatalf("delegate create calls = %d, want 1", delegate.createCalls)
	}
	if got := delegate.receivedSyntheticOCID; got != shared.OCID("10.0.0.3:8080") {
		t.Fatalf("delegate synthetic OCID = %q, want %q", got, "10.0.0.3:8080")
	}
	if got := resource.Status.OsokStatus.Ocid; got != "" {
		t.Fatalf("status.status.ocid = %q, want empty after wrapper restore", got)
	}
	if got := resource.Status.LoadBalancerId; got != resource.Spec.LoadBalancerId {
		t.Fatalf("status.loadBalancerId = %q, want %q", got, resource.Spec.LoadBalancerId)
	}
	if got := resource.Status.BackendSetName; got != resource.Spec.BackendSetName {
		t.Fatalf("status.backendSetName = %q, want %q", got, resource.Spec.BackendSetName)
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
	delegate := &fakeBackendDelegate{
		createResponse: servicemanager.OSOKResponse{IsSuccessful: true},
	}
	client := &backendRuntimeServiceClient{
		delegate: delegate,
		lookup:   lookup,
	}

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
	if got := delegate.receivedSyntheticOCID; got != "" {
		t.Fatalf("delegate synthetic OCID = %q, want empty for create path", got)
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

	delegate := &fakeBackendDelegate{deleteResponse: true}
	client := &backendRuntimeServiceClient{delegate: delegate}

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
	if delegate.deleteCalls != 1 {
		t.Fatalf("delegate delete calls = %d, want 1", delegate.deleteCalls)
	}
	if got := delegate.receivedSyntheticOCID; got != shared.OCID("10.0.0.3:8080") {
		t.Fatalf("delegate synthetic OCID = %q, want %q", got, "10.0.0.3:8080")
	}
	if got := delegate.receivedLoadBalancerID; got != "ocid1.loadbalancer.oc1..old" {
		t.Fatalf("delegate status.loadBalancerId = %q, want %q", got, "ocid1.loadbalancer.oc1..old")
	}
	if got := delegate.receivedBackendSetName; got != "old_backend_set" {
		t.Fatalf("delegate status.backendSetName = %q, want %q", got, "old_backend_set")
	}
	if got := resource.Status.OsokStatus.Ocid; got != "" {
		t.Fatalf("status.status.ocid = %q, want empty after wrapper restore", got)
	}
}

func TestCreateOrUpdateRejectsMissingPathIdentity(t *testing.T) {
	t.Parallel()

	client := &backendRuntimeServiceClient{
		delegate: &fakeBackendDelegate{
			createResponse: servicemanager.OSOKResponse{IsSuccessful: true},
		},
	}

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

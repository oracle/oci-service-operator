package bucket

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	objectstoragesdk "github.com/oracle/oci-go-sdk/v65/objectstorage"
	objectstoragev1beta1 "github.com/oracle/oci-service-operator/api/objectstorage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeBucketNamespaceGetter struct {
	requests []objectstoragesdk.GetNamespaceRequest
	response string
	err      error
}

func (f *fakeBucketNamespaceGetter) GetNamespace(_ context.Context, request objectstoragesdk.GetNamespaceRequest) (objectstoragesdk.GetNamespaceResponse, error) {
	f.requests = append(f.requests, request)
	if f.err != nil {
		return objectstoragesdk.GetNamespaceResponse{}, f.err
	}
	return objectstoragesdk.GetNamespaceResponse{Value: common.String(f.response)}, nil
}

type fakeBucketDelegate struct {
	createCalled      int
	deleteCalled      int
	receivedNamespace string
	response          servicemanager.OSOKResponse
	err               error
}

func (f *fakeBucketDelegate) CreateOrUpdate(_ context.Context, resource *objectstoragev1beta1.Bucket, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	f.createCalled++
	f.receivedNamespace = resource.Status.Namespace
	if f.receivedNamespace == "" {
		f.receivedNamespace = resource.Spec.Namespace
	}
	return f.response, f.err
}

func (f *fakeBucketDelegate) Delete(_ context.Context, resource *objectstoragev1beta1.Bucket) (bool, error) {
	f.deleteCalled++
	f.receivedNamespace = resource.Status.Namespace
	if f.receivedNamespace == "" {
		f.receivedNamespace = resource.Spec.Namespace
	}
	return true, f.err
}

func TestNamespaceResolvingBucketServiceClientCreateOrUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		specNamespace        string
		statusNamespace      string
		wantNamespace        string
		wantNamespaceLookups int
	}{
		{name: "resolves namespace from OCI", wantNamespace: "tenantnamespace", wantNamespaceLookups: 1},
		{name: "reuses status namespace", statusNamespace: "statusnamespace", wantNamespace: "statusnamespace", wantNamespaceLookups: 0},
		{name: "keeps spec namespace", specNamespace: "specnamespace", wantNamespace: "specnamespace", wantNamespaceLookups: 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			namespaceGetter := &fakeBucketNamespaceGetter{response: "tenantnamespace"}
			delegate := &fakeBucketDelegate{
				response: servicemanager.OSOKResponse{IsSuccessful: true},
			}
			client := &namespaceResolvingBucketServiceClient{
				delegate:        delegate,
				namespaceGetter: namespaceGetter,
			}
			resource := &objectstoragev1beta1.Bucket{
				Spec: objectstoragev1beta1.BucketSpec{
					Name:          "bucket-name",
					CompartmentId: "ocid1.compartment.oc1..exampleuniqueID",
					Namespace:     tc.specNamespace,
				},
				Status: objectstoragev1beta1.BucketStatus{Namespace: tc.statusNamespace},
			}

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %+v, want success", response)
			}
			if got := len(namespaceGetter.requests); got != tc.wantNamespaceLookups {
				t.Fatalf("namespace lookups = %d, want %d", got, tc.wantNamespaceLookups)
			}
			if tc.wantNamespaceLookups == 1 {
				got := ""
				if namespaceGetter.requests[0].CompartmentId != nil {
					got = *namespaceGetter.requests[0].CompartmentId
				}
				if got != resource.Spec.CompartmentId {
					t.Fatalf("lookup compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
				}
				if got := resource.Status.Namespace; got != tc.wantNamespace {
					t.Fatalf("status namespace = %q, want %q", got, tc.wantNamespace)
				}
				if resource.Spec.Namespace != "" {
					t.Fatalf("spec namespace = %q, want empty string", resource.Spec.Namespace)
				}
				if got := delegate.receivedNamespace; got != tc.wantNamespace {
					t.Fatalf("delegate namespace = %q, want %q", got, tc.wantNamespace)
				}
				return
			}

			if got := delegate.receivedNamespace; got != tc.wantNamespace {
				t.Fatalf("delegate namespace = %q, want %q", got, tc.wantNamespace)
			}
		})
	}
}

func TestNamespaceResolvingBucketServiceClientDelete(t *testing.T) {
	t.Parallel()

	namespaceGetter := &fakeBucketNamespaceGetter{response: "tenantnamespace"}
	delegate := &fakeBucketDelegate{}
	client := &namespaceResolvingBucketServiceClient{
		delegate:        delegate,
		namespaceGetter: namespaceGetter,
	}
	resource := &objectstoragev1beta1.Bucket{
		Spec: objectstoragev1beta1.BucketSpec{
			Name:          "bucket-name",
			CompartmentId: "ocid1.compartment.oc1..exampleuniqueID",
		},
	}

	done, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !done {
		t.Fatal("Delete() = false, want true")
	}
	if delegate.deleteCalled != 1 {
		t.Fatalf("delegate delete calls = %d, want 1", delegate.deleteCalled)
	}
	if got := delegate.receivedNamespace; got != "tenantnamespace" {
		t.Fatalf("delegate namespace = %q, want tenantnamespace", got)
	}
}

func TestNamespaceResolvingBucketServiceClientLookupError(t *testing.T) {
	t.Parallel()

	namespaceGetter := &fakeBucketNamespaceGetter{err: errors.New("lookup failed")}
	delegate := &fakeBucketDelegate{
		response: servicemanager.OSOKResponse{IsSuccessful: true},
	}
	client := &namespaceResolvingBucketServiceClient{
		delegate:        delegate,
		namespaceGetter: namespaceGetter,
	}
	resource := &objectstoragev1beta1.Bucket{
		Spec: objectstoragev1beta1.BucketSpec{
			Name:          "bucket-name",
			CompartmentId: "ocid1.compartment.oc1..exampleuniqueID",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want namespace lookup error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful response", response)
	}
	if delegate.createCalled != 0 {
		t.Fatalf("delegate create calls = %d, want 0", delegate.createCalled)
	}
}

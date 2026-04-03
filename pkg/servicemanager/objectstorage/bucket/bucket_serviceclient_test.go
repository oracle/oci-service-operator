//go:build legacyservicemanager

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

type fakeNamespaceGetter struct {
	requests []objectstoragesdk.GetNamespaceRequest
	response string
	err      error
}

func (f *fakeNamespaceGetter) GetNamespace(_ context.Context, request objectstoragesdk.GetNamespaceRequest) (objectstoragesdk.GetNamespaceResponse, error) {
	f.requests = append(f.requests, request)
	if f.err != nil {
		return objectstoragesdk.GetNamespaceResponse{}, f.err
	}
	return objectstoragesdk.GetNamespaceResponse{Value: common.String(f.response)}, nil
}

type fakeRuntimeBucketClient struct {
	createCalled      int
	receivedNamespace string
	response          servicemanager.OSOKResponse
	err               error
}

func (f *fakeRuntimeBucketClient) CreateOrUpdate(_ context.Context, resource *objectstoragev1beta1.Bucket, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	f.createCalled++
	f.receivedNamespace = resource.Spec.Namespace
	return f.response, f.err
}

func (f *fakeRuntimeBucketClient) Delete(_ context.Context, resource *objectstoragev1beta1.Bucket) (bool, error) {
	f.receivedNamespace = resource.Spec.Namespace
	return true, f.err
}

func TestBucketServiceClientNamespaceResolution(t *testing.T) {
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

			namespaceGetter := &fakeNamespaceGetter{response: "tenantnamespace"}
			delegate := &fakeRuntimeBucketClient{
				response: servicemanager.OSOKResponse{IsSuccessful: true},
			}
			client := defaultBucketServiceClient{
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
			if got := delegate.receivedNamespace; got != tc.wantNamespace {
				t.Fatalf("delegate namespace = %q, want %q", got, tc.wantNamespace)
			}
			if got := len(namespaceGetter.requests); got != tc.wantNamespaceLookups {
				t.Fatalf("namespace lookups = %d, want %d", got, tc.wantNamespaceLookups)
			}
		})
	}
}

func TestBucketServiceClientNamespaceLookupError(t *testing.T) {
	t.Parallel()

	namespaceGetter := &fakeNamespaceGetter{err: errors.New("lookup failed")}
	delegate := &fakeRuntimeBucketClient{
		response: servicemanager.OSOKResponse{IsSuccessful: true},
	}
	client := defaultBucketServiceClient{
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

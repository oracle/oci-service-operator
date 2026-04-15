package bucket

import (
	"context"
	"testing"

	objectstoragesdk "github.com/oracle/oci-go-sdk/v65/objectstorage"
	objectstoragev1beta1 "github.com/oracle/oci-service-operator/api/objectstorage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeBucketRuntimeOCIClient struct {
	createFn func(context.Context, objectstoragesdk.CreateBucketRequest) (objectstoragesdk.CreateBucketResponse, error)
	getFn    func(context.Context, objectstoragesdk.GetBucketRequest) (objectstoragesdk.GetBucketResponse, error)
	listFn   func(context.Context, objectstoragesdk.ListBucketsRequest) (objectstoragesdk.ListBucketsResponse, error)
	updateFn func(context.Context, objectstoragesdk.UpdateBucketRequest) (objectstoragesdk.UpdateBucketResponse, error)
	deleteFn func(context.Context, objectstoragesdk.DeleteBucketRequest) (objectstoragesdk.DeleteBucketResponse, error)
}

func (f *fakeBucketRuntimeOCIClient) CreateBucket(ctx context.Context, request objectstoragesdk.CreateBucketRequest) (objectstoragesdk.CreateBucketResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return objectstoragesdk.CreateBucketResponse{}, nil
}

func (f *fakeBucketRuntimeOCIClient) GetBucket(ctx context.Context, request objectstoragesdk.GetBucketRequest) (objectstoragesdk.GetBucketResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return objectstoragesdk.GetBucketResponse{}, nil
}

func (f *fakeBucketRuntimeOCIClient) ListBuckets(ctx context.Context, request objectstoragesdk.ListBucketsRequest) (objectstoragesdk.ListBucketsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return objectstoragesdk.ListBucketsResponse{}, nil
}

func (f *fakeBucketRuntimeOCIClient) UpdateBucket(ctx context.Context, request objectstoragesdk.UpdateBucketRequest) (objectstoragesdk.UpdateBucketResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return objectstoragesdk.UpdateBucketResponse{}, nil
}

func (f *fakeBucketRuntimeOCIClient) DeleteBucket(ctx context.Context, request objectstoragesdk.DeleteBucketRequest) (objectstoragesdk.DeleteBucketResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return objectstoragesdk.DeleteBucketResponse{}, nil
}

func TestBucketPlainGeneratedRuntimeCreateErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainCreateMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainMutationResult {
		client := newBucketGeneratedRuntimeMatrixClient(t, &fakeBucketRuntimeOCIClient{
			createFn: func(context.Context, objectstoragesdk.CreateBucketRequest) (objectstoragesdk.CreateBucketResponse, error) {
				return objectstoragesdk.CreateBucketResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
		})
		resource := newMatrixBucketResource()

		response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.GeneratedRuntimePlainMutationResult{
			Response:     response,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestBucketPlainGeneratedRuntimeDeleteErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainDeleteMatrix(t, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainDeleteResult {
		getCalls := 0

		client := newBucketGeneratedRuntimeMatrixClient(t, &fakeBucketRuntimeOCIClient{
			deleteFn: func(context.Context, objectstoragesdk.DeleteBucketRequest) (objectstoragesdk.DeleteBucketResponse, error) {
				return objectstoragesdk.DeleteBucketResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
			getFn: func(context.Context, objectstoragesdk.GetBucketRequest) (objectstoragesdk.GetBucketResponse, error) {
				getCalls++
				return objectstoragesdk.GetBucketResponse{
					Bucket: objectstoragesdk.Bucket{
						Id:            stringPtr("ocid1.bucket.oc1..matrix"),
						Name:          stringPtr("matrix-bucket"),
						Namespace:     stringPtr("tenantnamespace"),
						CompartmentId: stringPtr("ocid1.compartment.oc1..matrix"),
					},
				}, nil
			},
		})
		resource := newMatrixBucketResource()
		resource.Status.Namespace = "tenantnamespace"
		resource.Status.OsokStatus.Ocid = "ocid1.bucket.oc1..matrix"

		deleted, err := client.Delete(context.Background(), resource)
		if getCalls > 1 && !errortest.GeneratedRuntimePlainDeleteRequiresConfirmRead(candidate) {
			t.Fatalf("GetBucket() calls = %d, want 1 pre-delete read for case %s", getCalls, candidate.Name())
		}
		return errortest.GeneratedRuntimePlainDeleteResult{
			Deleted:      deleted,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func newBucketGeneratedRuntimeMatrixClient(
	t *testing.T,
	sdkClient *fakeBucketRuntimeOCIClient,
) *namespaceResolvingBucketServiceClient {
	t.Helper()

	delegate := defaultBucketServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*objectstoragev1beta1.Bucket](generatedruntime.Config[*objectstoragev1beta1.Bucket]{
			Kind:      "Bucket",
			SDKName:   "Bucket",
			Semantics: newBucketRuntimeSemantics(),
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &objectstoragesdk.CreateBucketRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.CreateBucket(ctx, *request.(*objectstoragesdk.CreateBucketRequest))
				},
				Fields: []generatedruntime.RequestField{
					bucketNamespaceField(),
					{FieldName: "CreateBucketDetails", Contribution: "body"},
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &objectstoragesdk.GetBucketRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.GetBucket(ctx, *request.(*objectstoragesdk.GetBucketRequest))
				},
				Fields: []generatedruntime.RequestField{
					bucketNamespaceField(),
					bucketNameField(),
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &objectstoragesdk.ListBucketsRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.ListBuckets(ctx, *request.(*objectstoragesdk.ListBucketsRequest))
				},
				Fields: []generatedruntime.RequestField{
					bucketNamespaceField(),
					{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &objectstoragesdk.UpdateBucketRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.UpdateBucket(ctx, *request.(*objectstoragesdk.UpdateBucketRequest))
				},
				Fields: []generatedruntime.RequestField{
					bucketNamespaceField(),
					bucketNameField(),
					{FieldName: "UpdateBucketDetails", Contribution: "body"},
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &objectstoragesdk.DeleteBucketRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.DeleteBucket(ctx, *request.(*objectstoragesdk.DeleteBucketRequest))
				},
				Fields: []generatedruntime.RequestField{
					bucketNamespaceField(),
					bucketNameField(),
				},
			},
		}),
	}

	return &namespaceResolvingBucketServiceClient{
		delegate:        delegate,
		namespaceGetter: &fakeBucketNamespaceGetter{response: "tenantnamespace"},
	}
}

func newMatrixBucketResource() *objectstoragev1beta1.Bucket {
	return &objectstoragev1beta1.Bucket{
		Spec: objectstoragev1beta1.BucketSpec{
			Name:          "matrix-bucket",
			CompartmentId: "ocid1.compartment.oc1..matrix",
		},
	}
}

func stringPtr(value string) *string {
	return &value
}

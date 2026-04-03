//go:build legacyservicemanager

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package objectstorage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociobjectstorage "github.com/oracle/oci-go-sdk/v65/objectstorage"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/objectstorage"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ---------------------------------------------------------------------------
// fakeCredentialClient — implements credhelper.CredentialClient for testing.
// ---------------------------------------------------------------------------

type fakeCredentialClient struct {
	createSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	deleteSecretFn func(ctx context.Context, name, ns string) (bool, error)
	getSecretFn    func(ctx context.Context, name, ns string) (map[string][]byte, error)
	updateSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	createCalled   bool
	deleteCalled   bool
}

func (f *fakeCredentialClient) CreateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	f.createCalled = true
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name, ns string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, ns)
	}
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, ns)
	}
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// fakeObjectStorageClient — implements ObjectStorageClientInterface for testing.
// ---------------------------------------------------------------------------

type fakeObjectStorageClient struct {
	getNamespaceFn func(ctx context.Context, req ociobjectstorage.GetNamespaceRequest) (ociobjectstorage.GetNamespaceResponse, error)
	createBucketFn func(ctx context.Context, req ociobjectstorage.CreateBucketRequest) (ociobjectstorage.CreateBucketResponse, error)
	getBucketFn    func(ctx context.Context, req ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error)
	updateBucketFn func(ctx context.Context, req ociobjectstorage.UpdateBucketRequest) (ociobjectstorage.UpdateBucketResponse, error)
	deleteBucketFn func(ctx context.Context, req ociobjectstorage.DeleteBucketRequest) (ociobjectstorage.DeleteBucketResponse, error)
}

type fakeServiceError struct {
	statusCode int
	code       string
	message    string
}

func (e fakeServiceError) Error() string           { return e.message }
func (e fakeServiceError) GetHTTPStatusCode() int  { return e.statusCode }
func (e fakeServiceError) GetMessage() string      { return e.message }
func (e fakeServiceError) GetCode() string         { return e.code }
func (e fakeServiceError) GetOpcRequestID() string { return "opc-request-id" }

func (f *fakeObjectStorageClient) GetNamespace(ctx context.Context, req ociobjectstorage.GetNamespaceRequest) (ociobjectstorage.GetNamespaceResponse, error) {
	if f.getNamespaceFn != nil {
		return f.getNamespaceFn(ctx, req)
	}
	return ociobjectstorage.GetNamespaceResponse{Value: common.String("mynamespace")}, nil
}

func (f *fakeObjectStorageClient) CreateBucket(ctx context.Context, req ociobjectstorage.CreateBucketRequest) (ociobjectstorage.CreateBucketResponse, error) {
	if f.createBucketFn != nil {
		return f.createBucketFn(ctx, req)
	}
	return ociobjectstorage.CreateBucketResponse{}, nil
}

func (f *fakeObjectStorageClient) GetBucket(ctx context.Context, req ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error) {
	if f.getBucketFn != nil {
		return f.getBucketFn(ctx, req)
	}
	return ociobjectstorage.GetBucketResponse{Bucket: ociobjectstorage.Bucket{Name: common.String("mybucket")}}, nil
}

func (f *fakeObjectStorageClient) UpdateBucket(ctx context.Context, req ociobjectstorage.UpdateBucketRequest) (ociobjectstorage.UpdateBucketResponse, error) {
	if f.updateBucketFn != nil {
		return f.updateBucketFn(ctx, req)
	}
	return ociobjectstorage.UpdateBucketResponse{}, nil
}

func (f *fakeObjectStorageClient) DeleteBucket(ctx context.Context, req ociobjectstorage.DeleteBucketRequest) (ociobjectstorage.DeleteBucketResponse, error) {
	if f.deleteBucketFn != nil {
		return f.deleteBucketFn(ctx, req)
	}
	return ociobjectstorage.DeleteBucketResponse{}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func defaultLog() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
}

func emptyProvider() common.ConfigurationProvider {
	return common.NewRawConfigurationProvider("", "", "", "", "", nil)
}

func mgrWithFake(credClient *fakeCredentialClient, fake *fakeObjectStorageClient) *ObjectStorageBucketServiceManager {
	mgr := NewObjectStorageBucketServiceManager(emptyProvider(), credClient, nil, defaultLog())
	ExportSetClientForTest(mgr, fake)
	return mgr
}

// ---------------------------------------------------------------------------
// TestGetCredentialMap
// ---------------------------------------------------------------------------

func TestGetCredentialMap(t *testing.T) {
	credMap := GetCredentialMapForTest("mynamespace", "mybucket")

	assert.Equal(t, "mynamespace", string(credMap["namespace"]))
	assert.Equal(t, "mybucket", string(credMap["bucketName"]))
	assert.Contains(t, string(credMap["apiEndpoint"]), "mynamespace")
	assert.Contains(t, string(credMap["apiEndpoint"]), "mybucket")
}

// ---------------------------------------------------------------------------
// TestGetCrdStatus
// ---------------------------------------------------------------------------

func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewObjectStorageBucketServiceManager(emptyProvider(), &fakeCredentialClient{}, nil, defaultLog())

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Status.OsokStatus.Ocid = "mynamespace/mybucket"

	status, err := mgr.GetCrdStatus(b)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("mynamespace/mybucket"), status.Ocid)
}

func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := NewObjectStorageBucketServiceManager(emptyProvider(), &fakeCredentialClient{}, nil, defaultLog())

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — type assertion
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := NewObjectStorageBucketServiceManager(emptyProvider(), &fakeCredentialClient{}, nil, defaultLog())

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — create new bucket
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_CreateNew(t *testing.T) {
	fake := &fakeObjectStorageClient{
		getNamespaceFn: func(_ context.Context, _ ociobjectstorage.GetNamespaceRequest) (ociobjectstorage.GetNamespaceResponse, error) {
			return ociobjectstorage.GetNamespaceResponse{Value: common.String("mynamespace")}, nil
		},
		createBucketFn: func(_ context.Context, _ ociobjectstorage.CreateBucketRequest) (ociobjectstorage.CreateBucketResponse, error) {
			return ociobjectstorage.CreateBucketResponse{}, nil
		},
	}
	credClient := &fakeCredentialClient{}
	mgr := mgrWithFake(credClient, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"
	b.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	b.Spec.Name = "mybucket"

	resp, err := mgr.CreateOrUpdate(context.Background(), b, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID("mynamespace/mybucket"), b.Status.OsokStatus.Ocid)
	assert.True(t, credClient.createCalled)
}

func TestCreateOrUpdate_CreateNew_GetNamespaceFailure(t *testing.T) {
	fake := &fakeObjectStorageClient{
		getNamespaceFn: func(_ context.Context, _ ociobjectstorage.GetNamespaceRequest) (ociobjectstorage.GetNamespaceResponse, error) {
			return ociobjectstorage.GetNamespaceResponse{}, errors.New("namespace lookup failed")
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	b.Spec.Name = "mybucket"

	resp, err := mgr.CreateOrUpdate(context.Background(), b, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "namespace lookup failed")
}

func TestCreateOrUpdate_CreateNew_CreateBucketFailure(t *testing.T) {
	fake := &fakeObjectStorageClient{
		createBucketFn: func(_ context.Context, _ ociobjectstorage.CreateBucketRequest) (ociobjectstorage.CreateBucketResponse, error) {
			return ociobjectstorage.CreateBucketResponse{}, errors.New("create bucket failed")
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	b.Spec.Name = "mybucket"

	resp, err := mgr.CreateOrUpdate(context.Background(), b, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — create with pre-set namespace in spec
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_CreateNew_WithSpecNamespace(t *testing.T) {
	var getNamespaceCalled bool
	fake := &fakeObjectStorageClient{
		getNamespaceFn: func(_ context.Context, _ ociobjectstorage.GetNamespaceRequest) (ociobjectstorage.GetNamespaceResponse, error) {
			getNamespaceCalled = true
			return ociobjectstorage.GetNamespaceResponse{Value: common.String("should-not-be-called")}, nil
		},
		createBucketFn: func(_ context.Context, req ociobjectstorage.CreateBucketRequest) (ociobjectstorage.CreateBucketResponse, error) {
			assert.Equal(t, "presetnamespace", *req.NamespaceName)
			return ociobjectstorage.CreateBucketResponse{}, nil
		},
	}
	credClient := &fakeCredentialClient{}
	mgr := mgrWithFake(credClient, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"
	b.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	b.Spec.Name = "mybucket"
	b.Spec.Namespace = "presetnamespace"

	resp, err := mgr.CreateOrUpdate(context.Background(), b, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, getNamespaceCalled, "GetNamespace should not be called when namespace is preset in spec")
	assert.Equal(t, ociv1beta1.OCID("presetnamespace/mybucket"), b.Status.OsokStatus.Ocid)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — bind to existing bucket
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_Bind(t *testing.T) {
	fake := &fakeObjectStorageClient{
		getBucketFn: func(_ context.Context, req ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error) {
			assert.Equal(t, "mynamespace", *req.NamespaceName)
			assert.Equal(t, "mybucket", *req.BucketName)
			return ociobjectstorage.GetBucketResponse{Bucket: ociobjectstorage.Bucket{Name: common.String("mybucket")}}, nil
		},
	}
	credClient := &fakeCredentialClient{}
	mgr := mgrWithFake(credClient, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"
	b.Spec.BucketId = "mynamespace/mybucket"
	b.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	b.Spec.Name = "mybucket"

	resp, err := mgr.CreateOrUpdate(context.Background(), b, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID("mynamespace/mybucket"), b.Status.OsokStatus.Ocid)
	assert.True(t, credClient.createCalled)
}

func TestCreateOrUpdate_Bind_InvalidId(t *testing.T) {
	mgr := mgrWithFake(&fakeCredentialClient{}, &fakeObjectStorageClient{})

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Spec.BucketId = "invalidformat"

	resp, err := mgr.CreateOrUpdate(context.Background(), b, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "namespace/bucketName")
}

func TestCreateOrUpdate_Bind_GetBucketError(t *testing.T) {
	fake := &fakeObjectStorageClient{
		getBucketFn: func(_ context.Context, _ ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error) {
			return ociobjectstorage.GetBucketResponse{}, errors.New("bucket not found")
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Spec.BucketId = "mynamespace/mybucket"

	resp, err := mgr.CreateOrUpdate(context.Background(), b, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — update existing bucket (status.ocid already set)
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_Update(t *testing.T) {
	var updateCalled bool
	fake := &fakeObjectStorageClient{
		getBucketFn: func(_ context.Context, _ ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error) {
			return ociobjectstorage.GetBucketResponse{Bucket: ociobjectstorage.Bucket{Name: common.String("mybucket")}}, nil
		},
		updateBucketFn: func(_ context.Context, _ ociobjectstorage.UpdateBucketRequest) (ociobjectstorage.UpdateBucketResponse, error) {
			updateCalled = true
			return ociobjectstorage.UpdateBucketResponse{}, nil
		},
	}
	credClient := &fakeCredentialClient{}
	mgr := mgrWithFake(credClient, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"
	b.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	b.Spec.Name = "mybucket"
	b.Spec.AccessType = "ObjectRead"
	b.Status.OsokStatus.Ocid = "mynamespace/mybucket"

	resp, err := mgr.CreateOrUpdate(context.Background(), b, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateBucket should have been called")
}

func TestCreateOrUpdate_UpdateSendsCompartmentMove(t *testing.T) {
	var updatedReq ociobjectstorage.UpdateBucketRequest
	fake := &fakeObjectStorageClient{
		getBucketFn: func(_ context.Context, _ ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error) {
			return ociobjectstorage.GetBucketResponse{
				Bucket: ociobjectstorage.Bucket{
					Name:          common.String("mybucket"),
					CompartmentId: common.String("ocid1.compartment.oc1..old"),
				},
			}, nil
		},
		updateBucketFn: func(_ context.Context, req ociobjectstorage.UpdateBucketRequest) (ociobjectstorage.UpdateBucketResponse, error) {
			updatedReq = req
			return ociobjectstorage.UpdateBucketResponse{}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"
	b.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	b.Status.OsokStatus.Ocid = "mynamespace/mybucket"

	resp, err := mgr.CreateOrUpdate(context.Background(), b, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.NotNil(t, updatedReq.CompartmentId)
	assert.Equal(t, string(b.Spec.CompartmentId), *updatedReq.CompartmentId)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — secret already exists
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_SecretAlreadyExists(t *testing.T) {
	fake := &fakeObjectStorageClient{}
	credClient := &fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return false, apierrors.NewAlreadyExists(schema.GroupResource{}, "my-bucket-cr")
		},
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return servicemanager.AddManagedSecretData(GetCredentialMapForTest("mynamespace", "mybucket"), "ObjectStorageBucket", "my-bucket-cr"), nil
		},
	}
	mgr := mgrWithFake(credClient, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"
	b.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	b.Spec.Name = "mybucket"
	b.Spec.Namespace = "mynamespace"
	b.Status.OsokStatus.Ocid = "mynamespace/mybucket"

	resp, err := mgr.CreateOrUpdate(context.Background(), b, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful, "AlreadyExists on secret should be treated as success")
}

// ---------------------------------------------------------------------------
// TestDelete
// ---------------------------------------------------------------------------

func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := NewObjectStorageBucketServiceManager(emptyProvider(), credClient, nil, defaultLog())

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"

	done, err := mgr.Delete(context.Background(), b)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled)
}

func TestDelete_Success(t *testing.T) {
	var deleteBucketCalled bool
	fake := &fakeObjectStorageClient{
		deleteBucketFn: func(_ context.Context, req ociobjectstorage.DeleteBucketRequest) (ociobjectstorage.DeleteBucketResponse, error) {
			deleteBucketCalled = true
			assert.Equal(t, "mynamespace", *req.NamespaceName)
			assert.Equal(t, "mybucket", *req.BucketName)
			return ociobjectstorage.DeleteBucketResponse{}, nil
		},
		getBucketFn: func(_ context.Context, _ ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error) {
			return ociobjectstorage.GetBucketResponse{}, fakeServiceError{statusCode: 404, code: "NotFound", message: "bucket not found"}
		},
	}
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return servicemanager.AddManagedSecretData(map[string][]byte{}, "ObjectStorageBucket", "my-bucket-cr"), nil
		},
	}
	mgr := mgrWithFake(credClient, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"
	b.Status.OsokStatus.Ocid = "mynamespace/mybucket"

	done, err := mgr.Delete(context.Background(), b)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteBucketCalled, "DeleteBucket should have been called")
	assert.True(t, credClient.deleteCalled, "DeleteSecret should have been called")
}

func TestDelete_NotFound(t *testing.T) {
	// 404 should be treated as already-deleted (graceful).
	fake := &fakeObjectStorageClient{
		deleteBucketFn: func(_ context.Context, _ ociobjectstorage.DeleteBucketRequest) (ociobjectstorage.DeleteBucketResponse, error) {
			return ociobjectstorage.DeleteBucketResponse{}, fakeServiceError{statusCode: 404, code: "NotFound", message: "bucket not found"}
		},
	}
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return servicemanager.AddManagedSecretData(map[string][]byte{}, "ObjectStorageBucket", "my-bucket-cr"), nil
		},
	}
	mgr := mgrWithFake(credClient, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"
	b.Status.OsokStatus.Ocid = "mynamespace/mybucket"

	done, err := mgr.Delete(context.Background(), b)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, credClient.deleteCalled)
}

func TestDelete_UsesSpecBucketIdWhenStatusIsEmpty(t *testing.T) {
	var deletedNamespace, deletedBucket string
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return nil, apierrors.NewNotFound(corev1.Resource("secret"), "my-bucket-cr")
		},
	}
	fake := &fakeObjectStorageClient{
		deleteBucketFn: func(_ context.Context, req ociobjectstorage.DeleteBucketRequest) (ociobjectstorage.DeleteBucketResponse, error) {
			deletedNamespace = *req.NamespaceName
			deletedBucket = *req.BucketName
			return ociobjectstorage.DeleteBucketResponse{}, nil
		},
		getBucketFn: func(_ context.Context, req ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error) {
			return ociobjectstorage.GetBucketResponse{}, fakeServiceError{
				statusCode: 404,
				code:       "NotFound",
				message:    *req.NamespaceName + "/" + *req.BucketName,
			}
		},
	}
	mgr := mgrWithFake(credClient, fake)

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"
	b.Spec.BucketId = "mynamespace/mybucket"

	done, err := mgr.Delete(context.Background(), b)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, "mynamespace", deletedNamespace)
	assert.Equal(t, "mybucket", deletedBucket)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should be skipped when the secret is already missing")
}

func TestDelete_MalformedOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := NewObjectStorageBucketServiceManager(emptyProvider(), credClient, nil, defaultLog())

	b := &ociv1beta1.ObjectStorageBucket{}
	b.Name = "my-bucket-cr"
	b.Namespace = "default"
	b.Status.OsokStatus.Ocid = "malformed"

	done, err := mgr.Delete(context.Background(), b)
	assert.Error(t, err)
	assert.False(t, done)
}

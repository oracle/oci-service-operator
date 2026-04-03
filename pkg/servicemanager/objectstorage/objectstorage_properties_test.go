//go:build legacyservicemanager

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package objectstorage_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/quick"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociobjectstorage "github.com/oracle/oci-go-sdk/v65/objectstorage"
	objectstoragev1beta1 "github.com/oracle/oci-service-operator/api/objectstorage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/shared"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/objectstorage"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ObjectStorageBucket = objectstoragev1beta1.Bucket

func TestObjectStorageBucket_PropertyCredentialMapNeverUsesPlaceholder(t *testing.T) {
	property := func(seed uint16) bool {
		namespace := fmt.Sprintf("ns-%d", seed)
		bucketName := fmt.Sprintf("bucket-%d", seed)
		credMap := GetCredentialMapForTest(namespace, bucketName)
		endpoint := string(credMap["apiEndpoint"])

		return !strings.Contains(endpoint, "<region>") &&
			strings.Contains(endpoint, namespace) &&
			strings.Contains(endpoint, bucketName)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestObjectStorageBucket_PropertyResolveNamespaceDoesNotMutateSpec(t *testing.T) {
	property := func(seed uint16) bool {
		namespace := fmt.Sprintf("ns-%d", seed)
		bucketName := fmt.Sprintf("bucket-%d", seed)
		fake := &fakeObjectStorageClient{
			getNamespaceFn: func(_ context.Context, _ ociobjectstorage.GetNamespaceRequest) (ociobjectstorage.GetNamespaceResponse, error) {
				return ociobjectstorage.GetNamespaceResponse{Value: common.String(namespace)}, nil
			},
			createBucketFn: func(_ context.Context, req ociobjectstorage.CreateBucketRequest) (ociobjectstorage.CreateBucketResponse, error) {
				if *req.NamespaceName != namespace {
					t.Fatalf("expected namespace %s, got %s", namespace, *req.NamespaceName)
				}
				return ociobjectstorage.CreateBucketResponse{}, nil
			},
		}

		mgr := mgrWithFake(&fakeCredentialClient{}, fake)
		bucket := &ObjectStorageBucket{}
		bucket.Name = "bucket-cr"
		bucket.Namespace = "default"
		bucket.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
		bucket.Spec.Name = bucketName

		resp, err := mgr.CreateOrUpdate(context.Background(), bucket, ctrl.Request{})
		return err == nil &&
			resp.IsSuccessful &&
			bucket.Spec.Namespace == "" &&
			bucket.Status.OsokStatus.Ocid == shared.OCID(namespace+"/"+bucketName)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestObjectStorageBucket_PropertyDeleteUsesSpecIDAndIgnoresMissingSecret(t *testing.T) {
	property := func(seed uint16) bool {
		namespace := fmt.Sprintf("ns-%d", seed)
		bucketName := fmt.Sprintf("bucket-%d", seed)
		var deletedNamespace, deletedBucket string

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
		credClient := &fakeCredentialClient{
			getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
				return nil, apierrors.NewNotFound(corev1.Resource("secret"), "bucket-cr")
			},
		}
		mgr := mgrWithFake(credClient, fake)

		bucket := &ObjectStorageBucket{}
		bucket.Name = "bucket-cr"
		bucket.Namespace = "default"
		bucket.Spec.BucketId = shared.OCID(namespace + "/" + bucketName)

		done, err := mgr.Delete(context.Background(), bucket)
		return err == nil &&
			done &&
			!credClient.deleteCalled &&
			deletedNamespace == namespace &&
			deletedBucket == bucketName
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestObjectStorageBucket_PropertyTagDriftTriggersUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		namespace := fmt.Sprintf("ns-%d", seed)
		bucketName := fmt.Sprintf("bucket-%d", seed)
		var updatedReq ociobjectstorage.UpdateBucketRequest

		fake := &fakeObjectStorageClient{
			getBucketFn: func(_ context.Context, req ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error) {
				return ociobjectstorage.GetBucketResponse{
					Bucket: ociobjectstorage.Bucket{
						Name:             common.String(bucketName),
						Namespace:        common.String(namespace),
						PublicAccessType: ociobjectstorage.BucketPublicAccessTypeObjectread,
						Versioning:       ociobjectstorage.BucketVersioningEnabled,
						FreeformTags:     map[string]string{"team": "old"},
						DefinedTags: map[string]map[string]interface{}{
							"ops": {"env": "dev"},
						},
					},
				}, nil
			},
			updateBucketFn: func(_ context.Context, req ociobjectstorage.UpdateBucketRequest) (ociobjectstorage.UpdateBucketResponse, error) {
				updatedReq = req
				return ociobjectstorage.UpdateBucketResponse{}, nil
			},
		}

		mgr := mgrWithFake(&fakeCredentialClient{}, fake)
		bucket := &ObjectStorageBucket{}
		bucket.Name = "bucket-cr"
		bucket.Namespace = "default"
		bucket.Status.OsokStatus.Ocid = shared.OCID(namespace + "/" + bucketName)
		bucket.Spec.FreeFormTags = map[string]string{"team": "platform"}
		bucket.Spec.DefinedTags = map[string]shared.MapValue{
			"ops": {"env": "prod"},
		}

		resp, err := mgr.CreateOrUpdate(context.Background(), bucket, ctrl.Request{})
		return err == nil &&
			resp.IsSuccessful &&
			updatedReq.FreeformTags["team"] == "platform" &&
			updatedReq.DefinedTags["ops"]["env"] == "prod"
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestObjectStorageBucket_PropertyCompartmentDriftTriggersUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		namespace := fmt.Sprintf("ns-%d", seed)
		bucketName := fmt.Sprintf("bucket-%d", seed)
		var updatedReq ociobjectstorage.UpdateBucketRequest

		fake := &fakeObjectStorageClient{
			getBucketFn: func(_ context.Context, req ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error) {
				return ociobjectstorage.GetBucketResponse{
					Bucket: ociobjectstorage.Bucket{
						Name:          req.BucketName,
						Namespace:     req.NamespaceName,
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
		bucket := &ObjectStorageBucket{}
		bucket.Name = "bucket-cr"
		bucket.Namespace = "default"
		bucket.Status.OsokStatus.Ocid = shared.OCID(namespace + "/" + bucketName)
		bucket.Spec.CompartmentId = "ocid1.compartment.oc1..new"

		resp, err := mgr.CreateOrUpdate(context.Background(), bucket, ctrl.Request{})
		return err == nil &&
			resp.IsSuccessful &&
			updatedReq.CompartmentId != nil &&
			*updatedReq.CompartmentId == string(bucket.Spec.CompartmentId)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestObjectStorageBucket_PropertyImmutableDriftFailsBeforeUpdate(t *testing.T) {
	namespace := "ns-immutable"
	bucketName := "bucket-immutable"
	updateCalled := false

	fake := &fakeObjectStorageClient{
		getBucketFn: func(_ context.Context, req ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error) {
			return ociobjectstorage.GetBucketResponse{
				Bucket: ociobjectstorage.Bucket{
					Name:        req.BucketName,
					Namespace:   req.NamespaceName,
					StorageTier: ociobjectstorage.BucketStorageTierArchive,
				},
			}, nil
		},
		updateBucketFn: func(_ context.Context, _ ociobjectstorage.UpdateBucketRequest) (ociobjectstorage.UpdateBucketResponse, error) {
			updateCalled = true
			return ociobjectstorage.UpdateBucketResponse{}, nil
		},
	}

	mgr := mgrWithFake(&fakeCredentialClient{}, fake)
	bucket := &ObjectStorageBucket{}
	bucket.Name = "bucket-cr"
	bucket.Namespace = "default"
	bucket.Status.OsokStatus.Ocid = shared.OCID(namespace + "/" + bucketName)
	bucket.Spec.StorageType = "Standard"

	resp, err := mgr.CreateOrUpdate(context.Background(), bucket, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "storageType cannot be updated in place")
	assert.False(t, updateCalled)
}

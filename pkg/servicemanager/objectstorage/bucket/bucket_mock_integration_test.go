/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package bucket

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	objectstoragesdk "github.com/oracle/oci-go-sdk/v65/objectstorage"
	objectstoragev1beta1 "github.com/oracle/oci-service-operator/api/objectstorage/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	mockBucketID        = "ocid1.bucket.oc1..mock"
	mockBucketNamespace = "mock-object-storage-namespace"
	mockBucketName      = "mock-bucket"
)

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/objectstorage/objectstoragebucket and formal/imports/objectstorage/objectstoragebucket.json
//   - resource runtime: bucket_runtime_client.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/objectstorage
func TestMockIntegrationBucketCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &objectstoragev1beta1.Bucket{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-bucket", Namespace: "default", UID: types.UID("mock-bucket-uid")},
		Spec: objectstoragev1beta1.BucketSpec{
			Namespace:        mockBucketNamespace,
			Name:             mockBucketName,
			CompartmentId:    "ocid1.compartment.oc1..mock",
			Metadata:         map[string]string{"phase": "create"},
			PublicAccessType: string(objectstoragesdk.CreateBucketDetailsPublicAccessTypeNopublicaccess),
			StorageTier:      string(objectstoragesdk.CreateBucketDetailsStorageTierStandard),
			FreeformTags:     map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newBucketMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{
		Host:      "https://objectstorage.mock.invalid",
		BasePath:  "n",
		Responder: responder,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Bucket OCI mock: %v", err)
		}
	})

	sdkClient := objectstoragesdk.ObjectStorageClient{BaseClient: session.BaseClient()}
	manager := &BucketServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newBucketDefaultRuntimeHooks(sdkClient)
	applyBucketRuntimeHooks(&hooks)
	client := defaultBucketServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*objectstoragev1beta1.Bucket](buildBucketGeneratedRuntimeConfig(manager, hooks)),
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*objectstoragev1beta1.Bucket]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *objectstoragev1beta1.Bucket) error {
			if current.Status.Id != mockBucketID ||
				current.Status.Namespace != mockBucketNamespace ||
				current.Status.Name != mockBucketName ||
				current.Status.Metadata["phase"] != "create" ||
				current.Status.FreeformTags["osok-mock"] != "create" {
				return fmt.Errorf("created Bucket status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *objectstoragev1beta1.Bucket) {
			current.Spec.Metadata = map[string]string{"phase": "update"}
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *objectstoragev1beta1.Bucket) error {
			if current.Status.Id != mockBucketID ||
				current.Status.Metadata["phase"] != "update" ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Bucket status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func newBucketMockResponder(resource *objectstoragev1beta1.Bucket) (*ocimock.CRUDResponder[objectstoragesdk.Bucket], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.August, 31, 20, 0, 50, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[objectstoragesdk.Bucket]{
		CollectionPath:     "/n/" + mockBucketNamespace + "/b",
		ItemPath:           "/n/" + mockBucketNamespace + "/b/" + mockBucketName,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		Create: func(request ocimock.Request) (objectstoragesdk.Bucket, ocimock.Response, error) {
			var details objectstoragesdk.CreateBucketDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return objectstoragesdk.Bucket{}, ocimock.Response{}, err
			}
			expected := objectstoragesdk.CreateBucketDetails{
				Name:             common.String(resource.Spec.Name),
				CompartmentId:    common.String(resource.Spec.CompartmentId),
				Metadata:         map[string]string{"phase": "create"},
				PublicAccessType: objectstoragesdk.CreateBucketDetailsPublicAccessTypeNopublicaccess,
				StorageTier:      objectstoragesdk.CreateBucketDetailsStorageTierStandard,
				FreeformTags:     map[string]string{"osok-mock": "create"},
			}
			if !reflect.DeepEqual(details, expected) {
				return objectstoragesdk.Bucket{}, ocimock.Response{}, fmt.Errorf("CreateBucket details = %+v, want %+v", details, expected)
			}
			state := objectstoragesdk.Bucket{
				Namespace:           common.String(mockBucketNamespace),
				Name:                details.Name,
				CompartmentId:       details.CompartmentId,
				Metadata:            details.Metadata,
				CreatedBy:           common.String("mock-user"),
				TimeCreated:         &createdAt,
				Etag:                common.String("mock-etag-create"),
				PublicAccessType:    objectstoragesdk.BucketPublicAccessTypeNopublicaccess,
				StorageTier:         objectstoragesdk.BucketStorageTierStandard,
				ObjectEventsEnabled: common.Bool(resource.Spec.ObjectEventsEnabled),
				FreeformTags:        details.FreeformTags,
				Id:                  common.String(mockBucketID),
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state objectstoragesdk.Bucket) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state objectstoragesdk.Bucket) (objectstoragesdk.Bucket, ocimock.Response, error) {
			var details objectstoragesdk.UpdateBucketDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return objectstoragesdk.Bucket{}, ocimock.Response{}, err
			}
			expected := objectstoragesdk.UpdateBucketDetails{
				Metadata:     map[string]string{"phase": "update"},
				FreeformTags: map[string]string{"osok-mock": "update"},
			}
			if !reflect.DeepEqual(details, expected) {
				return objectstoragesdk.Bucket{}, ocimock.Response{}, fmt.Errorf("UpdateBucket details = %+v, want %+v", details, expected)
			}
			state.Metadata = details.Metadata
			state.FreeformTags = details.FreeformTags
			state.Etag = common.String("mock-etag-update")
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(request ocimock.Request, _ objectstoragesdk.Bucket) (ocimock.Response, error) {
			if len(request.Body) != 0 {
				return ocimock.Response{}, fmt.Errorf("DeleteBucket body = %s", request.Body)
			}
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
		NotFound: func(_ ocimock.Request) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusNotFound, map[string]string{
				"code":    "BucketNotFound",
				"message": "mock bucket not found",
			})
		},
	})
}

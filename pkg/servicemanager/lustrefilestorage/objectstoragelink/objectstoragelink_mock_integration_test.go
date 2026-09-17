/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package objectstoragelink

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	lustrefilestoragesdk "github.com/oracle/oci-go-sdk/v65/lustrefilestorage"
	lustrefilestoragev1beta1 "github.com/oracle/oci-service-operator/api/lustrefilestorage/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationObjectStorageLinkWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &lustrefilestoragev1beta1.ObjectStorageLink{}
	ocimock.InitializeResource(resource, "mock-objectstoragelink")
	resource.Spec = ocimock.MustJSONFixture[lustrefilestoragev1beta1.ObjectStorageLinkSpec](t, `{
  "availabilityDomain": "Uocm:PHX-AD-1",
  "compartmentId": "<ocid:1>",
  "displayName": "sample-object-storage-link",
  "fileSystemPath": "/mnt/lustre/link",
  "freeformTags": {
    "team": "storage"
  },
  "isOverwrite": true,
  "lustreFileSystemId": "<ocid:2>",
  "objectStoragePrefix": "namespace:/bucket/prefix"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "sample-object-storage-link-updated"
}`)

	createRequest := ocimock.MustJSONFixture[lustrefilestoragesdk.CreateObjectStorageLinkDetails](t, `{
  "availabilityDomain": "Uocm:PHX-AD-1",
  "compartmentId": "<ocid:1>",
  "displayName": "sample-object-storage-link",
  "fileSystemPath": "/mnt/lustre/link",
  "freeformTags": {
    "team": "storage"
  },
  "isOverwrite": true,
  "lustreFileSystemId": "<ocid:2>",
  "objectStoragePrefix": "namespace:/bucket/prefix"
}`)
	updateRequest := ocimock.MustJSONFixture[lustrefilestoragesdk.UpdateObjectStorageLinkDetails](t, `{
  "displayName": "sample-object-storage-link-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[lustrefilestoragesdk.ObjectStorageLink](t, `{
  "availabilityDomain": "Uocm:PHX-AD-1",
  "compartmentId": "<ocid:1>",
  "currentJobId": "<ocid:3>",
  "definedTags": null,
  "displayName": "sample-object-storage-link",
  "fileSystemPath": "/mnt/lustre/link",
  "freeformTags": {
    "team": "storage"
  },
  "id": "<ocid:4>",
  "isOverwrite": true,
  "lastJobId": "<ocid:5>",
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "lustreFileSystemId": "<ocid:2>",
  "objectStoragePrefix": "namespace:/bucket/prefix",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-02T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[lustrefilestoragesdk.ObjectStorageLink](t, `{
  "availabilityDomain": "Uocm:PHX-AD-1",
  "compartmentId": "<ocid:1>",
  "currentJobId": "<ocid:3>",
  "definedTags": null,
  "displayName": "sample-object-storage-link-updated",
  "fileSystemPath": "/mnt/lustre/link",
  "freeformTags": {
    "team": "storage"
  },
  "id": "<ocid:4>",
  "isOverwrite": true,
  "lastJobId": "<ocid:5>",
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "lustreFileSystemId": "<ocid:2>",
  "objectStoragePrefix": "namespace:/bucket/prefix",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-02T03:04:05Z"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[lustrefilestoragesdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:6>",
  "operationType": "DELETE_OBJECT_STORAGE_LINK",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "objectstoragelink",
      "entityUri": null,
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-01-02T03:04:05Z",
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[lustrefilestoragesdk.ObjectStorageLink, lustrefilestoragesdk.CreateObjectStorageLinkDetails, lustrefilestoragesdk.UpdateObjectStorageLinkDetails]{
		CollectionPath: "/20250228/objectStorageLinks", ItemPath: "/20250228/objectStorageLinks/<ocid:4>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: nil, UpdateHeaders: nil, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ lustrefilestoragesdk.CreateObjectStorageLinkDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/<ocid:6>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://lustre-file-storage.us-ashburn-1.oci.oraclecloud.com", BasePath: "20250228", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := lustrefilestoragesdk.LustreFileStorageClient{BaseClient: session.BaseClient()}
	manager := &ObjectStorageLinkServiceManager{Log: log}
	hooks := newObjectStorageLinkDefaultRuntimeHooks(sdkClient)
	applyObjectStorageLinkRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapObjectStorageLinkGeneratedClient(hooks, defaultObjectStorageLinkServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*lustrefilestoragev1beta1.ObjectStorageLink](buildObjectStorageLinkGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*lustrefilestoragev1beta1.ObjectStorageLink]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *lustrefilestoragev1beta1.ObjectStorageLink) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ObjectStorageLink status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *lustrefilestoragev1beta1.ObjectStorageLink) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *lustrefilestoragev1beta1.ObjectStorageLink) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ObjectStorageLink status = %+v", current.Status)
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

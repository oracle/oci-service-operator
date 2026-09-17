/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package lustrefilesystem

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

// Contract evidence: explicit SDK fixtures selected from the production
// service manager, reviewed formal lifecycle, and vendored SDK contract.
func TestMockIntegrationLustreFileSystemWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &lustrefilestoragev1beta1.LustreFileSystem{}
	ocimock.InitializeResource(resource, "mock-lustrefilesystem")
	resource.Spec = ocimock.MustJSONFixture[lustrefilestoragev1beta1.LustreFileSystemSpec](t, `{
  "compartmentId": "<ocid:2>",
  "availabilityDomain": "PHX-AD-1",
  "fileSystemName": "lustre01",
  "capacityInGBs": 5120,
  "subnetId": "<ocid:3>",
  "performanceTier": "MBPS_PER_TB_125",
  "rootSquashConfiguration": {
    "identitySquash": "ROOT",
    "squashUid": 65534,
    "squashGid": 65534,
    "clientExceptions": [
      "10.0.0.10@tcp"
    ]
  },
  "displayName": "lustre-sample",
  "fileSystemDescription": "lustre file system",
  "freeformTags": {
    "env": "test"
  },
  "nsgIds": [
    "<ocid:4>"
  ],
  "kmsKeyId": "<ocid:5>"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "fileSystemDescription": "LustreFileSystem updated"
}`)
	createRequest := ocimock.MustJSONFixture[lustrefilestoragesdk.CreateLustreFileSystemDetails](t, `{
  "compartmentId": "<ocid:2>",
  "availabilityDomain": "PHX-AD-1",
  "fileSystemName": "lustre01",
  "capacityInGBs": 5120,
  "subnetId": "<ocid:3>",
  "performanceTier": "MBPS_PER_TB_125",
  "rootSquashConfiguration": {
    "identitySquash": "ROOT",
    "squashUid": 65534,
    "squashGid": 65534,
    "clientExceptions": [
      "10.0.0.10@tcp"
    ]
  },
  "displayName": "lustre-sample",
  "fileSystemDescription": "lustre file system",
  "freeformTags": {
    "env": "test"
  },
  "nsgIds": [
    "<ocid:4>"
  ],
  "kmsKeyId": "<ocid:5>"
}`)
	updateRequest := ocimock.MustJSONFixture[lustrefilestoragesdk.UpdateLustreFileSystemDetails](t, `{
  "fileSystemDescription": "LustreFileSystem updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[lustrefilestoragesdk.LustreFileSystem](t, `{
  "compartmentId": "<ocid:2>",
  "availabilityDomain": "PHX-AD-1",
  "fileSystemName": "lustre01",
  "capacityInGBs": 5120,
  "subnetId": "<ocid:3>",
  "performanceTier": "MBPS_PER_TB_125",
  "rootSquashConfiguration": {
    "identitySquash": "ROOT",
    "squashUid": 65534,
    "squashGid": 65534,
    "clientExceptions": [
      "10.0.0.10@tcp"
    ]
  },
  "displayName": "lustre-sample",
  "fileSystemDescription": "lustre file system",
  "freeformTags": {
    "env": "test"
  },
  "nsgIds": [
    "<ocid:4>"
  ],
  "kmsKeyId": "<ocid:5>",
  "managementServiceAddress": "10.0.0.4",
  "lnet": "tcp",
  "majorVersion": "2.15",
  "id": "<ocid:1>",
  "lifecycleState": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[lustrefilestoragesdk.LustreFileSystem](t, `{
  "compartmentId": "<ocid:2>",
  "availabilityDomain": "PHX-AD-1",
  "fileSystemName": "lustre01",
  "capacityInGBs": 5120,
  "subnetId": "<ocid:3>",
  "performanceTier": "MBPS_PER_TB_125",
  "rootSquashConfiguration": {
    "identitySquash": "ROOT",
    "squashUid": 65534,
    "squashGid": 65534,
    "clientExceptions": [
      "10.0.0.10@tcp"
    ]
  },
  "displayName": "lustre-sample",
  "fileSystemDescription": "LustreFileSystem updated",
  "freeformTags": {
    "env": "test"
  },
  "nsgIds": [
    "<ocid:4>"
  ],
  "kmsKeyId": "<ocid:5>",
  "managementServiceAddress": "10.0.0.4",
  "lnet": "tcp",
  "majorVersion": "2.15",
  "id": "<ocid:1>",
  "lifecycleState": "ACTIVE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[lustrefilestoragesdk.WorkRequest](t, `{
  "id": "wr-create",
  "operationType": "CREATE_LUSTRE_FILE_SYSTEM",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:2>",
  "percentComplete": 100,
  "resources": [
    {
      "entityType": "lustrefilesystem",
      "actionType": "CREATED",
      "identifier": "<ocid:1>"
    }
  ]
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[lustrefilestoragesdk.WorkRequest](t, `{
  "id": "wr-update",
  "operationType": "UPDATE_LUSTRE_FILE_SYSTEM",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:2>",
  "percentComplete": 100,
  "resources": [
    {
      "entityType": "lustrefilesystem",
      "actionType": "UPDATED",
      "identifier": "<ocid:1>"
    }
  ]
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[lustrefilestoragesdk.WorkRequest](t, `{
  "id": "wr-delete",
  "operationType": "DELETE_LUSTRE_FILE_SYSTEM",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:2>",
  "percentComplete": 100,
  "resources": [
    {
      "entityType": "lustrefilesystem",
      "actionType": "DELETED",
      "identifier": "<ocid:1>"
    }
  ]
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[lustrefilestoragesdk.LustreFileSystem, lustrefilestoragesdk.CreateLustreFileSystemDetails, lustrefilestoragesdk.UpdateLustreFileSystemDetails]{
		CollectionPath: "/20250228/lustreFileSystems", ItemPath: "/20250228/lustreFileSystems/<ocid:1>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: http.StatusAccepted, UpdateStatus: http.StatusAccepted, DeleteStatus: http.StatusAccepted, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-create"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-delete"}},
		ValidateCreate: func(request ocimock.Request, _ lustrefilestoragesdk.CreateLustreFileSystemDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/wr-create", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/wr-update", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/wr-delete", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://lustrefilestorage.mock.invalid", BasePath: "20250228", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := lustrefilestoragesdk.LustreFileStorageClient{BaseClient: session.BaseClient()}
	manager := &LustreFileSystemServiceManager{Log: log}
	hooks := newLustreFileSystemDefaultRuntimeHooks(sdkClient)
	applyLustreFileSystemRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapLustreFileSystemGeneratedClient(hooks, defaultLustreFileSystemServiceClient{ServiceClient: generatedruntime.NewServiceClient[*lustrefilestoragev1beta1.LustreFileSystem](buildLustreFileSystemGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*lustrefilestoragev1beta1.LustreFileSystem]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *lustrefilestoragev1beta1.LustreFileSystem) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.FileSystemDescription != resource.Spec.FileSystemDescription || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created LustreFileSystem status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *lustrefilestoragev1beta1.LustreFileSystem) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *lustrefilestoragev1beta1.LustreFileSystem) error {
			if current.Status.FileSystemDescription != current.Spec.FileSystemDescription || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated LustreFileSystem status = %+v", current.Status)
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

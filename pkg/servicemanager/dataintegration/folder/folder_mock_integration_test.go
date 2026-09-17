/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package folder

import (
	"context"
	"fmt"
	"testing"

	dataintegrationsdk "github.com/oracle/oci-go-sdk/v65/dataintegration"
	dataintegrationv1beta1 "github.com/oracle/oci-service-operator/api/dataintegration/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationFolderCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.Folder{}
	ocimock.InitializeResource(resource, "mock-folder")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.FolderSpec](t, `{
  "workspaceId": "<ocid:1>",
  "aggregatorKey": "aggregator-key",
  "name": "folder create",
  "identifier": "FOLDER_CREATE",
  "registryMetadata": {
    "aggregatorKey": "aggregator-key"
  },
  "modelType": "FOLDER",
  "objectVersion": 1,
  "description": "create"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateFolderDetails](t, `{
  "name": "folder create",
  "identifier": "FOLDER_CREATE",
  "description": "create",
  "registryMetadata": {
    "aggregatorKey": "aggregator-key"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateFolderDetails](t, `{
  "description": "updated",
  "modelType": "FOLDER",
  "objectVersion": 1,
  "key": "resource-key",
  "registryMetadata": {
    "aggregatorKey": "aggregator-key",
    "isFavorite": false
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.Folder](t, `{
  "aggregatorKey": "aggregator-key",
  "name": "folder create",
  "identifier": "FOLDER_CREATE",
  "modelType": "FOLDER",
  "objectVersion": 1,
  "description": "create",
  "key": "resource-key"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.Folder](t, `{
  "aggregatorKey": "aggregator-key",
  "name": "folder create",
  "identifier": "FOLDER_CREATE",
  "modelType": "FOLDER",
  "objectVersion": 1,
  "description": "updated",
  "key": "resource-key"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.Folder, dataintegrationsdk.CreateFolderDetails, dataintegrationsdk.UpdateFolderDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/folders", ItemPath: "/20200430/workspaces/<ocid:1>/folders/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateFolderDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dataintegration.mock.invalid", BasePath: "20200430", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := dataintegrationsdk.DataIntegrationClient{BaseClient: session.BaseClient()}
	manager := &FolderServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newFolderRuntimeHooks(manager, sdkClient)
	client := wrapFolderGeneratedClient(hooks, defaultFolderServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.Folder](buildFolderGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.Folder]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.Folder) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created Folder status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.Folder) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.Folder) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated Folder status = %+v", current.Status)
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

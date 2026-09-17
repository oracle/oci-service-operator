/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package dataasset

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

func TestMockIntegrationDataAssetCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.DataAsset{}
	ocimock.InitializeResource(resource, "mock-dataasset")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.DataAssetSpec](t, `{
  "workspaceId": "<ocid:1>",
  "name": "data asset create",
  "identifier": "DATA_ASSET_CREATE",
  "objectVersion": 1,
  "modelType": "ORACLE_OBJECT_STORAGE_DATA_ASSET",
  "description": "create"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateDataAssetFromObjectStorage](t, `{
  "name": "data asset create",
  "identifier": "DATA_ASSET_CREATE",
  "description": "create"
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateDataAssetFromObjectStorage](t, `{
  "description": "updated",
  "key": "resource-key",
  "objectVersion": 1
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.DataAssetFromObjectStorageDetails](t, `{
  "name": "data asset create",
  "identifier": "DATA_ASSET_CREATE",
  "objectVersion": 1,
  "modelType": "ORACLE_OBJECT_STORAGE_DATA_ASSET",
  "description": "create",
  "key": "resource-key"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.DataAssetFromObjectStorageDetails](t, `{
  "name": "data asset create",
  "identifier": "DATA_ASSET_CREATE",
  "objectVersion": 1,
  "modelType": "ORACLE_OBJECT_STORAGE_DATA_ASSET",
  "description": "updated",
  "key": "resource-key"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.DataAssetFromObjectStorageDetails, dataintegrationsdk.CreateDataAssetFromObjectStorage, dataintegrationsdk.UpdateDataAssetFromObjectStorage]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/dataAssets", ItemPath: "/20200430/workspaces/<ocid:1>/dataAssets/resource-key",
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "modelType", "ORACLE_OBJECT_STORAGE_DATA_ASSET", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "modelType", "ORACLE_OBJECT_STORAGE_DATA_ASSET", updateRequest)
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
	manager := &DataAssetServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDataAssetRuntimeHooks(manager, sdkClient)
	client := wrapDataAssetGeneratedClient(hooks, defaultDataAssetServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.DataAsset](buildDataAssetGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.DataAsset]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.DataAsset) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created DataAsset status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.DataAsset) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.DataAsset) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated DataAsset status = %+v", current.Status)
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

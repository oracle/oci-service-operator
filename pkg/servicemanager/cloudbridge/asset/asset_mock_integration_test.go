/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package asset

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/cloudbridge"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/cloudbridge/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationAssetCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Asset](t, `
{
  "metadata": {"name": "mock-asset", "namespace": "default"},
  "spec": {
  "assetType": "AWS_EBS",
  "awsEbs": {
    "isEncrypted": false,
    "isMultiAttachEnabled": false,
    "sizeInGiBs": 1,
    "volumeKey": "mock-volumekey",
    "volumeType": "mock-volumetype"
  },
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "externalAssetKey": "mock-externalassetkey",
  "inventoryId": "<ocid:required>",
  "sourceKey": "mock-sourcekey"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-asset")
	resource.Status = apiv1beta1.AssetStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateAwsEbsAssetDetails](t, `{
  "awsEbs": {
    "isEncrypted": false,
    "isMultiAttachEnabled": false,
    "sizeInGiBs": 1,
    "volumeKey": "mock-volumekey",
    "volumeType": "mock-volumetype"
  },
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "externalAssetKey": "mock-externalassetkey",
  "inventoryId": "<ocid:required>",
  "sourceKey": "mock-sourcekey"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateAwsEbsAssetDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.AwsEbsAsset](t, `{
  "awsEbs": {
    "isEncrypted": false,
    "isMultiAttachEnabled": false,
    "sizeInGiBs": 1,
    "volumeKey": "mock-volumekey",
    "volumeType": "mock-volumetype"
  },
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "externalAssetKey": "mock-externalassetkey",
  "id": "<ocid:1>",
  "inventoryId": "<ocid:required>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "sourceKey": "mock-sourcekey",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.AwsEbsAsset](t, `{
  "awsEbs": {
    "isEncrypted": false,
    "isMultiAttachEnabled": false,
    "sizeInGiBs": 1,
    "volumeKey": "mock-volumekey",
    "volumeType": "mock-volumetype"
  },
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "externalAssetKey": "mock-externalassetkey",
  "id": "<ocid:1>",
  "inventoryId": "<ocid:required>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "sourceKey": "mock-sourcekey",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)

	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.AwsEbsAsset, sdksvc.CreateAwsEbsAssetDetails, sdksvc.UpdateAwsEbsAssetDetails]{
		CollectionPath: "/20220509/assets", ItemPath: "/20220509/assets/<ocid:1>",
		CreatePath: "/20220509/assets", CreateMethod: http.MethodPost,
		UpdatePath: "/20220509/assets/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220509/assets/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},

		CreatedState: &createdState,

		UpdatedState:      &updatedState,
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "assetType", "AWS_EBS", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "assetType", "AWS_EBS", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudbridge.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220509", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.InventoryClient{BaseClient: session.BaseClient()}
	manager := &AssetServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAssetRuntimeHooks(manager, sdkClient)
	client := wrapAssetGeneratedClient(hooks, defaultAssetServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Asset](buildAssetGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Asset]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Asset) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Asset status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Asset) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Asset) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Asset status = %+v", current.Status)
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

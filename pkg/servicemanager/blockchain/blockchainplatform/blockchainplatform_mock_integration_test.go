/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package blockchainplatform

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/blockchain"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/blockchain/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationBlockchainPlatformCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.BlockchainPlatform](t, `
{
  "metadata": {"name": "mock-blockchainplatform", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "computeShape": "STANDARD",
  "description": "mock-description-initial",
  "displayName": "mock-displayname",
  "idcsAccessToken": "mock-idcsaccesstoken",
  "isByol": true,
  "platformRole": "FOUNDER",
  "platformVersion": "mock-platformversion-updated"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-blockchainplatform")
	resource.Status = apiv1beta1.BlockchainPlatformStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateBlockchainPlatformDetails](t, `{
  "compartmentId": "<ocid:required>",
  "computeShape": "STANDARD",
  "description": "mock-description-initial",
  "displayName": "mock-displayname",
  "idcsAccessToken": "mock-idcsaccesstoken",
  "isByol": true,
  "platformRole": "FOUNDER",
  "platformVersion": "mock-platformversion-updated"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateBlockchainPlatformDetails](t, `{
  "description": "mock-description-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.BlockchainPlatform](t, `{
  "compartmentId": "<ocid:required>",
  "computeShape": "STANDARD",
  "description": "mock-description-initial",
  "displayName": "mock-displayname",
  "id": "<ocid:1>",
  "idcsAccessToken": "mock-idcsaccesstoken",
  "isByol": true,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "platformRole": "FOUNDER",
  "platformVersion": "mock-platformversion-updated",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.BlockchainPlatform](t, `{
  "compartmentId": "<ocid:required>",
  "computeShape": "STANDARD",
  "description": "mock-description-updated",
  "displayName": "mock-displayname",
  "id": "<ocid:1>",
  "idcsAccessToken": "mock-idcsaccesstoken",
  "isByol": true,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "platformRole": "FOUNDER",
  "platformVersion": "mock-platformversion-updated",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEBLOCKCHAINPLATFORM",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "BlockchainPlatform", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEBLOCKCHAINPLATFORM",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "BlockchainPlatform", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEBLOCKCHAINPLATFORM",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "BlockchainPlatform", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.BlockchainPlatform, sdksvc.CreateBlockchainPlatformDetails, sdksvc.UpdateBlockchainPlatformDetails]{
		CollectionPath: "/20191010/blockchainPlatforms", ItemPath: "/20191010/blockchainPlatforms/<ocid:1>",
		CreatePath: "/20191010/blockchainPlatforms", CreateMethod: http.MethodPost,
		UpdatePath: "/20191010/blockchainPlatforms/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20191010/blockchainPlatforms/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateBlockchainPlatformDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateBlockchainPlatformDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateBlockchainPlatformDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20191010/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20191010/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20191010/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://blockchain.us-ashburn-1.oci.oraclecloud.com", BasePath: "20191010", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.BlockchainPlatformClient{BaseClient: session.BaseClient()}
	manager := &BlockchainPlatformServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newBlockchainPlatformRuntimeHooks(manager, sdkClient)
	client := wrapBlockchainPlatformGeneratedClient(hooks, defaultBlockchainPlatformServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.BlockchainPlatform](buildBlockchainPlatformGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.BlockchainPlatform]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.BlockchainPlatform) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.Description != "mock-description-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created BlockchainPlatform status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.BlockchainPlatform) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "mock-description-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.BlockchainPlatform) error {
			if current.Status.Description != "mock-description-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated BlockchainPlatform status = %+v", current.Status)
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

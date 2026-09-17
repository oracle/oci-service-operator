/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package enterprisemanagerbridge

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationEnterpriseManagerBridgeWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.EnterpriseManagerBridge](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "enterprise-manager-bridge",
    "namespace": "default",
    "uid": "uid-enterprise-manager-bridge"
  },
  "spec": {
    "compartmentId": "<ocid:1>",
    "description": "bridge description",
    "displayName": "em-bridge",
    "objectStorageBucketName": "em-bucket"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-enterprisemanagerbridge")
	resource.Status = opsiv1beta1.EnterpriseManagerBridgeStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateEnterpriseManagerBridgeDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "bridge description",
  "displayName": "em-bridge",
  "objectStorageBucketName": "em-bucket"
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateEnterpriseManagerBridgeDetails](t, `
{
  "description": "updated bridge"
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.EnterpriseManagerBridge](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "bridge description",
  "displayName": "em-bridge",
  "freeformTags": {},
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "objectStorageBucketName": "em-bucket",
  "objectStorageBucketStatusDetails": null,
  "objectStorageNamespaceName": "object-storage-namespace",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.EnterpriseManagerBridge](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "updated bridge",
  "displayName": "em-bridge",
  "freeformTags": {},
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "objectStorageBucketName": "em-bucket",
  "objectStorageBucketStatusDetails": null,
  "objectStorageNamespaceName": "object-storage-namespace",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null
}
`)
	deletedState := ocimock.MustOCIResponseFixture[opsisdk.EnterpriseManagerBridge](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "updated bridge",
  "displayName": "em-bridge",
  "freeformTags": {},
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "objectStorageBucketName": "em-bucket",
  "objectStorageBucketStatusDetails": null,
  "objectStorageNamespaceName": "object-storage-namespace",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "EnterpriseManagerBridge",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "EnterpriseManagerBridge",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "EnterpriseManagerBridge",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.EnterpriseManagerBridge, opsisdk.CreateEnterpriseManagerBridgeDetails, opsisdk.UpdateEnterpriseManagerBridgeDetails]{
		CollectionPath: "/20200630/enterpriseManagerBridges", ItemPath: "/20200630/enterpriseManagerBridges/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ opsisdk.CreateEnterpriseManagerBridgeDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://opsi.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := opsisdk.OperationsInsightsClient{BaseClient: session.BaseClient()}
	client := newEnterpriseManagerBridgeServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.EnterpriseManagerBridge]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.EnterpriseManagerBridge) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created EnterpriseManagerBridge status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.EnterpriseManagerBridge) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "updated bridge"
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.EnterpriseManagerBridge) error {
			if !(current.Status.Description == "updated bridge") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated EnterpriseManagerBridge status = %+v", current.Status)
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

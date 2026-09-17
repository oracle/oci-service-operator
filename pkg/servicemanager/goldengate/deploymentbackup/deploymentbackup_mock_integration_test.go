/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package deploymentbackup

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/goldengate"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/goldengate/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationDeploymentBackupCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.DeploymentBackup](t, `
{
  "metadata": {"name": "mock-deploymentbackup", "namespace": "default"},
  "spec": {
  "bucketName": "mock-bucketname",
  "compartmentId": "<ocid:required>",
  "deploymentId": "<ocid:required>",
  "displayName": "mock-displayname",
  "freeformTags": {
    "mock": "initial"
  },
  "isMetadataOnly": true,
  "namespaceName": "mock-namespacename",
  "objectName": "mock-objectname"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-deploymentbackup")
	resource.Status = apiv1beta1.DeploymentBackupStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateDeploymentBackupDetails](t, `{
  "bucketName": "mock-bucketname",
  "compartmentId": "<ocid:required>",
  "deploymentId": "<ocid:required>",
  "displayName": "mock-displayname",
  "freeformTags": {
    "mock": "initial"
  },
  "isMetadataOnly": true,
  "namespaceName": "mock-namespacename",
  "objectName": "mock-objectname"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateDeploymentBackupDetails](t, `{
  "freeformTags": {
    "mock": "updated"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.DeploymentBackup](t, `{
  "bucketName": "mock-bucketname",
  "compartmentId": "<ocid:required>",
  "deploymentId": "<ocid:required>",
  "displayName": "mock-displayname",
  "freeformTags": {
    "mock": "initial"
  },
  "id": "<ocid:1>",
  "isMetadataOnly": true,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "namespaceName": "mock-namespacename",
  "objectName": "mock-objectname",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.DeploymentBackup](t, `{
  "bucketName": "mock-bucketname",
  "compartmentId": "<ocid:required>",
  "deploymentId": "<ocid:required>",
  "displayName": "mock-displayname",
  "freeformTags": {
    "mock": "updated"
  },
  "id": "<ocid:1>",
  "isMetadataOnly": true,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "namespaceName": "mock-namespacename",
  "objectName": "mock-objectname",
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
  "operationType": "CREATEDEPLOYMENTBACKUP",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "DeploymentBackup", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEDEPLOYMENTBACKUP",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "DeploymentBackup", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.DeploymentBackup, sdksvc.CreateDeploymentBackupDetails, sdksvc.UpdateDeploymentBackupDetails]{
		CollectionPath: "/20200407/deploymentBackups", ItemPath: "/20200407/deploymentBackups/<ocid:1>",
		CreatePath: "/20200407/deploymentBackups", CreateMethod: http.MethodPost,
		UpdatePath: "/20200407/deploymentBackups/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20200407/deploymentBackups/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateDeploymentBackupDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateDeploymentBackupDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 200, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateDeploymentBackupDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://goldengate.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200407", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.GoldenGateClient{BaseClient: session.BaseClient()}
	manager := &DeploymentBackupServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDeploymentBackupRuntimeHooks(manager, sdkClient)
	client := wrapDeploymentBackupGeneratedClient(hooks, defaultDeploymentBackupServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.DeploymentBackup](buildDeploymentBackupGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.DeploymentBackup]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.DeploymentBackup) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.FreeformTags["mock"] != "initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DeploymentBackup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.DeploymentBackup) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "freeformTags": {
    "mock": "updated"
  }
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.DeploymentBackup) error {
			if current.Status.FreeformTags["mock"] != "updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DeploymentBackup status = %+v", current.Status)
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

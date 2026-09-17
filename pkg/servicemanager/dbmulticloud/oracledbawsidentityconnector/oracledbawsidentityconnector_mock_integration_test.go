/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package oracledbawsidentityconnector

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/dbmulticloud"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/dbmulticloud/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationOracleDbAwsIdentityConnectorCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.OracleDbAwsIdentityConnector](t, `
{
  "metadata": {"name": "mock-oracledbawsidentityconnector", "namespace": "default"},
  "spec": {
  "awsLocation": "mock-awslocation",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "issuerUrl": "mock-issuerurl",
  "oidcScope": "mock-oidcscope",
  "resourceId": "<ocid:required>",
  "serviceRoleDetails": [
    {
      "roleArn": "mock-rolearn",
      "servicePrivateEndpoint": "mock-serviceprivateendpoint",
      "serviceType": "KMS"
    }
  ]
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-oracledbawsidentityconnector")
	resource.Status = apiv1beta1.OracleDbAwsIdentityConnectorStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateOracleDbAwsIdentityConnectorDetails](t, `{
  "awsLocation": "mock-awslocation",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "issuerUrl": "mock-issuerurl",
  "oidcScope": "mock-oidcscope",
  "resourceId": "<ocid:required>",
  "serviceRoleDetails": [
    {
      "roleArn": "mock-rolearn",
      "servicePrivateEndpoint": "mock-serviceprivateendpoint",
      "serviceType": "KMS"
    }
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateOracleDbAwsIdentityConnectorDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.OracleDbAwsIdentityConnector](t, `{
  "awsLocation": "mock-awslocation",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "issuerUrl": "mock-issuerurl",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "oidcScope": "mock-oidcscope",
  "resourceId": "<ocid:required>",
  "serviceRoleDetails": [
    {
      "roleArn": "mock-rolearn",
      "servicePrivateEndpoint": "mock-serviceprivateendpoint",
      "serviceType": "KMS"
    }
  ],
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.OracleDbAwsIdentityConnector](t, `{
  "awsLocation": "mock-awslocation",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "issuerUrl": "mock-issuerurl",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "oidcScope": "mock-oidcscope",
  "resourceId": "<ocid:required>",
  "serviceRoleDetails": [
    {
      "roleArn": "mock-rolearn",
      "servicePrivateEndpoint": "mock-serviceprivateendpoint",
      "serviceType": "KMS"
    }
  ],
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
  "operationType": "CREATEORACLEDBAWSIDENTITYCONNECTOR",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "OracleDbAwsIdentityConnector", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEORACLEDBAWSIDENTITYCONNECTOR",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "OracleDbAwsIdentityConnector", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEORACLEDBAWSIDENTITYCONNECTOR",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "OracleDbAwsIdentityConnector", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.OracleDbAwsIdentityConnector, sdksvc.CreateOracleDbAwsIdentityConnectorDetails, sdksvc.UpdateOracleDbAwsIdentityConnectorDetails]{
		CollectionPath: "/20240501/oracleDbAwsIdentityConnector", ItemPath: "/20240501/oracleDbAwsIdentityConnector/<ocid:1>",
		CreatePath: "/20240501/oracleDbAwsIdentityConnector", CreateMethod: http.MethodPost,
		UpdatePath: "/20240501/oracleDbAwsIdentityConnector/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20240501/oracleDbAwsIdentityConnector/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateOracleDbAwsIdentityConnectorDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateOracleDbAwsIdentityConnectorDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateOracleDbAwsIdentityConnectorDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20240501/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20240501/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20240501/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dbmulticloud.us-ashburn-1.oci.oraclecloud.com", BasePath: "20240501", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := OracleDbAwsIdentityConnectorSDKClients{dbMulticloudAwsProviderClient: sdksvc.DbMulticloudAwsProviderClient{BaseClient: session.BaseClient()}, workRequestClient: sdksvc.WorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &OracleDbAwsIdentityConnectorServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newOracleDbAwsIdentityConnectorRuntimeHooks(manager, sdkClient)
	client := wrapOracleDbAwsIdentityConnectorGeneratedClient(hooks, defaultOracleDbAwsIdentityConnectorServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.OracleDbAwsIdentityConnector](buildOracleDbAwsIdentityConnectorGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.OracleDbAwsIdentityConnector]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.OracleDbAwsIdentityConnector) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OracleDbAwsIdentityConnector status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.OracleDbAwsIdentityConnector) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.OracleDbAwsIdentityConnector) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OracleDbAwsIdentityConnector status = %+v", current.Status)
			}
			return nil
		},
		ValidateStable: func(*apiv1beta1.OracleDbAwsIdentityConnector) error {
			if got := responder.OperationCounts()[ocimock.OperationUpdate]; got != 1 {
				return fmt.Errorf("stable OracleDbAwsIdentityConnector update calls = %d, want 1", got)
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

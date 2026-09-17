/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package sensitivecolumn

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationSensitiveColumnWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[datasafev1beta1.SensitiveColumn](t, `
{
  "metadata": {
    "annotations": {
      "datasafe.oracle.com/sensitive-data-model-id": "ocid1.sensitivedatamodel.oc1..example"
    },
    "creationTimestamp": null,
    "name": "sensitive-column",
    "namespace": "default",
    "uid": "sensitive-column-uid"
  },
  "spec": {
    "appName": "APP",
    "columnName": "SSN",
    "dataType": "VARCHAR2",
    "objectName": "CUSTOMERS",
    "objectType": "TABLE",
    "parentColumnKeys": [
      "parent-1"
    ],
    "relationType": "NONE",
    "schemaName": "APP",
    "sensitiveTypeId": "<ocid:1>",
    "status": "VALID"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-sensitivecolumn")
	resource.Status = datasafev1beta1.SensitiveColumnStatus{}
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateSensitiveColumnDetails](t, `
{
  "appName": "APP",
  "columnName": "SSN",
  "dataType": "VARCHAR2",
  "objectName": "CUSTOMERS",
  "objectType": "TABLE",
  "parentColumnKeys": [
    "parent-1"
  ],
  "relationType": "NONE",
  "schemaName": "APP",
  "sensitiveTypeId": "<ocid:1>",
  "status": "VALID"
}
`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateSensitiveColumnDetails](t, `
{
  "sensitiveTypeId": "ocid1.sensitivetype.oc1..updated"
}
`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.SensitiveColumn](t, `
{
  "appDefinedChildColumnKeys": null,
  "appName": "APP",
  "columnGroups": null,
  "columnName": "SSN",
  "confidenceLevelDetails": null,
  "dataType": "VARCHAR2",
  "dbDefinedChildColumnKeys": null,
  "estimatedDataValueCount": 10,
  "id": "42",
  "key": "42",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "objectName": "CUSTOMERS",
  "objectType": "TABLE",
  "parentColumnKeys": [
    "parent-1"
  ],
  "relationType": "NONE",
  "sampleDataValues": null,
  "schemaName": "APP",
  "sensitiveDataModelId": "ocid1.sensitivedatamodel.oc1..example",
  "sensitiveTypeId": "<ocid:1>",
  "source": "MANUAL",
  "status": "VALID",
  "timeCreated": null,
  "timeUpdated": null
}
`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.SensitiveColumn](t, `
{
  "appDefinedChildColumnKeys": null,
  "appName": "APP",
  "columnGroups": null,
  "columnName": "SSN",
  "confidenceLevelDetails": null,
  "dataType": "VARCHAR2",
  "dbDefinedChildColumnKeys": null,
  "estimatedDataValueCount": 10,
  "id": "42",
  "key": "42",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "objectName": "CUSTOMERS",
  "objectType": "TABLE",
  "parentColumnKeys": [
    "parent-1"
  ],
  "relationType": "NONE",
  "sampleDataValues": null,
  "schemaName": "APP",
  "sensitiveDataModelId": "ocid1.sensitivedatamodel.oc1..example",
  "sensitiveTypeId": "ocid1.sensitivetype.oc1..updated",
  "source": "MANUAL",
  "status": "VALID",
  "timeCreated": null,
  "timeUpdated": null
}
`)
	deletedState := ocimock.MustOCIResponseFixture[datasafesdk.SensitiveColumn](t, `
{
  "appDefinedChildColumnKeys": null,
  "appName": "APP",
  "columnGroups": null,
  "columnName": "SSN",
  "confidenceLevelDetails": null,
  "dataType": "VARCHAR2",
  "dbDefinedChildColumnKeys": null,
  "estimatedDataValueCount": 10,
  "id": "42",
  "key": "42",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "objectName": "CUSTOMERS",
  "objectType": "TABLE",
  "parentColumnKeys": [
    "parent-1"
  ],
  "relationType": "NONE",
  "sampleDataValues": null,
  "schemaName": "APP",
  "sensitiveDataModelId": "ocid1.sensitivedatamodel.oc1..example",
  "sensitiveTypeId": "ocid1.sensitivetype.oc1..updated",
  "source": "MANUAL",
  "status": "VALID",
  "timeCreated": null,
  "timeUpdated": null
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_SENSITIVE_COLUMN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "SensitiveColumn",
      "identifier": "42"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_SENSITIVE_COLUMN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "SensitiveColumn",
      "identifier": "42"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.SensitiveColumn, datasafesdk.CreateSensitiveColumnDetails, datasafesdk.UpdateSensitiveColumnDetails]{
		CollectionPath: "/20181201/sensitiveDataModels/ocid1.sensitivedatamodel.oc1..example/sensitiveColumns", ItemPath: "/20181201/sensitiveDataModels/ocid1.sensitivedatamodel.oc1..example/sensitiveColumns/42",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateSensitiveColumnDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datasafe.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &SensitiveColumnServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSensitiveColumnDefaultRuntimeHooks(sdkClient)
	applySensitiveColumnRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapSensitiveColumnGeneratedClient(hooks, defaultSensitiveColumnServiceClient{ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SensitiveColumn](buildSensitiveColumnGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.SensitiveColumn]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.SensitiveColumn) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.SensitiveTypeId != resource.Spec.SensitiveTypeId || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created SensitiveColumn status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.SensitiveColumn) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "sensitiveTypeId": "ocid1.sensitivetype.oc1..updated"
}`)
		},
		ValidateUpdated: func(current *datasafev1beta1.SensitiveColumn) error {
			if !(current.Status.SensitiveTypeId == "ocid1.sensitivetype.oc1..updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated SensitiveColumn status = %+v", current.Status)
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

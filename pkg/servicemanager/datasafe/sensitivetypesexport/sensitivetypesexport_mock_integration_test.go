/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package sensitivetypesexport

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

const mockSensitiveTypesExportName = "osok-mock-sensitive-types-export-v1"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationSensitiveTypesExportWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &datasafev1beta1.SensitiveTypesExport{}
	ocimock.InitializeResource(resource, "mock-sensitivetypesexport")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.SensitiveTypesExportSpec](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-sensitive-types-export-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isIncludeAllSensitiveTypes": true
}
`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateSensitiveTypesExportDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-sensitive-types-export-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isIncludeAllSensitiveTypes": true
}
`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateSensitiveTypesExportDetails](t, `
{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.SensitiveTypesExport](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T03:39:37.475Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-sensitive-types-export-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "isIncludeAllSensitiveTypes": true,
  "lifecycleState": "ACTIVE",
  "sensitiveTypeIdsForExport": [],
  "systemTags": {},
  "timeCreated": "2026-09-03T03:39:37.554Z",
  "timeUpdated": "2026-09-03T03:39:42.356Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.SensitiveTypesExport](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T03:39:37.475Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-sensitive-types-export-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isIncludeAllSensitiveTypes": true,
  "lifecycleState": "ACTIVE",
  "sensitiveTypeIdsForExport": [],
  "systemTags": {},
  "timeCreated": "2026-09-03T03:39:37.554Z",
  "timeUpdated": "2026-09-03T03:39:47.248Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_SENSITIVE_TYPES_EXPORT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "sensitiveTypesExport",
      "entityUri": "/sensitiveTypesExports/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T03:39:37.652Z",
  "timeFinished": "2026-09-03T03:39:42.427Z",
  "timeStarted": "2026-09-03T03:39:42.237Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_SENSITIVE_TYPES_EXPORT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "sensitiveTypesExport",
      "entityUri": "/sensitiveTypesExports/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T03:39:44.576Z",
  "timeFinished": "2026-09-03T03:39:47.277Z",
  "timeStarted": "2026-09-03T03:39:46.965Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.SensitiveTypesExport, datasafesdk.CreateSensitiveTypesExportDetails, datasafesdk.UpdateSensitiveTypesExportDetails]{
		CollectionPath: "/20181201/sensitiveTypesExports", ItemPath: "/20181201/sensitiveTypesExports/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateSensitiveTypesExportDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:2>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datasafe.us-ashburn-1.oci.oraclecloud.com", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &SensitiveTypesExportServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSensitiveTypesExportDefaultRuntimeHooks(sdkClient)
	applySensitiveTypesExportRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapSensitiveTypesExportGeneratedClient(hooks, defaultSensitiveTypesExportServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SensitiveTypesExport](buildSensitiveTypesExportGeneratedRuntimeConfig(manager, hooks)),
	})
	mockValidateCreated := func(current *datasafev1beta1.SensitiveTypesExport) error {
		if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != mockSensitiveTypesExportName {
			return fmt.Errorf("created SensitiveTypesExport status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *datasafev1beta1.SensitiveTypesExport) error {
		if current.Status.Description != "recorded update" || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated SensitiveTypesExport status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.SensitiveTypesExport]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.SensitiveTypesExport) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.SensitiveTypesExport) {
			current.Spec.Description = "recorded update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *datasafev1beta1.SensitiveTypesExport) error {
			if err := mockValidateUpdated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
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

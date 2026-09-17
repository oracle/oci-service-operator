/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package maskingcolumn

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
func TestMockIntegrationMaskingColumnWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[datasafev1beta1.MaskingColumn](t, `
{
  "metadata": {
    "annotations": {
      "datasafe.oracle.com/masking-policy-id": "ocid1.maskingpolicy.oc1..example"
    },
    "creationTimestamp": null,
    "name": "masking-column-alpha",
    "namespace": "default",
    "uid": "masking-column-uid"
  },
  "spec": {
    "columnName": "SSN",
    "isMaskingEnabled": false,
    "maskingColumnGroup": "group-a",
    "maskingFormats": [
      {
        "description": "fixed value",
        "formatEntries": [
          {
            "fixedString": "MASKED",
            "type": "FIXED_STRING"
          }
        ]
      }
    ],
    "objectName": "CUSTOMERS",
    "objectType": "TABLE",
    "schemaName": "APP",
    "sensitiveTypeId": "<ocid:1>"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-maskingcolumn")
	resource.Status = datasafev1beta1.MaskingColumnStatus{}
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateMaskingColumnDetails](t, `
{
  "columnName": "SSN",
  "isMaskingEnabled": false,
  "maskingColumnGroup": "group-a",
  "maskingFormats": [
    {
      "description": "fixed value",
      "formatEntries": [
        {
          "fixedString": "MASKED",
          "type": "FIXED_STRING"
        }
      ]
    }
  ],
  "objectName": "CUSTOMERS",
  "objectType": "TABLE",
  "schemaName": "APP",
  "sensitiveTypeId": "<ocid:1>"
}
`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateMaskingColumnDetails](t, `
{
  "maskingColumnGroup": "group-b"
}
`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.MaskingColumn](t, `
{
  "childColumns": null,
  "columnName": "SSN",
  "dataType": null,
  "id": "42",
  "isMaskingEnabled": false,
  "key": "42",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "maskingColumnGroup": "group-a",
  "maskingFormats": [
    {
      "condition": null,
      "description": "fixed value",
      "formatEntries": [
        {
          "description": null,
          "fixedString": "MASKED",
          "type": "FIXED_STRING"
        }
      ]
    }
  ],
  "maskingPolicyId": "ocid1.maskingpolicy.oc1..example",
  "objectName": "CUSTOMERS",
  "objectType": "TABLE",
  "schemaName": "APP",
  "sensitiveTypeId": "<ocid:1>",
  "timeCreated": null,
  "timeUpdated": null
}
`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.MaskingColumn](t, `
{
  "childColumns": null,
  "columnName": "SSN",
  "dataType": null,
  "id": "42",
  "isMaskingEnabled": false,
  "key": "42",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "maskingColumnGroup": "group-b",
  "maskingFormats": [
    {
      "condition": null,
      "description": "fixed value",
      "formatEntries": [
        {
          "description": null,
          "fixedString": "MASKED",
          "type": "FIXED_STRING"
        }
      ]
    }
  ],
  "maskingPolicyId": "ocid1.maskingpolicy.oc1..example",
  "objectName": "CUSTOMERS",
  "objectType": "TABLE",
  "schemaName": "APP",
  "sensitiveTypeId": "<ocid:1>",
  "timeCreated": null,
  "timeUpdated": null
}
`)
	deletedState := ocimock.MustOCIResponseFixture[datasafesdk.MaskingColumn](t, `
{
  "childColumns": null,
  "columnName": "SSN",
  "dataType": null,
  "id": "42",
  "isMaskingEnabled": false,
  "key": "42",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "maskingColumnGroup": "group-b",
  "maskingFormats": [
    {
      "condition": null,
      "description": "fixed value",
      "formatEntries": [
        {
          "description": null,
          "fixedString": "MASKED",
          "type": "FIXED_STRING"
        }
      ]
    }
  ],
  "maskingPolicyId": "ocid1.maskingpolicy.oc1..example",
  "objectName": "CUSTOMERS",
  "objectType": "TABLE",
  "schemaName": "APP",
  "sensitiveTypeId": "<ocid:1>",
  "timeCreated": null,
  "timeUpdated": null
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_MASKING_COLUMN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "MaskingColumn",
      "identifier": "42"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_MASKING_COLUMN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "MaskingColumn",
      "identifier": "42"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.MaskingColumn, datasafesdk.CreateMaskingColumnDetails, datasafesdk.UpdateMaskingColumnDetails]{
		CollectionPath: "/20181201/maskingPolicies/ocid1.maskingpolicy.oc1..example/maskingColumns", ItemPath: "/20181201/maskingPolicies/ocid1.maskingpolicy.oc1..example/maskingColumns/42",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateMaskingColumnDetails) error {
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
	manager := &MaskingColumnServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newMaskingColumnDefaultRuntimeHooks(sdkClient)
	applyMaskingColumnRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapMaskingColumnGeneratedClient(hooks, defaultMaskingColumnServiceClient{ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.MaskingColumn](buildMaskingColumnGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.MaskingColumn]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.MaskingColumn) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.MaskingColumnGroup != resource.Spec.MaskingColumnGroup || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created MaskingColumn status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.MaskingColumn) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "maskingColumnGroup": "group-b"
}`)
		},
		ValidateUpdated: func(current *datasafev1beta1.MaskingColumn) error {
			if !(current.Status.MaskingColumnGroup == "group-b") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated MaskingColumn status = %+v", current.Status)
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

/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package sensitivetypegroup

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
func TestMockIntegrationSensitiveTypeGroupWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[datasafev1beta1.SensitiveTypeGroup](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "sensitive-type-group",
    "namespace": "default"
  },
  "spec": {
    "compartmentId": "<ocid:1>",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "description": "customer sensitive type group",
    "displayName": "customer-sensitive-types",
    "freeformTags": {
      "owner": "data-safe"
    }
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-sensitivetypegroup")
	resource.Status = datasafev1beta1.SensitiveTypeGroupStatus{}
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateSensitiveTypeGroupDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "customer sensitive type group",
  "displayName": "customer-sensitive-types",
  "freeformTags": {
    "owner": "data-safe"
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateSensitiveTypeGroupDetails](t, `
{
  "description": "updated sensitive type group"
}
`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.SensitiveTypeGroup](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "customer sensitive type group",
  "displayName": "customer-sensitive-types",
  "freeformTags": {
    "owner": "data-safe"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "sensitiveTypeCount": 3,
  "systemTags": null,
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.SensitiveTypeGroup](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated sensitive type group",
  "displayName": "customer-sensitive-types",
  "freeformTags": {
    "owner": "data-safe"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "sensitiveTypeCount": 3,
  "systemTags": null,
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[datasafesdk.SensitiveTypeGroup](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated sensitive type group",
  "displayName": "customer-sensitive-types",
  "freeformTags": {
    "owner": "data-safe"
  },
  "id": "<ocid:3>",
  "lifecycleState": "DELETED",
  "sensitiveTypeCount": 3,
  "systemTags": null,
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_SENSITIVE_TYPE_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "SensitiveTypeGroup",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_SENSITIVE_TYPE_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "SensitiveTypeGroup",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_SENSITIVE_TYPE_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "SensitiveTypeGroup",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.SensitiveTypeGroup, datasafesdk.CreateSensitiveTypeGroupDetails, datasafesdk.UpdateSensitiveTypeGroupDetails]{
		CollectionPath: "/20181201/sensitiveTypeGroups", ItemPath: "/20181201/sensitiveTypeGroups/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateSensitiveTypeGroupDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
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
	manager := &SensitiveTypeGroupServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSensitiveTypeGroupDefaultRuntimeHooks(sdkClient)
	applySensitiveTypeGroupRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapSensitiveTypeGroupGeneratedClient(hooks, defaultSensitiveTypeGroupServiceClient{ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SensitiveTypeGroup](buildSensitiveTypeGroupGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.SensitiveTypeGroup]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.SensitiveTypeGroup) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created SensitiveTypeGroup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.SensitiveTypeGroup) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "updated sensitive type group"
}`)
		},
		ValidateUpdated: func(current *datasafev1beta1.SensitiveTypeGroup) error {
			if !(current.Status.Description == "updated sensitive type group") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated SensitiveTypeGroup status = %+v", current.Status)
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

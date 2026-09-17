/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package namedcredential

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	managementagentsdk "github.com/oracle/oci-go-sdk/v65/managementagent"
	managementagentv1beta1 "github.com/oracle/oci-service-operator/api/managementagent/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationNamedCredentialWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[managementagentv1beta1.NamedCredential](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "db-credential",
    "namespace": "default"
  },
  "spec": {
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "description": "initial description",
    "freeformTags": {
      "env": "test"
    },
    "managementAgentId": "<ocid:1>",
    "name": "db-credential",
    "properties": [
      {
        "name": "username",
        "value": "app-user",
        "valueCategory": "CLEAR_TEXT"
      },
      {
        "name": "password",
        "value": "<ocid:2>",
        "valueCategory": "SECRET_IDENTIFIER"
      }
    ],
    "type": "DB"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-namedcredential")
	resource.Status = managementagentv1beta1.NamedCredentialStatus{}
	createRequest := ocimock.MustJSONFixture[managementagentsdk.CreateNamedCredentialDetails](t, `
{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "freeformTags": {
    "env": "test"
  },
  "managementAgentId": "<ocid:1>",
  "name": "db-credential",
  "properties": [
    {
      "name": "username",
      "value": "app-user",
      "valueCategory": "CLEAR_TEXT"
    },
    {
      "name": "password",
      "value": "<ocid:2>",
      "valueCategory": "SECRET_IDENTIFIER"
    }
  ],
  "type": "DB"
}
`)
	updateRequest := ocimock.MustJSONFixture[managementagentsdk.UpdateNamedCredentialDetails](t, `
{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated named credential",
  "freeformTags": {
    "env": "test"
  },
  "properties": [
    {
      "name": "username",
      "value": "app-user",
      "valueCategory": "CLEAR_TEXT"
    },
    {
      "name": "password",
      "value": "<ocid:2>",
      "valueCategory": "SECRET_IDENTIFIER"
    }
  ]
}
`)
	createdState := ocimock.MustOCIResponseFixture[managementagentsdk.NamedCredential](t, `
{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "managementAgentId": "<ocid:1>",
  "name": "db-credential",
  "properties": [
    {
      "name": "username",
      "value": "app-user",
      "valueCategory": "CLEAR_TEXT"
    },
    {
      "name": "password",
      "value": "<ocid:2>",
      "valueCategory": "SECRET_IDENTIFIER"
    }
  ],
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null,
  "type": "DB"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[managementagentsdk.NamedCredential](t, `
{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated named credential",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "managementAgentId": "<ocid:1>",
  "name": "db-credential",
  "properties": [
    {
      "name": "username",
      "value": "app-user",
      "valueCategory": "CLEAR_TEXT"
    },
    {
      "name": "password",
      "value": "<ocid:2>",
      "valueCategory": "SECRET_IDENTIFIER"
    }
  ],
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null,
  "type": "DB"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[managementagentsdk.NamedCredential](t, `
{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated named credential",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:4>",
  "lifecycleState": "DELETED",
  "managementAgentId": "<ocid:1>",
  "name": "db-credential",
  "properties": [
    {
      "name": "username",
      "value": "app-user",
      "valueCategory": "CLEAR_TEXT"
    },
    {
      "name": "password",
      "value": "<ocid:2>",
      "valueCategory": "SECRET_IDENTIFIER"
    }
  ],
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null,
  "type": "DB"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[managementagentsdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_NAMEDCREDENTIALS",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "NamedCredential",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[managementagentsdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_NAMEDCREDENTIALS",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "NamedCredential",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[managementagentsdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_NAMEDCREDENTIALS",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "NamedCredential",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[managementagentsdk.NamedCredential, managementagentsdk.CreateNamedCredentialDetails, managementagentsdk.UpdateNamedCredentialDetails]{
		CollectionPath: "/20200202/namedCredentials", ItemPath: "/20200202/namedCredentials/<ocid:4>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ managementagentsdk.CreateNamedCredentialDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200202/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200202/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200202/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://managementagent.mock.invalid", BasePath: "20200202", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := managementagentsdk.ManagementAgentClient{BaseClient: session.BaseClient()}
	client := newNamedCredentialServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*managementagentv1beta1.NamedCredential]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *managementagentv1beta1.NamedCredential) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created NamedCredential status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *managementagentv1beta1.NamedCredential) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "updated named credential"
}`)
		},
		ValidateUpdated: func(current *managementagentv1beta1.NamedCredential) error {
			if !(current.Status.Description == "updated named credential") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated NamedCredential status = %+v", current.Status)
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

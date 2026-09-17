/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package targetalertpolicyassociation

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
func TestMockIntegrationTargetAlertPolicyAssociationWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[datasafev1beta1.TargetAlertPolicyAssociation](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "target-alert-policy-association",
    "namespace": "default"
  },
  "spec": {
    "compartmentId": "<ocid:1>",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "description": "customer target alert policy association",
    "displayName": "customer-target-alert-policy",
    "freeformTags": {
      "owner": "runtime"
    },
    "isEnabled": true,
    "policyId": "<ocid:2>",
    "targetId": "<ocid:3>"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-targetalertpolicyassociation")
	resource.Status = datasafev1beta1.TargetAlertPolicyAssociationStatus{}
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateTargetAlertPolicyAssociationDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "customer target alert policy association",
  "displayName": "customer-target-alert-policy",
  "freeformTags": {
    "owner": "runtime"
  },
  "isEnabled": true,
  "policyId": "<ocid:2>",
  "targetId": "<ocid:3>"
}
`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateTargetAlertPolicyAssociationDetails](t, `
{
  "description": "updated association"
}
`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.TargetAlertPolicyAssociation](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "customer target alert policy association",
  "displayName": "customer-target-alert-policy",
  "freeformTags": {
    "owner": "runtime"
  },
  "id": "<ocid:5>",
  "isEnabled": true,
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "policyId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "targetId": "<ocid:3>",
  "timeCreated": "2026-05-05T10:00:00Z",
  "timeUpdated": "2026-05-05T10:05:00Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.TargetAlertPolicyAssociation](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated association",
  "displayName": "customer-target-alert-policy",
  "freeformTags": {
    "owner": "runtime"
  },
  "id": "<ocid:5>",
  "isEnabled": true,
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "policyId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "targetId": "<ocid:3>",
  "timeCreated": "2026-05-05T10:00:00Z",
  "timeUpdated": "2026-05-05T10:05:00Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[datasafesdk.TargetAlertPolicyAssociation](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated association",
  "displayName": "customer-target-alert-policy",
  "freeformTags": {
    "owner": "runtime"
  },
  "id": "<ocid:5>",
  "isEnabled": true,
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "DELETED",
  "policyId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "targetId": "<ocid:3>",
  "timeCreated": "2026-05-05T10:00:00Z",
  "timeUpdated": "2026-05-05T10:05:00Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "TargetAlertPolicyAssociation",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "TargetAlertPolicyAssociation",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "TargetAlertPolicyAssociation",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.TargetAlertPolicyAssociation, datasafesdk.CreateTargetAlertPolicyAssociationDetails, datasafesdk.UpdateTargetAlertPolicyAssociationDetails]{
		CollectionPath: "/20181201/targetAlertPolicyAssociations", ItemPath: "/20181201/targetAlertPolicyAssociations/<ocid:5>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateTargetAlertPolicyAssociationDetails) error {
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
	manager := &TargetAlertPolicyAssociationServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTargetAlertPolicyAssociationDefaultRuntimeHooks(sdkClient)
	applyTargetAlertPolicyAssociationRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapTargetAlertPolicyAssociationGeneratedClient(hooks, defaultTargetAlertPolicyAssociationServiceClient{ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.TargetAlertPolicyAssociation](buildTargetAlertPolicyAssociationGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.TargetAlertPolicyAssociation]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.TargetAlertPolicyAssociation) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created TargetAlertPolicyAssociation status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.TargetAlertPolicyAssociation) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "updated association"
}`)
		},
		ValidateUpdated: func(current *datasafev1beta1.TargetAlertPolicyAssociation) error {
			if !(current.Status.Description == "updated association") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated TargetAlertPolicyAssociation status = %+v", current.Status)
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

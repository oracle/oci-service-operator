/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package opainstance

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	opasdk "github.com/oracle/oci-go-sdk/v65/opa"
	opav1beta1 "github.com/oracle/oci-service-operator/api/opa/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationOpaInstanceWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &opav1beta1.OpaInstance{}
	ocimock.InitializeResource(resource, "mock-opainstance")
	resource.Spec = ocimock.MustJSONFixture[opav1beta1.OpaInstanceSpec](t, `{
  "compartmentId": "<ocid:1>",
  "consumptionModel": "UCM",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "opa instance description",
  "displayName": "opa-instance",
  "freeformTags": {
    "environment": "dev"
  },
  "idcsAt": "opaque-idcs-token",
  "isBreakglassEnabled": true,
  "meteringType": "USERS",
  "shapeName": "DEVELOPMENT"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "opa instance updated"
}`)

	createRequest := ocimock.MustJSONFixture[opasdk.CreateOpaInstanceDetails](t, `{
  "compartmentId": "<ocid:1>",
  "consumptionModel": "UCM",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "opa instance description",
  "displayName": "opa-instance",
  "freeformTags": {
    "environment": "dev"
  },
  "idcsAt": "opaque-idcs-token",
  "isBreakglassEnabled": true,
  "meteringType": "USERS",
  "shapeName": "DEVELOPMENT"
}`)
	updateRequest := ocimock.MustJSONFixture[opasdk.UpdateOpaInstanceDetails](t, `{
  "description": "opa instance updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[opasdk.OpaInstance](t, `{
  "id": "<ocid:3>",
  "displayName": "opa-instance",
  "compartmentId": "<ocid:1>",
  "shapeName": "DEVELOPMENT",
  "lifecycleState": "ACTIVE",
  "description": "opa instance description",
  "consumptionModel": "UCM",
  "meteringType": "USERS",
  "isBreakglassEnabled": true,
  "freeformTags": {
    "environment": "dev"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[opasdk.OpaInstance](t, `{
  "id": "<ocid:3>",
  "displayName": "opa-instance",
  "compartmentId": "<ocid:1>",
  "shapeName": "DEVELOPMENT",
  "lifecycleState": "ACTIVE",
  "description": "opa instance updated",
  "consumptionModel": "UCM",
  "meteringType": "USERS",
  "isBreakglassEnabled": true,
  "freeformTags": {
    "environment": "dev"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  }
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[opasdk.WorkRequest](t, `{
  "id": "<ocid:2>",
  "operationType": "CREATE_OPA_INSTANCE",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "OpaInstance",
      "actionType": "CREATED",
      "identifier": "<ocid:3>"
    }
  ],
  "percentComplete": 100
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[opasdk.WorkRequest](t, `{
  "id": "<ocid:4>",
  "operationType": "UPDATE_OPA_INSTANCE",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "OpaInstance",
      "actionType": "UPDATED",
      "identifier": "<ocid:3>"
    }
  ],
  "percentComplete": 100
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[opasdk.WorkRequest](t, `{
  "id": "<ocid:5>",
  "operationType": "DELETE_OPA_INSTANCE",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:1>",
  "resources": [
    {
      "entityType": "OpaInstance",
      "actionType": "DELETED",
      "identifier": "<ocid:3>"
    }
  ],
  "percentComplete": 100
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opasdk.OpaInstance, opasdk.CreateOpaInstanceDetails, opasdk.UpdateOpaInstanceDetails]{
		CollectionPath: "/20210621/opaInstances", ItemPath: "/20210621/opaInstances/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ opasdk.CreateOpaInstanceDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20210621/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest),
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20210621/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest),
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20210621/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://process.us-ashburn-1.oci.oraclecloud.com", BasePath: "20210621", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := opasdk.OpaInstanceClient{BaseClient: session.BaseClient()}
	client := newOpaInstanceServiceClientWithOCIClient(log, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opav1beta1.OpaInstance]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opav1beta1.OpaInstance) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OpaInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opav1beta1.OpaInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *opav1beta1.OpaInstance) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OpaInstance status = %+v", current.Status)
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

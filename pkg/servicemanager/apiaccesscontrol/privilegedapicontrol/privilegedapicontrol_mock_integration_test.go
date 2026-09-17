/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package privilegedapicontrol

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	apiaccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/apiaccesscontrol"
	apiaccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/apiaccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationPrivilegedApiControlWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &apiaccesscontrolv1beta1.PrivilegedApiControl{}
	ocimock.InitializeResource(resource, "mock-privilegedapicontrol")
	resource.Spec = ocimock.MustJSONFixture[apiaccesscontrolv1beta1.PrivilegedApiControlSpec](t, `{
  "approverGroupIdList": [
    "<ocid:1>",
    "<ocid:2>"
  ],
  "compartmentId": "<ocid:3>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "protect launch instance",
  "displayName": "privileged-api-control-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "notificationTopicId": "<ocid:4>",
  "numberOfApprovers": 2,
  "privilegedOperationList": [
    {
      "apiName": "LaunchInstance",
      "attributeNames": [
        "shape"
      ],
      "entityType": "instance"
    }
  ],
  "resourceType": "core-instance",
  "resources": [
    "<ocid:5>"
  ]
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "privileged API control updated"
}`)

	createRequest := ocimock.MustJSONFixture[apiaccesscontrolsdk.CreatePrivilegedApiControlDetails](t, `{
  "approverGroupIdList": [
    "<ocid:1>",
    "<ocid:2>"
  ],
  "compartmentId": "<ocid:3>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "protect launch instance",
  "displayName": "privileged-api-control-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "notificationTopicId": "<ocid:4>",
  "numberOfApprovers": 2,
  "privilegedOperationList": [
    {
      "apiName": "LaunchInstance",
      "attributeNames": [
        "shape"
      ],
      "entityType": "instance"
    }
  ],
  "resourceType": "core-instance",
  "resources": [
    "<ocid:5>"
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[apiaccesscontrolsdk.UpdatePrivilegedApiControlDetails](t, `{
  "description": "privileged API control updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[apiaccesscontrolsdk.PrivilegedApiControl](t, `{
  "approverGroupIdList": [
    "<ocid:1>",
    "<ocid:2>"
  ],
  "compartmentId": "<ocid:3>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "protect launch instance",
  "displayName": "privileged-api-control-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:7>",
  "lifecycleDetails": "reviewed runtime",
  "lifecycleState": "ACTIVE",
  "notificationTopicId": "<ocid:4>",
  "numberOfApprovers": 2,
  "privilegedOperationList": [
    {
      "apiName": "LaunchInstance",
      "attributeNames": [
        "shape"
      ],
      "entityType": "instance"
    }
  ],
  "resourceType": "core-instance",
  "resources": [
    "<ocid:5>"
  ],
  "state": "ACTIVE",
  "stateDetails": null,
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeDeleted": null,
  "timeUpdated": "1970-01-01T00:00:00Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[apiaccesscontrolsdk.PrivilegedApiControl](t, `{
  "approverGroupIdList": [
    "<ocid:1>",
    "<ocid:2>"
  ],
  "compartmentId": "<ocid:3>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "privileged API control updated",
  "displayName": "privileged-api-control-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:7>",
  "lifecycleDetails": "reviewed runtime",
  "lifecycleState": "ACTIVE",
  "notificationTopicId": "<ocid:4>",
  "numberOfApprovers": 2,
  "privilegedOperationList": [
    {
      "apiName": "LaunchInstance",
      "attributeNames": [
        "shape"
      ],
      "entityType": "instance"
    }
  ],
  "resourceType": "core-instance",
  "resources": [
    "<ocid:5>"
  ],
  "state": "ACTIVE",
  "stateDetails": null,
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeDeleted": null,
  "timeUpdated": "1970-01-01T00:00:00Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[apiaccesscontrolsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:6>",
  "operationType": "CREATE_PRIVILEGED_API_CONTROL",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "PrivilegedApiControl",
      "entityUri": null,
      "identifier": "<ocid:7>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[apiaccesscontrolsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "id": "wr-update",
  "operationType": "UPDATE_PRIVILEGED_API_CONTROL",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "PrivilegedApiControl",
      "entityUri": null,
      "identifier": "<ocid:7>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[apiaccesscontrolsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:8>",
  "operationType": "DELETE_PRIVILEGED_API_CONTROL",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "PrivilegedApiControl",
      "entityUri": null,
      "identifier": "<ocid:7>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[apiaccesscontrolsdk.PrivilegedApiControl, apiaccesscontrolsdk.CreatePrivilegedApiControlDetails, apiaccesscontrolsdk.UpdatePrivilegedApiControlDetails]{
		CollectionPath:     "/20241130/privilegedApiControls",
		ItemPath:           "/20241130/privilegedApiControls/<ocid:7>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		ListShape:          ocimock.ListShapeItems,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:8>"}},
		ValidateCreate: func(request ocimock.Request, _ apiaccesscontrolsdk.CreatePrivilegedApiControlDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20241130/workRequests/<ocid:6>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20241130/workRequests/wr-update", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20241130/workRequests/<ocid:8>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://api-access-control.us-ashburn-1.oci.oraclecloud.com", BasePath: "20241130", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	base := session.BaseClient()
	sdkClient := mockPrivilegedApiControlClient{
		PrivilegedApiControlClient:     apiaccesscontrolsdk.PrivilegedApiControlClient{BaseClient: base},
		PrivilegedApiWorkRequestClient: apiaccesscontrolsdk.PrivilegedApiWorkRequestClient{BaseClient: base},
	}
	client := newPrivilegedApiControlServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiaccesscontrolv1beta1.PrivilegedApiControl]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiaccesscontrolv1beta1.PrivilegedApiControl) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created PrivilegedApiControl status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiaccesscontrolv1beta1.PrivilegedApiControl) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *apiaccesscontrolv1beta1.PrivilegedApiControl) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated PrivilegedApiControl status = %+v", current.Status)
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

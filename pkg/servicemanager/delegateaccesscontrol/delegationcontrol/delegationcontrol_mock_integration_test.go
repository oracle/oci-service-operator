/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package delegationcontrol

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	delegateaccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/delegateaccesscontrol"
	delegateaccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/delegateaccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationDelegationControlWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &delegateaccesscontrolv1beta1.DelegationControl{}
	ocimock.InitializeResource(resource, "mock-delegationcontrol")
	resource.Spec = ocimock.MustJSONFixture[delegateaccesscontrolv1beta1.DelegationControlSpec](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "delegationSubscriptionIds": [
    "<ocid:2>",
    "<ocid:3>"
  ],
  "description": "delegate managed access to an Exadata resource",
  "displayName": "delegation-control-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "isAutoApproveDuringMaintenance": true,
  "notificationMessageFormat": "JSON",
  "notificationTopicId": "<ocid:4>",
  "numApprovalsRequired": 2,
  "preApprovedServiceProviderActionNames": [
    "PATCH_CLUSTER",
    "VIEW_CLUSTER"
  ],
  "resourceIds": [
    "<ocid:5>",
    "<ocid:6>"
  ],
  "resourceType": "VMCLUSTER"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "delegation control updated"
}`)

	createRequest := ocimock.MustJSONFixture[delegateaccesscontrolsdk.CreateDelegationControlDetails](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "delegationSubscriptionIds": [
    "<ocid:2>",
    "<ocid:3>"
  ],
  "description": "delegate managed access to an Exadata resource",
  "displayName": "delegation-control-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "isAutoApproveDuringMaintenance": true,
  "notificationMessageFormat": "JSON",
  "notificationTopicId": "<ocid:4>",
  "numApprovalsRequired": 2,
  "preApprovedServiceProviderActionNames": [
    "PATCH_CLUSTER",
    "VIEW_CLUSTER"
  ],
  "resourceIds": [
    "<ocid:5>",
    "<ocid:6>"
  ],
  "resourceType": "VMCLUSTER"
}`)
	updateRequest := ocimock.MustJSONFixture[delegateaccesscontrolsdk.UpdateDelegationControlDetails](t, `{
  "description": "delegation control updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[delegateaccesscontrolsdk.DelegationControl](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "delegationSubscriptionIds": [
    "<ocid:2>",
    "<ocid:3>"
  ],
  "description": "delegate managed access to an Exadata resource",
  "displayName": "delegation-control-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:8>",
  "isAutoApproveDuringMaintenance": true,
  "lifecycleState": "ACTIVE",
  "lifecycleStateDetails": "reviewed runtime",
  "notificationMessageFormat": "JSON",
  "notificationTopicId": "<ocid:4>",
  "numApprovalsRequired": 2,
  "preApprovedServiceProviderActionNames": [
    "PATCH_CLUSTER",
    "VIEW_CLUSTER"
  ],
  "resourceIds": [
    "<ocid:5>",
    "<ocid:6>"
  ],
  "resourceType": "VMCLUSTER",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeDeleted": null,
  "timeUpdated": "1970-01-01T00:00:00Z",
  "vaultId": null,
  "vaultKeyId": null
}`)
	updatedState := ocimock.MustOCIResponseFixture[delegateaccesscontrolsdk.DelegationControl](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "delegationSubscriptionIds": [
    "<ocid:2>",
    "<ocid:3>"
  ],
  "description": "delegation control updated",
  "displayName": "delegation-control-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:8>",
  "isAutoApproveDuringMaintenance": true,
  "lifecycleState": "ACTIVE",
  "lifecycleStateDetails": "reviewed runtime",
  "notificationMessageFormat": "JSON",
  "notificationTopicId": "<ocid:4>",
  "numApprovalsRequired": 2,
  "preApprovedServiceProviderActionNames": [
    "PATCH_CLUSTER",
    "VIEW_CLUSTER"
  ],
  "resourceIds": [
    "<ocid:5>",
    "<ocid:6>"
  ],
  "resourceType": "VMCLUSTER",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeDeleted": null,
  "timeUpdated": "1970-01-01T00:00:00Z",
  "vaultId": null,
  "vaultKeyId": null
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[delegateaccesscontrolsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:7>",
  "operationType": "CREATE_DELEGATION_CONTROL",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "DelegationControl",
      "entityUri": null,
      "identifier": "<ocid:8>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[delegateaccesscontrolsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "wr-update",
  "operationType": "UPDATE_DELEGATION_CONTROL",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "DelegationControl",
      "entityUri": null,
      "identifier": "<ocid:8>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[delegateaccesscontrolsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:9>",
  "operationType": "DELETE_DELEGATION_CONTROL",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "DelegationControl",
      "entityUri": null,
      "identifier": "<ocid:8>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[delegateaccesscontrolsdk.DelegationControl, delegateaccesscontrolsdk.CreateDelegationControlDetails, delegateaccesscontrolsdk.UpdateDelegationControlDetails]{
		CollectionPath:     "/20230801/delegationControls",
		ItemPath:           "/20230801/delegationControls/<ocid:8>",
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
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:7>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:9>"}},
		ValidateCreate: func(request ocimock.Request, _ delegateaccesscontrolsdk.CreateDelegationControlDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20230801/workRequests/<ocid:7>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20230801/workRequests/wr-update", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20230801/workRequests/<ocid:9>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://delegate-access-control.us-ashburn-1.oci.oraclecloud.com", BasePath: "20230801", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	base := session.BaseClient()
	sdkClient := mockDelegationControlClient{
		DelegateAccessControlClient: delegateaccesscontrolsdk.DelegateAccessControlClient{BaseClient: base},
		WorkRequestClient:           delegateaccesscontrolsdk.WorkRequestClient{BaseClient: base},
	}
	client := newDelegationControlServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*delegateaccesscontrolv1beta1.DelegationControl]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *delegateaccesscontrolv1beta1.DelegationControl) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DelegationControl status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *delegateaccesscontrolv1beta1.DelegationControl) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *delegateaccesscontrolv1beta1.DelegationControl) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DelegationControl status = %+v", current.Status)
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

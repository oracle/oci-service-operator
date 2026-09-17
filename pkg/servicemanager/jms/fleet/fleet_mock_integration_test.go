/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package fleet

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	jmssdk "github.com/oracle/oci-go-sdk/v65/jms"
	jmsv1beta1 "github.com/oracle/oci-service-operator/api/jms/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationFleetWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &jmsv1beta1.Fleet{}
	ocimock.InitializeResource(resource, "mock-fleet")
	resource.Spec = ocimock.MustJSONFixture[jmsv1beta1.FleetSpec](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK recorded JMS fleet",
  "displayName": "osok-mock-jms-fleet",
  "freeformTags": {
    "osok-mock": "create"
  },
  "inventoryLog": {
    "logGroupId": "<ocid:2>",
    "logId": "<ocid:3>"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded JMS fleet updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)

	createRequest := ocimock.MustJSONFixture[jmssdk.CreateFleetDetails](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK recorded JMS fleet",
  "displayName": "osok-mock-jms-fleet",
  "freeformTags": {
    "osok-mock": "create"
  },
  "inventoryLog": {
    "logGroupId": "<ocid:2>",
    "logId": "<ocid:3>"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[jmssdk.UpdateFleetDetails](t, `{
  "description": "OSOK recorded JMS fleet updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[jmssdk.Fleet](t, `{
  "approximateApplicationCount": 0,
  "approximateInstallationCount": 0,
  "approximateJavaServerCount": 0,
  "approximateJreCount": 0,
  "approximateLibraryCount": 0,
  "approximateLibraryVulnerabilityCount": 0,
  "approximateManagedInstanceCount": 0,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T19:51:21.696Z"
    }
  },
  "description": "OSOK recorded JMS fleet",
  "displayName": "osok-mock-jms-fleet",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:7>",
  "inventoryLog": {
    "logGroupId": "<ocid:2>",
    "logId": "<ocid:3>"
  },
  "isAdvancedFeaturesEnabled": false,
  "isExportSettingEnabled": false,
  "lifecycleState": "ACTIVE",
  "operationLog": null,
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-02T19:51:21.999Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[jmssdk.Fleet](t, `{
  "approximateApplicationCount": 0,
  "approximateInstallationCount": 0,
  "approximateJavaServerCount": 0,
  "approximateJreCount": 0,
  "approximateLibraryCount": 0,
  "approximateLibraryVulnerabilityCount": 0,
  "approximateManagedInstanceCount": 0,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T19:51:21.696Z"
    }
  },
  "description": "OSOK recorded JMS fleet updated",
  "displayName": "osok-mock-jms-fleet",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:7>",
  "inventoryLog": {
    "logGroupId": "<ocid:2>",
    "logId": "<ocid:3>"
  },
  "isAdvancedFeaturesEnabled": false,
  "isExportSettingEnabled": false,
  "lifecycleState": "ACTIVE",
  "operationLog": null,
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-02T19:51:21.999Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[jmssdk.Fleet](t, `{
  "approximateApplicationCount": 0,
  "approximateInstallationCount": 0,
  "approximateJavaServerCount": 0,
  "approximateJreCount": 0,
  "approximateLibraryCount": 0,
  "approximateLibraryVulnerabilityCount": 0,
  "approximateManagedInstanceCount": 0,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T19:51:21.696Z"
    }
  },
  "description": "OSOK recorded JMS fleet updated",
  "displayName": "osok-mock-jms-fleet",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:7>",
  "inventoryLog": {
    "logGroupId": "<ocid:2>",
    "logId": "<ocid:3>"
  },
  "isAdvancedFeaturesEnabled": false,
  "isExportSettingEnabled": false,
  "lifecycleState": "DELETED",
  "operationLog": null,
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-02T19:51:21.999Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[jmssdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "completedTaskCount": 1,
  "createdBy": {
    "displayName": "<redacted>",
    "id": "<redacted>"
  },
  "id": "<ocid:4>",
  "operationType": "CREATE_FLEET",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "user",
      "entityUri": "https://cloud.oracle.com/identity/users/<ocid:5>",
      "identifier": "<ocid:5>"
    },
    {
      "actionType": "CREATED",
      "entityType": "tenancy",
      "entityUri": "https://cloud.oracle.com/tenancy",
      "identifier": "<ocid:6>"
    },
    {
      "actionType": "CREATED",
      "entityType": "fleet",
      "entityUri": "/fleets/<ocid:7>",
      "identifier": "<ocid:7>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T19:51:21.999Z",
  "timeFinished": "2026-09-02T19:51:48.474Z",
  "timeLastUpdated": "2026-09-02T19:51:48.474Z",
  "timeStarted": "2026-09-02T19:51:47.889Z",
  "totalTaskCount": 1
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[jmssdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "completedTaskCount": 1,
  "createdBy": {
    "displayName": "<redacted>",
    "id": "<redacted>"
  },
  "id": "<ocid:8>",
  "operationType": "UPDATE_FLEET",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "user",
      "entityUri": "https://cloud.oracle.com/identity/users/<ocid:5>",
      "identifier": "<ocid:5>"
    },
    {
      "actionType": "UPDATED",
      "entityType": "tenancy",
      "entityUri": "https://cloud.oracle.com/tenancy",
      "identifier": "<ocid:6>"
    },
    {
      "actionType": "UPDATED",
      "entityType": "fleet",
      "entityUri": "/fleets/<ocid:7>",
      "identifier": "<ocid:7>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T19:51:53.990Z",
  "timeFinished": "2026-09-02T19:52:17.994Z",
  "timeLastUpdated": "2026-09-02T19:52:17.994Z",
  "timeStarted": "2026-09-02T19:52:17.362Z",
  "totalTaskCount": 1
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[jmssdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "completedTaskCount": 1,
  "createdBy": {
    "displayName": "<redacted>",
    "id": "<redacted>"
  },
  "id": "<ocid:9>",
  "operationType": "DELETE_FLEET",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "user",
      "entityUri": "https://cloud.oracle.com/identity/users/<ocid:5>",
      "identifier": "<ocid:5>"
    },
    {
      "actionType": "DELETED",
      "entityType": "tenancy",
      "entityUri": "https://cloud.oracle.com/tenancy",
      "identifier": "<ocid:6>"
    },
    {
      "actionType": "DELETED",
      "entityType": "fleet",
      "entityUri": "/fleets/<ocid:7>",
      "identifier": "<ocid:7>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T19:52:26.132Z",
  "timeFinished": "2026-09-02T19:52:54.361Z",
  "timeLastUpdated": "2026-09-02T19:52:54.361Z",
  "timeStarted": "2026-09-02T19:52:53.942Z",
  "totalTaskCount": 1
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[jmssdk.Fleet, jmssdk.CreateFleetDetails, jmssdk.UpdateFleetDetails]{
		CollectionPath:    "/20210610/fleets",
		ItemPath:          "/20210610/fleets/<ocid:7>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		ListShape:         ocimock.ListShapeItems,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		DeletedState:      &deletedState, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:8>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:9>"}},
		ValidateCreate: func(request ocimock.Request, _ jmssdk.CreateFleetDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20210610/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20210610/workRequests/<ocid:8>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20210610/workRequests/<ocid:9>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://javamanagement.us-ashburn-1.oci.oraclecloud.com", BasePath: "20210610", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := jmssdk.JavaManagementServiceClient{BaseClient: session.BaseClient()}
	client := newFleetServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*jmsv1beta1.Fleet]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *jmsv1beta1.Fleet) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Fleet status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *jmsv1beta1.Fleet) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *jmsv1beta1.Fleet) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Fleet status = %+v", current.Status)
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

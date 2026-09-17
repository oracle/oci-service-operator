/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package alertpolicy

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

const mockAlertPolicyName = "osok-mock-alert-policy-v2"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationAlertPolicyWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &datasafev1beta1.AlertPolicy{}
	ocimock.InitializeResource(resource, "mock-alertpolicy")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.AlertPolicySpec](t, `
{
  "alertPolicyType": "AUDITING",
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-alert-policy-v2",
  "freeformTags": {
    "osok-mock": "create"
  },
  "severity": "LOW"
}
`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateAlertPolicyDetails](t, `
{
  "alertPolicyType": "AUDITING",
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-alert-policy-v2",
  "freeformTags": {
    "osok-mock": "create"
  },
  "severity": "LOW"
}
`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateAlertPolicyDetails](t, `
{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  },
  "severity": "MEDIUM"
}
`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.AlertPolicy](t, `
{
  "alertPolicyType": "AUDITING",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T03:29:46.211Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-alert-policy-v2",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "isUserDefined": true,
  "lifecycleState": "ACTIVE",
  "severity": "LOW",
  "timeCreated": "2026-09-03T03:29:46.284Z",
  "timeUpdated": "2026-09-03T03:29:48.944Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.AlertPolicy](t, `
{
  "alertPolicyType": "AUDITING",
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "recorded update",
  "displayName": "osok-mock-alert-policy-v2",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isUserDefined": true,
  "lifecycleState": "ACTIVE",
  "severity": "MEDIUM",
  "timeCreated": "2026-09-03T03:29:46.284Z",
  "timeUpdated": "2026-09-03T03:29:58.364Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[datasafesdk.AlertPolicy](t, `
{
  "alertPolicyType": "AUDITING",
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "recorded update",
  "displayName": "osok-mock-alert-policy-v2",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isUserDefined": true,
  "lifecycleState": "DELETED",
  "severity": "MEDIUM",
  "timeCreated": "2026-09-03T03:29:46.284Z",
  "timeUpdated": "2026-09-03T03:30:07.998Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_ALERT_POLICY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "alertPolicy",
      "entityUri": "/alertPolicies/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T03:29:46.298Z",
  "timeFinished": "2026-09-03T03:29:49.118Z",
  "timeStarted": "2026-09-03T03:29:48.747Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_ALERT_POLICY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "alertPolicy",
      "entityUri": "/alertPolicies/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T03:29:52.399Z",
  "timeFinished": "2026-09-03T03:29:58.480Z",
  "timeStarted": "2026-09-03T03:29:57.276Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_ALERT_POLICY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "alertPolicy",
      "entityUri": "/alertPolicies/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T03:30:03.560Z",
  "timeFinished": "2026-09-03T03:30:08.774Z",
  "timeStarted": "2026-09-03T03:30:07.708Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.AlertPolicy, datasafesdk.CreateAlertPolicyDetails, datasafesdk.UpdateAlertPolicyDetails]{
		CollectionPath: "/20181201/alertPolicies", ItemPath: "/20181201/alertPolicies/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState,
		ListShape:    ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateAlertPolicyDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:2>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
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
	manager := &AlertPolicyServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAlertPolicyDefaultRuntimeHooks(sdkClient)
	applyAlertPolicyRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapAlertPolicyGeneratedClient(hooks, defaultAlertPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.AlertPolicy](buildAlertPolicyGeneratedRuntimeConfig(manager, hooks)),
	})
	mockValidateCreated := func(current *datasafev1beta1.AlertPolicy) error {
		if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != mockAlertPolicyName || !current.Status.IsUserDefined {
			return fmt.Errorf("created AlertPolicy status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *datasafev1beta1.AlertPolicy) error {
		if current.Status.Description != "recorded update" || current.Status.Severity != "MEDIUM" || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated AlertPolicy status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.AlertPolicy]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.AlertPolicy) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.AlertPolicy) {
			current.Spec.Description = "recorded update"
			current.Spec.Severity = "MEDIUM"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *datasafev1beta1.AlertPolicy) error {
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

/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package protectionpolicy

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	recoverysdk "github.com/oracle/oci-go-sdk/v65/recovery"
	recoveryv1beta1 "github.com/oracle/oci-service-operator/api/recovery/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationProtectionPolicyWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &recoveryv1beta1.ProtectionPolicy{}
	ocimock.InitializeResource(resource, "mock-protectionpolicy")
	resource.Spec = ocimock.MustJSONFixture[recoveryv1beta1.ProtectionPolicySpec](t, `
{
  "backupRetentionPeriodInDays": 14,
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-protection-policy",
  "freeformTags": {
    "osok-mock": "create"
  },
  "mustEnforceCloudLocality": false
}
`)
	createRequest := ocimock.MustJSONFixture[recoverysdk.CreateProtectionPolicyDetails](t, `
{
  "backupRetentionPeriodInDays": 14,
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-protection-policy",
  "freeformTags": {
    "osok-mock": "create"
  },
  "mustEnforceCloudLocality": false
}
`)
	updateRequest := ocimock.MustJSONFixture[recoverysdk.UpdateProtectionPolicyDetails](t, `
{
  "backupRetentionPeriodInDays": 15,
  "displayName": "osok-mock-protection-policy-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[recoverysdk.ProtectionPolicy](t, `
{
  "backupRetentionPeriodInDays": 14,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T17:53:56.756Z"
    }
  },
  "displayName": "osok-mock-protection-policy",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "isArchivalEnabled": false,
  "isPredefinedPolicy": false,
  "lifecycleDetails": "ProtectionPolicy was created successfully",
  "lifecycleState": "ACTIVE",
  "mustEnforceCloudLocality": false,
  "policyLockedDateTime": null,
  "schedules": null,
  "systemTags": {},
  "targetRegions": null,
  "timeCreated": "2026-09-02T17:53:57.039Z",
  "timeUpdated": "2026-09-02T17:54:16.874Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[recoverysdk.ProtectionPolicy](t, `
{
  "backupRetentionPeriodInDays": 15,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T17:53:56.756Z"
    }
  },
  "displayName": "osok-mock-protection-policy-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isArchivalEnabled": false,
  "isPredefinedPolicy": false,
  "lifecycleDetails": "ProtectionPolicy was updated successfully",
  "lifecycleState": "ACTIVE",
  "mustEnforceCloudLocality": false,
  "policyLockedDateTime": null,
  "schedules": null,
  "systemTags": {},
  "targetRegions": null,
  "timeCreated": "2026-09-02T17:53:57.039Z",
  "timeUpdated": "2026-09-02T17:54:48.640Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[recoverysdk.ProtectionPolicy](t, `
{
  "backupRetentionPeriodInDays": 15,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T17:53:56.756Z"
    }
  },
  "displayName": "osok-mock-protection-policy-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isArchivalEnabled": false,
  "isPredefinedPolicy": false,
  "lifecycleDetails": "ProtectionPolicy was deleted successfully",
  "lifecycleState": "DELETED",
  "mustEnforceCloudLocality": false,
  "policyLockedDateTime": null,
  "schedules": null,
  "systemTags": {},
  "targetRegions": null,
  "timeCreated": "2026-09-02T17:53:57.039Z",
  "timeUpdated": "2026-09-02T17:55:08.762Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_PROTECTION_POLICY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "protectionPolicy",
      "entityUri": "/protectionPolicies/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T17:53:57.036Z",
  "timeFinished": "2026-09-02T17:54:16.867Z",
  "timeStarted": "2026-09-02T17:54:16.686Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_PROTECTION_POLICY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "protectionPolicy",
      "entityUri": "/protectionPolicies/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T17:54:19.558Z",
  "timeFinished": "2026-09-02T17:54:48.631Z",
  "timeStarted": "2026-09-02T17:54:48.385Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_PROTECTION_POLICY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "protectionPolicy",
      "entityUri": "/protectionPolicies/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T17:54:52.089Z",
  "timeFinished": "2026-09-02T17:55:08.751Z",
  "timeStarted": "2026-09-02T17:55:08.662Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[recoverysdk.ProtectionPolicy, recoverysdk.CreateProtectionPolicyDetails, recoverysdk.UpdateProtectionPolicyDetails]{
		CollectionPath: "/20210216/protectionPolicies", ItemPath: "/20210216/protectionPolicies/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState,
		ListShape:    ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ recoverysdk.CreateProtectionPolicyDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:2>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://recovery.us-ashburn-1.oci.oraclecloud.com", BasePath: "20210216", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := recoverysdk.DatabaseRecoveryClient{BaseClient: session.BaseClient()}
	client := newProtectionPolicyServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	mockValidateCreated := func(current *recoveryv1beta1.ProtectionPolicy) error {
		if current.Status.DisplayName != "osok-mock-protection-policy" || current.Status.BackupRetentionPeriodInDays != 14 {
			return fmt.Errorf("created ProtectionPolicy status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *recoveryv1beta1.ProtectionPolicy) error {
		if current.Status.DisplayName != "osok-mock-protection-policy-updated" || current.Status.BackupRetentionPeriodInDays != 15 {
			return fmt.Errorf("updated ProtectionPolicy status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*recoveryv1beta1.ProtectionPolicy]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *recoveryv1beta1.ProtectionPolicy) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *recoveryv1beta1.ProtectionPolicy) {
			current.Spec.DisplayName = "osok-mock-protection-policy-updated"
			current.Spec.BackupRetentionPeriodInDays = 15
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *recoveryv1beta1.ProtectionPolicy) error {
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

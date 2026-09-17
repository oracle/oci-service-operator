/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package securitypolicyconfig

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

const mockSecurityPolicyConfigName = "osok-mock-security-policy-config-v1"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationSecurityPolicyConfigWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &datasafev1beta1.SecurityPolicyConfig{}
	ocimock.InitializeResource(resource, "mock-securitypolicyconfig")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.SecurityPolicyConfigSpec](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-security-policy-config-v1",
  "firewallConfig": {
    "excludeJob": "INCLUDED",
    "status": "ENABLED",
    "violationLogAutoPurge": "DISABLED"
  },
  "freeformTags": {
    "osok-mock": "create"
  },
  "securityPolicyId": "<binding:security-policy-id>",
  "unifiedAuditPolicyConfig": {
    "excludeDatasafeUser": "ENABLED"
  }
}
`)
	moveSpec := resource.Spec
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateSecurityPolicyConfigDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-security-policy-config-v1",
  "firewallConfig": {
    "excludeJob": "INCLUDED",
    "status": "ENABLED",
    "violationLogAutoPurge": "DISABLED"
  },
  "freeformTags": {
    "osok-mock": "create"
  },
  "securityPolicyId": "<binding:security-policy-id>",
  "unifiedAuditPolicyConfig": {
    "excludeDatasafeUser": "ENABLED"
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateSecurityPolicyConfigDetails](t, `
{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.SecurityPolicyConfig](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:54:35.904Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-security-policy-config-v1",
  "firewallConfig": {
    "excludeJob": "INCLUDED",
    "status": "ENABLED",
    "violationLogAutoPurge": "DISABLED"
  },
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Security policy config is active",
  "lifecycleState": "ACTIVE",
  "securityPolicyId": "<binding:security-policy-id>",
  "systemTags": {},
  "timeCreated": "2026-09-03T17:54:36.226Z",
  "timeUpdated": "2026-09-03T17:54:42.018Z",
  "unifiedAuditPolicyConfig": {
    "excludeDatasafeUser": "ENABLED"
  }
}
`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.SecurityPolicyConfig](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:54:35.904Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-security-policy-config-v1",
  "firewallConfig": {
    "excludeJob": "INCLUDED",
    "status": "ENABLED",
    "violationLogAutoPurge": "DISABLED"
  },
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Security policy config is active",
  "lifecycleState": "ACTIVE",
  "securityPolicyId": "<binding:security-policy-id>",
  "systemTags": {},
  "timeCreated": "2026-09-03T17:54:36.226Z",
  "timeUpdated": "2026-09-03T17:54:51.913Z",
  "unifiedAuditPolicyConfig": {
    "excludeDatasafeUser": "ENABLED"
  }
}
`)
	deletedState := ocimock.MustOCIResponseFixture[datasafesdk.SecurityPolicyConfig](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:54:35.904Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-security-policy-config-v1",
  "firewallConfig": {
    "excludeJob": "INCLUDED",
    "status": "ENABLED",
    "violationLogAutoPurge": "DISABLED"
  },
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Security policy config is deleted",
  "lifecycleState": "DELETED",
  "securityPolicyId": "<binding:security-policy-id>",
  "systemTags": {},
  "timeCreated": "2026-09-03T17:54:36.226Z",
  "timeUpdated": "2026-09-03T17:55:01.231Z",
  "unifiedAuditPolicyConfig": {
    "excludeDatasafeUser": "ENABLED"
  }
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_SECURITY_POLICY_CONFIG",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "securityPolicyConfig",
      "entityUri": "/securityPolicyConfigs/<ocid:3>",
      "identifier": "<ocid:3>"
    },
    {
      "actionType": "CREATED",
      "entityType": "securityPolicy",
      "entityUri": "/securityPolicies/<binding:security-policy-id>",
      "identifier": "<binding:security-policy-id>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T17:54:36.459Z",
  "timeFinished": "2026-09-03T17:54:42.109Z",
  "timeStarted": "2026-09-03T17:54:41.078Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_SECURITY_POLICY_CONFIG",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "securityPolicyConfig",
      "entityUri": "/securityPolicyConfigs/<ocid:3>",
      "identifier": "<ocid:3>"
    },
    {
      "actionType": "UPDATED",
      "entityType": "securityPolicy",
      "entityUri": "/securityPolicies/<binding:security-policy-id>",
      "identifier": "<binding:security-policy-id>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T17:54:48.394Z",
  "timeFinished": "2026-09-03T17:54:52.021Z",
  "timeStarted": "2026-09-03T17:54:50.793Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_SECURITY_POLICY_CONFIG",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "securityPolicyConfig",
      "entityUri": "/securityPolicyConfigs/<ocid:3>",
      "identifier": "<ocid:3>"
    },
    {
      "actionType": "DELETED",
      "entityType": "securityPolicy",
      "entityUri": "/securityPolicies/<binding:security-policy-id>",
      "identifier": "<binding:security-policy-id>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T17:54:55.501Z",
  "timeFinished": "2026-09-03T17:55:01.777Z",
  "timeStarted": "2026-09-03T17:55:00.900Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.SecurityPolicyConfig, datasafesdk.CreateSecurityPolicyConfigDetails, datasafesdk.UpdateSecurityPolicyConfigDetails]{
		CollectionPath: "/20181201/securityPolicyConfigs", ItemPath: "/20181201/securityPolicyConfigs/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState,
		ListShape:    ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateSecurityPolicyConfigDetails) error {
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
	client := newSecurityPolicyConfigServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		sdkClient,
	)
	mockValidateCreated := func(current *datasafev1beta1.SecurityPolicyConfig) error {
		if current.Status.Id == "" || current.Status.DisplayName != mockSecurityPolicyConfigName || current.Status.SecurityPolicyId != resource.Spec.SecurityPolicyId {
			return fmt.Errorf("created SecurityPolicyConfig status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *datasafev1beta1.SecurityPolicyConfig) error {
		if current.Status.Description != "recorded update" || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated SecurityPolicyConfig status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.SecurityPolicyConfig]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.SecurityPolicyConfig) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.SecurityPolicyConfig) {
			current.Spec.Description = "recorded update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *datasafev1beta1.SecurityPolicyConfig) error {
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

	moveResource := &datasafev1beta1.SecurityPolicyConfig{Spec: moveSpec}
	ocimock.InitializeResource(moveResource, "mock-securitypolicyconfig-move")
	moveResource.Spec.CompartmentId = "<ocid:moved-compartment>"
	moveResource.Status.Id = "<ocid:3>"
	moveResource.Status.OsokStatus.Ocid = "<ocid:3>"
	moveDetails := datasafesdk.ChangeSecurityPolicyConfigCompartmentDetails{CompartmentId: &moveResource.Spec.CompartmentId}
	moveResponder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[datasafesdk.SecurityPolicyConfig]{
		CollectionPath: "/20181201/securityPolicyConfigs", ItemPath: "/20181201/securityPolicyConfigs/<ocid:3>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationRead}, InitialState: &createdState,
		Read: func(_ ocimock.Request, state datasafesdk.SecurityPolicyConfig) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		AdditionalRoutes: []ocimock.Route{{
			Name: "change-compartment", Method: http.MethodPost, Path: "/20181201/securityPolicyConfigs/<ocid:3>/actions/changeCompartment", MinimumCalls: 1,
			Respond: func(request ocimock.Request) (ocimock.Response, error) {
				if err := ocimock.ValidateJSONRequest(request, moveDetails); err != nil {
					return ocimock.Response{}, err
				}
				wantToken := securityPolicyConfigCompartmentMoveRetryToken(moveResource, moveResource.Spec.CompartmentId)
				if wantToken == nil || request.Header.Get("opc-retry-token") != *wantToken {
					return ocimock.Response{}, fmt.Errorf("change compartment retry token = %q, want %v", request.Header.Get("opc-retry-token"), wantToken)
				}
				return ocimock.Response{StatusCode: http.StatusAccepted, Header: http.Header{
					"Opc-Work-Request-Id": []string{"<ocid:move-work-request>"},
					"Opc-Request-Id":      []string{"mock-move-request"},
				}}, nil
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	moveSession, err := ocimock.Open(ocimock.Options{Host: "https://datasafe.us-ashburn-1.oci.oraclecloud.com", BasePath: "20181201", Responder: moveResponder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = moveSession.Close() })
	moveSDKClient := datasafesdk.DataSafeClient{BaseClient: moveSession.BaseClient()}
	moveClient := newSecurityPolicyConfigServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration-move")},
		moveSDKClient,
	)
	moveResponse, err := moveClient.CreateOrUpdate(context.Background(), moveResource, ctrl.Request{})
	if err != nil {
		t.Fatal(err)
	}
	currentMove := moveResource.Status.OsokStatus.Async.Current
	if !moveResponse.IsSuccessful || !moveResponse.ShouldRequeue || currentMove == nil ||
		currentMove.WorkRequestID != "<ocid:move-work-request>" || currentMove.RawOperationType != string(datasafesdk.WorkRequestOperationTypeChangeSecurityPolicyConfigCompartment) ||
		string(currentMove.Phase) != "update" || string(currentMove.NormalizedClass) != "pending" || moveResource.Status.OsokStatus.OpcRequestID != "mock-move-request" {
		t.Fatalf("SecurityPolicyConfig compartment move response=%+v async=%+v", moveResponse, currentMove)
	}
	if err := moveSession.Close(); err != nil {
		t.Fatal(err)
	}
}

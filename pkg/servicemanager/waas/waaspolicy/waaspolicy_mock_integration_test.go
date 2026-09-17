/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package waaspolicy

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// runtime, and package-owned typed OCI fixtures. Volatile WAAS work-request
// logs and service-defaulted WAF fields are intentionally omitted.
func TestMockIntegrationWaasPolicyWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &waasv1beta1.WaasPolicy{}
	ocimock.InitializeResource(resource, "mock-waaspolicy")
	resource.Spec = ocimock.MustJSONFixture[waasv1beta1.WaasPolicySpec](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-waas-policy-recorded-v3",
  "domain": "osok-mock-waas-recorded-v3.example.com",
  "freeformTags": {"osok-mock": "create"},
  "origins": {"primary": {"uri": "www.example.com"}},
  "wafConfig": {
    "addressRateLimiting": {"isEnabled": false},
    "deviceFingerprintChallenge": {"isEnabled": false},
    "humanInteractionChallenge": {"isEnabled": false},
    "jsChallenge": {"isEnabled": false},
    "origin": "primary"
  }
}`)
	moveSpec := resource.Spec
	createRequest := ocimock.MustJSONFixture[waassdk.CreateWaasPolicyDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-waas-policy-recorded-v3",
  "domain": "osok-mock-waas-recorded-v3.example.com",
  "freeformTags": {"osok-mock": "create"},
  "origins": {"primary": {"uri": "www.example.com"}},
  "wafConfig": {
    "addressRateLimiting": {"isEnabled": false},
    "deviceFingerprintChallenge": {"isEnabled": false},
    "humanInteractionChallenge": {"isEnabled": false},
    "jsChallenge": {"isEnabled": false},
    "origin": "primary"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[waassdk.UpdateWaasPolicyDetails](t, `{
  "freeformTags": {"osok-mock": "update"},
  "wafConfig": {
    "addressRateLimiting": {"isEnabled": false},
    "deviceFingerprintChallenge": {"isEnabled": false},
    "humanInteractionChallenge": {"isEnabled": false, "isNatEnabled": false},
    "jsChallenge": {"areRedirectsChallenged": false, "isEnabled": false, "isNatEnabled": false},
    "origin": "primary"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[waassdk.WaasPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-waas-policy-recorded-v3",
  "domain": "osok-mock-waas-recorded-v3.example.com",
  "freeformTags": {"osok-mock": "create"},
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "origins": {"primary": {"uri": "www.example.com"}}
}`)
	updatedState := ocimock.MustOCIResponseFixture[waassdk.WaasPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-waas-policy-recorded-v3",
  "domain": "osok-mock-waas-recorded-v3.example.com",
  "freeformTags": {"osok-mock": "update"},
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "origins": {"primary": {"uri": "www.example.com"}}
}`)
	deletedState := ocimock.MustOCIResponseFixture[waassdk.WaasPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-waas-policy-recorded-v3",
  "domain": "osok-mock-waas-recorded-v3.example.com",
  "freeformTags": {"osok-mock": "update"},
  "id": "<ocid:3>",
  "lifecycleState": "DELETED"
}`)
	createWorkRequest := mockWaasPolicyWorkRequest(t, "<ocid:2>", "CREATE_WAAS_POLICY", "CREATED")
	updateWorkRequest := mockWaasPolicyWorkRequest(t, "<ocid:4>", "UPDATE_WAAS_POLICY", "UPDATED")
	deleteWorkRequest := mockWaasPolicyWorkRequest(t, "<ocid:5>", "DELETE_WAAS_POLICY", "DELETED")

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[waassdk.WaasPolicy, waassdk.CreateWaasPolicyDetails, waassdk.UpdateWaasPolicyDetails]{
		CollectionPath: "/20181116/waasPolicies", ItemPath: "/20181116/waasPolicies/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ waassdk.CreateWaasPolicyDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			mockWaasPolicyWorkRequestRoute("create-work-request", "<ocid:2>", createWorkRequest),
			mockWaasPolicyWorkRequestRoute("update-work-request", "<ocid:4>", updateWorkRequest),
			mockWaasPolicyWorkRequestRoute("delete-work-request", "<ocid:5>", deleteWorkRequest),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://waas.us-ashburn-1.oci.oraclecloud.com", BasePath: "20181116", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newWaasPolicyServiceClientWithOCIClient(waassdk.WaasClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*waasv1beta1.WaasPolicy]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *waasv1beta1.WaasPolicy) error {
			if current.Status.Id == "" || current.Status.LifecycleState != string(waassdk.LifecycleStatesActive) ||
				current.Status.DisplayName != resource.Spec.DisplayName || current.Status.FreeformTags["osok-mock"] != "create" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created WaasPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *waasv1beta1.WaasPolicy) {
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *waasv1beta1.WaasPolicy) error {
			if current.Status.LifecycleState != string(waassdk.LifecycleStatesActive) || current.Status.DisplayName != resource.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated WaasPolicy status = %+v", current.Status)
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

	moveResource := &waasv1beta1.WaasPolicy{Spec: moveSpec}
	ocimock.InitializeResource(moveResource, "mock-waaspolicy-move")
	moveResource.Spec.CompartmentId = "<ocid:moved-compartment>"
	moveResource.Status.Id = "<ocid:3>"
	moveResource.Status.OsokStatus.Ocid = "<ocid:3>"
	moveDetails := waassdk.ChangeWaasPolicyCompartmentDetails{CompartmentId: &moveResource.Spec.CompartmentId}
	moveResponder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[waassdk.WaasPolicy]{
		CollectionPath: "/20181116/waasPolicies", ItemPath: "/20181116/waasPolicies/<ocid:3>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationRead}, InitialState: &createdState,
		Read: func(_ ocimock.Request, state waassdk.WaasPolicy) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		AdditionalRoutes: []ocimock.Route{{
			Name: "change-compartment", Method: http.MethodPost, Path: "/20181116/waasPolicies/<ocid:3>/actions/changeCompartment", MinimumCalls: 1,
			Respond: func(request ocimock.Request) (ocimock.Response, error) {
				if err := ocimock.ValidateJSONRequest(request, moveDetails); err != nil {
					return ocimock.Response{}, err
				}
				wantToken := waasPolicyCompartmentMoveRetryToken(moveResource, moveResource.Spec.CompartmentId)
				if request.Header.Get("opc-retry-token") != wantToken {
					return ocimock.Response{}, fmt.Errorf("change compartment retry token = %q, want %q", request.Header.Get("opc-retry-token"), wantToken)
				}
				return ocimock.Response{StatusCode: http.StatusAccepted, Header: http.Header{
					"Opc-Request-Id": []string{"mock-move-request"},
				}}, nil
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	moveSession, err := ocimock.Open(ocimock.Options{Host: "https://waas.us-ashburn-1.oci.oraclecloud.com", BasePath: "20181116", Responder: moveResponder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = moveSession.Close() })
	moveClient := newWaasPolicyServiceClientWithOCIClient(waassdk.WaasClient{BaseClient: moveSession.BaseClient()})
	moveResponse, err := moveClient.CreateOrUpdate(context.Background(), moveResource, ctrl.Request{})
	if err != nil {
		t.Fatal(err)
	}
	currentMove := moveResource.Status.OsokStatus.Async.Current
	if !moveResponse.IsSuccessful || !moveResponse.ShouldRequeue || currentMove == nil ||
		currentMove.WorkRequestID != "" || currentMove.RawOperationType != "CHANGE_WAAS_POLICY_COMPARTMENT" || string(currentMove.Source) != "lifecycle" ||
		string(currentMove.Phase) != "update" || string(currentMove.NormalizedClass) != "pending" || moveResource.Status.OsokStatus.OpcRequestID != "mock-move-request" {
		t.Fatalf("WaasPolicy compartment move response=%+v async=%+v", moveResponse, currentMove)
	}
	if err := moveSession.Close(); err != nil {
		t.Fatal(err)
	}
}

func mockWaasPolicyWorkRequest(t *testing.T, id, operationType, actionType string) waassdk.WorkRequest {
	t.Helper()
	return ocimock.MustOCIResponseFixture[waassdk.WorkRequest](t, fmt.Sprintf(`{
  "compartmentId": "<ocid:1>",
  "errors": [],
  "id": %q,
  "logs": [],
  "operationType": %q,
  "percentComplete": 100,
  "resources": [{"actionType": %q, "entityType": "waas", "entityUri": "/20181116/waasPolicies/<ocid:3>", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}`, id, operationType, actionType))
}

func mockWaasPolicyWorkRequestRoute(name, id string, workRequest waassdk.WorkRequest) ocimock.Route {
	return ocimock.Route{
		Name: name, Method: http.MethodGet, Path: "/20181116/workRequests/" + id, MinimumCalls: 1,
		Respond: func(ocimock.Request) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, workRequest)
		},
	}
}

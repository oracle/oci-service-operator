/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package webappfirewallpolicy

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	wafsdk "github.com/oracle/oci-go-sdk/v65/waf"
	wafv1beta1 "github.com/oracle/oci-service-operator/api/waf/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockWebAppFirewallPolicyName = "osok-mock-waf-policy-v1"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationWebAppFirewallPolicyWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &wafv1beta1.WebAppFirewallPolicy{}
	ocimock.InitializeResource(resource, "mock-webappfirewallpolicy")
	resource.Spec = ocimock.MustJSONFixture[wafv1beta1.WebAppFirewallPolicySpec](t, `
{
  "actions": [],
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-waf-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}
`)
	moveSpec := resource.Spec
	createRequest := ocimock.MustJSONFixture[wafsdk.CreateWebAppFirewallPolicyDetails](t, `
{
  "actions": [],
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-waf-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[wafsdk.UpdateWebAppFirewallPolicyDetails](t, `
{
  "actions": [],
  "displayName": "osok-mock-waf-policy-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[wafsdk.WebAppFirewallPolicy](t, `
{
  "actions": [],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:08:06.453Z"
    }
  },
  "displayName": "osok-mock-waf-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "requestAccessControl": null,
  "requestProtection": null,
  "requestRateLimiting": null,
  "responseAccessControl": null,
  "responseProtection": null,
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T23:08:06.555Z",
  "timeUpdated": "2026-09-01T23:08:33.878Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[wafsdk.WebAppFirewallPolicy](t, `
{
  "actions": [],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:08:06.453Z"
    }
  },
  "displayName": "osok-mock-waf-policy-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "requestAccessControl": null,
  "requestProtection": null,
  "requestRateLimiting": null,
  "responseAccessControl": null,
  "responseProtection": null,
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T23:08:06.555Z",
  "timeUpdated": "2026-09-01T23:08:36.754Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[wafsdk.WebAppFirewallPolicy](t, `
{
  "actions": [],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:08:06.453Z"
    }
  },
  "displayName": "osok-mock-waf-policy-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "requestAccessControl": null,
  "requestProtection": null,
  "requestRateLimiting": null,
  "responseAccessControl": null,
  "responseProtection": null,
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T23:08:06.555Z",
  "timeUpdated": "2026-09-01T23:09:12.284Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[wafsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:3>",
  "operationType": "CREATE_WAF_POLICY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "webAppFirewallPolicy",
      "entityUri": "/webAppFirewallPolicies/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T23:08:06.579Z",
  "timeFinished": "2026-09-01T23:08:33.896Z",
  "timeStarted": "2026-09-01T23:08:33.729Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[wafsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_WAF_POLICY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "webAppFirewallPolicy",
      "entityUri": "/webAppFirewallPolicies/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T23:08:36.761Z",
  "timeFinished": "2026-09-01T23:08:36.777Z",
  "timeStarted": "2026-09-01T23:08:36.777Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[wafsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_WAF_POLICY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "webAppFirewallPolicy",
      "entityUri": "/webAppFirewallPolicies/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T23:08:38.845Z",
  "timeFinished": "2026-09-01T23:09:12.318Z",
  "timeStarted": "2026-09-01T23:09:12.013Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[wafsdk.WebAppFirewallPolicy, wafsdk.CreateWebAppFirewallPolicyDetails, wafsdk.UpdateWebAppFirewallPolicyDetails]{
		CollectionPath: "/20210930/webAppFirewallPolicies", ItemPath: "/20210930/webAppFirewallPolicies/<ocid:2>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState,
		ListShape:    ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:3>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ wafsdk.CreateWebAppFirewallPolicyDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20210930/workRequests/<ocid:3>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20210930/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20210930/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://waf.us-ashburn-1.oci.oraclecloud.com", BasePath: "20210930", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := wafsdk.WafClient{BaseClient: session.BaseClient()}
	client := newWebAppFirewallPolicyServiceClientWithOCIClient(sdkClient)
	mockValidateCreated := func(current *wafv1beta1.WebAppFirewallPolicy) error {
		if current.Status.DisplayName != mockWebAppFirewallPolicyName || current.Status.Id == "" {
			return fmt.Errorf("created WebAppFirewallPolicy status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *wafv1beta1.WebAppFirewallPolicy) error {
		if current.Status.DisplayName != current.Spec.DisplayName || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated WebAppFirewallPolicy status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*wafv1beta1.WebAppFirewallPolicy]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *wafv1beta1.WebAppFirewallPolicy) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *wafv1beta1.WebAppFirewallPolicy) {
			current.Spec.DisplayName = mockWebAppFirewallPolicyName + "-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *wafv1beta1.WebAppFirewallPolicy) error {
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

	moveResource := &wafv1beta1.WebAppFirewallPolicy{Spec: moveSpec}
	ocimock.InitializeResource(moveResource, "mock-webappfirewallpolicy-move")
	moveResource.Spec.CompartmentId = "<ocid:moved-compartment>"
	moveResource.Status.Id = "<ocid:2>"
	moveResource.Status.OsokStatus.Ocid = "<ocid:2>"
	moveDetails := wafsdk.ChangeWebAppFirewallPolicyCompartmentDetails{CompartmentId: &moveResource.Spec.CompartmentId}
	moveResponder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[wafsdk.WebAppFirewallPolicy]{
		CollectionPath: "/20210930/webAppFirewallPolicies", ItemPath: "/20210930/webAppFirewallPolicies/<ocid:2>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationRead}, InitialState: &createdState,
		Read: func(_ ocimock.Request, state wafsdk.WebAppFirewallPolicy) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		AdditionalRoutes: []ocimock.Route{{
			Name: "change-compartment", Method: http.MethodPost, Path: "/20210930/webAppFirewallPolicies/<ocid:2>/actions/changeCompartment", MinimumCalls: 1,
			Respond: func(request ocimock.Request) (ocimock.Response, error) {
				if err := ocimock.ValidateJSONRequest(request, moveDetails); err != nil {
					return ocimock.Response{}, err
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
	moveSession, err := ocimock.Open(ocimock.Options{Host: "https://waf.us-ashburn-1.oci.oraclecloud.com", BasePath: "20210930", Responder: moveResponder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = moveSession.Close() })
	moveClient := newWebAppFirewallPolicyServiceClientWithOCIClient(wafsdk.WafClient{BaseClient: moveSession.BaseClient()})
	moveResponse, err := moveClient.CreateOrUpdate(context.Background(), moveResource, ctrl.Request{})
	if err != nil {
		t.Fatal(err)
	}
	currentMove := moveResource.Status.OsokStatus.Async.Current
	if !moveResponse.IsSuccessful || !moveResponse.ShouldRequeue || currentMove == nil ||
		currentMove.WorkRequestID != "<ocid:move-work-request>" || currentMove.RawOperationType != string(wafsdk.WorkRequestOperationTypeMoveWafPolicy) ||
		string(currentMove.Phase) != "update" || string(currentMove.NormalizedClass) != "pending" || moveResource.Status.OsokStatus.OpcRequestID != "mock-move-request" {
		t.Fatalf("WebAppFirewallPolicy compartment move response=%+v async=%+v", moveResponse, currentMove)
	}
	if err := moveSession.Close(); err != nil {
		t.Fatal(err)
	}
}

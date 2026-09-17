/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package attributeset

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

const mockAttributeSetName = "osok-mock-attribute-set-v2"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationAttributeSetWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &datasafev1beta1.AttributeSet{}
	ocimock.InitializeResource(resource, "mock-attributeset")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.AttributeSetSpec](t, `
{
  "attributeSetType": "DATABASE_USER",
  "attributeSetValues": [
    "OSOK_REPLAY_USER"
  ],
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-attribute-set-v2",
  "freeformTags": {
    "osok-mock": "create"
  }
}
`)
	moveSpec := resource.Spec
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateAttributeSetDetails](t, `
{
  "attributeSetType": "DATABASE_USER",
  "attributeSetValues": [
    "OSOK_REPLAY_USER"
  ],
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-attribute-set-v2",
  "freeformTags": {
    "osok-mock": "create"
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateAttributeSetDetails](t, `
{
  "attributeSetValues": [
    "OSOK_REPLAY_USER",
    "OSOK_REPLAY_USER_2"
  ],
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.AttributeSet](t, `
{
  "attributeSetType": "DATABASE_USER",
  "attributeSetValues": [
    "OSOK_REPLAY_USER"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:12:43.070Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-attribute-set-v2",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "inUse": "NO",
  "isUserDefined": true,
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-09-03T17:12:43.131Z",
  "timeUpdated": "2026-09-03T17:12:46.667Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.AttributeSet](t, `
{
  "attributeSetType": "DATABASE_USER",
  "attributeSetValues": [
    "OSOK_REPLAY_USER",
    "OSOK_REPLAY_USER_2"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T17:12:43.070Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-attribute-set-v2",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "inUse": "NO",
  "isUserDefined": true,
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-09-03T17:12:43.131Z",
  "timeUpdated": "2026-09-03T17:12:52.617Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_ATTRIBUTE_SET",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "/attributeSets",
      "entityUri": "/attributeSets/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T17:12:43.149Z",
  "timeFinished": "2026-09-03T17:12:46.712Z",
  "timeStarted": "2026-09-03T17:12:45.731Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_ATTRIBUTE_SET",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "/attributeSets",
      "entityUri": "/attributeSets/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T17:12:50.165Z",
  "timeFinished": "2026-09-03T17:12:52.665Z",
  "timeStarted": "2026-09-03T17:12:52.530Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_ATTRIBUTE_SET",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "/attributeSets",
      "entityUri": "/attributeSets/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T17:12:56.670Z",
  "timeFinished": "2026-09-03T17:13:03.033Z",
  "timeStarted": "2026-09-03T17:13:02.491Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.AttributeSet, datasafesdk.CreateAttributeSetDetails, datasafesdk.UpdateAttributeSetDetails]{
		CollectionPath: "/20181201/attributeSets", ItemPath: "/20181201/attributeSets/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateAttributeSetDetails) error {
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
	manager := &AttributeSetServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAttributeSetDefaultRuntimeHooks(sdkClient)
	applyAttributeSetRuntimeHooks(manager, &hooks, sdkClient, nil)
	client := wrapAttributeSetGeneratedClient(hooks, defaultAttributeSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.AttributeSet](buildAttributeSetGeneratedRuntimeConfig(manager, hooks)),
	})
	mockValidateCreated := func(current *datasafev1beta1.AttributeSet) error {
		if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != mockAttributeSetName || current.Status.AttributeSetType != "DATABASE_USER" {
			return fmt.Errorf("created AttributeSet status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *datasafev1beta1.AttributeSet) error {
		if current.Status.Description != "recorded update" || len(current.Status.AttributeSetValues) != 2 || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated AttributeSet status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.AttributeSet]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.AttributeSet) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.AttributeSet) {
			current.Spec.AttributeSetValues = []string{"OSOK_REPLAY_USER", "OSOK_REPLAY_USER_2"}
			current.Spec.Description = "recorded update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *datasafev1beta1.AttributeSet) error {
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

	moveResource := &datasafev1beta1.AttributeSet{Spec: moveSpec}
	ocimock.InitializeResource(moveResource, "mock-attributeset-move")
	moveResource.Spec.CompartmentId = "<ocid:moved-compartment>"
	moveResource.Status.Id = "<ocid:3>"
	moveResource.Status.OsokStatus.Ocid = "<ocid:3>"
	moveDetails := datasafesdk.ChangeAttributeSetCompartmentDetails{CompartmentId: &moveResource.Spec.CompartmentId}
	moveResponder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[datasafesdk.AttributeSet]{
		CollectionPath: "/20181201/attributeSets", ItemPath: "/20181201/attributeSets/<ocid:3>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationRead}, InitialState: &createdState,
		Read: func(_ ocimock.Request, state datasafesdk.AttributeSet) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		AdditionalRoutes: []ocimock.Route{{
			Name: "change-compartment", Method: http.MethodPost, Path: "/20181201/attributeSets/<ocid:3>/actions/changeCompartment", MinimumCalls: 1,
			Respond: func(request ocimock.Request) (ocimock.Response, error) {
				if err := ocimock.ValidateJSONRequest(request, moveDetails); err != nil {
					return ocimock.Response{}, err
				}
				wantToken := attributeSetCompartmentMoveRetryToken(moveResource, moveResource.Spec.CompartmentId)
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
	moveManager := &AttributeSetServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration-move")}}
	moveHooks := newAttributeSetDefaultRuntimeHooks(moveSDKClient)
	applyAttributeSetRuntimeHooks(moveManager, &moveHooks, moveSDKClient, nil)
	moveClient := wrapAttributeSetGeneratedClient(moveHooks, defaultAttributeSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.AttributeSet](buildAttributeSetGeneratedRuntimeConfig(moveManager, moveHooks)),
	})
	moveResponse, err := moveClient.CreateOrUpdate(context.Background(), moveResource, ctrl.Request{})
	if err != nil {
		t.Fatal(err)
	}
	currentMove := moveResource.Status.OsokStatus.Async.Current
	if !moveResponse.IsSuccessful || !moveResponse.ShouldRequeue || currentMove == nil ||
		currentMove.WorkRequestID != "<ocid:move-work-request>" || currentMove.RawOperationType != string(datasafesdk.WorkRequestOperationTypeChangeAttributeSetCompartment) ||
		string(currentMove.Phase) != "update" || string(currentMove.NormalizedClass) != "pending" || moveResource.Status.OsokStatus.OpcRequestID != "mock-move-request" {
		t.Fatalf("AttributeSet compartment move response=%+v async=%+v", moveResponse, currentMove)
	}
	if err := moveSession.Close(); err != nil {
		t.Fatal(err)
	}
}

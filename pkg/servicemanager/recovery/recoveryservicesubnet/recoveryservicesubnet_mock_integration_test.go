/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package recoveryservicesubnet

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

const mockRecoveryServiceSubnetName = "osok-mock-recovery-subnet"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationRecoveryServiceSubnetWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &recoveryv1beta1.RecoveryServiceSubnet{}
	ocimock.InitializeResource(resource, "mock-recoveryservicesubnet")
	resource.Spec = ocimock.MustJSONFixture[recoveryv1beta1.RecoveryServiceSubnetSpec](t, `
{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-recovery-subnet",
  "freeformTags": {
    "osok-mock": "create"
  },
  "subnets": [
    "<ocid:2>"
  ],
  "vcnId": "<ocid:3>"
}
`)
	moveSpec := resource.Spec
	createRequest := ocimock.MustJSONFixture[recoverysdk.CreateRecoveryServiceSubnetDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-recovery-subnet",
  "freeformTags": {
    "osok-mock": "create"
  },
  "subnets": [
    "<ocid:2>"
  ],
  "vcnId": "<ocid:3>"
}
`)
	updateRequest := ocimock.MustJSONFixture[recoverysdk.UpdateRecoveryServiceSubnetDetails](t, `
{
  "displayName": "osok-mock-recovery-subnet-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[recoverysdk.RecoveryServiceSubnet](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T21:05:11.142Z"
    }
  },
  "displayName": "osok-mock-recovery-subnet",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:5>",
  "isAutoCreated": false,
  "lifecycleDetails": "RecoveryServiceSubnet was created successfully",
  "lifecycleState": "ACTIVE",
  "nsgIds": null,
  "securityAttributes": null,
  "subnetId": "<ocid:2>",
  "subnets": [
    "<ocid:2>"
  ],
  "systemTags": {},
  "timeCreated": "2026-09-03T21:05:12.316Z",
  "timeUpdated": "2026-09-03T21:05:26.427Z",
  "vcnId": "<ocid:3>"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[recoverysdk.RecoveryServiceSubnet](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T21:05:11.142Z"
    }
  },
  "displayName": "osok-mock-recovery-subnet-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:5>",
  "isAutoCreated": false,
  "lifecycleDetails": "RecoveryServiceSubnet was updated successfully",
  "lifecycleState": "ACTIVE",
  "nsgIds": null,
  "securityAttributes": null,
  "subnetId": "<ocid:2>",
  "subnets": [
    "<ocid:2>"
  ],
  "systemTags": {},
  "timeCreated": "2026-09-03T21:05:12.316Z",
  "timeUpdated": "2026-09-03T21:05:34.267Z",
  "vcnId": "<ocid:3>"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[recoverysdk.RecoveryServiceSubnet](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T21:05:11.142Z"
    }
  },
  "displayName": "osok-mock-recovery-subnet-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:5>",
  "isAutoCreated": false,
  "lifecycleDetails": "RecoveryServiceSubnet was deleted successfully",
  "lifecycleState": "DELETED",
  "nsgIds": null,
  "securityAttributes": null,
  "subnetId": "<ocid:2>",
  "subnets": [
    "<ocid:2>"
  ],
  "systemTags": {},
  "timeCreated": "2026-09-03T21:05:12.316Z",
  "timeUpdated": "2026-09-03T21:06:22.686Z",
  "vcnId": "<ocid:3>"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "CREATE_RECOVERY_SERVICE_SUBNET",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "recoveryServiceSubnet",
      "entityUri": "/recoveryServiceSubnets/<ocid:5>",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T21:05:12.299Z",
  "timeFinished": "2026-09-03T21:05:26.420Z",
  "timeStarted": "2026-09-03T21:05:25.441Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:6>",
  "operationType": "UPDATE_RECOVERY_SERVICE_SUBNET",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "recoveryServiceSubnet",
      "entityUri": "/recoveryServiceSubnets/<ocid:5>",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T21:05:34.271Z",
  "timeFinished": "2026-09-03T21:05:34.290Z",
  "timeStarted": "2026-09-03T21:05:34.290Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[recoverysdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:7>",
  "operationType": "DELETE_RECOVERY_SERVICE_SUBNET",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "recoveryServiceSubnet",
      "entityUri": "/recoveryServiceSubnets/<ocid:5>",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T21:05:35.653Z",
  "timeFinished": "2026-09-03T21:06:22.670Z",
  "timeStarted": "2026-09-03T21:06:22.575Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[recoverysdk.RecoveryServiceSubnet, recoverysdk.CreateRecoveryServiceSubnetDetails, recoverysdk.UpdateRecoveryServiceSubnetDetails]{
		CollectionPath: "/20210216/recoveryServiceSubnets", ItemPath: "/20210216/recoveryServiceSubnets/<ocid:5>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState,
		ListShape:    ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:7>"}},
		ValidateCreate: func(request ocimock.Request, _ recoverysdk.CreateRecoveryServiceSubnetDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:6>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20210216/workRequests/<ocid:7>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
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
	client := newRecoveryServiceSubnetServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	mockValidateCreated := func(current *recoveryv1beta1.RecoveryServiceSubnet) error {
		if current.Status.Id == "" || current.Status.DisplayName != mockRecoveryServiceSubnetName {
			return fmt.Errorf("created RecoveryServiceSubnet status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *recoveryv1beta1.RecoveryServiceSubnet) error {
		if current.Status.DisplayName != current.Spec.DisplayName || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated RecoveryServiceSubnet status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*recoveryv1beta1.RecoveryServiceSubnet]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *recoveryv1beta1.RecoveryServiceSubnet) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *recoveryv1beta1.RecoveryServiceSubnet) {
			current.Spec.DisplayName = mockRecoveryServiceSubnetName + "-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *recoveryv1beta1.RecoveryServiceSubnet) error {
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

	moveResource := &recoveryv1beta1.RecoveryServiceSubnet{Spec: moveSpec}
	ocimock.InitializeResource(moveResource, "mock-recoveryservicesubnet-move")
	moveResource.Spec.CompartmentId = "<ocid:moved-compartment>"
	moveResource.Status.Id = "<ocid:5>"
	moveResource.Status.OsokStatus.Ocid = "<ocid:5>"
	moveDetails := recoverysdk.ChangeRecoveryServiceSubnetCompartmentDetails{CompartmentId: &moveResource.Spec.CompartmentId}
	moveResponder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[recoverysdk.RecoveryServiceSubnet]{
		CollectionPath: "/20210216/recoveryServiceSubnets", ItemPath: "/20210216/recoveryServiceSubnets/<ocid:5>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationRead}, InitialState: &createdState,
		Read: func(_ ocimock.Request, state recoverysdk.RecoveryServiceSubnet) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		AdditionalRoutes: []ocimock.Route{{
			Name: "change-compartment", Method: http.MethodPost, Path: "/20210216/recoveryServiceSubnets/<ocid:5>/actions/changeCompartment", MinimumCalls: 1,
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
	moveSession, err := ocimock.Open(ocimock.Options{Host: "https://recovery.us-ashburn-1.oci.oraclecloud.com", BasePath: "20210216", Responder: moveResponder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = moveSession.Close() })
	moveSDKClient := recoverysdk.DatabaseRecoveryClient{BaseClient: moveSession.BaseClient()}
	moveClient := newRecoveryServiceSubnetServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration-move")}, moveSDKClient)
	moveResponse, err := moveClient.CreateOrUpdate(context.Background(), moveResource, ctrl.Request{})
	if err != nil {
		t.Fatal(err)
	}
	currentMove := moveResource.Status.OsokStatus.Async.Current
	if !moveResponse.IsSuccessful || !moveResponse.ShouldRequeue || currentMove == nil ||
		currentMove.WorkRequestID != "<ocid:move-work-request>" || currentMove.RawOperationType != string(recoverysdk.OperationTypeMoveRecoveryServiceSubnet) ||
		string(currentMove.Phase) != "update" || string(currentMove.NormalizedClass) != "pending" || moveResource.Status.OsokStatus.OpcRequestID != "mock-move-request" {
		t.Fatalf("RecoveryServiceSubnet compartment move response=%+v async=%+v", moveResponse, currentMove)
	}
	if err := moveSession.Close(); err != nil {
		t.Fatal(err)
	}
}

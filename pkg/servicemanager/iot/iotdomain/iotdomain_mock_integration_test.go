/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package iotdomain

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	iotsdk "github.com/oracle/oci-go-sdk/v65/iot"
	iotv1beta1 "github.com/oracle/oci-service-operator/api/iot/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationIotDomainWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &iotv1beta1.IotDomain{}
	ocimock.InitializeResource(resource, "mock-iotdomain")
	resource.Spec = ocimock.MustJSONFixture[iotv1beta1.IotDomainSpec](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "OSOK recorded IoT domain",
  "displayName": "osok-mock-iot-domain",
  "freeformTags": {
    "osok-mock": "create"
  },
  "iotDomainGroupId": "<ocid:2>"
}
`)
	createRequest := ocimock.MustJSONFixture[iotsdk.CreateIotDomainDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "OSOK recorded IoT domain",
  "displayName": "osok-mock-iot-domain",
  "freeformTags": {
    "osok-mock": "create"
  },
  "iotDomainGroupId": "<ocid:2>"
}
`)
	updateRequest := ocimock.MustJSONFixture[iotsdk.UpdateIotDomainDetails](t, `
{
  "description": "OSOK recorded IoT domain updated",
  "displayName": "osok-mock-iot-domain",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[iotsdk.IotDomain](t, `
{
  "compartmentId": "<ocid:1>",
  "dataRetentionPeriodsInDays": {
    "historizedData": 30,
    "rawCommandData": 16,
    "rawData": 16,
    "rejectedData": 16
  },
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:18:34.350Z"
    }
  },
  "description": "OSOK recorded IoT domain",
  "deviceHost": "o4wwpd7oojrxo.device.iot.us-ashburn-1.oci.oraclecloud.com",
  "displayName": "osok-mock-iot-domain",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "iotDomainGroupId": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:18:34.504Z",
  "timeUpdated": "2026-09-02T20:20:58.934Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[iotsdk.IotDomain](t, `
{
  "compartmentId": "<ocid:1>",
  "dataRetentionPeriodsInDays": {
    "historizedData": 30,
    "rawCommandData": 16,
    "rawData": 16,
    "rejectedData": 16
  },
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:18:34.350Z"
    }
  },
  "description": "OSOK recorded IoT domain updated",
  "deviceHost": "o4wwpd7oojrxo.device.iot.us-ashburn-1.oci.oraclecloud.com",
  "displayName": "osok-mock-iot-domain",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "iotDomainGroupId": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:18:34.504Z",
  "timeUpdated": "2026-09-02T20:21:01.276Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[iotsdk.IotDomain](t, `
{
  "compartmentId": "<ocid:1>",
  "dataRetentionPeriodsInDays": {
    "historizedData": 30,
    "rawCommandData": 16,
    "rawData": 16,
    "rejectedData": 16
  },
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:18:34.350Z"
    }
  },
  "description": "OSOK recorded IoT domain updated",
  "deviceHost": "o4wwpd7oojrxo.device.iot.us-ashburn-1.oci.oraclecloud.com",
  "displayName": "osok-mock-iot-domain",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "iotDomainGroupId": "<ocid:2>",
  "lifecycleState": "DELETED",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:18:34.504Z",
  "timeUpdated": "2026-09-02T20:22:07.856Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[iotsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "CREATE_IOT_DOMAIN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "iotDomain",
      "entityUri": "/20250531/iotDomains/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T20:18:34.518Z",
  "timeFinished": "2026-09-02T20:20:58.922Z",
  "timeStarted": "2026-09-02T20:18:48.552Z",
  "timeUpdated": "2026-09-02T20:20:58.922Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[iotsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "UPDATE_IOT_DOMAIN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "iotDomain",
      "entityUri": "/20250531/iotDomains/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T20:21:01.254Z",
  "timeFinished": "2026-09-02T20:21:01.282Z",
  "timeStarted": "2026-09-02T20:21:01.282Z",
  "timeUpdated": "2026-09-02T20:21:01.282Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[iotsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:6>",
  "operationType": "DELETE_IOT_DOMAIN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "iotDomain",
      "entityUri": "/20250531/iotDomains/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T20:21:05.155Z",
  "timeFinished": "2026-09-02T20:22:07.837Z",
  "timeStarted": "2026-09-02T20:21:57.734Z",
  "timeUpdated": "2026-09-02T20:22:07.837Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[iotsdk.IotDomain, iotsdk.CreateIotDomainDetails, iotsdk.UpdateIotDomainDetails]{
		CollectionPath: "/20250531/iotDomains", ItemPath: "/20250531/iotDomains/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState,
		ListShape:    ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ iotsdk.CreateIotDomainDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20250531/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20250531/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20250531/workRequests/<ocid:6>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iot.us-ashburn-1.oci.oraclecloud.com", BasePath: "20250531", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := iotsdk.IotClient{BaseClient: session.BaseClient()}
	client := newIotDomainServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		sdkClient,
	)
	mockValidateCreated := func(current *iotv1beta1.IotDomain) error {
		if current.Status.DisplayName != "osok-mock-iot-domain" || current.Status.LifecycleState != string(iotsdk.IotDomainLifecycleStateActive) {
			return fmt.Errorf("created IotDomain status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *iotv1beta1.IotDomain) error {
		if current.Status.Description != "OSOK recorded IoT domain updated" || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated IotDomain status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*iotv1beta1.IotDomain]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *iotv1beta1.IotDomain) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *iotv1beta1.IotDomain) {
			current.Spec.Description = "OSOK recorded IoT domain updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *iotv1beta1.IotDomain) error {
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

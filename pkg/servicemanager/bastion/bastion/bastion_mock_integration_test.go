/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package bastion

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	bastionsdk "github.com/oracle/oci-go-sdk/v65/bastion"
	bastionv1beta1 "github.com/oracle/oci-service-operator/api/bastion/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockBastionName = "osok-mock-common-bastion-v1"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationBastionWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &bastionv1beta1.Bastion{}
	ocimock.InitializeResource(resource, "mock-bastion")
	resource.Spec = ocimock.MustJSONFixture[bastionv1beta1.BastionSpec](t, `
{
  "bastionType": "STANDARD",
  "clientCidrBlockAllowList": [
    "0.0.0.0/0"
  ],
  "compartmentId": "<ocid:1>",
  "dnsProxyStatus": "DISABLED",
  "freeformTags": {
    "osok-mock": "create"
  },
  "maxSessionTtlInSeconds": 1800,
  "name": "osok-mock-common-bastion-v1",
  "targetSubnetId": "<binding:subnet>"
}
`)
	createRequest := ocimock.MustJSONFixture[bastionsdk.CreateBastionDetails](t, `
{
  "bastionType": "STANDARD",
  "clientCidrBlockAllowList": [
    "0.0.0.0/0"
  ],
  "compartmentId": "<ocid:1>",
  "dnsProxyStatus": "DISABLED",
  "freeformTags": {
    "osok-mock": "create"
  },
  "maxSessionTtlInSeconds": 1800,
  "name": "osok-mock-common-bastion-v1",
  "targetSubnetId": "<binding:subnet>"
}
`)
	updateRequest := ocimock.MustJSONFixture[bastionsdk.UpdateBastionDetails](t, `
{
  "clientCidrBlockAllowList": [
    "10.0.0.0/8"
  ],
  "freeformTags": {
    "osok-mock": "update"
  },
  "maxSessionTtlInSeconds": 3600
}
`)
	createdState := ocimock.MustOCIResponseFixture[bastionsdk.Bastion](t, `
{
  "bastionRecordingConfig": null,
  "bastionRestrictionId": null,
  "bastionType": "STANDARD",
  "clientCidrBlockAllowList": [
    "0.0.0.0/0"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T05:45:13.084Z"
    }
  },
  "dnsProxyStatus": "DISABLED",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "isRecordingEnabled": false,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "maxSessionTtlInSeconds": 1800,
  "maxSessionsAllowed": 20,
  "name": "osok-mock-common-bastion-v1",
  "phoneBookEntry": null,
  "privateEndpointIpAddress": "10.0.10.160",
  "securityAttributes": {},
  "staticJumpHostIpAddresses": null,
  "systemTags": {},
  "targetSubnetId": "<binding:subnet>",
  "targetVcnId": "<ocid:4>",
  "timeCreated": "2026-09-01T05:45:14.223Z",
  "timeUpdated": "2026-09-01T05:46:05.662Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[bastionsdk.Bastion](t, `
{
  "bastionRecordingConfig": null,
  "bastionRestrictionId": null,
  "bastionType": "STANDARD",
  "clientCidrBlockAllowList": [
    "10.0.0.0/8"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T05:45:13.084Z"
    }
  },
  "dnsProxyStatus": "DISABLED",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "isRecordingEnabled": false,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "maxSessionTtlInSeconds": 3600,
  "maxSessionsAllowed": 20,
  "name": "osok-mock-common-bastion-v1",
  "phoneBookEntry": null,
  "privateEndpointIpAddress": "10.0.10.160",
  "securityAttributes": {},
  "staticJumpHostIpAddresses": null,
  "systemTags": {},
  "targetSubnetId": "<binding:subnet>",
  "targetVcnId": "<ocid:4>",
  "timeCreated": "2026-09-01T05:45:14.223Z",
  "timeUpdated": "2026-09-01T05:46:50.291Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[bastionsdk.Bastion](t, `
{
  "bastionRecordingConfig": null,
  "bastionRestrictionId": null,
  "bastionType": "STANDARD",
  "clientCidrBlockAllowList": [
    "10.0.0.0/8"
  ],
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T05:45:13.084Z"
    }
  },
  "dnsProxyStatus": "DISABLED",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "isRecordingEnabled": false,
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "maxSessionTtlInSeconds": 3600,
  "maxSessionsAllowed": 20,
  "name": "osok-mock-common-bastion-v1",
  "phoneBookEntry": null,
  "privateEndpointIpAddress": "10.0.10.160",
  "securityAttributes": {},
  "staticJumpHostIpAddresses": null,
  "systemTags": {},
  "targetSubnetId": "<binding:subnet>",
  "targetVcnId": "<ocid:4>",
  "timeCreated": "2026-09-01T05:45:14.223Z",
  "timeUpdated": "2026-09-01T05:47:43.587Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[bastionsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:3>",
  "operationType": "CREATE_BASTION",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "BastionsResource",
      "entityUri": "/bastions/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T05:45:14.223Z",
  "timeFinished": "2026-09-01T05:46:05.662Z",
  "timeStarted": "2026-09-01T05:45:25.870Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[bastionsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "UPDATE_BASTION",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "BastionsResource",
      "entityUri": "/bastions/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T05:46:16.082Z",
  "timeFinished": "2026-09-01T05:46:50.291Z",
  "timeStarted": "2026-09-01T05:46:27.680Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[bastionsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:6>",
  "operationType": "DELETE_BASTION",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "BastionsResource",
      "entityUri": "/bastions/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T05:46:57.782Z",
  "timeFinished": "2026-09-01T05:47:43.587Z",
  "timeStarted": "2026-09-01T05:47:04.878Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[bastionsdk.Bastion, bastionsdk.CreateBastionDetails, bastionsdk.UpdateBastionDetails]{
		CollectionPath: "/20210331/bastions", ItemPath: "/20210331/bastions/<ocid:2>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState,
		ListShape:    ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:3>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ bastionsdk.CreateBastionDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20210331/workRequests/<ocid:3>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20210331/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20210331/workRequests/<ocid:6>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://bastion.us-ashburn-1.oci.oraclecloud.com", BasePath: "20210331", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := bastionsdk.BastionClient{BaseClient: session.BaseClient()}
	client := newBastionServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		sdkClient,
	)
	mockValidateCreated := func(current *bastionv1beta1.Bastion) error {
		if current.Status.LifecycleState != string(bastionsdk.BastionLifecycleStateActive) || current.Status.Name != mockBastionName {
			return fmt.Errorf("created Bastion status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *bastionv1beta1.Bastion) error {
		if current.Status.MaxSessionTtlInSeconds != 3600 || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated Bastion status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*bastionv1beta1.Bastion]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *bastionv1beta1.Bastion) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *bastionv1beta1.Bastion) {
			current.Spec.ClientCidrBlockAllowList = []string{"10.0.0.0/8"}
			current.Spec.MaxSessionTtlInSeconds = 3600
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *bastionv1beta1.Bastion) error {
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

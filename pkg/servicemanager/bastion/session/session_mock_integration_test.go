/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package session

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

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationSessionWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &bastionv1beta1.Session{}
	ocimock.InitializeResource(resource, "mock-session")
	resource.Spec = ocimock.MustJSONFixture[bastionv1beta1.SessionSpec](t, `
{
  "bastionId": "<ocid:1>",
  "displayName": "osok-mock-port-forwarding-session",
  "keyDetails": {
    "publicKeyContent": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIF+41fa69AexhQVkrvqibhTb317Sd3K/i6j/ARj1cMNS osok-mock"
  },
  "keyType": "PUB",
  "sessionTtlInSeconds": 1800,
  "targetResourceDetails": {
    "sessionType": "PORT_FORWARDING",
    "targetResourcePort": 22,
    "targetResourcePrivateIpAddress": "10.99.1.10"
  }
}
`)
	createRequest := ocimock.MustJSONFixture[bastionsdk.CreateSessionDetails](t, `
{
  "bastionId": "<ocid:1>",
  "displayName": "osok-mock-port-forwarding-session",
  "keyDetails": {
    "publicKeyContent": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIF+41fa69AexhQVkrvqibhTb317Sd3K/i6j/ARj1cMNS osok-mock"
  },
  "keyType": "PUB",
  "sessionTtlInSeconds": 1800,
  "targetResourceDetails": {
    "sessionType": "PORT_FORWARDING",
    "targetResourcePort": 22,
    "targetResourcePrivateIpAddress": "10.99.1.10"
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[bastionsdk.UpdateSessionDetails](t, `
{
  "displayName": "osok-mock-port-forwarding-session-updated"
}
`)
	createdState := ocimock.MustOCIResponseFixture[bastionsdk.Session](t, `
{
  "bastionId": "<ocid:1>",
  "bastionName": "osok-mock-batch5-bastion",
  "bastionPublicHostKeyInfo": null,
  "bastionUserName": "<ocid:3>",
  "displayName": "osok-mock-port-forwarding-session",
  "id": "<ocid:3>",
  "keyDetails": {
    "publicKeyContent": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIF+41fa69AexhQVkrvqibhTb317Sd3K/i6j/ARj1cMNS osok-mock"
  },
  "keyType": "PUB",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "sessionTtlInSeconds": 1800,
  "sshMetadata": {
    "command": "ssh -i <privateKey> -N -L <localPort>:10.99.1.10:22 -p 22 <ocid:3>@host.bastion.us-ashburn-1.oci.oraclecloud.com"
  },
  "targetResourceDetails": {
    "sessionType": "PORT_FORWARDING",
    "targetResourceDisplayName": null,
    "targetResourceFqdn": null,
    "targetResourceId": null,
    "targetResourcePort": 22,
    "targetResourcePrivateIpAddress": "10.99.1.10"
  },
  "timeCreated": "2026-09-02T21:21:29.450Z",
  "timeUpdated": "2026-09-02T21:21:32.514Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[bastionsdk.Session](t, `
{
  "bastionId": "<ocid:1>",
  "bastionName": "osok-mock-batch5-bastion",
  "bastionPublicHostKeyInfo": null,
  "bastionUserName": "<ocid:3>",
  "displayName": "osok-mock-port-forwarding-session-updated",
  "id": "<ocid:3>",
  "keyDetails": {
    "publicKeyContent": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIF+41fa69AexhQVkrvqibhTb317Sd3K/i6j/ARj1cMNS osok-mock"
  },
  "keyType": "PUB",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "sessionTtlInSeconds": 1800,
  "sshMetadata": {
    "command": "ssh -i <privateKey> -N -L <localPort>:10.99.1.10:22 -p 22 <ocid:3>@host.bastion.us-ashburn-1.oci.oraclecloud.com"
  },
  "targetResourceDetails": {
    "sessionType": "PORT_FORWARDING",
    "targetResourceDisplayName": null,
    "targetResourceFqdn": null,
    "targetResourceId": null,
    "targetResourcePort": 22,
    "targetResourcePrivateIpAddress": "10.99.1.10"
  },
  "timeCreated": "2026-09-02T21:21:29.450Z",
  "timeUpdated": "2026-09-02T21:21:36.028Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[bastionsdk.Session](t, `
{
  "bastionId": "<ocid:1>",
  "bastionName": "osok-mock-batch5-bastion",
  "bastionPublicHostKeyInfo": null,
  "bastionUserName": "<ocid:3>",
  "displayName": "osok-mock-port-forwarding-session-updated",
  "id": "<ocid:3>",
  "keyDetails": {
    "publicKeyContent": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIF+41fa69AexhQVkrvqibhTb317Sd3K/i6j/ARj1cMNS osok-mock"
  },
  "keyType": "PUB",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "sessionTtlInSeconds": 1800,
  "sshMetadata": {
    "command": "ssh -i <privateKey> -N -L <localPort>:10.99.1.10:22 -p 22 <ocid:3>@host.bastion.us-ashburn-1.oci.oraclecloud.com"
  },
  "targetResourceDetails": {
    "sessionType": "PORT_FORWARDING",
    "targetResourceDisplayName": null,
    "targetResourceFqdn": null,
    "targetResourceId": null,
    "targetResourcePort": 22,
    "targetResourcePrivateIpAddress": "10.99.1.10"
  },
  "timeCreated": "2026-09-02T21:21:29.450Z",
  "timeUpdated": "2026-09-02T21:21:42.793Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[bastionsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:4>",
  "id": "<ocid:2>",
  "operationType": "CREATE_SESSION",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "SessionResource",
      "entityUri": "/sessions/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T21:21:29.454Z",
  "timeFinished": "2026-09-02T21:21:32.514Z",
  "timeStarted": "2026-09-02T21:21:32.108Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[bastionsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:4>",
  "id": "<ocid:5>",
  "operationType": "DELETE_SESSION",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "SessionResource",
      "entityUri": "/sessions/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T21:21:36.997Z",
  "timeFinished": "2026-09-02T21:21:42.793Z",
  "timeStarted": "2026-09-02T21:21:42.725Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[bastionsdk.Session, bastionsdk.CreateSessionDetails, bastionsdk.UpdateSessionDetails]{
		CollectionPath: "/20210331/sessions", ItemPath: "/20210331/sessions/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState,
		ListShape:    ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 200, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ bastionsdk.CreateSessionDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20210331/workRequests/<ocid:2>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20210331/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
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
	client := newSessionServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient,
	)
	mockValidateCreated := func(current *bastionv1beta1.Session) error {
		if current.Status.DisplayName != "osok-mock-port-forwarding-session" || current.Status.LifecycleState != string(bastionsdk.SessionLifecycleStateActive) {
			return fmt.Errorf("created Session status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *bastionv1beta1.Session) error {
		if current.Status.DisplayName != "osok-mock-port-forwarding-session-updated" {
			return fmt.Errorf("updated Session status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*bastionv1beta1.Session]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *bastionv1beta1.Session) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *bastionv1beta1.Session) {
			current.Spec.DisplayName = "osok-mock-port-forwarding-session-updated"
		},
		ValidateUpdated: func(current *bastionv1beta1.Session) error {
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

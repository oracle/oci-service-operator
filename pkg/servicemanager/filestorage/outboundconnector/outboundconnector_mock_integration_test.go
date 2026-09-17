/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package outboundconnector

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/filestorage"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/filestorage/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationOutboundConnectorCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.OutboundConnector](t, `
{
  "metadata": {"name": "mock-outboundconnector", "namespace": "default"},
  "spec": {
  "availabilityDomain": "mock-availabilitydomain",
  "bindDistinguishedName": "mock-binddistinguishedname",
  "compartmentId": "<ocid:required>",
  "connectorType": "LDAPBIND",
  "displayName": "mock-displayname-initial",
  "endpoints": [
    {
      "hostname": "mock-hostname",
      "port": 1
    }
  ],
  "passwordSecretId": "<ocid:9>",
  "passwordSecretVersion": 2
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-outboundconnector")
	resource.Status = apiv1beta1.OutboundConnectorStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateLdapBindAccountDetails](t, `{
  "availabilityDomain": "mock-availabilitydomain",
  "bindDistinguishedName": "mock-binddistinguishedname",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "endpoints": [
    {
      "hostname": "mock-hostname",
      "port": 1
    }
  ],
  "passwordSecretId": "<ocid:9>",
  "passwordSecretVersion": 2
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateOutboundConnectorDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.LdapBindAccount](t, `{
  "availabilityDomain": "mock-availabilitydomain",
  "bindDistinguishedName": "mock-binddistinguishedname",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "endpoints": [
    {
      "hostname": "mock-hostname",
      "port": 1
    }
  ],
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "passwordSecretId": "<ocid:9>",
  "passwordSecretVersion": 2,
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.LdapBindAccount](t, `{
  "availabilityDomain": "mock-availabilitydomain",
  "bindDistinguishedName": "mock-binddistinguishedname",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "endpoints": [
    {
      "hostname": "mock-hostname",
      "port": 1
    }
  ],
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "passwordSecretId": "<ocid:9>",
  "passwordSecretVersion": 2,
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)

	creatingState := createdState
	creatingState.LifecycleState = "CREATING"
	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.LdapBindAccount, sdksvc.CreateLdapBindAccountDetails, sdksvc.UpdateOutboundConnectorDetails]{
		CollectionPath: "/20171215/outboundConnectors", ItemPath: "/20171215/outboundConnectors/<ocid:1>",
		CreatePath: "/20171215/outboundConnectors", CreateMethod: http.MethodPost,
		UpdatePath: "/20171215/outboundConnectors/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20171215/outboundConnectors/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},

		CreatedState:      &creatingState,
		CreatedReadStates: ocimock.StateSequence(creatingState, createdState),
		UpdateRequest:     &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateOutboundConnectorDetails],
		UpdatedState:      &updatedState,
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "connectorType", "LDAPBIND", createRequest)
		},

		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://filestorage.us-ashburn-1.oci.oraclecloud.com", BasePath: "20171215", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.FileStorageClient{BaseClient: session.BaseClient()}
	manager := &OutboundConnectorServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newOutboundConnectorRuntimeHooks(manager, sdkClient)
	client := wrapOutboundConnectorGeneratedClient(hooks, defaultOutboundConnectorServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.OutboundConnector](buildOutboundConnectorGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.OutboundConnector]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.OutboundConnector) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OutboundConnector status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.OutboundConnector) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.OutboundConnector) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OutboundConnector status = %+v", current.Status)
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

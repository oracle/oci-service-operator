/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package agent

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/cloudbridge"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/cloudbridge/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationAgentCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Agent](t, `
{
  "metadata": {"name": "mock-agent", "namespace": "default"},
  "spec": {
  "agentType": "APPLIANCE",
  "agentVersion": "1.0",
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "environmentId": "<ocid:2>",
  "osVersion": "Oracle Linux 8"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-agent")
	resource.Status = apiv1beta1.AgentStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateAgentDetails](t, `{
  "agentType": "APPLIANCE",
  "agentVersion": "1.0",
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "environmentId": "<ocid:2>",
  "osVersion": "Oracle Linux 8"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateAgentDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.Agent](t, `{
  "agentType": "APPLIANCE",
  "agentVersion": "1.0",
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "environmentId": "<ocid:2>",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "osVersion": "Oracle Linux 8",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.Agent](t, `{
  "agentType": "APPLIANCE",
  "agentVersion": "1.0",
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-updated",
  "environmentId": "<ocid:2>",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "osVersion": "Oracle Linux 8",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)

	creatingState := createdState
	creatingState.LifecycleState = "CREATING"
	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.Agent, sdksvc.CreateAgentDetails, sdksvc.UpdateAgentDetails]{
		CollectionPath: "/20220509/agents", ItemPath: "/20220509/agents/<ocid:3>",
		CreatePath: "/20220509/agents", CreateMethod: http.MethodPost,
		UpdatePath: "/20220509/agents/<ocid:3>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220509/agents/<ocid:3>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateAgentDetails],
		CreatedState:      &creatingState,
		CreatedReadStates: ocimock.StateSequence(creatingState, createdState),
		UpdateRequest:     &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateAgentDetails],
		UpdatedState:      &updatedState,
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateAgentDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudbridge.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220509", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.OcbAgentSvcClient{BaseClient: session.BaseClient()}
	manager := &AgentServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAgentRuntimeHooks(manager, sdkClient)
	client := wrapAgentGeneratedClient(hooks, defaultAgentServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Agent](buildAgentGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Agent]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Agent) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Agent status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Agent) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Agent) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Agent status = %+v", current.Status)
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

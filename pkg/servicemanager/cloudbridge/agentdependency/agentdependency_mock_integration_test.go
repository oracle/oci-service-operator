/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package agentdependency

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
func TestMockIntegrationAgentDependencyCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.AgentDependency](t, `
{
  "metadata": {"name": "mock-agentdependency", "namespace": "default"},
  "spec": {
  "bucket": "mock-agent-dependencies",
  "compartmentId": "<ocid:1>",
  "dependencyName": "VDDK",
  "dependencyVersion": "1.0",
  "displayName": "mock-displayname-initial",
  "namespace": "mock",
  "objectName": "vddk.zip"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-agentdependency")
	resource.Status = apiv1beta1.AgentDependencyStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateAgentDependencyDetails](t, `{
  "bucket": "mock-agent-dependencies",
  "compartmentId": "<ocid:1>",
  "dependencyName": "VDDK",
  "dependencyVersion": "1.0",
  "displayName": "mock-displayname-initial",
  "namespace": "mock",
  "objectName": "vddk.zip"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateAgentDependencyDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.AgentDependency](t, `{
  "bucket": "mock-agent-dependencies",
  "compartmentId": "<ocid:1>",
  "dependencyName": "VDDK",
  "dependencyVersion": "1.0",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "namespace": "mock",
  "objectName": "vddk.zip",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.AgentDependency](t, `{
  "bucket": "mock-agent-dependencies",
  "compartmentId": "<ocid:1>",
  "dependencyName": "VDDK",
  "dependencyVersion": "1.0",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "namespace": "mock",
  "objectName": "vddk.zip",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEAGENTDEPENDENCY",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "AgentDependency", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEAGENTDEPENDENCY",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "AgentDependency", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.AgentDependency, sdksvc.CreateAgentDependencyDetails, sdksvc.UpdateAgentDependencyDetails]{
		CollectionPath: "/20220509/agentDependencies", ItemPath: "/20220509/agentDependencies/<ocid:2>",
		CreatePath: "/20220509/agentDependencies", CreateMethod: http.MethodPost,
		UpdatePath: "/20220509/agentDependencies/<ocid:2>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220509/agentDependencies/<ocid:2>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateAgentDependencyDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateAgentDependencyDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateAgentDependencyDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20220509/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20220509/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudbridge.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220509", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := AgentDependencySDKClients{ocbAgentSvcClient: sdksvc.OcbAgentSvcClient{BaseClient: session.BaseClient()}, commonClient: sdksvc.CommonClient{BaseClient: session.BaseClient()}}
	manager := &AgentDependencyServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAgentDependencyRuntimeHooks(manager, sdkClient)
	client := wrapAgentDependencyGeneratedClient(hooks, defaultAgentDependencyServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.AgentDependency](buildAgentDependencyGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.AgentDependency]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.AgentDependency) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created AgentDependency status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.AgentDependency) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.AgentDependency) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated AgentDependency status = %+v", current.Status)
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

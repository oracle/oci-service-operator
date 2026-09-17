/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package workspace

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/dataintegration"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/dataintegration/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationWorkspaceCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Workspace](t, `
{
  "metadata": {"name": "mock-workspace", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "description": "portable mock",
  "displayName": "mock-displayname-initial",
  "dnsServerIp": "mock-dnsserverip-updated",
  "dnsServerZone": "mock-dnsserverzone-updated",
  "endpointId": "<ocid:9>",
  "endpointName": "mock-endpointname-updated",
  "isPrivateNetworkEnabled": true,
  "registryId": "<ocid:9>",
  "subnetId": "<ocid:9>",
  "vcnId": "<ocid:9>"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-workspace")
	resource.Status = apiv1beta1.WorkspaceStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateWorkspaceDetails](t, `{
  "compartmentId": "<ocid:1>",
  "description": "portable mock",
  "displayName": "mock-displayname-initial",
  "dnsServerIp": "mock-dnsserverip-updated",
  "dnsServerZone": "mock-dnsserverzone-updated",
  "endpointId": "<ocid:9>",
  "endpointName": "mock-endpointname-updated",
  "isPrivateNetworkEnabled": true,
  "registryId": "<ocid:9>",
  "subnetId": "<ocid:9>",
  "vcnId": "<ocid:9>"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateWorkspaceDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.Workspace](t, `{
  "compartmentId": "<ocid:1>",
  "description": "portable mock",
  "displayName": "mock-displayname-initial",
  "dnsServerIp": "mock-dnsserverip-updated",
  "dnsServerZone": "mock-dnsserverzone-updated",
  "endpointId": "<ocid:9>",
  "endpointName": "mock-endpointname-updated",
  "id": "<ocid:2>",
  "isPrivateNetworkEnabled": true,
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "registryId": "<ocid:9>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "subnetId": "<ocid:9>",
  "vcnId": "<ocid:9>"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.Workspace](t, `{
  "compartmentId": "<ocid:1>",
  "description": "portable mock",
  "displayName": "mock-displayname-updated",
  "dnsServerIp": "mock-dnsserverip-updated",
  "dnsServerZone": "mock-dnsserverzone-updated",
  "endpointId": "<ocid:9>",
  "endpointName": "mock-endpointname-updated",
  "id": "<ocid:2>",
  "isPrivateNetworkEnabled": true,
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "registryId": "<ocid:9>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "subnetId": "<ocid:9>",
  "vcnId": "<ocid:9>"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEWORKSPACE",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "Workspace", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEWORKSPACE",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "Workspace", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEWORKSPACE",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "Workspace", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.Workspace, sdksvc.CreateWorkspaceDetails, sdksvc.UpdateWorkspaceDetails]{
		CollectionPath: "/20200430/workspaces", ItemPath: "/20200430/workspaces/<ocid:2>",
		CreatePath: "/20200430/workspaces", CreateMethod: http.MethodPost,
		UpdatePath: "/20200430/workspaces/<ocid:2>", UpdateMethod: http.MethodPut,
		DeletePath: "/20200430/workspaces/<ocid:2>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateWorkspaceDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateWorkspaceDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateWorkspaceDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200430/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200430/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200430/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dataintegration.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200430", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.DataIntegrationClient{BaseClient: session.BaseClient()}
	manager := &WorkspaceServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newWorkspaceRuntimeHooks(manager, sdkClient)
	client := wrapWorkspaceGeneratedClient(hooks, defaultWorkspaceServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Workspace](buildWorkspaceGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Workspace]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Workspace) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Workspace status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Workspace) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Workspace) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Workspace status = %+v", current.Status)
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

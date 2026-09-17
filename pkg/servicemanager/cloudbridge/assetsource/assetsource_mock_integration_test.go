/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package assetsource

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
func TestMockIntegrationAssetSourceCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.AssetSource](t, `
{
  "metadata": {"name": "mock-assetsource", "namespace": "default"},
  "spec": {
  "assetsCompartmentId": "<ocid:required>",
  "compartmentId": "<ocid:required>",
  "discoveryCredentials": {
    "secretId": "<ocid:required>",
    "type": "BASIC"
  },
  "displayName": "mock-displayname-initial",
  "environmentId": "<ocid:required>",
  "inventoryId": "<ocid:required>",
  "type": "VMWARE",
  "vcenterEndpoint": "mock-vcenterendpoint"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-assetsource")
	resource.Status = apiv1beta1.AssetSourceStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateVmWareAssetSourceDetails](t, `{
  "assetsCompartmentId": "<ocid:required>",
  "compartmentId": "<ocid:required>",
  "discoveryCredentials": {
    "secretId": "<ocid:required>",
    "type": "BASIC"
  },
  "displayName": "mock-displayname-initial",
  "environmentId": "<ocid:required>",
  "inventoryId": "<ocid:required>",
  "vcenterEndpoint": "mock-vcenterendpoint"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateVmWareAssetSourceDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.VmWareAssetSource](t, `{
  "assetsCompartmentId": "<ocid:required>",
  "compartmentId": "<ocid:required>",
  "discoveryCredentials": {
    "secretId": "<ocid:required>",
    "type": "BASIC"
  },
  "displayName": "mock-displayname-initial",
  "environmentId": "<ocid:required>",
  "id": "<ocid:1>",
  "inventoryId": "<ocid:required>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "vcenterEndpoint": "mock-vcenterendpoint"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.VmWareAssetSource](t, `{
  "assetsCompartmentId": "<ocid:required>",
  "compartmentId": "<ocid:required>",
  "discoveryCredentials": {
    "secretId": "<ocid:required>",
    "type": "BASIC"
  },
  "displayName": "mock-displayname-updated",
  "environmentId": "<ocid:required>",
  "id": "<ocid:1>",
  "inventoryId": "<ocid:required>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "vcenterEndpoint": "mock-vcenterendpoint"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEASSETSOURCE",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "AssetSource", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEASSETSOURCE",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "AssetSource", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEASSETSOURCE",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "AssetSource", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.VmWareAssetSource, sdksvc.CreateVmWareAssetSourceDetails, sdksvc.UpdateVmWareAssetSourceDetails]{
		CollectionPath: "/20220509/assetSources", ItemPath: "/20220509/assetSources/<ocid:1>",
		CreatePath: "/20220509/assetSources", CreateMethod: http.MethodPost,
		UpdatePath: "/20220509/assetSources/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220509/assetSources/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},

		CreatedState: &createdState,

		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "type", "VMWARE", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "type", "VMWARE", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20220509/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20220509/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20220509/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
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
	sdkClient := AssetSourceSDKClients{discoveryClient: sdksvc.DiscoveryClient{BaseClient: session.BaseClient()}, commonClient: sdksvc.CommonClient{BaseClient: session.BaseClient()}}
	manager := &AssetSourceServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAssetSourceRuntimeHooks(manager, sdkClient)
	client := wrapAssetSourceGeneratedClient(hooks, defaultAssetSourceServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.AssetSource](buildAssetSourceGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.AssetSource]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.AssetSource) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created AssetSource status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.AssetSource) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.AssetSource) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated AssetSource status = %+v", current.Status)
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

/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package metastore

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/datacatalog"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/datacatalog/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationMetastoreCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Metastore](t, `
{
  "metadata": {"name": "mock-metastore", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "defaultExternalTableLocation": "mock-defaultexternaltablelocation",
  "defaultManagedTableLocation": "mock-defaultmanagedtablelocation",
  "displayName": "mock-displayname-initial"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-metastore")
	resource.Status = apiv1beta1.MetastoreStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateMetastoreDetails](t, `{
  "compartmentId": "<ocid:required>",
  "defaultExternalTableLocation": "mock-defaultexternaltablelocation",
  "defaultManagedTableLocation": "mock-defaultmanagedtablelocation",
  "displayName": "mock-displayname-initial"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateMetastoreDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.Metastore](t, `{
  "compartmentId": "<ocid:required>",
  "defaultExternalTableLocation": "mock-defaultexternaltablelocation",
  "defaultManagedTableLocation": "mock-defaultmanagedtablelocation",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.Metastore](t, `{
  "compartmentId": "<ocid:required>",
  "defaultExternalTableLocation": "mock-defaultexternaltablelocation",
  "defaultManagedTableLocation": "mock-defaultmanagedtablelocation",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEMETASTORE",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "Metastore", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEMETASTORE",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "Metastore", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.Metastore, sdksvc.CreateMetastoreDetails, sdksvc.UpdateMetastoreDetails]{
		CollectionPath: "/20190325/metastores", ItemPath: "/20190325/metastores/<ocid:1>",
		CreatePath: "/20190325/metastores", CreateMethod: http.MethodPost,
		UpdatePath: "/20190325/metastores/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20190325/metastores/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateMetastoreDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateMetastoreDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 200, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateMetastoreDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20190325/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20190325/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datacatalog.us-ashburn-1.oci.oraclecloud.com", BasePath: "20190325", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.DataCatalogClient{BaseClient: session.BaseClient()}
	manager := &MetastoreServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newMetastoreRuntimeHooks(manager, sdkClient)
	client := wrapMetastoreGeneratedClient(hooks, defaultMetastoreServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Metastore](buildMetastoreGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Metastore]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Metastore) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Metastore status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Metastore) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Metastore) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Metastore status = %+v", current.Status)
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

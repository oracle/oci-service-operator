/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package pipeline

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/goldengate"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/goldengate/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationPipelineCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Pipeline](t, `
{
  "metadata": {"name": "mock-pipeline", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "licenseModel": "LICENSE_INCLUDED",
  "recipeType": "ZERO_ETL",
  "sourceConnectionDetails": {
    "connectionId": "<ocid:required>"
  },
  "targetConnectionDetails": {
    "connectionId": "<ocid:required>"
  }
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-pipeline")
	resource.Status = apiv1beta1.PipelineStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateZeroEtlPipelineDetails](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "licenseModel": "LICENSE_INCLUDED",
  "sourceConnectionDetails": {
    "connectionId": "<ocid:required>"
  },
  "targetConnectionDetails": {
    "connectionId": "<ocid:required>"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateZeroEtlPipelineDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.ZeroEtlPipeline](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "licenseModel": "LICENSE_INCLUDED",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "sourceConnectionDetails": {
    "connectionId": "<ocid:required>"
  },
  "state": "ACTIVE",
  "status": "ACTIVE",
  "targetConnectionDetails": {
    "connectionId": "<ocid:required>"
  },
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.ZeroEtlPipeline](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "licenseModel": "LICENSE_INCLUDED",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "sourceConnectionDetails": {
    "connectionId": "<ocid:required>"
  },
  "state": "ACTIVE",
  "status": "ACTIVE",
  "targetConnectionDetails": {
    "connectionId": "<ocid:required>"
  },
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEPIPELINE",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "Pipeline", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEPIPELINE",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "Pipeline", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEPIPELINE",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "Pipeline", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.ZeroEtlPipeline, sdksvc.CreateZeroEtlPipelineDetails, sdksvc.UpdateZeroEtlPipelineDetails]{
		CollectionPath: "/20200407/pipelines", ItemPath: "/20200407/pipelines/<ocid:1>",
		CreatePath: "/20200407/pipelines", CreateMethod: http.MethodPost,
		UpdatePath: "/20200407/pipelines/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20200407/pipelines/<ocid:1>", DeleteMethod: http.MethodDelete,
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
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "recipeType", "ZERO_ETL", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "recipeType", "ZERO_ETL", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://goldengate.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200407", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.GoldenGateClient{BaseClient: session.BaseClient()}
	manager := &PipelineServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newPipelineRuntimeHooks(manager, sdkClient)
	client := wrapPipelineGeneratedClient(hooks, defaultPipelineServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Pipeline](buildPipelineGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Pipeline]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Pipeline) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Pipeline status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Pipeline) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Pipeline) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Pipeline status = %+v", current.Status)
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

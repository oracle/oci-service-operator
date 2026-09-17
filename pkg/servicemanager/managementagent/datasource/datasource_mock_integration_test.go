/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package datasource

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	managementagentsdk "github.com/oracle/oci-go-sdk/v65/managementagent"
	managementagentv1beta1 "github.com/oracle/oci-service-operator/api/managementagent/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationDataSourceWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[managementagentv1beta1.DataSource](t, `
{
  "metadata": {
    "annotations": {
      "managementagent.oracle.com/management-agent-id": "management-agent-1"
    },
    "creationTimestamp": null,
    "name": "sample",
    "namespace": "default",
    "uid": "uid-123"
  },
  "spec": {
    "compartmentId": "compartment-1",
    "name": "sample-ds",
    "namespace": "oci_metrics",
    "type": "PROMETHEUS_EMITTER",
    "url": "http://prometheus.example"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-datasource")
	resource.Status = managementagentv1beta1.DataSourceStatus{}
	createRequest := ocimock.MustJSONFixture[managementagentsdk.CreatePrometheusEmitterDataSourceDetails](t, `
{
  "compartmentId": "compartment-1",
  "name": "sample-ds",
  "namespace": "oci_metrics",
  "url": "http://prometheus.example"
}
`)
	updateRequest := ocimock.MustJSONFixture[managementagentsdk.UpdatePrometheusEmitterDataSourceDetails](t, `
{
  "url": "http://prometheus-updated.example"
}
`)
	createdState := ocimock.MustOCIResponseFixture[managementagentsdk.PrometheusEmitterDataSource](t, `
{
  "compartmentId": "compartment-1",
  "id": "<ocid:2>",
  "key": "datasource-key",
  "lifecycleState": "ACTIVE",
  "managementAgentId": "management-agent-1",
  "name": "sample-ds",
  "namespace": "oci_metrics",
  "resourceId": "<ocid:2>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "type": "PROMETHEUS_EMITTER",
  "url": "http://prometheus.example"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[managementagentsdk.PrometheusEmitterDataSource](t, `
{
  "compartmentId": "compartment-1",
  "id": "<ocid:2>",
  "key": "datasource-key",
  "lifecycleState": "ACTIVE",
  "managementAgentId": "management-agent-1",
  "name": "sample-ds",
  "namespace": "oci_metrics",
  "resourceId": "<ocid:2>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "type": "PROMETHEUS_EMITTER",
  "url": "http://prometheus-updated.example"
}
`)

	createWorkRequest := ocimock.MustOCIResponseFixture[managementagentsdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_DATA_SOURCE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "DataSource",
      "identifier": "datasource-key"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[managementagentsdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_DATA_SOURCE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "DataSource",
      "identifier": "datasource-key"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[managementagentsdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_DATA_SOURCE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "DataSource",
      "identifier": "datasource-key"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[managementagentsdk.PrometheusEmitterDataSource, managementagentsdk.CreatePrometheusEmitterDataSourceDetails, managementagentsdk.UpdatePrometheusEmitterDataSourceDetails]{
		CollectionPath: "/20200202/managementAgents/management-agent-1/dataSources", ItemPath: "/20200202/managementAgents/management-agent-1/dataSources/datasource-key",
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeArray, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "PROMETHEUS_EMITTER", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "type", "PROMETHEUS_EMITTER", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200202/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200202/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200202/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://managementagent.mock.invalid", BasePath: "20200202", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := managementagentsdk.ManagementAgentClient{BaseClient: session.BaseClient()}
	client := newDataSourceRuntimeClient(sdkClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*managementagentv1beta1.DataSource]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *managementagentv1beta1.DataSource) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.State != "ACTIVE" || current.Status.Url != resource.Spec.Url ||
				current.Status.Type != resource.Spec.Type || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DataSource status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *managementagentv1beta1.DataSource) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "url": "http://prometheus-updated.example"
}`)
		},
		ValidateUpdated: func(current *managementagentv1beta1.DataSource) error {
			if !(current.Status.Url == "http://prometheus-updated.example") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DataSource status = %+v", current.Status)
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

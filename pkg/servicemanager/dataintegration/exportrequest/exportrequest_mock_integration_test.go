/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package exportrequest

import (
	"context"
	"fmt"
	"testing"

	dataintegrationsdk "github.com/oracle/oci-go-sdk/v65/dataintegration"
	dataintegrationv1beta1 "github.com/oracle/oci-service-operator/api/dataintegration/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationExportRequestCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.ExportRequest{}
	ocimock.InitializeResource(resource, "mock-exportrequest")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.ExportRequestSpec](t, `{
  "workspaceId": "<ocid:1>",
  "bucketName": "mock-bucket",
  "fileName": "export.zip"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"status":"TERMINATING"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateExportRequestDetails](t, `{
  "bucketName": "mock-bucket",
  "fileName": "export.zip"
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateExportRequestDetails](t, `{
  "status": "TERMINATING"
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.ExportRequest](t, `{
  "bucketName": "mock-bucket",
  "fileName": "export.zip",
  "key": "resource-key",
  "status": "SUCCESSFUL"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.ExportRequest](t, `{
  "bucketName": "mock-bucket",
  "fileName": "export.zip",
  "key": "resource-key",
  "status": "TERMINATED"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.ExportRequest, dataintegrationsdk.CreateExportRequestDetails, dataintegrationsdk.UpdateExportRequestDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/exportRequests", ItemPath: "/20200430/workspaces/<ocid:1>/exportRequests/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateExportRequestDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dataintegration.mock.invalid", BasePath: "20200430", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := dataintegrationsdk.DataIntegrationClient{BaseClient: session.BaseClient()}
	manager := &ExportRequestServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newExportRequestRuntimeHooks(manager, sdkClient)
	client := wrapExportRequestGeneratedClient(hooks, defaultExportRequestServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.ExportRequest](buildExportRequestGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.ExportRequest]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.ExportRequest) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Status != "SUCCESSFUL" {
				return fmt.Errorf("created ExportRequest status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.ExportRequest) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.ExportRequest) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Status != "TERMINATED" {
				return fmt.Errorf("updated ExportRequest status = %+v", current.Status)
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

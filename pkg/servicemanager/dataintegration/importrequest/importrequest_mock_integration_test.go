/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package importrequest

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

func TestMockIntegrationImportRequestCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.ImportRequest{}
	ocimock.InitializeResource(resource, "mock-importrequest")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.ImportRequestSpec](t, `{
  "workspaceId": "<ocid:1>",
  "bucketName": "mock-bucket",
  "fileName": "import.zip"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"status":"TERMINATING"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateImportRequestDetails](t, `{
  "bucketName": "mock-bucket",
  "fileName": "import.zip"
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateImportRequestDetails](t, `{
  "status": "TERMINATING"
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.ImportRequest](t, `{
  "bucketName": "mock-bucket",
  "fileName": "import.zip",
  "key": "resource-key",
  "status": "SUCCESSFUL"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.ImportRequest](t, `{
  "bucketName": "mock-bucket",
  "fileName": "import.zip",
  "key": "resource-key",
  "status": "TERMINATED"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.ImportRequest, dataintegrationsdk.CreateImportRequestDetails, dataintegrationsdk.UpdateImportRequestDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/importRequests", ItemPath: "/20200430/workspaces/<ocid:1>/importRequests/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateImportRequestDetails) error {
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
	manager := &ImportRequestServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newImportRequestRuntimeHooks(manager, sdkClient)
	client := wrapImportRequestGeneratedClient(hooks, defaultImportRequestServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.ImportRequest](buildImportRequestGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.ImportRequest]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.ImportRequest) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Status != "SUCCESSFUL" {
				return fmt.Errorf("created ImportRequest status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.ImportRequest) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.ImportRequest) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Status != "TERMINATED" {
				return fmt.Errorf("updated ImportRequest status = %+v", current.Status)
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

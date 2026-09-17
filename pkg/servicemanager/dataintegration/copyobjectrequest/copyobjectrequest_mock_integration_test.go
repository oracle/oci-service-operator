/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package copyobjectrequest

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

func TestMockIntegrationCopyObjectRequestCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.CopyObjectRequest{}
	ocimock.InitializeResource(resource, "mock-copyobjectrequest")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.CopyObjectRequestSpec](t, `{
  "workspaceId": "<ocid:1>",
  "sourceWorkspaceId": "<ocid:2>",
  "objectKeys": [
    "object-key"
  ],
  "copyConflictResolution": {
    "requestType": "RETAIN"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"status":"TERMINATING"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateCopyObjectRequestDetails](t, `{
  "sourceWorkspaceId": "<ocid:2>",
  "objectKeys": [
    "object-key"
  ],
  "copyConflictResolution": {
    "requestType": "RETAIN"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateCopyObjectRequestDetails](t, `{
  "status": "TERMINATING"
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.CopyObjectRequest](t, `{
  "sourceWorkspaceId": "<ocid:2>",
  "objectKeys": [
    "object-key"
  ],
  "copyConflictResolution": {
    "requestType": "RETAIN"
  },
  "key": "resource-key",
  "copyMetadataObjectRequestStatus": "SUCCESSFUL"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.CopyObjectRequest](t, `{
  "sourceWorkspaceId": "<ocid:2>",
  "objectKeys": [
    "object-key"
  ],
  "copyConflictResolution": {
    "requestType": "RETAIN"
  },
  "key": "resource-key",
  "copyMetadataObjectRequestStatus": "TERMINATED",
  "status": "TERMINATING"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.CopyObjectRequest, dataintegrationsdk.CreateCopyObjectRequestDetails, dataintegrationsdk.UpdateCopyObjectRequestDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/copyObjectRequests", ItemPath: "/20200430/workspaces/<ocid:1>/copyObjectRequests/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateCopyObjectRequestDetails) error {
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
	manager := &CopyObjectRequestServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newCopyObjectRequestRuntimeHooks(manager, sdkClient)
	client := wrapCopyObjectRequestGeneratedClient(hooks, defaultCopyObjectRequestServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.CopyObjectRequest](buildCopyObjectRequestGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.CopyObjectRequest]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.CopyObjectRequest) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.CopyMetadataObjectRequestStatus != "SUCCESSFUL" {
				return fmt.Errorf("created CopyObjectRequest status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.CopyObjectRequest) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.CopyObjectRequest) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.CopyMetadataObjectRequestStatus != "TERMINATED" {
				return fmt.Errorf("updated CopyObjectRequest status = %+v", current.Status)
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

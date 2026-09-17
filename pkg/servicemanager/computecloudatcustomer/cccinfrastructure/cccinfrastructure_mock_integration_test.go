/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package cccinfrastructure

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/computecloudatcustomer"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/computecloudatcustomer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationCccInfrastructureCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.CccInfrastructure](t, `
{
  "metadata": {"name": "mock-cccinfrastructure", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "description": "synthetic Compute Cloud at Customer infrastructure",
  "displayName": "mock-displayname-initial",
  "subnetId": "<ocid:2>"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-cccinfrastructure")
	resource.Status = apiv1beta1.CccInfrastructureStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateCccInfrastructureDetails](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic Compute Cloud at Customer infrastructure",
  "displayName": "mock-displayname-initial",
  "subnetId": "<ocid:2>"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateCccInfrastructureDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.CccInfrastructure](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic Compute Cloud at Customer infrastructure",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "subnetId": "<ocid:2>"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.CccInfrastructure](t, `{
  "compartmentId": "<ocid:1>",
  "description": "synthetic Compute Cloud at Customer infrastructure",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "subnetId": "<ocid:2>"
}`)

	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.CccInfrastructure, sdksvc.CreateCccInfrastructureDetails, sdksvc.UpdateCccInfrastructureDetails]{
		CollectionPath: "/20221208/cccInfrastructures", ItemPath: "/20221208/cccInfrastructures/<ocid:3>",
		CreatePath: "/20221208/cccInfrastructures", CreateMethod: http.MethodPost,
		UpdatePath: "/20221208/cccInfrastructures/<ocid:3>", UpdateMethod: http.MethodPut,
		DeletePath: "/20221208/cccInfrastructures/<ocid:3>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateCccInfrastructureDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateCccInfrastructureDetails],
		UpdatedState:      &updatedState,
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateCccInfrastructureDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://ccc.us-ashburn-1.oci.oraclecloud.com", BasePath: "20221208", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.ComputeCloudAtCustomerClient{BaseClient: session.BaseClient()}
	manager := &CccInfrastructureServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newCccInfrastructureRuntimeHooks(manager, sdkClient)
	client := wrapCccInfrastructureGeneratedClient(hooks, defaultCccInfrastructureServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.CccInfrastructure](buildCccInfrastructureGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.CccInfrastructure]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.CccInfrastructure) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created CccInfrastructure status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.CccInfrastructure) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.CccInfrastructure) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated CccInfrastructure status = %+v", current.Status)
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

/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package target

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/cloudguard"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationTargetCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Target](t, `
{
  "metadata": {"name": "mock-target", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic target",
  "displayName": "mock-displayname-initial",
  "targetResourceId": "<ocid:1>",
  "targetResourceType": "COMPARTMENT"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-target")
	resource.Status = apiv1beta1.TargetStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateTargetDetails](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic target",
  "displayName": "mock-displayname-initial",
  "targetResourceId": "<ocid:1>",
  "targetResourceType": "COMPARTMENT"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateTargetDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.Target](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic target",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "targetResourceId": "<ocid:1>",
  "targetResourceType": "COMPARTMENT"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.Target](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic target",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "targetResourceId": "<ocid:1>",
  "targetResourceType": "COMPARTMENT"
}`)

	creatingState := createdState
	creatingState.LifecycleState = "CREATING"
	updatingState := updatedState
	updatingState.LifecycleState = "UPDATING"
	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.Target, sdksvc.CreateTargetDetails, sdksvc.UpdateTargetDetails]{
		CollectionPath: "/20200131/targets", ItemPath: "/20200131/targets/<ocid:2>",
		CreatePath: "/20200131/targets", CreateMethod: http.MethodPost,
		UpdatePath: "/20200131/targets/<ocid:2>", UpdateMethod: http.MethodPut,
		DeletePath: "/20200131/targets/<ocid:2>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateTargetDetails],
		CreatedState:      &creatingState,
		CreatedReadStates: ocimock.StateSequence(creatingState, createdState),
		UpdateRequest:     &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateTargetDetails],
		UpdatedState:      &updatingState,
		UpdatedReadStates: ocimock.StateSequence(updatingState, updatedState),
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateTargetDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudguard-cp-api.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200131", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.CloudGuardClient{BaseClient: session.BaseClient()}
	manager := &TargetServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTargetRuntimeHooks(manager, sdkClient)
	client := wrapTargetGeneratedClient(hooks, defaultTargetServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Target](buildTargetGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Target]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Target) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Target status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Target) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Target) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Target status = %+v", current.Status)
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

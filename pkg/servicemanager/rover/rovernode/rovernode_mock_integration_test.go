/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package rovernode

import (
	"context"
	"fmt"
	roversdk "github.com/oracle/oci-go-sdk/v65/rover"
	roverv1beta1 "github.com/oracle/oci-service-operator/api/rover/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationRoverNodeLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &roverv1beta1.RoverNode{Spec: roverv1beta1.RoverNodeSpec{
		DisplayName: "osok-mock-rover-node", CompartmentId: "ocid1.compartment.oc1..mock", Shape: "Rover.Node.1.168",
	}}
	ocimock.InitializeResource(resource, "mock-rovernode")
	resource.Spec = ocimock.MustJSONFixture[roverv1beta1.RoverNodeSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-rover-node",
  "shape": "Rover.Node.1.168"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-rover-node-updated"
}`)
	createRequest := ocimock.MustJSONFixture[roversdk.CreateRoverNodeDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-rover-node",
  "shape": "Rover.Node.1.168"
}`)
	createdState := ocimock.MustOCIResponseFixture[roversdk.RoverNode](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-rover-node",
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "shape": "Rover.Node.1.168"
}`)
	updateRequest := ocimock.MustJSONFixture[roversdk.UpdateRoverNodeDetails](t, `{
  "displayName": "osok-mock-rover-node-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[roversdk.RoverNode](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-rover-node-updated",
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "shape": "Rover.Node.1.168"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		roversdk.RoverNode,
		roversdk.CreateRoverNodeDetails,
		roversdk.UpdateRoverNodeDetails,
	]{
		CollectionPath:     "/20201210/roverNodes",
		ItemPath:           "/20201210/roverNodes/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		CreatedReadStates:  ocimock.LifecycleStateSequence(t, createdState, "CREATING"),
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		DeleteEndsNotFound: true,
		UpdatedReadStates:  ocimock.LifecycleStateSequence(t, updatedState, "UPDATING"),
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ roversdk.CreateRoverNodeDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ roversdk.RoverNode) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20201210", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close RoverNode OCI mock: %v", err)
		}
	})
	sdkClient := roversdk.RoverNodeClient{BaseClient: session.BaseClient()}
	manager := &RoverNodeServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newRoverNodeRuntimeHooks(manager, sdkClient)
	client := wrapRoverNodeGeneratedClient(hooks, defaultRoverNodeServiceClient{ServiceClient: generatedruntime.NewServiceClient[*roverv1beta1.RoverNode](buildRoverNodeGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*roverv1beta1.RoverNode]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *roverv1beta1.RoverNode) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.Shape, current.Spec.Shape) {
				return fmt.Errorf("created RoverNode status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *roverv1beta1.RoverNode) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *roverv1beta1.RoverNode) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated RoverNode status = %+v", current.Status)
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

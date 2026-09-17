/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package operatorcontrol

import (
	"context"
	"fmt"
	operatoraccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/operatoraccesscontrol"
	operatoraccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/operatoraccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationOperatorControlLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &operatoraccesscontrolv1beta1.OperatorControl{Spec: operatoraccesscontrolv1beta1.OperatorControlSpec{
		OperatorControlName: "osok-mock-operator-control", ApproverGroupsList: []string{"ocid1.group.oc1..mock"},
		ResourceType: string(operatoraccesscontrolsdk.ResourceTypesExacc), CompartmentId: "ocid1.compartment.oc1..mock",
		Description: "synthetic operator control", NumberOfApprovers: 1,
	}}
	ocimock.InitializeResource(resource, "mock-operatorcontrol")
	resource.Spec = ocimock.MustJSONFixture[operatoraccesscontrolv1beta1.OperatorControlSpec](t, `{
  "approverGroupsList": [
    "\u003cocid:1\u003e"
  ],
  "compartmentId": "\u003cocid:2\u003e",
  "description": "synthetic operator control",
  "isFullyPreApproved": false,
  "numberOfApprovers": 1,
  "operatorControlName": "osok-mock-operator-control",
  "resourceType": "EXACC"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "approverGroupsList": [
    "\u003cocid:1\u003e"
  ],
  "description": "synthetic operator control-updated",
  "isFullyPreApproved": false,
  "operatorControlName": "osok-mock-operator-control"
}`)
	createRequest := ocimock.MustJSONFixture[operatoraccesscontrolsdk.CreateOperatorControlDetails](t, `{
  "approverGroupsList": [
    "\u003cocid:1\u003e"
  ],
  "compartmentId": "\u003cocid:2\u003e",
  "description": "synthetic operator control",
  "isFullyPreApproved": false,
  "numberOfApprovers": 1,
  "operatorControlName": "osok-mock-operator-control",
  "resourceType": "EXACC"
}`)
	createdState := ocimock.MustOCIResponseFixture[operatoraccesscontrolsdk.OperatorControl](t, `{
  "approverGroupsList": [
    "\u003cocid:1\u003e"
  ],
  "compartmentId": "\u003cocid:2\u003e",
  "description": "synthetic operator control",
  "id": "\u003cocid:3\u003e",
  "isFullyPreApproved": false,
  "lifecycleState": "CREATED",
  "numberOfApprovers": 1,
  "operatorControlName": "osok-mock-operator-control",
  "resourceType": "EXACC"
}`)
	updateRequest := ocimock.MustJSONFixture[operatoraccesscontrolsdk.UpdateOperatorControlDetails](t, `{
  "approverGroupsList": [
    "\u003cocid:1\u003e"
  ],
  "description": "synthetic operator control-updated",
  "isFullyPreApproved": false,
  "operatorControlName": "osok-mock-operator-control"
}`)
	updatedState := ocimock.MustOCIResponseFixture[operatoraccesscontrolsdk.OperatorControl](t, `{
  "approverGroupsList": [
    "\u003cocid:1\u003e"
  ],
  "compartmentId": "\u003cocid:2\u003e",
  "description": "synthetic operator control-updated",
  "id": "\u003cocid:3\u003e",
  "isFullyPreApproved": false,
  "lifecycleState": "CREATED",
  "numberOfApprovers": 1,
  "operatorControlName": "osok-mock-operator-control",
  "resourceType": "EXACC"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		operatoraccesscontrolsdk.OperatorControl,
		operatoraccesscontrolsdk.CreateOperatorControlDetails,
		operatoraccesscontrolsdk.UpdateOperatorControlDetails,
	]{
		CollectionPath:    "/20200630/operatorControls",
		ItemPath:          "/20200630/operatorControls/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ operatoraccesscontrolsdk.CreateOperatorControlDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ operatoraccesscontrolsdk.OperatorControl) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close OperatorControl OCI mock: %v", err)
		}
	})
	sdkClient := operatoraccesscontrolsdk.OperatorControlClient{BaseClient: session.BaseClient()}
	client := newOperatorControlServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*operatoraccesscontrolv1beta1.OperatorControl]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *operatoraccesscontrolv1beta1.OperatorControl) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "CREATED" ||
				!reflect.DeepEqual(current.Status.ApproverGroupsList, current.Spec.ApproverGroupsList) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.IsFullyPreApproved, current.Spec.IsFullyPreApproved) ||
				!reflect.DeepEqual(current.Status.NumberOfApprovers, current.Spec.NumberOfApprovers) ||
				!reflect.DeepEqual(current.Status.OperatorControlName, current.Spec.OperatorControlName) ||
				!reflect.DeepEqual(current.Status.ResourceType, current.Spec.ResourceType) {
				return fmt.Errorf("created OperatorControl status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *operatoraccesscontrolv1beta1.OperatorControl) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *operatoraccesscontrolv1beta1.OperatorControl) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "CREATED" ||
				!reflect.DeepEqual(current.Status.ApproverGroupsList, current.Spec.ApproverGroupsList) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.IsFullyPreApproved, current.Spec.IsFullyPreApproved) ||
				!reflect.DeepEqual(current.Status.OperatorControlName, current.Spec.OperatorControlName) {
				return fmt.Errorf("updated OperatorControl status = %+v", current.Status)
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

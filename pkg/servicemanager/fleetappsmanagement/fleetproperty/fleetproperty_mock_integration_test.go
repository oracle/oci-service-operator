/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package fleetproperty

import (
	"context"
	"fmt"
	"testing"

	fleetappsmanagementsdk "github.com/oracle/oci-go-sdk/v65/fleetappsmanagement"
	fleetappsmanagementv1beta1 "github.com/oracle/oci-service-operator/api/fleetappsmanagement/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationFleetPropertyCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &fleetappsmanagementv1beta1.FleetProperty{}
	ocimock.InitializeResource(resource, "mock-fleetproperty")
	resource.Spec = ocimock.MustJSONFixture[fleetappsmanagementv1beta1.FleetPropertySpec](t, `{
  "fleetId": "<ocid:1>",
  "value": "property-create",
  "propertyId": "<ocid:2>"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"value":"property-updated"}`)
	createRequest := ocimock.MustJSONFixture[fleetappsmanagementsdk.CreateFleetPropertyDetails](t, `{
  "value": "property-create",
  "propertyId": "<ocid:2>"
}`)
	updateRequest := ocimock.MustJSONFixture[fleetappsmanagementsdk.UpdateFleetPropertyDetails](t, `{
  "value": "property-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[fleetappsmanagementsdk.FleetProperty](t, `{
  "value": "property-create",
  "propertyId": "<ocid:2>",
  "id": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[fleetappsmanagementsdk.FleetProperty](t, `{
  "value": "property-updated",
  "propertyId": "<ocid:2>",
  "id": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[fleetappsmanagementsdk.FleetProperty, fleetappsmanagementsdk.CreateFleetPropertyDetails, fleetappsmanagementsdk.UpdateFleetPropertyDetails]{
		CollectionPath: "/20250228/fleets/<ocid:1>/fleetProperties", ItemPath: "/20250228/fleets/<ocid:1>/fleetProperties/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ fleetappsmanagementsdk.CreateFleetPropertyDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://fleetappsmanagement.mock.invalid", BasePath: "20250228", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := fleetappsmanagementsdk.FleetAppsManagementClient{BaseClient: session.BaseClient()}
	manager := &FleetPropertyServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newFleetPropertyRuntimeHooks(manager, sdkClient)
	client := wrapFleetPropertyGeneratedClient(hooks, defaultFleetPropertyServiceClient{ServiceClient: generatedruntime.NewServiceClient[*fleetappsmanagementv1beta1.FleetProperty](buildFleetPropertyGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*fleetappsmanagementv1beta1.FleetProperty]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *fleetappsmanagementv1beta1.FleetProperty) error {
			if current.Status.Value != resource.Spec.Value || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created FleetProperty status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *fleetappsmanagementv1beta1.FleetProperty) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *fleetappsmanagementv1beta1.FleetProperty) error {
			if current.Status.Value != current.Spec.Value {
				return fmt.Errorf("updated FleetProperty status = %+v", current.Status)
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

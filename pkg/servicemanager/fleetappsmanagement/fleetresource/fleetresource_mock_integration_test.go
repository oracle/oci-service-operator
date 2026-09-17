/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package fleetresource

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	fleetappsmanagementsdk "github.com/oracle/oci-go-sdk/v65/fleetappsmanagement"
	fleetappsmanagementv1beta1 "github.com/oracle/oci-service-operator/api/fleetappsmanagement/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationFleetResourceCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &fleetappsmanagementv1beta1.FleetResource{}
	ocimock.InitializeResource(resource, "mock-fleetresource")
	resource.Spec = ocimock.MustJSONFixture[fleetappsmanagementv1beta1.FleetResourceSpec](t, `{
  "fleetId": "<ocid:1>",
  "resourceId": "<ocid:2>",
  "tenancyId": "<ocid:3>",
  "compartmentId": "<ocid:4>",
  "resourceRegion": "us-ashburn-1",
  "resourceType": "INSTANCE"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"compartmentId":"<ocid:5>"}`)
	createRequest := ocimock.MustJSONFixture[fleetappsmanagementsdk.CreateFleetResourceDetails](t, `{
  "resourceId": "<ocid:2>",
  "tenancyId": "<ocid:3>",
  "compartmentId": "<ocid:4>",
  "resourceRegion": "us-ashburn-1",
  "resourceType": "INSTANCE"
}`)
	updateRequest := ocimock.MustJSONFixture[fleetappsmanagementsdk.UpdateFleetResourceDetails](t, `{
  "compartmentId": "<ocid:5>"
}`)
	createdState := ocimock.MustOCIResponseFixture[fleetappsmanagementsdk.FleetResource](t, `{
  "resourceId": "<ocid:2>",
  "tenancyId": "<ocid:3>",
  "compartmentId": "<ocid:4>",
  "resourceRegion": "us-ashburn-1",
  "resourceType": "INSTANCE",
  "id": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[fleetappsmanagementsdk.FleetResource](t, `{
  "resourceId": "<ocid:2>",
  "tenancyId": "<ocid:3>",
  "compartmentId": "<ocid:5>",
  "resourceRegion": "us-ashburn-1",
  "resourceType": "INSTANCE",
  "id": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[fleetappsmanagementsdk.FleetResource, fleetappsmanagementsdk.CreateFleetResourceDetails, fleetappsmanagementsdk.UpdateFleetResourceDetails]{
		CollectionPath: "/20250228/fleets/<ocid:1>/fleetResources", ItemPath: "/20250228/fleets/<ocid:1>/fleetResources/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ fleetappsmanagementsdk.CreateFleetResourceDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-create"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-delete"}},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/wr-create", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", fleetResourceWorkRequest("wr-create", fleetappsmanagementsdk.OperationTypeCreateFleetResource, fleetappsmanagementsdk.ActionTypeCreated))},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/wr-update", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", fleetResourceWorkRequest("wr-update", fleetappsmanagementsdk.OperationTypeUpdateFleetResource, fleetappsmanagementsdk.ActionTypeUpdated))},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/wr-delete", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", fleetResourceWorkRequest("wr-delete", fleetappsmanagementsdk.OperationTypeDeleteFleetResource, fleetappsmanagementsdk.ActionTypeDeleted))},
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
	workRequestClient := fleetappsmanagementsdk.FleetAppsManagementWorkRequestClient{BaseClient: session.BaseClient()}
	client := newFleetResourceServiceClientWithOCIClients(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient, workRequestClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*fleetappsmanagementv1beta1.FleetResource]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *fleetappsmanagementv1beta1.FleetResource) error {
			if current.Status.CompartmentId != resource.Spec.CompartmentId || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created FleetResource status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *fleetappsmanagementv1beta1.FleetResource) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *fleetappsmanagementv1beta1.FleetResource) error {
			if current.Status.CompartmentId != current.Spec.CompartmentId {
				return fmt.Errorf("updated FleetResource status = %+v", current.Status)
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

func fleetResourceWorkRequest(id string, operation fleetappsmanagementsdk.OperationTypeEnum, action fleetappsmanagementsdk.ActionTypeEnum) fleetappsmanagementsdk.WorkRequest {
	complete := float32(100)
	return fleetappsmanagementsdk.WorkRequest{
		OperationType: operation, Status: fleetappsmanagementsdk.OperationStatusSucceeded, Id: common.String(id),
		CompartmentId: common.String("<ocid:2>"), PercentComplete: &complete,
		Resources: []fleetappsmanagementsdk.WorkRequestResource{{EntityType: common.String("FleetResource"), ActionType: action, Identifier: common.String("resource-key")}},
	}
}

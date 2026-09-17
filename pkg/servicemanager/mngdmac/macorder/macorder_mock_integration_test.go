/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package macorder

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	mngdmacsdk "github.com/oracle/oci-go-sdk/v65/mngdmac"
	mngdmacv1beta1 "github.com/oracle/oci-service-operator/api/mngdmac/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationMacOrderWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &mngdmacv1beta1.MacOrder{}
	ocimock.InitializeResource(resource, "mock-macorder")
	resource.Spec = ocimock.MustJSONFixture[mngdmacv1beta1.MacOrderSpec](t, `{
  "commitmentTerm": "YEARS_3",
  "compartmentId": "<ocid:1>",
  "displayName": "mac-order-alpha",
  "ipRange": "10.0.0.0/24",
  "orderDescription": "Initial managed Mac order",
  "orderSize": 1,
  "shape": "M4_PRO_MAC_MINI_64GB_2TB"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "mac-order-updated"
}`)

	createRequest := ocimock.MustJSONFixture[mngdmacsdk.CreateMacOrderDetails](t, `{
  "commitmentTerm": "YEARS_3",
  "compartmentId": "<ocid:1>",
  "displayName": "mac-order-alpha",
  "ipRange": "10.0.0.0/24",
  "orderDescription": "Initial managed Mac order",
  "orderSize": 1,
  "shape": "M4_PRO_MAC_MINI_64GB_2TB"
}`)
	updateRequest := ocimock.MustJSONFixture[mngdmacsdk.UpdateMacOrderDetails](t, `{
  "displayName": "mac-order-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[mngdmacsdk.MacOrder](t, `{
  "cancelReason": null,
  "commitmentTerm": "YEARS_3",
  "compartmentId": "<ocid:1>",
  "displayName": "mac-order-alpha",
  "id": "<ocid:3>",
  "ipRange": "10.0.0.0/24",
  "isDocusigned": true,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "orderDescription": "Initial managed Mac order",
  "orderSize": 1,
  "orderStatus": "SUBMITTED",
  "shape": "M4_PRO_MAC_MINI_64GB_2TB",
  "timeBillingEnded": null,
  "timeBillingStarted": null,
  "timeCanceled": null,
  "timeCreated": null,
  "timeUpdated": null
}`)
	updatedState := ocimock.MustOCIResponseFixture[mngdmacsdk.MacOrder](t, `{
  "cancelReason": null,
  "commitmentTerm": "YEARS_3",
  "compartmentId": "<ocid:1>",
  "displayName": "mac-order-updated",
  "id": "<ocid:3>",
  "ipRange": "10.0.0.0/24",
  "isDocusigned": true,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "orderDescription": "Initial managed Mac order",
  "orderSize": 1,
  "orderStatus": "SUBMITTED",
  "shape": "M4_PRO_MAC_MINI_64GB_2TB",
  "timeBillingEnded": null,
  "timeBillingStarted": null,
  "timeCanceled": null,
  "timeCreated": null,
  "timeUpdated": null
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[mngdmacsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_MAC_ORDER",
  "percentComplete": null,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "MacOrder",
      "entityUri": null,
      "identifier": "<ocid:3>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[mngdmacsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "wr-update",
  "operationType": "UPDATE_MAC_ORDER",
  "percentComplete": null,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "MacOrder",
      "entityUri": null,
      "identifier": "<ocid:3>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[mngdmacsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "CANCEL_MAC_ORDER",
  "percentComplete": null,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "MacOrder",
      "entityUri": null,
      "identifier": "<ocid:3>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": null,
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[mngdmacsdk.MacOrder, mngdmacsdk.CreateMacOrderDetails, mngdmacsdk.UpdateMacOrderDetails]{
		CollectionPath: "/20250320/macOrders", ItemPath: "/20250320/macOrders/<ocid:3>",
		DeletePath: "/20250320/macOrders/<ocid:3>/actions/cancel", DeleteMethod: http.MethodPost,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		ValidateCreate: func(request ocimock.Request, _ mngdmacsdk.CreateMacOrderDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		ValidateDelete: func(request ocimock.Request, _ mngdmacsdk.MacOrder) error {
			if string(request.Body) != "{}" {
				return fmt.Errorf("cancel MacOrder body = %s, want {}", request.Body)
			}
			return nil
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20250320/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20250320/workRequests/wr-update", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20250320/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://mngdmac.us-ashburn-1.oci.oraclecloud.com", BasePath: "20250320", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := mngdmacsdk.MacOrderClient{BaseClient: session.BaseClient()}
	client := newMacOrderServiceClientWithOCIClient(log, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*mngdmacv1beta1.MacOrder]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *mngdmacv1beta1.MacOrder) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created MacOrder status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *mngdmacv1beta1.MacOrder) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *mngdmacv1beta1.MacOrder) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated MacOrder status = %+v", current.Status)
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

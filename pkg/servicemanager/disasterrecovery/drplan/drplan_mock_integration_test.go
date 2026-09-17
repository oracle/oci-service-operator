/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package drplan

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	disasterrecoverysdk "github.com/oracle/oci-go-sdk/v65/disasterrecovery"
	disasterrecoveryv1beta1 "github.com/oracle/oci-service-operator/api/disasterrecovery/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationDrPlanWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &disasterrecoveryv1beta1.DrPlan{}
	ocimock.InitializeResource(resource, "mock-drplan")
	resource.Spec = ocimock.MustJSONFixture[disasterrecoveryv1beta1.DrPlanSpec](t, `{
  "displayName": "drplan-sample",
  "drProtectionGroupId": "<ocid:1>",
  "type": "SWITCHOVER"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "drplan-sample-updated"
}`)

	createRequest := ocimock.MustJSONFixture[disasterrecoverysdk.CreateDrPlanDetails](t, `{
  "displayName": "drplan-sample",
  "drProtectionGroupId": "<ocid:1>",
  "type": "SWITCHOVER"
}`)
	updateRequest := ocimock.MustJSONFixture[disasterrecoverysdk.UpdateDrPlanDetails](t, `{
  "displayName": "drplan-sample-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[disasterrecoverysdk.DrPlan](t, `{
  "displayName": "drplan-sample",
  "drProtectionGroupId": "<ocid:1>",
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "type": "SWITCHOVER"
}`)
	updatedState := ocimock.MustOCIResponseFixture[disasterrecoverysdk.DrPlan](t, `{
  "displayName": "drplan-sample-updated",
  "drProtectionGroupId": "<ocid:1>",
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "type": "SWITCHOVER"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[disasterrecoverysdk.WorkRequest](t, `{
  "id": "<ocid:2>",
  "operationType": "CREATE_DR_PLAN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "DrPlan",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[disasterrecoverysdk.WorkRequest](t, `{
  "id": "wr-update",
  "operationType": "UPDATE_DR_PLAN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "DrPlan",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[disasterrecoverysdk.DrPlan, disasterrecoverysdk.CreateDrPlanDetails, disasterrecoverysdk.UpdateDrPlanDetails]{
		CollectionPath:     "/20220125/drPlans",
		ItemPath:           "/20220125/drPlans/<ocid:3>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		ListShape:          ocimock.ListShapeItems,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		DeleteEndsNotFound: false, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 204, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: nil,
		ValidateCreate: func(request ocimock.Request, _ disasterrecoverysdk.CreateDrPlanDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20220125/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest),
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20220125/workRequests/wr-update", MinimumCalls: 1,
				Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://disaster-recovery.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220125", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := disasterrecoverysdk.DisasterRecoveryClient{BaseClient: session.BaseClient()}
	hooks := newDrPlanDefaultRuntimeHooks(sdkClient)
	applyDrPlanRuntimeHooks(&hooks, sdkClient, nil)
	manager := &DrPlanServiceManager{Log: log}
	client := wrapDrPlanGeneratedClient(hooks, defaultDrPlanServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*disasterrecoveryv1beta1.DrPlan](buildDrPlanGeneratedRuntimeConfig(manager, hooks)),
	})

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*disasterrecoveryv1beta1.DrPlan]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate},
		Resource:            resource,
		Client:              client,
		CreateContext:       generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *disasterrecoveryv1beta1.DrPlan) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DrPlan status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *disasterrecoveryv1beta1.DrPlan) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *disasterrecoveryv1beta1.DrPlan) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DrPlan status = %+v", current.Status)
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

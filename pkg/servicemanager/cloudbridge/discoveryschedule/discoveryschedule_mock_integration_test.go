/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package discoveryschedule

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/cloudbridge"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/cloudbridge/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationDiscoveryScheduleCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.DiscoverySchedule](t, `
{
  "metadata": {"name": "mock-discoveryschedule", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "executionRecurrences": "FREQ=DAILY;INTERVAL=1"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-discoveryschedule")
	resource.Status = apiv1beta1.DiscoveryScheduleStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateDiscoveryScheduleDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "executionRecurrences": "FREQ=DAILY;INTERVAL=1"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateDiscoveryScheduleDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.DiscoverySchedule](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "executionRecurrences": "FREQ=DAILY;INTERVAL=1",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.DiscoverySchedule](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-updated",
  "executionRecurrences": "FREQ=DAILY;INTERVAL=1",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)

	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.DiscoverySchedule, sdksvc.CreateDiscoveryScheduleDetails, sdksvc.UpdateDiscoveryScheduleDetails]{
		CollectionPath: "/20220509/discoverySchedules", ItemPath: "/20220509/discoverySchedules/<ocid:2>",
		CreatePath: "/20220509/discoverySchedules", CreateMethod: http.MethodPost,
		UpdatePath: "/20220509/discoverySchedules/<ocid:2>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220509/discoverySchedules/<ocid:2>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateDiscoveryScheduleDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateDiscoveryScheduleDetails],
		UpdatedState:      &updatedState,
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateDiscoveryScheduleDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudbridge.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220509", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.DiscoveryClient{BaseClient: session.BaseClient()}
	manager := &DiscoveryScheduleServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDiscoveryScheduleRuntimeHooks(manager, sdkClient)
	client := wrapDiscoveryScheduleGeneratedClient(hooks, defaultDiscoveryScheduleServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.DiscoverySchedule](buildDiscoveryScheduleGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.DiscoverySchedule]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.DiscoverySchedule) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DiscoverySchedule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.DiscoverySchedule) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.DiscoverySchedule) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DiscoverySchedule status = %+v", current.Status)
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

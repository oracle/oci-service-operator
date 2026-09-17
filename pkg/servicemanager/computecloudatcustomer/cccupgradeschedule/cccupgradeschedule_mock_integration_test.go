/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package cccupgradeschedule

import (
	"context"
	"fmt"
	computecloudatcustomersdk "github.com/oracle/oci-go-sdk/v65/computecloudatcustomer"
	computecloudatcustomerv1beta1 "github.com/oracle/oci-service-operator/api/computecloudatcustomer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationCccUpgradeScheduleEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &computecloudatcustomerv1beta1.CccUpgradeSchedule{}
	ocimock.InitializeResource(resource, "mock-cccupgradeschedule")
	resource.Spec = ocimock.MustJSONFixture[computecloudatcustomerv1beta1.CccUpgradeScheduleSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-ccc-schedule",
  "events": [
    {
      "description": "OSOK mock maintenance window",
      "scheduleEventDuration": "PT49H",
      "scheduleEventRecurrences": "FREQ=MONTHLY",
      "timeStart": "2026-09-05T00:00:00Z"
    }
  ],
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "events": [
    {
      "description": "OSOK mock maintenance window",
      "scheduleEventDuration": "PT49H",
      "scheduleEventRecurrences": "FREQ=MONTHLY",
      "timeStart": "2026-09-05T00:00:00Z"
    }
  ],
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[computecloudatcustomersdk.CreateCccUpgradeScheduleDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-ccc-schedule",
  "events": [
    {
      "description": "OSOK mock maintenance window",
      "scheduleEventDuration": "PT49H",
      "scheduleEventRecurrences": "FREQ=MONTHLY",
      "timeStart": "2026-09-05T00:00:00Z"
    }
  ],
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[computecloudatcustomersdk.CccUpgradeSchedule](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T21:11:39.772Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-ccc-schedule",
  "events": [
    {
      "description": "OSOK mock maintenance window",
      "name": "26a7217c-6b89-4077-9158-e06a7b8f31aa",
      "scheduleEventDuration": "PT49H",
      "scheduleEventRecurrences": "FREQ=MONTHLY",
      "timeStart": "2026-09-05T00:00:00.000Z"
    }
  ],
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "infrastructureIds": null,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-03T21:11:39.919Z",
  "timeUpdated": "2026-09-03T21:11:39.919Z"
}`)
	updateRequest := ocimock.MustJSONFixture[computecloudatcustomersdk.UpdateCccUpgradeScheduleDetails](t, `{
  "description": "recorded update",
  "events": [
    {
      "description": "OSOK mock maintenance window",
      "scheduleEventDuration": "PT49H",
      "scheduleEventRecurrences": "FREQ=MONTHLY",
      "timeStart": "2026-09-05T00:00:00Z"
    }
  ],
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[computecloudatcustomersdk.CccUpgradeSchedule](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T21:11:39.772Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-ccc-schedule",
  "events": [
    {
      "description": "OSOK mock maintenance window",
      "name": "abc678cb-a487-4714-b0ae-f7ca6aea0066",
      "scheduleEventDuration": "PT49H",
      "scheduleEventRecurrences": "FREQ=MONTHLY",
      "timeStart": "2026-09-05T00:00:00.000Z"
    }
  ],
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "infrastructureIds": null,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-03T21:11:39.919Z",
  "timeUpdated": "2026-09-03T21:11:41.487Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[computecloudatcustomersdk.CccUpgradeSchedule](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T21:11:39.772Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-ccc-schedule",
  "events": [
    {
      "description": "OSOK mock maintenance window",
      "name": "abc678cb-a487-4714-b0ae-f7ca6aea0066",
      "scheduleEventDuration": "PT49H",
      "scheduleEventRecurrences": "FREQ=MONTHLY",
      "timeStart": "2026-09-05T00:00:00.000Z"
    }
  ],
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "infrastructureIds": null,
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "systemTags": {},
  "timeCreated": "2026-09-03T21:11:39.919Z",
  "timeUpdated": "2026-09-03T21:11:42.959Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		computecloudatcustomersdk.CccUpgradeSchedule,
		computecloudatcustomersdk.CreateCccUpgradeScheduleDetails,
		computecloudatcustomersdk.UpdateCccUpgradeScheduleDetails,
	]{
		CollectionPath:    "/20221208/cccUpgradeSchedules",
		ItemPath:          "/20221208/cccUpgradeSchedules/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ computecloudatcustomersdk.CreateCccUpgradeScheduleDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ computecloudatcustomersdk.CccUpgradeSchedule) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20221208", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close CccUpgradeSchedule OCI mock: %v", err)
		}
	})
	sdkClient := computecloudatcustomersdk.ComputeCloudAtCustomerClient{BaseClient: session.BaseClient()}
	manager := &CccUpgradeScheduleServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newCccUpgradeScheduleRuntimeHooks(manager, sdkClient)
	client := wrapCccUpgradeScheduleGeneratedClient(hooks, defaultCccUpgradeScheduleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*computecloudatcustomerv1beta1.CccUpgradeSchedule](buildCccUpgradeScheduleGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*computecloudatcustomerv1beta1.CccUpgradeSchedule]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *computecloudatcustomerv1beta1.CccUpgradeSchedule) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.Events, current.Spec.Events) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("created CccUpgradeSchedule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *computecloudatcustomerv1beta1.CccUpgradeSchedule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *computecloudatcustomerv1beta1.CccUpgradeSchedule) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Events, current.Spec.Events) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated CccUpgradeSchedule status = %+v", current.Status)
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

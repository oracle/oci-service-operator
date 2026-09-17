/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package schedule

import (
	"context"
	"fmt"
	resourceschedulersdk "github.com/oracle/oci-go-sdk/v65/resourcescheduler"
	resourceschedulerv1beta1 "github.com/oracle/oci-service-operator/api/resourcescheduler/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationScheduleEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &resourceschedulerv1beta1.Schedule{}
	ocimock.InitializeResource(resource, "mock-schedule")
	resource.Spec = ocimock.MustJSONFixture[resourceschedulerv1beta1.ScheduleSpec](t, `{
  "action": "START_RESOURCE",
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-resource-schedule-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "recurrenceDetails": "FREQ=DAILY;INTERVAL=1",
  "recurrenceType": "ICAL",
  "resourceFilters": [],
  "resources": [
    {
      "id": "\u003cocid:2\u003e",
      "parameters": []
    }
  ],
  "timeStarts": "2099-01-01T00:00:00Z"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  },
  "resourceFilters": [],
  "resources": [
    {
      "id": "\u003cocid:2\u003e",
      "parameters": []
    }
  ]
}`)
	createRequest := ocimock.MustJSONFixture[resourceschedulersdk.CreateScheduleDetails](t, `{
  "action": "START_RESOURCE",
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-resource-schedule-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "recurrenceDetails": "FREQ=DAILY;INTERVAL=1",
  "recurrenceType": "ICAL",
  "resourceFilters": [],
  "resources": [
    {
      "id": "\u003cocid:2\u003e",
      "parameters": []
    }
  ],
  "timeStarts": "2099-01-01T00:00:00Z"
}`)
	createdState := ocimock.MustOCIResponseFixture[resourceschedulersdk.Schedule](t, `{
  "action": "START_RESOURCE",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T22:53:38.276Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-resource-schedule-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:4>",
  "lastRunStatus": null,
  "lifecycleState": "ACTIVE",
  "localTimeZone": "UTC",
  "previousState": {
    "action": "START_RESOURCE",
    "compartmentId": "<ocid:1>",
    "definedTags": null,
    "description": "recorded create",
    "displayName": "osok-mock-resource-schedule-v1",
    "freeformTags": null,
    "id": "<ocid:4>",
    "lastRunStatus": null,
    "lifecycleState": "ACTIVE",
    "localTimeZone": "UTC",
    "previousState": null,
    "recurrenceDetails": "FREQ=DAILY;INTERVAL=1",
    "recurrenceType": "ICAL",
    "resourceFilters": [],
    "resources": [
      {
        "id": "<ocid:2>",
        "metadata": null,
        "parameters": []
      }
    ],
    "systemTags": null,
    "timeCreated": "2026-09-01T22:53:38.377Z",
    "timeEnds": null,
    "timeLastRun": null,
    "timeNextRun": "2099-01-01T00:00:00.000Z",
    "timeStarts": "2099-01-01T00:00:00.000Z",
    "timeUpdated": "2026-09-01T22:53:51.679Z"
  },
  "recurrenceDetails": "FREQ=DAILY;INTERVAL=1",
  "recurrenceType": "ICAL",
  "resourceFilters": [],
  "resources": [
    {
      "id": "<ocid:2>",
      "metadata": null,
      "parameters": []
    }
  ],
  "systemTags": {},
  "timeCreated": "2026-09-01T22:53:38.377Z",
  "timeEnds": null,
  "timeLastRun": null,
  "timeNextRun": "2099-01-01T00:00:00.000Z",
  "timeStarts": "2099-01-01T00:00:00.000Z",
  "timeUpdated": "2026-09-01T22:53:54.793Z"
}`)
	updateRequest := ocimock.MustJSONFixture[resourceschedulersdk.UpdateScheduleDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  },
  "resourceFilters": [],
  "resources": [
    {
      "id": "\u003cocid:2\u003e",
      "parameters": []
    }
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[resourceschedulersdk.Schedule](t, `{
  "action": "START_RESOURCE",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T22:53:38.276Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-resource-schedule-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "lastRunStatus": null,
  "lifecycleState": "ACTIVE",
  "localTimeZone": "UTC",
  "previousState": {
    "action": "START_RESOURCE",
    "compartmentId": "<ocid:1>",
    "definedTags": null,
    "description": "recorded create",
    "displayName": "osok-mock-resource-schedule-v1",
    "freeformTags": null,
    "id": "<ocid:4>",
    "lastRunStatus": null,
    "lifecycleState": "ACTIVE",
    "localTimeZone": "UTC",
    "previousState": null,
    "recurrenceDetails": "FREQ=DAILY;INTERVAL=1",
    "recurrenceType": "ICAL",
    "resourceFilters": [],
    "resources": [
      {
        "id": "<ocid:2>",
        "metadata": null,
        "parameters": []
      }
    ],
    "systemTags": null,
    "timeCreated": "2026-09-01T22:53:38.377Z",
    "timeEnds": null,
    "timeLastRun": null,
    "timeNextRun": "2099-01-01T00:00:00.000Z",
    "timeStarts": "2099-01-01T00:00:00.000Z",
    "timeUpdated": "2026-09-01T22:53:54.793Z"
  },
  "recurrenceDetails": "FREQ=DAILY;INTERVAL=1",
  "recurrenceType": "ICAL",
  "resourceFilters": [],
  "resources": [
    {
      "id": "<ocid:2>",
      "metadata": null,
      "parameters": []
    }
  ],
  "systemTags": {},
  "timeCreated": "2026-09-01T22:53:38.377Z",
  "timeEnds": null,
  "timeLastRun": null,
  "timeNextRun": "2099-01-01T00:00:00.000Z",
  "timeStarts": "2099-01-01T00:00:00.000Z",
  "timeUpdated": "2026-09-01T22:53:55.685Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[resourceschedulersdk.Schedule](t, `{
  "action": "START_RESOURCE",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T22:53:38.276Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-resource-schedule-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "lastRunStatus": null,
  "lifecycleState": "DELETED",
  "localTimeZone": "UTC",
  "previousState": {
    "action": "START_RESOURCE",
    "compartmentId": "<ocid:1>",
    "definedTags": null,
    "description": "recorded create",
    "displayName": "osok-mock-resource-schedule-v1",
    "freeformTags": null,
    "id": "<ocid:4>",
    "lastRunStatus": null,
    "lifecycleState": "ACTIVE",
    "localTimeZone": "UTC",
    "previousState": null,
    "recurrenceDetails": "FREQ=DAILY;INTERVAL=1",
    "recurrenceType": "ICAL",
    "resourceFilters": [],
    "resources": [
      {
        "id": "<ocid:2>",
        "metadata": null,
        "parameters": []
      }
    ],
    "systemTags": null,
    "timeCreated": "2026-09-01T22:53:38.377Z",
    "timeEnds": null,
    "timeLastRun": null,
    "timeNextRun": "2099-01-01T00:00:00.000Z",
    "timeStarts": "2099-01-01T00:00:00.000Z",
    "timeUpdated": "2026-09-01T22:53:54.793Z"
  },
  "recurrenceDetails": "FREQ=DAILY;INTERVAL=1",
  "recurrenceType": "ICAL",
  "resourceFilters": [],
  "resources": [
    {
      "id": "<ocid:2>",
      "metadata": null,
      "parameters": []
    }
  ],
  "systemTags": {},
  "timeCreated": "2026-09-01T22:53:38.377Z",
  "timeEnds": null,
  "timeLastRun": null,
  "timeNextRun": "2099-01-01T00:00:00.000Z",
  "timeStarts": "2099-01-01T00:00:00.000Z",
  "timeUpdated": "2026-09-01T22:53:56.784Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		resourceschedulersdk.Schedule,
		resourceschedulersdk.CreateScheduleDetails,
		resourceschedulersdk.UpdateScheduleDetails,
	]{
		CollectionPath:    "/20240430/schedules",
		ItemPath:          "/20240430/schedules/<ocid:4>",
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
		UpdateStatus:      202,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ resourceschedulersdk.CreateScheduleDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ resourceschedulersdk.Schedule) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20240430", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Schedule OCI mock: %v", err)
		}
	})
	sdkClient := resourceschedulersdk.ScheduleClient{BaseClient: session.BaseClient()}
	hooks := newScheduleRuntimeHooksWithOCIClient(sdkClient)
	applyScheduleRuntimeHooks(&hooks)
	manager := &ScheduleServiceManager{Log: loggerutil.OSOKLogger{}}
	client := wrapScheduleGeneratedClient(hooks, defaultScheduleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*resourceschedulerv1beta1.Schedule](buildScheduleGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*resourceschedulerv1beta1.Schedule]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *resourceschedulerv1beta1.Schedule) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Action, current.Spec.Action) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.RecurrenceDetails, current.Spec.RecurrenceDetails) ||
				!reflect.DeepEqual(current.Status.RecurrenceType, current.Spec.RecurrenceType) ||
				!reflect.DeepEqual(current.Status.ResourceFilters, current.Spec.ResourceFilters) ||
				!reflect.DeepEqual(current.Status.Resources, current.Spec.Resources) ||
				!reflect.DeepEqual(current.Status.TimeStarts, current.Spec.TimeStarts) {
				return fmt.Errorf("created Schedule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *resourceschedulerv1beta1.Schedule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *resourceschedulerv1beta1.Schedule) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ResourceFilters, current.Spec.ResourceFilters) ||
				!reflect.DeepEqual(current.Status.Resources, current.Spec.Resources) {
				return fmt.Errorf("updated Schedule status = %+v", current.Status)
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

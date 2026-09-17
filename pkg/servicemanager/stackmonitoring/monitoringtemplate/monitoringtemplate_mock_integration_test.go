/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package monitoringtemplate

import (
	"context"
	"fmt"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationMonitoringTemplateLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := testMonitoringTemplate()
	ocimock.InitializeResource(resource, "mock-monitoringtemplate")
	resource.Spec = ocimock.MustJSONFixture[stackmonitoringv1beta1.MonitoringTemplateSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "managed by osok",
  "destinations": [
    "\u003cocid:2\u003e"
  ],
  "displayName": "template",
  "freeformTags": {
    "owner": "osok"
  },
  "isAlarmsEnabled": false,
  "isSplitNotificationEnabled": false,
  "members": [
    {
      "id": "\u003cocid:3\u003e",
      "type": "RESOURCE_INSTANCE"
    }
  ],
  "messageFormat": "RAW",
  "repeatNotificationDuration": "PT4H"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "managed by osok-updated"
}`)
	createRequest := ocimock.MustJSONFixture[stackmonitoringsdk.CreateMonitoringTemplateDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "managed by osok",
  "destinations": [
    "\u003cocid:2\u003e"
  ],
  "displayName": "template",
  "freeformTags": {
    "owner": "osok"
  },
  "isAlarmsEnabled": false,
  "isSplitNotificationEnabled": false,
  "members": [
    {
      "id": "\u003cocid:3\u003e",
      "type": "RESOURCE_INSTANCE"
    }
  ],
  "messageFormat": "RAW",
  "repeatNotificationDuration": "PT4H"
}`)
	createdState := ocimock.MustOCIResponseFixture[stackmonitoringsdk.MonitoringTemplate](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "managed by osok",
  "destinations": [
    "\u003cocid:2\u003e"
  ],
  "displayName": "template",
  "freeformTags": {
    "owner": "osok"
  },
  "id": "\u003cocid:4\u003e",
  "isAlarmsEnabled": false,
  "isSplitNotificationEnabled": false,
  "lifecycleState": "ACTIVE",
  "members": [
    {
      "id": "\u003cocid:3\u003e",
      "type": "RESOURCE_INSTANCE"
    }
  ],
  "messageFormat": "RAW",
  "repeatNotificationDuration": "PT4H"
}`)
	updateRequest := ocimock.MustJSONFixture[stackmonitoringsdk.UpdateMonitoringTemplateDetails](t, `{
  "description": "managed by osok-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[stackmonitoringsdk.MonitoringTemplate](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "managed by osok-updated",
  "destinations": [
    "\u003cocid:2\u003e"
  ],
  "displayName": "template",
  "freeformTags": {
    "owner": "osok"
  },
  "id": "\u003cocid:4\u003e",
  "isAlarmsEnabled": false,
  "isSplitNotificationEnabled": false,
  "lifecycleState": "ACTIVE",
  "members": [
    {
      "id": "\u003cocid:3\u003e",
      "type": "RESOURCE_INSTANCE"
    }
  ],
  "messageFormat": "RAW",
  "repeatNotificationDuration": "PT4H"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		stackmonitoringsdk.MonitoringTemplate,
		stackmonitoringsdk.CreateMonitoringTemplateDetails,
		stackmonitoringsdk.UpdateMonitoringTemplateDetails,
	]{
		CollectionPath:    "/20210330/monitoringTemplates",
		ItemPath:          "/20210330/monitoringTemplates/<ocid:4>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ stackmonitoringsdk.CreateMonitoringTemplateDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ stackmonitoringsdk.MonitoringTemplate) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20210330", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close MonitoringTemplate OCI mock: %v", err)
		}
	})
	sdkClient := stackmonitoringsdk.StackMonitoringClient{BaseClient: session.BaseClient()}
	manager := &MonitoringTemplateServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newMonitoringTemplateRuntimeHooks(manager, sdkClient)
	client := wrapMonitoringTemplateGeneratedClient(hooks, defaultMonitoringTemplateServiceClient{ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.MonitoringTemplate](buildMonitoringTemplateGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*stackmonitoringv1beta1.MonitoringTemplate]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *stackmonitoringv1beta1.MonitoringTemplate) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Destinations, current.Spec.Destinations) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsAlarmsEnabled, current.Spec.IsAlarmsEnabled) ||
				!reflect.DeepEqual(current.Status.IsSplitNotificationEnabled, current.Spec.IsSplitNotificationEnabled) ||
				!reflect.DeepEqual(current.Status.Members, current.Spec.Members) ||
				!reflect.DeepEqual(current.Status.MessageFormat, current.Spec.MessageFormat) ||
				!reflect.DeepEqual(current.Status.RepeatNotificationDuration, current.Spec.RepeatNotificationDuration) {
				return fmt.Errorf("created MonitoringTemplate status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *stackmonitoringv1beta1.MonitoringTemplate) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *stackmonitoringv1beta1.MonitoringTemplate) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated MonitoringTemplate status = %+v", current.Status)
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

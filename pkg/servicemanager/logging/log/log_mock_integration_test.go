/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package log

import (
	"context"
	"fmt"
	loggingsdk "github.com/oracle/oci-go-sdk/v65/logging"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationLogCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &loggingv1beta1.Log{}
	ocimock.InitializeResource(resource, "mock-log")
	resource.Spec = ocimock.MustJSONFixture[loggingv1beta1.LogSpec](t, `{
  "displayName": "osok-mock-custom-log-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isEnabled": true,
  "logType": "CUSTOM",
  "retentionDuration": 30
}`)
	resource.Spec.LogGroupId = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "osok-mock": "update"
  },
  "isEnabled": false,
  "retentionDuration": 60
}`)
	createRequest := ocimock.MustJSONFixture[loggingsdk.CreateLogDetails](t, `{
  "displayName": "osok-mock-custom-log-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isEnabled": true,
  "logType": "CUSTOM",
  "retentionDuration": 30
}`)
	createdState := ocimock.MustOCIResponseFixture[loggingsdk.Log](t, `{
  "compartmentId": "<ocid:3>",
  "configuration": null,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T02:56:33.218Z"
    }
  },
  "displayName": "osok-mock-custom-log-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:4>",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "logGroupId": "<ocid:1>",
  "logType": "CUSTOM",
  "retentionDuration": 30,
  "systemTags": {},
  "tenancyId": "<ocid:5>",
  "timeCreated": "2026-09-01T02:56:33.293Z",
  "timeLastModified": "2026-09-01T02:56:33.293Z"
}`)
	updateRequest := ocimock.MustJSONFixture[loggingsdk.UpdateLogDetails](t, `{
  "freeformTags": {
    "osok-mock": "update"
  },
  "isEnabled": false,
  "retentionDuration": 60
}`)
	updatedState := ocimock.MustOCIResponseFixture[loggingsdk.Log](t, `{
  "compartmentId": "<ocid:3>",
  "configuration": null,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T02:56:33.218Z"
    }
  },
  "displayName": "osok-mock-custom-log-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "isEnabled": false,
  "lifecycleState": "INACTIVE",
  "logGroupId": "<ocid:1>",
  "logType": "CUSTOM",
  "retentionDuration": 60,
  "systemTags": {},
  "tenancyId": "<ocid:5>",
  "timeCreated": "2026-09-01T02:56:33.293Z",
  "timeLastModified": "2026-09-01T02:56:33.926Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		loggingsdk.Log,
		loggingsdk.CreateLogDetails,
		loggingsdk.UpdateLogDetails,
	]{
		CollectionPath:    "/20200531/logGroups/<ocid:1>/logs",
		ItemPath:          "/20200531/logGroups/<ocid:1>/logs/<ocid:4>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeArray,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      202,
		UpdateStatus:      202,
		DeleteStatus:      202,
		NotFoundCode:      "NotAuthorizedOrNotFound",
		ValidateCreate: func(request ocimock.Request, _ loggingsdk.CreateLogDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ loggingsdk.Log) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200531", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Log OCI mock: %v", err)
		}
	})
	sdkClient := loggingsdk.LoggingManagementClient{BaseClient: session.BaseClient()}
	client := newMockLogClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loggingv1beta1.Log]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loggingv1beta1.Log) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsEnabled, current.Spec.IsEnabled) ||
				!reflect.DeepEqual(current.Status.LogType, current.Spec.LogType) ||
				!reflect.DeepEqual(current.Status.RetentionDuration, current.Spec.RetentionDuration) {
				return fmt.Errorf("created Log status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loggingv1beta1.Log) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loggingv1beta1.Log) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "INACTIVE" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsEnabled, current.Spec.IsEnabled) ||
				!reflect.DeepEqual(current.Status.RetentionDuration, current.Spec.RetentionDuration) {
				return fmt.Errorf("updated Log status = %+v", current.Status)
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

/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package loganalyticsloggroup

import (
	"context"
	"fmt"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationLogAnalyticsLogGroupCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &loganalyticsv1beta1.LogAnalyticsLogGroup{}
	ocimock.InitializeResource(resource, "mock-loganalyticsloggroup")
	resource.Namespace = "<binding:loganalytics-namespace>"
	resource.Spec = ocimock.MustJSONFixture[loganalyticsv1beta1.LogAnalyticsLogGroupSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-log-analytics-group-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[loganalyticssdk.CreateLogAnalyticsLogGroupDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-log-analytics-group-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsLogGroup](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:50:18.336Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-log-analytics-group-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "timeCreated": "2026-09-01T21:50:19.024Z",
  "timeUpdated": "2026-09-01T21:50:19.024Z"
}`)
	createdReadStates := []loganalyticssdk.LogAnalyticsLogGroup{
		ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsLogGroup](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:50:18.336Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-log-analytics-group-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "timeCreated": "2026-09-01T21:50:19.024Z",
  "timeUpdated": "2026-09-01T21:50:19.024Z"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[loganalyticssdk.UpdateLogAnalyticsLogGroupDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsLogGroup](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:50:18.336Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-log-analytics-group-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "timeCreated": "2026-09-01T21:50:19.024Z",
  "timeUpdated": "2026-09-01T21:50:25.095Z"
}`)
	updatedReadStates := []loganalyticssdk.LogAnalyticsLogGroup{
		ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsLogGroup](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T21:50:18.336Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-log-analytics-group-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "timeCreated": "2026-09-01T21:50:19.024Z",
  "timeUpdated": "2026-09-01T21:50:25.095Z"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		loganalyticssdk.LogAnalyticsLogGroup,
		loganalyticssdk.CreateLogAnalyticsLogGroupDetails,
		loganalyticssdk.UpdateLogAnalyticsLogGroupDetails,
	]{
		CollectionPath:     "/20200601/namespaces/<binding:loganalytics-namespace>/logAnalyticsLogGroups",
		ItemPath:           "/20200601/namespaces/<binding:loganalytics-namespace>/logAnalyticsLogGroups/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       200,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotAuthorizedOrNotFound",
		ValidateCreate: func(request ocimock.Request, _ loganalyticssdk.CreateLogAnalyticsLogGroupDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ loganalyticssdk.LogAnalyticsLogGroup) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200601", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close LogAnalyticsLogGroup OCI mock: %v", err)
		}
	})
	sdkClient := loganalyticssdk.LogAnalyticsClient{BaseClient: session.BaseClient()}
	client := newLogAnalyticsLogGroupServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loganalyticsv1beta1.LogAnalyticsLogGroup]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loganalyticsv1beta1.LogAnalyticsLogGroup) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("created LogAnalyticsLogGroup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loganalyticsv1beta1.LogAnalyticsLogGroup) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loganalyticsv1beta1.LogAnalyticsLogGroup) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated LogAnalyticsLogGroup status = %+v", current.Status)
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

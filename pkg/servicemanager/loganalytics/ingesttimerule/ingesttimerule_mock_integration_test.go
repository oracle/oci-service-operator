/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package ingesttimerule

import (
	"context"
	"fmt"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"net/http"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationIngestTimeRuleCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &loganalyticsv1beta1.IngestTimeRule{}
	ocimock.InitializeResource(resource, "mock-ingesttimerule")
	resource.Spec = ocimock.MustJSONFixture[loganalyticsv1beta1.IngestTimeRuleSpec](t, `{
  "actions": [
    {
      "compartmentId": "\u003cocid:2\u003e",
      "dimensions": [
        "SOURCE_NAME"
      ],
      "metricName": "matched_records",
      "namespace": "osok_mock",
      "resourceGroup": "integration",
      "type": "METRIC_EXTRACTION"
    }
  ],
  "compartmentId": "\u003cocid:2\u003e",
  "conditions": {
    "fieldName": "mtag",
    "fieldOperator": "EQUAL",
    "fieldValue": "osok-mock",
    "kind": "FIELD"
  },
  "description": "OSOK recorded ingest-time rule",
  "displayName": "osok-mock-ingest-time-rule",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	resource.Spec.IsEnabled = true
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "\u003credacted\u003e",
      "CreatedOn": "2026-09-02T21:18:14.486Z"
    }
  },
  "description": "OSOK recorded ingest-time rule updated",
  "displayName": "osok-mock-ingest-time-rule",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "\u003cocid:3\u003e"
}`)
	createRequest := ocimock.MustJSONFixture[loganalyticssdk.CreateIngestTimeRuleDetails](t, `{
  "actions": [
    {
      "compartmentId": "\u003cocid:2\u003e",
      "dimensions": [
        "SOURCE_NAME"
      ],
      "metricName": "matched_records",
      "namespace": "osok_mock",
      "resourceGroup": "integration",
      "type": "METRIC_EXTRACTION"
    }
  ],
  "compartmentId": "\u003cocid:2\u003e",
  "conditions": {
    "fieldName": "mtag",
    "fieldOperator": "EQUAL",
    "fieldValue": "osok-mock",
    "kind": "FIELD"
  },
  "description": "OSOK recorded ingest-time rule",
  "displayName": "osok-mock-ingest-time-rule",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[loganalyticssdk.IngestTimeRule](t, `{
  "actions": [
    {
      "compartmentId": "<ocid:2>",
      "dimensions": [
        "SOURCE_NAME"
      ],
      "metricName": "matched_records",
      "namespace": "osok_mock",
      "resourceGroup": "integration",
      "type": "METRIC_EXTRACTION"
    }
  ],
  "compartmentId": "<ocid:2>",
  "conditions": {
    "additionalConditions": [],
    "fieldName": "mtag",
    "fieldOperator": "EQUAL",
    "fieldValue": "osok-mock",
    "kind": "FIELD"
  },
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T21:18:14.486Z"
    }
  },
  "description": "OSOK recorded ingest-time rule",
  "displayName": "osok-mock-ingest-time-rule",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-09-02T21:18:14.597Z",
  "timeUpdated": "2026-09-02T21:18:14.597Z"
}`)
	createdReadStates := []loganalyticssdk.IngestTimeRule{
		ocimock.MustOCIResponseFixture[loganalyticssdk.IngestTimeRule](t, `{
  "actions": [
    {
      "compartmentId": "<ocid:2>",
      "dimensions": [
        "SOURCE_NAME"
      ],
      "metricName": "matched_records",
      "namespace": "osok_mock",
      "resourceGroup": "integration",
      "type": "METRIC_EXTRACTION"
    }
  ],
  "compartmentId": "<ocid:2>",
  "conditions": {
    "additionalConditions": [],
    "fieldName": "mtag",
    "fieldOperator": "EQUAL",
    "fieldValue": "osok-mock",
    "kind": "FIELD"
  },
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T21:18:14.486Z"
    }
  },
  "description": "OSOK recorded ingest-time rule",
  "displayName": "osok-mock-ingest-time-rule",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-09-02T21:18:14.597Z",
  "timeUpdated": "2026-09-02T21:18:14.597Z"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[loganalyticssdk.IngestTimeRule](t, `{
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "\u003credacted\u003e",
      "CreatedOn": "2026-09-02T21:18:14.486Z"
    }
  },
  "description": "OSOK recorded ingest-time rule updated",
  "displayName": "osok-mock-ingest-time-rule",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "\u003cocid:3\u003e"
}`)
	updatedState := ocimock.MustOCIResponseFixture[loganalyticssdk.IngestTimeRule](t, `{
  "actions": [
    {
      "compartmentId": "<ocid:2>",
      "dimensions": [
        "SOURCE_NAME"
      ],
      "metricName": "matched_records",
      "namespace": "osok_mock",
      "resourceGroup": "integration",
      "type": "METRIC_EXTRACTION"
    }
  ],
  "compartmentId": "<ocid:2>",
  "conditions": {
    "additionalConditions": [],
    "fieldName": "mtag",
    "fieldOperator": "EQUAL",
    "fieldValue": "osok-mock",
    "kind": "FIELD"
  },
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T21:18:14.486Z"
    }
  },
  "description": "OSOK recorded ingest-time rule updated",
  "displayName": "osok-mock-ingest-time-rule",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-09-02T21:18:14.597Z",
  "timeUpdated": "2026-09-02T21:18:15.658Z"
}`)
	updatedReadStates := []loganalyticssdk.IngestTimeRule{
		ocimock.MustOCIResponseFixture[loganalyticssdk.IngestTimeRule](t, `{
  "actions": [
    {
      "compartmentId": "<ocid:2>",
      "dimensions": [
        "SOURCE_NAME"
      ],
      "metricName": "matched_records",
      "namespace": "osok_mock",
      "resourceGroup": "integration",
      "type": "METRIC_EXTRACTION"
    }
  ],
  "compartmentId": "<ocid:2>",
  "conditions": {
    "additionalConditions": [],
    "fieldName": "mtag",
    "fieldOperator": "EQUAL",
    "fieldValue": "osok-mock",
    "kind": "FIELD"
  },
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T21:18:14.486Z"
    }
  },
  "description": "OSOK recorded ingest-time rule updated",
  "displayName": "osok-mock-ingest-time-rule",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "timeCreated": "2026-09-02T21:18:14.597Z",
  "timeUpdated": "2026-09-02T21:18:15.658Z"
}`),
	}
	deletedReadStates := []loganalyticssdk.IngestTimeRule{
		ocimock.MustOCIResponseFixture[loganalyticssdk.IngestTimeRule](t, `{
  "actions": [
    {
      "compartmentId": "<ocid:2>",
      "dimensions": [
        "SOURCE_NAME"
      ],
      "metricName": "matched_records",
      "namespace": "osok_mock",
      "resourceGroup": "integration",
      "type": "METRIC_EXTRACTION"
    }
  ],
  "compartmentId": "<ocid:2>",
  "conditions": {
    "additionalConditions": [],
    "fieldName": "mtag",
    "fieldOperator": "EQUAL",
    "fieldValue": "osok-mock",
    "kind": "FIELD"
  },
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T21:18:14.486Z"
    }
  },
  "description": "OSOK recorded ingest-time rule updated",
  "displayName": "osok-mock-ingest-time-rule",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isEnabled": true,
  "lifecycleState": "DELETED",
  "timeCreated": "2026-09-02T21:18:14.597Z",
  "timeUpdated": "2026-09-02T21:18:15.658Z"
}`),
	}
	namespaceState := ocimock.MustOCIResponseFixture[loganalyticssdk.NamespaceCollection](t, `{"items":[{"compartmentId":"<ocid:1>","isArchivingEnabled":false,"isDataEverIngested":false,"isLogSetEnabled":false,"isOnboarded":true,"namespaceName":"iddevjmhjw0n"}]}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		loganalyticssdk.IngestTimeRule,
		loganalyticssdk.CreateIngestTimeRuleDetails,
		loganalyticssdk.IngestTimeRule,
	]{
		CollectionPath:    "/20200601/namespaces/iddevjmhjw0n/ingestTimeRules",
		ItemPath:          "/20200601/namespaces/iddevjmhjw0n/ingestTimeRules/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: createdReadStates,
		UpdatedReadStates: updatedReadStates,
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ loganalyticssdk.CreateIngestTimeRuleDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ loganalyticssdk.IngestTimeRule) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
		AdditionalRoutes: []ocimock.Route{{
			Name:         "resolve Log Analytics namespace",
			Method:       http.MethodGet,
			Path:         "/20200601/namespaces",
			MinimumCalls: 3,
			Respond: func(request ocimock.Request) (ocimock.Response, error) {
				if got := request.URL.Query().Get("compartmentId"); got != "<ocid:1>" {
					return ocimock.Response{}, fmt.Errorf("namespace compartmentId = %q", got)
				}
				return ocimock.JSONResponse(http.StatusOK, namespaceState)
			},
		}},
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
			t.Errorf("close IngestTimeRule OCI mock: %v", err)
		}
	})
	sdkClient := loganalyticssdk.LogAnalyticsClient{BaseClient: session.BaseClient()}
	client := newIngestTimeRuleServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient, "<ocid:1>")
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loganalyticsv1beta1.IngestTimeRule]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loganalyticsv1beta1.IngestTimeRule) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				current.Status.Conditions.Kind != "FIELD" ||
				current.Status.Conditions.FieldName != "mtag" ||
				current.Status.Conditions.FieldOperator != "EQUAL" ||
				len(current.Status.Actions) != 1 ||
				current.Status.Actions[0].Type != "METRIC_EXTRACTION" ||
				current.Status.Actions[0].MetricName != "matched_records" {
				return fmt.Errorf("created IngestTimeRule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loganalyticsv1beta1.IngestTimeRule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loganalyticsv1beta1.IngestTimeRule) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Id, current.Spec.Id) {
				return fmt.Errorf("updated IngestTimeRule status = %+v", current.Status)
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

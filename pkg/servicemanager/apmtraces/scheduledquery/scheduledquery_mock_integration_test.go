/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package scheduledquery

import (
	"context"
	"fmt"
	apmtracessdk "github.com/oracle/oci-go-sdk/v65/apmtraces"
	apmtracesv1beta1 "github.com/oracle/oci-service-operator/api/apmtraces/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationScheduledQueryLifecycleCRUD(t *testing.T) {
	t.Parallel()

	domainID := "ocid1.apmdomain.oc1..mock"
	resource := &apmtracesv1beta1.ScheduledQuery{
		Spec: apmtracesv1beta1.ScheduledQuerySpec{
			ApmDomainId:                           domainID,
			ScheduledQueryName:                    "osok-mock-scheduled-query",
			ScheduledQueryProcessingType:          "QUERY",
			ScheduledQueryProcessingSubType:       "NONE",
			ScheduledQueryText:                    "SHOW SPANS * FIRST 100 ROWS BETWEEN now() - 2 HOURS AND now()",
			ScheduledQuerySchedule:                "SCHEDULE STARTING AFTER 2099-01-01T00:00:00Z EVERY 720 MINUTES",
			ScheduledQueryDescription:             "OSOK synthetic scheduled query",
			ScheduledQueryMaximumRuntimeInSeconds: 60,
			ScheduledQueryRetentionCriteria:       "KEEP_DATA_UNTIL_RETENTION_PERIOD",
			ScheduledQueryRetentionPeriodInMs:     86400000,
			FreeformTags:                          map[string]string{"osok-mock": "create"},
		},
	}
	ocimock.InitializeResource(resource, "mock-scheduledquery")
	resource.Spec = ocimock.MustJSONFixture[apmtracesv1beta1.ScheduledQuerySpec](t, `{
  "freeformTags": {
    "osok-mock": "create"
  },
  "scheduledQueryDescription": "OSOK synthetic scheduled query",
  "scheduledQueryMaximumRuntimeInSeconds": 60,
  "scheduledQueryName": "osok-mock-scheduled-query",
  "scheduledQueryProcessingConfiguration": {
    "customMetric": {
      "isAnomalyDetectionEnabled": false,
      "isMetricPublished": false,
      "name": null
    }
  },
  "scheduledQueryProcessingSubType": "NONE",
  "scheduledQueryProcessingType": "QUERY",
  "scheduledQueryRetentionCriteria": "KEEP_DATA_UNTIL_RETENTION_PERIOD",
  "scheduledQueryRetentionPeriodInMs": 86400000,
  "scheduledQuerySchedule": "SCHEDULE STARTING AFTER 2099-01-01T00:00:00Z EVERY 720 MINUTES",
  "scheduledQueryText": "SHOW SPANS * FIRST 100 ROWS BETWEEN now() - 2 HOURS AND now()"
}`)
	resource.Spec.ApmDomainId = domainID
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "osok-mock": "update"
  },
  "scheduledQueryDescription": "OSOK synthetic scheduled query updated",
  "scheduledQueryProcessingConfiguration": {
    "customMetric": {
      "isAnomalyDetectionEnabled": false,
      "isMetricPublished": false,
      "name": null
    }
  },
  "scheduledQuerySchedule": "SCHEDULE STARTING AFTER 2099-01-01T00:00:00Z EVERY 360 MINUTES"
}`)
	createRequest := ocimock.MustJSONFixture[apmtracessdk.CreateScheduledQueryDetails](t, `{
  "freeformTags": {
    "osok-mock": "create"
  },
  "scheduledQueryDescription": "OSOK synthetic scheduled query",
  "scheduledQueryMaximumRuntimeInSeconds": 60,
  "scheduledQueryName": "osok-mock-scheduled-query",
  "scheduledQueryProcessingSubType": "NONE",
  "scheduledQueryProcessingType": "QUERY",
  "scheduledQueryRetentionCriteria": "KEEP_DATA_UNTIL_RETENTION_PERIOD",
  "scheduledQueryRetentionPeriodInMs": 86400000,
  "scheduledQuerySchedule": "SCHEDULE STARTING AFTER 2099-01-01T00:00:00Z EVERY 720 MINUTES",
  "scheduledQueryText": "SHOW SPANS * FIRST 100 ROWS BETWEEN now() - 2 HOURS AND now()"
}`)
	createdState := ocimock.MustOCIResponseFixture[apmtracessdk.ScheduledQuery](t, `{
  "apmDomainId": "<ocid:1>",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "scheduledQueryDescription": "OSOK synthetic scheduled query",
  "scheduledQueryMaximumRuntimeInSeconds": 60,
  "scheduledQueryName": "osok-mock-scheduled-query",
  "scheduledQueryProcessingConfiguration": {
    "customMetric": {
      "isAnomalyDetectionEnabled": false,
      "isMetricPublished": false,
      "name": null
    }
  },
  "scheduledQueryProcessingSubType": "NONE",
  "scheduledQueryProcessingType": "QUERY",
  "scheduledQueryRetentionCriteria": "KEEP_DATA_UNTIL_RETENTION_PERIOD",
  "scheduledQueryRetentionPeriodInMs": 86400000,
  "scheduledQuerySchedule": "SCHEDULE STARTING AFTER 2099-01-01T00:00:00Z EVERY 720 MINUTES",
  "scheduledQueryText": "SHOW SPANS * FIRST 100 ROWS BETWEEN now() - 2 HOURS AND now()"
}`)
	createdReadStates := []apmtracessdk.ScheduledQuery{
		ocimock.MustOCIResponseFixture[apmtracessdk.ScheduledQuery](t, `{
  "apmDomainId": "<ocid:1>",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "scheduledQueryDescription": "OSOK synthetic scheduled query",
  "scheduledQueryMaximumRuntimeInSeconds": 60,
  "scheduledQueryName": "osok-mock-scheduled-query",
  "scheduledQueryProcessingConfiguration": {
    "customMetric": {
      "isAnomalyDetectionEnabled": false,
      "isMetricPublished": false,
      "name": null
    }
  },
  "scheduledQueryProcessingSubType": "NONE",
  "scheduledQueryProcessingType": "QUERY",
  "scheduledQueryRetentionCriteria": "KEEP_DATA_UNTIL_RETENTION_PERIOD",
  "scheduledQueryRetentionPeriodInMs": 86400000,
  "scheduledQuerySchedule": "SCHEDULE STARTING AFTER 2099-01-01T00:00:00Z EVERY 720 MINUTES",
  "scheduledQueryText": "SHOW SPANS * FIRST 100 ROWS BETWEEN now() - 2 HOURS AND now()"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[apmtracessdk.UpdateScheduledQueryDetails](t, `{
  "freeformTags": {
    "osok-mock": "update"
  },
  "scheduledQueryDescription": "OSOK synthetic scheduled query updated",
  "scheduledQueryProcessingConfiguration": {
    "customMetric": {
      "isAnomalyDetectionEnabled": false,
      "isMetricPublished": false,
      "name": null
    }
  },
  "scheduledQuerySchedule": "SCHEDULE STARTING AFTER 2099-01-01T00:00:00Z EVERY 360 MINUTES"
}`)
	updatedState := ocimock.MustOCIResponseFixture[apmtracessdk.ScheduledQuery](t, `{
  "apmDomainId": "<ocid:1>",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "scheduledQueryDescription": "OSOK synthetic scheduled query updated",
  "scheduledQueryMaximumRuntimeInSeconds": 60,
  "scheduledQueryName": "osok-mock-scheduled-query",
  "scheduledQueryProcessingConfiguration": {
    "customMetric": {
      "isAnomalyDetectionEnabled": false,
      "isMetricPublished": false,
      "name": null
    }
  },
  "scheduledQueryProcessingSubType": "NONE",
  "scheduledQueryProcessingType": "QUERY",
  "scheduledQueryRetentionCriteria": "KEEP_DATA_UNTIL_RETENTION_PERIOD",
  "scheduledQueryRetentionPeriodInMs": 86400000,
  "scheduledQuerySchedule": "SCHEDULE STARTING AFTER 2099-01-01T00:00:00Z EVERY 360 MINUTES",
  "scheduledQueryText": "SHOW SPANS * FIRST 100 ROWS BETWEEN now() - 2 HOURS AND now()"
}`)
	updatedReadStates := []apmtracessdk.ScheduledQuery{
		ocimock.MustOCIResponseFixture[apmtracessdk.ScheduledQuery](t, `{
  "apmDomainId": "<ocid:1>",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "scheduledQueryDescription": "OSOK synthetic scheduled query updated",
  "scheduledQueryMaximumRuntimeInSeconds": 60,
  "scheduledQueryName": "osok-mock-scheduled-query",
  "scheduledQueryProcessingConfiguration": {
    "customMetric": {
      "isAnomalyDetectionEnabled": false,
      "isMetricPublished": false,
      "name": null
    }
  },
  "scheduledQueryProcessingSubType": "NONE",
  "scheduledQueryProcessingType": "QUERY",
  "scheduledQueryRetentionCriteria": "KEEP_DATA_UNTIL_RETENTION_PERIOD",
  "scheduledQueryRetentionPeriodInMs": 86400000,
  "scheduledQuerySchedule": "SCHEDULE STARTING AFTER 2099-01-01T00:00:00Z EVERY 360 MINUTES",
  "scheduledQueryText": "SHOW SPANS * FIRST 100 ROWS BETWEEN now() - 2 HOURS AND now()"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		apmtracessdk.ScheduledQuery,
		apmtracessdk.CreateScheduledQueryDetails,
		apmtracessdk.UpdateScheduledQueryDetails,
	]{
		CollectionPath:     "/20200630/scheduledQueries",
		ItemPath:           "/20200630/scheduledQueries/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		CreatedReadStates:  append(ocimock.LifecycleStates(t, createdState, "CREATING"), createdReadStates...),
		UpdatedReadStates:  append(ocimock.LifecycleStates(t, updatedState, "UPDATING"), updatedReadStates...),
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ apmtracessdk.CreateScheduledQueryDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ apmtracessdk.ScheduledQuery) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ScheduledQuery OCI mock: %v", err)
		}
	})
	sdkClient := apmtracessdk.ScheduledQueryClient{BaseClient: session.BaseClient()}
	client := newScheduledQueryServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("recorded-integration")},
		sdkClient,
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apmtracesv1beta1.ScheduledQuery]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apmtracesv1beta1.ScheduledQuery) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ScheduledQueryDescription, current.Spec.ScheduledQueryDescription) ||
				!reflect.DeepEqual(current.Status.ScheduledQueryMaximumRuntimeInSeconds, current.Spec.ScheduledQueryMaximumRuntimeInSeconds) ||
				!reflect.DeepEqual(current.Status.ScheduledQueryName, current.Spec.ScheduledQueryName) ||
				!reflect.DeepEqual(current.Status.ScheduledQueryProcessingSubType, current.Spec.ScheduledQueryProcessingSubType) ||
				!reflect.DeepEqual(current.Status.ScheduledQueryProcessingType, current.Spec.ScheduledQueryProcessingType) ||
				!reflect.DeepEqual(current.Status.ScheduledQueryRetentionCriteria, current.Spec.ScheduledQueryRetentionCriteria) ||
				!reflect.DeepEqual(current.Status.ScheduledQueryRetentionPeriodInMs, current.Spec.ScheduledQueryRetentionPeriodInMs) ||
				!reflect.DeepEqual(current.Status.ScheduledQuerySchedule, current.Spec.ScheduledQuerySchedule) ||
				!reflect.DeepEqual(current.Status.ScheduledQueryText, current.Spec.ScheduledQueryText) {
				return fmt.Errorf("created ScheduledQuery status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apmtracesv1beta1.ScheduledQuery) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *apmtracesv1beta1.ScheduledQuery) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ScheduledQueryDescription, current.Spec.ScheduledQueryDescription) ||
				!reflect.DeepEqual(current.Status.ScheduledQuerySchedule, current.Spec.ScheduledQuerySchedule) {
				return fmt.Errorf("updated ScheduledQuery status = %+v", current.Status)
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

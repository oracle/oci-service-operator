/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package loganalyticsentity

import (
	"context"
	"fmt"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"net/http"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationLogAnalyticsEntityCompositeCRUD(t *testing.T) {
	t.Parallel()

	resource := &loganalyticsv1beta1.LogAnalyticsEntity{}
	ocimock.InitializeResource(resource, "mock-loganalyticsentity")
	resource.Spec = ocimock.MustJSONFixture[loganalyticsv1beta1.LogAnalyticsEntitySpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "entityTypeName": "Palo Alto Networks",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hostname": "osok-mock-create.example.com",
  "name": "osok-mock-log-analytics-entity",
  "timezoneRegion": "UTC"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "osok-mock": "update"
  },
  "hostname": "osok-mock-update.example.com"
}`)
	createRequest := ocimock.MustJSONFixture[loganalyticssdk.CreateLogAnalyticsEntityDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "entityTypeName": "Palo Alto Networks",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hostname": "osok-mock-create.example.com",
  "name": "osok-mock-log-analytics-entity",
  "timezoneRegion": "UTC"
}`)
	createdState := ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsEntity](t, `{
  "areLogsCollected": false,
  "associatedSourcesCount": null,
  "cloudResourceId": null,
  "compartmentId": "<ocid:1>",
  "creationSource": null,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T04:02:43.967Z"
    }
  },
  "entityTypeInternalName": "oci_palo_alto",
  "entityTypeName": "Palo Alto Networks",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hostname": "osok-mock-create.example.com",
  "id": "<ocid:2>",
  "lifecycleDetails": "READY",
  "lifecycleState": "ACTIVE",
  "managementAgentCompartmentId": null,
  "managementAgentDisplayName": null,
  "managementAgentId": null,
  "metadata": {
    "items": []
  },
  "name": "osok-mock-log-analytics-entity",
  "properties": {},
  "sourceId": null,
  "timeCreated": "2026-09-02T04:02:44.150Z",
  "timeLastDiscovered": null,
  "timeUpdated": "2026-09-02T04:02:44.150Z",
  "timezoneRegion": "UTC"
}`)
	createdReadStates := []loganalyticssdk.LogAnalyticsEntity{
		ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsEntity](t, `{
  "areLogsCollected": false,
  "associatedSourcesCount": null,
  "cloudResourceId": null,
  "compartmentId": "<ocid:1>",
  "creationSource": null,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T04:02:43.967Z"
    }
  },
  "entityTypeInternalName": "oci_palo_alto",
  "entityTypeName": "Palo Alto Networks",
  "freeformTags": {
    "osok-mock": "create"
  },
  "hostname": "osok-mock-create.example.com",
  "id": "<ocid:2>",
  "lifecycleDetails": "READY",
  "lifecycleState": "ACTIVE",
  "managementAgentCompartmentId": null,
  "managementAgentDisplayName": null,
  "managementAgentId": null,
  "metadata": {
    "items": []
  },
  "name": "osok-mock-log-analytics-entity",
  "properties": {},
  "sourceId": null,
  "timeCreated": "2026-09-02T04:02:44.150Z",
  "timeLastDiscovered": null,
  "timeUpdated": "2026-09-02T04:02:44.150Z",
  "timezoneRegion": "UTC"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[loganalyticssdk.UpdateLogAnalyticsEntityDetails](t, `{
  "freeformTags": {
    "osok-mock": "update"
  },
  "hostname": "osok-mock-update.example.com"
}`)
	updatedState := ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsEntity](t, `{
  "areLogsCollected": false,
  "associatedSourcesCount": null,
  "cloudResourceId": null,
  "compartmentId": "<ocid:1>",
  "creationSource": null,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T04:02:43.967Z"
    }
  },
  "entityTypeInternalName": "oci_palo_alto",
  "entityTypeName": "Palo Alto Networks",
  "freeformTags": {
    "osok-mock": "update"
  },
  "hostname": "osok-mock-update.example.com",
  "id": "<ocid:2>",
  "lifecycleDetails": "READY",
  "lifecycleState": "ACTIVE",
  "managementAgentCompartmentId": null,
  "managementAgentDisplayName": null,
  "managementAgentId": null,
  "metadata": {
    "items": []
  },
  "name": "osok-mock-log-analytics-entity",
  "properties": {},
  "sourceId": null,
  "timeCreated": "2026-09-02T04:02:44.150Z",
  "timeLastDiscovered": null,
  "timeUpdated": "2026-09-02T04:02:44.739Z",
  "timezoneRegion": "UTC"
}`)
	updatedReadStates := []loganalyticssdk.LogAnalyticsEntity{
		ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsEntity](t, `{
  "areLogsCollected": false,
  "associatedSourcesCount": null,
  "cloudResourceId": null,
  "compartmentId": "<ocid:1>",
  "creationSource": null,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T04:02:43.967Z"
    }
  },
  "entityTypeInternalName": "oci_palo_alto",
  "entityTypeName": "Palo Alto Networks",
  "freeformTags": {
    "osok-mock": "update"
  },
  "hostname": "osok-mock-update.example.com",
  "id": "<ocid:2>",
  "lifecycleDetails": "READY",
  "lifecycleState": "ACTIVE",
  "managementAgentCompartmentId": null,
  "managementAgentDisplayName": null,
  "managementAgentId": null,
  "metadata": {
    "items": []
  },
  "name": "osok-mock-log-analytics-entity",
  "properties": {},
  "sourceId": null,
  "timeCreated": "2026-09-02T04:02:44.150Z",
  "timeLastDiscovered": null,
  "timeUpdated": "2026-09-02T04:02:44.739Z",
  "timezoneRegion": "UTC"
}`),
	}
	deletedReadStates := []loganalyticssdk.LogAnalyticsEntity{
		ocimock.MustOCIResponseFixture[loganalyticssdk.LogAnalyticsEntity](t, `{
  "areLogsCollected": false,
  "associatedSourcesCount": null,
  "cloudResourceId": null,
  "compartmentId": "<ocid:1>",
  "creationSource": null,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T04:02:43.967Z"
    }
  },
  "entityTypeInternalName": "oci_palo_alto",
  "entityTypeName": "Palo Alto Networks",
  "freeformTags": {
    "osok-mock": "update"
  },
  "hostname": "osok-mock-update.example.com",
  "id": "<ocid:2>",
  "lifecycleDetails": "READY",
  "lifecycleState": "DELETED",
  "managementAgentCompartmentId": null,
  "managementAgentDisplayName": null,
  "managementAgentId": null,
  "metadata": {
    "items": []
  },
  "name": "osok-mock-log-analytics-entity",
  "properties": {},
  "sourceId": null,
  "timeCreated": "2026-09-02T04:02:44.150Z",
  "timeLastDiscovered": null,
  "timeUpdated": "2026-09-02T04:02:45.291Z",
  "timezoneRegion": "UTC"
}`),
	}
	namespaceState := ocimock.MustOCIResponseFixture[loganalyticssdk.NamespaceCollection](t, `{"items":[{"compartmentId":"<ocid:1>","isArchivingEnabled":false,"isDataEverIngested":false,"isLogSetEnabled":false,"isOnboarded":true,"namespaceName":"<binding:loganalytics-namespace>"}]}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		loganalyticssdk.LogAnalyticsEntity,
		loganalyticssdk.CreateLogAnalyticsEntityDetails,
		loganalyticssdk.UpdateLogAnalyticsEntityDetails,
	]{
		CollectionPath:    "/20200601/namespaces/<binding:loganalytics-namespace>/logAnalyticsEntities",
		ItemPath:          "/20200601/namespaces/<binding:loganalytics-namespace>/logAnalyticsEntities/<ocid:2>",
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
		ValidateCreate: func(request ocimock.Request, _ loganalyticssdk.CreateLogAnalyticsEntityDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ loganalyticssdk.LogAnalyticsEntity) error {
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
			t.Errorf("close LogAnalyticsEntity OCI mock: %v", err)
		}
	})
	sdkClient := loganalyticssdk.LogAnalyticsClient{BaseClient: session.BaseClient()}
	hooks := newLogAnalyticsEntityRuntimeHooksWithOCIClient(sdkClient)
	applyLogAnalyticsEntityRuntimeHooks(&hooks, sdkClient, nil)
	manager := &LogAnalyticsEntityServiceManager{}
	client := wrapLogAnalyticsEntityGeneratedClient(hooks, defaultLogAnalyticsEntityServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loganalyticsv1beta1.LogAnalyticsEntity](buildLogAnalyticsEntityGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loganalyticsv1beta1.LogAnalyticsEntity]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loganalyticsv1beta1.LogAnalyticsEntity) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.EntityTypeName, current.Spec.EntityTypeName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Hostname, current.Spec.Hostname) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.TimezoneRegion, current.Spec.TimezoneRegion) {
				return fmt.Errorf("created LogAnalyticsEntity status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loganalyticsv1beta1.LogAnalyticsEntity) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loganalyticsv1beta1.LogAnalyticsEntity) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Hostname, current.Spec.Hostname) {
				return fmt.Errorf("updated LogAnalyticsEntity status = %+v", current.Status)
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

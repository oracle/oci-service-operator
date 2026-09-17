/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package managementsavedsearch

import (
	"context"
	"fmt"
	managementdashboardsdk "github.com/oracle/oci-go-sdk/v65/managementdashboard"
	managementdashboardv1beta1 "github.com/oracle/oci-service-operator/api/managementdashboard/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationManagementSavedSearchEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &managementdashboardv1beta1.ManagementSavedSearch{}
	ocimock.InitializeResource(resource, "mock-managementsavedsearch")
	resource.Spec = ocimock.MustJSONFixture[managementdashboardv1beta1.ManagementSavedSearchSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dataConfig": [
    {
      "query": "*"
    }
  ],
  "description": "OSOK recorded saved search",
  "displayName": "osok-mock-management-saved-search",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isOobSavedSearch": false,
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock"
  },
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "screenImage": "none",
  "type": "WIDGET_SHOW_IN_DASHBOARD",
  "uiConfig": {
    "visualization": "table"
  },
  "widgetTemplate": "\u003cdiv\u003e\u003c/div\u003e",
  "widgetVM": "{}"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded saved search updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[managementdashboardsdk.CreateManagementSavedSearchDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dataConfig": [
    {
      "query": "*"
    }
  ],
  "description": "OSOK recorded saved search",
  "displayName": "osok-mock-management-saved-search",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isOobSavedSearch": false,
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock"
  },
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "screenImage": "none",
  "type": "WIDGET_SHOW_IN_DASHBOARD",
  "uiConfig": {
    "visualization": "table"
  },
  "widgetTemplate": "\u003cdiv\u003e\u003c/div\u003e",
  "widgetVM": "{}"
}`)
	createdState := ocimock.MustOCIResponseFixture[managementdashboardsdk.ManagementSavedSearch](t, `{
  "compartmentId": "<ocid:1>",
  "createdBy": "<redacted>",
  "dataConfig": [
    {
      "query": "*"
    }
  ],
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:16:15.770Z"
    }
  },
  "description": "OSOK recorded saved search",
  "displayName": "osok-mock-management-saved-search",
  "drilldownConfig": [],
  "featuresConfig": {
    "crossService": {
      "shared": false
    }
  },
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "isOobSavedSearch": false,
  "lifecycleState": "ACTIVE",
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock"
  },
  "parametersConfig": [],
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "screenImage": "none",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:16:15.850Z",
  "timeUpdated": "2026-09-02T20:16:15.850Z",
  "type": "WIDGET_SHOW_IN_DASHBOARD",
  "uiConfig": {
    "visualization": "table"
  },
  "updatedBy": "<ocid:3>",
  "widgetTemplate": "<div></div>",
  "widgetVM": "{}"
}`)
	createdReadStates := []managementdashboardsdk.ManagementSavedSearch{
		ocimock.MustOCIResponseFixture[managementdashboardsdk.ManagementSavedSearch](t, `{
  "compartmentId": "<ocid:1>",
  "createdBy": "<redacted>",
  "dataConfig": [
    {
      "query": "*"
    }
  ],
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:16:15.770Z"
    }
  },
  "description": "OSOK recorded saved search",
  "displayName": "osok-mock-management-saved-search",
  "drilldownConfig": [],
  "featuresConfig": {
    "crossService": {
      "shared": false
    }
  },
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "isOobSavedSearch": false,
  "lifecycleState": "ACTIVE",
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock"
  },
  "parametersConfig": [],
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "screenImage": "none",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:16:15.850Z",
  "timeUpdated": "2026-09-02T20:16:15.850Z",
  "type": "WIDGET_SHOW_IN_DASHBOARD",
  "uiConfig": {
    "visualization": "table"
  },
  "updatedBy": "<ocid:3>",
  "widgetTemplate": "<div></div>",
  "widgetVM": "{}"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[managementdashboardsdk.UpdateManagementSavedSearchDetails](t, `{
  "description": "OSOK recorded saved search updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[managementdashboardsdk.ManagementSavedSearch](t, `{
  "compartmentId": "<ocid:1>",
  "createdBy": "<redacted>",
  "dataConfig": [
    {
      "query": "*"
    }
  ],
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:16:15.770Z"
    }
  },
  "description": "OSOK recorded saved search updated",
  "displayName": "osok-mock-management-saved-search",
  "drilldownConfig": [],
  "featuresConfig": {
    "crossService": {
      "shared": false
    }
  },
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "isOobSavedSearch": false,
  "lifecycleState": "ACTIVE",
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock"
  },
  "parametersConfig": [],
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "screenImage": "none",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:16:15.850Z",
  "timeUpdated": "2026-09-02T20:16:16.645Z",
  "type": "WIDGET_SHOW_IN_DASHBOARD",
  "uiConfig": {
    "visualization": "table"
  },
  "updatedBy": "<ocid:3>",
  "widgetTemplate": "<div></div>",
  "widgetVM": "{}"
}`)
	updatedReadStates := []managementdashboardsdk.ManagementSavedSearch{
		ocimock.MustOCIResponseFixture[managementdashboardsdk.ManagementSavedSearch](t, `{
  "compartmentId": "<ocid:1>",
  "createdBy": "<redacted>",
  "dataConfig": [
    {
      "query": "*"
    }
  ],
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:16:15.770Z"
    }
  },
  "description": "OSOK recorded saved search updated",
  "displayName": "osok-mock-management-saved-search",
  "drilldownConfig": [],
  "featuresConfig": {
    "crossService": {
      "shared": false
    }
  },
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "isOobSavedSearch": false,
  "lifecycleState": "ACTIVE",
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock"
  },
  "parametersConfig": [],
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "screenImage": "none",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:16:15.850Z",
  "timeUpdated": "2026-09-02T20:16:16.645Z",
  "type": "WIDGET_SHOW_IN_DASHBOARD",
  "uiConfig": {
    "visualization": "table"
  },
  "updatedBy": "<ocid:3>",
  "widgetTemplate": "<div></div>",
  "widgetVM": "{}"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		managementdashboardsdk.ManagementSavedSearch,
		managementdashboardsdk.CreateManagementSavedSearchDetails,
		managementdashboardsdk.UpdateManagementSavedSearchDetails,
	]{
		CollectionPath:     "/20200901/managementSavedSearches",
		ItemPath:           "/20200901/managementSavedSearches/<ocid:2>",
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
		DeleteStatus:       200,
		NotFoundCode:       "NotAuthorizedOrNotFound",
		ValidateCreate: func(request ocimock.Request, _ managementdashboardsdk.CreateManagementSavedSearchDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ managementdashboardsdk.ManagementSavedSearch) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200901", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ManagementSavedSearch OCI mock: %v", err)
		}
	})
	sdkClient := managementdashboardsdk.DashxApisClient{BaseClient: session.BaseClient()}
	client := newManagementSavedSearchServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*managementdashboardv1beta1.ManagementSavedSearch]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *managementdashboardv1beta1.ManagementSavedSearch) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsOobSavedSearch, current.Spec.IsOobSavedSearch) ||
				!reflect.DeepEqual(current.Status.MetadataVersion, current.Spec.MetadataVersion) ||
				!reflect.DeepEqual(current.Status.ProviderId, current.Spec.ProviderId) ||
				!reflect.DeepEqual(current.Status.ProviderName, current.Spec.ProviderName) ||
				!reflect.DeepEqual(current.Status.ProviderVersion, current.Spec.ProviderVersion) ||
				!reflect.DeepEqual(current.Status.ScreenImage, current.Spec.ScreenImage) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) ||
				!reflect.DeepEqual(current.Status.WidgetTemplate, current.Spec.WidgetTemplate) ||
				len(current.Status.DataConfig) != 1 ||
				current.Status.DataConfig[0].Raw == nil ||
				current.Status.Nls.Raw == nil ||
				current.Status.UiConfig.Raw == nil ||
				current.Status.WidgetVM != "{}" {
				return fmt.Errorf("created ManagementSavedSearch status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *managementdashboardv1beta1.ManagementSavedSearch) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *managementdashboardv1beta1.ManagementSavedSearch) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated ManagementSavedSearch status = %+v", current.Status)
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

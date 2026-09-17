/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package managementdashboard

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
func TestMockIntegrationManagementDashboardEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &managementdashboardv1beta1.ManagementDashboard{}
	ocimock.InitializeResource(resource, "mock-managementdashboard")
	resource.Spec = ocimock.MustJSONFixture[managementdashboardv1beta1.ManagementDashboardSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dataConfig": [
    {
      "source": "log-analytics"
    }
  ],
  "description": "OSOK recorded management dashboard",
  "displayName": "osok-mock-management-dashboard",
  "featuresConfig": {
    "crossService": {
      "shared": false
    }
  },
  "freeformTags": {
    "osok-mock": "create"
  },
  "isFavorite": false,
  "isOobDashboard": false,
  "isShowDescription": true,
  "isShowInHome": false,
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock dashboard"
  },
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "screenImage": "none",
  "tiles": [
    {
      "column": 1,
      "dataConfig": [
        {
          "query": "*"
        }
      ],
      "displayName": "OSOK mock tile",
      "drilldownConfig": [],
      "height": 4,
      "nls": {
        "title": "OSOK mock tile"
      },
      "parametersMap": {},
      "row": 1,
      "savedSearchId": "\u003cocid:2\u003e",
      "state": "DEFAULT",
      "uiConfig": {
        "visualization": "table"
      },
      "width": 6
    }
  ],
  "type": "NORMAL",
  "uiConfig": {
    "layout": "grid"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded management dashboard updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "isFavorite": true
}`)
	createRequest := ocimock.MustJSONFixture[managementdashboardsdk.CreateManagementDashboardDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dataConfig": [
    {
      "source": "log-analytics"
    }
  ],
  "description": "OSOK recorded management dashboard",
  "displayName": "osok-mock-management-dashboard",
  "featuresConfig": {
    "crossService": {
      "shared": false
    }
  },
  "freeformTags": {
    "osok-mock": "create"
  },
  "isFavorite": false,
  "isOobDashboard": false,
  "isShowDescription": true,
  "isShowInHome": false,
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock dashboard"
  },
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "screenImage": "none",
  "tiles": [
    {
      "column": 1,
      "dataConfig": [
        {
          "query": "*"
        }
      ],
      "displayName": "OSOK mock tile",
      "drilldownConfig": [],
      "height": 4,
      "nls": {
        "title": "OSOK mock tile"
      },
      "parametersMap": {},
      "row": 1,
      "savedSearchId": "\u003cocid:2\u003e",
      "state": "DEFAULT",
      "uiConfig": {
        "visualization": "table"
      },
      "width": 6
    }
  ],
  "type": "NORMAL",
  "uiConfig": {
    "layout": "grid"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[managementdashboardsdk.ManagementDashboard](t, `{
  "compartmentId": "<ocid:1>",
  "createdBy": "<redacted>",
  "dashboardId": "<ocid:3>",
  "dataConfig": [
    {
      "source": "log-analytics"
    }
  ],
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:55:16.003Z"
    }
  },
  "description": "OSOK recorded management dashboard",
  "displayName": "osok-mock-management-dashboard",
  "drilldownConfig": [],
  "featuresConfig": {
    "crossService": {
      "shared": false
    }
  },
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "isFavorite": false,
  "isOobDashboard": false,
  "isShowDescription": true,
  "isShowInHome": false,
  "lifecycleState": "ACTIVE",
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock dashboard"
  },
  "parametersConfig": [],
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "savedSearches": [
    {
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
          "CreatedOn": "2026-09-02T20:28:07.817Z"
        }
      },
      "description": "Temporary OSOK mock prerequisite",
      "displayName": "osok-mock-batch5-dashboard-prerequisite",
      "drilldownConfig": [],
      "featuresConfig": {
        "crossService": {
          "shared": false
        }
      },
      "freeformTags": {},
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
      "timeCreated": "2026-09-02T20:28:07.963Z",
      "timeUpdated": "2026-09-02T20:28:07.963Z",
      "type": "WIDGET_SHOW_IN_DASHBOARD",
      "uiConfig": {
        "visualization": "table"
      },
      "updatedBy": "<ocid:4>",
      "widgetTemplate": "<div></div>",
      "widgetVM": "{}"
    }
  ],
  "screenImage": "none",
  "systemTags": {},
  "tiles": [
    {
      "column": 1,
      "dataConfig": [
        {
          "query": "*"
        }
      ],
      "description": null,
      "displayName": "OSOK mock tile",
      "drilldownConfig": [],
      "height": 4,
      "nls": {
        "title": "OSOK mock tile"
      },
      "parametersMap": {},
      "row": 1,
      "savedSearchId": "<ocid:2>",
      "state": "DEFAULT",
      "uiConfig": {
        "visualization": "table"
      },
      "width": 6
    }
  ],
  "timeCreated": "2026-09-02T20:55:16.101Z",
  "timeUpdated": "2026-09-02T20:55:16.101Z",
  "type": "NORMAL",
  "uiConfig": {
    "layout": "grid"
  },
  "updatedBy": "<ocid:4>"
}`)
	createdReadStates := []managementdashboardsdk.ManagementDashboard{
		ocimock.MustOCIResponseFixture[managementdashboardsdk.ManagementDashboard](t, `{
  "compartmentId": "<ocid:1>",
  "createdBy": "<redacted>",
  "dashboardId": "<ocid:3>",
  "dataConfig": [
    {
      "source": "log-analytics"
    }
  ],
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:55:16.003Z"
    }
  },
  "description": "OSOK recorded management dashboard",
  "displayName": "osok-mock-management-dashboard",
  "drilldownConfig": [],
  "featuresConfig": {
    "crossService": {
      "shared": false
    }
  },
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "isFavorite": false,
  "isOobDashboard": false,
  "isShowDescription": true,
  "isShowInHome": false,
  "lifecycleState": "ACTIVE",
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock dashboard"
  },
  "parametersConfig": [],
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "savedSearches": [
    {
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
          "CreatedOn": "2026-09-02T20:28:07.817Z"
        }
      },
      "description": "Temporary OSOK mock prerequisite",
      "displayName": "osok-mock-batch5-dashboard-prerequisite",
      "drilldownConfig": [],
      "featuresConfig": {
        "crossService": {
          "shared": false
        }
      },
      "freeformTags": {},
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
      "timeCreated": "2026-09-02T20:28:07.963Z",
      "timeUpdated": "2026-09-02T20:28:07.963Z",
      "type": "WIDGET_SHOW_IN_DASHBOARD",
      "uiConfig": {
        "visualization": "table"
      },
      "updatedBy": "<ocid:4>",
      "widgetTemplate": "<div></div>",
      "widgetVM": "{}"
    }
  ],
  "screenImage": "none",
  "systemTags": {},
  "tiles": [
    {
      "column": 1,
      "dataConfig": [
        {
          "query": "*"
        }
      ],
      "description": null,
      "displayName": "OSOK mock tile",
      "drilldownConfig": [],
      "height": 4,
      "nls": {
        "title": "OSOK mock tile"
      },
      "parametersMap": {},
      "row": 1,
      "savedSearchId": "<ocid:2>",
      "state": "DEFAULT",
      "uiConfig": {
        "visualization": "table"
      },
      "width": 6
    }
  ],
  "timeCreated": "2026-09-02T20:55:16.101Z",
  "timeUpdated": "2026-09-02T20:55:16.101Z",
  "type": "NORMAL",
  "uiConfig": {
    "layout": "grid"
  },
  "updatedBy": "<ocid:4>"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[managementdashboardsdk.UpdateManagementDashboardDetails](t, `{
  "description": "OSOK recorded management dashboard updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "isFavorite": true
}`)
	updatedState := ocimock.MustOCIResponseFixture[managementdashboardsdk.ManagementDashboard](t, `{
  "compartmentId": "<ocid:1>",
  "createdBy": "<redacted>",
  "dashboardId": "<ocid:3>",
  "dataConfig": [
    {
      "source": "log-analytics"
    }
  ],
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:55:16.003Z"
    }
  },
  "description": "OSOK recorded management dashboard updated",
  "displayName": "osok-mock-management-dashboard",
  "drilldownConfig": [],
  "featuresConfig": {
    "crossService": {
      "shared": false
    }
  },
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isFavorite": true,
  "isOobDashboard": false,
  "isShowDescription": true,
  "isShowInHome": false,
  "lifecycleState": "ACTIVE",
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock dashboard"
  },
  "parametersConfig": [],
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "savedSearches": [
    {
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
          "CreatedOn": "2026-09-02T20:28:07.817Z"
        }
      },
      "description": "Temporary OSOK mock prerequisite",
      "displayName": "osok-mock-batch5-dashboard-prerequisite",
      "drilldownConfig": [],
      "featuresConfig": {
        "crossService": {
          "shared": false
        }
      },
      "freeformTags": {},
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
      "timeCreated": "2026-09-02T20:28:07.963Z",
      "timeUpdated": "2026-09-02T20:28:07.963Z",
      "type": "WIDGET_SHOW_IN_DASHBOARD",
      "uiConfig": {
        "visualization": "table"
      },
      "updatedBy": "<ocid:4>",
      "widgetTemplate": "<div></div>",
      "widgetVM": "{}"
    }
  ],
  "screenImage": "none",
  "systemTags": {},
  "tiles": [
    {
      "column": 1,
      "dataConfig": [
        {
          "query": "*"
        }
      ],
      "description": null,
      "displayName": "OSOK mock tile",
      "drilldownConfig": [],
      "height": 4,
      "nls": {
        "title": "OSOK mock tile"
      },
      "parametersMap": {},
      "row": 1,
      "savedSearchId": "<ocid:2>",
      "state": "DEFAULT",
      "uiConfig": {
        "visualization": "table"
      },
      "width": 6
    }
  ],
  "timeCreated": "2026-09-02T20:55:16.101Z",
  "timeUpdated": "2026-09-02T20:55:17.065Z",
  "type": "NORMAL",
  "uiConfig": {
    "layout": "grid"
  },
  "updatedBy": "<ocid:4>"
}`)
	updatedReadStates := []managementdashboardsdk.ManagementDashboard{
		ocimock.MustOCIResponseFixture[managementdashboardsdk.ManagementDashboard](t, `{
  "compartmentId": "<ocid:1>",
  "createdBy": "<redacted>",
  "dashboardId": "<ocid:3>",
  "dataConfig": [
    {
      "source": "log-analytics"
    }
  ],
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:55:16.003Z"
    }
  },
  "description": "OSOK recorded management dashboard updated",
  "displayName": "osok-mock-management-dashboard",
  "drilldownConfig": [],
  "featuresConfig": {
    "crossService": {
      "shared": false
    }
  },
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isFavorite": true,
  "isOobDashboard": false,
  "isShowDescription": true,
  "isShowInHome": false,
  "lifecycleState": "ACTIVE",
  "metadataVersion": "2.0",
  "nls": {
    "title": "OSOK mock dashboard"
  },
  "parametersConfig": [],
  "providerId": "log-analytics",
  "providerName": "Logging Analytics",
  "providerVersion": "3.0.0",
  "savedSearches": [
    {
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
          "CreatedOn": "2026-09-02T20:28:07.817Z"
        }
      },
      "description": "Temporary OSOK mock prerequisite",
      "displayName": "osok-mock-batch5-dashboard-prerequisite",
      "drilldownConfig": [],
      "featuresConfig": {
        "crossService": {
          "shared": false
        }
      },
      "freeformTags": {},
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
      "timeCreated": "2026-09-02T20:28:07.963Z",
      "timeUpdated": "2026-09-02T20:28:07.963Z",
      "type": "WIDGET_SHOW_IN_DASHBOARD",
      "uiConfig": {
        "visualization": "table"
      },
      "updatedBy": "<ocid:4>",
      "widgetTemplate": "<div></div>",
      "widgetVM": "{}"
    }
  ],
  "screenImage": "none",
  "systemTags": {},
  "tiles": [
    {
      "column": 1,
      "dataConfig": [
        {
          "query": "*"
        }
      ],
      "description": null,
      "displayName": "OSOK mock tile",
      "drilldownConfig": [],
      "height": 4,
      "nls": {
        "title": "OSOK mock tile"
      },
      "parametersMap": {},
      "row": 1,
      "savedSearchId": "<ocid:2>",
      "state": "DEFAULT",
      "uiConfig": {
        "visualization": "table"
      },
      "width": 6
    }
  ],
  "timeCreated": "2026-09-02T20:55:16.101Z",
  "timeUpdated": "2026-09-02T20:55:17.065Z",
  "type": "NORMAL",
  "uiConfig": {
    "layout": "grid"
  },
  "updatedBy": "<ocid:4>"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		managementdashboardsdk.ManagementDashboard,
		managementdashboardsdk.CreateManagementDashboardDetails,
		managementdashboardsdk.UpdateManagementDashboardDetails,
	]{
		CollectionPath:     "/20200901/managementDashboards",
		ItemPath:           "/20200901/managementDashboards/<ocid:3>",
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
		ValidateCreate: func(request ocimock.Request, _ managementdashboardsdk.CreateManagementDashboardDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ managementdashboardsdk.ManagementDashboard) error {
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
			t.Errorf("close ManagementDashboard OCI mock: %v", err)
		}
	})
	sdkClient := managementdashboardsdk.DashxApisClient{BaseClient: session.BaseClient()}
	client := newManagementDashboardServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*managementdashboardv1beta1.ManagementDashboard]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *managementdashboardv1beta1.ManagementDashboard) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsShowDescription, current.Spec.IsShowDescription) ||
				!reflect.DeepEqual(current.Status.ProviderId, current.Spec.ProviderId) ||
				!reflect.DeepEqual(current.Status.ProviderName, current.Spec.ProviderName) ||
				!reflect.DeepEqual(current.Status.ProviderVersion, current.Spec.ProviderVersion) ||
				len(current.Status.Tiles) != 1 ||
				current.Status.Tiles[0].SavedSearchId != "<ocid:2>" {
				return fmt.Errorf("created ManagementDashboard status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *managementdashboardv1beta1.ManagementDashboard) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *managementdashboardv1beta1.ManagementDashboard) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsFavorite, current.Spec.IsFavorite) {
				return fmt.Errorf("updated ManagementDashboard status = %+v", current.Status)
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

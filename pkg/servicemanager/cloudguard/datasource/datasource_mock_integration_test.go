/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package datasource

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationDataSourceWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[cloudguardv1beta1.DataSource](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "datasource-sample",
    "namespace": "default",
    "uid": "datasource-uid"
  },
  "spec": {
    "compartmentId": "<ocid:1>",
    "dataSourceDetails": {
      "dataSourceFeedProvider": "LOGGINGQUERY",
      "intervalInMinutes": 5,
      "loggingQueryDetails": {
        "keyEntitiesCount": 2,
        "loggingQueryType": "INSIGHT"
      },
      "loggingQueryType": "INSIGHT",
      "operator": "GREATER",
      "query": "search eventName",
      "queryStartTime": {
        "startPolicyType": "NO_DELAY_START_POLICY"
      },
      "regions": [
        "us-ashburn-1"
      ],
      "threshold": 10
    },
    "dataSourceFeedProvider": "LOGGINGQUERY",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "displayName": "runtime-datasource",
    "freeformTags": {
      "env": "dev"
    },
    "status": "ENABLED"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-datasource")
	resource.Status = cloudguardv1beta1.DataSourceStatus{}
	createRequest := ocimock.MustJSONFixture[cloudguardsdk.CreateDataSourceDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "dataSourceDetails": {
    "dataSourceFeedProvider": "LOGGINGQUERY",
    "intervalInMinutes": 5,
    "loggingQueryDetails": {
      "keyEntitiesCount": 2,
      "loggingQueryType": "INSIGHT"
    },
    "loggingQueryType": "INSIGHT",
    "operator": "GREATER",
    "query": "search eventName",
    "queryStartTime": {
      "startPolicyType": "NO_DELAY_START_POLICY"
    },
    "regions": [
      "us-ashburn-1"
    ],
    "threshold": 10
  },
  "dataSourceFeedProvider": "LOGGINGQUERY",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "runtime-datasource",
  "freeformTags": {
    "env": "dev"
  },
  "status": "ENABLED"
}
`)
	updateRequest := ocimock.MustJSONFixture[cloudguardsdk.UpdateDataSourceDetails](t, `
{
  "displayName": "runtime-datasource-updated"
}
`)
	createdState := ocimock.MustOCIResponseFixture[cloudguardsdk.DataSource](t, `
{
  "compartmentId": "<ocid:1>",
  "dataSourceDetails": {
    "additionalEntitiesCount": null,
    "dataSourceFeedProvider": "LOGGINGQUERY",
    "intervalInMinutes": 5,
    "loggingQueryDetails": {
      "keyEntitiesCount": 2,
      "loggingQueryType": "INSIGHT"
    },
    "loggingQueryType": "INSIGHT",
    "operator": "GREATER",
    "query": "search eventName",
    "queryStartTime": {
      "startPolicyType": "NO_DELAY_START_POLICY"
    },
    "regions": [
      "us-ashburn-1"
    ],
    "threshold": 10
  },
  "dataSourceDetectorMappingInfo": null,
  "dataSourceFeedProvider": "LOGGINGQUERY",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "runtime-datasource",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "locks": null,
  "regionStatusDetail": null,
  "status": "ENABLED",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null
}
`)
	updatedState := ocimock.MustOCIResponseFixture[cloudguardsdk.DataSource](t, `
{
  "compartmentId": "<ocid:1>",
  "dataSourceDetails": {
    "additionalEntitiesCount": null,
    "dataSourceFeedProvider": "LOGGINGQUERY",
    "intervalInMinutes": 5,
    "loggingQueryDetails": {
      "keyEntitiesCount": 2,
      "loggingQueryType": "INSIGHT"
    },
    "loggingQueryType": "INSIGHT",
    "operator": "GREATER",
    "query": "search eventName",
    "queryStartTime": {
      "startPolicyType": "NO_DELAY_START_POLICY"
    },
    "regions": [
      "us-ashburn-1"
    ],
    "threshold": 10
  },
  "dataSourceDetectorMappingInfo": null,
  "dataSourceFeedProvider": "LOGGINGQUERY",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "runtime-datasource-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "locks": null,
  "regionStatusDetail": null,
  "status": "ENABLED",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null
}
`)
	deletedState := ocimock.MustOCIResponseFixture[cloudguardsdk.DataSource](t, `
{
  "compartmentId": "<ocid:1>",
  "dataSourceDetails": {
    "additionalEntitiesCount": null,
    "dataSourceFeedProvider": "LOGGINGQUERY",
    "intervalInMinutes": 5,
    "loggingQueryDetails": {
      "keyEntitiesCount": 2,
      "loggingQueryType": "INSIGHT"
    },
    "loggingQueryType": "INSIGHT",
    "operator": "GREATER",
    "query": "search eventName",
    "queryStartTime": {
      "startPolicyType": "NO_DELAY_START_POLICY"
    },
    "regions": [
      "us-ashburn-1"
    ],
    "threshold": 10
  },
  "dataSourceDetectorMappingInfo": null,
  "dataSourceFeedProvider": "LOGGINGQUERY",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "runtime-datasource-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:3>",
  "lifecycleState": "DELETED",
  "locks": null,
  "regionStatusDetail": null,
  "status": "ENABLED",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[cloudguardsdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "DataSource",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[cloudguardsdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "DataSource",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[cloudguardsdk.DataSource, cloudguardsdk.CreateDataSourceDetails, cloudguardsdk.UpdateDataSourceDetails]{
		CollectionPath: "/20200131/dataSources", ItemPath: "/20200131/dataSources/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		ValidateCreate: func(request ocimock.Request, _ cloudguardsdk.CreateDataSourceDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200131/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200131/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudguard.mock.invalid", BasePath: "20200131", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := cloudguardsdk.CloudGuardClient{BaseClient: session.BaseClient()}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	manager := &DataSourceServiceManager{Log: log}
	hooks := newDataSourceRuntimeHooksWithOCIClient(sdkClient)
	applyDataSourceRuntimeHooksWithWorkRequestClient(manager, &hooks, sdkClient, nil)
	client := wrapDataSourceGeneratedClient(hooks, defaultDataSourceServiceClient{ServiceClient: generatedruntime.NewServiceClient[*cloudguardv1beta1.DataSource](buildDataSourceGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*cloudguardv1beta1.DataSource]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *cloudguardv1beta1.DataSource) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DataSource status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *cloudguardv1beta1.DataSource) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "runtime-datasource-updated"
}`)
		},
		ValidateUpdated: func(current *cloudguardv1beta1.DataSource) error {
			if !(current.Status.DisplayName == "runtime-datasource-updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DataSource status = %+v", current.Status)
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

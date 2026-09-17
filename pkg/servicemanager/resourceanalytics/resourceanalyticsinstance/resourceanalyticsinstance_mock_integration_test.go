/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package resourceanalyticsinstance

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	resourceanalyticssdk "github.com/oracle/oci-go-sdk/v65/resourceanalytics"
	resourceanalyticsv1beta1 "github.com/oracle/oci-service-operator/api/resourceanalytics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationResourceAnalyticsInstanceWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[resourceanalyticsv1beta1.ResourceAnalyticsInstance](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "rai",
    "namespace": "default"
  },
  "spec": {
    "adwAdminPassword": {
      "password": "ValidPassword1",
      "passwordType": "PLAIN_TEXT"
    },
    "compartmentId": "compartment-1",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "description": "analytics instance",
    "displayName": "resource-analytics",
    "freeformTags": {
      "env": "test"
    },
    "isMutualTlsRequired": true,
    "licenseModel": "LICENSE_INCLUDED",
    "nsgIds": [
      "nsg-2",
      "nsg-1"
    ],
    "subnetId": "subnet-1"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-resourceanalyticsinstance")
	resource.Status = resourceanalyticsv1beta1.ResourceAnalyticsInstanceStatus{}
	createRequest := ocimock.MustJSONFixture[resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails](t, `
{
  "adwAdminPassword": {
    "password": "ValidPassword1",
    "passwordType": "PLAIN_TEXT"
  },
  "compartmentId": "compartment-1",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "analytics instance",
  "displayName": "resource-analytics",
  "freeformTags": {
    "env": "test"
  },
  "isMutualTlsRequired": true,
  "licenseModel": "LICENSE_INCLUDED",
  "nsgIds": [
    "nsg-2",
    "nsg-1"
  ],
  "subnetId": "subnet-1"
}
`)
	updateRequest := ocimock.MustJSONFixture[resourceanalyticssdk.UpdateResourceAnalyticsInstanceDetails](t, `
{
  "description": "updated analytics instance"
}
`)
	createdState := ocimock.MustOCIResponseFixture[resourceanalyticssdk.ResourceAnalyticsInstance](t, `
{
  "adwAdminPassword": {
    "password": "ValidPassword1",
    "passwordType": "PLAIN_TEXT"
  },
  "compartmentId": "compartment-1",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "analytics instance",
  "displayName": "resource-analytics",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "isMutualTlsRequired": true,
  "key": "<ocid:1>",
  "licenseModel": "LICENSE_INCLUDED",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "nsg-2",
    "nsg-1"
  ],
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "subnetId": "subnet-1",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[resourceanalyticssdk.ResourceAnalyticsInstance](t, `
{
  "adwAdminPassword": {
    "password": "ValidPassword1",
    "passwordType": "PLAIN_TEXT"
  },
  "compartmentId": "compartment-1",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated analytics instance",
  "displayName": "resource-analytics",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "isMutualTlsRequired": true,
  "key": "<ocid:1>",
  "licenseModel": "LICENSE_INCLUDED",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "nsg-2",
    "nsg-1"
  ],
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "subnetId": "subnet-1",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[resourceanalyticssdk.ResourceAnalyticsInstance](t, `
{
  "adwAdminPassword": {
    "password": "ValidPassword1",
    "passwordType": "PLAIN_TEXT"
  },
  "compartmentId": "compartment-1",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated analytics instance",
  "displayName": "resource-analytics",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "isMutualTlsRequired": true,
  "key": "<ocid:1>",
  "licenseModel": "LICENSE_INCLUDED",
  "lifecycleState": "DELETED",
  "nsgIds": [
    "nsg-2",
    "nsg-1"
  ],
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "subnetId": "subnet-1",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[resourceanalyticssdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_RESOURCE_ANALYTICS_INSTANCE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "ResourceAnalyticsInstance",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[resourceanalyticssdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_RESOURCE_ANALYTICS_INSTANCE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "ResourceAnalyticsInstance",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[resourceanalyticssdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_RESOURCE_ANALYTICS_INSTANCE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "ResourceAnalyticsInstance",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[resourceanalyticssdk.ResourceAnalyticsInstance, resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails, resourceanalyticssdk.UpdateResourceAnalyticsInstanceDetails]{
		CollectionPath: "/20241031/resourceAnalyticsInstances", ItemPath: "/20241031/resourceAnalyticsInstances/<ocid:1>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20241031/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20241031/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20241031/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://resourceanalytics.mock.invalid", BasePath: "20241031", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := resourceanalyticssdk.ResourceAnalyticsInstanceClient{BaseClient: session.BaseClient()}
	manager := &ResourceAnalyticsInstanceServiceManager{}
	hooks := newResourceAnalyticsInstanceRuntimeHooksWithOCIClient(sdkClient)
	applyResourceAnalyticsInstanceRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapResourceAnalyticsInstanceGeneratedClient(hooks, defaultResourceAnalyticsInstanceServiceClient{ServiceClient: generatedruntime.NewServiceClient[*resourceanalyticsv1beta1.ResourceAnalyticsInstance](buildResourceAnalyticsInstanceGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*resourceanalyticsv1beta1.ResourceAnalyticsInstance]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *resourceanalyticsv1beta1.ResourceAnalyticsInstance) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ResourceAnalyticsInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *resourceanalyticsv1beta1.ResourceAnalyticsInstance) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "updated analytics instance"
}`)
		},
		ValidateUpdated: func(current *resourceanalyticsv1beta1.ResourceAnalyticsInstance) error {
			if !(current.Status.Description == "updated analytics instance") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ResourceAnalyticsInstance status = %+v", current.Status)
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

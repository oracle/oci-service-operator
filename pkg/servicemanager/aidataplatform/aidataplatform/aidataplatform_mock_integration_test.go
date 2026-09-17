/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package aidataplatform

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	aidataplatformsdk "github.com/oracle/oci-go-sdk/v65/aidataplatform"
	aidataplatformv1beta1 "github.com/oracle/oci-service-operator/api/aidataplatform/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationAiDataPlatformWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[aidataplatformv1beta1.AiDataPlatform](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "aidp-sample",
    "namespace": "default",
    "uid": "aidp-uid"
  },
  "spec": {
    "aiDataPlatformType": "DATA_LAKE",
    "compartmentId": "<ocid:1>",
    "defaultWorkspaceName": "workspace-default",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "displayName": "aidp-sample",
    "freeformTags": {
      "managed-by": "osok"
    },
    "systemTags": {
      "orcl-cloud": {
        "free-tier-retained": "true"
      }
    }
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-aidataplatform")
	resource.Status = aidataplatformv1beta1.AiDataPlatformStatus{}
	createRequest := ocimock.MustJSONFixture[aidataplatformsdk.CreateAiDataPlatformDetails](t, `
{
  "aiDataPlatformType": "DATA_LAKE",
  "compartmentId": "<ocid:1>",
  "defaultWorkspaceName": "workspace-default",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "aidp-sample",
  "freeformTags": {
    "managed-by": "osok"
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[aidataplatformsdk.UpdateAiDataPlatformDetails](t, `
{
  "displayName": "aidp-updated"
}
`)
	createdState := ocimock.MustOCIResponseFixture[aidataplatformsdk.AiDataPlatform](t, `
{
  "aiDataPlatformType": "DATA_LAKE",
  "aliasKey": null,
  "compartmentId": "<ocid:1>",
  "createdBy": null,
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "aidp-sample",
  "freeformTags": {
    "managed-by": "osok"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": null,
  "timeUpdated": null,
  "webSocketEndpoint": null
}
`)
	updatedState := ocimock.MustOCIResponseFixture[aidataplatformsdk.AiDataPlatform](t, `
{
  "aiDataPlatformType": "DATA_LAKE",
  "aliasKey": null,
  "compartmentId": "<ocid:1>",
  "createdBy": null,
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "aidp-updated",
  "freeformTags": {
    "managed-by": "osok"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": null,
  "timeUpdated": null,
  "webSocketEndpoint": null
}
`)
	deletedState := ocimock.MustOCIResponseFixture[aidataplatformsdk.AiDataPlatform](t, `
{
  "aiDataPlatformType": "DATA_LAKE",
  "aliasKey": null,
  "compartmentId": "<ocid:1>",
  "createdBy": null,
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "aidp-updated",
  "freeformTags": {
    "managed-by": "osok"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": null,
  "timeUpdated": null,
  "webSocketEndpoint": null
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[aidataplatformsdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "AiDataPlatform",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[aidataplatformsdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "AiDataPlatform",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[aidataplatformsdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "AiDataPlatform",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[aidataplatformsdk.AiDataPlatform, aidataplatformsdk.CreateAiDataPlatformDetails, aidataplatformsdk.UpdateAiDataPlatformDetails]{
		CollectionPath: "/20240831/aiDataPlatforms", ItemPath: "/20240831/aiDataPlatforms/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ aidataplatformsdk.CreateAiDataPlatformDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20240831/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20240831/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20240831/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://aidataplatform.mock.invalid", BasePath: "20240831", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := aidataplatformsdk.AiDataPlatformClient{BaseClient: session.BaseClient()}
	client := newAiDataPlatformServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*aidataplatformv1beta1.AiDataPlatform]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *aidataplatformv1beta1.AiDataPlatform) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created AiDataPlatform status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *aidataplatformv1beta1.AiDataPlatform) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "aidp-updated"
}`)
		},
		ValidateUpdated: func(current *aidataplatformv1beta1.AiDataPlatform) error {
			if !(current.Status.DisplayName == "aidp-updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated AiDataPlatform status = %+v", current.Status)
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

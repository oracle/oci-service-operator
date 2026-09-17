/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package exadatainsight

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationExadataInsightWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.ExadataInsight](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "exadata",
    "namespace": "default"
  },
  "spec": {
    "compartmentId": "compartment",
    "definedTags": {
      "ops": {
        "tier": "silver"
      }
    },
    "enterpriseManagerBridgeId": "bridge",
    "enterpriseManagerEntityIdentifier": "em-entity",
    "enterpriseManagerIdentifier": "em",
    "entitySource": "EM_MANAGED_EXTERNAL_EXADATA",
    "freeformTags": {
      "env": "test"
    },
    "isAutoSyncEnabled": false
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-exadatainsight")
	resource.Status = opsiv1beta1.ExadataInsightStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateEmManagedExternalExadataInsightDetails](t, `
{
  "compartmentId": "compartment",
  "definedTags": {
    "ops": {
      "tier": "silver"
    }
  },
  "enterpriseManagerBridgeId": "bridge",
  "enterpriseManagerEntityIdentifier": "em-entity",
  "enterpriseManagerIdentifier": "em",
  "freeformTags": {
    "env": "test"
  },
  "isAutoSyncEnabled": false
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateEmManagedExternalExadataInsightDetails](t, `
{
  "isAutoSyncEnabled": true
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.EmManagedExternalExadataInsight](t, `
{
  "compartmentId": "compartment",
  "definedTags": {
    "ops": {
      "tier": "silver"
    }
  },
  "enterpriseManagerBridgeId": "bridge",
  "enterpriseManagerEntityIdentifier": "em-entity",
  "enterpriseManagerIdentifier": "em",
  "entitySource": "EM_MANAGED_EXTERNAL_EXADATA",
  "exadataInfraId": "<binding:exadata-infrastructure>",
  "exadataName": "mock-exadata",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "status": "ENABLED",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.EmManagedExternalExadataInsight](t, `
{
  "compartmentId": "compartment",
  "definedTags": {
    "ops": {
      "tier": "silver"
    }
  },
  "enterpriseManagerBridgeId": "bridge",
  "enterpriseManagerEntityIdentifier": "em-entity",
  "enterpriseManagerIdentifier": "em",
  "entitySource": "EM_MANAGED_EXTERNAL_EXADATA",
  "exadataInfraId": "<binding:exadata-infrastructure>",
  "exadataName": "mock-exadata",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "isAutoSyncEnabled": true,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "status": "ENABLED",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[opsisdk.EmManagedExternalExadataInsight](t, `
{
  "compartmentId": "compartment",
  "definedTags": {
    "ops": {
      "tier": "silver"
    }
  },
  "enterpriseManagerBridgeId": "bridge",
  "enterpriseManagerEntityIdentifier": "em-entity",
  "enterpriseManagerIdentifier": "em",
  "entitySource": "EM_MANAGED_EXTERNAL_EXADATA",
  "exadataInfraId": "<binding:exadata-infrastructure>",
  "exadataName": "mock-exadata",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "isAutoSyncEnabled": true,
  "key": "<ocid:1>",
  "lifecycleState": "DELETED",
  "resourceId": "<ocid:1>",
  "status": "ENABLED",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_EXADATA_INSIGHT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "ExadataInsight",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_EXADATA_INSIGHT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "ExadataInsight",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_EXADATA_INSIGHT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "ExadataInsight",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.EmManagedExternalExadataInsight, opsisdk.CreateEmManagedExternalExadataInsightDetails, opsisdk.UpdateEmManagedExternalExadataInsightDetails]{
		CollectionPath: "/20200630/exadataInsights", ItemPath: "/20200630/exadataInsights/<ocid:1>",
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "entitySource", "EM_MANAGED_EXTERNAL_EXADATA", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "entitySource", "EM_MANAGED_EXTERNAL_EXADATA", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://opsi.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := opsisdk.OperationsInsightsClient{BaseClient: session.BaseClient()}
	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.ExadataInsight]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.ExadataInsight) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.EntitySource != resource.Spec.EntitySource || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ExadataInsight status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.ExadataInsight) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "isAutoSyncEnabled": true
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.ExadataInsight) error {
			if !(current.Status.IsAutoSyncEnabled) || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ExadataInsight status = %+v", current.Status)
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

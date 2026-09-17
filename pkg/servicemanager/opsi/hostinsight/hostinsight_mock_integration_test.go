/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package hostinsight

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
func TestMockIntegrationHostInsightWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.HostInsight](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "host-insight",
    "namespace": "default",
    "uid": "host-insight-uid"
  },
  "spec": {
    "compartmentId": "<ocid:1>",
    "computeId": "<ocid:2>",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "entitySource": "MACS_MANAGED_CLOUD_HOST",
    "freeformTags": {
      "env": "dev"
    }
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-hostinsight")
	resource.Status = opsiv1beta1.HostInsightStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateMacsManagedCloudHostInsightDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "computeId": "<ocid:2>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "dev"
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateMacsManagedCloudHostInsightDetails](t, `
{
  "freeformTags": {
    "env": "dev",
    "mock": "updated"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.MacsManagedCloudHostInsight](t, `
{
  "compartmentId": "<ocid:1>",
  "computeId": "<ocid:2>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "entitySource": "MACS_MANAGED_CLOUD_HOST",
  "freeformTags": {
    "env": "dev"
  },
  "hostName": "mock-host",
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "status": "ENABLED"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.MacsManagedCloudHostInsight](t, `
{
  "compartmentId": "<ocid:1>",
  "computeId": "<ocid:2>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "entitySource": "MACS_MANAGED_CLOUD_HOST",
  "freeformTags": {
    "mock": "updated"
  },
  "hostName": "mock-host",
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "status": "ENABLED"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[opsisdk.MacsManagedCloudHostInsight](t, `
{
  "compartmentId": "<ocid:1>",
  "computeId": "<ocid:2>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "entitySource": "MACS_MANAGED_CLOUD_HOST",
  "freeformTags": {
    "mock": "updated"
  },
  "hostName": "mock-host",
  "id": "<ocid:4>",
  "lifecycleState": "DELETED",
  "status": "ENABLED"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_HOST_INSIGHT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "HostInsight",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_HOST_INSIGHT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "HostInsight",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_HOST_INSIGHT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "HostInsight",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.MacsManagedCloudHostInsight, opsisdk.CreateMacsManagedCloudHostInsightDetails, opsisdk.UpdateMacsManagedCloudHostInsightDetails]{
		CollectionPath: "/20200630/hostInsights", ItemPath: "/20200630/hostInsights/<ocid:4>",
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
			return ocimock.ValidateDiscriminatedJSONRequest(request, "entitySource", "MACS_MANAGED_CLOUD_HOST", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "entitySource", "MACS_MANAGED_CLOUD_HOST", updateRequest)
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
	client := newHostInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.HostInsight]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.HostInsight) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.EntitySource != resource.Spec.EntitySource || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created HostInsight status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.HostInsight) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "freeformTags": {
    "mock": "updated"
  }
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.HostInsight) error {
			if !(current.Status.FreeformTags["mock"] == "updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated HostInsight status = %+v", current.Status)
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

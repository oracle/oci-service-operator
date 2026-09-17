/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package clusterplacementgroup

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	clusterplacementgroupssdk "github.com/oracle/oci-go-sdk/v65/clusterplacementgroups"
	clusterplacementgroupsv1beta1 "github.com/oracle/oci-service-operator/api/clusterplacementgroups/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationClusterPlacementGroupWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &clusterplacementgroupsv1beta1.ClusterPlacementGroup{}
	ocimock.InitializeResource(resource, "mock-clusterplacementgroup")
	resource.Spec = ocimock.MustJSONFixture[clusterplacementgroupsv1beta1.ClusterPlacementGroupSpec](t, `{
  "availabilityDomain": "<binding:availability-domain>",
  "clusterPlacementGroupType": "STANDARD",
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create"
  },
  "name": "osok-mock-async-cpg-v1"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)

	createRequest := ocimock.MustJSONFixture[clusterplacementgroupssdk.CreateClusterPlacementGroupDetails](t, `{
  "availabilityDomain": "<binding:availability-domain>",
  "clusterPlacementGroupType": "STANDARD",
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create"
  },
  "name": "osok-mock-async-cpg-v1"
}`)
	updateRequest := ocimock.MustJSONFixture[clusterplacementgroupssdk.UpdateClusterPlacementGroupDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[clusterplacementgroupssdk.ClusterPlacementGroup](t, `{
  "availabilityDomain": "<binding:availability-domain>",
  "capabilities": null,
  "clusterPlacementGroupType": "STANDARD",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-08-31T21:05:59.530Z"
    }
  },
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "name": "osok-mock-async-cpg-v1",
  "placementInstruction": null,
  "systemTags": {
  },
  "timeCreated": "2026-08-31T21:05:59.742Z",
  "timeUpdated": "2026-08-31T21:05:59.742Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[clusterplacementgroupssdk.ClusterPlacementGroup](t, `{
  "availabilityDomain": "<binding:availability-domain>",
  "capabilities": null,
  "clusterPlacementGroupType": "STANDARD",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-08-31T21:05:59.530Z"
    }
  },
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "name": "osok-mock-async-cpg-v1",
  "placementInstruction": null,
  "systemTags": {
  },
  "timeCreated": "2026-08-31T21:05:59.742Z",
  "timeUpdated": "2026-08-31T21:06:00.595Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[clusterplacementgroupssdk.ClusterPlacementGroup](t, `{
  "availabilityDomain": "<binding:availability-domain>",
  "capabilities": null,
  "clusterPlacementGroupType": "STANDARD",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-08-31T21:05:59.530Z"
    }
  },
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "name": "osok-mock-async-cpg-v1",
  "placementInstruction": null,
  "systemTags": {
  },
  "timeCreated": "2026-08-31T21:05:59.742Z",
  "timeUpdated": "2026-08-31T21:06:03.301Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[clusterplacementgroupssdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_CLUSTER_PLACEMENT_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "clusterplacementgroup",
      "entityUri": "/clusterPlacementGroups/<ocid:3>",
      "identifier": "<ocid:3>",
      "metadata": {
        "is_dry_run": "false"
      }
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-08-31T21:05:59.743Z",
  "timeFinished": "2026-08-31T21:05:59.749Z",
  "timeStarted": "2026-08-31T21:05:59.749Z",
  "timeUpdated": "2026-08-31T21:05:59.749Z"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[clusterplacementgroupssdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_CLUSTER_PLACEMENT_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "clusterplacementgroup",
      "entityUri": "/clusterPlacementGroups/<ocid:3>",
      "identifier": "<ocid:3>",
      "metadata": {
        "is_dry_run": "false"
      }
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-08-31T21:06:00.605Z",
  "timeFinished": "2026-08-31T21:06:00.613Z",
  "timeStarted": "2026-08-31T21:06:00.613Z",
  "timeUpdated": "2026-08-31T21:06:00.613Z"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[clusterplacementgroupssdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_CLUSTER_PLACEMENT_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "clusterplacementgroup",
      "entityUri": "/clusterPlacementGroups/<ocid:3>",
      "identifier": "<ocid:3>",
      "metadata": {
        "is_dry_run": "false"
      }
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-08-31T21:06:01.283Z",
  "timeFinished": "2026-08-31T21:06:03.368Z",
  "timeStarted": "2026-08-31T21:06:03.368Z",
  "timeUpdated": "2026-08-31T21:06:03.368Z"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[clusterplacementgroupssdk.ClusterPlacementGroup, clusterplacementgroupssdk.CreateClusterPlacementGroupDetails, clusterplacementgroupssdk.UpdateClusterPlacementGroupDetails]{
		CollectionPath:    "/20230801/clusterPlacementGroups",
		ItemPath:          "/20230801/clusterPlacementGroups/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		ListShape:         ocimock.ListShapeItems,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		DeletedState:      &deletedState, RequireDeleteRead: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ clusterplacementgroupssdk.CreateClusterPlacementGroupDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20230801/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20230801/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20230801/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://clusterplacementgroups.us-ashburn-1.oci.oraclecloud.com", BasePath: "20230801", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := clusterplacementgroupssdk.ClusterPlacementGroupsCPClient{BaseClient: session.BaseClient()}
	client := newClusterPlacementGroupServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*clusterplacementgroupsv1beta1.ClusterPlacementGroup]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *clusterplacementgroupsv1beta1.ClusterPlacementGroup) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ClusterPlacementGroup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *clusterplacementgroupsv1beta1.ClusterPlacementGroup) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *clusterplacementgroupsv1beta1.ClusterPlacementGroup) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ClusterPlacementGroup status = %+v", current.Status)
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

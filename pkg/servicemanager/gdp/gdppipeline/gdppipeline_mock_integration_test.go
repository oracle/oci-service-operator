/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package gdppipeline

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	gdpsdk "github.com/oracle/oci-go-sdk/v65/gdp"
	gdpv1beta1 "github.com/oracle/oci-service-operator/api/gdp/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationGdpPipelineWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &gdpv1beta1.GdpPipeline{}
	ocimock.InitializeResource(resource, "mock-gdppipeline")
	resource.Spec = ocimock.MustJSONFixture[gdpv1beta1.GdpPipelineSpec](t, `{
  "approvalKeyVaultId": "<ocid:1>",
  "authorizationDetails": "<redacted>",
  "bucketDetails": [
    {
      "bucketType": "SOURCE",
      "id": "<ocid:2>",
      "name": "source-bucket",
      "namespace": "namespace-a"
    },
    {
      "bucketType": "TRANSFER",
      "id": "<ocid:3>",
      "name": "transfer-bucket",
      "namespace": "namespace-a"
    }
  ],
  "compartmentId": "<ocid:4>",
  "definedTags": {
    "operations": {
      "owner": "team-a"
    }
  },
  "description": "pipeline description",
  "displayName": "osok-gdp-pipeline",
  "fileTypes": [
    ".pdf",
    ".xml"
  ],
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "isApprovalNeeded": true,
  "isChunkingEnabled": true,
  "isFileOverrideInDestinationEnabled": true,
  "isScanningEnabled": true,
  "peeringRegion": "us-phoenix-1",
  "pipelineType": "SENDER",
  "serviceLogGroupId": "<ocid:5>"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "pipeline description updated"
}`)

	createRequest := ocimock.MustJSONFixture[gdpsdk.CreateGdpPipelineDetails](t, `{
  "approvalKeyVaultId": "<ocid:1>",
  "authorizationDetails": "<redacted>",
  "bucketDetails": [
    {
      "bucketType": "SOURCE",
      "id": "<ocid:2>",
      "name": "source-bucket",
      "namespace": "namespace-a"
    },
    {
      "bucketType": "TRANSFER",
      "id": "<ocid:3>",
      "name": "transfer-bucket",
      "namespace": "namespace-a"
    }
  ],
  "compartmentId": "<ocid:4>",
  "definedTags": {
    "operations": {
      "owner": "team-a"
    }
  },
  "description": "pipeline description",
  "displayName": "osok-gdp-pipeline",
  "fileTypes": [
    ".pdf",
    ".xml"
  ],
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "isApprovalNeeded": true,
  "isChunkingEnabled": true,
  "isFileOverrideInDestinationEnabled": true,
  "isScanningEnabled": true,
  "peeringRegion": "us-phoenix-1",
  "pipelineType": "SENDER",
  "serviceLogGroupId": "<ocid:5>"
}`)
	updateRequest := ocimock.MustJSONFixture[gdpsdk.UpdateGdpPipelineDetails](t, `{
  "description": "pipeline description updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[gdpsdk.GdpPipeline](t, `{
  "approvalKeyVaultId": "<ocid:1>",
  "authorizationDetails": "<redacted>",
  "bucketDetails": [
    {
      "bucketType": "SOURCE",
      "id": "<ocid:2>",
      "name": "source-bucket",
      "namespace": "namespace-a"
    },
    {
      "bucketType": "TRANSFER",
      "id": "<ocid:3>",
      "name": "transfer-bucket",
      "namespace": "namespace-a"
    }
  ],
  "compartmentId": "<ocid:4>",
  "definedTags": {
    "operations": {
      "owner": "team-a"
    }
  },
  "description": "pipeline description",
  "displayName": "osok-gdp-pipeline",
  "fileTypes": [
    ".pdf",
    ".xml"
  ],
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "id": "<ocid:7>",
  "isApprovalNeeded": true,
  "isChunkingEnabled": true,
  "isFileOverrideInDestinationEnabled": true,
  "isScanningEnabled": true,
  "lifecycleDetails": "lifecycle details",
  "lifecycleState": "ACTIVE",
  "peeredGdpPipelineId": "<ocid:8>",
  "peeringRegion": "us-phoenix-1",
  "pipelineType": "SENDER",
  "serviceLogGroupId": "<ocid:5>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-05-06T12:00:00Z",
  "timeUpdated": "2026-05-06T13:00:00Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[gdpsdk.GdpPipeline](t, `{
  "approvalKeyVaultId": "<ocid:1>",
  "authorizationDetails": "<redacted>",
  "bucketDetails": [
    {
      "bucketType": "SOURCE",
      "id": "<ocid:2>",
      "name": "source-bucket",
      "namespace": "namespace-a"
    },
    {
      "bucketType": "TRANSFER",
      "id": "<ocid:3>",
      "name": "transfer-bucket",
      "namespace": "namespace-a"
    }
  ],
  "compartmentId": "<ocid:4>",
  "definedTags": {
    "operations": {
      "owner": "team-a"
    }
  },
  "description": "pipeline description updated",
  "displayName": "osok-gdp-pipeline",
  "fileTypes": [
    ".pdf",
    ".xml"
  ],
  "freeformTags": {
    "managed-by": "oci-service-operator"
  },
  "id": "<ocid:7>",
  "isApprovalNeeded": true,
  "isChunkingEnabled": true,
  "isFileOverrideInDestinationEnabled": true,
  "isScanningEnabled": true,
  "lifecycleDetails": "lifecycle details",
  "lifecycleState": "ACTIVE",
  "peeredGdpPipelineId": "<ocid:8>",
  "peeringRegion": "us-phoenix-1",
  "pipelineType": "SENDER",
  "serviceLogGroupId": "<ocid:5>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-05-06T12:00:00Z",
  "timeUpdated": "2026-05-06T13:00:00Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[gdpsdk.GdpWorkRequest](t, `{
  "compartmentId": "<ocid:4>",
  "id": "<ocid:6>",
  "operationType": "CREATE_GDP_PIPELINE",
  "percentComplete": 42,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "GdpPipeline",
      "entityUri": null,
      "identifier": "<ocid:7>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-05-06T12:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[gdpsdk.GdpWorkRequest](t, `{
  "compartmentId": "<ocid:4>",
  "id": "wr-update",
  "operationType": "UPDATE_GDP_PIPELINE",
  "percentComplete": 42,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "GdpPipeline",
      "entityUri": null,
      "identifier": "<ocid:7>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-05-06T12:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[gdpsdk.GdpWorkRequest](t, `{
  "compartmentId": "<ocid:4>",
  "id": "<ocid:9>",
  "operationType": "DELETE_GDP_PIPELINE",
  "percentComplete": 42,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "GdpPipeline",
      "entityUri": null,
      "identifier": "<ocid:7>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-05-06T12:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[gdpsdk.GdpPipeline, gdpsdk.CreateGdpPipelineDetails, gdpsdk.UpdateGdpPipelineDetails]{
		CollectionPath:     "/20230301/gdpPipelines",
		ItemPath:           "/20230301/gdpPipelines/<ocid:7>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		ListShape:          ocimock.ListShapeItems,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:9>"}},
		ValidateCreate: func(request ocimock.Request, _ gdpsdk.CreateGdpPipelineDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20230301/gdpWorkRequests/<ocid:6>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20230301/gdpWorkRequests/wr-update", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20230301/gdpWorkRequests/<ocid:9>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://gdp.us-ashburn-1.oci.oraclecloud.com", BasePath: "20230301", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := gdpsdk.GuardedDataPipelineClient{BaseClient: session.BaseClient()}
	client := newGdpPipelineServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*gdpv1beta1.GdpPipeline]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *gdpv1beta1.GdpPipeline) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created GdpPipeline status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *gdpv1beta1.GdpPipeline) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *gdpv1beta1.GdpPipeline) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated GdpPipeline status = %+v", current.Status)
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

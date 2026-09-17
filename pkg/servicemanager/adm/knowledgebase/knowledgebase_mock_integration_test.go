/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package knowledgebase

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	admsdk "github.com/oracle/oci-go-sdk/v65/adm"
	admv1beta1 "github.com/oracle/oci-service-operator/api/adm/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationKnowledgeBaseWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &admv1beta1.KnowledgeBase{}
	ocimock.InitializeResource(resource, "mock-knowledgebase")
	resource.Spec = ocimock.MustJSONFixture[admv1beta1.KnowledgeBaseSpec](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-async-adm-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-async-adm-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)

	createRequest := ocimock.MustJSONFixture[admsdk.CreateKnowledgeBaseDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-async-adm-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[admsdk.UpdateKnowledgeBaseDetails](t, `{
  "displayName": "osok-mock-async-adm-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[admsdk.KnowledgeBase](t, `{
  "auditLifecyclePolicies": null,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-08-31T21:00:50.820Z"
    }
  },
  "displayName": "osok-mock-async-adm-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-08-31T21:00:50.985Z",
  "timeUpdated": "2026-08-31T21:01:48.232Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[admsdk.KnowledgeBase](t, `{
  "auditLifecyclePolicies": null,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-08-31T21:00:50.820Z"
    }
  },
  "displayName": "osok-mock-async-adm-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-08-31T21:00:50.985Z",
  "timeUpdated": "2026-08-31T21:02:08.273Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[admsdk.KnowledgeBase](t, `{
  "auditLifecyclePolicies": null,
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-08-31T21:00:50.820Z"
    }
  },
  "displayName": "osok-mock-async-adm-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleState": "DELETED",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-08-31T21:00:50.985Z",
  "timeUpdated": "2026-08-31T21:02:13.512Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[admsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_KNOWLEDGE_BASE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "knowledgeBase",
      "entityUri": "/knowledgeBases/<ocid:3>",
      "identifier": "<ocid:3>",
      "metadata": {
      }
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-08-31T21:00:50.986Z",
  "timeFinished": "2026-08-31T21:01:48.250Z",
  "timeStarted": "2026-08-31T21:01:27.167Z"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[admsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_KNOWLEDGE_BASE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "knowledgeBase",
      "entityUri": "/knowledgeBases/<ocid:3>",
      "identifier": "<ocid:3>",
      "metadata": {
      }
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-08-31T21:01:53.798Z",
  "timeFinished": "2026-08-31T21:02:08.284Z",
  "timeStarted": "2026-08-31T21:01:53.802Z"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[admsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_KNOWLEDGE_BASE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "knowledgeBase",
      "entityUri": "/knowledgeBases/<ocid:3>",
      "identifier": "<ocid:3>",
      "metadata": {
      }
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-08-31T21:02:09.989Z",
  "timeFinished": "2026-08-31T21:02:13.517Z",
  "timeStarted": "2026-08-31T21:02:10.001Z"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[admsdk.KnowledgeBase, admsdk.CreateKnowledgeBaseDetails, admsdk.UpdateKnowledgeBaseDetails]{
		CollectionPath:    "/20220421/knowledgeBases",
		ItemPath:          "/20220421/knowledgeBases/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		ListShape:         ocimock.ListShapeItems,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		DeletedState:      &deletedState, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ admsdk.CreateKnowledgeBaseDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20220421/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20220421/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20220421/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://adm.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220421", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := admsdk.ApplicationDependencyManagementClient{BaseClient: session.BaseClient()}
	client := newKnowledgeBaseServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*admv1beta1.KnowledgeBase]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *admv1beta1.KnowledgeBase) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created KnowledgeBase status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *admv1beta1.KnowledgeBase) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *admv1beta1.KnowledgeBase) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated KnowledgeBase status = %+v", current.Status)
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

/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apmdomain

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	apmcontrolplanesdk "github.com/oracle/oci-go-sdk/v65/apmcontrolplane"
	apmcontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/apmcontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationApmDomainWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &apmcontrolplanev1beta1.ApmDomain{}
	ocimock.InitializeResource(resource, "mock-apmdomain")
	resource.Spec = ocimock.MustJSONFixture[apmcontrolplanev1beta1.ApmDomainSpec](t, `{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-apm-domain-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)

	createRequest := ocimock.MustJSONFixture[apmcontrolplanesdk.CreateApmDomainDetails](t, `{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-apm-domain-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[apmcontrolplanesdk.UpdateApmDomainDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[apmcontrolplanesdk.ApmDomain](t, `{
  "compartmentId": "<ocid:1>",
  "dataUploadEndpoint": "https://aaaadic6ok4kmaaaaaaaaahh2u.apm-agt.us-ashburn-1.oci.oraclecloud.com",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T19:29:31.731Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-apm-domain-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "isFreeTier": false,
  "lifecycleState": "ACTIVE",
  "logGroupId": null,
  "systemTags": {
  },
  "timeCreated": "2026-09-01T19:29:31.814Z",
  "timeUpdated": "2026-09-01T19:34:09.373Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[apmcontrolplanesdk.ApmDomain](t, `{
  "compartmentId": "<ocid:1>",
  "dataUploadEndpoint": "https://aaaadic6ok4kmaaaaaaaaahh2u.apm-agt.us-ashburn-1.oci.oraclecloud.com",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T19:29:31.731Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-apm-domain-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isFreeTier": false,
  "lifecycleState": "ACTIVE",
  "logGroupId": null,
  "systemTags": {
  },
  "timeCreated": "2026-09-01T19:29:31.814Z",
  "timeUpdated": "2026-09-01T19:34:39.192Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[apmcontrolplanesdk.ApmDomain](t, `{
  "compartmentId": "<ocid:1>",
  "dataUploadEndpoint": "https://aaaadic6ok4kmaaaaaaaaahh2u.apm-agt.us-ashburn-1.oci.oraclecloud.com",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T19:29:31.731Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-apm-domain-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "isFreeTier": false,
  "lifecycleState": "DELETED",
  "logGroupId": null,
  "systemTags": {
  },
  "timeCreated": "2026-09-01T19:29:31.814Z",
  "timeUpdated": "2026-09-01T19:36:53.873Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[apmcontrolplanesdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_APM_DOMAIN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "apmDomain",
      "entityUri": "/apmDomains/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T19:29:31.814Z",
  "timeFinished": "2026-09-01T19:34:09.373Z",
  "timeStarted": "2026-09-01T19:30:00.630Z"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[apmcontrolplanesdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_APM_DOMAIN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "apmDomain",
      "entityUri": "/apmDomains/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T19:34:11.341Z",
  "timeFinished": "2026-09-01T19:34:39.192Z",
  "timeStarted": "2026-09-01T19:34:39.024Z"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[apmcontrolplanesdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_APM_DOMAIN",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "apmDomain",
      "entityUri": "/apmDomains/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T19:34:43.906Z",
  "timeFinished": "2026-09-01T19:36:54.048Z",
  "timeStarted": "2026-09-01T19:35:23.614Z"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[apmcontrolplanesdk.ApmDomain, apmcontrolplanesdk.CreateApmDomainDetails, apmcontrolplanesdk.UpdateApmDomainDetails]{
		CollectionPath:    "/20200630/apmDomains",
		ItemPath:          "/20200630/apmDomains/<ocid:3>",
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
		ValidateCreate: func(request ocimock.Request, _ apmcontrolplanesdk.CreateApmDomainDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://apm-cp.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := apmcontrolplanesdk.ApmDomainClient{BaseClient: session.BaseClient()}
	client := newApmDomainServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apmcontrolplanev1beta1.ApmDomain]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apmcontrolplanev1beta1.ApmDomain) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ApmDomain status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apmcontrolplanev1beta1.ApmDomain) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *apmcontrolplanev1beta1.ApmDomain) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ApmDomain status = %+v", current.Status)
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

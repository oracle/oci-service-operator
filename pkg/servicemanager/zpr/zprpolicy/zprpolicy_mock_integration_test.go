/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package zprpolicy

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	zprsdk "github.com/oracle/oci-go-sdk/v65/zpr"
	zprv1beta1 "github.com/oracle/oci-service-operator/api/zpr/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: explicit SDK fixtures selected from the production
// service manager, reviewed formal lifecycle, and vendored SDK contract.
func TestMockIntegrationZprPolicyWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &zprv1beta1.ZprPolicy{}
	ocimock.InitializeResource(resource, "mock-zprpolicy")
	resource.Spec = ocimock.MustJSONFixture[zprv1beta1.ZprPolicySpec](t, `{
  "compartmentId": "<ocid:2>",
  "name": "policy-alpha",
  "description": "policy description",
  "statements": [
    "allow any-user to inspect zpr-policies in tenancy"
  ],
  "freeformTags": {
    "env": "dev"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "ZprPolicy updated"
}`)
	createRequest := ocimock.MustJSONFixture[zprsdk.CreateZprPolicyDetails](t, `{
  "compartmentId": "<ocid:2>",
  "name": "policy-alpha",
  "description": "policy description",
  "statements": [
    "allow any-user to inspect zpr-policies in tenancy"
  ],
  "freeformTags": {
    "env": "dev"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  }
}`)
	updateRequest := ocimock.MustJSONFixture[zprsdk.UpdateZprPolicyDetails](t, `{
  "description": "ZprPolicy updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[zprsdk.ZprPolicy](t, `{
  "compartmentId": "<ocid:2>",
  "name": "policy-alpha",
  "description": "policy description",
  "statements": [
    "allow any-user to inspect zpr-policies in tenancy"
  ],
  "freeformTags": {
    "env": "dev"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "lifecycleDetails": "ready",
  "id": "<ocid:1>",
  "lifecycleState": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[zprsdk.ZprPolicy](t, `{
  "compartmentId": "<ocid:2>",
  "name": "policy-alpha",
  "description": "ZprPolicy updated",
  "statements": [
    "allow any-user to inspect zpr-policies in tenancy"
  ],
  "freeformTags": {
    "env": "dev"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "lifecycleDetails": "ready",
  "id": "<ocid:1>",
  "lifecycleState": "ACTIVE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[zprsdk.WorkRequest](t, `{
  "id": "wr-create",
  "operationType": "CREATE_ZPR_POLICY",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:2>",
  "percentComplete": 100,
  "resources": [
    {
      "entityType": "zprpolicy",
      "actionType": "CREATED",
      "identifier": "<ocid:1>"
    }
  ]
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[zprsdk.WorkRequest](t, `{
  "id": "wr-update",
  "operationType": "UPDATE_ZPR_POLICY",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:2>",
  "percentComplete": 100,
  "resources": [
    {
      "entityType": "zprpolicy",
      "actionType": "UPDATED",
      "identifier": "<ocid:1>"
    }
  ]
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[zprsdk.WorkRequest](t, `{
  "id": "wr-delete",
  "operationType": "DELETE_ZPR_POLICY",
  "status": "SUCCEEDED",
  "compartmentId": "<ocid:2>",
  "percentComplete": 100,
  "resources": [
    {
      "entityType": "zprpolicy",
      "actionType": "DELETED",
      "identifier": "<ocid:1>"
    }
  ]
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[zprsdk.ZprPolicy, zprsdk.CreateZprPolicyDetails, zprsdk.UpdateZprPolicyDetails]{
		CollectionPath: "/20240301/zprPolicies", ItemPath: "/20240301/zprPolicies/<ocid:1>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: http.StatusAccepted, UpdateStatus: http.StatusAccepted, DeleteStatus: http.StatusAccepted, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-create"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-delete"}},
		ValidateCreate: func(request ocimock.Request, _ zprsdk.CreateZprPolicyDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20240301/zprPolicyWorkRequests/wr-create", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20240301/zprPolicyWorkRequests/wr-update", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20240301/zprPolicyWorkRequests/wr-delete", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://zpr.mock.invalid", BasePath: "20240301", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := zprsdk.ZprClient{BaseClient: session.BaseClient()}
	manager := &ZprPolicyServiceManager{Log: log}
	hooks := newZprPolicyDefaultRuntimeHooks(sdkClient)
	applyZprPolicyRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapZprPolicyGeneratedClient(hooks, defaultZprPolicyServiceClient{ServiceClient: generatedruntime.NewServiceClient[*zprv1beta1.ZprPolicy](buildZprPolicyGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*zprv1beta1.ZprPolicy]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *zprv1beta1.ZprPolicy) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ZprPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *zprv1beta1.ZprPolicy) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *zprv1beta1.ZprPolicy) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ZprPolicy status = %+v", current.Status)
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

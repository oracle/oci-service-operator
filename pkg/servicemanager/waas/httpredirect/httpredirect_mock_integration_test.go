/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package httpredirect

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type mockHttpRedirectOCIClient struct {
	waassdk.RedirectClient
	waassdk.WaasClient
}

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationHttpRedirectWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &waasv1beta1.HttpRedirect{}
	ocimock.InitializeResource(resource, "mock-httpredirect")
	resource.Spec = ocimock.MustJSONFixture[waasv1beta1.HttpRedirectSpec](t, `
{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-http-redirect",
  "domain": "osok-mock-redirect.example.com",
  "freeformTags": {
    "osok-mock": "create"
  },
  "responseCode": 301,
  "target": {
    "host": "www.example.com",
    "path": "/{path}",
    "protocol": "HTTPS",
    "query": "{query}"
  }
}
`)
	createRequest := ocimock.MustJSONFixture[waassdk.CreateHttpRedirectDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-http-redirect",
  "domain": "osok-mock-redirect.example.com",
  "freeformTags": {
    "osok-mock": "create"
  },
  "responseCode": 301,
  "target": {
    "host": "www.example.com",
    "path": "/{path}",
    "protocol": "HTTPS",
    "query": "{query}"
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[waassdk.UpdateHttpRedirectDetails](t, `
{
  "displayName": "osok-mock-http-redirect-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "responseCode": 302,
  "target": {
    "host": "www.example.com",
    "path": "/{path}",
    "protocol": "HTTPS",
    "query": "{query}"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[waassdk.HttpRedirect](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T03:26:28.941Z"
    }
  },
  "displayName": "osok-mock-http-redirect",
  "domain": "osok-mock-redirect.example.com",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "responseCode": 301,
  "target": {
    "host": "www.example.com",
    "path": "/{path}",
    "port": 443,
    "protocol": "HTTPS",
    "query": "{query}"
  },
  "timeCreated": "2026-09-02T03:26:29.801Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[waassdk.HttpRedirect](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T03:26:28.941Z"
    }
  },
  "displayName": "osok-mock-http-redirect-updated",
  "domain": "osok-mock-redirect.example.com",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "responseCode": 302,
  "target": {
    "host": "www.example.com",
    "path": "/{path}",
    "port": 443,
    "protocol": "HTTPS",
    "query": "{query}"
  },
  "timeCreated": "2026-09-02T03:26:29.801Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[waassdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "errors": [],
  "id": "<ocid:2>",
  "logs": [],
  "operationType": "CREATE_HTTP_REDIRECT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "redirect",
      "entityUri": "/20181116/httpRedirects/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T03:26:29.831Z",
  "timeFinished": "2026-09-02T03:26:32.095Z",
  "timeStarted": null
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[waassdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "errors": [],
  "id": "<ocid:4>",
  "logs": [],
  "operationType": "UPDATE_HTTP_REDIRECT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "redirect",
      "entityUri": "/20181116/httpRedirects/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T03:26:35.677Z",
  "timeFinished": "2026-09-02T03:26:42.093Z",
  "timeStarted": null
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[waassdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "errors": [],
  "id": "<ocid:5>",
  "logs": [],
  "operationType": "DELETE_HTTP_REDIRECT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "redirect",
      "entityUri": "/20181116/httpRedirects/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T03:26:46.597Z",
  "timeFinished": "2026-09-02T03:26:52.102Z",
  "timeStarted": null
}
`)
	deleteWorkRequestPending := ocimock.MustOCIResponseFixture[waassdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "errors": [],
  "id": "<ocid:5>",
  "logs": [],
  "operationType": "DELETE_HTTP_REDIRECT",
  "percentComplete": 0,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "redirect",
      "entityUri": "/20181116/httpRedirects/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "ACCEPTED",
  "timeAccepted": "2026-09-02T03:26:46.597Z",
  "timeFinished": null,
  "timeStarted": null
}
`)
	deletePolls := 0
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[waassdk.HttpRedirect, waassdk.CreateHttpRedirectDetails, waassdk.UpdateHttpRedirectDetails]{
		CollectionPath: "/20181116/httpRedirects", ItemPath: "/20181116/httpRedirects/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeArray, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ waassdk.CreateHttpRedirectDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20181116/workRequests/<ocid:2>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20181116/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20181116/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				deletePolls++
				if deletePolls == 1 {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequestPending)
				}
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://waas.us-ashburn-1.oci.oraclecloud.com", BasePath: "20181116", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	baseClient := session.BaseClient()
	sdkClient := mockHttpRedirectOCIClient{
		RedirectClient: waassdk.RedirectClient{BaseClient: baseClient},
		WaasClient:     waassdk.WaasClient{BaseClient: baseClient},
	}
	client := newHttpRedirectServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	mockValidateCreated := func(current *waasv1beta1.HttpRedirect) error {
		if current.Status.DisplayName != "osok-mock-http-redirect" || current.Status.ResponseCode != 301 {
			return fmt.Errorf("created HttpRedirect status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *waasv1beta1.HttpRedirect) error {
		if current.Status.DisplayName != "osok-mock-http-redirect-updated" || current.Status.ResponseCode != 302 {
			return fmt.Errorf("updated HttpRedirect status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*waasv1beta1.HttpRedirect]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *waasv1beta1.HttpRedirect) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *waasv1beta1.HttpRedirect) {
			current.Spec.DisplayName = "osok-mock-http-redirect-updated"
			current.Spec.ResponseCode = 302
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *waasv1beta1.HttpRedirect) error {
			if err := mockValidateUpdated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
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

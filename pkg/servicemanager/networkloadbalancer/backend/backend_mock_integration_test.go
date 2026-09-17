/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package backend

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	networkloadbalancersdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	nlbv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationBackendWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &nlbv1beta1.Backend{}
	ocimock.InitializeResource(resource, "mock-backend")
	resource.Spec = ocimock.MustJSONFixture[nlbv1beta1.BackendSpec](t, `
{
  "ipAddress": "10.0.20.200",
  "port": 8080,
  "weight": 1
}
`)
	resource.Spec.NetworkLoadBalancerId = "<ocid:1>"
	resource.Spec.BackendSetName = "<binding:nlb-backend-set>"
	createRequest := ocimock.MustJSONFixture[networkloadbalancersdk.CreateBackendDetails](t, `
{
  "ipAddress": "10.0.20.200",
  "port": 8080,
  "weight": 1
}
`)
	updateRequest := ocimock.MustJSONFixture[networkloadbalancersdk.UpdateBackendDetails](t, `
{
  "weight": 2
}
`)
	createdState := ocimock.MustOCIResponseFixture[networkloadbalancersdk.Backend](t, `
{
  "ipAddress": "10.0.20.200",
  "ipVersion": "IPV4",
  "isBackup": false,
  "isDrain": false,
  "isOffline": false,
  "name": "10.0.20.200:8080",
  "port": 8080,
  "targetId": null,
  "weight": 1
}
`)
	updatedState := ocimock.MustOCIResponseFixture[networkloadbalancersdk.Backend](t, `
{
  "ipAddress": "10.0.20.200",
  "ipVersion": "IPV4",
  "isBackup": false,
  "isDrain": false,
  "isOffline": false,
  "name": "10.0.20.200:8080",
  "port": 8080,
  "targetId": null,
  "weight": 2
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[networkloadbalancersdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:2>",
  "operationType": "CREATE_BACKEND",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "NetworkLoadBalancer",
      "entityUri": "/networkLoadBalancers/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T22:52:36.096Z",
  "timeFinished": "2026-09-01T22:52:43.837Z",
  "timeStarted": "2026-09-01T22:52:39.140Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[networkloadbalancersdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_BACKEND",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "NetworkLoadBalancer",
      "entityUri": "/networkLoadBalancers/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T22:52:53.889Z",
  "timeFinished": "2026-09-01T22:52:59.427Z",
  "timeStarted": "2026-09-01T22:52:56.008Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[networkloadbalancersdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:5>",
  "operationType": "DELETE_BACKEND",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "NetworkLoadBalancer",
      "entityUri": "/networkLoadBalancers/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T22:53:06.900Z",
  "timeFinished": "2026-09-01T22:53:12.786Z",
  "timeStarted": "2026-09-01T22:53:09.366Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[networkloadbalancersdk.Backend, networkloadbalancersdk.CreateBackendDetails, networkloadbalancersdk.UpdateBackendDetails]{
		CollectionPath: "/20200501/networkLoadBalancers/<ocid:1>/backendSets/<binding:nlb-backend-set>/backends", ItemPath: "/20200501/networkLoadBalancers/<ocid:1>/backendSets/<binding:nlb-backend-set>/backends/10.0.20.200:8080",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ networkloadbalancersdk.CreateBackendDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200501/workRequests/<ocid:2>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200501/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200501/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://network-load-balancer-api.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200501", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := networkloadbalancersdk.NetworkLoadBalancerClient{BaseClient: session.BaseClient()}
	hooks := newBackendRuntimeHooksWithOCIClient(sdkClient)
	applyBackendRuntimeHooks(&hooks, sdkClient, nil)
	manager := &BackendServiceManager{Log: loggerutil.OSOKLogger{}}
	client := wrapBackendGeneratedClient(hooks, defaultBackendServiceClient{ServiceClient: generatedruntime.NewServiceClient[*nlbv1beta1.Backend](buildBackendGeneratedRuntimeConfig(manager, hooks))})
	mockValidateCreated := func(current *nlbv1beta1.Backend) error {
		if current.Status.IpAddress != "10.0.20.200" {
			return fmt.Errorf("created Backend status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *nlbv1beta1.Backend) error {
		if current.Status.Weight != 2 {
			return fmt.Errorf("updated Backend status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*nlbv1beta1.Backend]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *nlbv1beta1.Backend) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *nlbv1beta1.Backend) { current.Spec.Weight = 2 },
		ValidateUpdated: func(current *nlbv1beta1.Backend) error {
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

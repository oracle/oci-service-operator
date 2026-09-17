/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package backendset

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	networkloadbalancersdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	nlbv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationBackendSetWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &nlbv1beta1.BackendSet{}
	ocimock.InitializeResource(resource, "mock-backendset")
	resource.Spec = ocimock.MustJSONFixture[nlbv1beta1.BackendSetSpec](t, `
{
  "areOperationallyActiveBackendsPreferred": false,
  "healthChecker": {
    "port": 8080,
    "protocol": "TCP"
  },
  "ipVersion": "IPV4",
  "isFailOpen": false,
  "isInstantFailoverEnabled": false,
  "isInstantFailoverTcpResetEnabled": false,
  "isPreserveSource": false,
  "name": "osok_mock_backend_set_child",
  "policy": "FIVE_TUPLE"
}
`)
	resource.Spec.NetworkLoadBalancerId = "<ocid:1>"
	createRequest := ocimock.MustJSONFixture[networkloadbalancersdk.CreateBackendSetDetails](t, `
{
  "areOperationallyActiveBackendsPreferred": false,
  "healthChecker": {
    "port": 8080,
    "protocol": "TCP"
  },
  "ipVersion": "IPV4",
  "isFailOpen": false,
  "isInstantFailoverEnabled": false,
  "isInstantFailoverTcpResetEnabled": false,
  "isPreserveSource": false,
  "name": "osok_mock_backend_set_child",
  "policy": "FIVE_TUPLE"
}
`)
	updateRequest := ocimock.MustJSONFixture[networkloadbalancersdk.UpdateBackendSetDetails](t, `
{
  "areOperationallyActiveBackendsPreferred": false,
  "healthChecker": {
    "port": 8080,
    "protocol": "TCP"
  },
  "ipVersion": "IPV4",
  "isFailOpen": false,
  "isInstantFailoverEnabled": false,
  "isInstantFailoverTcpResetEnabled": false,
  "isPreserveSource": false,
  "policy": "TWO_TUPLE"
}
`)
	createdState := ocimock.MustOCIResponseFixture[networkloadbalancersdk.BackendSet](t, `
{
  "areOperationallyActiveBackendsPreferred": false,
  "backends": [],
  "healthChecker": {
    "dns": null,
    "intervalInMillis": 10000,
    "port": 8080,
    "protocol": "TCP",
    "requestData": "",
    "responseBodyRegex": null,
    "responseData": "",
    "retries": 3,
    "returnCode": null,
    "timeoutInMillis": 3000,
    "urlPath": null
  },
  "ipVersion": "IPV4",
  "isFailOpen": false,
  "isInstantFailoverEnabled": false,
  "isInstantFailoverTcpResetEnabled": false,
  "isPreserveSource": false,
  "name": "osok_mock_backend_set_child",
  "policy": "FIVE_TUPLE"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[networkloadbalancersdk.BackendSet](t, `
{
  "areOperationallyActiveBackendsPreferred": false,
  "backends": [],
  "healthChecker": {
    "dns": null,
    "intervalInMillis": 10000,
    "port": 8080,
    "protocol": "TCP",
    "requestData": "",
    "responseBodyRegex": null,
    "responseData": "",
    "retries": 3,
    "returnCode": null,
    "timeoutInMillis": 3000,
    "urlPath": null
  },
  "ipVersion": "IPV4",
  "isFailOpen": false,
  "isInstantFailoverEnabled": false,
  "isInstantFailoverTcpResetEnabled": false,
  "isPreserveSource": false,
  "name": "osok_mock_backend_set_child",
  "policy": "TWO_TUPLE"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[networkloadbalancersdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:2>",
  "operationType": "CREATE_BACKENDSET",
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
  "timeAccepted": "2026-09-02T01:27:05.771Z",
  "timeFinished": "2026-09-02T01:27:13.001Z",
  "timeStarted": "2026-09-02T01:27:08.470Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[networkloadbalancersdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_BACKENDSET",
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
  "timeAccepted": "2026-09-02T01:27:17.821Z",
  "timeFinished": "2026-09-02T01:27:23.347Z",
  "timeStarted": "2026-09-02T01:27:20.786Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[networkloadbalancersdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:5>",
  "operationType": "DELETE_BACKENDSET",
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
  "timeAccepted": "2026-09-02T01:27:29.246Z",
  "timeFinished": "2026-09-02T01:27:44.380Z",
  "timeStarted": "2026-09-02T01:27:40.735Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[networkloadbalancersdk.BackendSet, networkloadbalancersdk.CreateBackendSetDetails, networkloadbalancersdk.UpdateBackendSetDetails]{
		CollectionPath: "/20200501/networkLoadBalancers/<ocid:1>/backendSets", ItemPath: "/20200501/networkLoadBalancers/<ocid:1>/backendSets/osok_mock_backend_set_child",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ networkloadbalancersdk.CreateBackendSetDetails) error {
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
	hooks := newBackendSetRuntimeHooksWithOCIClient(sdkClient)
	applyBackendSetRuntimeHooks(&hooks, sdkClient, nil)
	manager := &BackendSetServiceManager{}
	client := wrapBackendSetGeneratedClient(hooks, defaultBackendSetServiceClient{ServiceClient: generatedruntime.NewServiceClient[*nlbv1beta1.BackendSet](buildBackendSetGeneratedRuntimeConfig(manager, hooks))})
	mockValidateCreated := func(current *nlbv1beta1.BackendSet) error {
		if current.Status.Name != resource.Spec.Name || current.Status.NetworkLoadBalancerId != resource.Spec.NetworkLoadBalancerId {
			return fmt.Errorf("created BackendSet status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *nlbv1beta1.BackendSet) error {
		if current.Status.Policy != "TWO_TUPLE" {
			return fmt.Errorf("updated BackendSet status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*nlbv1beta1.BackendSet]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *nlbv1beta1.BackendSet) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *nlbv1beta1.BackendSet) { current.Spec.Policy = "TWO_TUPLE" },
		ValidateUpdated: func(current *nlbv1beta1.BackendSet) error {
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

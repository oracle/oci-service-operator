/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package listener

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
func TestMockIntegrationListenerWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &nlbv1beta1.Listener{}
	ocimock.InitializeResource(resource, "mock-listener")
	resource.Spec = ocimock.MustJSONFixture[nlbv1beta1.ListenerSpec](t, `
{
  "defaultBackendSetName": "<binding:nlb-listener-backend-set>",
  "isPpv2Enabled": false,
  "name": "osok_mock_listener",
  "port": 9080,
  "protocol": "TCP"
}
`)
	resource.Spec.NetworkLoadBalancerId = "<ocid:1>"
	createRequest := ocimock.MustJSONFixture[networkloadbalancersdk.CreateListenerDetails](t, `
{
  "defaultBackendSetName": "<binding:nlb-listener-backend-set>",
  "isPpv2Enabled": false,
  "name": "osok_mock_listener",
  "port": 9080,
  "protocol": "TCP"
}
`)
	updateRequest := ocimock.MustJSONFixture[networkloadbalancersdk.UpdateListenerDetails](t, `
{
  "defaultBackendSetName": "<binding:nlb-listener-backend-set>",
  "isPpv2Enabled": false,
  "port": 9081,
  "protocol": "TCP"
}
`)
	createdState := ocimock.MustOCIResponseFixture[networkloadbalancersdk.Listener](t, `
{
  "defaultBackendSetName": "<binding:nlb-listener-backend-set>",
  "ipVersion": "IPV4",
  "isPpv2Enabled": false,
  "l3IpIdleTimeout": null,
  "name": "osok_mock_listener",
  "port": 9080,
  "protocol": "TCP",
  "tcpIdleTimeout": 360,
  "udpIdleTimeout": null
}
`)
	updatedState := ocimock.MustOCIResponseFixture[networkloadbalancersdk.Listener](t, `
{
  "defaultBackendSetName": "<binding:nlb-listener-backend-set>",
  "ipVersion": "IPV4",
  "isPpv2Enabled": false,
  "l3IpIdleTimeout": null,
  "name": "osok_mock_listener",
  "port": 9081,
  "protocol": "TCP",
  "tcpIdleTimeout": 360,
  "udpIdleTimeout": null
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[networkloadbalancersdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:2>",
  "operationType": "CREATE_LISTENER",
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
  "timeAccepted": "2026-09-02T01:32:52.290Z",
  "timeFinished": "2026-09-02T01:32:55.395Z",
  "timeStarted": "2026-09-02T01:32:55.223Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[networkloadbalancersdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_LISTENER",
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
  "timeAccepted": "2026-09-02T01:32:58.454Z",
  "timeFinished": "2026-09-02T01:33:01.995Z",
  "timeStarted": "2026-09-02T01:33:01.856Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[networkloadbalancersdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:5>",
  "operationType": "DELETE_LISTENER",
  "percentComplete": 0,
  "resources": [
    {
      "actionType": "IN_PROGRESS",
      "entityType": "NetworkLoadBalancer",
      "entityUri": "/networkLoadBalancers/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "ACCEPTED",
  "timeAccepted": "2026-09-02T01:33:05.014Z",
  "timeFinished": null,
  "timeStarted": null
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[networkloadbalancersdk.Listener, networkloadbalancersdk.CreateListenerDetails, networkloadbalancersdk.UpdateListenerDetails]{
		CollectionPath: "/20200501/networkLoadBalancers/<ocid:1>/listeners", ItemPath: "/20200501/networkLoadBalancers/<ocid:1>/listeners/osok_mock_listener",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ networkloadbalancersdk.CreateListenerDetails) error {
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
	hooks := newListenerRuntimeHooksWithOCIClient(sdkClient)
	applyListenerRuntimeHooks(&hooks, sdkClient, nil)
	manager := &ListenerServiceManager{}
	client := wrapListenerGeneratedClient(hooks, defaultListenerServiceClient{ServiceClient: generatedruntime.NewServiceClient[*nlbv1beta1.Listener](buildListenerGeneratedRuntimeConfig(manager, hooks))})
	mockValidateCreated := func(current *nlbv1beta1.Listener) error {
		if current.Status.Name != resource.Spec.Name || current.Status.Port != 9080 {
			return fmt.Errorf("created Listener status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *nlbv1beta1.Listener) error {
		if current.Status.Port != 9081 {
			return fmt.Errorf("updated Listener status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*nlbv1beta1.Listener]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *nlbv1beta1.Listener) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *nlbv1beta1.Listener) { current.Spec.Port = 9081 },
		ValidateUpdated: func(current *nlbv1beta1.Listener) error {
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

/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package hostname

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	ctrl "sigs.k8s.io/controller-runtime"
)

type hostnameDelayedVisibilityResponder struct {
	delegate *ocimock.CRUDResponder[loadbalancersdk.Hostname]
}

func (r hostnameDelayedVisibilityResponder) Respond(request ocimock.Request) (ocimock.Response, error) {
	if request.Method == http.MethodGet && request.URL.Path == "/20170115/loadBalancers/<ocid:1>/hostnames" {
		return ocimock.JSONResponse(http.StatusOK, []loadbalancersdk.Hostname{})
	}
	return r.delegate.Respond(request)
}

func (r hostnameDelayedVisibilityResponder) Verify() error {
	return r.delegate.Verify()
}

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationHostnameWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &loadbalancerv1beta1.Hostname{}
	ocimock.InitializeResource(resource, "mock-hostname")
	resource.SetAnnotations(map[string]string{hostnameLoadBalancerIDAnnotation: "<ocid:1>"})
	resource.Spec = ocimock.MustJSONFixture[loadbalancerv1beta1.HostnameSpec](t, `{
  "hostname": "create.example.com",
  "name": "osok_mock_hostname_v1"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "hostname": "update.example.com"
}`)

	createRequest := ocimock.MustJSONFixture[loadbalancersdk.CreateHostnameDetails](t, `{
  "hostname": "create.example.com",
  "name": "osok_mock_hostname_v1"
}`)
	updateRequest := ocimock.MustJSONFixture[loadbalancersdk.UpdateHostnameDetails](t, `{
  "hostname": "update.example.com"
}`)
	createdState := ocimock.MustOCIResponseFixture[loadbalancersdk.Hostname](t, `{
  "hostname": "create.example.com",
  "name": "osok_mock_hostname_v1"
}`)
	updatedState := ocimock.MustOCIResponseFixture[loadbalancersdk.Hostname](t, `{
  "hostname": "update.example.com",
  "name": "osok_mock_hostname_v1"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[loadbalancersdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "errorDetails": [

  ],
  "id": "<ocid:2>",
  "lifecycleState": "SUCCEEDED",
  "loadBalancerId": "<ocid:1>",
  "message": "{\"eventId\":\"9502aa63-cb2f-4192-8a2f-8ce91549f72b\",\"loadBalancerId\":\"<ocid:1>\",\"workflowName\":\"AddHostnameWorkflow\",\"type\":\"SUCCESS\",\"message\":\"OK\",\"workRequestId\":\"<ocid:2>\"}",
  "timeAccepted": "2026-09-01T18:40:17.844Z",
  "timeFinished": "2026-09-01T18:40:23.890Z",
  "type": "CreateHostname"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[loadbalancersdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "errorDetails": [

  ],
  "id": "<ocid:4>",
  "lifecycleState": "SUCCEEDED",
  "loadBalancerId": "<ocid:1>",
  "message": "{\"eventId\":\"cce27916-9244-4a8a-850c-7185cab04c43\",\"loadBalancerId\":\"<ocid:1>\",\"workflowName\":\"PutHostnameWorkflow\",\"type\":\"SUCCESS\",\"message\":\"OK\",\"workRequestId\":\"<ocid:4>\"}",
  "timeAccepted": "2026-09-01T18:40:25.906Z",
  "timeFinished": "2026-09-01T18:40:32.229Z",
  "type": "UpdateHostname"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[loadbalancersdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "errorDetails": [

  ],
  "id": "<ocid:5>",
  "lifecycleState": "SUCCEEDED",
  "loadBalancerId": "<ocid:1>",
  "message": "{\"eventId\":\"39354293-a2ae-416b-a506-faffa5fb6868\",\"loadBalancerId\":\"<ocid:1>\",\"workflowName\":\"RemoveHostnameWorkflow\",\"type\":\"SUCCESS\",\"message\":\"OK\",\"workRequestId\":\"<ocid:5>\"}",
  "timeAccepted": "2026-09-01T18:40:37.577Z",
  "timeFinished": "2026-09-01T18:40:41.430Z",
  "type": "DeleteHostname"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[loadbalancersdk.Hostname, loadbalancersdk.CreateHostnameDetails, loadbalancersdk.UpdateHostnameDetails]{
		CollectionPath: "/20170115/loadBalancers/<ocid:1>/hostnames", ItemPath: "/20170115/loadBalancers/<ocid:1>/hostnames/osok_mock_hostname_v1",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		CreatedReadStatuses: []int{http.StatusNotFound, http.StatusOK},
		UpdatedReadStates:   []loadbalancersdk.Hostname{createdState, updatedState},
		ListShape:           ocimock.ListShapeArray, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: false, RequireDeleteRead: true,
		CreateStatus: 204, UpdateStatus: 204, DeleteStatus: 204, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ loadbalancersdk.CreateHostnameDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20170115/loadBalancerWorkRequests/<ocid:2>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20170115/loadBalancerWorkRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20170115/loadBalancerWorkRequests/<ocid:5>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{
		Host: "https://iaas.us-ashburn-1.oraclecloud.com", BasePath: "20170115",
		Responder: hostnameDelayedVisibilityResponder{delegate: responder},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	client := newHostnameRuntimeClient(loadbalancersdk.LoadBalancerClient{BaseClient: session.BaseClient()}, log)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loadbalancerv1beta1.Hostname]{
		Resource: resource, Client: client,
		ValidateCreated: func(current *loadbalancerv1beta1.Hostname) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Name != resource.Spec.Name || current.Status.Hostname != resource.Spec.Hostname || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Hostname status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loadbalancerv1beta1.Hostname) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loadbalancerv1beta1.Hostname) error {
			if current.Status.Hostname != current.Spec.Hostname || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Hostname status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("%v; requests=%+v", err, session.Requests())
	}
}

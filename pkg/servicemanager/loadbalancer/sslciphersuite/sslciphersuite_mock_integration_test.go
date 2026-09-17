/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package sslciphersuite

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed
// formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationSSLCipherSuiteWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &loadbalancerv1beta1.SSLCipherSuite{}
	ocimock.InitializeResource(resource, "mock-sslciphersuite")
	resource.SetAnnotations(map[string]string{sslCipherSuiteLoadBalancerIDAnnotation: "<ocid:1>"})
	resource.Spec = ocimock.MustJSONFixture[loadbalancerv1beta1.SSLCipherSuiteSpec](t, `{
  "ciphers": [
    "ECDHE-RSA-AES256-GCM-SHA384"
  ],
  "name": "osok_mock_ssl_cipher_v2"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "ciphers": [
    "ECDHE-RSA-AES256-GCM-SHA384",
    "ECDHE-RSA-AES128-GCM-SHA256"
  ]
}`)

	createRequest := ocimock.MustJSONFixture[loadbalancersdk.CreateSslCipherSuiteDetails](t, `{
  "ciphers": [
    "ECDHE-RSA-AES256-GCM-SHA384"
  ],
  "name": "osok_mock_ssl_cipher_v2"
}`)
	updateRequest := ocimock.MustJSONFixture[loadbalancersdk.UpdateSslCipherSuiteDetails](t, `{
  "ciphers": [
    "ECDHE-RSA-AES256-GCM-SHA384",
    "ECDHE-RSA-AES128-GCM-SHA256"
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[loadbalancersdk.SslCipherSuite](t, `{
  "ciphers": [
    "ECDHE-RSA-AES256-GCM-SHA384"
  ],
  "name": "osok_mock_ssl_cipher_v2"
}`)
	updatedState := ocimock.MustOCIResponseFixture[loadbalancersdk.SslCipherSuite](t, `{
  "ciphers": [
    "ECDHE-RSA-AES256-GCM-SHA384",
    "ECDHE-RSA-AES128-GCM-SHA256"
  ],
  "name": "osok_mock_ssl_cipher_v2"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[loadbalancersdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "errorDetails": [

  ],
  "id": "<ocid:2>",
  "lifecycleState": "SUCCEEDED",
  "loadBalancerId": "<ocid:1>",
  "message": "{\"eventId\":\"217da238-df11-43c1-a314-cd2de3f8fa1b\",\"loadBalancerId\":\"<ocid:1>\",\"workflowName\":\"AddCipherSuiteWorkflow\",\"type\":\"SUCCESS\",\"message\":\"OK\",\"workRequestId\":\"<ocid:2>\"}",
  "timeAccepted": "2026-09-01T19:18:22.782Z",
  "timeFinished": "2026-09-01T19:18:27.816Z",
  "type": "CreateCipherSuite"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[loadbalancersdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "errorDetails": [

  ],
  "id": "<ocid:4>",
  "lifecycleState": "SUCCEEDED",
  "loadBalancerId": "<ocid:1>",
  "message": "{\"eventId\":\"76044c5d-094d-43c0-9c3a-3e18a082641b\",\"loadBalancerId\":\"<ocid:1>\",\"workflowName\":\"PutCipherSuiteWorkflow\",\"type\":\"SUCCESS\",\"message\":\"OK\",\"workRequestId\":\"<ocid:4>\"}",
  "timeAccepted": "2026-09-01T19:18:30.158Z",
  "timeFinished": "2026-09-01T19:18:34.713Z",
  "type": "UpdateCipherSuite"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[loadbalancersdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "errorDetails": [

  ],
  "id": "<ocid:5>",
  "lifecycleState": "SUCCEEDED",
  "loadBalancerId": "<ocid:1>",
  "message": "{\"eventId\":\"8d4a7af8-fe2b-41ec-bd4e-3d2e3b671bbe\",\"loadBalancerId\":\"<ocid:1>\",\"workflowName\":\"RemoveCipherSuiteWorkflow\",\"type\":\"SUCCESS\",\"message\":\"OK\",\"workRequestId\":\"<ocid:5>\"}",
  "timeAccepted": "2026-09-01T19:18:37.375Z",
  "timeFinished": "2026-09-01T19:18:43.748Z",
  "type": "DeleteCipherSuite"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[loadbalancersdk.SslCipherSuite, loadbalancersdk.CreateSslCipherSuiteDetails, loadbalancersdk.UpdateSslCipherSuiteDetails]{
		CollectionPath: "/20170115/loadBalancers/<ocid:1>/sslCipherSuites", ItemPath: "/20170115/loadBalancers/<ocid:1>/sslCipherSuites/osok_mock_ssl_cipher_v2",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeArray, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: false, RequireDeleteRead: true,
		CreateStatus: 204, UpdateStatus: 204, DeleteStatus: 204, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ loadbalancersdk.CreateSslCipherSuiteDetails) error {
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
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.us-ashburn-1.oraclecloud.com", BasePath: "20170115", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := loadbalancersdk.LoadBalancerClient{BaseClient: session.BaseClient()}
	hooks := newSSLCipherSuiteRuntimeHooksWithOCIClient(sdkClient)
	applySSLCipherSuiteRuntimeHooks(&hooks, sdkClient, nil, log)
	client := wrapSSLCipherSuiteGeneratedClient(hooks, defaultSSLCipherSuiteServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.SSLCipherSuite](buildSSLCipherSuiteGeneratedRuntimeConfig(&SSLCipherSuiteServiceManager{Log: log}, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loadbalancerv1beta1.SSLCipherSuite]{
		Resource: resource, Client: client,
		ValidateCreated: func(current *loadbalancerv1beta1.SSLCipherSuite) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Name != resource.Spec.Name || len(current.Status.Ciphers) != len(resource.Spec.Ciphers) || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created SSLCipherSuite status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loadbalancerv1beta1.SSLCipherSuite) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loadbalancerv1beta1.SSLCipherSuite) error {
			if len(current.Status.Ciphers) != len(current.Spec.Ciphers) || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated SSLCipherSuite status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("%v; requests=%+v", err, session.Requests())
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

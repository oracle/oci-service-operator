/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package routingpolicy

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
func TestMockIntegrationRoutingPolicyWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &loadbalancerv1beta1.RoutingPolicy{}
	ocimock.InitializeResource(resource, "mock-routingpolicy")
	resource.Spec = ocimock.MustJSONFixture[loadbalancerv1beta1.RoutingPolicySpec](t, `{
  "conditionLanguageVersion": "V1",
  "name": "osok_mock_routing_policy",
  "rules": [
    {
      "actions": [
        {
          "backendSetName": "osok_mock_backend_set",
          "name": "FORWARD_TO_BACKENDSET"
        }
      ],
      "condition": "all(http.request.url.path sw '/images')",
      "name": "route_images"
    }
  ],
  "loadBalancerId": "<ocid:1>"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "conditionLanguageVersion": "V1",
  "rules": [
    {
      "actions": [
        {
          "backendSetName": "osok_mock_backend_set",
          "name": "FORWARD_TO_BACKENDSET"
        }
      ],
      "condition": "all(http.request.url.path sw '/assets')",
      "name": "route_images"
    }
  ]
}`)

	createRequest := ocimock.MustJSONFixture[loadbalancersdk.CreateRoutingPolicyDetails](t, `{
  "conditionLanguageVersion": "V1",
  "name": "osok_mock_routing_policy",
  "rules": [
    {
      "actions": [
        {
          "backendSetName": "osok_mock_backend_set",
          "name": "FORWARD_TO_BACKENDSET"
        }
      ],
      "condition": "all(http.request.url.path sw '/images')",
      "name": "route_images"
    }
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[loadbalancersdk.UpdateRoutingPolicyDetails](t, `{
  "conditionLanguageVersion": "V1",
  "rules": [
    {
      "actions": [
        {
          "backendSetName": "osok_mock_backend_set",
          "name": "FORWARD_TO_BACKENDSET"
        }
      ],
      "condition": "all(http.request.url.path sw '/assets')",
      "name": "route_images"
    }
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[loadbalancersdk.RoutingPolicy](t, `{
  "conditionLanguageVersion": "V1",
  "name": "osok_mock_routing_policy",
  "rules": [
    {
      "actions": [
        {
          "backendSetName": "osok_mock_backend_set",
          "name": "FORWARD_TO_BACKENDSET"
        }
      ],
      "condition": "all(http.request.url.path sw '/images')",
      "name": "route_images"
    }
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[loadbalancersdk.RoutingPolicy](t, `{
  "conditionLanguageVersion": "V1",
  "name": "osok_mock_routing_policy",
  "rules": [
    {
      "actions": [
        {
          "backendSetName": "osok_mock_backend_set",
          "name": "FORWARD_TO_BACKENDSET"
        }
      ],
      "condition": "all(http.request.url.path sw '/assets')",
      "name": "route_images"
    }
  ]
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[loadbalancersdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "errorDetails": [

  ],
  "id": "<ocid:2>",
  "lifecycleState": "SUCCEEDED",
  "loadBalancerId": "<ocid:1>",
  "message": "{\"eventId\":\"7f731190-426f-4d68-a30f-b4a965ae73ea\",\"loadBalancerId\":\"<ocid:1>\",\"workflowName\":\"AddRoutingPolicyWorkflow\",\"type\":\"SUCCESS\",\"message\":\"OK\",\"workRequestId\":\"<ocid:2>\"}",
  "timeAccepted": "2026-09-02T01:38:11.197Z",
  "timeFinished": "2026-09-02T01:38:16.476Z",
  "type": "AddRoutingPolicy"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[loadbalancersdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "errorDetails": [

  ],
  "id": "<ocid:4>",
  "lifecycleState": "SUCCEEDED",
  "loadBalancerId": "<ocid:1>",
  "message": "{\"eventId\":\"8a40b411-bc17-498b-a3cf-93185bc7f4fb\",\"loadBalancerId\":\"<ocid:1>\",\"workflowName\":\"UpdateRoutingPolicyWorkflow\",\"type\":\"SUCCESS\",\"message\":\"OK\",\"workRequestId\":\"<ocid:4>\"}",
  "timeAccepted": "2026-09-02T01:38:22.665Z",
  "timeFinished": "2026-09-02T01:38:28.071Z",
  "type": "UpdateRoutingPolicy"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[loadbalancersdk.WorkRequest](t, `{
  "compartmentId": "<ocid:3>",
  "errorDetails": [

  ],
  "id": "<ocid:5>",
  "lifecycleState": "SUCCEEDED",
  "loadBalancerId": "<ocid:1>",
  "message": "{\"eventId\":\"6ec0ce04-b43b-42b5-a109-78fdc95e1ae1\",\"loadBalancerId\":\"<ocid:1>\",\"workflowName\":\"RemoveRoutingPolicyWorkflow\",\"type\":\"SUCCESS\",\"message\":\"OK\",\"workRequestId\":\"<ocid:5>\"}",
  "timeAccepted": "2026-09-02T01:38:34.100Z",
  "timeFinished": "2026-09-02T01:38:42.052Z",
  "type": "RemoveRoutingPolicy"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[loadbalancersdk.RoutingPolicy, loadbalancersdk.CreateRoutingPolicyDetails, loadbalancersdk.UpdateRoutingPolicyDetails]{
		CollectionPath: "/20170115/loadBalancers/<ocid:1>/routingPolicies", ItemPath: "/20170115/loadBalancers/<ocid:1>/routingPolicies/osok_mock_routing_policy",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeArray, RequireCreateRead: true, RequireUpdateRead: true, DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 204, UpdateStatus: 204, DeleteStatus: 204, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ loadbalancersdk.CreateRoutingPolicyDetails) error {
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
	hooks := newRoutingPolicyRuntimeHooksWithOCIClient(sdkClient)
	applyRoutingPolicyRuntimeHooks(&hooks, sdkClient, nil)
	manager := &RoutingPolicyServiceManager{Log: log}
	client := wrapRoutingPolicyGeneratedClient(hooks, defaultRoutingPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.RoutingPolicy](buildRoutingPolicyGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loadbalancerv1beta1.RoutingPolicy]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loadbalancerv1beta1.RoutingPolicy) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Name != resource.Spec.Name || len(current.Status.Rules) != len(resource.Spec.Rules) || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created RoutingPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loadbalancerv1beta1.RoutingPolicy) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loadbalancerv1beta1.RoutingPolicy) error {
			if len(current.Status.Rules) != len(current.Spec.Rules) || current.Status.Rules[0].Condition != current.Spec.Rules[0].Condition || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated RoutingPolicy status = %+v", current.Status)
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

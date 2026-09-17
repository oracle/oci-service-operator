/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package networkloadbalancer

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	networkloadbalancersdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	nlbsdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	nlbv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockNetworkLoadBalancerName = "osok-mock-common-nlb-v1"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationNetworkLoadBalancerWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &nlbv1beta1.NetworkLoadBalancer{}
	ocimock.InitializeResource(resource, "mock-networkloadbalancer")
	resource.Spec = ocimock.MustJSONFixture[nlbv1beta1.NetworkLoadBalancerSpec](t, `
{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-common-nlb-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isPreserveSourceDestination": false,
  "isPrivate": true,
  "isSymmetricHashEnabled": false,
  "subnetId": "<ocid:2>"
}
`)
	nsgSpec := resource.Spec
	createRequest := ocimock.MustJSONFixture[networkloadbalancersdk.CreateNetworkLoadBalancerDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-common-nlb-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "isPreserveSourceDestination": false,
  "isPrivate": true,
  "isSymmetricHashEnabled": false,
  "subnetId": "<ocid:2>"
}
`)
	updateRequest := ocimock.MustJSONFixture[networkloadbalancersdk.UpdateNetworkLoadBalancerDetails](t, `
{
  "displayName": "osok-mock-common-nlb-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[networkloadbalancersdk.NetworkLoadBalancer](t, `
{
  "backendSets": {},
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T00:27:54.277Z"
    }
  },
  "displayName": "osok-mock-common-nlb-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:4>",
  "ipAddresses": [
    {
      "ipAddress": "10.103.1.201",
      "ipVersion": "IPV4",
      "isPublic": false,
      "reservedIp": null
    }
  ],
  "isPreserveSourceDestination": false,
  "isPrivate": true,
  "isSymmetricHashEnabled": false,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "listeners": {},
  "networkSecurityGroupIds": [],
  "nlbIpVersion": "IPV4",
  "securityAttributes": {},
  "subnetId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-09-01T00:27:54Z",
  "timeUpdated": "2026-09-01T00:28:03Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[networkloadbalancersdk.NetworkLoadBalancer](t, `
{
  "backendSets": {},
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T00:27:54.277Z"
    }
  },
  "displayName": "osok-mock-common-nlb-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "ipAddresses": [
    {
      "ipAddress": "10.103.1.201",
      "ipVersion": "IPV4",
      "isPublic": false,
      "reservedIp": null
    }
  ],
  "isPreserveSourceDestination": false,
  "isPrivate": true,
  "isSymmetricHashEnabled": false,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "listeners": {},
  "networkSecurityGroupIds": [],
  "nlbIpVersion": "IPV4",
  "securityAttributes": {},
  "subnetId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-09-01T00:27:54Z",
  "timeUpdated": "2026-09-01T00:28:13Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[nlbsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:3>",
  "operationType": "CREATE_NETWORK_LOAD_BALANCER",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "NetworkLoadBalancer",
      "entityUri": "/networkLoadBalancers/<ocid:4>",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T00:27:54.636Z",
  "timeFinished": "2026-09-01T00:28:03.798Z",
  "timeStarted": "2026-09-01T00:27:56.729Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[nlbsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "UPDATE_NETWORK_LOAD_BALANCER",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "NetworkLoadBalancer",
      "entityUri": "/networkLoadBalancers/<ocid:4>",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T00:28:06.223Z",
  "timeFinished": "2026-09-01T00:28:13.707Z",
  "timeStarted": "2026-09-01T00:28:07.692Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[nlbsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:6>",
  "operationType": "DELETE_NETWORK_LOAD_BALANCER",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "NetworkLoadBalancer",
      "entityUri": "/networkLoadBalancers/<ocid:4>",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T00:28:17.688Z",
  "timeFinished": "2026-09-01T00:28:28.206Z",
  "timeStarted": "2026-09-01T00:28:21.872Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[networkloadbalancersdk.NetworkLoadBalancer, networkloadbalancersdk.CreateNetworkLoadBalancerDetails, networkloadbalancersdk.UpdateNetworkLoadBalancerDetails]{
		CollectionPath: "/20200501/networkLoadBalancers", ItemPath: "/20200501/networkLoadBalancers/<ocid:4>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:3>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ networkloadbalancersdk.CreateNetworkLoadBalancerDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200501/workRequests/<ocid:3>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200501/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200501/workRequests/<ocid:6>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
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
	client := newNetworkLoadBalancerServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient,
	)
	mockValidateCreated := func(current *nlbv1beta1.NetworkLoadBalancer) error {
		if current.Status.LifecycleState != string(nlbsdk.LifecycleStateActive) || current.Status.DisplayName != mockNetworkLoadBalancerName {
			return fmt.Errorf("created NetworkLoadBalancer status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *nlbv1beta1.NetworkLoadBalancer) error {
		if current.Status.DisplayName != current.Spec.DisplayName || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated NetworkLoadBalancer status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*nlbv1beta1.NetworkLoadBalancer]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *nlbv1beta1.NetworkLoadBalancer) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *nlbv1beta1.NetworkLoadBalancer) {
			current.Spec.DisplayName = mockNetworkLoadBalancerName + "-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *nlbv1beta1.NetworkLoadBalancer) error {
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

	nsgResource := &nlbv1beta1.NetworkLoadBalancer{Spec: nsgSpec}
	ocimock.InitializeResource(nsgResource, "mock-networkloadbalancer-nsg-update")
	nsgResource.Spec.NetworkSecurityGroupIds = []string{"<ocid:updated-nsg>"}
	nsgResource.Status.Id = "<ocid:4>"
	nsgResource.Status.OsokStatus.Ocid = "<ocid:4>"
	nsgUpdateDetails := networkloadbalancersdk.UpdateNetworkSecurityGroupsDetails{
		NetworkSecurityGroupIds: append([]string(nil), nsgResource.Spec.NetworkSecurityGroupIds...),
	}
	nsgUpdatedState := createdState
	nsgUpdatedState.NetworkSecurityGroupIds = append([]string(nil), nsgResource.Spec.NetworkSecurityGroupIds...)
	nsgApplied := false
	nsgWorkRequest := ocimock.MustOCIResponseFixture[networkloadbalancersdk.WorkRequest](t, `
{
  "id": "<ocid:nsg-work-request>",
  "operationType": "UPDATE_NSGS",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "NetworkLoadBalancer",
      "entityUri": "/networkLoadBalancers/<ocid:4>",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	nsgResponder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[networkloadbalancersdk.NetworkLoadBalancer]{
		CollectionPath: "/20200501/networkLoadBalancers", ItemPath: "/20200501/networkLoadBalancers/<ocid:4>",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationRead}, InitialState: &createdState,
		ReadTransition: func(_ ocimock.Request, state networkloadbalancersdk.NetworkLoadBalancer) (networkloadbalancersdk.NetworkLoadBalancer, ocimock.Response, error) {
			if nsgApplied {
				state = nsgUpdatedState
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "update-network-security-groups", Method: http.MethodPut, Path: "/20200501/networkLoadBalancers/<ocid:4>/networkSecurityGroups", MinimumCalls: 1,
				Respond: func(request ocimock.Request) (ocimock.Response, error) {
					if err := ocimock.ValidateJSONRequest(request, nsgUpdateDetails); err != nil {
						return ocimock.Response{}, err
					}
					nsgApplied = true
					return ocimock.Response{StatusCode: http.StatusAccepted, Header: http.Header{
						"Opc-Work-Request-Id": []string{"<ocid:nsg-work-request>"},
						"Opc-Request-Id":      []string{"mock-nsg-update-request"},
					}}, nil
				},
			},
			{
				Name: "nsg-update-work-request", Method: http.MethodGet, Path: "/20200501/workRequests/<ocid:nsg-work-request>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, nsgWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	nsgSession, err := ocimock.Open(ocimock.Options{Host: "https://network-load-balancer-api.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200501", Responder: nsgResponder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nsgSession.Close() })
	nsgClient := newNetworkLoadBalancerServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration-nsg-update")},
		networkloadbalancersdk.NetworkLoadBalancerClient{BaseClient: nsgSession.BaseClient()},
	)
	var nsgResponse servicemanager.OSOKResponse
	for attempt := 0; attempt < 5; attempt++ {
		nsgResponse, err = nsgClient.CreateOrUpdate(context.Background(), nsgResource, ctrl.Request{})
		if err != nil {
			t.Fatal(err)
		}
		if !nsgResponse.ShouldRequeue {
			break
		}
	}
	if !nsgResponse.IsSuccessful || nsgResponse.ShouldRequeue || !nsgApplied ||
		len(nsgResource.Status.NetworkSecurityGroupIds) != 1 || nsgResource.Status.NetworkSecurityGroupIds[0] != "<ocid:updated-nsg>" ||
		nsgResource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("NetworkLoadBalancer NSG update response=%+v status=%+v", nsgResponse, nsgResource.Status)
	}
	if err := nsgSession.Close(); err != nil {
		t.Fatal(err)
	}
}

/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loadbalancer

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockLoadBalancerID = "ocid1.loadbalancer.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/loadbalancer/loadbalancer and formal/imports/loadbalancer/loadbalancer.json
//   - resource runtime: loadbalancer_runtime_client.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/loadbalancer
func TestMockIntegrationLoadBalancerLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &loadbalancerv1beta1.LoadBalancer{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-load-balancer", Namespace: "default", UID: types.UID("mock-load-balancer-uid")},
		Spec: loadbalancerv1beta1.LoadBalancerSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			DisplayName:   "mock-load-balancer",
			ShapeName:     "flexible",
			SubnetIds:     []string{"ocid1.subnet.oc1..mock"},
			ShapeDetails: loadbalancerv1beta1.LoadBalancerShapeDetails{
				MinimumBandwidthInMbps: 10,
				MaximumBandwidthInMbps: 10,
			},
			IsPrivate:    true,
			FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newLoadBalancerMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20170115", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close LoadBalancer OCI mock: %v", err)
		}
	})

	client := newGeneratedLoadBalancerServiceClient(
		loadbalancersdk.LoadBalancerClient{BaseClient: session.BaseClient()},
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		nil,
		nil,
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loadbalancerv1beta1.LoadBalancer]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loadbalancerv1beta1.LoadBalancer) error {
			if current.Status.Id != mockLoadBalancerID ||
				current.Status.DisplayName != "mock-load-balancer" ||
				current.Status.LifecycleState != string(loadbalancersdk.LoadBalancerLifecycleStateActive) {
				return fmt.Errorf("created LoadBalancer status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loadbalancerv1beta1.LoadBalancer) {
			current.Spec.DisplayName = "mock-load-balancer-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *loadbalancerv1beta1.LoadBalancer) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(loadbalancersdk.LoadBalancerLifecycleStateActive) {
				return fmt.Errorf("updated LoadBalancer status = %+v", current.Status)
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

func newLoadBalancerMockResponder(resource *loadbalancerv1beta1.LoadBalancer) (*ocimock.CRUDResponder[loadbalancersdk.LoadBalancer], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createReadObserved := false
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[loadbalancersdk.LoadBalancer]{
		CollectionPath:         "/20170115/loadBalancers",
		ItemPath:               "/20170115/loadBalancers/" + mockLoadBalancerID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		List: func(request ocimock.Request, present bool, state loadbalancersdk.LoadBalancer) (ocimock.Response, error) {
			query := request.URL.Query()
			if query.Get("compartmentId") != resource.Spec.CompartmentId ||
				query.Get("displayName") != resource.Spec.DisplayName {
				return ocimock.Response{}, fmt.Errorf("unexpected ListLoadBalancers query: %s", request.URL.RawQuery)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []loadbalancersdk.LoadBalancer{})
			}
			return ocimock.JSONResponse(http.StatusOK, []loadbalancersdk.LoadBalancer{state})
		},
		Create: func(request ocimock.Request) (loadbalancersdk.LoadBalancer, ocimock.Response, error) {
			var details loadbalancersdk.CreateLoadBalancerDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return loadbalancersdk.LoadBalancer{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return loadbalancersdk.LoadBalancer{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName ||
				details.ShapeName == nil || *details.ShapeName != resource.Spec.ShapeName ||
				len(details.SubnetIds) != 1 || details.SubnetIds[0] != resource.Spec.SubnetIds[0] ||
				details.ShapeDetails == nil || details.ShapeDetails.MinimumBandwidthInMbps == nil ||
				*details.ShapeDetails.MinimumBandwidthInMbps != 10 ||
				details.IsPrivate == nil || !*details.IsPrivate {
				return loadbalancersdk.LoadBalancer{}, ocimock.Response{}, fmt.Errorf("unexpected CreateLoadBalancer details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return loadbalancersdk.LoadBalancer{}, ocimock.Response{}, fmt.Errorf("CreateLoadBalancer opc-retry-token is empty")
			}
			state := loadbalancersdk.LoadBalancer{
				Id:             common.String(mockLoadBalancerID),
				CompartmentId:  details.CompartmentId,
				DisplayName:    details.DisplayName,
				ShapeName:      details.ShapeName,
				SubnetIds:      append([]string(nil), details.SubnetIds...),
				ShapeDetails:   details.ShapeDetails,
				IsPrivate:      details.IsPrivate,
				FreeformTags:   details.FreeformTags,
				TimeCreated:    &createdAt,
				LifecycleState: loadbalancersdk.LoadBalancerLifecycleStateCreating,
			}
			response := ocimock.EmptyResponse(http.StatusNoContent)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-load-balancer-create"}}
			return state, response, nil
		},
		ReadTransition: func(_ ocimock.Request, state loadbalancersdk.LoadBalancer) (loadbalancersdk.LoadBalancer, ocimock.Response, error) {
			switch state.LifecycleState {
			case loadbalancersdk.LoadBalancerLifecycleStateCreating:
				if createReadObserved {
					state.LifecycleState = loadbalancersdk.LoadBalancerLifecycleStateActive
				} else {
					createReadObserved = true
				}
			case loadbalancersdk.LoadBalancerLifecycleStateDeleting:
				if deleteReadObserved {
					state.LifecycleState = loadbalancersdk.LoadBalancerLifecycleStateDeleted
				} else {
					deleteReadObserved = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state loadbalancersdk.LoadBalancer) (loadbalancersdk.LoadBalancer, ocimock.Response, error) {
			var details loadbalancersdk.UpdateLoadBalancerDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return loadbalancersdk.LoadBalancer{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-load-balancer-updated" ||
				details.FreeformTags["osok-mock"] != "update" ||
				details.IpMode != "" || details.ReservedIps != nil {
				return loadbalancersdk.LoadBalancer{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateLoadBalancer details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.FreeformTags = details.FreeformTags
			response := ocimock.EmptyResponse(http.StatusNoContent)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-load-balancer-update"}}
			return state, response, nil
		},
		DeleteTransition: func(_ ocimock.Request, state loadbalancersdk.LoadBalancer) (loadbalancersdk.LoadBalancer, ocimock.Response, error) {
			state.LifecycleState = loadbalancersdk.LoadBalancerLifecycleStateDeleting
			response := ocimock.EmptyResponse(http.StatusNoContent)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-load-balancer-delete"}}
			return state, response, nil
		},
	})
}

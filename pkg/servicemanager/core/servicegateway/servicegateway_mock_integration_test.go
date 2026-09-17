/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicegateway

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockServiceGatewayID = "ocid1.servicegateway.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/core/servicegateway and formal/imports/core/servicegateway.json
//   - resource runtime: servicegateway_runtime.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/core
func TestMockIntegrationServiceGatewayLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &corev1beta1.ServiceGateway{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-service-gateway", Namespace: "default", UID: types.UID("mock-service-gateway-uid")},
		Spec: corev1beta1.ServiceGatewaySpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			VcnId:         "ocid1.vcn.oc1..mock",
			DisplayName:   "mock-service-gateway",
			Services:      []corev1beta1.ServiceGatewayService{{ServiceId: "ocid1.service.oc1..mock"}},
			FreeformTags:  map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newServiceGatewayMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ServiceGateway OCI mock: %v", err)
		}
	})

	manager := newServiceGatewayTestManager(nil)
	client := newTestServiceGatewayDelegate(manager, coresdk.VirtualNetworkClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*corev1beta1.ServiceGateway]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *corev1beta1.ServiceGateway) error {
			if current.Status.Id != mockServiceGatewayID ||
				current.Status.DisplayName != "mock-service-gateway" ||
				len(current.Status.Services) != 1 ||
				current.Status.LifecycleState != string(coresdk.ServiceGatewayLifecycleStateAvailable) {
				return fmt.Errorf("created ServiceGateway status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *corev1beta1.ServiceGateway) {
			current.Spec.DisplayName = "mock-service-gateway-updated"
			current.Spec.BlockTraffic = true
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *corev1beta1.ServiceGateway) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				!current.Status.BlockTraffic ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated ServiceGateway status = %+v", current.Status)
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

func newServiceGatewayMockResponder(resource *corev1beta1.ServiceGateway) (*ocimock.CRUDResponder[coresdk.ServiceGateway], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	serviceID := resource.Spec.Services[0].ServiceId
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[coresdk.ServiceGateway]{
		CollectionPath:         "/20160918/serviceGateways",
		ItemPath:               "/20160918/serviceGateways/" + mockServiceGatewayID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (coresdk.ServiceGateway, ocimock.Response, error) {
			var details coresdk.CreateServiceGatewayDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.ServiceGateway{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return coresdk.ServiceGateway{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.VcnId == nil || *details.VcnId != resource.Spec.VcnId ||
				len(details.Services) != 1 || details.Services[0].ServiceId == nil || *details.Services[0].ServiceId != serviceID ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return coresdk.ServiceGateway{}, ocimock.Response{}, fmt.Errorf("unexpected create ServiceGateway details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return coresdk.ServiceGateway{}, ocimock.Response{}, fmt.Errorf("create ServiceGateway opc-retry-token is empty")
			}
			state := coresdk.ServiceGateway{
				Id:             common.String(mockServiceGatewayID),
				CompartmentId:  details.CompartmentId,
				VcnId:          details.VcnId,
				DisplayName:    details.DisplayName,
				BlockTraffic:   common.Bool(false),
				Services:       []coresdk.ServiceIdResponseDetails{{ServiceId: common.String(serviceID), ServiceName: common.String("Mock OCI Service")}},
				FreeformTags:   details.FreeformTags,
				TimeCreated:    &createdAt,
				LifecycleState: coresdk.ServiceGatewayLifecycleStateProvisioning,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state coresdk.ServiceGateway) (coresdk.ServiceGateway, ocimock.Response, error) {
			switch state.LifecycleState {
			case coresdk.ServiceGatewayLifecycleStateProvisioning:
				state.LifecycleState = coresdk.ServiceGatewayLifecycleStateAvailable
			case coresdk.ServiceGatewayLifecycleStateTerminating:
				if deleteReadObserved {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
					return state, response, err
				}
				deleteReadObserved = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state coresdk.ServiceGateway) (coresdk.ServiceGateway, ocimock.Response, error) {
			var details coresdk.UpdateServiceGatewayDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.ServiceGateway{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-service-gateway-updated" ||
				details.BlockTraffic == nil || !*details.BlockTraffic ||
				details.FreeformTags["osok-mock"] != "update" ||
				details.Services != nil {
				return coresdk.ServiceGateway{}, ocimock.Response{}, fmt.Errorf("unexpected update ServiceGateway details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.BlockTraffic = details.BlockTraffic
			state.FreeformTags = details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state coresdk.ServiceGateway) (coresdk.ServiceGateway, ocimock.Response, error) {
			state.LifecycleState = coresdk.ServiceGatewayLifecycleStateTerminating
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

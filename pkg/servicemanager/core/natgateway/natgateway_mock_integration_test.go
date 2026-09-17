/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package natgateway

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

const mockNatGatewayID = "ocid1.natgateway.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/core/natgateway and formal/imports/core/natgateway.json
//   - resource runtime: natgateway_runtime.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/core
func TestMockIntegrationNatGatewayLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &corev1beta1.NatGateway{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-nat-gateway", Namespace: "default", UID: types.UID("mock-nat-gateway-uid")},
		Spec: corev1beta1.NatGatewaySpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			VcnId:         "ocid1.vcn.oc1..mock",
			DisplayName:   "mock-nat-gateway",
			BlockTraffic:  true,
			FreeformTags:  map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newNatGatewayMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close NatGateway OCI mock: %v", err)
		}
	})

	manager := newNatGatewayTestManager(nil)
	client := newTestNatGatewayDelegate(manager, coresdk.VirtualNetworkClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*corev1beta1.NatGateway]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *corev1beta1.NatGateway) error {
			if current.Status.Id != mockNatGatewayID ||
				current.Status.DisplayName != "mock-nat-gateway" ||
				current.Status.PublicIpId == "" ||
				!current.Status.BlockTraffic ||
				current.Status.LifecycleState != string(coresdk.NatGatewayLifecycleStateAvailable) {
				return fmt.Errorf("created NatGateway status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *corev1beta1.NatGateway) {
			current.Spec.DisplayName = "mock-nat-gateway-updated"
			current.Spec.BlockTraffic = false
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *corev1beta1.NatGateway) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.BlockTraffic ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated NatGateway status = %+v", current.Status)
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

func newNatGatewayMockResponder(resource *corev1beta1.NatGateway) (*ocimock.CRUDResponder[coresdk.NatGateway], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[coresdk.NatGateway]{
		CollectionPath:         "/20160918/natGateways",
		ItemPath:               "/20160918/natGateways/" + mockNatGatewayID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (coresdk.NatGateway, ocimock.Response, error) {
			var details coresdk.CreateNatGatewayDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.NatGateway{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return coresdk.NatGateway{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.VcnId == nil || *details.VcnId != resource.Spec.VcnId ||
				details.BlockTraffic == nil || !*details.BlockTraffic ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return coresdk.NatGateway{}, ocimock.Response{}, fmt.Errorf("unexpected create NatGateway details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return coresdk.NatGateway{}, ocimock.Response{}, fmt.Errorf("create NatGateway opc-retry-token is empty")
			}
			state := coresdk.NatGateway{
				Id:             common.String(mockNatGatewayID),
				CompartmentId:  details.CompartmentId,
				VcnId:          details.VcnId,
				DisplayName:    details.DisplayName,
				BlockTraffic:   details.BlockTraffic,
				FreeformTags:   details.FreeformTags,
				PublicIpId:     common.String("ocid1.publicip.oc1..mock"),
				NatIp:          common.String("192.0.2.10"),
				TimeCreated:    &createdAt,
				LifecycleState: coresdk.NatGatewayLifecycleStateProvisioning,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state coresdk.NatGateway) (coresdk.NatGateway, ocimock.Response, error) {
			switch state.LifecycleState {
			case coresdk.NatGatewayLifecycleStateProvisioning:
				state.LifecycleState = coresdk.NatGatewayLifecycleStateAvailable
			case coresdk.NatGatewayLifecycleStateTerminating:
				if deleteReadObserved {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
					return state, response, err
				}
				deleteReadObserved = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state coresdk.NatGateway) (coresdk.NatGateway, ocimock.Response, error) {
			var details coresdk.UpdateNatGatewayDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.NatGateway{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-nat-gateway-updated" ||
				details.BlockTraffic == nil || *details.BlockTraffic ||
				details.FreeformTags["osok-mock"] != "update" {
				return coresdk.NatGateway{}, ocimock.Response{}, fmt.Errorf("unexpected update NatGateway details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.BlockTraffic = details.BlockTraffic
			state.FreeformTags = details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state coresdk.NatGateway) (coresdk.NatGateway, ocimock.Response, error) {
			state.LifecycleState = coresdk.NatGatewayLifecycleStateTerminating
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

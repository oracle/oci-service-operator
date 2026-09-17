/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package internetgateway

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

const mockInternetGatewayID = "ocid1.internetgateway.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/core/internetgateway and formal/imports/core/internetgateway.json
//   - resource runtime: internetgateway_runtime.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/core
func TestMockIntegrationInternetGatewayLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &corev1beta1.InternetGateway{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-internet-gateway", Namespace: "default", UID: types.UID("mock-internet-gateway-uid")},
		Spec: corev1beta1.InternetGatewaySpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			IsEnabled:     true,
			VcnId:         "ocid1.vcn.oc1..mock",
			DisplayName:   "mock-internet-gateway",
			FreeformTags:  map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newInternetGatewayMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close InternetGateway OCI mock: %v", err)
		}
	})

	manager := newInternetGatewayTestManager(nil)
	client := newTestGeneratedInternetGatewayDelegate(manager, coresdk.VirtualNetworkClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*corev1beta1.InternetGateway]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *corev1beta1.InternetGateway) error {
			if current.Status.Id != mockInternetGatewayID ||
				current.Status.DisplayName != "mock-internet-gateway" ||
				!current.Status.IsEnabled ||
				current.Status.LifecycleState != string(coresdk.InternetGatewayLifecycleStateAvailable) {
				return fmt.Errorf("created InternetGateway status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *corev1beta1.InternetGateway) {
			current.Spec.DisplayName = "mock-internet-gateway-updated"
			current.Spec.IsEnabled = false
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *corev1beta1.InternetGateway) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.IsEnabled ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated InternetGateway status = %+v", current.Status)
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

func newInternetGatewayMockResponder(resource *corev1beta1.InternetGateway) (*ocimock.CRUDResponder[coresdk.InternetGateway], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[coresdk.InternetGateway]{
		CollectionPath:         "/20160918/internetGateways",
		ItemPath:               "/20160918/internetGateways/" + mockInternetGatewayID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (coresdk.InternetGateway, ocimock.Response, error) {
			var details coresdk.CreateInternetGatewayDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.InternetGateway{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return coresdk.InternetGateway{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.VcnId == nil || *details.VcnId != resource.Spec.VcnId ||
				details.IsEnabled == nil || !*details.IsEnabled ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return coresdk.InternetGateway{}, ocimock.Response{}, fmt.Errorf("unexpected create InternetGateway details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return coresdk.InternetGateway{}, ocimock.Response{}, fmt.Errorf("create InternetGateway opc-retry-token is empty")
			}
			state := coresdk.InternetGateway{
				Id:             common.String(mockInternetGatewayID),
				CompartmentId:  details.CompartmentId,
				VcnId:          details.VcnId,
				IsEnabled:      details.IsEnabled,
				DisplayName:    details.DisplayName,
				FreeformTags:   details.FreeformTags,
				TimeCreated:    &createdAt,
				LifecycleState: coresdk.InternetGatewayLifecycleStateProvisioning,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state coresdk.InternetGateway) (coresdk.InternetGateway, ocimock.Response, error) {
			switch state.LifecycleState {
			case coresdk.InternetGatewayLifecycleStateProvisioning:
				state.LifecycleState = coresdk.InternetGatewayLifecycleStateAvailable
			case coresdk.InternetGatewayLifecycleStateTerminating:
				if deleteReadObserved {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
					return state, response, err
				}
				deleteReadObserved = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state coresdk.InternetGateway) (coresdk.InternetGateway, ocimock.Response, error) {
			var details coresdk.UpdateInternetGatewayDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.InternetGateway{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-internet-gateway-updated" ||
				details.IsEnabled == nil || *details.IsEnabled ||
				details.FreeformTags["osok-mock"] != "update" {
				return coresdk.InternetGateway{}, ocimock.Response{}, fmt.Errorf("unexpected update InternetGateway details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.IsEnabled = details.IsEnabled
			state.FreeformTags = details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state coresdk.InternetGateway) (coresdk.InternetGateway, ocimock.Response, error) {
			state.LifecycleState = coresdk.InternetGatewayLifecycleStateTerminating
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

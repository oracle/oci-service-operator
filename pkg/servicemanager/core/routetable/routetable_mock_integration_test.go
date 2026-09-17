/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package routetable

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

const mockRouteTableID = "ocid1.routetable.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/core/routetable and formal/imports/core/routetable.json
//   - resource runtime: routetable_runtime.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/core
func TestMockIntegrationRouteTableLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &corev1beta1.RouteTable{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-route-table", Namespace: "default", UID: types.UID("mock-route-table-uid")},
		Spec: corev1beta1.RouteTableSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			VcnId:         "ocid1.vcn.oc1..mock",
			DisplayName:   "mock-route-table",
			RouteRules:    []corev1beta1.RouteTableRouteRule{},
			FreeformTags:  map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newRouteTableMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close RouteTable OCI mock: %v", err)
		}
	})

	manager := newRouteTableTestManager(nil)
	client := newTestGeneratedDelegate(manager, coresdk.VirtualNetworkClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*corev1beta1.RouteTable]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *corev1beta1.RouteTable) error {
			if current.Status.Id != mockRouteTableID ||
				current.Status.DisplayName != "mock-route-table" ||
				current.Status.LifecycleState != string(coresdk.RouteTableLifecycleStateAvailable) {
				return fmt.Errorf("created RouteTable status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *corev1beta1.RouteTable) {
			current.Spec.DisplayName = "mock-route-table-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
			current.Spec.RouteRules = []corev1beta1.RouteTableRouteRule{{
				NetworkEntityId: "ocid1.internetgateway.oc1..mock",
				Destination:     "0.0.0.0/0",
				DestinationType: "CIDR_BLOCK",
			}}
		},
		ValidateUpdated: func(current *corev1beta1.RouteTable) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				len(current.Status.RouteRules) != 1 ||
				current.Status.RouteRules[0].Destination != "0.0.0.0/0" {
				return fmt.Errorf("updated RouteTable status = %+v", current.Status)
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

func newRouteTableMockResponder(resource *corev1beta1.RouteTable) (*ocimock.CRUDResponder[coresdk.RouteTable], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteReadObserved := false
	updateReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[coresdk.RouteTable]{
		CollectionPath:         "/20160918/routeTables",
		ItemPath:               "/20160918/routeTables/" + mockRouteTableID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (coresdk.RouteTable, ocimock.Response, error) {
			var details coresdk.CreateRouteTableDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.RouteTable{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return coresdk.RouteTable{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.VcnId == nil || *details.VcnId != resource.Spec.VcnId ||
				details.RouteRules == nil || len(details.RouteRules) != 0 ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return coresdk.RouteTable{}, ocimock.Response{}, fmt.Errorf("unexpected create RouteTable details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return coresdk.RouteTable{}, ocimock.Response{}, fmt.Errorf("create RouteTable opc-retry-token is empty")
			}
			state := coresdk.RouteTable{
				Id:             common.String(mockRouteTableID),
				CompartmentId:  details.CompartmentId,
				VcnId:          details.VcnId,
				DisplayName:    details.DisplayName,
				RouteRules:     details.RouteRules,
				FreeformTags:   details.FreeformTags,
				TimeCreated:    &createdAt,
				LifecycleState: coresdk.RouteTableLifecycleStateProvisioning,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state coresdk.RouteTable) (coresdk.RouteTable, ocimock.Response, error) {
			switch state.LifecycleState {
			case coresdk.RouteTableLifecycleStateProvisioning:
				state.LifecycleState = coresdk.RouteTableLifecycleStateAvailable
			case "UPDATING":
				if updateReadObserved {
					state.LifecycleState = coresdk.RouteTableLifecycleStateAvailable
				} else {
					updateReadObserved = true
				}
			case coresdk.RouteTableLifecycleStateTerminating:
				if deleteReadObserved {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
					return state, response, err
				}
				deleteReadObserved = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state coresdk.RouteTable) (coresdk.RouteTable, ocimock.Response, error) {
			var details coresdk.UpdateRouteTableDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.RouteTable{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-route-table-updated" ||
				details.FreeformTags["osok-mock"] != "update" ||
				len(details.RouteRules) != 1 ||
				details.RouteRules[0].NetworkEntityId == nil ||
				*details.RouteRules[0].NetworkEntityId != "ocid1.internetgateway.oc1..mock" {
				return coresdk.RouteTable{}, ocimock.Response{}, fmt.Errorf("unexpected update RouteTable details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.FreeformTags = details.FreeformTags
			state.RouteRules = details.RouteRules
			state.LifecycleState = "UPDATING"
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state coresdk.RouteTable) (coresdk.RouteTable, ocimock.Response, error) {
			state.LifecycleState = coresdk.RouteTableLifecycleStateTerminating
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

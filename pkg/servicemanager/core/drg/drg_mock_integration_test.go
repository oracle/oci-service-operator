/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package drg

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

const mockDrgID = "ocid1.drg.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal DRG contract, and vendored OCI SDK.
func TestMockIntegrationDrgLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &corev1beta1.Drg{ObjectMeta: metav1.ObjectMeta{Name: "mock-drg", Namespace: "default", UID: types.UID("mock-drg-uid")}, Spec: corev1beta1.DrgSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-drg", FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newDrgMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newMockDrgClient(coresdk.VirtualNetworkClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*corev1beta1.Drg]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *corev1beta1.Drg) error {
			if current.Status.Id != mockDrgID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(coresdk.DrgLifecycleStateAvailable) {
				return fmt.Errorf("created DRG status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *corev1beta1.Drg) {
			current.Spec.DisplayName = "mock-drg-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *corev1beta1.Drg) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated DRG status = %+v", current.Status)
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

func newDrgMockResponder(resource *corev1beta1.Drg) (*ocimock.CRUDResponder[coresdk.Drg], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createRead, deleteRead := false, false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[coresdk.Drg]{
		CollectionPath: "/20160918/drgs", ItemPath: "/20160918/drgs/" + mockDrgID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (coresdk.Drg, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero coresdk.Drg
				return zero, ocimock.Response{}, err
			}
			var details coresdk.CreateDrgDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.Drg{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return coresdk.Drg{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName {
				return coresdk.Drg{}, ocimock.Response{}, fmt.Errorf("unexpected CreateDrg details: %+v", details)
			}
			state := coresdk.Drg{CompartmentId: details.CompartmentId, Id: common.String(mockDrgID), LifecycleState: coresdk.DrgLifecycleStateProvisioning,
				DisplayName: details.DisplayName, FreeformTags: details.FreeformTags, TimeCreated: &now,
				DefaultDrgRouteTables: &coresdk.DefaultDrgRouteTables{IpsecTunnel: common.String("ocid1.drgroutetable.oc1..mock-ipsec"), Vcn: common.String("ocid1.drgroutetable.oc1..mock-vcn"), RemotePeeringConnection: common.String("ocid1.drgroutetable.oc1..mock-rpc")}}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state coresdk.Drg) (coresdk.Drg, ocimock.Response, error) {
			if state.LifecycleState == coresdk.DrgLifecycleStateProvisioning {
				if createRead {
					state.LifecycleState = coresdk.DrgLifecycleStateAvailable
				} else {
					createRead = true
				}
			} else if state.LifecycleState == coresdk.DrgLifecycleStateTerminating {
				if deleteRead {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound", "message": "resource not found"})
					return state, response, err
				}
				deleteRead = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state coresdk.Drg) (coresdk.Drg, ocimock.Response, error) {
			var details coresdk.UpdateDrgDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.Drg{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-drg-updated" || details.FreeformTags["osok-mock"] != "update" {
				return coresdk.Drg{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateDrg details: %+v", details)
			}
			state.DisplayName, state.FreeformTags = details.DisplayName, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state coresdk.Drg) (coresdk.Drg, ocimock.Response, error) {
			state.LifecycleState = coresdk.DrgLifecycleStateTerminating
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

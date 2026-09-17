/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networksecuritygroup

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

const mockNetworkSecurityGroupID = "ocid1.networksecuritygroup.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/core/networksecuritygroup and formal/imports/core/networksecuritygroup.json
//   - resource runtime: networksecuritygroup_runtime.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/core
func TestMockIntegrationNetworkSecurityGroupLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &corev1beta1.NetworkSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-network-security-group", Namespace: "default", UID: types.UID("mock-network-security-group-uid")},
		Spec: corev1beta1.NetworkSecurityGroupSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			VcnId:         "ocid1.vcn.oc1..mock",
			DisplayName:   "mock-network-security-group",
			FreeformTags:  map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newNetworkSecurityGroupMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close NetworkSecurityGroup OCI mock: %v", err)
		}
	})

	manager := newNetworkSecurityGroupTestManager(nil)
	client := newTestGeneratedNetworkSecurityGroupDelegate(manager, coresdk.VirtualNetworkClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*corev1beta1.NetworkSecurityGroup]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *corev1beta1.NetworkSecurityGroup) error {
			if current.Status.Id != mockNetworkSecurityGroupID ||
				current.Status.DisplayName != "mock-network-security-group" ||
				current.Status.LifecycleState != string(coresdk.NetworkSecurityGroupLifecycleStateAvailable) {
				return fmt.Errorf("created NetworkSecurityGroup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *corev1beta1.NetworkSecurityGroup) {
			current.Spec.DisplayName = "mock-network-security-group-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *corev1beta1.NetworkSecurityGroup) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(coresdk.NetworkSecurityGroupLifecycleStateAvailable) {
				return fmt.Errorf("updated NetworkSecurityGroup status = %+v", current.Status)
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

func newNetworkSecurityGroupMockResponder(resource *corev1beta1.NetworkSecurityGroup) (*ocimock.CRUDResponder[coresdk.NetworkSecurityGroup], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteReadObserved := false
	updateReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[coresdk.NetworkSecurityGroup]{
		CollectionPath:         "/20160918/networkSecurityGroups",
		ItemPath:               "/20160918/networkSecurityGroups/" + mockNetworkSecurityGroupID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (coresdk.NetworkSecurityGroup, ocimock.Response, error) {
			var details coresdk.CreateNetworkSecurityGroupDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.NetworkSecurityGroup{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return coresdk.NetworkSecurityGroup{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.VcnId == nil || *details.VcnId != resource.Spec.VcnId ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName ||
				details.FreeformTags["osok-mock"] != "create" {
				return coresdk.NetworkSecurityGroup{}, ocimock.Response{}, fmt.Errorf("unexpected create NetworkSecurityGroup details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return coresdk.NetworkSecurityGroup{}, ocimock.Response{}, fmt.Errorf("create NetworkSecurityGroup opc-retry-token is empty")
			}
			state := coresdk.NetworkSecurityGroup{
				Id:             common.String(mockNetworkSecurityGroupID),
				CompartmentId:  details.CompartmentId,
				VcnId:          details.VcnId,
				DisplayName:    details.DisplayName,
				FreeformTags:   details.FreeformTags,
				TimeCreated:    &createdAt,
				LifecycleState: coresdk.NetworkSecurityGroupLifecycleStateProvisioning,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state coresdk.NetworkSecurityGroup) (coresdk.NetworkSecurityGroup, ocimock.Response, error) {
			switch state.LifecycleState {
			case coresdk.NetworkSecurityGroupLifecycleStateProvisioning:
				state.LifecycleState = coresdk.NetworkSecurityGroupLifecycleStateAvailable
			case "UPDATING":
				if updateReadObserved {
					state.LifecycleState = coresdk.NetworkSecurityGroupLifecycleStateAvailable
				} else {
					updateReadObserved = true
				}
			case coresdk.NetworkSecurityGroupLifecycleStateTerminating:
				if deleteReadObserved {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
					return state, response, err
				}
				deleteReadObserved = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state coresdk.NetworkSecurityGroup) (coresdk.NetworkSecurityGroup, ocimock.Response, error) {
			var details coresdk.UpdateNetworkSecurityGroupDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.NetworkSecurityGroup{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-network-security-group-updated" ||
				details.FreeformTags["osok-mock"] != "update" {
				return coresdk.NetworkSecurityGroup{}, ocimock.Response{}, fmt.Errorf("unexpected update NetworkSecurityGroup details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.FreeformTags = details.FreeformTags
			state.LifecycleState = "UPDATING"
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state coresdk.NetworkSecurityGroup) (coresdk.NetworkSecurityGroup, ocimock.Response, error) {
			state.LifecycleState = coresdk.NetworkSecurityGroupLifecycleStateTerminating
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

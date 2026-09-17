/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securitylist

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

const mockSecurityListID = "ocid1.securitylist.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/core/securitylist and formal/imports/core/securitylist.json
//   - resource runtime: securitylist_runtime.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/core
func TestMockIntegrationSecurityListLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &corev1beta1.SecurityList{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-security-list", Namespace: "default", UID: types.UID("mock-security-list-uid")},
		Spec: corev1beta1.SecurityListSpec{
			CompartmentId:        "ocid1.compartment.oc1..mock",
			VcnId:                "ocid1.vcn.oc1..mock",
			DisplayName:          "mock-security-list",
			EgressSecurityRules:  []corev1beta1.SecurityListEgressSecurityRule{},
			IngressSecurityRules: []corev1beta1.SecurityListIngressSecurityRule{},
			FreeformTags:         map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newSecurityListMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close SecurityList OCI mock: %v", err)
		}
	})

	manager := newSecurityListTestManager(nil)
	client := newSecurityListServiceClientWithOCIClient(manager.Log, coresdk.VirtualNetworkClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*corev1beta1.SecurityList]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *corev1beta1.SecurityList) error {
			if current.Status.Id != mockSecurityListID ||
				current.Status.DisplayName != "mock-security-list" ||
				current.Status.LifecycleState != string(coresdk.SecurityListLifecycleStateAvailable) {
				return fmt.Errorf("created SecurityList status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *corev1beta1.SecurityList) {
			current.Spec.DisplayName = "mock-security-list-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
			current.Spec.EgressSecurityRules = []corev1beta1.SecurityListEgressSecurityRule{{
				Destination: "10.99.0.0/16",
				Protocol:    "all",
			}}
		},
		ValidateUpdated: func(current *corev1beta1.SecurityList) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				len(current.Status.EgressSecurityRules) != 1 ||
				current.Status.EgressSecurityRules[0].Destination != "10.99.0.0/16" {
				return fmt.Errorf("updated SecurityList status = %+v", current.Status)
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

func newSecurityListMockResponder(resource *corev1beta1.SecurityList) (*ocimock.CRUDResponder[coresdk.SecurityList], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteReadObserved := false
	updateReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[coresdk.SecurityList]{
		CollectionPath:         "/20160918/securityLists",
		ItemPath:               "/20160918/securityLists/" + mockSecurityListID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (coresdk.SecurityList, ocimock.Response, error) {
			var details coresdk.CreateSecurityListDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.SecurityList{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return coresdk.SecurityList{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.VcnId == nil || *details.VcnId != resource.Spec.VcnId ||
				details.EgressSecurityRules == nil || len(details.EgressSecurityRules) != 0 ||
				details.IngressSecurityRules == nil || len(details.IngressSecurityRules) != 0 {
				return coresdk.SecurityList{}, ocimock.Response{}, fmt.Errorf("unexpected create SecurityList details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return coresdk.SecurityList{}, ocimock.Response{}, fmt.Errorf("create SecurityList opc-retry-token is empty")
			}
			state := coresdk.SecurityList{
				Id:                   common.String(mockSecurityListID),
				CompartmentId:        details.CompartmentId,
				VcnId:                details.VcnId,
				DisplayName:          details.DisplayName,
				EgressSecurityRules:  details.EgressSecurityRules,
				IngressSecurityRules: details.IngressSecurityRules,
				FreeformTags:         details.FreeformTags,
				TimeCreated:          &createdAt,
				LifecycleState:       coresdk.SecurityListLifecycleStateProvisioning,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state coresdk.SecurityList) (coresdk.SecurityList, ocimock.Response, error) {
			switch state.LifecycleState {
			case coresdk.SecurityListLifecycleStateProvisioning:
				state.LifecycleState = coresdk.SecurityListLifecycleStateAvailable
			case "UPDATING":
				if updateReadObserved {
					state.LifecycleState = coresdk.SecurityListLifecycleStateAvailable
				} else {
					updateReadObserved = true
				}
			case coresdk.SecurityListLifecycleStateTerminating:
				if deleteReadObserved {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
					return state, response, err
				}
				deleteReadObserved = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state coresdk.SecurityList) (coresdk.SecurityList, ocimock.Response, error) {
			var details coresdk.UpdateSecurityListDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.SecurityList{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-security-list-updated" ||
				details.FreeformTags["osok-mock"] != "update" ||
				len(details.EgressSecurityRules) != 1 ||
				details.EgressSecurityRules[0].Destination == nil ||
				*details.EgressSecurityRules[0].Destination != "10.99.0.0/16" {
				return coresdk.SecurityList{}, ocimock.Response{}, fmt.Errorf("unexpected update SecurityList details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.FreeformTags = details.FreeformTags
			state.EgressSecurityRules = details.EgressSecurityRules
			state.IngressSecurityRules = details.IngressSecurityRules
			state.LifecycleState = "UPDATING"
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state coresdk.SecurityList) (coresdk.SecurityList, ocimock.Response, error) {
			state.LifecycleState = coresdk.SecurityListLifecycleStateTerminating
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

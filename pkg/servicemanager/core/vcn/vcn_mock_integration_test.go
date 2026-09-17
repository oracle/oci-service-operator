/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vcn

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

const mockVcnID = "ocid1.vcn.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/core/vcn and formal/imports/core/vcn.json
//   - resource runtime: vcn_runtime.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/core
func TestMockIntegrationVcnLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &corev1beta1.Vcn{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-vcn", Namespace: "default", UID: types.UID("mock-vcn-uid")},
		Spec: corev1beta1.VcnSpec{
			CompartmentId: "ocid1.compartment.oc1..mock",
			CidrBlocks:    []string{"10.95.0.0/16"},
			DisplayName:   "mock-vcn",
			DnsLabel:      "mockvcn",
			FreeformTags:  map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newVcnMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Vcn OCI mock: %v", err)
		}
	})

	manager := newTestManager(nil)
	client := newTestGeneratedDelegate(manager, coresdk.VirtualNetworkClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*corev1beta1.Vcn]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *corev1beta1.Vcn) error {
			if current.Status.Id != mockVcnID ||
				current.Status.DisplayName != "mock-vcn" ||
				current.Status.LifecycleState != string(coresdk.VcnLifecycleStateAvailable) {
				return fmt.Errorf("created Vcn status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *corev1beta1.Vcn) {
			current.Spec.DisplayName = "mock-vcn-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *corev1beta1.Vcn) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(coresdk.VcnLifecycleStateAvailable) {
				return fmt.Errorf("updated Vcn status = %+v", current.Status)
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

func newVcnMockResponder(resource *corev1beta1.Vcn) (*ocimock.CRUDResponder[coresdk.Vcn], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[coresdk.Vcn]{
		CollectionPath:         "/20160918/vcns",
		ItemPath:               "/20160918/vcns/" + mockVcnID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (coresdk.Vcn, ocimock.Response, error) {
			var details coresdk.CreateVcnDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.Vcn{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return coresdk.Vcn{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				len(details.CidrBlocks) != 1 || details.CidrBlocks[0] != resource.Spec.CidrBlocks[0] ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName ||
				details.DnsLabel == nil || *details.DnsLabel != resource.Spec.DnsLabel {
				return coresdk.Vcn{}, ocimock.Response{}, fmt.Errorf("unexpected create Vcn details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return coresdk.Vcn{}, ocimock.Response{}, fmt.Errorf("create Vcn opc-retry-token is empty")
			}
			state := coresdk.Vcn{
				Id:             common.String(mockVcnID),
				CompartmentId:  details.CompartmentId,
				CidrBlock:      common.String(details.CidrBlocks[0]),
				CidrBlocks:     append([]string(nil), details.CidrBlocks...),
				DisplayName:    details.DisplayName,
				DnsLabel:       details.DnsLabel,
				FreeformTags:   details.FreeformTags,
				TimeCreated:    &createdAt,
				LifecycleState: coresdk.VcnLifecycleStateProvisioning,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state coresdk.Vcn) (coresdk.Vcn, ocimock.Response, error) {
			switch state.LifecycleState {
			case coresdk.VcnLifecycleStateProvisioning, coresdk.VcnLifecycleStateUpdating:
				state.LifecycleState = coresdk.VcnLifecycleStateAvailable
			case coresdk.VcnLifecycleStateTerminating:
				if deleteReadObserved {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
					return state, response, err
				}
				deleteReadObserved = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state coresdk.Vcn) (coresdk.Vcn, ocimock.Response, error) {
			var details coresdk.UpdateVcnDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.Vcn{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-vcn-updated" ||
				details.FreeformTags["osok-mock"] != "update" {
				return coresdk.Vcn{}, ocimock.Response{}, fmt.Errorf("unexpected update Vcn details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.FreeformTags = details.FreeformTags
			state.LifecycleState = coresdk.VcnLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state coresdk.Vcn) (coresdk.Vcn, ocimock.Response, error) {
			state.LifecycleState = coresdk.VcnLifecycleStateTerminating
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

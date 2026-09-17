/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subnet

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

const mockSubnetID = "ocid1.subnet.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/core/subnet and formal/imports/core/subnet.json
//   - resource runtime: subnet_runtime.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/core
func TestMockIntegrationSubnetLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &corev1beta1.Subnet{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-subnet", Namespace: "default", UID: types.UID("mock-subnet-uid")},
		Spec: corev1beta1.SubnetSpec{
			CidrBlock:               "10.96.1.0/24",
			CompartmentId:           "ocid1.compartment.oc1..mock",
			VcnId:                   "ocid1.vcn.oc1..mock",
			DisplayName:             "mock-subnet",
			DnsLabel:                "mocksub",
			ProhibitPublicIpOnVnic:  true,
			ProhibitInternetIngress: true,
			FreeformTags:            map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newSubnetMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Subnet OCI mock: %v", err)
		}
	})

	manager := newTestManager(nil)
	client := newTestGeneratedDelegate(manager, coresdk.VirtualNetworkClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*corev1beta1.Subnet]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *corev1beta1.Subnet) error {
			if current.Status.Id != mockSubnetID ||
				current.Status.DisplayName != "mock-subnet" ||
				!current.Status.ProhibitPublicIpOnVnic ||
				current.Status.LifecycleState != string(coresdk.SubnetLifecycleStateAvailable) {
				return fmt.Errorf("created Subnet status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *corev1beta1.Subnet) {
			current.Spec.DisplayName = "mock-subnet-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *corev1beta1.Subnet) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(coresdk.SubnetLifecycleStateAvailable) {
				return fmt.Errorf("updated Subnet status = %+v", current.Status)
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

func newSubnetMockResponder(resource *corev1beta1.Subnet) (*ocimock.CRUDResponder[coresdk.Subnet], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[coresdk.Subnet]{
		CollectionPath:         "/20160918/subnets",
		ItemPath:               "/20160918/subnets/" + mockSubnetID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (coresdk.Subnet, ocimock.Response, error) {
			var details coresdk.CreateSubnetDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.Subnet{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return coresdk.Subnet{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.VcnId == nil || *details.VcnId != resource.Spec.VcnId ||
				details.CidrBlock == nil || *details.CidrBlock != resource.Spec.CidrBlock ||
				details.ProhibitPublicIpOnVnic == nil || !*details.ProhibitPublicIpOnVnic {
				return coresdk.Subnet{}, ocimock.Response{}, fmt.Errorf("unexpected create Subnet details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return coresdk.Subnet{}, ocimock.Response{}, fmt.Errorf("create Subnet opc-retry-token is empty")
			}
			state := coresdk.Subnet{
				Id:                      common.String(mockSubnetID),
				CompartmentId:           details.CompartmentId,
				VcnId:                   details.VcnId,
				CidrBlock:               details.CidrBlock,
				Ipv4CidrBlocks:          []string{resource.Spec.CidrBlock},
				DisplayName:             details.DisplayName,
				DnsLabel:                details.DnsLabel,
				ProhibitPublicIpOnVnic:  details.ProhibitPublicIpOnVnic,
				ProhibitInternetIngress: details.ProhibitInternetIngress,
				FreeformTags:            details.FreeformTags,
				TimeCreated:             &createdAt,
				LifecycleState:          coresdk.SubnetLifecycleStateProvisioning,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state coresdk.Subnet) (coresdk.Subnet, ocimock.Response, error) {
			switch state.LifecycleState {
			case coresdk.SubnetLifecycleStateProvisioning, coresdk.SubnetLifecycleStateUpdating:
				state.LifecycleState = coresdk.SubnetLifecycleStateAvailable
			case coresdk.SubnetLifecycleStateTerminating:
				if deleteReadObserved {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
					return state, response, err
				}
				deleteReadObserved = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state coresdk.Subnet) (coresdk.Subnet, ocimock.Response, error) {
			var details coresdk.UpdateSubnetDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.Subnet{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-subnet-updated" ||
				details.FreeformTags["osok-mock"] != "update" {
				return coresdk.Subnet{}, ocimock.Response{}, fmt.Errorf("unexpected update Subnet details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.FreeformTags = details.FreeformTags
			state.LifecycleState = coresdk.SubnetLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state coresdk.Subnet) (coresdk.Subnet, ocimock.Response, error) {
			state.LifecycleState = coresdk.SubnetLifecycleStateTerminating
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

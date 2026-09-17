/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package instance

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

const mockInstanceID = "ocid1.instance.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/core/instance and formal/imports/core/instance.json
//   - resource runtime: instance_runtime.go
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/core/core_instance_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/core
func TestMockIntegrationInstanceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &corev1beta1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-instance", Namespace: "default", UID: types.UID("mock-instance-uid")},
		Spec: corev1beta1.InstanceSpec{
			AvailabilityDomain: "mock:AD-1",
			CompartmentId:      "ocid1.compartment.oc1..mock",
			DisplayName:        "mock-instance",
			FreeformTags:       map[string]string{"osok-mock": "create"},
			Shape:              "VM.Standard.E4.Flex",
			ShapeConfig: corev1beta1.InstanceShapeConfig{
				Ocpus:       1,
				MemoryInGBs: 16,
			},
			SourceDetails: corev1beta1.InstanceSourceDetails{
				SourceType: "image",
				ImageId:    "ocid1.image.oc1..mock",
			},
			InstanceOptions: corev1beta1.InstanceOptions{
				AreLegacyImdsEndpointsDisabled: true,
			},
			SubnetId: "ocid1.subnet.oc1..mock",
		},
	}
	responder, err := newInstanceMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://iaas.mock.invalid", BasePath: "20160918", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Instance OCI mock: %v", err)
		}
	})

	manager := newInstanceTestManager(nil)
	sdkClient := coresdk.ComputeClient{BaseClient: session.BaseClient()}
	client := defaultInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*corev1beta1.Instance](
			newInstanceRuntimeConfig(manager.Log, sdkClient),
		),
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*corev1beta1.Instance]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *corev1beta1.Instance) error {
			if current.Status.Id != mockInstanceID ||
				current.Status.DisplayName != "mock-instance" ||
				current.Status.LifecycleState != string(coresdk.InstanceLifecycleStateRunning) ||
				current.Status.ShapeConfig.Ocpus != 1 ||
				current.Status.ShapeConfig.MemoryInGBs != 16 ||
				current.Status.SourceDetails.ImageId != resource.Spec.SourceDetails.ImageId ||
				!current.Status.InstanceOptions.AreLegacyImdsEndpointsDisabled {
				return fmt.Errorf("created Instance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *corev1beta1.Instance) {
			current.Spec.DisplayName = "mock-instance-updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *corev1beta1.Instance) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(coresdk.InstanceLifecycleStateRunning) {
				return fmt.Errorf("updated Instance status = %+v", current.Status)
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

func newInstanceMockResponder(resource *corev1beta1.Instance) (*ocimock.CRUDResponder[coresdk.Instance], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	provisioningReadObserved := false
	updateReadObserved := false
	terminatingReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[coresdk.Instance]{
		CollectionPath:         "/20160918/instances",
		ItemPath:               "/20160918/instances/" + mockInstanceID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (coresdk.Instance, ocimock.Response, error) {
			var details coresdk.LaunchInstanceDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.Instance{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return coresdk.Instance{}, ocimock.Response{}, err
			}
			source, ok := details.SourceDetails.(coresdk.InstanceSourceViaImageDetails)
			if !ok ||
				source.ImageId == nil || *source.ImageId != resource.Spec.SourceDetails.ImageId ||
				details.AvailabilityDomain == nil || *details.AvailabilityDomain != resource.Spec.AvailabilityDomain ||
				details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.Shape == nil || *details.Shape != resource.Spec.Shape ||
				details.SubnetId == nil || *details.SubnetId != resource.Spec.SubnetId ||
				details.ShapeConfig == nil || details.ShapeConfig.Ocpus == nil || *details.ShapeConfig.Ocpus != 1 ||
				details.ShapeConfig.MemoryInGBs == nil || *details.ShapeConfig.MemoryInGBs != 16 ||
				details.InstanceOptions == nil || details.InstanceOptions.AreLegacyImdsEndpointsDisabled == nil ||
				!*details.InstanceOptions.AreLegacyImdsEndpointsDisabled {
				return coresdk.Instance{}, ocimock.Response{}, fmt.Errorf("unexpected LaunchInstance details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return coresdk.Instance{}, ocimock.Response{}, fmt.Errorf("LaunchInstance opc-retry-token is empty")
			}
			state := coresdk.Instance{
				Id:                 common.String(mockInstanceID),
				AvailabilityDomain: details.AvailabilityDomain,
				CompartmentId:      details.CompartmentId,
				DisplayName:        details.DisplayName,
				FreeformTags:       details.FreeformTags,
				LifecycleState:     coresdk.InstanceLifecycleStateProvisioning,
				Region:             common.String("iad"),
				Shape:              details.Shape,
				ShapeConfig: &coresdk.InstanceShapeConfig{
					Ocpus:       details.ShapeConfig.Ocpus,
					MemoryInGBs: details.ShapeConfig.MemoryInGBs,
					Vcpus:       common.Int(2),
				},
				SourceDetails:   source,
				InstanceOptions: details.InstanceOptions,
				TimeCreated:     &createdAt,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-launch"}}
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state coresdk.Instance) (coresdk.Instance, ocimock.Response, error) {
			switch state.LifecycleState {
			case coresdk.InstanceLifecycleStateProvisioning:
				if provisioningReadObserved {
					state.LifecycleState = coresdk.InstanceLifecycleStateRunning
				} else {
					provisioningReadObserved = true
				}
			case coresdk.InstanceLifecycleStateMoving:
				if updateReadObserved {
					state.LifecycleState = coresdk.InstanceLifecycleStateStopping
				} else {
					updateReadObserved = true
				}
			case coresdk.InstanceLifecycleStateStopping:
				state.LifecycleState = coresdk.InstanceLifecycleStateRunning
			case coresdk.InstanceLifecycleStateTerminating:
				if terminatingReadObserved {
					state.LifecycleState = coresdk.InstanceLifecycleStateTerminated
				} else {
					terminatingReadObserved = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state coresdk.Instance) (coresdk.Instance, ocimock.Response, error) {
			var details coresdk.UpdateInstanceDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return coresdk.Instance{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-instance-updated" ||
				details.FreeformTags["osok-mock"] != "update" ||
				details.Shape != nil ||
				details.SourceDetails != nil {
				return coresdk.Instance{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateInstance details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return coresdk.Instance{}, ocimock.Response{}, fmt.Errorf("UpdateInstance opc-retry-token is empty")
			}
			state.DisplayName = details.DisplayName
			state.FreeformTags = details.FreeformTags
			state.LifecycleState = coresdk.InstanceLifecycleStateMoving
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(request ocimock.Request, state coresdk.Instance) (coresdk.Instance, ocimock.Response, error) {
			if request.URL.RawQuery != "" {
				return coresdk.Instance{}, ocimock.Response{}, fmt.Errorf("unexpected TerminateInstance query: %s", request.URL.RawQuery)
			}
			state.LifecycleState = coresdk.InstanceLifecycleStateTerminating
			response := ocimock.EmptyResponse(http.StatusNoContent)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-terminate"}}
			return state, response, nil
		},
	})
}
